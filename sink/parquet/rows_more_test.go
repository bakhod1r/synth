package parquet_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	synthparquet "github.com/bakhod1r/synth/sink/parquet"
	pq "github.com/parquet-go/parquet-go"
)

// Map-shaped rows are what every non-Go frontend produces, so they have to
// survive the file with their column order and their values intact.
func TestWriteRowsRoundTripsEveryValueType(t *testing.T) {
	id := uuid.New()
	when := time.Date(2024, 6, 1, 12, 30, 0, 0, time.UTC)
	cols := []string{"name", "count", "ratio", "flag", "when", "uid", "missing"}
	rows := []map[string]any{
		{"name": "a", "count": 1, "ratio": 1.5, "flag": true, "when": when, "uid": id, "missing": nil},
		{"name": "b", "count": int64(2), "ratio": float32(2.5), "flag": false, "when": when, "uid": id},
	}

	path := filepath.Join(t.TempDir(), "rows.parquet")
	if err := synthparquet.WriteRows(path, cols, rows); err != nil {
		t.Fatal(err)
	}

	back := readRows(t, path)
	if len(back) != 2 {
		t.Fatalf("read %d rows, wrote 2", len(back))
	}
	if got := back[0]["name"]; got != "a" {
		t.Errorf("name = %#v", got)
	}
	if got := back[0]["count"]; got != int64(1) {
		t.Errorf("count = %#v, want an int64", got)
	}
	if got := back[0]["flag"]; got != true {
		t.Errorf("flag = %#v", got)
	}
	// A time is stored as RFC 3339 text so it reads back the same in any query
	// engine, without a timezone surprise.
	if got := back[0]["when"]; got != when.Format(time.RFC3339) {
		t.Errorf("when = %#v, want %q", got, when.Format(time.RFC3339))
	}
	// A uuid.UUID is a Stringer, not a scalar: it must not become a Go-syntax
	// dump.
	if got := back[0]["uid"]; got != id.String() {
		t.Errorf("uid = %#v, want %q", got, id.String())
	}
	// A nil in the first row decides the column's type; it must not crash and
	// must round-trip as an empty string.
	if got := back[0]["missing"]; got != "" {
		t.Errorf("missing = %#v, want an empty string", got)
	}
	if got := back[1]["ratio"]; got != 2.5 {
		t.Errorf("float32 did not widen to float64: %#v", got)
	}
	// A column absent from a row is as null as an explicit nil.
	if got := back[1]["missing"]; got != "" {
		t.Errorf("absent column = %#v", got)
	}
}

func TestWriteRowsRefusesAnEmptySet(t *testing.T) {
	err := synthparquet.WriteRows(filepath.Join(t.TempDir(), "x.parquet"), []string{"a"}, nil)
	if err == nil {
		t.Fatal("writing no rows should error: there is no schema to infer")
	}
}

func TestWriteRowsReportsAnUnwritablePath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "missing-dir", "x.parquet")
	err := synthparquet.WriteRows(bad, []string{"a"}, []map[string]any{{"a": "x"}})
	if err == nil {
		t.Fatal("expected a create error")
	}
}

func TestWriteStructsReportsAnUnwritablePath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "missing-dir", "x.parquet")
	type row struct {
		A string `parquet:"a"`
	}
	if err := synthparquet.WriteStructs(bad, []row{{A: "x"}}); err == nil {
		t.Fatal("expected a create error")
	}
}

// Only the columns named are written: a spec's column list is the contract, and
// an extra key in a row is not part of it.
func TestWriteRowsWritesOnlyTheNamedColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cols.parquet")
	rows := []map[string]any{{"keep": "yes", "drop": "no"}}
	if err := synthparquet.WriteRows(path, []string{"keep"}, rows); err != nil {
		t.Fatal(err)
	}
	back := readRows(t, path)
	if _, ok := back[0]["drop"]; ok {
		t.Fatalf("an unnamed column was written: %v", back[0])
	}
}

// readRows reads a map-shaped Parquet file back using the file's own schema —
// the reader cannot infer one from map[string]any.
func readRows(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	pf, err := pq.OpenFile(f, fi.Size())
	if err != nil {
		t.Fatal(err)
	}
	r := pq.NewGenericReader[map[string]any](f, pf.Schema())
	defer r.Close()
	out := make([]map[string]any, pf.NumRows())
	for i := range out {
		out[i] = map[string]any{}
	}
	if _, err := r.Read(out); err != nil && err.Error() != "EOF" {
		t.Fatal(err)
	}
	return out
}
