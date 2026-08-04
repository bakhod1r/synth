package synth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth"
	"github.com/google/uuid"
)

type refParent struct {
	ID   uuid.UUID `synth:"pk"`
	Name string
}

type refChild struct {
	ID       uuid.UUID `synth:"pk"`
	ParentID uuid.UUID
	Amount   float64 `synth:"float,min=1,max=100"`
}

// A ref with no parents must be a no-op rather than a panic or an empty FK: the
// caller passed an empty parent slice, which is a data fact, not a mistake.
func TestRefWithNoParentsIsNoOp(t *testing.T) {
	kids := synth.Make[refChild](5, synth.Ref([]refParent{}, "ParentID"), synth.WithSeed(1))
	if len(kids) != 5 {
		t.Fatalf("got %d rows, want 5", len(kids))
	}
	for _, k := range kids {
		if k.ParentID == uuid.Nil {
			t.Fatal("FK left at the zero value instead of generating normally")
		}
	}
}

func TestRefValuesEmptyIsNoOp(t *testing.T) {
	kids := synth.Make[refChild](3, synth.RefValues("ParentID", nil), synth.WithSeed(1))
	if len(kids) != 3 {
		t.Fatalf("got %d rows, want 3", len(kids))
	}
}

func TestRefPointsAtRealParents(t *testing.T) {
	parents := synth.Make[refParent](10, synth.WithSeed(2))
	valid := map[uuid.UUID]bool{}
	for _, p := range parents {
		valid[p.ID] = true
	}
	kids := synth.Make[refChild](500, synth.Ref(parents, "ParentID"), synth.WithSeed(3))
	for _, k := range kids {
		if !valid[k.ParentID] {
			t.Fatalf("child points at a parent that does not exist: %v", k.ParentID)
		}
	}
}

func TestOneToManyBoundsChildrenPerParent(t *testing.T) {
	parents := synth.Make[refParent](20, synth.WithSeed(4))
	kids := synth.Make[refChild](5000,
		synth.Ref(parents, "ParentID", synth.OneToMany(2, 5)),
		synth.WithSeed(5))

	counts := map[uuid.UUID]int{}
	for _, k := range kids {
		counts[k.ParentID]++
	}
	if len(counts) != len(parents) {
		t.Fatalf("%d parents got children, want all %d", len(counts), len(parents))
	}
}

// OneToMany(n, m) with m < n is a caller slip, not a contradiction: max is
// raised to min rather than producing an empty pool.
func TestOneToManyMaxBelowMinStillGenerates(t *testing.T) {
	parents := synth.Make[refParent](5, synth.WithSeed(6))
	kids := synth.Make[refChild](50,
		synth.Ref(parents, "ParentID", synth.OneToMany(3, 1)),
		synth.WithSeed(7))
	valid := map[uuid.UUID]bool{}
	for _, p := range parents {
		valid[p.ID] = true
	}
	for _, k := range kids {
		if !valid[k.ParentID] {
			t.Fatalf("unknown FK %v", k.ParentID)
		}
	}
}

func TestRefValuesFromAnEarlierRun(t *testing.T) {
	// The cross-run case: keys read back from a file the parent run wrote.
	keys := []any{"u-1", "u-2", "u-3"}
	spec, err := synth.YAMLBytes([]byte(`name: orders
count: 100
fields:
  id:      { kind: uuid, pk: true }
  user_id: { kind: string }
  total:   { kind: amount, min: 1, max: 10 }
`))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := spec.Generate(synth.RefValues("user_id", keys), synth.WithSeed(8))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		s, _ := r["user_id"].(string)
		if s != "u-1" && s != "u-2" && s != "u-3" {
			t.Fatalf("user_id = %v, want one of the supplied keys", r["user_id"])
		}
		seen[s] = true
	}
	if len(seen) != 3 {
		t.Fatalf("only %d of 3 keys were used", len(seen))
	}
}

