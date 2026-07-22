package providers_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/providers"
)

// A brand column sitting next to a card column must describe that card. A
// "MasterCard" label on a 4539… number is exactly the incoherence that makes
// fake data useless for payment code, and it is the kind of thing a faker gets
// wrong because it draws the two independently.
func TestCardBrandMatchesTheNumber(t *testing.T) {
	type Payment struct {
		Card  string `synth:"card"`
		Brand string `synth:"cardbrand"`
	}
	rows := synth.Make[Payment](500, synth.WithSeed(1))
	for i, p := range rows {
		want := providers.BrandOf(p.Card)
		if want == "" {
			t.Fatalf("row %d: generated card %q belongs to no known brand", i, p.Card)
		}
		if p.Brand != want {
			t.Fatalf("row %d: card %q is a %s but the brand column says %q",
				i, p.Card, want, p.Brand)
		}
	}
}

// The link must be made without the user wiring it: inference sees a card
// column and a brand column and connects them.
func TestCardBrandLinksWithoutTags(t *testing.T) {
	type Payment struct {
		CardNumber string
		CardBrand  string
	}
	for i, p := range synth.Make[Payment](200, synth.WithSeed(3)) {
		if got := providers.BrandOf(p.CardNumber); got != p.CardBrand {
			t.Fatalf("row %d: untagged columns disagree — %q is %q, column says %q",
				i, p.CardNumber, got, p.CardBrand)
		}
	}
}

// Asking for a brand explicitly must produce a number from that brand's real
// issuer range, not a relabelled random number.
func TestCardBrandParamIsHonored(t *testing.T) {
	type Card struct {
		Number string `synth:"card,brand=american express"`
		Brand  string `synth:"cardbrand,from=Number"`
	}
	for _, c := range synth.Make[Card](200, synth.WithSeed(5)) {
		if len(c.Number) != 15 {
			t.Fatalf("Amex numbers are 15 digits, got %q", c.Number)
		}
		if !strings.HasPrefix(c.Number, "34") && !strings.HasPrefix(c.Number, "37") {
			t.Fatalf("%q is not in an American Express issuer range", c.Number)
		}
		if c.Brand != "American Express" {
			t.Fatalf("brand column says %q", c.Brand)
		}
	}
}

// BrandOf must prefer the longest matching prefix. Visa claims both "4" and
// "4539"; a shorter, coincidental match must not win.
func TestBrandOfPrefersLongestPrefix(t *testing.T) {
	cases := map[string]string{
		"4539578763621486": "VISA",
		"5555555555554444": "MasterCard",
		"378282246310005":  "American Express",
		"6011111111111117": "Discover",
		"36227206271667":   "Diners Club",
		"":                 "",
		"1234":             "",
		"not-a-card":       "",
	}
	for number, want := range cases {
		if got := providers.BrandOf(number); got != want {
			t.Errorf("BrandOf(%q) = %q, want %q", number, got, want)
		}
	}
}

// A brand column on its own still has to produce a real brand name.
func TestCardBrandAloneIsAValidBrand(t *testing.T) {
	type Row struct {
		Brand string `synth:"cardbrand"`
	}
	known := map[string]bool{
		"VISA": true, "MasterCard": true, "American Express": true,
		"Discover": true, "JCB": true, "Diners Club": true,
	}
	for _, r := range synth.Make[Row](100, synth.WithSeed(7)) {
		if !known[r.Brand] {
			t.Fatalf("unknown card brand %q", r.Brand)
		}
	}
}

// Locale-issued cards must be recognized too. An Uzbek card is HUMO or UZCARD,
// and a brand column next to it must say so rather than falling back to a
// global brand that the number does not belong to.
func TestLocaleCardsAreRecognized(t *testing.T) {
	type Payment struct {
		Card  string `synth:"card"`
		Brand string `synth:"cardbrand"`
	}
	for _, tc := range []struct {
		locale string
		brands map[string]bool
	}{
		{"uz_UZ", map[string]bool{"HUMO": true, "UZCARD": true}},
		{"ru_RU", map[string]bool{"Mir": true, "VISA": true}},
	} {
		for i, p := range synth.Make[Payment](200, synth.WithSeed(2), synth.WithLocale(tc.locale)) {
			if !tc.brands[p.Brand] {
				t.Fatalf("%s row %d: card %q got brand %q, which is not issued there",
					tc.locale, i, p.Card, p.Brand)
			}
			if providers.BrandOf(p.Card) != p.Brand {
				t.Fatalf("%s row %d: %q is %q but the column says %q",
					tc.locale, i, p.Card, providers.BrandOf(p.Card), p.Brand)
			}
		}
	}
}

// Every generated card must have the length its scheme actually uses. An
// Amex-prefixed number padded to sixteen digits fails validation everywhere.
func TestCardLengthMatchesItsScheme(t *testing.T) {
	type Card struct {
		Number string `synth:"card"`
	}
	want := map[string]int{
		"American Express": 15, "Diners Club": 14,
		"VISA": 16, "MasterCard": 16, "Discover": 16, "JCB": 16,
		"HUMO": 16, "UZCARD": 16, "Mir": 16,
	}
	for _, locale := range []string{"en_US", "uz_UZ", "ru_RU"} {
		for i, c := range synth.Make[Card](500, synth.WithSeed(11), synth.WithLocale(locale)) {
			brand := providers.BrandOf(c.Number)
			if brand == "" {
				t.Fatalf("%s row %d: %q belongs to no scheme", locale, i, c.Number)
			}
			if len(c.Number) != want[brand] {
				t.Fatalf("%s row %d: %s number %q is %d digits, want %d",
					locale, i, brand, c.Number, len(c.Number), want[brand])
			}
		}
	}
}
