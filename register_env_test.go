package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
)

// A custom provider reaches past R for the record's locale and its already
// generated fields. Both are what an out-of-tree provider (ext/phonex,
// ext/devicex) needs to stay coherent with the rest of the record.
func TestEnvExposesLocaleAndSiblings(t *testing.T) {
	synth.Register("env_locale", func(r synth.R) any {
		env, ok := r.(synth.Env)
		if !ok {
			t.Fatal("R handed to a provider does not satisfy Env")
		}
		return env.LocaleName() + " " + env.CountryCode()
	})
	synth.Register("env_echo", func(r synth.R) any {
		env := r.(synth.Env)
		s, _ := env.From().(string)
		return "<" + s + ">"
	})
	synth.Register("env_by_name", func(r synth.R) any {
		env := r.(synth.Env)
		s, _ := env.Sibling("Src").(string)
		return "[" + s + "]"
	})

	type row struct {
		Where string `synth:"env_locale"`
		Src   string `synth:"env_locale"`
		Echo  string `synth:"env_echo,from=Src"`
		ByRef string `synth:"env_by_name,from=Src"`
	}
	for _, r := range synth.Make[row](5, synth.WithSeed(1), synth.WithLocale("uz_UZ")) {
		if r.Where != "uz_UZ +998" {
			t.Fatalf("locale = %q, want %q", r.Where, "uz_UZ +998")
		}
		if want := "<" + r.Src + ">"; r.Echo != want {
			t.Errorf("From() = %q, want %q", r.Echo, want)
		}
		if want := "[" + r.Src + "]"; r.ByRef != want {
			t.Errorf("Sibling() = %q, want %q", r.ByRef, want)
		}
	}
}

// A field with no from= has nothing to read, and must say so rather than
// return some other field's value.
func TestEnvFromIsNilWithoutFrom(t *testing.T) {
	synth.Register("env_nofrom", func(r synth.R) any {
		if v := r.(synth.Env).From(); v != nil {
			return "unexpected"
		}
		return "nil"
	})
	type row struct {
		V string `synth:"env_nofrom"`
	}
	for _, r := range synth.Make[row](3, synth.WithSeed(1)) {
		if r.V != "nil" {
			t.Fatalf("From() without from= returned %q", r.V)
		}
	}
}
