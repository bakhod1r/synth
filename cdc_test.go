package synth_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/google/uuid"
)

type CDCUser struct {
	ID    uuid.UUID `synth:"pk"`
	Name  string
	Email string
	City  string
}

func collectEvents(t *testing.T, n int, cfg synth.CDCConfig) []synth.CDCEvent {
	t.Helper()
	s, err := synth.CDC[CDCUser](cfg)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.WriteJSONL(&buf, n); err != nil {
		t.Fatal(err)
	}
	var out []synth.CDCEvent
	dec := json.NewDecoder(&buf)
	for dec.More() {
		var ev synth.CDCEvent
		if err := dec.Decode(&ev); err != nil {
			t.Fatal(err)
		}
		out = append(out, ev)
	}
	return out
}

// The history must be coherent: no update or delete of a row that was never
// inserted, and no touching a row after it was deleted.
func TestCDCHistoryIsCoherent(t *testing.T) {
	evs := collectEvents(t, 2000, synth.CDCConfig{
		Table: "users", Seed: 1, Key: "ID",
		UpdateRate: 0.35, DeleteRate: 0.15, Snapshot: 10,
	})
	if len(evs) < 1900 {
		t.Fatalf("only %d events", len(evs))
	}
	live := map[string]bool{}
	deleted := map[string]bool{}
	ops := map[string]int{}
	for i, ev := range evs {
		ops[string(ev.Op)]++
		switch ev.Op {
		case "r", "c":
			id := key(t, ev.After)
			if live[id] {
				t.Fatalf("event %d: inserted an already-live row", i)
			}
			live[id] = true
		case "u":
			id := key(t, ev.After)
			if !live[id] {
				t.Fatalf("event %d: updated a row that is not live", i)
			}
			if deleted[id] {
				t.Fatalf("event %d: updated a deleted row", i)
			}
			// before/after must share the key and differ somewhere.
			if key(t, ev.Before) != id {
				t.Fatalf("event %d: update changed the primary key", i)
			}
			if sameMap(ev.Before, ev.After) {
				t.Fatalf("event %d: update changed nothing", i)
			}
		case "d":
			id := key(t, ev.Before)
			if !live[id] {
				t.Fatalf("event %d: deleted a row that is not live", i)
			}
			if ev.After != nil {
				t.Fatalf("event %d: delete must have no after image", i)
			}
			delete(live, id)
			deleted[id] = true
		default:
			t.Fatalf("event %d: unknown op %q", i, ev.Op)
		}
	}
	// All three operations must actually occur at the configured rates.
	for _, op := range []string{"c", "u", "d", "r"} {
		if ops[op] == 0 {
			t.Fatalf("no %q events generated: %v", op, ops)
		}
	}
	if ops["r"] != 10 {
		t.Fatalf("snapshot produced %d read events, want 10", ops["r"])
	}
}

// Timestamps and LSNs must advance monotonically, like a real log.
func TestCDCLogOrdering(t *testing.T) {
	evs := collectEvents(t, 200, synth.CDCConfig{Table: "users", Seed: 2, Key: "ID", UpdateRate: 0.3, DeleteRate: 0.1})
	for i := 1; i < len(evs); i++ {
		if evs[i].TsMs <= evs[i-1].TsMs {
			t.Fatalf("event %d: timestamp did not advance", i)
		}
		if evs[i].Source.LSN != evs[i-1].Source.LSN+1 {
			t.Fatalf("event %d: LSN not sequential", i)
		}
	}
}

// Same seed, same history.
func TestCDCDeterministic(t *testing.T) {
	cfg := synth.CDCConfig{Table: "users", Seed: 7, Key: "ID", UpdateRate: 0.3, DeleteRate: 0.1}
	a := collectEvents(t, 300, cfg)
	b := collectEvents(t, 300, cfg)
	if len(a) != len(b) {
		t.Fatal("different event counts")
	}
	for i := range a {
		if a[i].Op != b[i].Op || key2(a[i]) != key2(b[i]) {
			t.Fatalf("event %d differs between runs", i)
		}
	}
}

func key(t *testing.T, m map[string]any) string {
	t.Helper()
	if m == nil {
		t.Fatal("nil row image")
	}
	v, ok := m["ID"]
	if !ok {
		t.Fatalf("row has no ID: %v", m)
	}
	return v.(string)
}

func key2(ev synth.CDCEvent) string {
	m := ev.After
	if m == nil {
		m = ev.Before
	}
	if m == nil {
		return ""
	}
	s, _ := m["ID"].(string)
	return s
}

func sameMap(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
