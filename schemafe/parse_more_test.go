package schemafe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func fieldNamed(t *testing.T, tbl *Table, name string) schema.Field {
	t.Helper()
	for _, f := range tbl.Schema.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("field %q is not in the parsed schema", name)
	return schema.Field{}
}

// JSON's object order is lost by map decoding, so declaration order is read
// back from the raw document. A CSV written from this schema has to have the
// columns the author wrote.
func TestJSONSchemaKeepsDeclarationOrder(t *testing.T) {
	doc := []byte(`{
	  "title": "Account",
	  "type": "object",
	  "properties": {
	    "zulu":  {"type": "string"},
	    "alpha": {"type": "integer"},
	    "mike":  {"type": "boolean"},
	    "bravo": {"type": "number"}
	  }
	}`)
	tbl, err := ParseJSONSchema(doc)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "Account" {
		t.Fatalf("name = %q, want the title", tbl.Name)
	}
	want := "zulu,alpha,mike,bravo"
	if got := strings.Join(tbl.Order, ","); got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestJSONSchemaTypesAndConstraints(t *testing.T) {
	doc := []byte(`{
	  "type": "object",
	  "properties": {
	    "email":   {"type": "string", "format": "email"},
	    "website": {"type": "string", "format": "uri"},
	    "when":    {"type": "string", "format": "date-time"},
	    "uid":     {"type": "string", "format": "uuid"},
	    "age":     {"type": "integer", "minimum": 18, "maximum": 65},
	    "score":   {"type": "number", "minimum": 0, "maximum": 1},
	    "active":  {"type": "boolean"},
	    "tier":    {"type": "string", "enum": ["gold", "silver", "bronze"]},
	    "short":   {"type": "string", "maxLength": 12}
	  }
	}`)
	tbl, err := ParseJSONSchema(doc)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "data" {
		t.Fatalf("a title-less schema should be named data, got %q", tbl.Name)
	}

	if k := fieldNamed(t, tbl, "email").Kind; k != schema.KindEmail {
		t.Errorf("email kind = %q", k)
	}
	if k := fieldNamed(t, tbl, "uid").Kind; k != schema.KindUUID {
		t.Errorf("uuid kind = %q", k)
	}
	if k := fieldNamed(t, tbl, "when").Kind; k != schema.KindTime {
		t.Errorf("date-time kind = %q", k)
	}
	if k := fieldNamed(t, tbl, "active").Kind; k != schema.KindBool {
		t.Errorf("boolean kind = %q", k)
	}

	age := fieldNamed(t, tbl, "age")
	if age.Params["min"] != "18" || age.Params["max"] != "65" {
		t.Errorf("age bounds = %v", age.Params)
	}
	tier := fieldNamed(t, tbl, "tier")
	if tier.Kind != schema.KindEnum || len(tier.Choices) != 3 {
		t.Errorf("enum not carried over: %+v", tier)
	}
	if got := fieldNamed(t, tbl, "short").Params["maxlen"]; got != "12" {
		t.Errorf("maxLength = %q, want 12", got)
	}
}

func TestJSONSchemaErrors(t *testing.T) {
	if _, err := ParseJSONSchema([]byte("{")); err == nil {
		t.Error("truncated JSON should error")
	}
	if _, err := ParseJSONSchema([]byte(`{"type":"object","properties":{}}`)); err == nil {
		t.Error("a schema with no properties should error")
	}
}

func TestAvroUnionsAndLogicalTypes(t *testing.T) {
	doc := []byte(`{
	  "type": "record",
	  "name": "Payment",
	  "fields": [
	    {"name": "id",        "type": {"type": "string", "logicalType": "uuid"}},
	    {"name": "note",      "type": ["null", "string"]},
	    {"name": "amount",    "type": {"type": "bytes", "logicalType": "decimal"}},
	    {"name": "created",   "type": {"type": "long", "logicalType": "timestamp-millis"}},
	    {"name": "day",       "type": {"type": "int", "logicalType": "date"}},
	    {"name": "count",     "type": "int"},
	    {"name": "ratio",     "type": "double"},
	    {"name": "ok",        "type": "boolean"},
	    {"name": "wrapped",   "type": {"type": "string"}}
	  ]
	}`)
	tbl, err := ParseAvro(doc)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "Payment" {
		t.Fatalf("name = %q", tbl.Name)
	}
	if got := strings.Join(tbl.Order, ","); got != "id,note,amount,created,day,count,ratio,ok,wrapped" {
		t.Fatalf("order = %q", got)
	}
	checks := map[string]schema.Kind{
		"id":      schema.KindUUID,
		"amount":  schema.KindFloat,
		"created": schema.KindTime,
		"day":     schema.KindTime,
		"count":   schema.KindInt,
		"ratio":   schema.KindFloat,
		"ok":      schema.KindBool,
	}
	for name, want := range checks {
		if got := fieldNamed(t, tbl, name).Kind; got != want {
			t.Errorf("%s kind = %q, want %q", name, got, want)
		}
	}
	// A ["null", X] union takes X: the nullability is a column property, not a
	// type of its own.
	if k := fieldNamed(t, tbl, "note").Kind; k == schema.KindUnknown {
		t.Error("a nullable union resolved to nothing")
	}
}

func TestAvroErrors(t *testing.T) {
	if _, err := ParseAvro([]byte("not json")); err == nil {
		t.Error("invalid JSON should error")
	}
	if _, err := ParseAvro([]byte(`{"type":"enum","name":"x"}`)); err == nil {
		t.Error("a non-record schema should error")
	}
	if _, err := ParseAvro([]byte(`{"type":"record","name":"x","fields":[]}`)); err == nil {
		t.Error("a record with no fields should error")
	}
}

func TestAvroNamelessRecordFallsBackToData(t *testing.T) {
	tbl, err := ParseAvro([]byte(`{"type":"record","fields":[{"name":"a","type":"string"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "data" {
		t.Fatalf("name = %q, want data", tbl.Name)
	}
}

// Load picks the dialect from the content, not the file extension: both are
// JSON, and the extension is whatever the exporter chose.
func TestLoadDetectsDialect(t *testing.T) {
	dir := t.TempDir()

	js := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(js, []byte(`{"title":"J","type":"object","properties":{"a":{"type":"string"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tbl, err := Load(js)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "J" {
		t.Fatalf("JSON Schema not detected: %+v", tbl)
	}

	av := filepath.Join(dir, "record.json")
	if err := os.WriteFile(av, []byte(`{"type":"record","name":"A","fields":[{"name":"a","type":"string"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tbl, err = Load(av)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Name != "A" {
		t.Fatalf("Avro not detected: %+v", tbl)
	}

	if _, err := Load(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("a missing file should error")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"type":"something-else"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad); err == nil {
		t.Error("a document that is neither dialect should error")
	}
}
