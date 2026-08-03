package webspec

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestBuildYAMLExplicitOrder(t *testing.T) {
	req := &Request{
		Name:   "users",
		Count:  3,
		Locale: "uz",
		Seed:   42,
		Order:  []string{"id", "age", "active", "score", "tags", "missing"},
		Fields: map[string]map[string]any{
			"id":     {"kind": "uuid"},
			"age":    {"min": float64(18), "max": float64(65)},
			"active": {"kind": "bool", "flag": true},
			"score":  {"weight": float64(1.5)},
			"tags":   {"choices": []any{"a", "b", 3}},
			// "missing" is in Order but not Fields: skipped.
		},
	}
	out, err := BuildYAML(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"name: users", "count: 3", "locale: uz", "seed: 42",
		`"id": {kind: "uuid"}`, "flag: true", "weight: 1.5", `choices: ["a", "b", "3"]`} {
		if !strings.Contains(s, want) {
			t.Fatalf("YAML missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "missing") {
		t.Fatal("Order entry absent from Fields should be skipped")
	}
}

func TestBuildYAMLDefaultOrderAndFallbacks(t *testing.T) {
	req := &Request{
		Fields: map[string]map[string]any{
			"b": {"kind": "int"},
			"a": {"other": struct{}{}}, // default case in renderDef
		},
	}
	out, _ := BuildYAML(req)
	s := string(out)
	if !strings.Contains(s, "name: data") { // NonEmpty fallback
		t.Fatalf("missing name fallback:\n%s", s)
	}
	if strings.Contains(s, "locale:") {
		t.Fatal("empty locale must be omitted")
	}
	// Sorted default order: "a" line before "b" line.
	if strings.Index(s, `"a"`) > strings.Index(s, `"b"`) {
		t.Fatal("default order not sorted")
	}
}

func TestNonEmpty(t *testing.T) {
	if NonEmpty("x", "y") != "x" {
		t.Fatal("NonEmpty should keep non-empty")
	}
	if NonEmpty("", "y") != "y" {
		t.Fatal("NonEmpty should use fallback")
	}
}

func TestWriteCSV(t *testing.T) {
	var b strings.Builder
	WriteCSV(&b, []string{"id", "name"}, []map[string]any{
		{"id": 1, "name": "Ann"},
		{"id": 2, "name": "Bo"},
	})
	got := b.String()
	if !strings.Contains(got, "id,name") || !strings.Contains(got, "1,Ann") {
		t.Fatalf("CSV wrong:\n%s", got)
	}
}

func TestWriteSQL(t *testing.T) {
	var b strings.Builder
	WriteSQL(&b, "t", []string{"id", "name", "ok", "note", "z"}, []map[string]any{
		{"id": 1, "name": "O'Brien", "ok": true, "note": nil, "z": false},
	})
	got := b.String()
	for _, want := range []string{"INSERT INTO t", "'O''Brien'", "TRUE", "NULL", "FALSE"} {
		if !strings.Contains(got, want) {
			t.Fatalf("SQL missing %q:\n%s", want, got)
		}
	}
}

func TestCategoryOf(t *testing.T) {
	if CategoryOf(schema.KindEmail) != "person" {
		t.Fatal("email should be person")
	}
	if CategoryOf(schema.KindGeoJSONPoint) != "location" { // fragment "geo"
		t.Fatalf("geojson should match location fragment, got %q", CategoryOf(schema.KindGeoJSONPoint))
	}
	if CategoryOf(schema.Kind("totallyunknownkind")) != "other" {
		t.Fatal("unknown kind should be other")
	}
}
