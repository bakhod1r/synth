// Package mask anonymizes a real data export: it replaces personal data with
// synthetic values while preserving the FORMAT and the referential structure of
// the original, so the result still exercises the same code paths.
//
// This is the GDPR path — take a production dump, hand back something safe to
// share with developers. Synth reads and writes files only; it never connects
// to a database, so the dump you feed it is one you exported yourself.
//
// Guarantees:
//   - Consistency: the same input value always maps to the same replacement
//     (within one run), so joins and foreign keys still line up.
//   - Format preservation: an email stays an email, a card stays Luhn-valid at
//     the same length, digits stay digits, letter case is kept.
//   - Irreversibility: replacements are drawn from a keyed hash of the input,
//     not an encoding of it, so the original cannot be recovered from output.
package mask

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// Strategy decides how a column's values are replaced.
type Strategy string

const (
	// Keep leaves the column untouched (non-personal data).
	Keep Strategy = "keep"
	// Fake replaces values with synthetic ones of the same kind, consistently.
	Fake Strategy = "fake"
	// Redact replaces every character with a fixed mask, keeping the shape.
	Redact Strategy = "redact"
	// Drop blanks the column entirely.
	Drop Strategy = "drop"
	// DP adds Laplace noise to a numeric value, calibrated to an epsilon
	// budget — the Laplace mechanism of differential privacy. It bounds how
	// much any single record shows through the released number.
	DP Strategy = "dp"
)

// Rule binds a column to a strategy (and, for Fake, to a kind).
type Rule struct {
	Column   string
	Strategy Strategy
	// Kind is what Fake should generate. When empty it is inferred from the
	// column name and the observed value's format.
	Kind schema.Kind
	// Epsilon is the per-column privacy budget for the DP strategy: smaller
	// means more noise. Sensitivity is how much one record can move the value
	// (for a released numeric column, its plausible range). Both are required
	// for DP; the noise scale is Sensitivity/Epsilon.
	Epsilon, Sensitivity float64
}

// Masker rewrites values according to its rules.
type Masker struct {
	key   []byte
	loc   *locale.Locale
	rules map[string]Rule
	// cache keeps value→replacement stable within a run so joins survive.
	cache map[string]string
	// err holds the first fatal masking error (a DP rule on a non-numeric
	// value). Value cannot return an error per cell, so it records here and the
	// file layer surfaces it.
	err error
}

// New returns a Masker. The key makes replacements deterministic across runs
// for the same input — use the same key when masking related dumps so foreign
// keys still match; use a fresh key to make runs unlinkable.
func New(key string, localeName string) *Masker {
	return &Masker{
		key:   []byte(key),
		loc:   locale.Get(localeName),
		rules: map[string]Rule{},
		cache: map[string]string{},
	}
}

// Rule registers how a column is handled.
func (m *Masker) Rule(r Rule) { m.rules[strings.ToLower(r.Column)] = r }

// hasDP reports whether a column carries a differential-privacy rule, so the
// JSONL path knows to route its numeric values through the masker.
func (m *Masker) hasDP(column string) bool {
	return m.rules[strings.ToLower(column)].Strategy == DP
}

// Value masks one cell of a column.
func (m *Masker) Value(column, value string) string {
	if value == "" {
		return value
	}
	r, ok := m.rules[strings.ToLower(column)]
	if !ok {
		// Unlisted columns are faked when they look personal, kept otherwise —
		// safe by default: it is better to over-mask than to leak.
		if k, personal := personalKind(column, value); personal {
			r = Rule{Column: column, Strategy: Fake, Kind: k}
		} else {
			// Free-text columns can still carry PII inside them ("contact
			// bob@corp.io"), so scrub embedded identifiers before keeping.
			return m.scrubEmbedded(column, value)
		}
	}
	switch r.Strategy {
	case Keep:
		return value
	case Drop:
		return ""
	case Redact:
		return redact(value)
	case Fake:
		return m.fake(column, value, r.Kind)
	case DP:
		return m.dpNoise(column, value, r)
	}
	return value
}

// dpNoise adds Laplace noise to a numeric value, calibrated to the rule's
// epsilon budget — the Laplace mechanism. The noise is drawn from an
// HMAC-of-the-value seed, so a masked file reproduces under the key: this is
// input perturbation for testable fixtures, not query-time DP, and the
// reproducibility is a deliberate trade (equal inputs get equal noise).
//
// A non-numeric value cannot be noised; the first one records an error the file
// layer returns, rather than silently passing the raw value through.
func (m *Masker) dpNoise(column, value string, r Rule) string {
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		if m.err == nil {
			m.err = fmt.Errorf("mask: column %q has a DP rule but value %q is not numeric",
				column, value)
		}
		return value
	}
	eps := r.Epsilon
	if eps <= 0 {
		eps = 1
	}
	scale := r.Sensitivity / eps
	seed := binary.BigEndian.Uint64(m.mac(column + "\x00" + value)[:8])
	noised := v + laplace(rng.New(seed), scale)
	return strconv.FormatFloat(noised, 'f', -1, 64)
}

// laplace draws from a zero-mean Laplace distribution with the given scale,
// using inverse-transform sampling: with u uniform on (-0.5, 0.5),
// x = -scale*sign(u)*ln(1-2|u|).
func laplace(r *rng.Rand, scale float64) float64 {
	if scale <= 0 {
		return 0
	}
	u := r.Float64() - 0.5
	sign := 1.0
	if u < 0 {
		sign = -1
	}
	return -scale * sign * math.Log(1-2*math.Abs(u))
}

// fake produces a stable synthetic replacement for value.
func (m *Masker) fake(column, value string, kind schema.Kind) string {
	ck := column + "\x00" + value
	if v, ok := m.cache[ck]; ok {
		return v
	}
	if kind == "" {
		kind, _ = personalKind(column, value)
	}
	// Seed a per-value RNG from an HMAC of the input: same input → same output,
	// but the output carries no recoverable trace of the input.
	seed := binary.BigEndian.Uint64(m.mac(ck)[:8])
	r := rng.New(seed)
	place := m.loc.Places[r.Pick(len(m.loc.Places))]
	gender := "male"
	if r.Bool(0.5) {
		gender = "female"
	}

	var out string
	if p := providers.Get(kind); p != nil && kind != "" {
		out = fmt.Sprint(p(providers.Ctx{
			Rand: r, Locale: m.loc, Params: map[string]string{},
			Field: &schema.Field{Name: column}, Place: &place, Gender: gender,
		}))
	} else {
		out = shapePreserving(value, r)
	}
	m.cache[ck] = out
	return out
}

func (m *Masker) mac(s string) []byte {
	h := hmac.New(sha256.New, m.key)
	h.Write([]byte(s))
	return h.Sum(nil)
}

// shapePreserving replaces characters class-by-class: digits stay digits,
// letters stay letters with the same case, everything else is untouched.
// Used when no specific kind applies, so lengths and separators survive.
func shapePreserving(v string, r *rng.Rand) string {
	var b strings.Builder
	for _, ch := range v {
		switch {
		case unicode.IsDigit(ch):
			b.WriteRune(rune('0' + r.Intn(10)))
		case unicode.IsUpper(ch):
			b.WriteRune(rune('A' + r.Intn(26)))
		case unicode.IsLower(ch):
			b.WriteRune(rune('a' + r.Intn(26)))
		default:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// redact keeps the shape but reveals nothing: letters and digits become '*'.
func redact(v string) string {
	var b strings.Builder
	for _, ch := range v {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			b.WriteRune('*')
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