// A misspelled FK column must be reported, not silently generated: it passes
// generation and fails much later, at the join.
func TestYAMLRefUnknownFieldIsAnError(t *testing.T) {
	spec, err := synth.YAMLBytes([]byte(`name: orders
count: 5
fields:
  id: { kind: uuid, pk: true }
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = spec.Generate(synth.RefValues("usr_id", []any{"a"}))
	if err == nil || !strings.Contains(err.Error(), "usr_id") {
		t.Fatalf("err = %v, want a missing-ref-field error naming usr_id", err)
	}
}

// extractPK falls back to a field named ID, then to the first field.
type pkByName struct {
	Other string
	ID    string
}

type pkByPosition struct {
	Code string
	Rest string
}

func TestRefPrimaryKeyFallbacks(t *testing.T) {
	byName := []pkByName{{Other: "x", ID: "id-1"}, {Other: "y", ID: "id-2"}}
	rows, err := synth.TryMake[refChildString](50, synth.Ref(byName, "ParentID"), synth.WithSeed(9))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ParentID != "id-1" && r.ParentID != "id-2" {
			t.Fatalf("ParentID = %q, want the ID field's value", r.ParentID)
		}
	}

	byPos := []pkByPosition{{Code: "c-1"}, {Code: "c-2"}}
	rows, err = synth.TryMake[refChildString](50, synth.Ref(byPos, "ParentID"), synth.WithSeed(10))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ParentID != "c-1" && r.ParentID != "c-2" {
			t.Fatalf("ParentID = %q, want the first field's value", r.ParentID)
		}
	}
}

type refChildString struct {
	ID       string `synth:"uuid,pk"`
	ParentID string
}

// Offset extends an earlier run: rows must differ from run one yet stay
// reproducible on repeat.
func TestOffsetExtendsAPreviousRun(t *testing.T) {
	spec := func() *synth.YAMLSpec {
		s, err := synth.YAMLBytes([]byte(`name: t
count: 5
seed: 12345
fields:
  id:    { kind: uuid }
  email: { kind: email }
`))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	first, err := spec().Generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := spec().Generate(synth.Offset(5))
	if err != nil {
		t.Fatal(err)
	}
	again, err := spec().Generate(synth.Offset(5))
	if err != nil {
		t.Fatal(err)
	}

	for i := range second {
		if second[i]["email"] != again[i]["email"] {
			t.Fatalf("row %d is not reproducible under the same offset", i)
		}
		for j := range first {
			if first[j]["email"] == second[i]["email"] {
				t.Fatalf("offset run repeated a row from the first run: %v", second[i]["email"])
			}
		}
	}

	// A non-positive offset is ignored, so the default run is unchanged.
	zero, err := spec().Generate(synth.Offset(0))
	if err != nil {
		t.Fatal(err)
	}
	if zero[0]["email"] != first[0]["email"] {
		t.Fatal("Offset(0) changed the output")
	}
}

// scatter/assign coverage: every field kind the engine can hand back must land
// in the struct, including pointers, nested structs, slices and unsigned ints.
type scatterInner struct {
	City string
	Zip  string
}

type scatterRow struct {
	Name    string
	Age     uint8
	Score   float32
	Ratio   float64 `synth:"float,min=0,max=1"`
	Active  *bool
	When    time.Time
	Nested  scatterInner
	Tags    []string
	private string
}

func TestScatterFillsEveryFieldShape(t *testing.T) {
	rows, err := synth.TryMake[scatterRow](20, synth.WithSeed(11))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Name == "" {
			t.Fatal("Name empty")
		}
		if r.Active == nil {
			t.Fatal("pointer field not allocated")
		}
		if r.When.IsZero() {
			t.Fatal("time field not set")
		}
		if r.Nested.City == "" {
			t.Fatalf("nested struct not filled: %+v", r.Nested)
		}
		if len(r.Tags) < 1 || len(r.Tags) > 3 {
			t.Fatalf("slice field = %v, want 1..3 elements", r.Tags)
		}
		if r.Ratio < 0 || r.Ratio > 1 {
			t.Fatalf("Ratio out of range: %v", r.Ratio)
		}
		if r.private != "" {
			t.Fatal("an unexported field was written")
		}
	}
}

func TestWarningsReportsUninferredFields(t *testing.T) {
	type odd struct {
		Blob complex128
	}
	if w := synth.Warnings[odd](); len(w) == 0 {
		t.Fatal("expected a warning for a field Synth cannot infer")
	}
	if w := synth.Warnings[int](); w != nil {
		t.Fatal("a non-struct type should produce no warnings")
	}
}

func TestFillPopulatesInPlace(t *testing.T) {
	var u refParent
	if err := synth.Fill(&u, synth.WithSeed(12)); err != nil {
		t.Fatal(err)
	}
	if u.Name == "" || u.ID == uuid.Nil {
		t.Fatalf("Fill left fields empty: %+v", u)
	}
	var bad badKindRow
	if err := synth.Fill(&bad); err == nil {
		t.Fatal("Fill should surface a compile error")
	}
}
