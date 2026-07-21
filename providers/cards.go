package providers

import (
	"strconv"

	"github.com/bakhodir/synth/internal/rng"
)

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

var cardBrandKeys = func() []string {
	ks := make([]string, 0, len(creditCards))
	for k := range creditCards {
		ks = append(ks, k)
	}
	return ks
}()

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
