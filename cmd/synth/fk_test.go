package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

// The cross-run flow: write a parent file, then generate a child whose FK is
// drawn from the parent's key column.
func TestGenFKFromParentFile(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "users.csv")
	if err := runGen([]string{"--preset", "user", "-o", parent, "-n", "50"}); err != nil {
		t.Fatal(err)
	}
	parentIDs := columnSet(t, parent, "id")

	childSpec := filepath.Join(dir, "orders.yaml")
	os.WriteFile(childSpec, []byte(
		"name: orders\ncount: 5\nfields:\n  id: {kind: uuid, pk: true}\n  user_id: {kind: uuid}\n"), 0o600)
	child := filepath.Join(dir, "orders.csv")
	err := runGen([]string{"-s", childSpec, "-o", child, "-n", "200",
		"--fk", "user_id=" + parent + ":id"})
	if err != nil {
		t.Fatal(err)
	}

	for i, id := range columnValues(t, child, "user_id") {
		if !parentIDs[id] {
			t.Fatalf("child row %d: user_id %q is not a parent id", i, id)
		}
	}
}

func TestGenFKErrors(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "s.yaml")
	os.WriteFile(spec, []byte("name: t\ncount: 1\nfields:\n  fk: {kind: uuid}\n"), 0o600)
	parent := filepath.Join(dir, "p.csv")
	runGen([]string{"--preset", "user", "-o", parent, "-n", "3"})

	for _, tc := range []struct{ name, fk string }{
		{"missing file", "fk=/no/such.csv:id"},
		{"missing key column", "fk=" + parent + ":nope"},
		{"malformed", "fk=" + parent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(dir, "o.csv")
			if err := runGen([]string{"-s", spec, "-o", out, "--fk", tc.fk}); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func columnValues(t *testing.T, path, col string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	idx := -1
	for i, h := range rows[0] {
		if h == col {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("column %q not found in %s", col, path)
	}
	var out []string
	for _, r := range rows[1:] {
		out = append(out, r[idx])
	}
	return out
}

func columnSet(t *testing.T, path, col string) map[string]bool {
	set := map[string]bool{}
	for _, v := range columnValues(t, path, col) {
		set[v] = true
	}
	return set
}
