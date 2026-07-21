package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

const jsonSchemaDoc = `{
  "title": "Customer",
  "type": "object",
  "required": ["id", "email"],
  "properties": {
    "id":         {"type": "string", "format": "uuid"},
    "email":      {"type": "string", "format": "email"},
    "website":    {"type": "string", "format": "uri"},
    "age":        {"type": "integer", "minimum": 18, "maximum": 65},
    "score":      {"type": "number", "minimum": 0, "maximum": 100},
    "active":     {"type": "boolean"},
    "tier":       {"type": "string", "enum": ["free", "pro", "enterprise"]},
    "created_at": {"type": "string", "format": "date-time"}
  }
}`

func TestJSONSchemaFrontend(t *testing.T) {
	s, err := synth.JSONSchemaBytes([]byte(jsonSchemaDoc))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "Customer" {
		t.Fatalf("name %q", s.Name())
	}
	if got := s.Columns(); len(got) != 8 || got[0] != "id" {
		t.Fatalf("columns %v", got)
	}
	rows, err := s.Generate(300, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	tiers := map[string]bool{"free": true, "pro": true, "enterprise": true}
	for _, r := range rows {
		if !strings.Contains(r["email"].(string), "@") {
			t.Fatalf("email %v", r["email"])
		}
		if !strings.HasPrefix(r["website"].(string), "http") {
			t.Fatalf("website %v", r["website"])
		}
		age := r["age"].(int)
		if age < 18 || age > 65 {
			t.Fatalf("age %d outside 18..65", age)
		}
		if !tiers[r["tier"].(string)] {
			t.Fatalf("tier %v not in enum", r["tier"])
		}
		if _, ok := r["active"].(bool); !ok {
			t.Fatalf("active not bool: %T", r["active"])
		}
	}
}

const avroDoc = `{
  "type": "record",
  "name": "Payment",
  "fields": [
    {"name": "id",       "type": {"type": "string", "logicalType": "uuid"}},
    {"name": "amount",   "type": "double"},
    {"name": "attempts", "type": "int"},
    {"name": "email",    "type": ["null", "string"]},
    {"name": "settled",  "type": "boolean"},
    {"name": "created",  "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}`

func TestAvroFrontend(t *testing.T) {
	s, err := synth.AvroBytes([]byte(avroDoc))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "Payment" {
		t.Fatalf("name %q", s.Name())
	}
	if got := s.Columns(); len(got) != 6 || got[1] != "amount" {
		t.Fatalf("columns %v", got)
	}
	rows, err := s.Generate(200, synth.WithSeed(2))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if _, ok := r["amount"].(float64); !ok {
			t.Fatalf("amount not float: %T", r["amount"])
		}
		if _, ok := r["attempts"].(int); !ok {
			t.Fatalf("attempts not int: %T", r["attempts"])
		}
		if _, ok := r["settled"].(bool); !ok {
			t.Fatalf("settled not bool: %T", r["settled"])
		}
		// union ["null","string"] on a field named email -> email values
		if !strings.Contains(r["email"].(string), "@") {
			t.Fatalf("email %v", r["email"])
		}
	}
}
