package synth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/bakhod1r/synth"
)

func TestRefValuesEmptyIsNoop(t *testing.T) {
	// Empty values slice is a no-op guard.
	_ = synth.Make[ChildRec](2, synth.WithSeed(1), synth.RefValues("Parent", nil))
}

type hiddenRec struct {
	secret int //nolint
	Name   string
}

type nestedHidden struct {
	secret int //nolint
	City   string
}

type withNestedHidden struct {
	ID  uuid.UUID `synth:"pk"`
	Obj nestedHidden
}

func TestScatterSkipsUnexported(t *testing.T) {
	// Make (not encode) tolerates unexported fields via the CanSet guards.
	_ = synth.Make[hiddenRec](2, synth.WithSeed(1))
	_ = synth.Make[withNestedHidden](2, synth.WithSeed(1))
}

func TestRefWithPKTaggedParent(t *testing.T) {
	// CoverRow has a synth:"pk" field, so extractPK/pkFieldIndex takes the pk
	// branch.
	parents := synth.Make[CoverRow](4, synth.WithSeed(1))
	_ = synth.Make[ChildRec](6, synth.WithSeed(2), synth.Ref(parents, "Parent"))
}

type Coerce struct {
	IntFromFloat int     `synth:"float,min=1,max=9"`
	FloatFromInt float64 `synth:"int,min=1,max=9"`
	StrFromInt   string  `synth:"int,min=1,max=9"`
}

func TestAssignNumericCoercion(t *testing.T) {
	// Kind/Go-type mismatches drive assign's cross-type coercion branches.
	rows := synth.Make[Coerce](5, synth.WithSeed(1))
	if len(rows) != 5 {
		t.Fatalf("coerce rows = %d", len(rows))
	}
	if rows[0].StrFromInt == "" {
		t.Fatal("string field not coerced from int")
	}
}

func TestRateUnpaced(t *testing.T) {
	// PerSecond <= 0 means run unpaced; Total bounds it.
	rs := synth.Rate[CoverRow](synth.RateConfig{Total: 15, Burst: 4}, synth.WithSeed(1))
	if err := rs.Run(context.Background(), func(CoverRow) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestYAMLRefToUnknownFieldErrors(t *testing.T) {
	y, err := synth.YAMLBytes([]byte("name: t\ncount: 3\nfields:\n  a: { kind: int }\n"))
	if err != nil {
		t.Fatal(err)
	}
	// A Ref naming a field the spec does not have is a checked error.
	parents := []CoverRow{{ID: uuid.New()}}
	if _, err := y.GenerateN(3, synth.Ref(parents, "ghostField")); err == nil {
		t.Fatal("ref to unknown field should error")
	}
}
