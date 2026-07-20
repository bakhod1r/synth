package synth

import (
	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/providers"
	"github.com/bakhodir/synth/schema"
)

// R is the minimal randomness surface handed to a custom provider. It keeps
// user code decoupled from Synth internals while staying deterministic (it is
// the record's own seeded stream).
type R interface {
	// Intn returns a value in [0,n).
	Intn(n int) int
	// IntRange returns a value in [min,max].
	IntRange(min, max int) int
	// Float64 returns a value in [0,1).
	Float64() float64
	// Pick returns a random element of s (empty string if s is empty).
	Pick(s []string) string
	// Digits returns n random decimal digits.
	Digits(n int) string
}

type rAdapter struct{ c providers.Ctx }

func (a rAdapter) Intn(n int) int            { return a.c.Rand.Intn(n) }
func (a rAdapter) IntRange(min, max int) int { return a.c.Rand.IntRange(min, max) }
func (a rAdapter) Float64() float64          { return a.c.Rand.Float64() }
func (a rAdapter) Pick(s []string) string    { return providers.PickString(a.c, s) }
func (a rAdapter) Digits(n int) string       { return a.c.Rand.Digits(n) }

// Register adds a custom field type. After this, a field can select it by tag
// (`synth:"cinema"`) or by a matching field name, and its value is produced by
// fn. fn must be deterministic given R for reproducible output.
//
//	synth.Register("cinema", func(r synth.R) any {
//	    return r.Pick([]string{"Inception", "Interstellar", "Tenet"})
//	})
func Register(name string, fn func(r R) any) {
	k := schema.Kind(name)
	providers.Register(k, func(c providers.Ctx) any {
		return fn(rAdapter{c: c})
	})
	infer.Alias(name, k)
}

// RegisterSet is the common case: a custom type that picks uniformly from a
// fixed set of values (e.g. movie titles for a "cinema" type).
//
//	synth.RegisterSet("cinema", "Inception", "Interstellar", "Tenet", "Dune")
func RegisterSet(name string, values ...string) {
	Register(name, func(r R) any { return r.Pick(values) })
}
