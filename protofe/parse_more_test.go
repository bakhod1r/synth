package protofe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

func msgNamed(t *testing.T, ms []*Message, name string) *Message {
	t.Helper()
	for _, m := range ms {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("message %q was not parsed", name)
	return nil
}

func fieldNamed(t *testing.T, m *Message, name string) schema.Field {
	t.Helper()
	for _, f := range m.Schema.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("field %q is not in message %q", name, m.Name)
	return schema.Field{}
}

const shopProto = `
syntax = "proto3";

// A comment mentioning message Ghost { } must not create a message.
enum Status {
  PENDING = 0;
  PAID = 1;
  REFUNDED = 2;
}

message Address {
  string city = 1;
  string postcode = 2;
}

message Order {
  string id = 1;
  string email = 2;
  int32 quantity = 3;
  int64 total_cents = 4;
  double rate = 5;
  bool is_gift = 6;
  bytes payload = 7;
  Status status = 8;
  Address shipping = 9;
  repeated string tags = 10;
  repeated Address history = 11;
  map<string, string> labels = 12;
  Unknown mystery = 13;
}
`

func TestParseResolvesEveryFieldShape(t *testing.T) {
	ms, err := Parse(shopProto)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("parsed %d messages, want 2 (a comment created one?)", len(ms))
	}
	order := msgNamed(t, ms, "Order")

	if k := fieldNamed(t, order, "quantity").Kind; k != schema.KindInt {
		t.Errorf("int32 kind = %q", k)
	}
	if k := fieldNamed(t, order, "rate").Kind; k != schema.KindFloat {
		t.Errorf("double kind = %q", k)
	}
	if k := fieldNamed(t, order, "is_gift").Kind; k != schema.KindBool {
		t.Errorf("bool kind = %q", k)
	}
	if k := fieldNamed(t, order, "email").Kind; k != schema.KindEmail {
		t.Errorf("a field named email should infer an address, got %q", k)
	}

	status := fieldNamed(t, order, "status")
	if status.Kind != schema.KindEnum || strings.Join(status.Choices, ",") != "PENDING,PAID,REFUNDED" {
		t.Errorf("enum = %+v", status)
	}

	shipping := fieldNamed(t, order, "shipping")
	if shipping.Kind != schema.KindObject || shipping.Nested == nil || len(shipping.Nested.Fields) != 2 {
		t.Errorf("nested message not resolved: %+v", shipping)
	}

	tags := fieldNamed(t, order, "tags")
	if tags.Kind != schema.KindArray || tags.Elem == nil {
		t.Fatalf("repeated scalar = %+v", tags)
	}
	history := fieldNamed(t, order, "history")
	if history.Kind != schema.KindArray || history.Elem == nil || history.Elem.Kind != schema.KindObject {
		t.Fatalf("repeated message = %+v", history)
	}

	// Maps are not modeled, and a type defined nowhere cannot be resolved:
	// both are skipped rather than guessed at.
	for _, skipped := range []string{"labels", "mystery"} {
		for _, name := range order.Order {
			if name == skipped {
				t.Errorf("field %q should have been skipped", skipped)
			}
		}
	}
}

// A message referencing itself must not recurse forever.
func TestParseStopsAtRecursiveMessages(t *testing.T) {
	ms, err := Parse(`
message Node {
  string id = 1;
  Node parent = 2;
  repeated Node children = 3;
}
`)
	if err != nil {
		t.Fatal(err)
	}
	node := msgNamed(t, ms, "Node")
	if strings.Join(node.Order, ",") != "id" {
		t.Fatalf("order = %v, want the self-references dropped", node.Order)
	}
}

// Mutual recursion: A holds a B, B holds an A. One level resolves, then stops.
func TestParseHandlesMutualRecursion(t *testing.T) {
	ms, err := Parse(`
message A {
  string a_id = 1;
  B b = 2;
}
message B {
  string b_id = 1;
  A a = 2;
}
`)
	if err != nil {
		t.Fatal(err)
	}
	a := msgNamed(t, ms, "A")
	b := fieldNamed(t, a, "b")
	if b.Kind != schema.KindObject || b.Nested == nil {
		t.Fatalf("B not resolved inside A: %+v", b)
	}
	// Inside that B, the back-reference to A is dropped rather than expanded.
	for _, f := range b.Nested.Fields {
		if f.Name == "a" {
			t.Fatal("the mutual back-reference was expanded")
		}
	}
}

func TestParseLabelsAndProto2(t *testing.T) {
	ms, err := Parse(`
syntax = "proto2";
message Legacy {
  required string name = 1;
  optional int32 age = 2;
  repeated string alias = 3;
}
`)
	if err != nil {
		t.Fatal(err)
	}
	m := msgNamed(t, ms, "Legacy")
	if strings.Join(m.Order, ",") != "name,age,alias" {
		t.Fatalf("order = %v", m.Order)
	}
	if fieldNamed(t, m, "alias").Kind != schema.KindArray {
		t.Fatal("repeated not honored under proto2")
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse(`syntax = "proto3";`); err == nil {
		t.Error("a file with no message should error")
	}
	if _, err := Parse(""); err == nil {
		t.Error("empty source should error")
	}
}

func TestLoadReadsAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shop.proto")
	if err := os.WriteFile(path, []byte(shopProto), 0o644); err != nil {
		t.Fatal(err)
	}
	ms, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("parsed %d messages, want 2", len(ms))
	}
	if _, err := Load(filepath.Join(dir, "missing.proto")); err == nil {
		t.Error("a missing file should error")
	}
}

// The parsed schema has to be generatable — a frontend that parses but does not
// compile is no use.
func TestParsedSchemaGenerates(t *testing.T) {
	ms, err := Parse(shopProto)
	if err != nil {
		t.Fatal(err)
	}
	order := msgNamed(t, ms, "Order")
	eng, err := gen.Compile(order.Schema, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	base := rng.New(1)
	for i := 0; i < 20; i++ {
		rec := eng.Record(base, i)
		for _, name := range order.Order {
			if _, ok := rec[name]; !ok {
				t.Fatalf("record %d is missing field %q", i, name)
			}
		}
		if _, ok := rec["shipping"].(map[string]any); !ok {
			t.Fatalf("nested message did not generate an object: %#v", rec["shipping"])
		}
		tags, ok := rec["tags"].([]any)
		if !ok || len(tags) < 1 || len(tags) > 3 {
			t.Fatalf("repeated field = %#v", rec["tags"])
		}
	}
}
