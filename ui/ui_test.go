package ui_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/ui"
)

func post(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	ui.Handler().ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	ui.Handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

// The palette must come from the real registry, or it will drift from what
// the engine can actually generate.
func TestTypesEndpointComesFromTheRegistry(t *testing.T) {
	rec := get(t, "/api/types")
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var got []struct {
		Kind      string `json:"kind"`
		Category  string `json:"category"`
		Localized bool   `json:"localized"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) < 150 {
		t.Fatalf("only %d types exposed; the registry has %d", len(got), len(providers.Kinds()))
	}
	seen := map[string]bool{}
	for _, ti := range got {
		if ti.Category == "" {
			t.Fatalf("type %q has no category", ti.Kind)
		}
		if seen[ti.Kind] {
			t.Fatalf("type %q listed twice", ti.Kind)
		}
		seen[ti.Kind] = true
	}
	// A type the engine supports must not be missing from the palette.
	for _, k := range providers.Kinds() {
		if string(k) == "object" || string(k) == "array" || k == "" {
			continue
		}
		if !seen[string(k)] {
			t.Fatalf("registry has %q but the palette does not list it", k)
		}
	}
}

// Locale coverage must be reported honestly: not every type follows the locale.
func TestTypesReportLocaleCoverageHonestly(t *testing.T) {
	rec := get(t, "/api/types")
	var got []struct {
		Kind      string `json:"kind"`
		Localized bool   `json:"localized"`
	}
	json.NewDecoder(rec.Body).Decode(&got)

	byKind := map[string]bool{}
	for _, ti := range got {
		byKind[ti.Kind] = ti.Localized
	}
	if !byKind["name"] {
		t.Error("name follows the locale but is not marked localized")
	}
	if byKind["superhero"] {
		t.Error("superhero returns English values in every locale; marking it " +
			"localized would mislead the user")
	}
}

func TestLocalesEndpoint(t *testing.T) {
	rec := get(t, "/api/locales")
	var names []string
	json.NewDecoder(rec.Body).Decode(&names)
	if len(names) < 10 {
		t.Fatalf("only %d locales offered", len(names))
	}
}

// Preview must return real generated rows for the posted schema.
func TestPreviewReturnsRows(t *testing.T) {
	body := `{"fields":{"name":{"kind":"name"},"email":{"kind":"email"}},
	          "order":["name","email"],"locale":"uz_UZ","count":10,"seed":1}`
	rec := post(t, "/api/preview", body)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var rows []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Fatalf("got %d preview rows, want 10", len(rows))
	}
	email, _ := rows[0]["email"].(string)
	if !strings.Contains(email, "@") {
		t.Fatalf("preview email is not real: %v", rows[0])
	}
}

// The same seed must give the same preview, or the UI teaches the wrong thing
// about reproducibility.
func TestPreviewIsReproducible(t *testing.T) {
	body := `{"fields":{"name":{"kind":"name"}},"order":["name"],"count":5,"seed":99}`
	first := post(t, "/api/preview", body).Body.String()
	second := post(t, "/api/preview", body).Body.String()
	if first != second {
		t.Fatalf("same seed gave different previews:\n%s\n%s", first, second)
	}
}

// A misspelled kind must be named in the error. Making mistakes visible is
// the reason the workbench exists.
func TestPreviewRejectsUnknownKind(t *testing.T) {
	rec := post(t, "/api/preview", `{"fields":{"x":{"kind":"nope"}},"order":["x"]}`)
	if rec.Code != 400 {
		t.Fatalf("status %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nope") {
		t.Fatalf("error does not name the bad kind: %s", rec.Body)
	}
}

// An empty schema is a mistake worth explaining, not an empty table.
func TestPreviewRejectsEmptySchema(t *testing.T) {
	rec := post(t, "/api/preview", `{"fields":{}}`)
	if rec.Code != 400 {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

// Malformed JSON must produce a message, not a panic.
func TestPreviewRejectsMalformedBody(t *testing.T) {
	rec := post(t, "/api/preview", `{not json`)
	if rec.Code != 400 {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

// Preview must be capped so a huge count cannot hang the browser.
func TestPreviewCountIsCapped(t *testing.T) {
	rec := post(t, "/api/preview",
		`{"fields":{"name":{"kind":"name"}},"order":["name"],"count":1000000}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var rows []map[string]any
	json.NewDecoder(rec.Body).Decode(&rows)
	if len(rows) > 100 {
		t.Fatalf("preview returned %d rows; it must be capped", len(rows))
	}
}

// Download must produce a real file with the right headers.
func TestGenerateDownloads(t *testing.T) {
	for _, tc := range []struct{ format, contentType, contains string }{
		{"csv", "text/csv", "name,email"},
		{"jsonl", "application/x-ndjson", `"email"`},
		{"sql", "application/sql", "INSERT INTO"},
	} {
		body := `{"name":"users","fields":{"name":{"kind":"name"},"email":{"kind":"email"}},
		          "order":["name","email"],"count":5,"format":"` + tc.format + `"}`
		rec := post(t, "/api/generate", body)
		if rec.Code != 200 {
			t.Fatalf("%s: status %d: %s", tc.format, rec.Code, rec.Body)
		}
		if ct := rec.Header().Get("Content-Type"); ct != tc.contentType {
			t.Errorf("%s: content type %q, want %q", tc.format, ct, tc.contentType)
		}
		if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "users."+tc.format) {
			t.Errorf("%s: bad disposition %q", tc.format, cd)
		}
		if !strings.Contains(rec.Body.String(), tc.contains) {
			t.Errorf("%s: body missing %q:\n%s", tc.format, tc.contains, rec.Body.String())
		}
	}
}

// The generate endpoint must not be capped the way preview is: a download of
// 50,000 rows is the normal case.
func TestGenerateIsNotCapped(t *testing.T) {
	rec := post(t, "/api/generate",
		`{"fields":{"name":{"kind":"name"}},"order":["name"],"count":5000,"format":"csv"}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if n := strings.Count(rec.Body.String(), "\n"); n < 5000 {
		t.Fatalf("download produced %d lines, want 5001", n)
	}
}

// The page must not reference any external origin. A workbench that phones
// home would break the promise the whole project rests on.
func TestPageHasNoExternalResources(t *testing.T) {
	for _, path := range []string{"/", "/app.js", "/app.css", "/i18n.js"} {
		body := get(t, path).Body.String()
		// The SVG XML namespace is a URI by spec, not a fetch: it identifies the
		// vocabulary of an inline data: favicon, and the browser never requests
		// it. Stripping it keeps the check on real external references.
		body = strings.ReplaceAll(body, "http://www.w3.org/2000/svg", "")
		for _, bad := range []string{"http://", "https://", "//cdn", "fonts.googleapis", "googletagmanager"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s references an external origin: %q", path, bad)
			}
		}
	}
}

// Binding anything but loopback must be refused, not merely discouraged.
func TestServeRefusesNonLoopback(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:8080", "192.168.1.5:8080", "example.com:8080"} {
		err := ui.Serve(addr)
		if err == nil {
			t.Fatalf("Serve(%q) was allowed", addr)
		}
		if !strings.Contains(err.Error(), "loopback") {
			t.Fatalf("Serve(%q) failed for the wrong reason: %v", addr, err)
		}
	}
}

// A malformed address must be reported as such.
func TestServeRejectsMalformedAddress(t *testing.T) {
	if err := ui.Serve("not-an-address"); err == nil {
		t.Fatal("expected an error for a malformed address")
	}
}
