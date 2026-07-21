package synth_test

import (
	"testing"

	"github.com/bakhodir/synth"
)

// Open-ended types must reach at least ~1000 distinct values with low
// repetition. Bounded real-world types (zodiac=12, continent=7) legitimately
// cannot and are not asserted here.
func TestOpenEndedCardinality1k(t *testing.T) {
	type Rec struct {
		ID     int
		Name   string // first × last (>=1000 combos in primary locales)
		Street string // house number × street (thousands)
		Card   string `synth:"card"`  // format-generated
		IBAN   string `synth:"iban"`  // format-generated
		EAN    string `synth:"ean13"` // format-generated
	}
	const n = 5000
	for _, loc := range []string{"en_US", "uz_UZ", "ru_RU", "de_DE", "fr_FR", "es_ES", "it_IT", "ja_JP", "tr_TR", "pt_BR", "nl_NL", "pl_PL", "ko_KR", "uk_UA"} {
		recs := synth.Make[Rec](n, synth.WithSeed(1), synth.WithLocale(loc))
		names := map[string]bool{}
		streets := map[string]bool{}
		cards := map[string]bool{}
		for _, r := range recs {
			names[r.Name] = true
			streets[r.Street] = true
			cards[r.Card] = true
		}
		if len(names) < 1000 {
			t.Errorf("%s: only %d distinct names (want >=1000)", loc, len(names))
		}
		if len(streets) < 1000 {
			t.Errorf("%s: only %d distinct streets (want >=1000)", loc, len(streets))
		}
		if len(cards) < 4500 {
			t.Errorf("%s: only %d distinct cards (want near-unique)", loc, len(cards))
		}
	}
}
