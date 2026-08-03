package providers

import (
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

// sweepCtx builds a fully-populated Ctx so every provider has what it may read.
func sweepCtx(seed uint64, loc string, params map[string]string) Ctx {
	l := locale.Get(loc)
	p := l.Places[seed%uint64(len(l.Places))]
	if params == nil {
		params = map[string]string{}
	}
	gender := "male"
	if seed%2 == 0 {
		gender = "female"
	}
	f := &schema.Field{
		Name:    "field",
		Params:  params,
		Choices: []string{"a", "b", "c"},
		Weights: []float64{1, 2, 3},
	}
	return Ctx{
		Rand: rng.New(seed), Locale: l, Params: params, Field: f,
		Place: &p, Gender: gender,
		Sibling: func(name string) any {
			switch name {
			case "__from__":
				return "Jane Doe"
			default:
				return "123456789"
			}
		},
	}
}

// TestSweepAllProviders calls every registered provider across seeds and
// locales, with and without common params, asserting only that none panic and
// each returns non-nil. This drives the bulk of each provider body.
func TestSweepAllProviders(t *testing.T) {
	locales := []string{"en_US", "uz_UZ", "ru_RU"}
	paramSets := []map[string]string{
		{},
		{"min": "1", "max": "100"},
		{"min": "1.5", "max": "9.5", "decimals": "2"},
		{"from": "field", "match": "field"},
		{"expired": "true"},
		{"dist": "zipf"},
	}
	for k := range registry {
		p := Get(k)
		if p == nil {
			t.Fatalf("Get(%q) nil despite registry entry", k)
		}
		for _, loc := range locales {
			for seed := uint64(0); seed < 8; seed++ {
				for _, ps := range paramSets {
					c := sweepCtx(seed, loc, ps)
					c.Field.Kind = k
					got := p(c)
					if got == nil {
						t.Fatalf("provider %q returned nil (loc=%s seed=%d params=%v)", k, loc, seed, ps)
					}
				}
			}
		}
	}
}

func TestKindsAndHelpers(t *testing.T) {
	ks := Kinds()
	if len(ks) == 0 {
		t.Fatal("Kinds() empty")
	}
	// Every kind Kinds() reports must resolve to a provider.
	for _, k := range ks {
		if Get(k) == nil {
			t.Fatalf("Kinds() lists %q but Get returns nil", k)
		}
	}
	if !IsNumericKind(schema.KindInt) || IsNumericKind(schema.KindName) {
		t.Fatal("IsNumericKind wrong")
	}
	if len(LocalizableKinds()) == 0 {
		t.Fatal("LocalizableKinds empty")
	}
}
