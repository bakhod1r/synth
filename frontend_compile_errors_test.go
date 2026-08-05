package synth

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/constraint"
	"github.com/bakhod1r/synth/profile"
	"github.com/bakhod1r/synth/protofe"
	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/schemafe"
)

// badSchema is the smallest schema gen.Compile rejects: an array field with no
// element type has nothing to generate, so compilation fails rather than
// dereferencing nil on the first record.
func badSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{{
		Name:   "items",
		Kind:   schema.KindArray,
		Params: map[string]string{},
	}}}
}

// Every frontend compiles the schema it parsed before generating, and each has
// to hand the compile error back rather than generate a broken column. The
// frontends are exercised through their own structs because a well-formed
// source file cannot produce a schema its own parser would reject — the error
// belongs to the shared compile step, not to any one parser.
func TestFrontendsReturnCompileErrors(t *testing.T) {
	bad := badSchema()
	cases := []struct {
		name string
		gen  func(int, ...Option) ([]map[string]any, error)
	}{
		{"ddl", (&DDLTable{name: "t", order: []string{"items"}, schema: bad}).Generate},
		{"schemafile", (&SchemaFile{tbl: &schemafe.Table{Name: "t", Order: []string{"items"}, Schema: bad}}).Generate},
		{"proto", (&ProtoMessage{msg: &protofe.Message{Name: "T", Order: []string{"items"}, Schema: bad}}).Generate},
		{"profiled", (&Profiled{res: &profile.Result{Schema: bad, Order: []string{"items"}}}).Generate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := tc.gen(3)
			if err == nil {
				t.Fatalf("Generate = %v, want an error for an array field with no element type", rows)
			}
			if !strings.Contains(err.Error(), "no element type") {
				t.Errorf("error = %q, want it to name the uncompilable field", err)
			}
			if rows != nil {
				t.Errorf("rows = %v, want nil alongside the error", rows)
			}
		})
	}
}

// Generate on a preset delegates to the YAML frontend; an unknown preset never
// reaches it.
func TestGenerateUnknownPreset(t *testing.T) {
	if _, err := Generate(Preset("no-such-preset"), 1); err == nil {
		t.Fatal("Generate = nil error, want one for an unknown preset")
	}
}

// A profile carries mined invariants as well as a schema, and enforcement can
// fail: a range whose bounds cross has no satisfying value, so no amount of
// repair makes the row hold. Generate reports that instead of returning rows
// that quietly violate the constraints it was given.
func TestProfiledGenerateReportsUnsatisfiableConstraint(t *testing.T) {
	p := &Profiled{
		res: &profile.Result{
			Order: []string{"n"},
			Schema: &schema.Schema{Fields: []schema.Field{{
				Name:   "n",
				Kind:   schema.KindInt,
				Params: map[string]string{"min": "0", "max": "100"},
			}}},
		},
		cons: []constraint.Constraint{{Kind: constraint.Range, Left: "n", Lo: 10, Hi: 0}},
	}
	rows, err := p.Generate(4)
	if err == nil {
		t.Fatalf("Generate = %v, want an error: no value satisfies 10 <= n <= 0", rows)
	}
	if !strings.Contains(err.Error(), "constraint") {
		t.Errorf("error = %q, want it to say which constraint could not hold", err)
	}
}

// exhaustibleSchema declares a column unique whose value space holds two
// values, so a run of any length must report exhaustion instead of repeating
// one of them.
func exhaustibleSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{{
		Name:    "status",
		Kind:    schema.KindEnum,
		Choices: []string{"new", "open"},
		Unique:  true,
		Params:  map[string]string{},
	}}}
}

// Exhaustion is found while generating, not while compiling, so every frontend
// has to check for it after its loop — a frontend that forgets returns rows
// with a duplicate in a column it promised was unique. As above, the frontends
// are driven through their own structs: most parsers have no syntax for
// uniqueness, but they all share the generation step that enforces it.
func TestFrontendsReturnExhaustionErrors(t *testing.T) {
	bad := exhaustibleSchema()
	cases := []struct {
		name string
		gen  func(int, ...Option) ([]map[string]any, error)
	}{
		{"ddl", (&DDLTable{name: "t", order: []string{"status"}, schema: bad}).Generate},
		{"schemafile", (&SchemaFile{tbl: &schemafe.Table{Name: "t", Order: []string{"status"}, Schema: bad}}).Generate},
		{"proto", (&ProtoMessage{msg: &protofe.Message{Name: "T", Order: []string{"status"}, Schema: bad}}).Generate},
		{"profiled", (&Profiled{res: &profile.Result{Schema: bad, Order: []string{"status"}}}).Generate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := tc.gen(50)
			if err == nil {
				t.Fatalf("Generate = %v, want an exhaustion error", rows)
			}
			if !strings.Contains(err.Error(), "ran out of unique values") {
				t.Errorf("error = %q, want it to name the exhausted field", err)
			}
			if rows != nil {
				t.Errorf("rows = %v, want nil alongside the error", rows)
			}
		})
	}
}
