package synth_test

import (
	"testing"

	"github.com/bakhodir/synth"
)

func TestRealDatasetsInferred(t *testing.T) {
	type Media struct {
		ID        int
		Book      string
		Movie     string
		Celebrity string
		Food      string
		Animal    string
		Sport     string
		Language  string
	}
	recs := synth.Make[Media](50, synth.WithSeed(1))
	for _, r := range recs {
		if r.Book == "" || r.Movie == "" || r.Celebrity == "" ||
			r.Food == "" || r.Animal == "" || r.Sport == "" || r.Language == "" {
			t.Fatalf("empty dataset field: %+v", r)
		}
	}
}

// Recognizable real titles must still appear, and the combinatorial space must
// keep repetition low across a large dataset (>=1000 distinct in 10k rows).
func TestDatasetsCardinality(t *testing.T) {
	type M struct {
		ID    int
		Movie string `synth:"movie"`
	}
	seen := map[string]bool{}
	real := 0
	known := map[string]bool{"Inception": true, "The Matrix": true, "Parasite": true, "Dune": true}
	for _, r := range synth.Make[M](10000, synth.WithSeed(2)) {
		seen[r.Movie] = true
		if known[r.Movie] {
			real++
		}
	}
	if real == 0 {
		t.Fatal("expected some recognizable real titles to appear")
	}
	if len(seen) < 1000 {
		t.Fatalf("too few distinct movies (%d); repetition too high", len(seen))
	}
}
