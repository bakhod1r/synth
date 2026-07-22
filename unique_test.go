package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/google/uuid"
)

func TestUniqueTag(t *testing.T) {
	type Acc struct {
		ID       int
		Username string `synth:"username,unique"`
	}
	recs := synth.Make[Acc](500, synth.WithSeed(1))
	seen := map[string]bool{}
	for _, r := range recs {
		if seen[r.Username] {
			t.Fatalf("duplicate username %q", r.Username)
		}
		seen[r.Username] = true
	}
}

// Integer PKs must be unique across a large dataset.
func TestUniqueIntPK(t *testing.T) {
	type Row struct {
		ID int `synth:"pk"`
	}
	recs := synth.Make[Row](50000, synth.WithSeed(2))
	seen := map[int]bool{}
	for _, r := range recs {
		if seen[r.ID] {
			t.Fatal("duplicate int PK")
		}
		seen[r.ID] = true
	}
}

// UUID PKs are unique by construction.
func TestUniqueUUIDPK(t *testing.T) {
	type Row struct {
		ID uuid.UUID `synth:"pk"`
	}
	recs := synth.Make[Row](10000, synth.WithSeed(3))
	seen := map[uuid.UUID]bool{}
	for _, r := range recs {
		if seen[r.ID] {
			t.Fatal("duplicate uuid PK")
		}
		seen[r.ID] = true
	}
}

func TestParallelRejectsUnique(t *testing.T) {
	type Acc struct {
		Username string `synth:"username,unique"`
	}
	if _, err := synth.MakeParallel[Acc](100, 4); err == nil {
		t.Fatal("expected error for unique field in MakeParallel")
	}
}
