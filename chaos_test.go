package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

func TestChaosInjectsEdgeCases(t *testing.T) {
	type Rec struct {
		ID   uuid.UUID `synth:"pk"`
		Name string
		Age  int
	}
	// High chaos rate so edge cases are very likely to appear in a sample.
	recs := synth.Make[Rec](2000, synth.WithSeed(1), synth.WithChaos(0.5))
	var sawLong, sawEmpty, sawBoundaryAge bool
	for _, r := range recs {
		if len(r.Name) > 1000 {
			sawLong = true
		}
		if r.Name == "" {
			sawEmpty = true
		}
		if r.Age < 0 || r.Age == 0 {
			sawBoundaryAge = true
		}
	}
	if !sawLong || !sawEmpty || !sawBoundaryAge {
		t.Fatalf("chaos did not surface edge cases: long=%v empty=%v boundary=%v", sawLong, sawEmpty, sawBoundaryAge)
	}
}

// Chaos must never corrupt referential keys.
func TestChaosPreservesRefs(t *testing.T) {
	type User struct {
		ID uuid.UUID `synth:"pk"`
	}
	type Order struct {
		ID     uuid.UUID `synth:"pk"`
		UserID uuid.UUID
	}
	users := synth.Make[User](50, synth.WithSeed(1))
	orders := synth.Make[Order](1000, synth.WithSeed(2), synth.Ref(users, "UserID"), synth.WithChaos(0.9))
	valid := map[uuid.UUID]bool{}
	for _, u := range users {
		valid[u.ID] = true
	}
	for _, o := range orders {
		if !valid[o.UserID] {
			t.Fatal("chaos corrupted a foreign key")
		}
	}
}

// Zero chaos leaves data clean (no 10k-char strings).
func TestNoChaosByDefault(t *testing.T) {
	type Rec struct {
		ID   int
		Name string
	}
	for _, r := range synth.Make[Rec](500, synth.WithSeed(3)) {
		if len(r.Name) > 1000 || strings.Contains(r.Name, "DROP TABLE") {
			t.Fatal("edge case leaked without WithChaos")
		}
	}
}
