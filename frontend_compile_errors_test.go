package synth

import (
	"errors"
	"os"
	"path/filepath"
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

// An OpenAPI request body reaches the same check through a real document: a
// property declared `type: array` with no `items` is a spec a user can and does
// write, and Payloads has to report it rather than panic.
func TestPayloadsRejectsArrayWithoutItems(t *testing.T) {
	doc := `openapi: 3.0.0
info: {title: t, version: "1"}
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                tags: {type: array}
`
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	api, err := OpenAPI(path)
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	if _, err := api.Payloads("post", "/users", 2); err == nil {
		t.Fatal("Payloads = nil error, want a compile error for an array with no items")
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

// errWriter fails every write, standing in for a full disk or a closed pipe.
type errWriter struct{ err error }

func (e errWriter) Write([]byte) (int, error) { return 0, e.err }

// encodeCSV writes the header before any record, so a writer that fails
// immediately proves the header's error is propagated rather than dropped —
// the case where the caller would otherwise be told a truncated file is fine.
func TestEncodeCSVHeaderWriteError(t *testing.T) {
	boom := errors.New("disk full")
	type row struct{ A string }
	err := encodeCSV(errWriter{boom}, []row{{A: "x"}})
	if !errors.Is(err, boom) {
		t.Fatalf("encodeCSV = %v, want the writer's error", err)
	}
}
