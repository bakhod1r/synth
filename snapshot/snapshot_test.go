package snapshot_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/bakhod1r/synth/cdc"
	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/snapshot"
)

func orderSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "customer", Kind: schema.KindName, Params: map[string]string{}},
		{Name: "amount", Kind: schema.KindAmount, Params: map[string]string{"min": "1", "max": "1000"}},
		{Name: "status", Kind: schema.KindOrderStatus, Params: map[string]string{}},
		{Name: "created_at", Kind: schema.KindTime, Params: map[string]string{}},
		{Name: "updated_at", Kind: schema.KindTime, Params: map[string]string{}},
	}}
}

var (
	start = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t0    = time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	t1    = time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
)

func newTimeline(t *testing.T, cfg snapshot.Config) *snapshot.Timeline {
	t.Helper()
	tl, err := snapshot.New(orderSchema(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tl
}

// This is the whole contract: the event log between two instants must
// reconstruct the later snapshot from the earlier one, exactly.
func TestEventsReconstructLaterSnapshot(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{
		Table: "orders", Rows: 500, Start: start,
		Churn: 2.0, DeleteFrac: 0.2, Seed: 42,
	})

	before := tl.At(t0)
	after := tl.At(t1)
	events := tl.Between(t0, t1)
	if len(events) == 0 {
		t.Fatal("no events between the two instants — the test proves nothing")
	}

	got := tl.Apply(before, events)
	if !equalByKey(got, after, tl.Key()) {
		t.Fatalf("replaying %d events over %d rows gave %d rows, want %d",
			len(events), len(before), len(got), len(after))
	}
}

// equalByKey compares two row sets ignoring order.
func equalByKey(a, b []map[string]any, key string) bool {
	if len(a) != len(b) {
		return false
	}
	index := make(map[string]map[string]any, len(b))
	for _, r := range b {
		index[str(r[key])] = r
	}
	for _, r := range a {
		other, ok := index[str(r[key])]
		if !ok || !reflect.DeepEqual(r, other) {
			return false
		}
	}
	return true
}

// str renders a key value. Keys are not always strings — a UUID column may
// carry a typed value — so this must not depend on the concrete type.
func str(v any) string { return fmt.Sprint(v) }

// Consecutive ranges must tile: t0→tm→t1 must equal t0→t1.
func TestConsecutiveRangesTile(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{
		Table: "orders", Rows: 300, Start: start,
		Churn: 3.0, DeleteFrac: 0.15, Seed: 7,
	})
	mid := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	stepwise := tl.Apply(tl.Apply(tl.At(t0), tl.Between(t0, mid)), tl.Between(mid, t1))
	direct := tl.At(t1)
	if !equalByKey(stepwise, direct, tl.Key()) {
		t.Fatalf("two-step replay gave %d rows, one-step gave %d", len(stepwise), len(direct))
	}
}

// The same instant and seed must always give the same output.
func TestSnapshotIsDeterministic(t *testing.T) {
	cfg := snapshot.Config{Table: "orders", Rows: 200, Start: start, Churn: 2, Seed: 7}
	a := newTimeline(t, cfg).At(t1)
	b := newTimeline(t, cfg).At(t1)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two timelines with the same seed disagree at the same instant")
	}
}

// A different seed must give a different world, or the seed does nothing.
func TestSeedChangesTheWorld(t *testing.T) {
	a := newTimeline(t, snapshot.Config{Rows: 200, Start: start, Churn: 2, Seed: 1}).At(t1)
	b := newTimeline(t, snapshot.Config{Rows: 200, Start: start, Churn: 2, Seed: 2}).At(t1)
	if reflect.DeepEqual(a, b) {
		t.Fatal("changing the seed changed nothing")
	}
}

// A snapshot before the table existed is empty, not an error.
func TestSnapshotBeforeStartIsEmpty(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 100, Start: start, Churn: 1, Seed: 1})
	if got := tl.At(start.AddDate(-1, 0, 0)); len(got) != 0 {
		t.Fatalf("got %d rows before the table existed", len(got))
	}
}

// A row must never appear in a snapshot taken before it was created.
func TestSnapshotExcludesFutureRows(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 400, Start: start, Churn: 2, Seed: 5})
	for _, r := range tl.At(t0) {
		created, err := time.Parse(time.RFC3339, r["created_at"].(string))
		if err != nil {
			t.Fatalf("created_at is not a timestamp: %v", r["created_at"])
		}
		if created.After(t0) {
			t.Fatalf("row created at %s appears in the snapshot at %s", created, t0)
		}
	}
}

// updated_at must never precede created_at, at any version.
func TestUpdatedNeverPrecedesCreated(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 400, Start: start, Churn: 4, Seed: 11})
	for _, r := range tl.At(t1) {
		created, _ := time.Parse(time.RFC3339, r["created_at"].(string))
		updated, _ := time.Parse(time.RFC3339, r["updated_at"].(string))
		if updated.Before(created) {
			t.Fatalf("updated_at %s precedes created_at %s", updated, created)
		}
	}
}

