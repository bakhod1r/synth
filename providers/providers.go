// Package providers holds atomic value generators, one per schema.Kind.
// Each is a small function of (rng, locale, params, related-values) → value.
// Adding a new type means adding one entry here, touching nothing else.
package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/locale"
	"github.com/bakhodir/synth/schema"
	"github.com/google/uuid"
)

// Ctx carries everything a provider may need: the record's rng, its locale,
// the field's parsed params, and access to already-generated sibling values
// (for from=/match= coherence).
type Ctx struct {
	Rand   *rng.Rand
	Locale *locale.Locale
	Params map[string]string
	// Place is the record's chosen locale.Place, shared across city/region/
	// postcode/phone so they stay coherent within one record.
	Place *locale.Place
	// Sibling returns an already-generated field value by name (for from=).
	Sibling func(name string) any
}

// Provider produces one value.
type Provider func(c Ctx) any

var registry = map[schema.Kind]Provider{}

func init() {
	registry[schema.KindUUID] = func(c Ctx) any { return uuidFrom(c.Rand) }
	registry[schema.KindFirstName] = func(c Ctx) any { return pick(c.Rand, c.Locale.FirstNames) }
	registry[schema.KindLastName] = func(c Ctx) any { return pick(c.Rand, c.Locale.LastNames) }
	registry[schema.KindName] = func(c Ctx) any {
		return pick(c.Rand, c.Locale.FirstNames) + " " + pick(c.Rand, c.Locale.LastNames)
	}
	registry[schema.KindEmail] = email
	registry[schema.KindPhone] = phone
	registry[schema.KindRegion] = func(c Ctx) any { return c.Place.Region }
	registry[schema.KindCity] = func(c Ctx) any { return c.Place.City }
	registry[schema.KindPostcode] = func(c Ctx) any { return c.Place.Postcode }
	registry[schema.KindCountry] = func(c Ctx) any { return c.Locale.Country }
	registry[schema.KindInt] = intProvider
	registry[schema.KindFloat] = floatProvider
	registry[schema.KindBool] = func(c Ctx) any { return c.Rand.Bool(0.5) }
	registry[schema.KindTime] = timeProvider
	registry[schema.KindLorem] = lorem
	registry[schema.KindIBAN] = iban
	registry[schema.KindCard] = card
	registry[schema.KindPassport] = passport
	registry[schema.KindCompany] = func(c Ctx) any { return pick(c.Rand, c.Locale.Companies) }
	registry[schema.KindCurrency] = func(c Ctx) any { return c.Locale.Currency }
	registry[schema.KindUsername] = username
	registry[schema.KindIPv4] = ipv4
	registry[schema.KindURL] = urlProvider
	registry[schema.KindAmount] = amount
}

func username(c Ctx) any {
	first := strings.ToLower(pick(c.Rand, c.Locale.FirstNames))
	return fmt.Sprintf("%s_%s", first, c.Rand.Digits(3))
}

func ipv4(c Ctx) any {
	return fmt.Sprintf("%d.%d.%d.%d", c.Rand.IntRange(1, 223), c.Rand.Intn(256), c.Rand.Intn(256), c.Rand.IntRange(1, 254))
}

func urlProvider(c Ctx) any {
	return "https://" + strings.ToLower(strings.ReplaceAll(pick(c.Rand, c.Locale.Companies), " ", "")) + ".com"
}

// amount returns a monetary value in [min,max] rounded to 2 decimals.
func amount(c Ctx) any {
	min := paramInt(c.Params, "min", 1)
	max := paramInt(c.Params, "max", 100000)
	v := float64(min) + c.Rand.Float64()*float64(max-min)
	return float64(int(v*100)) / 100
}

// Get returns the provider for a kind, or nil if unknown.
func Get(k schema.Kind) Provider { return registry[k] }

func pick(r *rng.Rand, s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[r.Pick(len(s))]
}

