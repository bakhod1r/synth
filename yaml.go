package synth

import (
	"fmt"

	"github.com/bakhodir/synth/constraint"
	"github.com/bakhodir/synth/gen"
	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/schema"
	"github.com/bakhodir/synth/yamlfe"
)

// YAMLSpec is a parsed declarative data definition (see the yamlfe package).
type YAMLSpec struct {
	spec *yamlfe.Spec
}

// LoadYAML parses a YAML data-definition file.
func LoadYAML(path string) (*YAMLSpec, error) {
	s, err := yamlfe.Load(path)
	if err != nil {
		return nil, err
	}
	return &YAMLSpec{spec: s}, nil
}

// YAMLBytes parses a YAML data definition from bytes.
func YAMLBytes(data []byte) (*YAMLSpec, error) {
	s, err := yamlfe.Parse(data)
	if err != nil {
		return nil, err
	}
	return &YAMLSpec{spec: s}, nil
}

// Columns returns the field names in declaration order (for CSV/SQL headers).
func (y *YAMLSpec) Columns() []string { return y.spec.Order }

// Name returns the spec's declared dataset name (used as an SQL table name).
func (y *YAMLSpec) Name() string { return y.spec.Name }

// Count returns the number of rows the spec requests.
func (y *YAMLSpec) Count() int { return y.spec.Count }

// Generate produces the spec's records as field→value maps. Options override
// the spec's seed/locale when provided.
func (y *YAMLSpec) Generate(opts ...Option) ([]map[string]any, error) {
	cfg := config{seed: y.spec.Seed, locale: y.spec.Locale}
	for _, o := range opts {
		o(&cfg)
	}
	// Clone schema so per-call overrides don't mutate the parsed spec.
	s := &schema.Schema{Fields: append([]schema.Field(nil), y.spec.Schema.Fields...)}
	applyWeighted(s, cfg.weighted)
	if cfg.unmask {
		stripMasks(s)
	}
	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	eng.Chaos = cfg.chaos
	base := rng.New(cfg.seed)
	out := make([]map[string]any, y.spec.Count)
	for i := 0; i < y.spec.Count; i++ {
		out[i] = eng.Record(base, i)
	}
	if err := enforceAll(y.spec.Constraints, out); err != nil {
		return nil, err
	}
	return out, nil
}

// stripMasks removes every mask= setting, so the raw generated value is
// returned. The field's Params map is shared with the cached spec, so it is
// copied rather than edited in place — otherwise one unmasked call would
// silently unmask every later one.
func stripMasks(s *schema.Schema) {
	for i := range s.Fields {
		if _, ok := s.Fields[i].Params["mask"]; !ok {
			continue
		}
		p := make(map[string]string, len(s.Fields[i].Params))
		for k, v := range s.Fields[i].Params {
			if k != "mask" {
				p[k] = v
			}
		}
		s.Fields[i].Params = p
	}
}

// Constraints returns the spec's cross-column invariants.
func (y *YAMLSpec) Constraints() []constraint.Constraint { return y.spec.Constraints }

// enforceAll repairs each record so the spec's invariants hold. A record that
// cannot be repaired means the constraints contradict each other, which is an
// error in the spec — reporting it beats emitting data that quietly violates
// what the spec promises.
func enforceAll(cs []constraint.Constraint, recs []map[string]any) error {
	if len(cs) == 0 {
		return nil
	}
	for i, rec := range recs {
		if !constraint.Enforce(cs, rec) {
			for _, c := range cs {
				if !c.Holds(rec) {
					return fmt.Errorf("synth: row %d cannot satisfy constraint %q — "+
						"the spec's constraints contradict each other", i, c)
				}
			}
		}
	}
	return nil
}
