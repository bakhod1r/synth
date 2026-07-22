package ui

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// A generation must be counted from what was actually written, so the reported
// size describes the bytes that left rather than what was asked for.
func TestGenerateIsRecorded(t *testing.T) {
	rec := newMemoryRecorder()
	h := Handler(WithRecorder(rec))

	body := `{"name":"t","count":25,"fields":{"a":{"kind":"city"},"b":{"kind":"email"}},"order":["a","b"]}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/generate", strings.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}

	got, err := rec.Totals()
	if err != nil {
		t.Fatal(err)
	}
	if got.Files != 1 {
		t.Fatalf("files = %d, want 1", got.Files)
	}
	if got.Rows != 25 {
		t.Fatalf("rows = %d, want 25", got.Rows)
	}
	if want := int64(25 * 2); got.Cells != want {
		t.Fatalf("cells = %d, want %d", got.Cells, want)
	}
	if got.Bytes != int64(w.Body.Len()) {
		t.Fatalf("bytes = %d, want the %d actually written", got.Bytes, w.Body.Len())
	}
}

// A rejected request must not be counted. A schema that failed to generate is
// not a file anyone has.
func TestFailedGenerationIsNotCounted(t *testing.T) {
	rec := newMemoryRecorder()
	h := Handler(WithRecorder(rec))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/generate",
		strings.NewReader(`{"count":5,"fields":{"a":{"kind":"nope"}},"order":["a"]}`)))
	if w.Code == 200 {
		t.Fatal("an unknown kind was accepted")
	}
	got, _ := rec.Totals()
	if got.Files != 0 {
		t.Fatalf("a failed run was counted: %+v", got)
	}
}

// Cells is rows × columns, which is the number that says how much data there
// is. Ten thousand rows of two columns and of two hundred are the same row
// count and nothing alike.
func TestCellsCountsColumnsToo(t *testing.T) {
	narrow := Run{Rows: 10_000, Columns: 2}
	wide := Run{Rows: 10_000, Columns: 200}
	if narrow.Cells() == wide.Cells() {
		t.Fatal("column count does not affect the cell total")
	}
	if wide.Cells() != 2_000_000 {
		t.Fatalf("cells = %d, want 2,000,000", wide.Cells())
	}
}

func TestStatsEndpoint(t *testing.T) {
	rec := newMemoryRecorder()
	h := Handler(WithRecorder(rec))
	rec.Record(Run{Rows: 10, Columns: 3, Bytes: 512})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/stats", nil))
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
	var totals Totals
	if err := json.Unmarshal(w.Body.Bytes(), &totals); err != nil {
		t.Fatal(err)
	}
	if totals.Rows != 10 || totals.Cells != 30 || totals.Bytes != 512 {
		t.Fatalf("got %+v", totals)
	}
	// The in-memory recorder forgets on exit, and must say so rather than
	// letting a number that silently resets look like data loss.
	if totals.Persistent {
		t.Fatal("the in-memory recorder claims to be persistent")
	}
}

// Counting must be safe from several requests at once; the server handles them
// concurrently.
func TestRecorderIsConcurrencySafe(t *testing.T) {
	rec := newMemoryRecorder()
	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			rec.Record(Run{Rows: 10, Columns: 2, Bytes: 100})
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
	got, _ := rec.Totals()
	if got.Files != 50 || got.Rows != 500 || got.Cells != 1000 {
		t.Fatalf("lost counts under concurrency: %+v", got)
	}
}
