// Package providers holds atomic value generators, one per schema.Kind.
// Each is a small function of (rng, locale, params, related-values) → value.
// Adding a new type means adding one entry here, touching nothing else.
package providers

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/bakhod1r/synth/dist"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
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
	registry[schema.KindTimeSeries] = timeSeriesProvider
	// true= is the share of true values, so a profiled column that was 90% true
	// generates that way rather than an even split.
	registry[schema.KindBool] = func(c Ctx) any {
		p := 0.5
		if v, ok := floatParam(c.Params, "true"); ok && v >= 0 && v <= 1 {
			p = v
		}
		return c.Rand.Bool(p)
	}
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
	registry[schema.KindColor] = func(c Ctx) any { return localized(c, schema.KindColor, locale.Colors) }
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
	first, last = emailSafe(first), emailSafe(last)
	// A name in a script foldASCII has no table for — Chinese, Thai, Arabic —
	// leaves nothing to build a local part from. Rather than emit an address
	// with no mailbox in front of the @, fall back to a Latin handle, which is
	// what speakers of those languages tend to register anyway.
	if first == "" {
		first = latinHandle(c.Rand)
	}
	if last == "" {
		last = latinHandle(c.Rand)
	}
	dom := pick(c.Rand, c.Locale.EmailDomain)
	return strings.ToLower(emailLocal(c.Rand, first, last) + "@" + dom)
}

// emailSafe reduces a name to what a mailbox local part may hold: ASCII letters
// and digits. Punctuation that is legal in a name but not here — apostrophes,
// hyphens, spaces, dots — is dropped, and non-ASCII letters are transliterated.
//
// The transliteration is the point. A local part is ASCII unless the whole mail
// path speaks SMTPUTF8 (RFC 6531), which most of it still does not, so
// "денис.борисов@example.com" is a fixture that the first validator it meets
// rejects. See asciifold.go.
func emailSafe(s string) string { return foldASCII(s) }

// latinHandle builds a pronounceable Latin handle for names that do not
// transliterate. It is syllables rather than random letters because an address
// is read by people as often as by code.
func latinHandle(r *rng.Rand) string {
	onsets := []string{"b", "d", "j", "k", "l", "m", "n", "r", "s", "t", "v", "z",
		"ch", "sh", "th", "kr", "pl", "st"}
	nuclei := []string{"a", "e", "i", "o", "u", "ai", "ei", "ia", "oo"}
	n := r.IntRange(2, 3)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(pick(r, onsets))
		b.WriteString(pick(r, nuclei))
	}
	return b.String()
}

// emailLocal builds the local-part in one of the shapes real mail providers
// see in the wild, so a generated column is not one repeated pattern.
func emailLocal(r *rng.Rand, first, last string) string {
	fi, li := initial(first), initial(last)
	n := r.IntRange(1, 99)
	switch r.Pick(8) {
	case 0:
		return fmt.Sprintf("%s.%s", first, last)
	case 1:
		return fmt.Sprintf("%s.%s%d", first, last, n)
	case 2:
		return fmt.Sprintf("%s%s", first, last)
	case 3:
		return fmt.Sprintf("%s_%s%d", first, last, n)
	case 4:
		return fmt.Sprintf("%s%s", fi, last)
	case 5:
		return fmt.Sprintf("%s.%s", last, first)
	case 6:
		return fmt.Sprintf("%s%s%d", first, li, n)
	default:
		return fmt.Sprintf("%s%d", first, r.IntRange(1900, 2010))
	}
}

