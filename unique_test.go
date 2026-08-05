package synth_test

import (
	"strings"
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

// A field whose value space is smaller than the row count must say so, rather
// than quietly emitting duplicates in a column declared unique.
func TestUniqueExhaustionErrors(t *testing.T) {
	type Row struct {
		Status string `synth:"enum,choices=new|open|closed,unique"`
	}
	_, err := synth.TryMake[Row](100, synth.WithSeed(4))
	if err == nil {
		t.Fatal("expected exhaustion error, got nil")
	}
	if !strings.Contains(err.Error(), "ran out of unique values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// unique=counter derives distinctness from the record index, so it holds at row
// counts far past what any value space could cover, without tracking state.
func TestUniqueCounter(t *testing.T) {
	type Row struct {
		Status string `synth:"enum,choices=new|open|closed,unique=counter"`
		Email  string `synth:"email,unique=counter"`
		ID     int    `synth:"int,unique=counter"`
	}
	const n = 200000
	recs := synth.Make[Row](n, synth.WithSeed(5))
	status := map[string]bool{}
	email := map[string]bool{}
	id := map[int]bool{}
	for _, r := range recs {
		if status[r.Status] || email[r.Email] || id[r.ID] {
			t.Fatalf("duplicate in counter-unique row %+v", r)
		}
		status[r.Status], email[r.Email], id[r.ID] = true, true, true
		// The suffix goes before the @, so the address stays an address.
		if strings.Count(r.Email, "@") != 1 || strings.HasSuffix(r.Email, "@") {
			t.Fatalf("counter broke email shape: %q", r.Email)
		}
	}
}

// Counter uniqueness needs no shared state, so parallel generation is allowed.
func TestUniqueCounterParallel(t *testing.T) {
	type Row struct {
		Username string `synth:"username,unique=counter"`
	}
	recs, err := synth.MakeParallel[Row](5000, 4, synth.WithSeed(6))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range recs {
		if seen[r.Username] {
			t.Fatalf("duplicate username %q", r.Username)
		}
		seen[r.Username] = true
	}
}

func TestUnknownUniqueMode(t *testing.T) {
	type Row struct {
		Name string `synth:"name,unique=sometimes"`
	}
	_, err := synth.TryMake[Row](1)
	if err == nil || !strings.Contains(err.Error(), "unknown unique mode") {
		t.Fatalf("expected unknown mode error, got %v", err)
	}
}
