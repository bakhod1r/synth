package providers

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// Password generation.
//
// The point of a generated password in test data is usually to exercise a
// policy: "at least one digit and one symbol", "no ambiguous characters", "at
// least 12 characters". A generator that draws freely from one alphabet
// satisfies those rules only most of the time, so a validation test built on it
// fails at random and gets muted. Every enabled character class is therefore
// guaranteed to appear at least once.

// Character classes. Ambiguous look-alikes (0/O, 1/l/I) are excluded by
// default: they are a real source of support tickets, and a fixture that
// contains them tends to be transcribed wrongly during manual testing.
const (
	lowerSafe  = "abcdefghijkmnpqrstuvwxyz"
	upperSafe  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitSafe  = "23456789"
	symbolSet  = "!@#$%^&*()-_=+[]{};:,.?"
	lowerAll   = "abcdefghijklmnopqrstuvwxyz"
	upperAll   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitAll   = "0123456789"
	defaultMin = 12
	defaultMax = 16
)

// fixtureIterations is the PBKDF2 cost used for generated password hashes.
//
// It is deliberately far below what a real credential store should use. A
// production cost of several hundred thousand iterations takes roughly a tenth
// of a second per row, which turns a million-row seed into a day of work. The
// cost is written into the hash string, so anyone looking at the value can see
// it is a fixture and not mistake it for a real credential.
const fixtureIterations = 1000

func init() {
	registry[schema.KindPassword] = password
	registry[schema.KindPassphrase] = passphrase
	registry[schema.KindPasswordHash] = passwordHashProvider
}

// password builds a password honoring the field's policy params:
//
//	length=16                 exact length
//	min=8,max=20              length range
//	upper|lower|digits|symbols=false   drop a class
//	ambiguous=true            allow 0/O/1/l/I
func password(c Ctx) any {
	return generatePassword(c.Rand, c.Params)
}

// strengths are five named policies, from a PIN to something a manager would
// store. Naming them beats making every caller assemble the same four toggles,
// and it gives the interface one control instead of six.
var strengths = map[string]map[string]string{
	"pin":         {"lower": "false", "upper": "false", "symbols": "false", "min": "4", "max": "6"},
	"weak":        {"upper": "false", "digits": "false", "symbols": "false", "min": "6", "max": "8"},
	"medium":      {"upper": "false", "symbols": "false", "min": "8", "max": "10"},
	"strong":      {"symbols": "false", "min": "12", "max": "14"},
	"very-strong": {"min": "16", "max": "20"},
}

// StrengthNames lists the policies in increasing order, for interfaces that
// offer them as a choice.
var StrengthNames = []string{"pin", "weak", "medium", "strong", "very-strong"}

// applyStrength expands a strength name into params. An explicit param always
// wins: the named policy is a starting point, not a cage.
func applyStrength(params map[string]string) map[string]string {
	preset, ok := strengths[strings.ToLower(params["strength"])]
	if !ok {
		return params
	}
	merged := make(map[string]string, len(preset)+len(params))
	for k, v := range preset {
		merged[k] = v
	}
	for k, v := range params {
		if k != "strength" {
			merged[k] = v
		}
	}
	return merged
}

