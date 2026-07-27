package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.csv.synthstate")
	want := genState{Rows: 1000, Seed: 42}
	if err := writeState(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

// A first --append with no prior state is a clean first run, not an error: the
// user should not have to know whether the file was seeded with --append.
func TestReadStateAbsentIsZero(t *testing.T) {
	got, err := readState(filepath.Join(t.TempDir(), "nope.synthstate"))
	if err != nil {
		t.Fatalf("absent state should not error: %v", err)
	}
	if got != (genState{}) {
		t.Errorf("absent state = %+v, want zero", got)
	}
}

func TestStatePath(t *testing.T) {
	if got := statePath("users.csv"); got != "users.csv.synthstate" {
		t.Errorf("statePath = %q", got)
	}
	if got := statePath("users.jsonl.gz"); got != "users.jsonl.gz.synthstate" {
		t.Errorf("statePath = %q", got)
	}
}

func TestReadStateRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.synthstate")
	os.WriteFile(path, []byte("not json"), 0o600)
	if _, err := readState(path); err == nil {
		t.Error("garbage state should error, not silently reset the offset to 0")
	}
}
