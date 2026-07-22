// Package snapshot makes time an explicit axis of Synth's determinism.
//
// One seed already fixes a whole dataset. This package adds: ask for the table
// as it stood at any instant, or for the change events between two instants,
// and the two agree exactly. Applying the events from t0 to t1 to the snapshot
// at t0 reproduces the snapshot at t1, byte for byte.
//
// That equivalence is what makes the package useful for migration and
// incremental-ETL tests, so it is guaranteed by construction rather than by
// luck: both At and Between read the same per-row life, derived from the row's
// index. Nothing is simulated forward, so asking for a distant instant costs
// no more than asking for a near one.
package snapshot

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/bakhod1r/synth/cdc"
	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// maxVersions caps how many times a row may change. The cap keeps each row's
// version space disjoint from every other row's, which is what lets a version
// be regenerated on demand instead of replayed.
const maxVersions = 64

// Config describes the table's life.
type Config struct {
	Table string
	// Key names the column that identifies a row across its versions. Empty
	// means the schema's primary key.
	Key string
	// Rows is how many rows are ever born.
	Rows int
	// Start is when the table came into existence. A snapshot before it is
	// empty.
	Start time.Time
	// Window is the span over which rows are born and change. Default: one year.
	Window time.Duration
	// Churn is the mean number of updates per row over the window.
	Churn float64
	// DeleteFrac is the fraction of rows that are eventually deleted.
	DeleteFrac float64
	// Seed fixes the whole timeline.
	Seed uint64
	// Locale for generated values.
	Locale string
}

// Timeline answers questions about a table across time.
type Timeline struct {
	cfg   Config
	eng   *gen.Engine
	base  *rng.Rand
	lives []life
	// createdCol and updatedCol are timestamp columns the timeline owns, so a
	// row's own timestamps agree with when it was actually born and changed.
	createdCol, updatedCol string
}

// life is one row's whole history, derived from its index.
type life struct {
	index int
	born  time.Time
	// changed holds the instant of each update, in order.
	changed []time.Time
	// died is zero for a row that is never deleted.
	died time.Time
}

