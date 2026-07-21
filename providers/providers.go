// Package providers holds atomic value generators, one per schema.Kind.
// Each is a small function of (rng, locale, params, related-values) → value.
// Adding a new type means adding one entry here, touching nothing else.
package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/bakhodir/synth/dist"
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
	// Field is the full schema field (for enum choices/weights, etc.).
	Field *schema.Field
	// Place is the record's chosen locale.Place, shared across city/region/
	// postcode/phone so they stay coherent within one record.
	Place *locale.Place
	// Gender is the record's chosen gender ("male"/"female"), shared so first
	// name, last name and the gender field agree (in gendered-surname locales
	// a male first name gets a male surname form).
	Gender string
	// Sibling returns an already-generated field value by name (for from=).
	Sibling func(name string) any
}

// Provider produces one value.
type Provider func(c Ctx) any

var registry = map[schema.Kind]Provider{}

func init() {
	registry[schema.KindUUID] = func(c Ctx) any { return uuidFrom(c.Rand) }
	registry[schema.KindFirstName] = func(c Ctx) any { return pick(c.Rand, c.Locale.FirstNamesFor(c.Gender)) }
	registry[schema.KindLastName] = func(c Ctx) any { return pick(c.Rand, c.Locale.LastNamesFor(c.Gender)) }
	registry[schema.KindName] = func(c Ctx) any {
		return pick(c.Rand, c.Locale.FirstNamesFor(c.Gender)) + " " + pick(c.Rand, c.Locale.LastNamesFor(c.Gender))
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
	registry[schema.KindEnum] = enum
	registry[schema.KindStreet] = func(c Ctx) any {
		// House number (1..9999) × native street → thousands of distinct
		// addresses per locale even from a small street list.
		return fmt.Sprintf("%d %s", c.Rand.IntRange(1, 9999), pick(c.Rand, c.Locale.Streets))
	}
	registry[schema.KindColor] = func(c Ctx) any { return pick(c.Rand, locale.Colors) }
	registry[schema.KindHexColor] = func(c Ctx) any { return fmt.Sprintf("#%06x", c.Rand.Intn(0x1000000)) }
	registry[schema.KindJob] = func(c Ctx) any { return pick(c.Rand, c.Locale.Jobs) }
	registry[schema.KindProduct] = func(c Ctx) any { return pick(c.Rand, c.Locale.Products) }
	registry[schema.KindGender] = func(c Ctx) any {
		if c.Gender != "" {
			return c.Gender // coherent with the record's name
		}
		return pick(c.Rand, locale.Genders)
	}
	registry[schema.KindMAC] = func(c Ctx) any {
		b := make([]string, 6)
		for i := range b {
			b[i] = fmt.Sprintf("%02x", c.Rand.Intn(256))
		}
		return strings.Join(b, ":")
	}
}

// enum picks from Field.Choices: uniform, weighted, or Zipf-skewed depending
// on the field's Weights and dist param.
func enum(c Ctx) any {
	if c.Field == nil || len(c.Field.Choices) == 0 {
		return ""
	}
	ch := c.Field.Choices
	if len(c.Field.Weights) == len(ch) {
		return ch[weightedPick(c.Rand, c.Field.Weights)]
	}
	if c.Params["dist"] == "zipf" {
		z := dist.NewZipf(len(ch), paramFloat(c.Params, "s", 1.07))
		return ch[z.Rank(c.Rand)]
	}
	return ch[c.Rand.Pick(len(ch))]
}

func weightedPick(r *rng.Rand, w []float64) int {
	sum := 0.0
	for _, x := range w {
		sum += x
	}
	u := r.Float64() * sum
	acc := 0.0
	for i, x := range w {
		acc += x
		if u < acc {
			return i
		}
	}
	return len(w) - 1
}

// sampleDist returns a value from the field's dist param, or ok=false if none.
func sampleDist(c Ctx) (float64, bool) {
	name := c.Params["dist"]
	if name == "" {
		return 0, false
	}
	var d dist.Dist
	switch name {
	case "normal":
		d = dist.Normal{Mu: paramFloat(c.Params, "mu", 0), Sigma: paramFloat(c.Params, "sigma", 1)}
	case "lognormal":
		d = dist.LogNormal{Mu: paramFloat(c.Params, "mu", 0), Sigma: paramFloat(c.Params, "sigma", 1)}
	case "exp", "exponential":
		d = dist.Exponential{Rate: paramFloat(c.Params, "rate", 1)}
	default:
		return 0, false
	}
	return d.Sample(c.Rand), true
}

func username(c Ctx) any {
	first := strings.ToLower(pick(c.Rand, c.Locale.FirstNamesFor(c.Gender)))
	return fmt.Sprintf("%s_%s", first, c.Rand.Digits(3))
}

func ipv4(c Ctx) any {
	// First octet from the locale's allocated blocks, so the IP plausibly
	// geolocates to the record's country.
	first := c.Rand.IntRange(1, 223)
	if len(c.Locale.IPBlocks) > 0 {
		first = c.Locale.IPBlocks[c.Rand.Pick(len(c.Locale.IPBlocks))]
	}
	return fmt.Sprintf("%d.%d.%d.%d", first, c.Rand.Intn(256), c.Rand.Intn(256), c.Rand.IntRange(1, 254))
}

func urlProvider(c Ctx) any {
	return "https://" + strings.ToLower(strings.ReplaceAll(pick(c.Rand, c.Locale.Companies), " ", "")) + ".com"
}

// amount returns a monetary value in [min,max] rounded to 2 decimals.
func amount(c Ctx) any {
	min := paramInt(c.Params, "min", 1)
	max := paramInt(c.Params, "max", 100000)
	var v float64
	if s, ok := sampleDist(c); ok {
		v = clampFloat(s, float64(min), float64(max))
	} else {
		v = float64(min) + c.Rand.Float64()*float64(max-min)
	}
	return float64(int(v*100)) / 100
}

// Get returns the provider for a kind, or nil if unknown.
func Get(k schema.Kind) Provider { return registry[k] }

// Register adds or overrides a provider for a kind. Used by the public
// synth.Register / synth.RegisterSet to support user-defined types (e.g. a
// "cinema" type drawing from movie data).
func Register(k schema.Kind, p Provider) { registry[k] = p }

// PickString returns a random element from s, exposed so user providers built
// on top of Ctx can reuse the same picking logic.
func PickString(c Ctx, s []string) string { return pick(c.Rand, s) }

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
		first = pick(c.Rand, c.Locale.FirstNamesFor(c.Gender))
	}
	if last == "" {
		last = pick(c.Rand, c.Locale.LastNamesFor(c.Gender))
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
	if v, ok := sampleDist(c); ok {
		return clampInt(int(v), min, max)
	}
	return c.Rand.IntRange(min, max)
}

