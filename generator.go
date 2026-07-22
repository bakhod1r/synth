package synth

import (
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// Config configures a standalone Generator.
type Config struct {
	Seed   uint64
	Locale string // "uz_UZ", "en_US", ...
}

// Generator is a stateful, per-instance value source. Each Generator owns its
// own RNG, so different goroutines using different Generators never contend —
// unlike a package-level faker guarded by a global mutex. Not safe for
// concurrent use by ONE Generator; give each goroutine its own.
type Generator struct {
	rng   *rng.Rand
	loc   *locale.Locale
	place locale.Place
}

// New returns a Generator. A zero Config uses a random seed and en_US.
func New(cfg ...Config) *Generator {
	c := Config{Locale: "en_US"}
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Locale == "" {
			c.Locale = "en_US"
		}
	}
	loc := locale.Get(c.Locale)
	r := rng.New(c.Seed)
	return &Generator{rng: r, loc: loc, place: loc.Places[r.Pick(len(loc.Places))]}
}

func (g *Generator) val(k schema.Kind, params map[string]string) any {
	return providers.Get(k)(providers.Ctx{
		Rand: g.rng, Locale: g.loc, Params: params, Place: &g.place,
	})
}

// Convenience single-value accessors. Each reuses the Generator's RNG.
func (g *Generator) Name() string      { return g.val(schema.KindName, nil).(string) }
func (g *Generator) FirstName() string { return g.val(schema.KindFirstName, nil).(string) }
func (g *Generator) Email() string     { return g.val(schema.KindEmail, nil).(string) }
func (g *Generator) Phone() string     { return g.val(schema.KindPhone, nil).(string) }
func (g *Generator) City() string      { return g.val(schema.KindCity, nil).(string) }
func (g *Generator) Region() string    { return g.val(schema.KindRegion, nil).(string) }
func (g *Generator) Country() string   { return g.val(schema.KindCountry, nil).(string) }
func (g *Generator) Postcode() string  { return g.val(schema.KindPostcode, nil).(string) }
func (g *Generator) Company() string   { return g.val(schema.KindCompany, nil).(string) }
func (g *Generator) Username() string  { return g.val(schema.KindUsername, nil).(string) }
func (g *Generator) Card() string      { return g.val(schema.KindCard, nil).(string) }
func (g *Generator) IBAN() string      { return g.val(schema.KindIBAN, nil).(string) }
func (g *Generator) IPv4() string      { return g.val(schema.KindIPv4, nil).(string) }
func (g *Generator) URL() string       { return g.val(schema.KindURL, nil).(string) }
func (g *Generator) Currency() string  { return g.val(schema.KindCurrency, nil).(string) }

// Amount returns a monetary value in [min,max].
func (g *Generator) Amount(min, max int) float64 {
	return g.val(schema.KindAmount, map[string]string{"min": itoa(min), "max": itoa(max)}).(float64)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
