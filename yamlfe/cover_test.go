package yamlfe_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/constraint"
	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/yamlfe"
)

func TestRenderRoundTripCover(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "email", Kind: schema.KindEmail, Unique: true, From: "name", Match: "city",
			Params: map[string]string{"min": "1", "max": "9"}},
		{Name: "status", Kind: schema.KindEnum, Choices: []string{"a", "b"}, Weights: []float64{1, 2},
			Params: map[string]string{}},
	}}
	cs := []constraint.Constraint{
		{Kind: constraint.Ordering, Left: "a", Right: "b", Support: 10},
		{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c", Support: 5},
		{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t", Exclusive: true, Support: 3},
		{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 9, Support: 8},
		{Kind: constraint.Kind("unknown")}, // default branch: skipped
	}
	out, err := yamlfe.Render(s, nil, "users", 100, cs) // nil order -> default
	if err != nil {
		t.Fatal(err)
	}
	sp, err := yamlfe.Parse(out)
	if err != nil {
		t.Fatalf("round-trip parse: %v\n%s", err, out)
	}
	if sp.Schema.FieldByName("status") == nil {
		t.Fatal("status lost in round-trip")
	}
	if len(sp.Constraints) != 4 {
		t.Fatalf("constraints = %d, want 4", len(sp.Constraints))
	}
}

func TestRenderEdgeCases(t *testing.T) {
	if _, err := yamlfe.Render(nil, nil, "", 0, nil); err == nil {
		t.Fatal("nil schema should error")
	}
	s := &schema.Schema{Fields: []schema.Field{{Name: "a", Kind: schema.KindInt, Params: map[string]string{}}}}
	// order names a column that is not in the schema -> skipped.
	if _, err := yamlfe.Render(s, []string{"a", "ghost"}, "", 0, nil); err != nil {
		t.Fatal(err)
	}
	// A name with control characters, separators and invalid UTF-8 renders.
	weird := &schema.Schema{Fields: []schema.Field{
		{Name: "x\r\ty  " + string([]byte{0xff}), Kind: schema.KindLorem, Params: map[string]string{}},
	}}
	if _, err := yamlfe.Render(weird, nil, "", 0, nil); err != nil {
		t.Fatal(err)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := yamlfe.Parse([]byte("a:\n\t- b\n  c: [\n")); err == nil {
		t.Fatal("invalid yaml should error")
	}
	if _, err := yamlfe.Parse([]byte("fields: [a, b]\n")); err == nil {
		t.Fatal("non-mapping fields should error")
	}
	if _, err := yamlfe.Parse([]byte("fields:\n  a: notamapping\n")); err == nil {
		t.Fatal("bad field def should error")
	}
	if _, err := yamlfe.Parse([]byte("fields: {}\nconstraints:\n  - {kind: bogus}\n")); err == nil {
		t.Fatal("bad constraint should error")
	}
}

func TestParseParamTypes(t *testing.T) {
	// int min, date max, string param, float mu/sigma -> setVal/setNum branches.
	spec := `
name: t
fields:
  born: { kind: time, min: 1960-01-01, max: 2000-01-01 }
  n: { kind: int, min: 18, mu: 0.5, sigma: 1.2, dist: normal, gap: 1h }
  p: { kind: password, strength: strong }
`
	sp, err := yamlfe.Parse([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	if sp.Schema.FieldByName("p").Params["strength"] != "strong" {
		t.Fatal("extra param not passed through")
	}
	if !strings.HasPrefix(sp.Schema.FieldByName("born").Params["min"], "1960") {
		t.Fatalf("date bound = %q", sp.Schema.FieldByName("born").Params["min"])
	}
}

func TestToFieldExtraBranches(t *testing.T) {
	// A pk field with no kind defaults to uuid; an int min; a bool min hits the
	// setVal default (fmt.Sprint) branch.
	spec := `
fields:
  id: { pk: true }
  n: { kind: int, min: 42 }
  f: { kind: float, min: 42.5 }
  b: { kind: int, min: true }
`
	sp, err := yamlfe.Parse([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	if sp.Schema.FieldByName("id").Kind != schema.KindUUID {
		t.Fatal("pk without kind should default to uuid")
	}
	if sp.Schema.FieldByName("n").Params["min"] != "42" {
		t.Fatalf("int min = %q", sp.Schema.FieldByName("n").Params["min"])
	}
	if sp.Schema.FieldByName("b").Params["min"] != "true" {
		t.Fatalf("bool min = %q", sp.Schema.FieldByName("b").Params["min"])
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.yaml")
	os.WriteFile(p, []byte("name: t\nfields:\n  a: { kind: int }\n"), 0o644)
	if _, err := yamlfe.Load(p); err != nil {
		t.Fatal(err)
	}
	if _, err := yamlfe.Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("missing file should error")
	}
}
