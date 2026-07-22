package ui

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

// A fixture must be one request, with no schema to write.
func TestPreviewFromPreset(t *testing.T) {
	w := get(t, "/api/preview?preset=transaction&n=5")
	if w.Code != 200 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	if card, _ := rows[0]["card_number"].(string); !strings.Contains(card, "*") {
		t.Fatalf("the API served an unmasked card number: %q", card)
	}
}

// Preview caps its row count: the browser re-renders it on every keystroke.
func TestPreviewCapsRows(t *testing.T) {
	w := get(t, "/api/preview?preset=user&n=100000")
	var rows []map[string]any
	json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != maxPreview {
		t.Fatalf("got %d rows, want the %d cap", len(rows), maxPreview)
	}
}

// generate is not capped, and honours the requested format.
func TestGenerateFormats(t *testing.T) {
	for _, tc := range []struct{ format, want string }{
		{"csv", "text/csv"},
		{"json", "application/json"},
		{"jsonl", "application/x-ndjson"},
		{"sql", "application/sql"},
	} {
		w := get(t, "/api/generate?preset=user&n=500&format="+tc.format)
		if w.Code != 200 {
			t.Fatalf("%s: got %d: %s", tc.format, w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.want) {
			t.Errorf("%s: content type %q, want %q", tc.format, ct, tc.want)
		}
	}
}

// The same seed must give the same bytes, or the API cannot back a golden test.
func TestSeedIsReproducibleOverHTTP(t *testing.T) {
	a := get(t, "/api/generate?preset=order&n=50&seed=42&format=csv").Body.String()
	b := get(t, "/api/generate?preset=order&n=50&seed=42&format=csv").Body.String()
	if a != b {
		t.Fatal("the same seed produced different output")
	}
}

func TestUnmaskedIsOptIn(t *testing.T) {
	w := get(t, "/api/preview?preset=transaction&n=5&unmasked=true")
	var rows []map[string]any
	json.Unmarshal(w.Body.Bytes(), &rows)
	if card, _ := rows[0]["card_number"].(string); strings.Contains(card, "*") {
		t.Fatalf("unmasked=true still masked: %q", card)
	}
}

// A wrong name must say so, and say where to look.
func TestUnknownPresetExplainsItself(t *testing.T) {
	w := get(t, "/api/preview?preset=nope")
	if w.Code != 400 {
		t.Fatalf("got %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "/api/presets") {
		t.Fatalf("the error does not point anywhere: %q", w.Body.String())
	}
}

func TestBadSeedIsRejected(t *testing.T) {
	if w := get(t, "/api/preview?preset=user&seed=soon"); w.Code != 400 {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestPresetsAreListedWithTheirYAML(t *testing.T) {
	w := get(t, "/api/presets")
	var out []struct{ Name, YAML string }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("no presets listed")
	}
	for _, e := range out {
		if e.YAML == "" {
			t.Errorf("preset %q has no spec", e.Name)
		}
	}
}

// The POST path the page uses must keep working.
func TestPostedSchemaStillWorks(t *testing.T) {
	body := `{"count":3,"fields":{"a":{"kind":"city"}},"order":["a"]}`
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest("POST", "/api/preview", strings.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
}