func floatProvider(c Ctx) any {
	min := paramInt(c.Params, "min", 0)
	max := paramInt(c.Params, "max", 1000)
	if v, ok := sampleDist(c); ok {
		return clampFloat(v, float64(min), float64(max))
	}
	return float64(min) + c.Rand.Float64()*float64(max-min)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// anchorTime is a fixed reference so time generation stays deterministic
// (a real time.Now() would break same-seed reproducibility).
var anchorTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func timeProvider(c Ctx) any {
	// Temporal causality: if this field comes `after` another time field,
	// generate a point strictly after it, by a realistic gap. This is how a
	// record's lifecycle stays ordered (created < paid < shipped < delivered).
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		if prev, ok := c.Sibling(c.Field.From).(time.Time); ok {
			min, max := parseGap(c.Params["gap"])
			d := min + time.Duration(c.Rand.Float64()*float64(max-min))
			return prev.Add(d)
		}
	}
	// Default: a random point in the 2 years before the anchor.
	back := c.Rand.IntRange(0, 2*365*24*3600)
	return anchorTime.Add(-time.Duration(back) * time.Second)
}

// parseGap parses "1h..48h" into a duration range. Defaults to 1m..72h.
func parseGap(s string) (min, max time.Duration) {
	min, max = time.Minute, 72*time.Hour
	if s == "" {
		return
	}
	lo, hi, ok := strings.Cut(s, "..")
	if !ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d, d
		}
		return
	}
	if d, err := time.ParseDuration(lo); err == nil {
		min = d
	}
	if d, err := time.ParseDuration(hi); err == nil {
		max = d
	}
	if max < min {
		max = min
	}
	return
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

// card returns a Luhn-valid card number. Precedence:
//  1. an explicit brand param (synth:"card,brand=visa")
//  2. the locale's own BINs (e.g. HUMO/UZCARD for uz_UZ)
//  3. a random global brand (Visa, MasterCard, Amex, ...)
func card(c Ctx) any {
	if brand := c.Params["brand"]; brand != "" {
		return generateCard(c.Rand, brand)
	}
	if len(c.Locale.CardBINs) > 0 {
		bin := pick(c.Rand, c.Locale.CardBINs)
		body := bin + c.Rand.Digits(15-len(bin))
		return body + string(luhnCheck(body))
	}
	return generateCard(c.Rand, "")
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

func paramFloat(p map[string]string, key string, def float64) float64 {
	if p == nil {
		return def
	}
	if v, ok := p[key]; ok {
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil {
			return f
		}
	}
	return def
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
