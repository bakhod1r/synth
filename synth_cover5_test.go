package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
)

func TestPoolWorkersDefaultAndFeatures(t *testing.T) {
	// workers <= 0 uses GOMAXPROCS.
	rows, err := synth.MakeParallel[CoverRow](10, 0, synth.WithSeed(1), synth.WithChaos(0.1))
	if err != nil || len(rows) != 10 {
		t.Fatalf("MakeParallel default workers: %v", err)
	}
}

func TestDDLBytesAndGenerate(t *testing.T) {
	tables, err := synth.DDLBytes([]byte("CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) == 0 {
		t.Fatal("no tables parsed")
	}
	rows, err := tables[0].Generate(3, synth.WithSeed(1))
	if err != nil || len(rows) != 3 {
		t.Fatalf("ddl generate: %v", err)
	}
}

func TestOpenAPIPayloads(t *testing.T) {
	spec := []byte(`
openapi: 3.0.0
info: { title: t, version: "1" }
paths:
  /u:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: { type: string, format: uuid }
                age: { type: integer, minimum: 1, maximum: 9 }
`)
	api, err := synth.OpenAPIBytes(spec)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := api.Payloads("post", "/u", 2, synth.WithSeed(1))
	if err != nil || len(rows) != 2 {
		t.Fatalf("payloads: %v", err)
	}
	if _, err := api.PayloadJSON("post", "/u", synth.WithSeed(1)); err != nil {
		t.Fatal(err)
	}
	// Unknown operation surfaces the schema error on both paths.
	if _, err := api.Payloads("get", "/missing", 1); err == nil {
		t.Fatal("unknown op Payloads should error")
	}
	if _, err := api.PayloadJSON("get", "/missing"); err == nil {
		t.Fatal("unknown op PayloadJSON should error")
	}
}

func TestYAMLGenerateNUnmaskAndOffset(t *testing.T) {
	spec := []byte(`
name: t
count: 4
fields:
  id: { kind: uuid, pk: true }
  card: { kind: card, mask: partial }
`)
	y, err := synth.YAMLBytes(spec)
	if err != nil {
		t.Fatal(err)
	}
	// Unmasked + offset exercise those branches in GenerateN.
	rows, err := y.GenerateN(4, synth.Unmasked(), synth.Offset(100), synth.WithSeed(1))
	if err != nil || len(rows) != 4 {
		t.Fatalf("generateN: %v", err)
	}
}
