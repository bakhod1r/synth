package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// genState is the sidecar an --append run reads and rewrites. It records the
// one fact the data file does not carry: how much has been generated, so the
// next run can continue rather than repeat.
type genState struct {
	// Rows is the record index the next run starts from. Each row is seeded
	// from its index, so continuing the index gives fresh, still-deterministic
	// rows instead of a byte-for-byte repeat of the first run.
	Rows int `json:"rows"`
	// Seed is the seed the file was generated with. An append with a different
	// seed would splice two incoherent halves together, so a mismatch is an
	// error the user has to resolve.
	Seed uint64 `json:"seed"`
}

// statePath is the sidecar's path for a given output file. It keeps the full
// name — users.jsonl.gz.synthstate — so the sidecar for a compressed file does
// not collide with one for the uncompressed name.
func statePath(out string) string { return out + ".synthstate" }

// readState reads the sidecar. An absent file is the zero state, not an error:
// a first --append against a file that was written without one should behave
// like a fresh run rather than refuse.
//
// Malformed JSON, though, IS an error. Silently resetting to the zero state
// there would restart the record index at 0 and reproduce the first run's rows
// — the exact duplication --append exists to prevent.
func readState(path string) (genState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return genState{}, nil
	}
	if err != nil {
		return genState{}, err
	}
	var s genState
	if err := json.Unmarshal(data, &s); err != nil {
		return genState{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return s, nil
}

func writeState(path string, s genState) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
