// Package gen is the engine: schema.Schema + rng → records. It knows nothing
// about Go structs — it produces map[string]any keyed by field name, which the
// public API scatters back into T.
package gen

import (
	"fmt"

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
	return e.generate(r, &place)
}

// generate fills one record using the given rng and locale place (shared so
// nested objects stay locale-coherent with their parent).
func (e *Engine) generate(r *rng.Rand, place *locale.Place) map[string]any {
	values := make(map[string]any, len(e.schema.Fields))
	for _, i := range e.order {
		f := &e.schema.Fields[i]
		v := e.field(r, f, place, values)
		if e.Chaos > 0 && f.FromRef == nil && r.Bool(e.Chaos) {
			v = chaosValue(r, v)
		}
		// Unique enforcement: resample until a fresh value is found. Uses a
		// generous cap; on exhaustion it keeps the last value (callers should
		// ensure the field's space exceeds the row count).
		if seen := e.seen[f.Name]; seen != nil {
			for attempt := 0; seen[v] && attempt < 1000; attempt++ {
				v = e.field(r, f, place, values)
			}
			seen[v] = true
		}
		values[f.Name] = v
	}
	return values
}

func (e *Engine) field(r *rng.Rand, f *schema.Field, place *locale.Place, values map[string]any) any {
	// Foreign key: draw from the referenced parent's PK values.
	if f.FromRef != nil {
		return f.FromRef[r.Pick(len(f.FromRef))]
	}
	if f.Kind == schema.KindUnknown {
		return nil
	}
	// Nested object: generate a sub-record with the same rng and place.
	if f.Kind == schema.KindObject {
		if sub := e.sub[f.Nested]; sub != nil {
			return sub.generate(r, place)
		}
		return nil
	}
	// Array: generate a slice of elements (scalars or nested objects).
	if f.Kind == schema.KindArray {
		return e.array(r, f, place, values)
	}
	p := providers.Get(f.Kind)
	c := providers.Ctx{
		Rand:   r,
		Locale: e.loc,
		Params: f.Params,
		Field:  f,
		Place:  place,
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
	return p(c)
}

// array generates a slice value for a KindArray field.
func (e *Engine) array(r *rng.Rand, f *schema.Field, place *locale.Place, values map[string]any) []any {
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
				out[i] = sub.generate(r, place)
				continue
			}
			out[i] = nil
			continue
		}
		out[i] = e.field(r, f.Elem, place, values)
	}
	return out
}

// Schema exposes the underlying schema (for encoders needing column order).
func (e *Engine) Schema() *schema.Schema { return e.schema }

// HasUnique reports whether any field requires unique values. Unique tracking
// is stateful, so callers must use serial generation when this is true.
func (e *Engine) HasUnique() bool { return e.seen != nil }
