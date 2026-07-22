package providers

import (
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

func init() {
	registry[schema.KindCardBrand] = cardBrand
}

// cardBrand names the payment brand. When the field is linked to a card column
// with from=, the brand is derived from that number so the pair is coherent;
// otherwise it is drawn on its own.
func cardBrand(c Ctx) any {
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		if v, ok := c.Sibling(c.Field.From).(string); ok && v != "" {
			if brand := BrandOf(v); brand != "" {
				return brand
			}
		}
	}
	return creditCards[cardBrandKeys[c.Rand.Pick(len(cardBrandKeys))]].name
}

// creditCard describes a card brand: display name, total length, and valid
// issuer identification prefixes (IIN/BIN ranges).
type creditCard struct {
	name     string
	length   int
	prefixes []int
}

// creditCards holds real-world brand prefix ranges and lengths. The remaining
// digits are filled randomly and a Luhn check digit is appended, so every
// generated number both starts with a valid brand prefix and passes Luhn.
var creditCards = map[string]creditCard{
	"visa":             {"VISA", 16, []int{4539, 4556, 4916, 4532, 4929, 40240071, 4485, 4716, 4}},
	"mastercard":       {"MasterCard", 16, []int{51, 52, 53, 54, 55}},
	"american express": {"American Express", 15, []int{34, 37}},
	"discover":         {"Discover", 16, []int{6011}},
	"jcb":              {"JCB", 16, []int{3528, 3538, 3548, 3558, 3568, 3578, 3588}},
	"diners club":      {"Diners Club", 14, []int{36, 38, 39}},
}

// nationalSchemes are card systems Synth recognizes but does not draw at
// random: they belong to their own market, so they appear through a locale's
// BIN list rather than in a global brand lottery. BrandOf still has to know
// them, or a locale-issued card would come back with no brand at all.
var nationalSchemes = []creditCard{
	{"HUMO", 16, []int{9860}},
	{"UZCARD", 16, []int{8600}},
	{"Mir", 16, []int{2200, 2201, 2202, 2203, 2204}},
}

// allSchemes is every system BrandOf can identify: the global brands plus the
// national ones.
var allSchemes = func() []creditCard {
	out := append([]creditCard(nil), nationalSchemes...)
	for _, c := range creditCards {
		out = append(out, c)
	}
	return out
}()

var cardBrandKeys = func() []string {
	ks := make([]string, 0, len(creditCards))
	for k := range creditCards {
		ks = append(ks, k)
	}
	return ks
}()

// lengthForBIN reports how long a number starting with this issuer prefix must
// be. American Express is fifteen digits, not sixteen; padding every BIN to the
// same length produces numbers that no real validator accepts.
func lengthForBIN(bin string) int {
	best, bestLen := 16, -1
	for _, c := range allSchemes {
		for _, p := range c.prefixes {
			prefix := strconv.Itoa(p)
			if len(prefix) > bestLen && strings.HasPrefix(bin, prefix) {
				best, bestLen = c.length, len(prefix)
			}
		}
	}
	return best
}

// generateCard builds a Luhn-valid number for the named brand. If brand is
// empty or unknown, a random brand is chosen.
func generateCard(r *rng.Rand, brand string) string {
	c, ok := creditCards[brand]
	if !ok {
		c = creditCards[cardBrandKeys[r.Pick(len(cardBrandKeys))]]
	}
	prefix := strconv.Itoa(c.prefixes[r.Pick(len(c.prefixes))])
	// Fill up to length-1 digits, then append the Luhn check digit.
	body := prefix + r.Digits(c.length-1-len(prefix))
	return body + string(luhnCheck(body))
}

// BrandOf reports the card brand a number belongs to, by its issuer prefix.
// This is the same rule a payment gateway applies, so a generated brand column
// and its card column cannot disagree: the brand is read off the number rather
// than drawn separately and hoped to match.
//
// It returns "" when no known brand claims the prefix.
func BrandOf(number string) string {
	digits := onlyDigits(number)
	best, bestLen := "", -1
	for _, c := range allSchemes {
		if len(digits) != c.length {
			continue
		}
		for _, p := range c.prefixes {
			prefix := strconv.Itoa(p)
			// Longest matching prefix wins: "4" and "4539" both claim a Visa,
			// and a two-digit brand must not out-rank a four-digit one.
			if len(prefix) > bestLen && strings.HasPrefix(digits, prefix) {
				best, bestLen = c.name, len(prefix)
			}
		}
	}
	return best
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
