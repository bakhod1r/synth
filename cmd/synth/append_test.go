package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Two appends of n rows produce 2n rows with a single header and no duplicated
// record — the whole point of append over re-running (which would overwrite and
// reproduce the same rows).
func TestAppendExtendsWithoutRepeating(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "s.yaml")
	os.WriteFile(spec, []byte(
		"name: users\ncount: 10\nseed: 7\nfields:\n  id: {kind: uuid, pk: true}\n  name: {kind: name}\n"), 0o600)
	out := filepath.Join(dir, "users.csv")

	for i := 0; i < 2; i++ {
		if err := runGen([]string{"-s", spec, "-o", out, "-n", "100", "--append", "--seed", "7"}); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	ids := columnValues(t, out, "id")
	if len(ids) != 200 {
		t.Fatalf("got %d rows, want 200", len(ids))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %q — the second run repeated the first", id)
		}
		seen[id] = true
	}

	// Exactly one header line.
	data, _ := os.ReadFile(out)
	if n := strings.Count(string(data), "id,name"); n != 1 {
		t.Errorf("found %d headers, want 1", n)
	}
}

// A first --append with no prior state behaves as a first run and leaves state
// behind for the next.
func TestAppendFirstRunSeedsState(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "s.yaml")
	os.WriteFile(spec, []byte("name: t\ncount: 5\nfields:\n  id: {kind: uuid}\n"), 0o600)
	out := filepath.Join(dir, "t.csv")
	if err := runGen([]string{"-s", spec, "-o", out, "-n", "5", "--append"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(statePath(out)); err != nil {
		t.Fatalf("no state written after first append: %v", err)
	}
	st, _ := readState(statePath(out))
	if st.Rows != 5 {
		t.Errorf("state rows = %d, want 5", st.Rows)
	}
}

func TestAppendSeedMismatchErrors(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "s.yaml")
	os.WriteFile(spec, []byte("name: t\ncount: 5\nfields:\n  id: {kind: uuid}\n"), 0o600)
	out := filepath.Join(dir, "t.csv")
	if err := runGen([]string{"-s", spec, "-o", out, "-n", "5", "--append", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}
	err := runGen([]string{"-s", spec, "-o", out, "-n", "5", "--append", "--seed", "2"})
	if err == nil || !strings.Contains(err.Error(), "seed") {
		t.Errorf("expected a seed-mismatch error, got %v", err)
	}
}

func TestAppendRejectsPgcopy(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "s.yaml")
	os.WriteFile(spec, []byte("name: t\ncount: 5\nfields:\n  id: {kind: uuid}\n"), 0o600)
	err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "t.pgbin"), "--append"})
	if err == nil {
		t.Error("expected --append to reject a binary COPY target")
	}
}
