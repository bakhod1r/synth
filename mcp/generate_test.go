package mcp

import (
	"strings"
	"testing"
)

func TestGenerateFromPreset(t *testing.T) {
	out, err := handleGenerate(generateArgs{Preset: "transaction", Rows: 5, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := out.([]map[string]any)
	if !ok {
		t.Fatalf("got %T, want rows", out)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
}

func TestGenerateDefaultsItsRowCount(t *testing.T) {
	out, err := handleGenerate(generateArgs{Preset: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(out.([]map[string]any)); got != defaultRows {
		t.Fatalf("got %d rows, want the %d default", got, defaultRows)
	}
}

// The masking default must survive the trip through MCP. An assistant pasting a
// generated card number into a chat log is exactly the accident it prevents.
func TestGenerateMasksByDefault(t *testing.T) {
	out, err := handleGenerate(generateArgs{Preset: "transaction", Rows: 20, Seed: 2})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range out.([]map[string]any) {
		card, _ := r["card_number"].(string)
		if !strings.Contains(card, "*") {
			t.Fatalf("row %d: unmasked card %q", i, card)
		}
	}
}

func TestGenerateUnmaskedIsOptIn(t *testing.T) {
	out, err := handleGenerate(generateArgs{Preset: "transaction", Rows: 10, Seed: 2, Unmasked: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range out.([]map[string]any) {
		if card, _ := r["card_number"].(string); strings.Contains(card, "*") {
			t.Fatalf("row %d: unmasked=true still masked %q", i, card)
		}
	}
}

func TestGenerateFromSpec(t *testing.T) {
	spec := "name: t\nfields:\n  city: { kind: city }\n"
	out, err := handleGenerate(generateArgs{Spec: spec, Rows: 3, Locale: "uz_UZ", Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.([]map[string]any)) != 3 {
		t.Fatal("wrong row count")
	}
}

// Neither preset nor spec is a mistake worth naming, and so is both.
func TestGenerateRejectsAmbiguousInput(t *testing.T) {
	if _, err := handleGenerate(generateArgs{Rows: 1}); err == nil {
		t.Fatal("a call with no preset and no spec was accepted")
	}
	if _, err := handleGenerate(generateArgs{Preset: "user", Spec: "name: t", Rows: 1}); err == nil {
		t.Fatal("a call with both preset and spec was accepted")
	}
}

func TestGenerateRejectsUnknownPreset(t *testing.T) {
	_, err := handleGenerate(generateArgs{Preset: "nope", Rows: 1})
	if err == nil || !strings.Contains(err.Error(), "list_presets") {
		t.Fatalf("error %v does not point at list_presets", err)
	}
}

func TestGenerateRejectsABrokenSpec(t *testing.T) {
	if _, err := handleGenerate(generateArgs{Spec: "fields: [not a map", Rows: 1}); err == nil {
		t.Fatal("an unparseable spec was accepted")
	}
}

func TestGenerateHonoursTheRowLimit(t *testing.T) {
	if _, err := handleGenerate(generateArgs{Preset: "user", Rows: maxRows + 1}); err == nil {
		t.Fatal("a request above the row limit was accepted")
	}
}

func TestGenerateIsReproducible(t *testing.T) {
	a, err := handleGenerate(generateArgs{Preset: "order", Rows: 20, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := handleGenerate(generateArgs{Preset: "order", Rows: 20, Seed: 7})
	ar, br := a.([]map[string]any), b.([]map[string]any)
	for i := range ar {
		if ar[i]["id"] != br[i]["id"] {
			t.Fatalf("row %d differs between runs with the same seed", i)
		}
	}
}

func TestListTypes(t *testing.T) {
	out, err := handleListTypes(listTypesArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(out.([]typeInfo)); n < 200 {
		t.Fatalf("the catalog has only %d entries", n)
	}
	filtered, _ := handleListTypes(listTypesArgs{Search: "card"})
	got := filtered.([]typeInfo)
	if len(got) == 0 {
		t.Fatal(`searching for "card" matched nothing`)
	}
	for _, ty := range got {
		if !strings.Contains(ty.Kind, "card") {
			t.Fatalf("search returned an unrelated kind %q", ty.Kind)
		}
	}
}

// An empty result must be an empty list, not null: a model handed `null` cannot
// tell "no matches" from "the tool broke".
func TestListTypesReturnsAnEmptyListNotNull(t *testing.T) {
	out, err := handleListTypes(listTypesArgs{Search: "zzzznope"})
	if err != nil {
		t.Fatal(err)
	}
	if out.([]typeInfo) == nil {
		t.Fatal("a search with no matches returned nil")
	}
}

func TestListPresetsCarriesTheSpec(t *testing.T) {
	out, err := handleListPresets()
	if err != nil {
		t.Fatal(err)
	}
	presets := out.([]presetInfo)
	if len(presets) == 0 {
		t.Fatal("no presets listed")
	}
	for _, p := range presets {
		if p.YAML == "" {
			t.Errorf("preset %q has no spec", p.Name)
		}
	}
}

// A listed preset must be one generate accepts, or the two tools disagree.
func TestListedPresetsAllGenerate(t *testing.T) {
	out, _ := handleListPresets()
	for _, p := range out.([]presetInfo) {
		if _, err := handleGenerate(generateArgs{Preset: p.Name, Rows: 2, Seed: 1}); err != nil {
			t.Errorf("listed preset %q does not generate: %v", p.Name, err)
		}
	}
}
