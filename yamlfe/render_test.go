package yamlfe_test

import (
	"reflect"
	"strings"
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
	doc, err := yamlfe.Render(first.Schema, first.Order, first.Name, first.Count, first.Constraints)
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

	doc, err := yamlfe.Render(s, order, "t", 10, nil)
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
	doc, err := yamlfe.Render(s, []string{"mystery"}, "t", 5, nil)
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

// A constraints block must survive rendering and parsing: profiling writes it
// and generation reads it, so a loss here silently drops an invariant.
func TestRenderRoundTripsConstraints(t *testing.T) {
	src := `name: orders
count: 10
fields:
  subtotal:  { kind: amount }
  tax:       { kind: amount }
  total:     { kind: amount }
  status:    { kind: enum, choices: ["paid", "refunded"] }
  refund_at: { kind: time }
constraints:
  - {kind: sum, parts: [subtotal, tax], whole: total}
  - {kind: ordering, left: subtotal, right: total}
  - {kind: implication, when: status, equals: "refunded", then: refund_at, exclusive: true}
  - {kind: range, left: tax, lo: 0, hi: 500}
`
	first, err := yamlfe.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Constraints) != 4 {
		t.Fatalf("parsed %d constraints, want 4", len(first.Constraints))
	}
	doc, err := yamlfe.Render(first.Schema, first.Order, first.Name, first.Count, first.Constraints)
	if err != nil {
		t.Fatal(err)
	}
	second, err := yamlfe.Parse(doc)
	if err != nil {
		t.Fatalf("rendered spec does not parse: %v\n%s", err, doc)
	}
	if !reflect.DeepEqual(first.Constraints, second.Constraints) {
		t.Fatalf("constraints changed:\n%+v\n%+v\n%s", first.Constraints, second.Constraints, doc)
	}
}

// An unknown constraint kind must be an error, not a silently dropped rule.
func TestParseRejectsUnknownConstraintKind(t *testing.T) {
	_, err := yamlfe.Parse([]byte(`name: t
fields:
  a: { kind: int }
constraints:
  - {kind: telepathy, left: a, right: a}
`))
	if err == nil {
		t.Fatal("expected an error for an unknown constraint kind")
	}
	if !strings.Contains(err.Error(), "telepathy") {
		t.Fatalf("error does not name the bad kind: %v", err)
	}
}

// A spec and an equivalent Go struct must generate the same data. Coherence
// links used to be applied only on the struct path, so a card brand written in
// YAML disagreed with its card number while the same schema in Go was fine.
func TestSpecGetsTheSameCoherenceAsAStruct(t *testing.T) {
	sp, err := yamlfe.Parse([]byte(`name: payments
count: 5
fields:
  fullname:  { kind: name }
  email:     { kind: email }
  card:      { kind: card }
  cardbrand: { kind: cardbrand }
  created_at: { kind: time }
  updated_at: { kind: time }
`))
	if err != nil {
		t.Fatal(err)
	}
	from := map[string]string{}
	for _, f := range sp.Schema.Fields {
		from[f.Name] = f.From
	}
	if from["email"] != "fullname" {
		t.Errorf("email should derive from the name, got from=%q", from["email"])
	}
	if from["cardbrand"] != "card" {
		t.Errorf("cardbrand should derive from the card, got from=%q", from["cardbrand"])
	}
	if from["updated_at"] != "created_at" {
		t.Errorf("updated_at should come after created_at, got from=%q", from["updated_at"])
	}
}

// An explicit from= is the author's decision and must survive the automatic
// linking.
func TestExplicitFromIsNotOverwritten(t *testing.T) {
	sp, err := yamlfe.Parse([]byte(`name: t
count: 2
fields:
  fullname: { kind: name }
  nickname: { kind: firstname }
  email:    { kind: email, from: nickname }
`))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range sp.Schema.Fields {
		if f.Name == "email" && f.From != "nickname" {
			t.Fatalf("explicit from= was overwritten with %q", f.From)
		}
	}
}