// New builds a timeline. It generates nothing until asked.
func New(s *schema.Schema, cfg Config) (*Timeline, error) {
	if cfg.Rows < 0 {
		return nil, fmt.Errorf("snapshot: Rows must not be negative")
	}
	if cfg.Locale == "" {
		cfg.Locale = "en_US"
	}
	if cfg.Table == "" {
		cfg.Table = "data"
	}
	if cfg.Window <= 0 {
		cfg.Window = 365 * 24 * time.Hour
	}
	if cfg.Start.IsZero() {
		cfg.Start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if cfg.DeleteFrac < 0 || cfg.DeleteFrac > 1 {
		return nil, fmt.Errorf("snapshot: DeleteFrac must be between 0 and 1")
	}
	if cfg.Key == "" {
		cfg.Key = primaryKey(s)
	}
	eng, err := gen.Compile(s, cfg.Locale)
	if err != nil {
		return nil, err
	}
	t := &Timeline{
		cfg:        cfg,
		eng:        eng,
		base:       rng.New(cfg.Seed),
		createdCol: findTimeColumn(s, "created", "opened", "registered"),
		updatedCol: findTimeColumn(s, "updated", "modified", "changed"),
	}
	t.lives = make([]life, cfg.Rows)
	for i := range t.lives {
		t.lives[i] = t.lifeOf(i)
	}
	return t, nil
}

// lifeOf derives a row's whole history from its index. It is a pure function
// of the seed and i, so it does not matter in what order lives are computed,
// nor how many of them are.
func (t *Timeline) lifeOf(i int) life {
	r := t.base.Fork(uint64(i))
	span := float64(t.cfg.Window)

	l := life{index: i}
	l.born = t.cfg.Start.Add(time.Duration(r.Float64() * span))

	// Draw an update count with mean Churn. A geometric draw gives the right
	// mean with one parameter and no table of factorials.
	p := t.cfg.Churn / (t.cfg.Churn + 1)
	at := l.born
	remaining := t.cfg.Start.Add(t.cfg.Window).Sub(l.born)
	for len(l.changed) < maxVersions-1 && r.Float64() < p {
		if remaining <= 0 {
			break
		}
		// Exponential gaps, so changes cluster the way real edits do rather
		// than arriving on a metronome.
		gap := time.Duration(-math.Log(1-r.Float64()) * float64(remaining) / 3)
		at = at.Add(gap)
		if at.After(t.cfg.Start.Add(t.cfg.Window)) {
			break
		}
		l.changed = append(l.changed, at)
		remaining = t.cfg.Start.Add(t.cfg.Window).Sub(at)
	}

	if r.Float64() < t.cfg.DeleteFrac {
		end := t.cfg.Start.Add(t.cfg.Window)
		if rest := end.Sub(at); rest > 0 {
			death := at.Add(time.Duration(r.Float64() * float64(rest)))
			if death.After(at) {
				l.died = death
			}
		}
	}
	return l
}

// versionAt returns how many updates had happened by when, and whether the row
// exists at all at that instant.
func (l life) versionAt(when time.Time) (int, bool) {
	if when.Before(l.born) {
		return 0, false
	}
	if !l.died.IsZero() && !when.Before(l.died) {
		return 0, false
	}
	v := 0
	for _, c := range l.changed {
		if c.After(when) {
			break
		}
		v++
	}
	return v, true
}

// At returns the table as it stood at when. Rows not yet born and rows already
// deleted are absent; every row present carries the values of its version as
// of that instant.
func (t *Timeline) At(when time.Time) []map[string]any {
	var out []map[string]any
	for _, l := range t.lives {
		v, alive := l.versionAt(when)
		if !alive {
			continue
		}
		out = append(out, t.state(l, v))
	}
	return out
}

// state materializes version v of a row. The key column is pinned to version
// zero's value so a row keeps its identity as its other fields change, and the
// timestamp columns are stamped from the timeline rather than drawn at random,
// so a row's own `created_at` agrees with when it was actually born.
func (t *Timeline) state(l life, v int) map[string]any {
	rec := t.eng.Record(t.base, l.index*maxVersions+v)
	if v != 0 {
		first := t.eng.Record(t.base, l.index*maxVersions)
		if key, ok := first[t.cfg.Key]; ok {
			rec[t.cfg.Key] = key
		}
	}
	if t.createdCol != "" {
		rec[t.createdCol] = l.born.UTC().Format(time.RFC3339)
	}
	if t.updatedCol != "" {
		at := l.born
		if v > 0 {
			at = l.changed[v-1]
		}
		rec[t.updatedCol] = at.UTC().Format(time.RFC3339)
	}
	return rec
}

// Between returns the change events in (from, to], in timestamp order.
//
// Applying these events to At(from) yields At(to). Both read the same lives,
// so the two cannot drift apart.
func (t *Timeline) Between(from, to time.Time) []cdc.Event {
	type timed struct {
		at time.Time
		ev cdc.Event
	}
	var all []timed
	for _, l := range t.lives {
		// Birth.
		if inWindow(l.born, from, to) {
			all = append(all, timed{l.born, t.event(cdc.OpCreate, l.born, nil, t.state(l, 0))})
		}
		// Updates.
		for v, at := range l.changed {
			if !inWindow(at, from, to) {
				continue
			}
			all = append(all, timed{at, t.event(cdc.OpUpdate, at, t.state(l, v), t.state(l, v+1))})
		}
		// Death.
		if !l.died.IsZero() && inWindow(l.died, from, to) {
			last, _ := l.versionAt(l.died.Add(-time.Nanosecond))
			all = append(all, timed{l.died, t.event(cdc.OpDelete, l.died, t.state(l, last), nil)})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })

	out := make([]cdc.Event, len(all))
	for i, e := range all {
		// The log sequence number is assigned in emission order, so a consumer
		// can order events without re-parsing timestamps.
		e.ev.Source.LSN = int64(i + 1)
		out[i] = e.ev
	}
	return out
}

// inWindow reports whether at falls in (from, to]. The half-open form is what
// makes consecutive ranges tile without gaps or double-counting.
func inWindow(at, from, to time.Time) bool {
	return at.After(from) && !at.After(to)
}

func (t *Timeline) event(op cdc.Op, at time.Time, before, after map[string]any) cdc.Event {
	ms := at.UnixMilli()
	return cdc.Event{
		Op: op, Before: before, After: after, TsMs: ms,
		Source: cdc.Source{
			Connector: "synth", Name: t.cfg.Table, Table: t.cfg.Table, TsMs: ms,
		},
	}
}

// Key returns the column that identifies a row across its versions.
func (t *Timeline) Key() string { return t.cfg.Key }

// Apply replays events onto a snapshot and returns the resulting rows, keyed
// by the timeline's key column. It is what a consumer of the event log would
// do, and it is how the log and the snapshots are checked against each other.
func (t *Timeline) Apply(rows []map[string]any, events []cdc.Event) []map[string]any {
	byKey := make(map[string]map[string]any, len(rows))
	var order []string
	for _, r := range rows {
		k := fmt.Sprint(r[t.cfg.Key])
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		byKey[k] = r
	}
	for _, ev := range events {
		switch ev.Op {
		case cdc.OpCreate, cdc.OpUpdate, cdc.OpRead:
			k := fmt.Sprint(ev.After[t.cfg.Key])
			if _, seen := byKey[k]; !seen {
				order = append(order, k)
			}
			byKey[k] = ev.After
		case cdc.OpDelete:
			delete(byKey, fmt.Sprint(ev.Before[t.cfg.Key]))
		}
	}
	out := make([]map[string]any, 0, len(byKey))
	for _, k := range order {
		if r, ok := byKey[k]; ok {
			out = append(out, r)
		}
	}
	return out
}

func primaryKey(s *schema.Schema) string {
	for _, f := range s.Fields {
		if f.PK {
			return f.Name
		}
	}
	if len(s.Fields) > 0 {
		return s.Fields[0].Name
	}
	return "id"
}

// findTimeColumn returns the schema's timestamp column matching one of the
// given name fragments, if exactly one does. An ambiguous match is left alone
// rather than guessed at.
func findTimeColumn(s *schema.Schema, frags ...string) string {
	var hit string
	for _, f := range s.Fields {
		if f.Kind != schema.KindTime {
			continue
		}
		name := strings.ToLower(f.Name)
		for _, frag := range frags {
			if strings.Contains(name, frag) {
				if hit != "" && hit != f.Name {
					return ""
				}
				hit = f.Name
			}
		}
	}
	return hit
}
