package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
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

// Every generated movie must be a real, curated title (no synthesized values).
func TestDatasetsAreReal(t *testing.T) {
	type M struct {
		ID    int
		Movie string `synth:"movie"`
	}
	realTitles := map[string]bool{
		"The Shawshank Redemption": true, "The Godfather": true, "Inception": true, "Pulp Fiction": true,
		"The Dark Knight": true, "Forrest Gump": true, "Interstellar": true, "Fight Club": true,
		"The Matrix": true, "Goodfellas": true, "Parasite": true, "Whiplash": true,
		"Gladiator": true, "Titanic": true, "Dune": true, "Oppenheimer": true,
		"Schindler's List": true, "The Lord of the Rings": true, "Se7en": true, "The Silence of the Lambs": true,
		"Saving Private Ryan": true, "The Green Mile": true, "Léon: The Professional": true, "The Prestige": true,
		"Casablanca": true, "Spirited Away": true, "Django Unchained": true, "The Departed": true,
		"Joker": true, "1917": true, "La La Land": true, "Mad Max: Fury Road": true,
		"No Country for Old Men": true, "There Will Be Blood": true, "The Grand Budapest Hotel": true, "Blade Runner 2049": true,
		"Everything Everywhere All at Once": true, "Get Out": true, "Arrival": true, "Her": true,
		"The Social Network": true, "Toy Story": true, "Coco": true, "Up": true,
		"Amélie": true, "City of God": true, "Oldboy": true, "Memento": true,
	}
	for _, r := range synth.Make[M](3000, synth.WithSeed(2)) {
		if !realTitles[r.Movie] {
			t.Fatalf("non-real movie value: %q", r.Movie)
		}
	}
}
