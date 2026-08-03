package protofe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

const sample = `
syntax = "proto3";

// a comment
enum Status {
  ACTIVE = 0;
  BANNED = 1;
}

message Address {
  string city = 1;
  string zip = 2; // inferred postcode
}

message User {
  string email = 1;
  int64 age = 2;
  double score = 3;
  bool active = 4;
  bytes blob = 5;
  google.protobuf.Timestamp created = 6;
  Status status = 7;
  Address home = 8;
  repeated string tags = 9;
  map<string, string> meta = 10;
  Unresolvable ghost = 11;
  User manager = 12;
}
`

func TestParse(t *testing.T) {
	msgs, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	var user *Message
	for _, m := range msgs {
		if m.Name == "User" {
			user = m
		}
	}
	if user == nil {
		t.Fatal("User message missing")
	}
	by := func(name string) *schema.Field { return user.Schema.FieldByName(name) }

	if by("email").Kind != schema.KindEmail {
		t.Fatal("email not inferred from name")
	}
	if by("age").Kind != schema.KindInt {
		t.Fatal("age not int")
	}
	if by("score").Kind != schema.KindFloat {
		t.Fatal("score not float")
	}
	if by("active").Kind != schema.KindBool {
		t.Fatal("active not bool")
	}
	if by("blob").Kind != schema.KindLorem { // bytes, no name match
		t.Fatal("blob not lorem")
	}
	if by("created").Kind != schema.KindTime {
		t.Fatal("created not time")
	}
	if f := by("status"); f.Kind != schema.KindEnum || len(f.Choices) != 2 {
		t.Fatalf("status enum wrong: %+v", f)
	}
	if f := by("home"); f.Kind != schema.KindObject || f.Nested == nil {
		t.Fatal("home not nested object")
	}
	if f := by("tags"); f.Kind != schema.KindArray || f.Elem == nil {
		t.Fatal("tags not array")
	}
	if by("meta") != nil {
		t.Fatal("map field should be skipped")
	}
	if by("ghost") != nil {
		t.Fatal("unresolvable type should be skipped")
	}
	if by("manager") != nil {
		t.Fatal("self-recursive message field should be skipped")
	}
}

func TestBuildMessageUnknown(t *testing.T) {
	// Direct call with a name absent from bodies hits the guard.
	if _, err := buildMessage("Nope", map[string]string{}, nil, map[string]bool{}); err == nil {
		t.Fatal("expected error for unknown message")
	}
}

func TestParseNoMessage(t *testing.T) {
	if _, err := Parse("syntax = \"proto3\";\n"); err == nil {
		t.Fatal("expected error for no message")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.proto")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(dir, "nope.proto")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
