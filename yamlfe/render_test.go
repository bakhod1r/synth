package yamlfe_test

import (
	"reflect"
	"testing"

	"github.com/bakhodir/synth/schema"
	"github.com/bakhodir/synth/yamlfe"
)

// Render must produce a spec Parse reads back to the same schema. Profiling
// writes a spec through Render and generation reads it through Parse, so a
// mismatch here silently changes what gets generated.
func TestRenderRoundTrip(t *testing.T) {
	src := `name: users
count: 100
fields:
  id:     { kind: uuid, pk: true }
  name:   { kind: name }
  email:  { kind: email, from: name }
  age:    { kind: int, min: 18, max: 65 }
  status: { kind: enum, choices: ["active", "on hold"], weights: [0.8, 0.2] }
`
	first, err := yamlfe.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := yamlfe.Render(first.Schema, first.Order, first.Name, first.Count)
	if err != nil {
		t.Fatal(err)
	}
	second, err := yamlfe.Parse(doc)
	if err != nil {
		t.Fatalf("rendered spec does not parse: %v\n%s", err, doc)
	}

	if !reflect.DeepEqual(first.Order, second.Order) {
		t.Fatalf("column order changed: %v -> %v", first.Order, second.Order)
	}
	if len(first.Schema.Fields) != len(second.Schema.Fields) {
		t.Fatalf("field count changed: %d -> %d",
			len(first.Schema.Fields), len(second.Schema.Fields))
	}
	for i, want := range first.Schema.Fields {
		got := second.Schema.Fields[i]
		if got.Name != want.Name || got.Kind != want.Kind {
			t.Fatalf("field %d changed: %+v -> %+v", i, want, got)
		}
		if got.From != want.From || got.PK != want.PK || got.Unique != want.Unique {
			t.Fatalf("field %q lost a modifier: %+v -> %+v", want.Name, want, got)
		}
		if !reflect.DeepEqual(got.Choices, want.Choices) {
			t.Fatalf("field %q choices changed: %v -> %v", want.Name, want.Choices, got.Choices)
		}
		if !reflect.DeepEqual(got.Weights, want.Weights) {
			t.Fatalf("field %q weights changed: %v -> %v", want.Name, want.Weights, got.Weights)
		}
		if !reflect.DeepEqual(got.Params, want.Params) {
			t.Fatalf("field %q params changed: %v -> %v", want.Name, want.Params, got.Params)
		}
	}
}

// Column names that YAML would otherwise read as booleans or numbers must
// survive rendering — real exports have columns called "no" and "2024".
func TestRenderQuotesHazardousColumnNames(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "no", Kind: schema.KindBool, Params: map[string]string{}},
		{Name: "2024", Kind: schema.KindInt, Params: map[string]string{}},
		{Name: "order id", Kind: schema.KindUUID, Params: map[string]string{}},
	}}
	order := []string{"no", "2024", "order id"}

	doc, err := yamlfe.Render(s, order, "t", 10)
	if err != nil {
		t.Fatal(err)
	}
	got, err := yamlfe.Parse(doc)
	if err != nil {
		t.Fatalf("rendered spec does not parse: %v\n%s", err, doc)
	}
	if !reflect.DeepEqual(got.Order, order) {
		t.Fatalf("column names mangled: %v -> %v\n%s", order, got.Order, doc)
	}
}

// A column whose kind could not be inferred still needs a generator, or the
// rendered spec would produce an empty column.
func TestRenderGivesUninferredColumnsAKind(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "mystery", Kind: schema.KindUnknown, Params: map[string]string{}},
	}}
	doc, err := yamlfe.Render(s, []string{"mystery"}, "t", 5)
	if err != nil {
		t.Fatal(err)
	}
	got, err := yamlfe.Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema.Fields[0].Kind == schema.KindUnknown {
		t.Fatalf("uninferred column rendered with no kind:\n%s", doc)
	}
}
