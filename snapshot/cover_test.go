package snapshot_test

import (
	"testing"
	"time"

	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/snapshot"
)

func TestNewValidation(t *testing.T) {
	if _, err := snapshot.New(orderSchema(), snapshot.Config{Rows: -1}); err == nil {
		t.Fatal("negative Rows should error")
	}
	if _, err := snapshot.New(orderSchema(), snapshot.Config{Rows: 1, DeleteFrac: 1.5}); err == nil {
		t.Fatal("out-of-range DeleteFrac should error")
	}
	// A schema the engine cannot compile surfaces Compile's error.
	bad := &schema.Schema{Fields: []schema.Field{
		{Name: "x", Kind: schema.Kind("nope"), Params: map[string]string{}},
	}}
	if _, err := snapshot.New(bad, snapshot.Config{Rows: 1}); err == nil {
		t.Fatal("uncompilable schema should error")
	}
}

func TestPrimaryKeyEmptySchema(t *testing.T) {
	// An empty schema falls all the way back to "id".
	if _, err := snapshot.New(&schema.Schema{}, snapshot.Config{Rows: 3, Start: start}); err != nil {
		t.Fatal(err)
	}
}

func TestPrimaryKeyFallbacks(t *testing.T) {
	// No PK field: Key defaults to the first column.
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "col1", Kind: schema.KindName, Params: map[string]string{}},
		{Name: "made", Kind: schema.KindTime, Params: map[string]string{}},
	}}
	if _, err := snapshot.New(s, snapshot.Config{Rows: 5, Start: start}); err != nil {
		t.Fatal(err)
	}
}

func TestAmbiguousTimeColumnLeftAlone(t *testing.T) {
	// Two columns both containing "creat" make the created-column match
	// ambiguous, which must not panic.
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "created_at", Kind: schema.KindTime, Params: map[string]string{}},
		{Name: "recreated_at", Kind: schema.KindTime, Params: map[string]string{}},
	}}
	tl, err := snapshot.New(s, snapshot.Config{Rows: 10, Start: start, Window: 90 * 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	_ = tl.At(start.Add(30 * 24 * time.Hour))
}

func TestHighChurnHitsWindowEnd(t *testing.T) {
	// A large churn with a short window drives the remaining<=0 break.
	for seed := uint64(0); seed < 30; seed++ {
		tl := newTimeline(t, snapshot.Config{Rows: 200, Start: start,
			Window: time.Hour, Churn: 200, Seed: seed})
		_ = tl.At(start.Add(30 * time.Minute))
	}
}
