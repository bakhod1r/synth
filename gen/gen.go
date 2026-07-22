// Package gen is the engine: schema.Schema + rng → records. It knows nothing
// about Go structs — it produces map[string]any keyed by field name, which the
// public API scatters back into T.
package gen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/locale"
	"github.com/bakhodir/synth/providers"
	"github.com/bakhodir/synth/schema"
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
//	mask=partial   keep the last 4 characters, star the rest
//	mask=hash      SHA-256 hex of the value
//	mask=redact    a fixed marker, no shape preserved
//	mask=token     an opaque reference, unlinkable to the value
func maskValue(f *schema.Field, v any) any {
	mode := f.Params["mask"]
	if mode == "" || v == nil {
		return v
	}
	s := fmt.Sprint(v)
	switch mode {
	case "partial":
		return partialMask(s)
	case "hash":
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])
	case "redact":
		return "[REDACTED]"
	case "token":
		sum := sha256.Sum256([]byte(s))
		return "tok_" + hex.EncodeToString(sum[:12])
	}
	return v
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
