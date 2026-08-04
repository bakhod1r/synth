package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

// Every frontend compiles the same engine, so every frontend must report the
// same schema mistake rather than generating a column of empty values.

const badKindYAML = `name: bad
count: 3
fields:
  a: { kind: no-such-kind }
`

func TestYAMLGenerateReportsUnknownKind(t *testing.T) {
	spec, err := synth.YAMLBytes([]byte(badKindYAML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = spec.Generate()
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("err = %v, want an unknown-kind error", err)
	}
}

func TestYAMLGenerateReportsBadDerive(t *testing.T) {
	spec, err := synth.YAMLBytes([]byte(`name: bad
count: 2
fields:
  city:   { kind: city }
  income: { kind: float, derive: city }
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := spec.Generate(); err == nil {
		t.Fatal("deriving a number from a text column should be an error")
	}
}

func TestGenerateNOverridesCount(t *testing.T) {
	spec, err := synth.Spec(synth.PresetUser)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := spec.GenerateN(7, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 7 {
		t.Fatalf("got %d rows, want 7", len(rows))
	}
	// A non-positive n leaves the spec's own count alone.
	spec.SetCount(4)
	rows, err = spec.GenerateN(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want the spec's 4", len(rows))
	}
}

func TestPresetErrors(t *testing.T) {
	if _, err := synth.Generate("nope", 1); err == nil {
		t.Error("Generate: unknown preset should error")
	}
	if _, err := synth.Spec("nope"); err == nil {
		t.Error("Spec: unknown preset should error")
	}
	if _, ok := synth.PresetSpec("nope"); ok {
		t.Error("PresetSpec: unknown preset should report !ok")
	}
	for _, p := range synth.Presets() {
		if _, ok := synth.PresetSpec(p); !ok {
			t.Errorf("preset %q has no spec", p)
		}
	}
}

func TestDDLBytesIgnoresNonDDLText(t *testing.T) {
	tables, err := synth.DDLBytes([]byte("-- a comment, no CREATE TABLE here\nSELECT 1;"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 0 {
		t.Fatalf("got %d tables, want 0", len(tables))
	}
}

func TestDDLGenerateRoundTrip(t *testing.T) {
	tables, err := synth.DDLBytes([]byte(`CREATE TABLE customers (
  id UUID PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  age INT,
  balance NUMERIC(12,2),
  created_at TIMESTAMP
);`))
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	tbl := tables[0]
	if tbl.Name() != "customers" {
		t.Fatalf("name = %q", tbl.Name())
	}
	want := []string{"id", "email", "age", "balance", "created_at"}
	if strings.Join(tbl.Columns(), ",") != strings.Join(want, ",") {
		t.Fatalf("columns = %v, want %v", tbl.Columns(), want)
	}
	rows, err := tbl.Generate(20, synth.WithSeed(4))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 20 {
		t.Fatalf("got %d rows, want 20", len(rows))
	}
	for _, r := range rows {
		for _, c := range want {
			if _, ok := r[c]; !ok {
				t.Fatalf("row is missing column %q: %v", c, r)
			}
		}
	}
}

func TestProtoBytesRejectsGarbage(t *testing.T) {
	if _, err := synth.ProtoBytes([]byte("syntax = \"proto3\";")); err == nil {
		t.Fatal("a file with no message should error")
	}
}

func TestJSONSchemaBytesErrors(t *testing.T) {
	if _, err := synth.JSONSchemaBytes([]byte("{not json")); err == nil {
		t.Error("invalid JSON should error")
	}
	if _, err := synth.JSONSchemaBytes([]byte(`{"type":"object"}`)); err == nil {
		t.Error("a schema with no properties should error")
	}
}

func TestAvroBytesErrors(t *testing.T) {
	if _, err := synth.AvroBytes([]byte("{not json")); err == nil {
		t.Error("invalid JSON should error")
	}
}

func TestOpenAPIBytesErrors(t *testing.T) {
	if _, err := synth.OpenAPIBytes([]byte("::: not yaml :::")); err == nil {
		t.Error("invalid YAML should error")
	}
}

func TestOpenAPIUnknownRoute(t *testing.T) {
	spec, err := synth.OpenAPIBytes([]byte(`openapi: 3.0.0
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
                email: {type: string, format: email}
                age:   {type: integer, minimum: 18, maximum: 90}
`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Payload("POST", "/nope"); err == nil {
		t.Error("unknown path should error")
	}
	if _, err := spec.Payloads("DELETE", "/users", 1); err == nil {
		t.Error("unknown method should error")
	}
	if _, err := spec.PayloadJSON("GET", "/users"); err == nil {
		t.Error("PayloadJSON should surface the route error")
	}

	rec, err := spec.Payload("POST", "/users", synth.WithSeed(5))
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := rec["email"].(string); !strings.Contains(s, "@") {
		t.Fatalf("email = %v", rec["email"])
	}
	js, err := spec.PayloadJSON("POST", "/users", synth.WithSeed(5))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"email"`) {
		t.Fatalf("payload JSON = %s", js)
	}
}

func TestMakeParallelRejectsNonStruct(t *testing.T) {
	if _, err := synth.MakeParallel[int](10, 2); err == nil {
		t.Fatal("expected a struct-type error")
	}
}

func TestMakeParallelReportsCompileError(t *testing.T) {
	if _, err := synth.MakeParallel[badKindRow](10, 2); err == nil {
		t.Fatal("expected an unknown-kind error")
	}
}

type uniqueRow struct {
	Email string `synth:"email,unique"`
}

func TestMakeParallelRefusesUniqueFields(t *testing.T) {
	_, err := synth.MakeParallel[uniqueRow](10, 4)
	if err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("err = %v, want a unique-field refusal", err)
	}
}

// Worker count must not change the data: record i is seeded from i.
func TestMakeParallelIndependentOfWorkerCount(t *testing.T) {
	one, err := synth.MakeParallel[streamRow](200, 1, synth.WithSeed(11))
	if err != nil {
		t.Fatal(err)
	}
	eight, err := synth.MakeParallel[streamRow](200, 8, synth.WithSeed(11))
	if err != nil {
		t.Fatal(err)
	}
	auto, err := synth.MakeParallel[streamRow](200, 0, synth.WithSeed(11))
	if err != nil {
		t.Fatal(err)
	}
	for i := range one {
		if one[i] != eight[i] || one[i] != auto[i] {
			t.Fatalf("row %d differs across worker counts: %+v %+v %+v", i, one[i], eight[i], auto[i])
		}
	}
}

func TestMakeParallelMoreWorkersThanRows(t *testing.T) {
	rows, err := synth.MakeParallel[streamRow](3, 16, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
}
