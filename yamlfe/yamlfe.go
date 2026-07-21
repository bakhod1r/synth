// Package yamlfe is a structless frontend: it builds a Synth schema from a YAML
// document, so data can be described declaratively (and driven from the CLI)
// without writing Go types.
//
// Example:
//
//	name: users
//	count: 1000
//	locale: uz_UZ
//	seed: 42
//	fields:
//	  id:       { kind: uuid, pk: true }
//	  fullname: { kind: name }
//	  email:    { kind: email, from: fullname }
//	  status:   { kind: enum, choices: [active, inactive], weights: [0.9, 0.1] }
//	  amount:   { kind: amount, min: 100, max: 5000, dist: lognormal, mu: 8, sigma: 1 }
package yamlfe

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bakhodir/synth/constraint"
	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/schema"
	"gopkg.in/yaml.v3"
)

// Spec is a parsed YAML data definition.
type Spec struct {
	Name   string
	Count  int
	Locale string
	Seed   uint64
	Schema *schema.Schema
	// Order preserves field declaration order for stable CSV/SQL columns.
	Order []string
	// Constraints are cross-column invariants, either mined from a real
	// sample by the constraint package or written by hand. Generation
	// enforces them after the record's fields are produced.
	Constraints []constraint.Constraint
}

type doc struct {
	Name        string          `yaml:"name"`
	Count       int             `yaml:"count"`
	Locale      string          `yaml:"locale"`
	Seed        uint64          `yaml:"seed"`
	Fields      yaml.Node       `yaml:"fields"`
	Constraints []constraintDef `yaml:"constraints"`
}

// constraintDef is the YAML form of one invariant.
type constraintDef struct {
	Kind      string   `yaml:"kind"`
	Left      string   `yaml:"left"`
	Right     string   `yaml:"right"`
	Parts     []string `yaml:"parts"`
	Whole     string   `yaml:"whole"`
	When      string   `yaml:"when"`
	Equals    string   `yaml:"equals"`
	Then      string   `yaml:"then"`
	Exclusive bool     `yaml:"exclusive"`
	Lo        float64  `yaml:"lo"`
	Hi        float64  `yaml:"hi"`
}

type fieldDef struct {
	Kind  string `yaml:"kind"`
	From  string `yaml:"from"`
	Match string `yaml:"match"`
	// Min and Max are untyped: a numeric field bounds with numbers, a date
	// field bounds with dates, and declaring them as float64 made a spec
	// saying min: 2026-01-01 fail to parse at all.
	Min     any       `yaml:"min"`
	Max     any       `yaml:"max"`
	Dist    string    `yaml:"dist"`
	Mu      *float64  `yaml:"mu"`
	Sigma   *float64  `yaml:"sigma"`
	S       *float64  `yaml:"s"`
	Rate    *float64  `yaml:"rate"`
	Gap     string    `yaml:"gap"`
	Choices []string  `yaml:"choices"`
	Weights []float64 `yaml:"weights"`
	Unique  bool      `yaml:"unique"`
	PK      bool      `yaml:"pk"`
}

// Load parses a YAML spec file.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses a YAML spec from bytes.
func Parse(data []byte) (*Spec, error) {
	var d doc
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("yamlfe: parse: %w", err)
	}
	sp := &Spec{Name: d.Name, Count: d.Count, Locale: d.Locale, Seed: d.Seed, Schema: &schema.Schema{}}
	if sp.Locale == "" {
		sp.Locale = "en_US"
	}
	if sp.Count == 0 {
		sp.Count = 10
	}
	// fields is a mapping node: iterate key/value pairs to keep order.
	if d.Fields.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("yamlfe: 'fields' must be a mapping")
	}
	for i := 0; i < len(d.Fields.Content); i += 2 {
		name := d.Fields.Content[i].Value
		var fd fieldDef
		if err := d.Fields.Content[i+1].Decode(&fd); err != nil {
			return nil, fmt.Errorf("yamlfe: field %q: %w", name, err)
		}
		// Providers take their own parameters — strength, format, brand, words
		// — and there is no useful list of them here: each provider owns its
		// own. Anything not consumed above is passed through, so a spec can
		// reach every knob a tag can.
		var extra map[string]any
		if err := d.Fields.Content[i+1].Decode(&extra); err != nil {
			return nil, fmt.Errorf("yamlfe: field %q: %w", name, err)
		}
		sp.Schema.Fields = append(sp.Schema.Fields, toField(name, fd, extra))
		sp.Order = append(sp.Order, name)
	}
	// Wire up the automatic coherence the struct frontend applies: an email
	// derived from the name, a card brand read off its card number, timestamps
	// in causal order. A spec and an equivalent Go struct must produce the same
	// data — a field explicitly given a from= keeps it, since this only fills
	// links the author left blank.
	infer.LinkDependencies(sp.Schema)

	for i, cd := range d.Constraints {
		c, err := toConstraint(cd)
		if err != nil {
			return nil, fmt.Errorf("yamlfe: constraint %d: %w", i, err)
		}
		sp.Constraints = append(sp.Constraints, c)
	}
	return sp, nil
}

