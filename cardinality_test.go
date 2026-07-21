package synth_test

import (
	"testing"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

func TestOneToMany(t *testing.T) {
	type User struct {
		ID uuid.UUID `synth:"pk"`
	}
	type Order struct {
		ID     uuid.UUID `synth:"pk"`
		UserID uuid.UUID
	}
	users := synth.Make[User](20, synth.WithSeed(1))
	orders := synth.Make[Order](2000, synth.WithSeed(2),
		synth.Ref(users, "UserID", synth.OneToMany(2, 8)))

	counts := map[uuid.UUID]int{}
	valid := map[uuid.UUID]bool{}
	for _, u := range users {
		valid[u.ID] = true
	}
	for _, o := range orders {
		if !valid[o.UserID] {
			t.Fatal("order references non-existent user")
		}
		counts[o.UserID]++
	}
	// Every parent that can appear in the pool should own at least one child,
	// and the spread should not collapse onto a single parent.
	if len(counts) < 10 {
		t.Fatalf("cardinality too concentrated: only %d parents used", len(counts))
	}
}
