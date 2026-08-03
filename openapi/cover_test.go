package openapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

const formatsSpec = `
openapi: 3.0.0
info: { title: t, version: "1" }
paths:
  /e:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                status: { type: string, enum: [open, closed] }
                when: { type: string, format: date-time }
                born: { type: string, format: date }
                site: { type: string, format: uri }
                homepage: { type: string, format: url }
                ip: { type: string, format: ipv4 }
                score: { type: number }
                flag: { type: boolean }
`

func TestFormatAndEnumMapping(t *testing.T) {
	s, err := Parse([]byte(formatsSpec))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := s.Schema("post", "/e")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]schema.Kind{
		"status":   schema.KindEnum,
		"when":     schema.KindTime,
		"born":     schema.KindTime,
		"site":     schema.KindURL,
		"homepage": schema.KindURL,
		"ip":       schema.KindIPv4,
		"score":    schema.KindFloat,
		"flag":     schema.KindBool,
	}
	for col, kind := range want {
		if f := sc.FieldByName(col); f == nil || f.Kind != kind {
			t.Fatalf("%s = %v, want %v", col, f, kind)
		}
	}
	if f := sc.FieldByName("status"); len(f.Choices) != 2 {
		t.Fatalf("enum choices = %v", f.Choices)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(p, []byte(formatsSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("missing file should error")
	}
}

func TestPayloadJSON(t *testing.T) {
	b, err := PayloadJSON(map[string]any{"a": 1, "b": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty payload")
	}
}
