package schemafe

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

const jsonDoc = `{
  "title": "User",
  "type": "object",
  "required": ["id", "email"],
  "properties": {
    "id":      {"type": "string", "format": "uuid"},
    "email":   {"type": "string", "format": "email"},
    "age":     {"type": "integer", "minimum": 18, "maximum": 90},
    "active":  {"type": "boolean"},
    "score":   {"type": "number"},
    "status":  {"type": "string", "enum": ["new", "open", "closed"]},
    "tags":    {"type": "array", "items": {"type": "string"}},
    "address": {"type": "object", "properties": {"city": {"type": "string"}}}
  }
}`

const avroDoc = `{
  "type": "record",
  "name": "User",
  "fields": [
    {"name": "id",     "type": "string"},
    {"name": "age",    "type": "int"},
    {"name": "score",  "type": "double"},
    {"name": "active", "type": "boolean"},
    {"name": "note",   "type": ["null", "string"]}
  ]
}`

func TestJSONSchemaTypes(t *testing.T) {
	s, err := ParseJSONSchema([]byte(jsonDoc))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "User" {
		t.Errorf("name = %q, want User", s.Name)
	}
	for col, kind := range map[string]schema.Kind{
		"id":     schema.KindUUID,
		"email":  schema.KindEmail,
		"age":    schema.KindInt,
		"active": schema.KindBool,
		"score":  schema.KindFloat,
		"status": schema.KindEnum,
	} {
		f := s.Schema.FieldByName(col)
		if f == nil {
			t.Errorf("no property %q", col)
			continue
		}
		if f.Kind != kind {
			t.Errorf("%s: kind = %q, want %q", col, f.Kind, kind)
		}
	}
}

// An enum whose choices are lost generates arbitrary strings, and the consumer
// rejects every one of them.
func TestEnumChoices(t *testing.T) {
	s, err := ParseJSONSchema([]byte(jsonDoc))
	if err != nil {
		t.Fatal(err)
	}
	f := s.Schema.FieldByName("status")
	if f == nil {
		t.Fatal("no status")
	}
	if len(f.Choices) != 3 {
		t.Fatalf("choices = %v, want three", f.Choices)
	}
}

func TestNumericBounds(t *testing.T) {
	s, err := ParseJSONSchema([]byte(jsonDoc))
	if err != nil {
		t.Fatal(err)
	}
	f := s.Schema.FieldByName("age")
	if f == nil {
		t.Fatal("no age")
	}
	if f.Params["min"] != "18" || f.Params["max"] != "90" {
		t.Fatalf("bounds = %v..%v, want 18..90", f.Params["min"], f.Params["max"])
	}
}

// Property order decides the column order of a generated CSV, and a map's order
// is random — so a schema that did not preserve it would produce a different
// header on every run.
func TestPropertyOrderIsStable(t *testing.T) {
	first, err := ParseJSONSchema([]byte(jsonDoc))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		again, err := ParseJSONSchema([]byte(jsonDoc))
		if err != nil {
			t.Fatal(err)
		}
		if len(again.Order) != len(first.Order) {
			t.Fatalf("column count changed between parses")
		}
		for j := range first.Order {
			if again.Order[j] != first.Order[j] {
				t.Fatalf("column order changed between parses: %v then %v",
					first.Order, again.Order)
			}
		}
	}
}

func TestAvroTypes(t *testing.T) {
	s, err := ParseAvro([]byte(avroDoc))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "User" {
		t.Errorf("name = %q, want User", s.Name)
	}
	for col, kind := range map[string]schema.Kind{
		"age":    schema.KindInt,
		"score":  schema.KindFloat,
		"active": schema.KindBool,
	} {
		f := s.Schema.FieldByName(col)
		if f == nil {
			t.Errorf("no field %q", col)
			continue
		}
		if f.Kind != kind {
			t.Errorf("%s: kind = %q, want %q", col, f.Kind, kind)
		}
	}
}

// ["null", "string"] is how Avro spells an optional field, and it is everywhere
// in real schemas. Failing to unwrap the union would leave the field untyped.
func TestAvroNullableUnion(t *testing.T) {
	s, err := ParseAvro([]byte(avroDoc))
	if err != nil {
		t.Fatal(err)
	}
	f := s.Schema.FieldByName("note")
	if f == nil {
		t.Fatal("no note field")
	}
	if f.Kind == "" || f.Kind == schema.KindUnknown {
		t.Fatalf("note: kind = %q, the union was not unwrapped", f.Kind)
	}
}

// Detection has to work on content, because the two formats arrive in files
// named alike and a JSON Schema read as Avro produces nothing useful.
func TestDetection(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"json schema", jsonDoc},
		{"avro", avroDoc},
	} {
		s, err := Parse([]byte(tc.src))
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if len(s.Schema.Fields) == 0 {
			t.Errorf("%s: no fields", tc.name)
		}
	}
}

func TestMalformedInput(t *testing.T) {
	for _, src := range []string{
		"", "{", "null", "[]", `{"type":"object"}`, `{"type":"record"}`,
		`{"type":"object","properties":"not a map"}`,
		`{"type":"record","fields":"not a list"}`,
		`{"type":"object","properties":{"a":{"$ref":"#"}}}`,
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panicked: %v", src, r)
				}
			}()
			Parse([]byte(src))
		}()
	}
}
