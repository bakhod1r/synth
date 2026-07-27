package cdc

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func softSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindInt, GoType: "int", PK: true, Params: map[string]string{}},
		{Name: "name", Kind: schema.KindName, GoType: "string", Params: map[string]string{}},
	}}
}

// Under soft delete a drawn delete is an update that stamps deleted_at, not an
// op=d that drops the row. A consumer that has to handle both workloads can be
// exercised against either by the flag alone.
func TestSoftDeleteEmitsUpdate(t *testing.T) {
	s, err := New(softSchema(), Config{
		Table: "users", DeleteRate: 0.9, Snapshot: 3, SoftDelete: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var sawSoft bool
	for i := 0; i < 20; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Op == OpDelete {
			t.Fatal("soft delete must not emit op=d")
		}
		if ev.Op == OpUpdate {
			if ev.After["deleted_at"] == nil {
				t.Error("soft-deleted row has no deleted_at in the after image")
			}
			if ev.Before != nil && ev.Before["deleted_at"] != nil {
				t.Error("the before image should predate the deletion")
			}
			sawSoft = true
		}
	}
	if !sawSoft {
		t.Fatal("no soft-delete update was emitted")
	}
}

// A soft-deleted row is logically gone: it must not be updated again.
func TestSoftDeletedRowNotTouchedAgain(t *testing.T) {
	s, err := New(softSchema(), Config{
		Table: "users", UpdateRate: 0.4, DeleteRate: 0.5, Snapshot: 5, SoftDelete: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	deleted := map[any]bool{}
	for i := 0; i < 200; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Op != OpUpdate {
			continue
		}
		id := ev.After["id"]
		if ev.After["deleted_at"] != nil {
			deleted[id] = true
		} else if deleted[id] {
			t.Fatalf("row %v was updated after being soft-deleted", id)
		}
	}
}

// Without the flag, deletes are still hard deletes — the default is unchanged.
func TestHardDeleteStillDefault(t *testing.T) {
	s, err := New(softSchema(), Config{Table: "users", DeleteRate: 0.9, Snapshot: 3})
	if err != nil {
		t.Fatal(err)
	}
	var sawHard bool
	for i := 0; i < 10; i++ {
		ev := s.Next()
		if ev == nil {
			break
		}
		if ev.Op == OpDelete {
			sawHard = true
		}
	}
	if !sawHard {
		t.Fatal("expected a hard delete without --soft-delete")
	}
}
