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

// CascadeConfig controls a two-table change stream with referential cascade
// deletes. See Cascade.
type CascadeConfig struct {
	ParentTable, ChildTable string
	// ParentKey / ChildKey are the primary-key columns; each defaults to its
	// schema's PK. ChildFK is the child column holding the parent's key and is
	// required — it is what makes a cascade possible.
	ParentKey, ChildKey, ChildFK string
	// ChildrenPerParent is how many children each new parent is created with.
	// Defaults to 3.
	ChildrenPerParent int
	// UpdateRate and DeleteRate are the per-step probabilities of mutating or
	// deleting instead of inserting; they must sum to < 1.
	UpdateRate, DeleteRate float64
	Seed                   uint64
	Locale                 string
	// Snapshot emits this many initial parents (with their children) as read
	// events before the change stream begins.
	Snapshot int
	Start    time.Time
	Interval time.Duration
}

// CascadeStream is a change stream over a parent table and one child table.
// A parent delete cascades: every child referencing that parent is deleted
// too, children first, so the stream never leaves a dangling reference.
type CascadeStream struct {
	cfg           CascadeConfig
	parentEng     *gen.Engine
	childEng      *gen.Engine
	base, rnd     *rng.Rand
	lsn           int64
	nextTs        time.Time
	seq           int
	snapshotsLeft int
	parents       []map[string]any         // live parent rows
	children      map[any][]map[string]any // parent key -> live children
	pending       []*Event                 // events queued by the current step
}

