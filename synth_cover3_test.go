package synth_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bakhod1r/synth"
)

type NestedAddr struct {
	City string
	Zip  string
}

type DeepRec struct {
	ID    uuid.UUID `synth:"pk"`
	Ptr   *int
	Obj   NestedAddr
	Tags  []string
	Count uint32
	Score float64
}

func TestAssignNestedSlicePtrUint(t *testing.T) {
	rows := synth.Make[DeepRec](5, synth.WithSeed(1))
	if len(rows) != 5 {
		t.Fatalf("Make len = %d", len(rows))
	}
	// The pointer field must be populated (assign pointer branch).
	if rows[0].Ptr == nil {
		t.Fatal("pointer field not assigned")
	}
}

type ParentID struct {
	ID   uuid.UUID // no pk tag -> pkFieldIndex uses the "ID" name
	Name string
}

type ParentNoKey struct {
	Foo string // no pk, no ID -> pkFieldIndex falls back to field 0
}

type ChildRec struct {
	ID     uuid.UUID `synth:"pk"`
	Parent uuid.UUID
}

func TestRefVariants(t *testing.T) {
	parents := synth.Make[ParentID](4, synth.WithSeed(1))
	kids := synth.Make[ChildRec](6, synth.WithSeed(2),
		synth.Ref(parents, "Parent", synth.OneToMany(1, 3)))
	if len(kids) != 6 {
		t.Fatalf("ref kids = %d", len(kids))
	}
	// Parent with no key column at all still extracts (field 0).
	pnk := synth.Make[ParentNoKey](2, synth.WithSeed(1))
	_ = synth.Make[ChildRec](3, synth.WithSeed(3), synth.Ref(pnk, "Parent"))

	// RefValues with an explicit value set.
	_ = synth.Make[ChildRec](3, synth.WithSeed(4),
		synth.RefValues("Parent", []any{uuid.New(), uuid.New()}))
}

func TestStreamToJSONL(t *testing.T) {
	if err := synth.Stream[DeepRec](3, synth.WithSeed(1)).
		ToJSONL(filepath.Join(t.TempDir(), "s.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratorAmountBounds(t *testing.T) {
	g := synth.New()
	// Zero and negative arguments drive itoa's other branches.
	_ = g.Amount(0, 0)
	_ = g.Amount(-50, 50)
}

func TestRateStreamJitterAndError(t *testing.T) {
	// Fast pacing with jitter drives the jitter and behind-schedule branches.
	rs := synth.Rate[DeepRec](synth.RateConfig{PerSecond: 1e6, Burst: 2, Jitter: 0.5, Total: 20},
		synth.WithSeed(1))
	if err := rs.Run(context.Background(), func(DeepRec) error { return nil }); err != nil {
		t.Fatal(err)
	}
	// A non-struct type surfaces the engine error.
	bad := synth.Rate[int](synth.RateConfig{Total: 1})
	if err := bad.Run(context.Background(), func(int) error { return nil }); err == nil {
		t.Fatal("non-struct rate should error")
	}
}

func TestRateStreamPacedCancel(t *testing.T) {
	// A slow rate plus an already-cancelled context exits via ctx.Err().
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	rs := synth.Rate[DeepRec](synth.RateConfig{PerSecond: 5, Burst: 1}, synth.WithSeed(1))
	_ = rs.Run(ctx, func(DeepRec) error { return nil })
}