func initial(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

func phone(c Ctx) any {
	return fmt.Sprintf("%s%s%s", c.Locale.CountryCode, c.Place.PhonePrefix, c.Rand.Digits(7))
}

func intProvider(c Ctx) any {
	min := paramInt(c.Params, "min", 0)
	max := paramInt(c.Params, "max", 1000)
	if v, ok := derived(c); ok {
		return clampInt(int(math.Round(v)), min, max)
	}
	if v, ok := sampleDist(c); ok {
		return clampInt(int(v), min, max)
	}
	return c.Rand.IntRange(min, max)
}

func floatProvider(c Ctx) any {
	min := paramInt(c.Params, "min", 0)
	max := paramInt(c.Params, "max", 1000)
	if v, ok := derived(c); ok {
		return clampFloat(v, float64(min), float64(max))
	}
	if v, ok := sampleDist(c); ok {
		return clampFloat(v, float64(min), float64(max))
	}
	return float64(min) + c.Rand.Float64()*float64(max-min)
}

// tsEpoch is the default origin for a time-series axis: t is measured from
// here, so trend and seasonality are reproducible without a cross-row pass.
var tsEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// timeSeriesProvider evaluates a numeric value that follows a curve over time:
//
//	base + trend*t_days + amplitude*sin(2π*t_sec/period) + noise
//
// t is the offset of the row's own axis timestamp from `start`, so every term
// is local to the row — no knowledge of other rows is needed, and the result
// is deterministic under the seed. This is what lets metrics and IoT series be
// generated in the same streaming, constant-memory pass as everything else.
func timeSeriesProvider(c Ctx) any {
	base := paramFloat(c.Params, "base", 0)
	axis := c.Params["axis"]
	if axis == "" || c.Sibling == nil {
		return base
	}
	ts, ok := c.Sibling(axis).(time.Time)
	if !ok {
		return base // axis blanked; Compile has already checked its type
	}
	start := tsEpoch
	if s, ok := parseInstant(c.Params["start"]); ok {
		start = s
	}
	off := ts.Sub(start)
	v := base + paramFloat(c.Params, "trend", 0)*(off.Hours()/24)
	if amp := paramFloat(c.Params, "amplitude", 0); amp != 0 {
		if period, err := time.ParseDuration(strDefault(c.Params["period"], "24h")); err == nil && period > 0 {
			v += amp * math.Sin(2*math.Pi*off.Seconds()/period.Seconds())
		}
	}
	if sd := paramFloat(c.Params, "noise", 0); sd > 0 {
		v += dist.Normal{Mu: 0, Sigma: sd}.Sample(c.Rand)
	}
	if min, ok := floatParam(c.Params, "min"); ok {
		if max, ok := floatParam(c.Params, "max"); ok {
			return clampFloat(v, min, max)
		}
	}
	return v
}

func strDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// derived computes a numeric field as a linear function of another field in the
// same row, plus gaussian noise — the correlated-numeric case (income rises
// with age). It returns ok=false when the field has no derive= param, leaving
// the field to its normal draw.
//
// The noise standard deviation scales with the value's magnitude (noise*|v|),
// so a 10% spread stays 10% whether the line sits at 20 or 20000. With noise
// omitted the relation is exact.
func derived(c Ctx) (float64, bool) {
	target := c.Params["derive"]
	if target == "" || c.Sibling == nil {
		return 0, false
	}
	x, ok := toFloat(c.Sibling(target))
	if !ok {
		// Compile checks the target is numeric; a nil here means the sibling
		// was blanked. Fall back to no derivation rather than panic.
		return 0, false
	}
	v := paramFloat(c.Params, "slope", 1)*x + paramFloat(c.Params, "intercept", 0)
	if sd := paramFloat(c.Params, "noise", 0); sd > 0 {
		v += dist.Normal{Mu: 0, Sigma: sd * math.Abs(v)}.Sample(c.Rand)
	}
	return v, true
}

// toFloat coerces a generated numeric value to float64.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

// numericKinds are the kinds whose generated value is a number, so a derive= or
// axis= target may point at them. Kept beside the providers that produce them.
var numericKinds = map[schema.Kind]bool{
	schema.KindInt: true, schema.KindFloat: true, schema.KindAge: true,
	schema.KindYear: true, schema.KindUnixTime: true, schema.KindAmount: true,
	schema.KindBalance: true, schema.KindSalary: true, schema.KindRating: true,
	schema.KindLatitude: true, schema.KindLongitude: true, schema.KindPercentage: true,
	schema.KindPort: true, schema.KindHTTPStatus: true, schema.KindTemperature: true,
}

// IsNumericKind reports whether a kind generates a number, so a correlated
// field can derive from it.
func IsNumericKind(k schema.Kind) bool { return numericKinds[k] }

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
	// An explicit window. The bounds are named min/max, matching numeric
	// fields — "from" is already taken by the dependency reference that gives
	// a timestamp its causal predecessor, and overloading it would make
	// "from=CreatedAt" and "from=2026-01-01" mean different things.
	from, hasFrom := parseInstant(c.Params["min"])
	to, hasTo := parseInstant(c.Params["max"])
	switch {
	case hasFrom && hasTo:
		if !to.After(from) {
			return from
		}
		span := to.Sub(from)
		return from.Add(time.Duration(c.Rand.Float64() * float64(span)))
	case hasFrom:
		return from.Add(time.Duration(c.Rand.Float64() * float64(2*365*24*time.Hour)))
	case hasTo:
		return to.Add(-time.Duration(c.Rand.Float64() * float64(2*365*24*time.Hour)))
	}

	// Default: a random point in the 2 years before the anchor.
	back := c.Rand.IntRange(0, 2*365*24*3600)
	return anchorTime.Add(-time.Duration(back) * time.Second)
}

// parseInstant reads a date or timestamp from a field param. Only the forms a
// person would actually type are accepted; anything else is reported as absent
// rather than silently becoming the zero time, which would put every row in
// the year 1.
func parseInstant(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
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
		// The length belongs to the scheme the BIN identifies. Amex is fifteen
		// digits; padding it to sixteen makes a number that fails validation
		// everywhere it matters.
		body := bin + c.Rand.Digits(lengthForBIN(bin)-1-len(bin))
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

// Kinds returns every kind the engine can generate, sorted. The UI and the
// docs read this rather than keeping their own list, so the catalog they show
// cannot drift from what actually works.
func Kinds() []schema.Kind {
	out := make([]schema.Kind, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Has reports whether a kind is registered.
func Has(k schema.Kind) bool {
	_, ok := registry[k]
	return ok
}