// Cascade compiles a parent and a child schema into a two-table change stream.
// The child's ChildFK column is filled with an existing parent key, so a child
// always points at a parent that exists; deleting a parent deletes its children
// first, then the parent.
func Cascade(parent, child *schema.Schema, cfg CascadeConfig) (*CascadeStream, error) {
	if cfg.ChildFK == "" {
		return nil, fmt.Errorf("cdc: CascadeConfig.ChildFK is required — it names " +
			"the child column that references the parent")
	}
	if child.FieldByName(cfg.ChildFK) == nil {
		return nil, fmt.Errorf("cdc: child schema has no column %q for ChildFK", cfg.ChildFK)
	}
	if cfg.Locale == "" {
		cfg.Locale = "en_US"
	}
	if cfg.ParentTable == "" {
		cfg.ParentTable = "parent"
	}
	if cfg.ChildTable == "" {
		cfg.ChildTable = "child"
	}
	if cfg.ParentKey == "" {
		cfg.ParentKey = primaryKeyName(parent)
	}
	if cfg.ChildKey == "" {
		cfg.ChildKey = primaryKeyName(child)
	}
	if cfg.ChildrenPerParent <= 0 {
		cfg.ChildrenPerParent = 3
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
	pe, err := gen.Compile(parent, cfg.Locale)
	if err != nil {
		return nil, err
	}
	ce, err := gen.Compile(child, cfg.Locale)
	if err != nil {
		return nil, err
	}
	return &CascadeStream{
		cfg: cfg, parentEng: pe, childEng: ce,
		base:          rng.New(cfg.Seed),
		rnd:           rng.New(cfg.Seed ^ 0xca5cade),
		nextTs:        cfg.Start,
		snapshotsLeft: cfg.Snapshot,
		children:      map[any][]map[string]any{},
	}, nil
}

// Next returns the next event, or nil when the history is exhausted. A single
// insert or cascade delete produces several events; they are queued and handed
// out one per call, so the caller always sees a flat, ordered stream.
func (s *CascadeStream) Next() *Event {
	if len(s.pending) == 0 {
		s.step()
	}
	if len(s.pending) == 0 {
		return nil
	}
	ev := s.pending[0]
	s.pending = s.pending[1:]
	return ev
}

// step performs one action and queues its events into pending.
func (s *CascadeStream) step() {
	if s.snapshotsLeft > 0 {
		s.snapshotsLeft--
		s.insertFamily(OpRead)
		return
	}
	u := s.rnd.Float64()
	switch {
	case len(s.liveRows()) > 0 && u < s.cfg.UpdateRate:
		s.update()
	case len(s.parents) > 0 && u < s.cfg.UpdateRate+s.cfg.DeleteRate:
		s.cascadeDelete()
	default:
		s.insertFamily(OpCreate)
	}
}

// insertFamily creates a parent and its children, queueing the parent event
// first so a consumer sees the parent before the rows that reference it.
func (s *CascadeStream) insertFamily(op Op) {
	parent := s.parentEng.Record(s.base, s.seq)
	s.seq++
	s.parents = append(s.parents, parent)
	key := parent[s.cfg.ParentKey]
	s.queue(op, s.cfg.ParentTable, nil, parent)

	for i := 0; i < s.cfg.ChildrenPerParent; i++ {
		child := s.childEng.Record(s.base, s.seq)
		s.seq++
		child[s.cfg.ChildFK] = key // point the child at its parent
		s.children[key] = append(s.children[key], child)
		s.queue(op, s.cfg.ChildTable, nil, child)
	}
}

// cascadeDelete removes a random parent and all of its live children — children
// first, so the parent is never deleted while a child still references it.
func (s *CascadeStream) cascadeDelete() {
	i := s.rnd.Pick(len(s.parents))
	parent := s.parents[i]
	key := parent[s.cfg.ParentKey]

	for _, child := range s.children[key] {
		s.queue(OpDelete, s.cfg.ChildTable, child, nil)
	}
	delete(s.children, key)

	s.parents = append(s.parents[:i], s.parents[i+1:]...)
	s.queue(OpDelete, s.cfg.ParentTable, parent, nil)
}

// update mutates one random live row — parent or child — and queues its event
// carrying the true before/after pair. The row's key and, for a child, its
// foreign key are preserved: an update must not move a row or reparent it.
func (s *CascadeStream) update() {
	rows := s.liveRows()
	sel := rows[s.rnd.Pick(len(rows))]
	eng := s.parentEng
	keep := map[string]bool{sel.key: true}
	if sel.table == s.cfg.ChildTable {
		eng = s.childEng
		keep[s.cfg.ChildFK] = true
	}

	before := cloneRow(sel.row)
	fresh := eng.Record(s.base, s.seq+1_000_000)
	s.seq++
	for k, v := range fresh {
		if keep[k] {
			continue
		}
		sel.row[k] = v // write the mutation into live state
	}
	s.queue(OpUpdate, sel.table, before, cloneRow(sel.row))
}

// liveRow ties a mutable live row to its table and key column, so update can
// pick uniformly across both tables.
type liveRow struct {
	row   map[string]any
	table string
	key   string
}

func (s *CascadeStream) liveRows() []liveRow {
	var out []liveRow
	for _, p := range s.parents {
		out = append(out, liveRow{p, s.cfg.ParentTable, s.cfg.ParentKey})
	}
	for _, cs := range s.children {
		for _, c := range cs {
			out = append(out, liveRow{c, s.cfg.ChildTable, s.cfg.ChildKey})
		}
	}
	return out
}

// cloneRow copies a row so an event's before/after image is not aliased to the
// mutable live state.
func cloneRow(r map[string]any) map[string]any {
	out := make(map[string]any, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}

// queue appends one event with a fresh LSN and timestamp.
func (s *CascadeStream) queue(op Op, table string, before, after map[string]any) {
	s.lsn++
	ts := s.nextTs.UnixMilli()
	s.nextTs = s.nextTs.Add(s.cfg.Interval)
	s.pending = append(s.pending, &Event{
		Op: op, Before: before, After: after, TsMs: ts,
		Source: Source{
			Connector: "synth", Name: table + "-server",
			Table: table, TsMs: ts, LSN: s.lsn,
		},
	})
}

// WriteJSONL writes n events as newline-delimited JSON.
func (s *CascadeStream) WriteJSONL(w io.Writer, n int) error {
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