func uuidFrom(r *rng.Rand) uuid.UUID {
	var b [16]byte
	hi, lo := r.Uint64(), r.Uint64()
	for i := 0; i < 8; i++ {
		b[i] = byte(hi >> (8 * i))
		b[8+i] = byte(lo >> (8 * i))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return uuid.UUID(b)
}

func email(c Ctx) any {
	var first, last string
	// Derive from a from= sibling name when available.
	if c.Sibling != nil {
		if v, ok := c.Sibling("__from__").(string); ok && v != "" {
			parts := strings.Fields(v)
			if len(parts) > 0 {
				first = parts[0]
			}
			if len(parts) > 1 {
				last = parts[len(parts)-1]
			}
		}
	}
	if first == "" {
		first = pick(c.Rand, c.Locale.FirstNames)
	}
	if last == "" {
		last = pick(c.Rand, c.Locale.LastNames)
	}
	dom := pick(c.Rand, c.Locale.EmailDomain)
	return strings.ToLower(fmt.Sprintf("%s.%s%d@%s", first, last, c.Rand.IntRange(1, 99), dom))
}

func phone(c Ctx) any {
	return fmt.Sprintf("%s%s%s", c.Locale.CountryCode, c.Place.PhonePrefix, c.Rand.Digits(7))
}

func intProvider(c Ctx) any {
	min := paramInt(c.Params, "min", 0)
	max := paramInt(c.Params, "max", 1000)
	return c.Rand.IntRange(min, max)
}

func floatProvider(c Ctx) any {
	min := paramInt(c.Params, "min", 0)
	max := paramInt(c.Params, "max", 1000)
	return float64(min) + c.Rand.Float64()*float64(max-min)
}

// anchorTime is a fixed reference so time generation stays deterministic
// (a real time.Now() would break same-seed reproducibility).
var anchorTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func timeProvider(c Ctx) any {
	// Default: a random point in the 2 years before the anchor.
	back := c.Rand.IntRange(0, 2*365*24*3600)
	return anchorTime.Add(-time.Duration(back) * time.Second)
}

var words = strings.Fields("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor")

func lorem(c Ctx) any {
	n := c.Rand.IntRange(3, 8)
	out := make([]string, n)
	for i := range out {
		out[i] = words[c.Rand.Pick(len(words))]
	}
	return strings.Join(out, " ")
}

// card returns a Luhn-valid number from a locale BIN.
func card(c Ctx) any {
	bin := pick(c.Rand, c.Locale.CardBINs)
	body := bin + c.Rand.Digits(15-len(bin))
	return body + string(luhnCheck(body))
}

// luhnCheck returns the check digit that makes body+digit Luhn-valid.
func luhnCheck(body string) byte {
	sum, alt := 0, true
	for i := len(body) - 1; i >= 0; i-- {
		d := int(body[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return byte('0' + (10-sum%10)%10)
}

// iban returns a mod-97 checksum-valid IBAN for the locale country.
func iban(c Ctx) any {
	country := c.Locale.IBANCountry
	bbanLen := c.Locale.IBANLength - 4
	bban := c.Rand.Digits(bbanLen)
	check := ibanCheckDigits(country, bban)
	return country + check + bban
}

func ibanCheckDigits(country, bban string) string {
	// Rearrange: BBAN + country + "00", convert letters to numbers, mod 97.
	rearranged := bban + country + "00"
	var num strings.Builder
	for _, ch := range rearranged {
		if ch >= 'A' && ch <= 'Z' {
			fmt.Fprintf(&num, "%d", ch-'A'+10)
		} else {
			num.WriteRune(ch)
		}
	}
	mod := mod97(num.String())
	return fmt.Sprintf("%02d", 98-mod)
}

func mod97(s string) int {
	rem := 0
	for _, ch := range s {
		rem = (rem*10 + int(ch-'0')) % 97
	}
	return rem
}

// passport returns an AA1234567-style ID.
func passport(c Ctx) any {
	l := func() byte { return byte('A' + c.Rand.Intn(26)) }
	return fmt.Sprintf("%c%c%s", l(), l(), c.Rand.Digits(7))
}

func paramInt(p map[string]string, key string, def int) int {
	if p == nil {
		return def
	}
	if v, ok := p[key]; ok {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
