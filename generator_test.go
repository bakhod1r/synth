package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

func TestGeneratorFluent(t *testing.T) {
	g := synth.New(synth.Config{Seed: 1, Locale: "uz_UZ"})
	if !strings.HasPrefix(g.Phone(), "+998") {
		t.Fatal("uz phone")
	}
	if g.Currency() != "UZS" {
		t.Fatalf("want UZS, got %s", g.Currency())
	}
	if g.Name() == "" || g.Company() == "" || g.Username() == "" {
		t.Fatal("empty fluent value")
	}
	amt := g.Amount(1000, 5000)
	if amt < 1000 || amt > 5000 {
		t.Fatalf("amount out of range: %v", amt)
	}
}

// Independent Generators with the same seed produce the same sequence.
func TestGeneratorDeterministic(t *testing.T) {
	a := synth.New(synth.Config{Seed: 5})
	b := synth.New(synth.Config{Seed: 5})
	for i := 0; i < 20; i++ {
		if a.Email() != b.Email() {
			t.Fatal("same-seed generators diverged")
		}
	}
}
