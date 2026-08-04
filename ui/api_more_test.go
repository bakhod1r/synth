package ui

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The workbench API is what a `curl` user drives, so its contract is the
// response body, not the page: every route below is exercised over a real
// http.Handler.

func TestGenerateEveryFormat(t *testing.T) {
	cases := []struct {
		format      string
		contentType string
		check       func(t *testing.T, body string)
	}{
		{"csv", "text/csv", func(t *testing.T, body string) {
			recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
			if err != nil {
				t.Fatalf("csv does not parse: %v", err)
			}
			if len(recs) != 6 {
				t.Fatalf("got %d lines incl. header, want 6", len(recs))
			}
		}},
		{"json", "application/json", func(t *testing.T, body string) {
			var rows []map[string]any
			if err := json.Unmarshal([]byte(body), &rows); err != nil {
				t.Fatalf("json does not parse: %v", err)
			}
			if len(rows) != 5 {
				t.Fatalf("got %d rows, want 5", len(rows))
			}
		}},
		{"jsonl", "application/x-ndjson", func(t *testing.T, body string) {
			lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
			if len(lines) != 5 {
				t.Fatalf("got %d lines, want 5", len(lines))
			}
			for i, l := range lines {
				var obj map[string]any
				if err := json.Unmarshal([]byte(l), &obj); err != nil {
					t.Fatalf("line %d does not parse: %v", i+1, err)
				}
			}
		}},
		{"sql", "application/sql", func(t *testing.T, body string) {
			if n := strings.Count(body, "INSERT INTO user"); n != 5 {
				t.Fatalf("got %d INSERTs, want 5:\n%s", n, body)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.format, func(t *testing.T) {
			w := do(t, "GET", "/api/generate?preset=user&n=5&seed=1&format="+c.format, "")
			if w.Code != http.StatusOK {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, c.contentType) {
				t.Fatalf("Content-Type = %q, want %q", ct, c.contentType)
			}
			c.check(t, w.Body.String())
		})
	}
}

func TestGenerateDefaultsToCSV(t *testing.T) {
	w := do(t, "GET", "/api/generate?preset=order&n=3&seed=2", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="order.csv"`) {
		t.Fatalf("Content-Disposition = %q", cd)
	}
}

func TestBadRequests(t *testing.T) {
	cases := []struct {
		name, method, target, body string
	}{
		{"GET without a preset", "GET", "/api/preview", ""},
		{"unknown preset", "GET", "/api/preview?preset=nope&n=1", ""},
		{"non-numeric n", "GET", "/api/preview?preset=user&n=abc", ""},
		{"zero n", "GET", "/api/preview?preset=user&n=0", ""},
		{"negative n", "GET", "/api/preview?preset=user&n=-5", ""},
		{"non-numeric seed", "GET", "/api/preview?preset=user&seed=xyz", ""},
		{"body is not JSON", "POST", "/api/preview", "{not json"},
		{"schema has no fields", "POST", "/api/preview", `{"name":"t","fields":{}}`},
		{"field without a kind", "POST", "/api/preview", `{"name":"t","fields":{"a":{}}}`},
		{"unknown kind", "POST", "/api/preview", `{"name":"t","fields":{"a":{"kind":"nope"}}}`},
		{"generate rejects the same", "POST", "/api/generate", `{"name":"t","fields":{"a":{"kind":"nope"}}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := do(t, c.method, c.target, c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400; body: %s", w.Code, w.Body.String())
			}
			if strings.TrimSpace(w.Body.String()) == "" {
				t.Fatal("a 400 with no explanation")
			}
		})
	}
}

// The preview cap exists because the browser re-renders on every keystroke.
func TestPreviewClampsRowCount(t *testing.T) {
	w := do(t, "GET", "/api/preview?preset=user&n=100000", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != maxPreview {
		t.Fatalf("got %d rows, want the %d-row cap", len(rows), maxPreview)
	}
}

// Generate is not capped: that is the download path.
func TestGenerateIsNotCapped(t *testing.T) {
	w := do(t, "GET", "/api/generate?preset=user&n=500&format=json", "")
	var rows []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 500 {
		t.Fatalf("got %d rows, want 500", len(rows))
	}
}

func TestPostedSchemaGenerates(t *testing.T) {
	body := `{"name":"people","count":4,"fields":{
		"id":{"kind":"uuid"},
		"email":{"kind":"email"},
		"tier":{"kind":"enum","choices":["gold","silver"]}
	}}`
	w := do(t, "POST", "/api/preview", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for _, r := range rows {
		if s, _ := r["email"].(string); !strings.Contains(s, "@") {
			t.Fatalf("email = %v", r["email"])
		}
		if tier, _ := r["tier"].(string); tier != "gold" && tier != "silver" {
			t.Fatalf("tier = %v, want an enum choice", r["tier"])
		}
	}
}

func TestPostedSchemaCountDefaults(t *testing.T) {
	w := do(t, "POST", "/api/preview", `{"name":"t","fields":{"a":{"kind":"word"}}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Fatalf("got %d rows, want the default 10", len(rows))
	}
}

// Unmasking has to be asked for by name.
func TestUnmaskedIsOptInOverTheAPI(t *testing.T) {
	masked := do(t, "GET", "/api/generate?preset=payment&n=20&seed=3&format=json", "").Body.String()
	if !strings.Contains(masked, "*") {
		t.Fatal("the default output has no masked value")
	}
	raw := do(t, "GET", "/api/generate?preset=payment&n=20&seed=3&format=json&unmasked=true", "").Body.String()
	if strings.Contains(raw, `card_number":"`+"****") {
		t.Fatal("unmasked=true still returned a masked card number")
	}
}

func TestTypesRouteDescribesProviders(t *testing.T) {
	w := do(t, "GET", "/api/types", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var types []typeInfo
	if err := json.Unmarshal(w.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if len(types) < 50 {
		t.Fatalf("only %d types listed", len(types))
	}
	seen := map[string]typeInfo{}
	for _, ti := range types {
		if ti.Kind == "" || ti.Category == "" {
			t.Fatalf("type with no kind or category: %+v", ti)
		}
		seen[ti.Kind] = ti
	}
	// Structural kinds are not value types and must not be offered.
	for _, k := range []string{"object", "array", ""} {
		if _, ok := seen[k]; ok {
			t.Fatalf("structural kind %q is listed as a value type", k)
		}
	}
	if !seen["firstname"].Localized {
		t.Fatal("firstname should be reported as localized")
	}
}

func TestLocalesAndPresetsRoutes(t *testing.T) {
	w := do(t, "GET", "/api/locales", "")
	var names []string
	if err := json.Unmarshal(w.Body.Bytes(), &names); err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no locales listed")
	}

	w = do(t, "GET", "/api/presets", "")
	var presets []struct {
		Name string `json:"name"`
		YAML string `json:"yaml"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &presets); err != nil {
		t.Fatal(err)
	}
	if len(presets) == 0 {
		t.Fatal("no presets listed")
	}
	for _, p := range presets {
		if p.Name == "" || !strings.Contains(p.YAML, "fields:") {
			t.Fatalf("preset %q has no usable YAML", p.Name)
		}
	}
}

func TestStaticPageIsServed(t *testing.T) {
	w := do(t, "GET", "/", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), "<html") {
		t.Fatal("the workbench page was not served")
	}
}

// Serve must refuse a non-loopback bind: a data generator answering to the
// network is a different, riskier thing.
func TestServeRefusesNonLoopback(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:7777", "8.8.8.8:7777", "nonsense"} {
		if err := Serve(addr); err == nil {
			t.Fatalf("Serve(%q) did not refuse", addr)
		}
	}
}
