// Package gen is the engine: schema.Schema + rng → records. It knows nothing
// about Go structs — it produces map[string]any keyed by field name, which the
// public API scatters back into T.
package gen

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// Engine holds a compiled, validated schema ready to generate records.
type Engine struct {
	schema *schema.Schema
	loc    *locale.Locale
	order  []int // field indices in dependency order
	// Chaos is the probability [0,1] that a string/numeric field carries an
	// edge-case value instead of a normal one (see WithChaos).
	Chaos float64
	// seen tracks generated values per unique field, to enforce distinctness.
	seen map[string]map[any]bool
	// sub holds compiled engines for nested object schemas, keyed by the
	// nested *schema.Schema pointer.
	sub map[*schema.Schema]*Engine
}

// Compile validates the schema (unknown kinds, from= cycles) and computes the
// field generation order once. Returns an error for configuration problems.
func Compile(s *schema.Schema, localeName string) (*Engine, error) {
	order, err := topoOrder(s)
	if err != nil {
		return nil, err
	}
	for _, f := range s.Fields {
		if f.Kind == schema.KindUnknown {
			continue // handled as zero value + warning upstream
		}
		if f.FromRef != nil {
			continue // filled from a parent slice, no provider needed
		}
		if f.Kind == schema.KindObject || f.Kind == schema.KindArray {
			continue // handled by recursive sub-engines, not a provider
		}
		if providers.Get(f.Kind) == nil {
			return nil, fmt.Errorf("synth: field %q has unknown kind %q", f.Name, f.Kind)
		}
	}
	e := &Engine{schema: s, loc: locale.Get(localeName), order: order}
	// Compile nested object schemas recursively (sharing the same locale).
	for i := range s.Fields {
		f := &s.Fields[i]
		var nested *schema.Schema
		if f.Kind == schema.KindObject {
			nested = f.Nested
		} else if f.Kind == schema.KindArray && f.Elem != nil && f.Elem.Kind == schema.KindObject {
			nested = f.Elem.Nested
		}
		if nested != nil {
			sub, err := Compile(nested, localeName)
			if err != nil {
				return nil, err
			}
			if e.sub == nil {
				e.sub = map[*schema.Schema]*Engine{}
			}
			e.sub[nested] = sub
		}
	}
	for _, f := range s.Fields {
		// UUIDs are unique by construction — no stateful tracking needed, which
		// also keeps them safe for parallel generation.
		if f.Unique && f.Kind != schema.KindUUID {
			if e.seen == nil {
				e.seen = map[string]map[any]bool{}
			}
			e.seen[f.Name] = map[any]bool{}
		}
	}
	return e, nil
}

