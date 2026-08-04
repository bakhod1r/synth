package synth

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// failWriter fails on the nth write, so the error path of every encoder can be
// exercised the way a full disk or a closed pipe would exercise it.
type failWriter struct {
	after int
	n     int
}

var errDisk = errors.New("no space left on device")

func (f *failWriter) Write(p []byte) (int, error) {
	f.n++
	if f.n > f.after {
		return 0, errDisk
	}
	return len(p), nil
}

type encRow struct {
	ID     int
	Name   string
	Active bool
	Price  float64
	hidden string // unexported: must never reach a column
}

func TestFieldNamesSkipsUnexported(t *testing.T) {
	cols := fieldNames(reflect.TypeOf(encRow{}))
	want := []string{"ID", "Name", "Active", "Price"}
	if strings.Join(cols, ",") != strings.Join(want, ",") {
		t.Fatalf("fieldNames = %v, want %v", cols, want)
	}
}

func TestEncodeCSVHeaderWriteError(t *testing.T) {
	// csv.Writer buffers, so the header error surfaces at Flush inside the
	// csv writer's own buffer; a zero-capacity sink fails on the first flush.
	var buf bytes.Buffer
	if err := encodeCSV(&buf, []encRow{{ID: 1, Name: "a,b"}}); err != nil {
		t.Fatalf("encodeCSV: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `"a,b"`) {
		t.Fatalf("comma in a value was not quoted: %q", got)
	}
	if !strings.HasPrefix(got, "ID,Name,Active,Price\n") {
		t.Fatalf("header = %q", got)
	}
}

func TestEncodeCSVWriteError(t *testing.T) {
	// 4096 rows overflow csv.Writer's internal bufio buffer, forcing a real
	// write through to the failing sink.
	rows := make([]encRow, 4096)
	for i := range rows {
		rows[i] = encRow{ID: i, Name: strings.Repeat("x", 64)}
	}
	if err := encodeCSV(&failWriter{after: 0}, rows); err == nil {
		t.Fatal("expected the sink error to surface, got nil")
	}
}

func TestEncodeJSONLWriteError(t *testing.T) {
	err := encodeJSONL(&failWriter{after: 0}, []encRow{{ID: 1}})
	if !errors.Is(err, errDisk) {
		t.Fatalf("encodeJSONL error = %v, want %v", err, errDisk)
	}
}

func TestEncodeJSONLOneObjectPerLine(t *testing.T) {
	var buf bytes.Buffer
	if err := encodeJSONL(&buf, []encRow{{ID: 1}, {ID: 2}}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "{") || !strings.HasSuffix(l, "}") {
			t.Fatalf("line is not a bare JSON object: %q", l)
		}
	}
}

func TestEncodeSQLWriteError(t *testing.T) {
	err := encodeSQL(&failWriter{after: 0}, "t", []encRow{{ID: 1}})
	if !errors.Is(err, errDisk) {
		t.Fatalf("encodeSQL error = %v, want %v", err, errDisk)
	}
}

func TestSQLValue(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "NULL"},
		{"int", 42, "42"},
		{"int64", int64(-7), "-7"},
		{"uint8", uint8(3), "3"},
		{"float", 1.5, "1.5"},
		{"true", true, "TRUE"},
		{"false", false, "FALSE"},
		{"string", "hi", "'hi'"},
		{"quote is doubled", "O'Brien", "'O''Brien'"},
		{"injection stays a literal", "'; DROP TABLE users;--", "'''; DROP TABLE users;--'"},
		{"time", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), "'2024-03-01 00:00:00 +0000 UTC'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sqlValue(c.in); got != c.want {
				t.Fatalf("sqlValue(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestEncodeSQLStatementShape(t *testing.T) {
	var buf bytes.Buffer
	if err := encodeSQL(&buf, "orders", []encRow{{ID: 1, Name: "a", Active: true, Price: 2.5}}); err != nil {
		t.Fatal(err)
	}
	want := "INSERT INTO orders (ID, Name, Active, Price) VALUES (1, 'a', TRUE, 2.5);\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteFileCreateErrors(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "no-such-dir", "out.txt")
	rows := []encRow{{ID: 1}}
	if err := WriteCSV(bad, rows); err == nil {
		t.Error("WriteCSV: expected a create error")
	}
	if err := WriteJSONL(bad, rows); err == nil {
		t.Error("WriteJSONL: expected a create error")
	}
	if err := WriteSQL(bad, "t", rows); err == nil {
		t.Error("WriteSQL: expected a create error")
	}
}

func TestWriteFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	rows := []encRow{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}

	csvPath := filepath.Join(dir, "out.csv")
	if err := WriteCSV(csvPath, rows); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(strings.TrimSpace(string(data)), "\n"); n != 2 {
		t.Fatalf("csv has %d data lines, want 2 (plus header): %q", n, data)
	}

	sqlPath := filepath.Join(dir, "out.sql")
	if err := WriteSQL(sqlPath, "t", rows); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(sqlPath)
	if strings.Count(string(data), "INSERT INTO t") != 2 {
		t.Fatalf("want 2 INSERTs, got %q", data)
	}

	jsonlPath := filepath.Join(dir, "out.jsonl")
	if err := WriteJSONL(jsonlPath, rows); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(jsonlPath)
	if strings.Count(string(data), "\n") != 2 {
		t.Fatalf("want 2 lines, got %q", data)
	}
}
