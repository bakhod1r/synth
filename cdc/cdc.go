// Package cdc emits a deterministic stream of change events — insert, update,
// delete — in Debezium's envelope shape. Feed it to anything that consumes CDC
// (a connector, a stream processor, a replication test) without needing a real
// database or Kafka: Synth writes the events to a file or an io.Writer.
//
// The stream is a coherent history, not random noise: a row is inserted before
// it is updated, updates carry the true `before` image, and a deleted row is
// never touched again. Same seed, same history.
package cdc

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// Op is a change operation, using Debezium's single-letter codes.
type Op string

const (
	OpCreate Op = "c" // insert
	OpUpdate Op = "u"
	OpDelete Op = "d"
	OpRead   Op = "r" // snapshot read
)

// Event is a Debezium-shaped change event.
type Event struct {
	Op     Op             `json:"op"`
	Before map[string]any `json:"before"`
	After  map[string]any `json:"after"`
	Source Source         `json:"source"`
	TsMs   int64          `json:"ts_ms"`
}

// Source identifies where a change came from.
type Source struct {
	Connector string `json:"connector"`
	Name      string `json:"name"`
	Table     string `json:"table"`
	TsMs      int64  `json:"ts_ms"`
	LSN       int64  `json:"lsn"`
}

// Config controls the shape of a generated history.
type Config struct {
	Table string
	// Seed makes the whole history reproducible.
	Seed uint64
	// Locale for the generated values.
	Locale string
	// Key names the primary-key column used to identify rows across events.
	Key string
	// UpdateRate and DeleteRate are the probabilities that a step mutates an
	// existing row instead of inserting a new one. They must sum to < 1; the
	// remainder is inserts.
	UpdateRate, DeleteRate float64
	// Start is the timestamp of the first event; each step advances by Interval.
	Start    time.Time
	Interval time.Duration
	// Snapshot emits the initial rows as "r" (read) events, like Debezium's
	// initial snapshot, before the change stream begins.
	Snapshot int
	// SoftDelete turns a drawn delete into an update that stamps
	// SoftDeleteColumn with the event time, rather than an op=d that removes the
	// row. The row is still logically gone — it is not touched again.
	SoftDelete bool
	// SoftDeleteColumn is the column stamped on a soft delete. Defaults to
	// "deleted_at".
	SoftDeleteColumn string
}

// Stream generates change events over a schema.
type Stream struct {
	cfg    Config
	eng    *gen.Engine
	base   *rng.Rand
	rnd    *rng.Rand
	live   []map[string]any // rows currently present, in insertion order
	lsn    int64
	nextTs time.Time
	seq    int
}

// New compiles a schema into a change-event stream.
func New(s *schema.Schema, cfg Config) (*Stream, error) {
	if cfg.Locale == "" {
		cfg.Locale = "en_US"
	}
	if cfg.Table == "" {
		cfg.Table = "data"
	}
	if cfg.Key == "" {
		cfg.Key = primaryKeyName(s)
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.Start.IsZero() {
		cfg.Start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if cfg.UpdateRate+cfg.DeleteRate >= 1 {
		return nil, fmt.Errorf("cdc: UpdateRate+DeleteRate must be < 1")
	}
	if cfg.SoftDeleteColumn == "" {
		cfg.SoftDeleteColumn = "deleted_at"
	}
	eng, err := gen.Compile(s, cfg.Locale)
	if err != nil {
		return nil, err
	}
	return &Stream{
		cfg: cfg, eng: eng,
		base:   rng.New(cfg.Seed),
		rnd:    rng.New(cfg.Seed ^ 0x5cd),
		nextTs: cfg.Start,
	}, nil
}

// primaryKeyName picks the schema's PK column, else the first field.
func primaryKeyName(s *schema.Schema) string {
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

// Next produces the next event, or nil when the history is exhausted (which
// only happens if a delete is drawn with no live rows and no inserts remain).
func (s *Stream) Next() *Event {
	// Initial snapshot: emit existing rows as reads before any change.
	if s.seq < s.cfg.Snapshot {
		row := s.newRow()
		s.live = append(s.live, row)
		s.seq++
		return s.event(OpRead, nil, row)
	}

	u := s.rnd.Float64()
	switch {
	case len(s.live) > 0 && u < s.cfg.UpdateRate:
		i := s.rnd.Pick(len(s.live))
		before := s.live[i]
		after := s.mutate(before)
		s.live[i] = after
		s.seq++
		return s.event(OpUpdate, before, after)

	case len(s.live) > 0 && u < s.cfg.UpdateRate+s.cfg.DeleteRate:
		i := s.rnd.Pick(len(s.live))
		before := s.live[i]
		// Remove the row from the mutable pool either way — deleted, hard or
		// soft, it is never touched again.
		s.live = append(s.live[:i], s.live[i+1:]...)
		s.seq++
		if s.cfg.SoftDelete {
			after := make(map[string]any, len(before)+1)
			for k, v := range before {
				after[k] = v
			}
			after[s.cfg.SoftDeleteColumn] = s.nextTs.UTC().Format(time.RFC3339)
			return s.event(OpUpdate, before, after)
		}
		return s.event(OpDelete, before, nil)

	default:
		row := s.newRow()
		s.live = append(s.live, row)
		s.seq++
		return s.event(OpCreate, nil, row)
	}
}

// newRow generates a fresh record from the schema.
func (s *Stream) newRow() map[string]any {
	return s.eng.Record(s.base, s.seq)
}

// mutate returns a copy of row with some non-key fields changed, so an update
// carries a real `before`/`after` pair sharing the same key.
func (s *Stream) mutate(row map[string]any) map[string]any {
	fresh := s.eng.Record(s.base, s.seq+1_000_000)
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = v
	}
	changed := false
	for k := range out {
		if k == s.cfg.Key {
			continue // the key identifies the row and must survive
		}
		if s.rnd.Bool(0.4) && fresh[k] != out[k] {
			out[k] = fresh[k]
			changed = true
		}
	}
	// An update must actually change something. A freshly drawn value can equal
	// the current one (small value sets), so keep drawing until one differs.
	for attempt := 1; !changed && attempt <= 20; attempt++ {
		cand := s.eng.Record(s.base, s.seq+1_000_000*(attempt+1))
		for k := range out {
			if k != s.cfg.Key && cand[k] != out[k] {
				out[k] = cand[k]
				changed = true
				break
			}
		}
	}
	return out
}

func (s *Stream) event(op Op, before, after map[string]any) *Event {
	s.lsn++
	ts := s.nextTs.UnixMilli()
	s.nextTs = s.nextTs.Add(s.cfg.Interval)
	return &Event{
		Op: op, Before: before, After: after, TsMs: ts,
		Source: Source{
			Connector: "synth", Name: s.cfg.Table + "-server",
			Table: s.cfg.Table, TsMs: ts, LSN: s.lsn,
		},
	}
}

// WriteJSONL writes n events as newline-delimited JSON.
func (s *Stream) WriteJSONL(w io.Writer, n int) error {
	enc := json.NewEncoder(w)
	for i := 0; i < n; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	return nil
}
