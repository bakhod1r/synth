package providers_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/bakhodir/synth"
)

// An expiry date is in the future unless a lapsed card was asked for. Getting
// an expired card by accident turns a happy-path fixture into a failing one.
func TestCardExpiryIsInTheFuture(t *testing.T) {
	type Card struct {
		Expiry string `synth:"cardexpiry"`
	}
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range synth.Make[Card](500, synth.WithSeed(1)) {
		when, err := time.Parse("01/06", c.Expiry)
		if err != nil {
			t.Fatalf("row %d: %q is not MM/YY: %v", i, c.Expiry, err)
		}
		if !when.After(anchor) {
			t.Fatalf("row %d: %q has already expired", i, c.Expiry)
		}
	}
}

// Asking for an expired card must actually give one.
func TestExpiredCardExpiry(t *testing.T) {
	type Card struct {
		Expiry string `synth:"cardexpiry,expired=true"`
	}
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range synth.Make[Card](300, synth.WithSeed(2)) {
		when, _ := time.Parse("01/06", c.Expiry)
		if when.After(anchor) {
			t.Fatalf("row %d: %q is still valid", i, c.Expiry)
		}
	}
}

// The output must be stable across runs, so a golden file does not start
// failing when the calendar moves.
func TestCardExpiryIsDeterministic(t *testing.T) {
	type Card struct {
		Expiry string `synth:"cardexpiry"`
	}
	a := synth.Make[Card](50, synth.WithSeed(3))
	b := synth.Make[Card](50, synth.WithSeed(3))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs between runs: %q vs %q", i, a[i].Expiry, b[i].Expiry)
		}
	}
}

func TestCardExpiryFormats(t *testing.T) {
	type Long struct {
		E string `synth:"cardexpiry,format=MM/YYYY"`
	}
	type ISO struct {
		E string `synth:"cardexpiry,format=YYYY-MM"`
	}
	for _, c := range synth.Make[Long](50, synth.WithSeed(4)) {
		if len(c.E) != 7 || c.E[2] != '/' {
			t.Fatalf("MM/YYYY expected, got %q", c.E)
		}
	}
	for _, c := range synth.Make[ISO](50, synth.WithSeed(5)) {
		if len(c.E) != 7 || c.E[4] != '-' {
			t.Fatalf("YYYY-MM expected, got %q", c.E)
		}
	}
}

// American Express uses a four-digit code and everyone else three. Linked to a
// card column, the CVV must follow that card rather than being wrong a fixed
// share of the time.
func TestCVVLengthFollowsTheCard(t *testing.T) {
	type Payment struct {
		Card  string `synth:"card"`
		Brand string `synth:"cardbrand"`
		CVV   string `synth:"cvv"`
	}
	sawAmex := false
	for i, p := range synth.Make[Payment](1000, synth.WithSeed(6)) {
		want := 3
		if p.Brand == "American Express" {
			want, sawAmex = 4, true
		}
		if len(p.CVV) != want {
			t.Fatalf("row %d: %s card has a %d-digit CVV %q, want %d",
				i, p.Brand, len(p.CVV), p.CVV, want)
		}
		if strings.IndexFunc(p.CVV, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			t.Fatalf("row %d: CVV %q is not all digits", i, p.CVV)
		}
	}
	if !sawAmex {
		t.Fatal("no Amex cards were generated — the test proves nothing")
	}
}

// A balance may be negative: overdrafts exist, and code that assumes otherwise
// breaks the first time it meets one.
func TestBalanceIncludesOverdrafts(t *testing.T) {
	type Account struct {
		Balance float64 `synth:"balance"`
	}
	rows := synth.Make[Account](5000, synth.WithSeed(7))
	negative := 0
	for _, a := range rows {
		if a.Balance < 0 {
			negative++
		}
		// Money has two decimal places; a third would break currency columns.
		// The comparison allows for float64's inability to hold 183.06 exactly
		// — the value must be *a* cent amount, not exactly representable.
		if cents := a.Balance * 100; math.Abs(cents-math.Round(cents)) > 1e-6 {
			t.Fatalf("balance %v has sub-cent precision", a.Balance)
		}
	}
	if negative == 0 {
		t.Fatal("no overdrawn accounts in 5000 rows — the negative path is never exercised")
	}
	if share := float64(negative) / float64(len(rows)); share > 0.15 {
		t.Fatalf("%.1f%% of accounts are overdrawn, which is not realistic", share*100)
	}
}

// Explicit bounds must be respected.
func TestBalanceRespectsBounds(t *testing.T) {
	type Account struct {
		Balance float64 `synth:"balance,min=100,max=200,negative=0"`
	}
	for _, a := range synth.Make[Account](500, synth.WithSeed(8)) {
		if a.Balance < 100 || a.Balance > 200 {
			t.Fatalf("balance %v is outside 100..200", a.Balance)
		}
	}
}

// Untagged columns must be inferred from their names.
func TestFinanceFieldsInferFromNames(t *testing.T) {
	type Account struct {
		CardNumber string
		Expiry     string
		CVV        string
		Balance    float64
	}
	for i, a := range synth.Make[Account](200, synth.WithSeed(9)) {
		if !strings.Contains(a.Expiry, "/") {
			t.Fatalf("row %d: Expiry %q was not inferred as a card expiry", i, a.Expiry)
		}
		if len(a.CVV) < 3 || len(a.CVV) > 4 {
			t.Fatalf("row %d: CVV %q was not inferred", i, a.CVV)
		}
		if a.CardNumber == "" {
			t.Fatalf("row %d: CardNumber is empty", i)
		}
	}
}

// The security code has a different name on every network. Someone typing the
// name their own payment provider uses should find it rather than conclude
// Synth has no such type.
func TestSecurityCodeAliases(t *testing.T) {
	for _, alias := range []string{"cvv", "cvc", "cvv2", "cvc2", "csc", "cid"} {
		y, err := synth.YAMLBytes([]byte(
			"name: t\ncount: 50\nseed: 7\nfields:\n  c: { kind: " + alias + " }\n"))
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		rows, err := y.Generate()
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		for i, r := range rows {
			v, _ := r["c"].(string)
			if len(v) != 3 || strings.Trim(v, "0123456789") != "" {
				t.Fatalf("%s row %d: %q is not a 3-digit code", alias, i, v)
			}
		}
	}
}

// Every alias must follow the card it is linked to, not just the plain form.
// Amex uses four digits; a fixture that is uniformly wrong a fifth of the time
// makes a length validator look flaky.
func TestAliasesFollowTheCardLength(t *testing.T) {
	for _, alias := range []string{"cvv", "cvc", "cid"} {
		y, err := synth.YAMLBytes([]byte(
			"name: t\ncount: 50\nseed: 7\nfields:\n" +
				"  card: { kind: card, brand: american express }\n" +
				"  c: { kind: " + alias + ", from: card }\n"))
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		rows, _ := y.Generate()
		for i, r := range rows {
			if v := r["c"].(string); len(v) != 4 {
				t.Fatalf("%s row %d: %q is %d digits beside an Amex card, want 4",
					alias, i, v, len(v))
			}
		}
	}
}
