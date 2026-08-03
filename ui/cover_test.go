package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, method, url, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, r)
	return w
}

func TestPreviewQueryFormAllOptions(t *testing.T) {
	// Preset via query with locale, seed and unmasked exercises resolveSpec's
	// query path, count parsing and the option wiring.
	w := do(t, "GET", "/api/preview?preset=user&n=3&locale=uz_UZ&seed=7&unmasked=true", "")
	if w.Code != 200 {
		t.Fatalf("preview code = %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveSpecErrors(t *testing.T) {
	// GET without a preset is an error.
	if w := do(t, "GET", "/api/preview", ""); w.Code != 400 {
		t.Fatalf("GET without preset = %d", w.Code)
	}
	// Unknown preset.
	if w := do(t, "GET", "/api/preview?preset=nope", ""); w.Code != 400 {
		t.Fatalf("unknown preset = %d", w.Code)
	}
	// Bad n.
	if w := do(t, "GET", "/api/preview?preset=user&n=abc", ""); w.Code != 400 {
		t.Fatalf("bad n = %d", w.Code)
	}
	// Bad seed.
	if w := do(t, "GET", "/api/preview?preset=user&seed=xx", ""); w.Code != 400 {
		t.Fatalf("bad seed = %d", w.Code)
	}
}

func TestGenerateAllFormats(t *testing.T) {
	for _, f := range []string{"json", "jsonl", "sql", "csv"} {
		w := do(t, "GET", "/api/generate?preset=user&n=3&format="+f, "")
		if w.Code != 200 {
			t.Fatalf("generate %s = %d: %s", f, w.Code, w.Body.String())
		}
	}
}

func TestTypesAndLocalesAndPresets(t *testing.T) {
	if w := do(t, "GET", "/api/types", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "kind") {
		t.Fatalf("types = %d", w.Code)
	}
	if w := do(t, "GET", "/api/locales", ""); w.Code != 200 {
		t.Fatalf("locales = %d", w.Code)
	}
	if w := do(t, "GET", "/api/presets", ""); w.Code != 200 {
		t.Fatalf("presets = %d", w.Code)
	}
}

func TestPostSchemaPreviewAndErrors(t *testing.T) {
	good := `{"count":2,"fields":{"a":{"kind":"int"}},"order":["a"]}`
	if w := do(t, "POST", "/api/preview", good); w.Code != 200 {
		t.Fatalf("post preview = %d: %s", w.Code, w.Body.String())
	}
	// Malformed body.
	if w := do(t, "POST", "/api/preview", "{"); w.Code != 400 {
		t.Fatal("malformed body should 400")
	}
	// No fields.
	if w := do(t, "POST", "/api/preview", `{"fields":{}}`); w.Code != 400 {
		t.Fatal("empty schema should 400")
	}
	// Field with no kind.
	if w := do(t, "POST", "/api/preview", `{"fields":{"a":{}}}`); w.Code != 400 {
		t.Fatal("field without kind should 400")
	}
	// Unknown kind.
	if w := do(t, "POST", "/api/preview", `{"fields":{"a":{"kind":"zzz"}}}`); w.Code != 400 {
		t.Fatal("unknown kind should 400")
	}
}

func TestPostPreviewCountDefaultAndGenerateErrors(t *testing.T) {
	// No count -> parseSpec defaults it.
	if w := do(t, "POST", "/api/preview", `{"fields":{"a":{"kind":"int"}},"order":["a"]}`); w.Code != 200 {
		t.Fatalf("no-count preview = %d: %s", w.Code, w.Body.String())
	}
	// A spec that parses but references an unknown field fails at generation.
	badRef := `{"fields":{"a":{"kind":"email","from":"ghost"}},"order":["a"]}`
	if w := do(t, "POST", "/api/preview", badRef); w.Code != 400 {
		t.Fatalf("bad from= preview should 400, got %d", w.Code)
	}
	if w := do(t, "POST", "/api/generate", badRef); w.Code != 400 {
		t.Fatalf("bad from= generate should 400, got %d", w.Code)
	}
	// handleGenerate with no preset (GET) is a resolveSpec error.
	if w := do(t, "GET", "/api/generate", ""); w.Code != 400 {
		t.Fatalf("GET generate without preset should 400, got %d", w.Code)
	}
}

func TestPortOf(t *testing.T) {
	if portOf(":9999") != "9999" {
		t.Fatal("portOf explicit")
	}
	if portOf("garbage") != "8080" {
		t.Fatal("portOf fallback")
	}
}

func TestServeValidation(t *testing.T) {
	// Malformed address.
	if err := Serve("not-an-addr-with-no-colon"); err == nil {
		t.Fatal("malformed address should error")
	}
	// Non-loopback IP is refused.
	if err := Serve("8.8.8.8:9999"); err == nil {
		t.Fatal("non-loopback should be refused")
	}
	// Non-loopback, non-localhost hostname is refused.
	if err := Serve("example.com:9999"); err == nil {
		t.Fatal("non-loopback host should be refused")
	}
	// host == "" defaults to loopback, exercises portOf, then fails to bind an
	// invalid port rather than blocking.
	if err := Serve(":-1"); err == nil {
		t.Fatal("invalid port should surface a bind error")
	}
}