// A row keeps its identity across updates: the key must not change.
func TestKeyIsStableAcrossVersions(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 200, Start: start, Churn: 5, Seed: 3})
	for _, ev := range tl.Between(start, t1) {
		if ev.Op != cdc.OpUpdate {
			continue
		}
		if ev.Before[tl.Key()] != ev.After[tl.Key()] {
			t.Fatalf("update changed the key: %v -> %v", ev.Before[tl.Key()], ev.After[tl.Key()])
		}
		if reflect.DeepEqual(ev.Before, ev.After) {
			t.Fatalf("update changed nothing: %v", ev.After)
		}
	}
}

// An update's before image must be the row's real previous state, or a
// consumer reconstructing history from the log gets the wrong answer.
func TestUpdateCarriesTrueBeforeImage(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 150, Start: start, Churn: 3, Seed: 9})
	events := tl.Between(start, t1)

	current := map[string]map[string]any{}
	for _, ev := range events {
		switch ev.Op {
		case cdc.OpCreate:
			current[str(ev.After[tl.Key()])] = ev.After
		case cdc.OpUpdate:
			k := str(ev.After[tl.Key()])
			if prev, ok := current[k]; ok && !reflect.DeepEqual(prev, ev.Before) {
				t.Fatalf("before image does not match the previous state for %s", k)
			}
			current[k] = ev.After
		case cdc.OpDelete:
			delete(current, str(ev.Before[tl.Key()]))
		}
	}
}

// A deleted row must never be touched again.
func TestDeletedRowsStayDeleted(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{
		Rows: 300, Start: start, Churn: 3, DeleteFrac: 0.5, Seed: 13,
	})
	dead := map[string]bool{}
	for _, ev := range tl.Between(start, start.AddDate(1, 0, 0)) {
		switch ev.Op {
		case cdc.OpDelete:
			dead[str(ev.Before[tl.Key()])] = true
		default:
			if dead[str(ev.After[tl.Key()])] {
				t.Fatalf("row %s changed after being deleted", str(ev.After[tl.Key()]))
			}
		}
	}
	if len(dead) == 0 {
		t.Fatal("no deletions occurred — the test proves nothing")
	}
}

// Events must be ordered, and their sequence numbers must be monotonic.
func TestEventsAreOrdered(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 200, Start: start, Churn: 3, DeleteFrac: 0.1, Seed: 4})
	events := tl.Between(start, t1)
	for i := 1; i < len(events); i++ {
		if events[i].TsMs < events[i-1].TsMs {
			t.Fatalf("event %d is older than the one before it", i)
		}
		if events[i].Source.LSN <= events[i-1].Source.LSN {
			t.Fatalf("LSN is not monotonic at event %d", i)
		}
	}
}

// Asking for a distant instant must not cost more than a near one: the whole
// point of deriving each row's life is that nothing is simulated forward.
func TestDistantInstantIsNotMoreWork(t *testing.T) {
	// No deletions, so both instants return the same 5000 rows. The only
	// difference is how far past the window the second one asks — which must
	// cost nothing, because no history is replayed to get there.
	tl := newTimeline(t, snapshot.Config{Rows: 5000, Start: start, Churn: 5, Seed: 1})
	endOfWindow := start.AddDate(1, 0, 0)

	near := time.Now()
	nearRows := tl.At(endOfWindow)
	nearCost := time.Since(near)

	far := time.Now()
	farRows := tl.At(start.AddDate(100, 0, 0))
	farCost := time.Since(far)

	if len(nearRows) != len(farRows) {
		t.Fatalf("the two instants return different row counts (%d vs %d); "+
			"this test cannot compare their cost", len(nearRows), len(farRows))
	}
	// A forward simulation would be orders of magnitude slower for the distant
	// instant. Allow a wide margin so this measures the algorithm, not the
	// scheduler.
	if farCost > 20*nearCost+10*time.Millisecond {
		t.Fatalf("a snapshot 100 years out took %v vs %v at the window edge — "+
			"is it simulating forward?", farCost, nearCost)
	}
}

// Zero rows is a valid, empty table.
func TestZeroRows(t *testing.T) {
	tl := newTimeline(t, snapshot.Config{Rows: 0, Start: start, Seed: 1})
	if got := tl.At(t1); len(got) != 0 {
		t.Fatalf("got %d rows from an empty table", len(got))
	}
	if got := tl.Between(start, t1); len(got) != 0 {
		t.Fatalf("got %d events from an empty table", len(got))
	}
}

// An invalid delete fraction is a configuration error, not silently clamped.
func TestRejectsInvalidDeleteFraction(t *testing.T) {
	if _, err := snapshot.New(orderSchema(), snapshot.Config{Rows: 1, DeleteFrac: 1.5}); err == nil {
		t.Fatal("expected an error for a delete fraction above 1")
	}
}