func generatePassword(r *rng.Rand, params map[string]string) string {
	params = applyStrength(params)
	ambiguous := boolParam(params, "ambiguous", false)

	type class struct {
		name  string
		chars string
	}
	var classes []class
	if boolParam(params, "lower", true) {
		classes = append(classes, class{"lower", pickSet(ambiguous, lowerAll, lowerSafe)})
	}
	if boolParam(params, "upper", true) {
		classes = append(classes, class{"upper", pickSet(ambiguous, upperAll, upperSafe)})
	}
	if boolParam(params, "digits", true) {
		classes = append(classes, class{"digits", pickSet(ambiguous, digitAll, digitSafe)})
	}
	if boolParam(params, "symbols", true) {
		classes = append(classes, class{"symbols", symbolSet})
	}
	// Turning every class off leaves nothing to draw from; lowercase is the
	// least surprising fallback.
	if len(classes) == 0 {
		classes = append(classes, class{"lower", lowerSafe})
	}

	length := passwordLength(r, params, len(classes))

	// One character from each class first, so the result always satisfies a
	// policy that requires each of them.
	out := make([]byte, 0, length)
	for _, cl := range classes {
		out = append(out, cl.chars[r.Pick(len(cl.chars))])
	}
	var all strings.Builder
	for _, cl := range classes {
		all.WriteString(cl.chars)
	}
	pool := all.String()
	for len(out) < length {
		out = append(out, pool[r.Pick(len(pool))])
	}

	// Shuffle, or the first characters would always follow class order.
	for i := len(out) - 1; i > 0; i-- {
		j := r.Pick(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// passwordLength resolves the requested length, never returning fewer
// characters than there are required classes.
func passwordLength(r *rng.Rand, params map[string]string, classes int) int {
	if n, ok := intParam(params, "length"); ok {
		return max(n, classes)
	}
	lo, okLo := intParam(params, "min")
	hi, okHi := intParam(params, "max")
	if !okLo {
		lo = defaultMin
	}
	if !okHi {
		hi = defaultMax
	}
	if hi < lo {
		hi = lo
	}
	return max(r.IntRange(lo, hi), classes)
}

func pickSet(ambiguous bool, all, safe string) string {
	if ambiguous {
		return all
	}
	return safe
}

// passphrase builds a memorable password from the locale's own words, so an
// Uzbek fixture reads as Uzbek rather than as English words in a Cyrillic
// table. Params: words=4, sep=-, capitalize=false, number=false.
func passphrase(c Ctx) any {
	n, ok := intParam(c.Params, "words")
	if !ok || n < 2 {
		n = 4
	}
	sep := c.Params["sep"]
	if sep == "" {
		sep = "-"
	}
	words := make([]string, n)
	for i := range words {
		w := localized(c, schema.KindPassphrase, passphraseWords)
		if boolParam(c.Params, "capitalize", false) {
			w = strings.ToUpper(w[:1]) + w[1:]
		}
		words[i] = w
	}
	out := strings.Join(words, sep)
	if boolParam(c.Params, "number", false) {
		out += sep + c.Rand.Digits(2)
	}
	return out
}

// passwordHashProvider emits a hash of a freshly generated password, in the
// self-describing form a password column actually stores. Seeding a users table
// needs this: a plaintext column is wrong, and a bare digest cannot be verified
// later because the salt and cost are missing.
func passwordHashProvider(c Ctx) any {
	plain := generatePassword(c.Rand, c.Params)
	salt := c.Rand.Digits(16)
	iter := fixtureIterations
	if n, ok := intParam(c.Params, "iterations"); ok && n > 0 {
		iter = n
	}
	key, err := pbkdf2.Key(sha256.New, plain, []byte(salt), iter, 32)
	if err != nil {
		// The parameters here are fixed and valid, so this cannot fail in
		// practice; returning the error text beats panicking in a generator.
		return "pbkdf2-error:" + err.Error()
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter,
		base64.RawStdEncoding.EncodeToString([]byte(salt)),
		base64.RawStdEncoding.EncodeToString(key))
}

func boolParam(params map[string]string, key string, fallback bool) bool {
	v, ok := params[key]
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func intParam(params map[string]string, key string) (int, bool) {
	v, ok := params[key]
	if !ok || v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// passphraseWords is the English fallback: short, common, unambiguous words.
var passphraseWords = []string{
	"apple", "anchor", "arrow", "basket", "bridge", "butter", "candle", "carpet",
	"cherry", "cloud", "copper", "cotton", "crystal", "desert", "dragon", "eagle",
	"ember", "falcon", "forest", "garden", "ginger", "granite", "harbor", "hazel",
	"honey", "island", "jungle", "kettle", "ladder", "lantern", "lemon", "maple",
	"marble", "meadow", "mirror", "monkey", "mountain", "nectar", "needle", "ocean",
	"orange", "orchid", "otter", "pepper", "pigeon", "pillow", "planet", "pocket",
	"puzzle", "rabbit", "rainbow", "raven", "ribbon", "river", "rocket", "saddle",
	"salmon", "shadow", "silver", "sparrow", "spider", "spring", "squirrel", "stable",
	"summer", "sunset", "temple", "thunder", "tiger", "timber", "tomato", "tunnel",
	"turtle", "valley", "velvet", "village", "walnut", "willow", "window", "winter",
}
