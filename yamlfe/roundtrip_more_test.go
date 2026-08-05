package yamlfe

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/constraint"
	"github.com/bakhod1r/synth/schema"
)

// Render's contract is that Parse reads back what it wrote. A spec that renders
// but does not re-parse is worse than none: it is committed, and it fails on
// the run that needs it.
func TestRenderParseRoundTrip(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, PK: true, Params: map[string]string{}},
		{Name: "email", Kind: schema.KindEmail, Unique: true, Params: map[string]string{}},
		{Name: "age", Kind: schema.KindInt, Params: map[string]string{"min": "18", "max": "80"}},
		{Name: "salary", Kind: schema.KindFloat, Params: map[string]string{
			"dist": "lognormal", "mu": "10.5", "sigma": "0.4",
		}},
		{Name: "tier", Kind: schema.KindEnum, Choices: []string{"gold", "silver"},
			Weights: []float64{0.3, 0.7}, Params: map[string]string{}},
		{Name: "note", Kind: schema.KindLorem, Params: map[string]string{"blank": "25"}},
		{Name: "created_at", Kind: schema.KindTime, Params: map[string]string{}},
		{Name: "shipped_at", Kind: schema.KindTime, Params: map[string]string{}},
	}}
	order := []string{"id", "email", "age", "salary", "tier", "note", "created_at", "shipped_at"}
	cs := []constraint.Constraint{constraint.Constraint{Kind: constraint.Ordering, Left: "created_at", Right: "shipped_at"}}

	doc, err := Render(s, order, "people", 250, cs)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(doc)
	if err != nil {
		t.Fatalf("the rendered spec does not parse: %v\n%s", err, doc)
	}

	if spec.Name != "people" || spec.Count != 250 {
		t.Fatalf("name/count = %q/%d", spec.Name, spec.Count)
	}
	if strings.Join(spec.Order, ",") != strings.Join(order, ",") {
		t.Fatalf("order = %v, want %v", spec.Order, order)
	}
	if len(spec.Constraints) != 1 {
		t.Fatalf("got %d constraints, want 1", len(spec.Constraints))
	}

	got := map[string]schema.Field{}
	for _, f := range spec.Schema.Fields {
		got[f.Name] = f
	}
	if !got["id"].PK {
		t.Error("pk lost in the round trip")
	}
	if !got["email"].Unique {
		t.Error("unique lost in the round trip")
	}
	if got["age"].Params["min"] != "18" || got["age"].Params["max"] != "80" {
		t.Errorf("age params = %v", got["age"].Params)
	}
	// mu/sigma are typed as numbers on the parse side: quoting them would make
	// the document unreadable to its own parser.
	if got["salary"].Params["mu"] != "10.5" || got["salary"].Params["sigma"] != "0.4" {
		t.Errorf("distribution params = %v", got["salary"].Params)
	}
	if strings.Join(got["tier"].Choices, ",") != "gold,silver" {
		t.Errorf("choices = %v", got["tier"].Choices)
	}
	if len(got["tier"].Weights) != 2 || got["tier"].Weights[0] != 0.3 {
		t.Errorf("weights = %v", got["tier"].Weights)
	}
	if got["note"].Params["blank"] != "25" {
		t.Errorf("blank = %v", got["note"].Params)
	}
}

// Values profiled from real data are arbitrary text: a category containing a
// colon, a quote or a leading dash must not break the document.
func TestRenderQuotesAwkwardValues(t *testing.T) {
	awkward := []string{"a: b", `has "quotes"`, "-leading dash", "#hash", "", "*star", "{brace}", "tab\there"}
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "weird", Kind: schema.KindEnum, Choices: awkward, Params: map[string]string{}},
		{Name: "sep", Kind: schema.KindLorem, Params: map[string]string{"format": "a: b"}},
	}}
	doc, err := Render(s, nil, "t", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(doc)
	if err != nil {
		t.Fatalf("awkward values broke the document: %v\n%s", err, doc)
	}
	got := spec.Schema.Fields[0].Choices
	if len(got) != len(awkward) {
		t.Fatalf("got %d choices, want %d: %v", len(got), len(awkward), got)
	}
	for i := range awkward {
		if got[i] != awkward[i] {
			t.Fatalf("choice %d = %q, want %q", i, got[i], awkward[i])
		}
	}
	if spec.Schema.Fields[1].Params["format"] != "a: b" {
		t.Fatalf("param value = %q", spec.Schema.Fields[1].Params["format"])
	}
}

// A column name too long for a simple YAML key falls back to the explicit-key
// form rather than producing a document that cannot be parsed.
func TestRenderHandlesAVeryLongColumnName(t *testing.T) {
	long := strings.Repeat("n", maxSimpleKey+50)
	s := &schema.Schema{Fields: []schema.Field{
		{Name: long, Kind: schema.KindLorem, Params: map[string]string{}},
	}}
	doc, err := Render(s, nil, "t", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(doc)
	if err != nil {
		t.Fatalf("a long column name broke the document: %v", err)
	}
	if len(spec.Order) != 1 || spec.Order[0] != long {
		t.Fatalf("the long column did not survive: %v", spec.Order)
	}
}

func TestRenderNilSchemaIsAnError(t *testing.T) {
	if _, err := Render(nil, nil, "t", 1, nil); err == nil {
		t.Fatal("expected an error for a nil schema")
	}
}

// Order names the columns to write; a name the schema does not have is skipped
// rather than emitted as an empty field.
func TestRenderSkipsUnknownOrderEntries(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "a", Kind: schema.KindLorem, Params: map[string]string{}},
	}}
	doc, err := Render(s, []string{"a", "ghost"}, "t", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(spec.Order, ",") != "a" {
		t.Fatalf("order = %v, want just a", spec.Order)
	}
}

// With no order given, the schema's own field order is used.
func TestRenderDefaultsToSchemaOrder(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "z", Kind: schema.KindLorem, Params: map[string]string{}},
		{Name: "a", Kind: schema.KindLorem, Params: map[string]string{}},
	}}
	doc, err := Render(s, nil, "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(spec.Order, ",") != "z,a" {
		t.Fatalf("order = %v, want z,a", spec.Order)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct{ name, doc string }{
		{"not yaml", "\t: :::"},
		{"no fields", "name: t\ncount: 5\n"},
		{"field is not a mapping", "fields:\n  a: 3\n"},
		{"unknown constraint kind", "fields:\n  a: {kind: word}\nconstraints:\n  - {kind: nope}\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Parse([]byte(c.doc)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// unique_mode names how uniqueness is enforced, so it has to survive the round
// trip: a spec that came back as default-mode would build a tracking set for a
// column meant to run in constant memory.
func TestUniqueModeRoundTrip(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{{
		Name: "slug", Kind: schema.KindUsername, Unique: true,
		UniqueMode: "counter", Params: map[string]string{},
	}}}
	doc, err := Render(s, []string{"slug"}, "rows", 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "unique_mode: counter") {
		t.Fatalf("unique_mode not rendered:\n%s", doc)
	}
	back, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if f := back.Schema.Fields[0]; !f.Unique || f.UniqueMode != "counter" {
		t.Fatalf("unique_mode lost in the round trip: %+v", f)
	}
}
