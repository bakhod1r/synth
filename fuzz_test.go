package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/constraint"
)

// Fuzzing the parsers.
//
// These are the entry points that take input Synth did not write. A YAML spec
// or a DDL file is usually the author's own, but `profile`, `verify` and `mask`
// are pointed at exports from elsewhere, and the MCP server hands a model's
// output straight to them. A malformed file must produce an error, never a
// panic: a panic in a library takes down whatever embedded it, and in the MCP
// server it kills the connection for every other tool too.
//
// Each target asserts only that: parse or fail, never crash. Correctness of the
// parse is the ordinary tests' job.
//
// Run longer than the seed corpus with:
//
//	go test -run='^$' -fuzz=FuzzYAMLSpec -fuzztime=60s .

func FuzzYAMLSpec(f *testing.F) {
	f.Add("name: t\nfields:\n  a: { kind: city }\n")
	f.Add("name: t\ncount: 5\nseed: 1\nfields:\n  id: { kind: uuid, pk: true }\n")
	f.Add("fields:\n  a: { kind: time, min: 1960-01-01, max: 2006-12-31 }\n")
	f.Add("fields:\n  a: { kind: enum, choices: [x, y], weights: [0.5, 0.5] }\n")
	f.Add("fields:\n  a: { kind: int, min: 9223372036854775807 }\n")
	f.Add("count: -1\nfields:\n  a: { kind: card, mask: hash, digest: 0 }\n")
	f.Add("")
	f.Add("\x00")
	f.Add("fields: [not, a, map]")

	f.Fuzz(func(t *testing.T, src string) {
		spec, err := synth.YAMLBytes([]byte(src))
		if err != nil {
			return // rejecting bad input is the correct outcome
		}
		// A spec that parsed must also generate without panicking. Parsing is
		// only half the surface: a bound or a reference that survives parsing
		// and then explodes in the engine is the more dangerous half.
		spec.SetCount(3)
		if _, err := spec.Generate(); err != nil {
			return
		}
	})
}

func FuzzDDL(f *testing.F) {
	f.Add("CREATE TABLE users (id UUID PRIMARY KEY, email TEXT NOT NULL);")
	f.Add("create table t(a int, b varchar(10), c timestamp default now());")
	f.Add("CREATE TABLE t (a NUMERIC(10,2), b BOOLEAN, c JSONB);")
	f.Add("CREATE TABLE (")
	f.Add("CREATE TABLE t (a INT,,,);")
	f.Add("")

	f.Fuzz(func(t *testing.T, src string) {
		tables, err := synth.DDLBytes([]byte(src))
		if err != nil {
			return
		}
		for _, tb := range tables {
			if _, err := tb.Generate(3); err != nil {
				return
			}
		}
	})
}

func FuzzOpenAPI(f *testing.F) {
	f.Add(`{"openapi":"3.0.0","components":{"schemas":{"U":{"type":"object","properties":{"id":{"type":"string","format":"uuid"}}}}}}`)
	f.Add(`openapi: 3.0.0
components:
  schemas:
    U:
      type: object
      properties:
        email: { type: string, format: email }
`)
	f.Add(`{"components":{"schemas":{"A":{"$ref":"#/components/schemas/A"}}}}`) // self-reference
	f.Add(`{"openapi":`)
	f.Add("")

	f.Fuzz(func(t *testing.T, src string) {
		doc, err := synth.OpenAPIBytes([]byte(src))
		if err != nil || doc == nil {
			return
		}
		// The path is arbitrary: a spec that parsed must reject an unknown
		// operation with an error rather than panicking on the lookup.
		if _, err := doc.Payloads("post", "/users", 2); err != nil {
			return
		}
	})
}

func FuzzJSONSchema(f *testing.F) {
	f.Add(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}`)
	f.Add(`{"type":"record","name":"U","fields":[{"name":"id","type":"string"}]}`) // Avro
	f.Add(`{"type":"object","properties":{"a":{"$ref":"#"}}}`)                     // recursive
	f.Add(`{"type":`)
	f.Add("")

	f.Fuzz(func(t *testing.T, src string) {
		spec, err := synth.JSONSchemaBytes([]byte(src))
		if err != nil || spec == nil {
			return
		}
		if _, err := spec.Generate(2); err != nil {
			return
		}
	})
}

// The sample reader is what verify, profile and mask all read through, and it
// is pointed at files the caller did not produce.
func FuzzReadSample(f *testing.F) {
	f.Add("id,name\n1,Ann\n2,Bo\n", "csv")
	f.Add("{\"id\":1}\n{\"id\":2}\n", "jsonl")
	f.Add("a,b\n1\n2,3,4\n", "csv") // ragged rows
	f.Add("\"unterminated\n", "csv")
	f.Add("{\"a\":", "jsonl")
	f.Add("", "csv")
	f.Add(strings.Repeat("a,", 10_000)+"\n", "csv") // very wide

	f.Fuzz(func(t *testing.T, src, format string) {
		rows, err := constraint.ReadSample(strings.NewReader(src), format, 100)
		if err != nil {
			return
		}
		// Rows that parsed must also survive mining, which reads across whole
		// records and is where a ragged or absurd row would show up.
		constraint.Mine(rows, 1.0)
	})
}

// Profiling reads a real export and infers a schema; the inferred spec is then
// handed straight back to the generator, so a spec that cannot round-trip is a
// broken feature rather than a bad input.
func FuzzProfile(f *testing.F) {
	f.Add("email,age\na@example.com,31\nb@example.com,44\n", "csv")
	f.Add("a\n\n\n", "csv")
	f.Add("a,b\n\"x\",\n", "csv")
	f.Add("{\"n\":1e400}\n", "jsonl") // out of float range
	f.Add("", "csv")

	f.Fuzz(func(t *testing.T, src, format string) {
		p, err := synth.ProfileBytes([]byte(src), format)
		if err != nil || p == nil {
			return
		}
		spec, err := p.YAML("t", 3)
		if err != nil {
			return
		}
		// The inferred spec must at least parse. Producing YAML that Synth
		// itself rejects would make profiling useless on exactly the inputs
		// that need it most.
		if _, err := synth.YAMLBytes(spec); err != nil {
			t.Fatalf("profile produced a spec Synth cannot parse: %v\n%s", err, spec)
		}
	})
}