// knownKeys are the keys fieldDef already consumes. Everything else in a field
// mapping is a provider parameter.
var knownKeys = map[string]bool{
	"kind": true, "from": true, "match": true, "min": true, "max": true,
	"dist": true, "mu": true, "sigma": true, "s": true, "rate": true,
	"gap": true, "choices": true, "weights": true, "unique": true, "pk": true,
}

func toField(name string, fd fieldDef, extra map[string]any) schema.Field {
	f := schema.Field{Name: name, Params: map[string]string{}, From: fd.From, Match: fd.Match,
		Choices: fd.Choices, Weights: fd.Weights, Unique: fd.Unique, PK: fd.PK}
	if fd.PK {
		f.Unique = true
		if fd.Kind == "" {
			fd.Kind = "uuid"
		}
	}
	f.Kind = schema.Kind(fd.Kind)
	setVal(f.Params, "min", fd.Min)
	setVal(f.Params, "max", fd.Max)
	setNum(f.Params, "mu", fd.Mu)
	setNum(f.Params, "sigma", fd.Sigma)
	setNum(f.Params, "s", fd.S)
	setNum(f.Params, "rate", fd.Rate)
	if fd.Dist != "" {
		f.Params["dist"] = fd.Dist
	}
	if fd.Gap != "" {
		f.Params["gap"] = fd.Gap
	}
	for k, v := range extra {
		if knownKeys[k] || v == nil {
			continue
		}
		f.Params[k] = fmt.Sprint(v)
	}
	return f
}

func setNum(m map[string]string, key string, v *float64) {
	if v != nil {
		m[key] = fmt.Sprintf("%g", *v)
	}
}

// setVal records a bound that may be a number or a date. Numbers are written
// in %g form so "18" does not become "18.000000"; anything else is kept as the
// author wrote it, and the provider decides how to read it.
func setVal(m map[string]string, key string, v any) {
	switch x := v.(type) {
	case nil:
		return
	case float64:
		m[key] = fmt.Sprintf("%g", x)
	case int:
		m[key] = strconv.Itoa(x)
	case string:
		if x != "" {
			m[key] = x
		}
	default:
		m[key] = fmt.Sprint(x)
	}
}

// toConstraint converts the YAML form to a constraint, rejecting a kind it
// does not know rather than silently dropping an invariant the author meant
// to enforce.
func toConstraint(cd constraintDef) (constraint.Constraint, error) {
	switch constraint.Kind(cd.Kind) {
	case constraint.Ordering:
		return constraint.Constraint{Kind: constraint.Ordering, Left: cd.Left, Right: cd.Right}, nil
	case constraint.SumEquals:
		return constraint.Constraint{Kind: constraint.SumEquals, Parts: cd.Parts, Whole: cd.Whole}, nil
	case constraint.Implication:
		return constraint.Constraint{Kind: constraint.Implication, When: cd.When,
			Equals: cd.Equals, Then: cd.Then, Exclusive: cd.Exclusive}, nil
	case constraint.Range:
		return constraint.Constraint{Kind: constraint.Range, Left: cd.Left, Lo: cd.Lo, Hi: cd.Hi}, nil
	default:
		return constraint.Constraint{}, fmt.Errorf("unknown kind %q (want ordering, sum, implication or range)", cd.Kind)
	}
}