// topoOrder returns field indices such that every from=/match= dependency is
// generated before its dependent. Errors on cycles.
func topoOrder(s *schema.Schema) ([]int, error) {
	idx := map[string]int{}
	for i, f := range s.Fields {
		idx[f.Name] = i
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make([]int, len(s.Fields))
	var order []int
	var visit func(i int) error
	visit = func(i int) error {
		switch color[i] {
		case gray:
			return fmt.Errorf("synth: dependency cycle involving field %q", s.Fields[i].Name)
		case black:
			return nil
		}
		color[i] = gray
		for _, dep := range []string{s.Fields[i].From, s.Fields[i].Match} {
			if dep == "" {
				continue
			}
			j, ok := idx[dep]
			if !ok {
				return fmt.Errorf("synth: field %q references unknown field %q", s.Fields[i].Name, dep)
			}
			if err := visit(j); err != nil {
				return err
			}
		}
		color[i] = black
		order = append(order, i)
		return nil
	}
	for i := range s.Fields {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// Record generates one record. seq is the record index, used with the base
// rng to fork a deterministic per-record stream.
func (e *Engine) Record(base *rng.Rand, seq int) map[string]any {
	r := base.Fork(uint64(seq))
	place := e.loc.Places[r.Pick(len(e.loc.Places))]
	gender := "male"
	if r.Bool(0.5) {
		gender = "female"
	}
	return e.generate(r, &place, gender)
}

// generate fills one record using the given rng, locale place and gender
// (shared so nested objects and name/gender fields stay coherent).
func (e *Engine) generate(r *rng.Rand, place *locale.Place, gender string) map[string]any {
	values := make(map[string]any, len(e.schema.Fields))
	for _, i := range e.order {
		f := &e.schema.Fields[i]
		v := e.field(r, f, place, gender, values)
		if e.Chaos > 0 && f.FromRef == nil && r.Bool(e.Chaos) {
			v = chaosValue(r, v)
		}
		// Unique enforcement: resample until a fresh value is found. Uses a
		// generous cap; on exhaustion it keeps the last value (callers should
		// ensure the field's space exceeds the row count).
		if seen := e.seen[f.Name]; seen != nil {
			for attempt := 0; seen[v] && attempt < 1000; attempt++ {
				v = e.field(r, f, place, gender, values)
			}
			seen[v] = true
		}
		values[f.Name] = v
	}
	return values
}

func (e *Engine) field(r *rng.Rand, f *schema.Field, place *locale.Place, gender string, values map[string]any) any {
	// A blank share applies to every kind, so it is handled here rather than
	// asked of each provider. Real tables have missing values, and code that
	// has only ever seen complete rows breaks the first time it meets a null.
	//
	// The draw happens before anything else so the cost of a blanked field is
	// nothing, and a primary key is never blanked: a row without an identity
	// is not missing data, it is a broken row.
	if blank := blankShare(f); blank > 0 && !f.PK && !f.Unique && r.Bool(blank) {
		return nil
	}
	// Foreign key: draw from the referenced parent's PK values.
	if f.FromRef != nil {
		return f.FromRef[r.Pick(len(f.FromRef))]
	}
	if f.Kind == schema.KindUnknown {
		return nil
	}
	// Nested object: generate a sub-record with the same rng, place and gender.
	if f.Kind == schema.KindObject {
		if sub := e.sub[f.Nested]; sub != nil {
			return sub.generate(r, place, gender)
		}
		return nil
	}
	// Array: generate a slice of elements (scalars or nested objects).
	if f.Kind == schema.KindArray {
		return e.array(r, f, place, gender, values)
	}
	p := providers.Get(f.Kind)
	c := providers.Ctx{
		Rand:   r,
		Locale: e.loc,
		Params: f.Params,
		Field:  f,
		Place:  place,
		Gender: gender,
		Sibling: func(name string) any {
			if name == "__from__" {
				if f.From != "" {
					return values[f.From]
				}
				return nil
			}
			return values[name]
		},
	}
	return maskValue(f, p(c))
}

// maskValue applies the field's mask= setting to a generated value.
//
// This is what makes a fixture safe to paste into a ticket or a screenshot: a
// card number or a national identifier is realistic in shape but reveals
// nothing, even by accident. It runs on the generated value rather than on real
// data — for masking an actual export, see the mask package.
//
//	mask=partial   keep the last 4 characters, star the rest — on a card
//	               number, the first 4 as well (the BIN, which identifies the
//	               issuer and not the cardholder)
//	mask=hash      SHA-256 hex of the value
//	mask=redact    a fixed marker, no shape preserved
//	mask=token     an opaque reference, unlinkable to the value
//
// hash and token take three further params:
//
//	salt=…         mixed in before hashing, so two datasets built from the same
//	               values do not share digests
//	secret=…       switches to HMAC-SHA256 with this key. Without the key the
//	               digest cannot be recomputed, which is what stops an attacker
//	               from confirming a guessed value by hashing it.
//	digest=16      truncate the hex to this many characters
func maskValue(f *schema.Field, v any) any {
	mode := f.Params["mask"]
	if mode == "" || v == nil {
		return v
	}
	s := fmt.Sprint(v)
	switch mode {
	case "partial":
		if f.Kind == schema.KindCard {
			return cardMask(s)
		}
		return partialMask(s)
	case "hash":
		return truncate(f, digest(f, s))
	case "redact":
		return "[REDACTED]"
	case "token":
		// A token is a reference, not a fingerprint: 24 hex characters is
		// already far more than a table will ever need to stay collision-free,
		// and a shorter value is easier to read in a log.
		d := truncate(f, digest(f, s))
		if len(d) > 24 {
			d = d[:24]
		}
		return "tok_" + d
	}
	return v
}

// scoped binds a masked value to its column before hashing.
//
// Without this, the same national identifier masked in two different columns
// produces the same digest, so anyone holding both tables can join on the
// masked value and re-link the rows the mask was meant to separate. It also
// makes a single rainbow table work against every column at once.
//
// Mixing the column name in keeps the property that matters — the same value in
// the same column always masks the same way, so the column is still joinable —
// while making cross-column correlation useless. The separator cannot occur in
// a column name, so "ab"+"c" and "a"+"bc" cannot collide.
func scoped(f *schema.Field, s string) []byte {
	return []byte(f.Name + "\x00" + f.Params["salt"] + "\x00" + s)
}

// digest hashes a value for the hash and token masks.
//
// With secret= it is an HMAC rather than a bare hash. That difference matters
// whenever the set of possible values is small: a national identifier or a
// phone number can simply be enumerated and hashed until the digest matches, so
// a plain hash of one hides nothing. A key the attacker does not have removes
// that attack entirely.
func digest(f *schema.Field, s string) string {
	if key := f.Params["secret"]; key != "" {
		m := hmac.New(sha256.New, []byte(key))
		m.Write(scoped(f, s))
		return hex.EncodeToString(m.Sum(nil))
	}
	sum := sha256.Sum256(scoped(f, s))
	return hex.EncodeToString(sum[:])
}

// truncate shortens a digest to digest=N characters.
//
// Shortening raises the chance that two different values collide, which in a
// masked column means two different people silently becoming one row. Sixteen
// hex characters is the floor: below that, collisions stop being theoretical
// in a table of any size.
func truncate(f *schema.Field, s string) string {
	n, err := strconv.Atoi(f.Params["digest"])
	if err != nil || n <= 0 || n >= len(s) {
		return s
	}
	if n < 16 {
		n = 16
	}
	return s[:n]
}

// partialMask keeps the last four characters. Fewer than four would reveal
// most of a short value, so a short value is starred completely rather than
// half-shown.
func partialMask(s string) string {
	const keep = 4
	r := []rune(s)
	if len(r) <= keep {
		return strings.Repeat("*", len(r))
	}
	var b strings.Builder
	for i, c := range r {
		switch {
		case i >= len(r)-keep:
			b.WriteRune(c)
		case c == '-' || c == ' ' || c == '/':
			b.WriteRune(c) // keep the shape readable
		default:
			b.WriteRune('*')
		}
	}
	return b.String()
}

// cardMask keeps the leading four digits and the trailing four. The first four
// are the start of the BIN: they identify the issuer, not the cardholder, and
// keeping them lets a fixture stay recognizably a Visa or a HUMO card while the
// account number stays hidden. PCI DSS permits up to the first six and the last
// four; four is the stricter choice.
//
// Below twelve digits the two windows would meet and reveal nearly everything,
// so a short value falls back to the trailing-four rule.
func cardMask(s string) string {
	const lead, tail = 4, 4
	r := []rune(s)
	digits := 0
	for _, c := range r {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	if digits < 12 {
		return partialMask(s)
	}
	var b strings.Builder
	seen := 0
	for _, c := range r {
		if c < '0' || c > '9' {
			b.WriteRune(c) // keep the shape readable
			continue
		}
		seen++
		if seen <= lead || seen > digits-tail {
			b.WriteRune(c)
		} else {
			b.WriteRune('*')
		}
	}
	return b.String()
}

// array generates a slice value for a KindArray field.
func (e *Engine) array(r *rng.Rand, f *schema.Field, place *locale.Place, gender string, values map[string]any) []any {
	min, max := f.ArrMin, f.ArrMax
	if max < min {
		max = min
	}
	if max == 0 {
		min, max = 1, 3
	}
	n := r.IntRange(min, max)
	out := make([]any, n)
	for i := 0; i < n; i++ {
		if f.Elem.Kind == schema.KindObject {
			if sub := e.sub[f.Elem.Nested]; sub != nil {
				out[i] = sub.generate(r, place, gender)
				continue
			}
			out[i] = nil
			continue
		}
		out[i] = e.field(r, f.Elem, place, gender, values)
	}
	return out
}

// Schema exposes the underlying schema (for encoders needing column order).
func (e *Engine) Schema() *schema.Schema { return e.schema }

// HasUnique reports whether any field requires unique values. Unique tracking
// is stateful, so callers must use serial generation when this is true.
func (e *Engine) HasUnique() bool { return e.seen != nil }

// blankShare reads the field's blank probability. It accepts a fraction
// ("0.15") or a percentage ("15%"), because both readings of "blank: 15" are
// natural and guessing wrong by a factor of a hundred is an unpleasant
// surprise: a bare number is read as a percentage, matching the label users
// see in the interface.
func blankShare(f *schema.Field) float64 {
	raw, ok := f.Params["blank"]
	if !ok || raw == "" {
		return 0
	}
	raw = strings.TrimSpace(raw)
	percent := strings.HasSuffix(raw, "%")
	raw = strings.TrimSuffix(raw, "%")

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0
	}
	// "0.15" means fifteen percent; "15" and "15%" mean the same thing.
	if percent || v > 1 {
		v /= 100
	}
	if v > 1 {
		return 1
	}
	return v
}
