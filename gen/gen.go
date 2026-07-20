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
		if providers.Get(f.Kind) == nil {
			return nil, fmt.Errorf("synth: field %q has unknown kind %q", f.Name, f.Kind)
		}
	}
	return &Engine{schema: s, loc: locale.Get(localeName), order: order}, nil
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
	values := make(map[string]any, len(e.schema.Fields))
	for _, i := range e.order {
		f := &e.schema.Fields[i]
		values[f.Name] = e.field(r, f, &place, values)
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

// Schema exposes the underlying schema (for encoders needing column order).
func (e *Engine) Schema() *schema.Schema { return e.schema }
