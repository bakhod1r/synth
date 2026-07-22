package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The data file is useless without a table to load it into, so gen writes both.
func TestGenPgCopyBinaryWritesDataAndDDL(t *testing.T) {
	out := filepath.Join(t.TempDir(), "users.pgbin")
	if err := runGen([]string{"-s", "../../examples/users.yaml", "-o", out, "-n", "5"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("PGCOPY\n\377\r\n\x00")) {
		t.Error("data file is missing the binary COPY signature")
	}
	if !bytes.HasSuffix(data, []byte{0xFF, 0xFF}) {
		t.Error("data file is missing the trailer, so it is truncated")
	}

	ddl, err := os.ReadFile(out + ".sql")
	if err != nil {
		t.Fatalf("no DDL was written alongside the data: %v", err)
	}
	for _, want := range []string{"CREATE TABLE", "WITH (FORMAT binary)", out} {
		if !strings.Contains(string(ddl), want) {
			t.Errorf("DDL missing %q:\n%s", want, ddl)
		}
	}
}

func TestGenPgCopyTextRoundTrip(t *testing.T) {
	out := filepath.Join(t.TempDir(), "users.pgcopy")
	if err := runGen([]string{"-s", "../../examples/users.yaml", "-o", out, "-n", "5"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d rows, want 5", len(lines))
	}
	// 11 columns in the example spec, and every row must have the same count —
	// a short row means an unescaped tab or newline shifted the columns.
	for i, l := range lines {
		if n := len(strings.Split(l, "\t")); n != 11 {
			t.Errorf("row %d has %d columns, want 11", i, n)
		}
	}
	ddl, err := os.ReadFile(out + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ddl), "FORMAT binary") {
		t.Error("text output was given the binary COPY command")
	}
}

// Binary data down a pipe is unusable and the DDL has nowhere to go, so this
// asks for a file rather than writing something the user cannot load.
func TestGenPgCopyBinaryRejectsStdout(t *testing.T) {
	err := runGen([]string{"-s", "../../examples/users.yaml", "-f", "pgcopy-binary", "-n", "1"})
	if err == nil {
		t.Fatal("expected an error when writing binary COPY to stdout")
	}
	if !strings.Contains(err.Error(), "-o") {
		t.Errorf("error should point at -o, got: %v", err)
	}
}

func TestPgCopyFormatFromExtension(t *testing.T) {
	for path, want := range map[string]string{
		"users.pgcopy":    "pgcopy",
		"users.pgbin":     "pgcopy-binary",
		"users.pgcopy.gz": "pgcopy",
	} {
		if got := formatFromExt(path); got != want {
			t.Errorf("formatFromExt(%q) = %q, want %q", path, got, want)
		}
	}
}
