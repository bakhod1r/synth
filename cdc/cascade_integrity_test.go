package cdc

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/schema"
)

func integParent() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "name", Kind: schema.KindName, Params: map[string]string{}},
	}}
}

func integChild() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "parent_id", Kind: schema.KindUUID, Params: map[string]string{}},
		{Name: "total", Kind: schema.KindAmount, Params: map[string]string{}},
	}}
}

func newIntegCascade(t *testing.T, cfg CascadeConfig) *CascadeStream {
	t.Helper()
	s, err := Cascade(integParent(), integChild(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The promise of a cascade stream: replaying it never leaves a child pointing
// at a parent that is not there — not for one event, not at any point.
func TestCascadeNeverDanglesAReference(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{
		ParentTable: "users", ChildTable: "orders", ChildFK: "parent_id",
		ChildrenPerParent: 3, UpdateRate: 0.3, DeleteRate: 0.2,
		Seed: 99, Snapshot: 5,
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Interval: time.Second,
	})

	parents := map[any]bool{}
	children := map[any]any{} // child key -> parent key
	var lastLSN int64
	var lastTs int64

	for i := 0; i < 3000; i++ {
		ev := s.Next()
		if ev == nil {
			t.Fatalf("the stream ended after %d events", i)
		}
		if ev.Source.LSN <= lastLSN {
			t.Fatalf("event %d has a non-increasing LSN: %d after %d", i, ev.Source.LSN, lastLSN)
		}
		lastLSN = ev.Source.LSN
		if ev.TsMs < lastTs {
			t.Fatalf("event %d goes back in time", i)
		}
		lastTs = ev.TsMs

		switch ev.Source.Table {
		case "users":
			switch ev.Op {
			case OpRead, OpCreate:
				parents[ev.After["id"]] = true
			case OpUpdate:
				if !parents[ev.After["id"]] {
					t.Fatalf("event %d updates an unknown parent", i)
				}
				if ev.Before["id"] != ev.After["id"] {
					t.Fatalf("event %d moved a parent's primary key", i)
				}
			case OpDelete:
				key := ev.Before["id"]
				if !parents[key] {
					t.Fatalf("event %d deletes an unknown parent", i)
				}
				for ck, pk := range children {
					if pk == key {
						t.Fatalf("event %d deletes parent %v while child %v still references it", i, key, ck)
					}
				}
				delete(parents, key)
			}
		case "orders":
			switch ev.Op {
			case OpRead, OpCreate:
				fk := ev.After["parent_id"]
				if !parents[fk] {
					t.Fatalf("event %d inserts a child before its parent %v", i, fk)
				}
				children[ev.After["id"]] = fk
			case OpUpdate:
				key := ev.After["id"]
				if _, ok := children[key]; !ok {
					t.Fatalf("event %d updates an unknown child", i)
				}
				if ev.Before["parent_id"] != ev.After["parent_id"] {
					t.Fatalf("event %d reparented a child", i)
				}
				if ev.Before["id"] != ev.After["id"] {
					t.Fatalf("event %d moved a child's primary key", i)
				}
			case OpDelete:
				key := ev.Before["id"]
				if _, ok := children[key]; !ok {
					t.Fatalf("event %d deletes an unknown child", i)
				}
				delete(children, key)
			}
		default:
			t.Fatalf("event %d names an unexpected table %q", i, ev.Source.Table)
		}
	}
}

// The before image must be the row as it actually was, not a fresh sample: an
// update carries the true previous values.
func TestCascadeUpdateCarriesTheTrueBeforeImage(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{
		ParentTable: "users", ChildTable: "orders", ChildFK: "parent_id",
		UpdateRate: 0.6, Seed: 4, Snapshot: 3,
	})
	state := map[any]map[string]any{}
	updates := 0
	for i := 0; i < 800; i++ {
		ev := s.Next()
		switch ev.Op {
		case OpRead, OpCreate:
			state[ev.After["id"]] = ev.After
		case OpUpdate:
			prev := state[ev.After["id"]]
			for k, v := range ev.Before {
				if prev[k] != v {
					t.Fatalf("before image disagrees with the live row on %q: %v vs %v", k, v, prev[k])
				}
			}
			state[ev.After["id"]] = ev.After
			updates++
		case OpDelete:
			delete(state, ev.Before["id"])
		}
	}
	if updates == 0 {
		t.Fatal("no update was produced at a 0.6 update rate")
	}
}

// An emitted event is a snapshot, not a window onto live state: an insert event
// held by a consumer must not change when the row is later updated.
func TestCascadeEventsAreImmutableSnapshots(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{
		ParentTable: "users", ChildTable: "orders", ChildFK: "parent_id",
		UpdateRate: 0.9, Seed: 8, Snapshot: 2, ChildrenPerParent: 1,
	})

	kept := map[any]map[string]any{}   // key -> the insert event's after image
	frozen := map[any]map[string]any{} // key -> a copy taken at insert time
	for i := 0; i < 400; i++ {
		ev := s.Next()
		switch ev.Op {
		case OpRead, OpCreate:
			key := ev.After["id"]
			kept[key] = ev.After
			frozen[key] = cloneRow(ev.After)
		}
	}
	for key, held := range kept {
		for k, v := range frozen[key] {
			if held[k] != v {
				t.Fatalf("the insert event for %v changed after the fact on %q: %v became %v",
					key, k, v, held[k])
			}
		}
	}
}

