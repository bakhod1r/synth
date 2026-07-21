package providers

import (
	"fmt"
	"math"
	"time"

	"github.com/bakhodir/synth/schema"
)

// Card expiry, security codes and account balances.

func init() {
	registry[schema.KindCardExpiry] = cardExpiry
	registry[schema.KindCVV] = cvv
	registry[schema.KindBalance] = balance
}

// expiryAnchor is the "now" expiry dates are measured from. It is fixed rather
// than time.Now() for the same reason every other date in Synth is: the same
// seed must produce the same data next year as it does today, and a test with
// a golden file must not start failing at midnight.
var expiryAnchor = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// cardExpiry returns a card expiry date. It is in the future by default,
// because a card that has already lapsed is an edge case you should ask for
// rather than receive by accident.
//
// Params:
//
//	expired=true    a date in the past instead
//	years=5         how far ahead (or behind) to reach
//	format=MM/YY    also MM/YYYY, YYYY-MM
func cardExpiry(c Ctx) any {
	years, ok := intParam(c.Params, "years")
	if !ok || years <= 0 {
		years = 4
	}
	months := c.Rand.IntRange(1, years*12)
	when := expiryAnchor.AddDate(0, months, 0)
	if boolParam(c.Params, "expired", false) {
		when = expiryAnchor.AddDate(0, -months, 0)
	}

	switch c.Params["format"] {
	case "MM/YYYY":
		return fmt.Sprintf("%02d/%04d", int(when.Month()), when.Year())
	case "YYYY-MM":
		return fmt.Sprintf("%04d-%02d", when.Year(), int(when.Month()))
	default:
		return fmt.Sprintf("%02d/%02d", int(when.Month()), when.Year()%100)
	}
}

// cvv returns a security code. American Express uses four digits and everyone
// else uses three, so when the field is linked to a card column with from= the
// length follows that card rather than being uniformly wrong a fraction of the
// time.
func cvv(c Ctx) any {
	digits := 3
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		if number, ok := c.Sibling(c.Field.From).(string); ok {
			if BrandOf(number) == "American Express" {
				digits = 4
			}
		}
	}
	if n, ok := intParam(c.Params, "digits"); ok && (n == 3 || n == 4) {
		digits = n
	}
	return c.Rand.Digits(digits)
}

// balance returns an account balance.
//
// Unlike an amount, a balance can be negative: overdrafts, refunds and
// corrections all produce them, and code that assumes a balance is positive
// breaks the first time it meets one. A small share of negative values is
// generated on purpose so that path gets exercised.
//
// Params: min, max, negative (share of overdrawn accounts, default 0.05),
// currency-style rounding is always to two decimal places.
func balance(c Ctx) any {
	lo, okLo := floatParam(c.Params, "min")
	hi, okHi := floatParam(c.Params, "max")
	if !okLo {
		lo = 0
	}
	if !okHi {
		hi = 25_000
	}
	if hi < lo {
		hi = lo
	}

	negShare := 0.05
	if v, ok := floatParam(c.Params, "negative"); ok && v >= 0 && v <= 1 {
		negShare = v
	}

	// An explicit negative floor means the caller is already asking for
	// overdrafts across the whole range; adding more on top would double-count.
	if lo < 0 {
		negShare = 0
	}

	v := lo + c.Rand.Float64()*(hi-lo)
	if c.Rand.Bool(negShare) {
		// Overdrafts are shallow next to balances; a large negative number is
		// a different scenario and should be asked for with min.
		v = -c.Rand.Float64() * math.Max(hi*0.05, 100)
	}
	return math.Round(v*100) / 100
}

func floatParam(params map[string]string, key string) (float64, bool) {
	v, ok := params[key]
	if !ok || v == "" {
		return 0, false
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%g", &f); err != nil {
		return 0, false
	}
	return f, true
}
