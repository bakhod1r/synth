package pgcopy

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTextWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewText(&buf, []string{"id", "name", "age", "score", "active", "created_at"})
	rows := []map[string]any{
		{
			"id": "a1", "name": "Aleksandr", "age": int64(30),
			"score": 99.5, "active": true,
			"created_at": time.Date(2026, 7, 22, 10, 30, 0, 0, time.UTC),
		},
		{
			"id": "a2", "name": nil, "age": int64(-1),
			"score": 0.0, "active": false,
			"created_at": nil,
		},
	}
	for _, r := range rows {
		if err := w.WriteRow(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	want := "a1\tAleksandr\t30\t99.5\tt\t2026-07-22T10:30:00Z\n" +
		"a2\t\\N\t-1\t0\tf\t\\N\n"
	if got := buf.String(); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// COPY text is delimited by real tabs and newlines, so a value containing one
// would silently shift every following column. These four escapes are the
// whole reason the format is not just "join with tabs".
func TestTextWriterEscapes(t *testing.T) {
	tests := map[string]string{
		"a\tb":  `a\tb`,
		"a\nb":  `a\nb`,
		"a\rb":  `a\rb`,
		`a\b`:   `a\\b`,
		`\N`:    `\\N`, // a literal backslash-N, not a NULL
		"plain": "plain",
		"":      "", // empty string, distinct from NULL
	}
	for in, want := range tests {
		var buf bytes.Buffer
		w := NewText(&buf, []string{"v"})
		if err := w.WriteRow(map[string]any{"v": in}); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSuffix(buf.String(), "\n"); got != want {
			t.Errorf("%q escaped to %q, want %q", in, got, want)
		}
	}
}

// An empty string and a NULL are different values in Postgres, and a loader
// that cannot tell them apart turns every missing value into "".
func TestTextWriterNullVersusEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := NewText(&buf, []string{"a", "b"})
	if err := w.WriteRow(map[string]any{"a": "", "b": nil}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "\t\\N\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A missing key is a NULL, not a panic: sparse rows come out of the JSON
// Schema and profiling frontends routinely.
func TestTextWriterMissingKey(t *testing.T) {
	var buf bytes.Buffer
	w := NewText(&buf, []string{"a", "b"})
	if err := w.WriteRow(map[string]any{"a": "x"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "x\t\\N\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
