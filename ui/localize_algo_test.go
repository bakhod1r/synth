package ui_test

import (
	"encoding/json"
	"testing"
)

// preview posts a schema to /api/preview and returns the generated rows.
func preview(t *testing.T, body string) []map[string]any {
	t.Helper()
	rec := post(t, "/api/preview", body)
	if rec.Code != 200 {
		t.Fatalf("preview status %d: %s", rec.Code, rec.Body)
	}
	var rows []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&rows); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("preview produced no rows")
	}
	return rows
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// The localize= toggle the dialog adds must reach the engine: a localizable
// field carrying localize=false is generated as if the locale were en_US, even
// though the dataset locale is ru_RU. Names are the clearest witness — a ru_RU
// name is Cyrillic, its de-localized form is ASCII.
func TestLocalizeFalseReachesEngine(t *testing.T) {
	localized := preview(t,
		`{"count":16,"locale":"ru_RU","fields":{"n":{"kind":"name"}},"order":["n"]}`)
	sawCyrillic := false
	for _, row := range localized {
		if !isASCII(row["n"].(string)) {
			sawCyrillic = true
			break
		}
	}
	if !sawCyrillic {
		t.Fatal("ru_RU names should be non-ASCII; test premise is wrong")
	}

	opted := preview(t,
		`{"count":16,"locale":"ru_RU","fields":{"n":{"kind":"name","localize":"false"}},"order":["n"]}`)
	for _, row := range opted {
		if v := row["n"].(string); !isASCII(v) {
			t.Fatalf("localize=false under ru_RU still produced a localized name: %q", v)
		}
	}
}

// The algo= option the hash dialog adds must reach the engine: a hash mask with
// algo=sha512 produces a 128-hex digest, where the default SHA-256 produces 64.
func TestHashAlgorithmReachesEngine(t *testing.T) {
	def := preview(t,
		`{"count":3,"fields":{"h":{"kind":"lorem","mask":"hash"}},"order":["h"]}`)
	if got := len(def[0]["h"].(string)); got != 64 {
		t.Fatalf("default hash digest = %d chars, want 64 (sha256)", got)
	}

	big := preview(t,
		`{"count":3,"fields":{"h":{"kind":"lorem","mask":"hash","algo":"sha512"}},"order":["h"]}`)
	if got := len(big[0]["h"].(string)); got != 128 {
		t.Fatalf("algo=sha512 hash digest = %d chars, want 128", got)
	}
}
