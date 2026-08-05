package gen

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

func uniqueField(name string, k schema.Kind, mode string) schema.Field {
	f := mkField(name, k, nil)
	f.Unique, f.UniqueMode = true, mode
	return f
}

// A UUID is unique by construction, so it needs neither tracking nor a counter.
func TestCompileUUIDNeedsNoTracking(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{uniqueField("id", schema.KindUUID, "")}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	if e.HasUnique() {
		t.Fatal("uuid should not be tracked")
	}
}

// Tracked unique fields are stateful; counter fields are not, and callers use
// HasUnique to decide whether parallel generation is safe.
func TestHasUniqueIgnoresCounter(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{uniqueField("slug", schema.KindUsername, "counter")}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	if e.HasUnique() {
		t.Fatal("counter mode needs no shared state, so it must not report HasUnique")
	}
}

func TestCompileRejectsUnknownUniqueMode(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{uniqueField("name", schema.KindName, "maybe")}}
	if _, err := Compile(s, "en_US"); err == nil || !strings.Contains(err.Error(), "unknown unique mode") {
		t.Fatalf("got %v", err)
	}
}

// A counter suffix is applied to a rendered value, which a composite field
// does not have.
func TestCompileRejectsCounterOnComposite(t *testing.T) {
	nested := &schema.Schema{Fields: []schema.Field{mkField("x", schema.KindInt, nil)}}
	f := uniqueField("obj", schema.KindObject, "counter")
	f.Nested = nested
	s := &schema.Schema{Fields: []schema.Field{f}}
	if _, err := Compile(s, "en_US"); err == nil || !strings.Contains(err.Error(), "needs a scalar field") {
		t.Fatalf("got %v", err)
	}
}

// Exhaustion is reported once, on the record that hit it, and the run's first
// error is the one kept.
func TestUniqueExhaustionRecorded(t *testing.T) {
	f := uniqueField("status", schema.KindEnum, "")
	f.Choices = []string{"new", "open"}
	s := &schema.Schema{Fields: []schema.Field{f}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	base := rng.New(1)
	for i := 0; i < 20; i++ {
		e.Record(base, i)
	}
	if e.Err() == nil || !strings.Contains(e.Err().Error(), "ran out of unique values") {
		t.Fatalf("got %v", e.Err())
	}
	first := e.Err()
	e.fail(errFake{})
	if e.Err() != first {
		t.Fatal("a later error overwrote the first")
	}
}

type errFake struct{}

func (errFake) Error() string { return "second" }

// A field with room to spare never trips the exhaustion path.
func TestUniqueNoErrorWhenSpaceIsAmple(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{uniqueField("email", schema.KindEmail, "")}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	base := rng.New(2)
	for i := 0; i < 500; i++ {
		e.Record(base, i)
	}
	if e.Err() != nil {
		t.Fatal(e.Err())
	}
}

func TestWithCounter(t *testing.T) {
	cases := []struct {
		name string
		in   any
		seq  int
		want any
	}{
		{"nil takes the index itself", nil, 7, 7},
		{"int", 42, 7, 7},
		{"int64 stays int64", int64(42), 7, int64(7)},
		{"float64 stays float64", 4.2, 7, float64(7)},
		{"email keeps its domain", "ann@mail.com", 7, "ann7@mail.com"},
		{"plain string is suffixed", "ann", 7, "ann_7"},
		{"anything else is rendered", true, 7, "true_7"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := withCounter(c.in, c.seq); got != c.want {
				t.Fatalf("withCounter(%v, %d) = %v, want %v", c.in, c.seq, got, c.want)
			}
		})
	}
}

// Elements of one record's array must not share a counter index either.
func TestElemSeqDistinctWithinRecord(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 100; i++ {
		s := elemSeq(5, i)
		if seen[s] {
			t.Fatalf("elemSeq collision at element %d", i)
		}
		seen[s] = true
	}
	if elemSeq(5, 0) == elemSeq(6, 0) {
		t.Fatal("elemSeq must stay distinct between records")
	}
}

// The counter reaches fields inside a nested object, whose engine is a separate
// compiled sub-engine.
func TestCounterInsideNestedObject(t *testing.T) {
	nested := &schema.Schema{Fields: []schema.Field{uniqueField("slug", schema.KindUsername, "counter")}}
	f := mkField("profile", schema.KindObject, nil)
	f.Nested = nested
	s := &schema.Schema{Fields: []schema.Field{f}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	base := rng.New(3)
	seen := map[any]bool{}
	for i := 0; i < 200; i++ {
		sub, ok := e.Record(base, i)["profile"].(map[string]any)
		if !ok {
			t.Fatal("expected a nested record")
		}
		if seen[sub["slug"]] {
			t.Fatalf("duplicate nested slug %v", sub["slug"])
		}
		seen[sub["slug"]] = true
	}
}
