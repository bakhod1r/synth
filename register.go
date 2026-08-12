package synth

import (
	"github.com/bakhod1r/synth/infer"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
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

// Env is the wider surface a custom provider may reach for when randomness
// alone is not enough: the record's locale, and the fields already generated
// for it. The R handed to a provider always satisfies it, so a provider that
// needs more than R asserts for it:
//
//	synth.Register("device_brand", func(r synth.R) any {
//	    env, ok := r.(synth.Env)
//	    ...
//	})
//
// It is a separate interface rather than more methods on R so that adding to
// it later does not break providers written against R.
type Env interface {
	R
	// From returns the value of the field named by this field's from=, or nil
	// when it has none. It is how a provider stays coherent with a sibling:
	// a device name reads the model code it must match.
	From() any
	// Sibling returns an already-generated field of the same record by name,
	// or nil when that field does not exist or has not been generated yet.
	// Field order follows the from=/match=/derive= dependency graph, so a
	// provider that needs a sibling should be declared with from=.
	Sibling(name string) any
	// LocaleName is the record's locale, e.g. "uz_UZ".
	LocaleName() string
	// CountryCode is the record locale's international dialling prefix with
	// its leading '+', e.g. "+998".
	CountryCode() string
	// PhonePrefix is the operator or area digits of the place this record was
	// given, e.g. "90" for Tashkent or "213" for Los Angeles. It is what keeps
	// a generated number in the same city as the record's address.
	PhonePrefix() string
}

type rAdapter struct{ c providers.Ctx }

func (a rAdapter) Intn(n int) int            { return a.c.Rand.Intn(n) }
func (a rAdapter) IntRange(min, max int) int { return a.c.Rand.IntRange(min, max) }
func (a rAdapter) Float64() float64          { return a.c.Rand.Float64() }
func (a rAdapter) Pick(s []string) string    { return providers.PickString(a.c, s) }
func (a rAdapter) Digits(n int) string       { return a.c.Rand.Digits(n) }

func (a rAdapter) From() any { return a.Sibling("__from__") }

func (a rAdapter) Sibling(name string) any {
	if a.c.Sibling == nil {
		return nil
	}
	return a.c.Sibling(name)
}

func (a rAdapter) LocaleName() string {
	if a.c.Locale == nil {
		return ""
	}
	return a.c.Locale.Name
}

func (a rAdapter) CountryCode() string {
	if a.c.Locale == nil {
		return ""
	}
	return a.c.Locale.CountryCode
}

func (a rAdapter) PhonePrefix() string {
	if a.c.Place == nil {
		return ""
	}
	return a.c.Place.PhonePrefix
}

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
