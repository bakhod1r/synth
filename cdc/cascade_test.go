package cdc

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func parentSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, GoType: "string", PK: true, Params: map[string]string{}},
		{Name: "name", Kind: schema.KindName, GoType: "string", Params: map[string]string{}},
	}}
}

func childSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, GoType: "string", PK: true, Params: map[string]string{}},
		{Name: "order_id", Kind: schema.KindUUID, GoType: "string", Params: map[string]string{}},
		{Name: "sku", Kind: schema.KindLorem, GoType: "string", Params: map[string]string{}},
	}}
}

func newCascade(t *testing.T, cfg CascadeConfig) *CascadeStream {
	t.Helper()
	cfg.ChildFK = "order_id"
	cfg.ParentTable, cfg.ChildTable = "orders", "items"
	s, err := Cascade(parentSchema(), childSchema(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// ChildFK is required — without it there is nothing to cascade along.
func TestCascadeRequiresChildFK(t *testing.T) {
	_, err := Cascade(parentSchema(), childSchema(), CascadeConfig{})
	if err == nil {
		t.Fatal("expected an error when ChildFK is unset")
	}
}

// Every child created references a parent key that exists at the time.
func TestCascadeInsertKeepsIntegrity(t *testing.T) {
	s := newCascade(t, CascadeConfig{ChildrenPerParent: 2})
	liveParents := map[any]bool{}
	for i := 0; i < 30; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Op != OpCreate {
			continue
		}
		if ev.Source.Table == "orders" {
			liveParents[ev.After["id"]] = true
		} else {
			if !liveParents[ev.After["order_id"]] {
				t.Fatalf("child references parent %v which does not exist", ev.After["order_id"])
			}
		}
	}
}

// A parent delete is preceded by a delete of each of its children, and no child
// is touched after its parent is gone. LSNs strictly increase throughout.
func TestCascadeDeleteRemovesChildrenFirst(t *testing.T) {
	s := newCascade(t, CascadeConfig{DeleteRate: 0.4, ChildrenPerParent: 3})

	childrenOf := map[any]map[any]bool{} // parent key -> set of child ids
	deletedParents := map[any]bool{}
	var lastLSN int64
	sawCascade := false

	for i := 0; i < 400; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Source.LSN <= lastLSN {
			t.Fatalf("LSN did not increase: %d after %d", ev.Source.LSN, lastLSN)
		}
		lastLSN = ev.Source.LSN

		switch {
		case ev.Op == OpCreate && ev.Source.Table == "orders":
			childrenOf[ev.After["id"]] = map[any]bool{}
		case ev.Op == OpCreate && ev.Source.Table == "items":
			p := ev.After["order_id"]
			if childrenOf[p] != nil {
				childrenOf[p][ev.After["id"]] = true
			}
		case ev.Op == OpDelete && ev.Source.Table == "items":
			// its parent must not have been deleted yet
			p := ev.Before["order_id"]
			if deletedParents[p] {
				t.Fatalf("child deleted after its parent %v was already gone", p)
			}
			delete(childrenOf[p], ev.Before["id"])
		case ev.Op == OpDelete && ev.Source.Table == "orders":
			p := ev.Before["id"]
			// by now every child of p must already be deleted
			if len(childrenOf[p]) != 0 {
				t.Fatalf("parent %v deleted with %d children still live", p, len(childrenOf[p]))
			}
			deletedParents[p] = true
			sawCascade = true
		}
	}
	if !sawCascade {
		t.Fatal("no parent delete occurred; raise DeleteRate or iterations")
	}
}

// Same seed, identical event sequence.
func TestCascadeDeterministic(t *testing.T) {
	seq := func() []string {
		s := newCascade(t, CascadeConfig{Seed: 7, UpdateRate: 0.3, DeleteRate: 0.2})
		var out []string
		for i := 0; i < 100; i++ {
			ev := s.Next()
			if ev == nil {
				break
			}
			out = append(out, string(ev.Op)+":"+ev.Source.Table)
		}
		return out
	}
	a, b := seq(), seq()
	if len(a) != len(b) {
		t.Fatalf("lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs: %q vs %q", i, a[i], b[i])
		}
	}
}

// An update preserves a row's key and a child's foreign key — an update must
// not move a row or reparent it.
func TestCascadeUpdatePreservesKeys(t *testing.T) {
	s := newCascade(t, CascadeConfig{UpdateRate: 0.6, ChildrenPerParent: 2})
	for i := 0; i < 200; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Op != OpUpdate {
			continue
		}
		if ev.Before["id"] != ev.After["id"] {
			t.Fatalf("update changed the primary key: %v → %v", ev.Before["id"], ev.After["id"])
		}
		if ev.Source.Table == "items" && ev.Before["order_id"] != ev.After["order_id"] {
			t.Fatalf("update reparented a child: %v → %v", ev.Before["order_id"], ev.After["order_id"])
		}
	}
}
