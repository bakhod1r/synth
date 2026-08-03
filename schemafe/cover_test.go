package schemafe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

const jsonSchemaAll = `{
  "title": "User",
  "properties": {
    "email":    {"type": "string", "format": "email"},
    "id":       {"type": "string", "format": "uuid"},
    "when":     {"type": "string", "format": "date-time"},
    "born":     {"type": "string", "format": "date"},
    "site":     {"type": "string", "format": "uri"},
    "home":     {"type": "string", "format": "url"},
    "ip4":      {"type": "string", "format": "ipv4"},
    "ip6":      {"type": "string", "format": "ipv6"},
    "host":     {"type": "string", "format": "hostname"},
    "age":      {"type": "integer", "minimum": 18, "maximum": 90},
    "score":    {"type": "number"},
    "active":   {"type": "boolean"},
    "city":     {"type": "string"},
    "freeform": {"type": "string", "maxLength": 40},
    "status":   {"enum": ["a", "b"]},
    "mystery":  {}
  }
}`

func TestParseJSONSchemaAll(t *testing.T) {
	tbl, err := ParseJSONSchema([]byte(jsonSchemaAll))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]schema.Kind{
		"email": schema.KindEmail, "id": schema.KindUUID, "when": schema.KindTime,
		"born": schema.KindTime, "site": schema.KindURL, "home": schema.KindURL,
		"ip4": schema.KindIPv4, "ip6": schema.KindIPv6, "host": schema.KindDomain,
		"age": schema.KindInt, "score": schema.KindFloat, "active": schema.KindBool,
		"city": schema.KindCity, "freeform": schema.KindLorem, "status": schema.KindEnum,
	}
	for col, kind := range want {
		if f := tbl.Schema.FieldByName(col); f == nil || f.Kind != kind {
			t.Fatalf("%s = %v, want %v", col, f, kind)
		}
	}
	if tbl.Schema.FieldByName("age").Params["min"] != "18" {
		t.Fatal("min not applied")
	}
	if tbl.Schema.FieldByName("freeform").Params["maxlen"] != "40" {
		t.Fatal("maxlen not applied")
	}
	if len(tbl.Schema.FieldByName("status").Choices) != 2 {
		t.Fatal("enum not applied")
	}
}

func TestParseJSONSchemaErrors(t *testing.T) {
	if _, err := ParseJSONSchema([]byte("{bad")); err == nil {
		t.Fatal("invalid json should error")
	}
	if _, err := ParseJSONSchema([]byte(`{"properties":{}}`)); err == nil {
		t.Fatal("no properties should error")
	}
}

const avroAll = `{
  "type": "record",
  "name": "Event",
  "fields": [
    {"name": "n", "type": "int"},
    {"name": "big", "type": "long"},
    {"name": "f", "type": "float"},
    {"name": "d", "type": "double"},
    {"name": "ok", "type": "boolean"},
    {"name": "city", "type": "string"},
    {"name": "raw", "type": "bytes"},
    {"name": "opt", "type": ["null", "string"]},
    {"name": "uid", "type": {"type": "string", "logicalType": "uuid"}},
    {"name": "ts", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "amt", "type": {"type": "bytes", "logicalType": "decimal"}},
    {"name": "nested", "type": {"type": "string"}},
    {"name": "weird", "type": 42}
  ]
}`

func TestParseAvroAll(t *testing.T) {
	tbl, err := ParseAvro([]byte(avroAll))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]schema.Kind{
		"n": schema.KindInt, "big": schema.KindInt, "f": schema.KindFloat,
		"d": schema.KindFloat, "ok": schema.KindBool, "city": schema.KindCity,
		"opt": schema.KindLorem, "uid": schema.KindUUID, "ts": schema.KindTime,
		"amt": schema.KindFloat, "weird": schema.KindLorem,
	}
	for col, kind := range want {
		if f := tbl.Schema.FieldByName(col); f == nil || f.Kind != kind {
			t.Fatalf("%s = %v, want %v", col, f, kind)
		}
	}
}

func TestParseAvroErrors(t *testing.T) {
	if _, err := ParseAvro([]byte("{bad")); err == nil {
		t.Fatal("invalid json should error")
	}
	if _, err := ParseAvro([]byte(`{"type":"enum","fields":[]}`)); err == nil {
		t.Fatal("non-record should error")
	}
}

func TestParseAvroNoNameAndUnknownPrimitive(t *testing.T) {
	// No name defaults to "data"; an unknown primitive type falls back to lorem.
	tbl, err := ParseAvro([]byte(`{"type":"record","fields":[{"name":"x","type":"fixed"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "data" {
		t.Fatalf("name = %q, want data", tbl.Name)
	}
	if tbl.Schema.FieldByName("x").Kind != schema.KindLorem {
		t.Fatal("unknown primitive should be lorem")
	}
}

func TestParseDialectDetection(t *testing.T) {
	if _, err := Parse([]byte(avroAll)); err != nil {
		t.Fatalf("avro via Parse: %v", err)
	}
	if _, err := Parse([]byte(jsonSchemaAll)); err != nil {
		t.Fatalf("json via Parse: %v", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.json")
	os.WriteFile(p, []byte(jsonSchemaAll), 0o644)
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("missing file should error")
	}
}

func TestMapKeysFallback(t *testing.T) {
	// mapKeys is the order fallback; call it directly for determinism coverage.
	if got := mapKeys(map[string]jsonSchemaDoc{"a": {}, "b": {}}); len(got) != 2 {
		t.Fatalf("mapKeys = %v", got)
	}
	// orderedKeys on data with no such field errors.
	if _, err := orderedKeys([]byte(`{"x":1}`), "properties"); err == nil {
		t.Fatal("missing field should error")
	}
	if _, err := orderedKeys([]byte(`not json`), "properties"); err == nil {
		t.Fatal("invalid json should error")
	}
}