func TestCascadeSnapshotEventsComeFirst(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{ChildFK: "parent_id", ChildrenPerParent: 2, Snapshot: 4, Seed: 1})
	// 4 families of 1 parent + 2 children = 12 read events.
	for i := 0; i < 12; i++ {
		ev := s.Next()
		if ev.Op != OpRead {
			t.Fatalf("event %d is %q, want a snapshot read", i, ev.Op)
		}
	}
	if ev := s.Next(); ev.Op != OpCreate {
		t.Fatalf("the first post-snapshot event is %q, want create", ev.Op)
	}
}

func TestCascadeDefaultsFillThemselvesIn(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{ChildFK: "parent_id", Seed: 2})
	ev := s.Next()
	if ev.Source.Table != "parent" {
		t.Fatalf("default parent table = %q", ev.Source.Table)
	}
	if s.cfg.ChildTable != "child" || s.cfg.ChildrenPerParent != 3 || s.cfg.Interval != time.Second {
		t.Fatalf("defaults not applied: %+v", s.cfg)
	}
	if s.cfg.ParentKey != "id" || s.cfg.ChildKey != "id" {
		t.Fatalf("keys not taken from the schemas: %q / %q", s.cfg.ParentKey, s.cfg.ChildKey)
	}
	if s.cfg.Start.IsZero() {
		t.Fatal("Start left at the zero time")
	}
}

func TestCascadeConfigErrors(t *testing.T) {
	if _, err := Cascade(integParent(), integChild(), CascadeConfig{}); err == nil {
		t.Error("a missing ChildFK should be refused")
	}
	if _, err := Cascade(integParent(), integChild(), CascadeConfig{ChildFK: "nope"}); err == nil {
		t.Error("a ChildFK the child schema lacks should be refused")
	}
	_, err := Cascade(integParent(), integChild(), CascadeConfig{
		ChildFK: "parent_id", UpdateRate: 0.7, DeleteRate: 0.5,
	})
	if err == nil || !strings.Contains(err.Error(), "< 1") {
		t.Errorf("rates summing above 1 should be refused, got %v", err)
	}

	bad := &schema.Schema{Fields: []schema.Field{{Name: "a", Kind: "no-such-kind", Params: map[string]string{}}}}
	if _, err := Cascade(bad, integChild(), CascadeConfig{ChildFK: "parent_id"}); err == nil {
		t.Error("a bad parent schema should surface the compile error")
	}
	badChild := &schema.Schema{Fields: []schema.Field{
		{Name: "parent_id", Kind: schema.KindUUID, Params: map[string]string{}},
		{Name: "a", Kind: "no-such-kind", Params: map[string]string{}},
	}}
	if _, err := Cascade(integParent(), badChild, CascadeConfig{ChildFK: "parent_id"}); err == nil {
		t.Error("a bad child schema should surface the compile error")
	}
}

type breakingWriter struct{ ok int }

var errWrite = errors.New("pipe closed")

func (b *breakingWriter) Write(p []byte) (int, error) {
	if b.ok <= 0 {
		return 0, errWrite
	}
	b.ok--
	return len(p), nil
}

func TestCascadeWriteJSONLSurfacesWriteError(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{ChildFK: "parent_id", Seed: 3})
	if err := s.WriteJSONL(&breakingWriter{ok: 2}, 50); !errors.Is(err, errWrite) {
		t.Fatalf("err = %v, want %v", err, errWrite)
	}
}

func TestCascadeWriteJSONLLineCount(t *testing.T) {
	s := newIntegCascade(t, CascadeConfig{ChildFK: "parent_id", Seed: 3})
	var b strings.Builder
	if err := s.WriteJSONL(&b, 40); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(b.String(), "\n"); n != 40 {
		t.Fatalf("wrote %d lines, want 40", n)
	}
}
