package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

const protoSrc = `
syntax = "proto3";
package shop;

// Status of an order.
enum Status {
  PENDING = 0;
  PAID = 1;
  SHIPPED = 2;
}

message Address {
  string city = 1;
  string postcode = 2;
}

message Customer {
  string id = 1;          // uuid by name inference
  string email = 2;
  string phone = 3;
  int32 age = 4;
  double balance = 5;
  bool active = 6;
  Status status = 7;
  Address address = 8;
  repeated string tags = 9;
}
`

func TestProtoFrontend(t *testing.T) {
	msgs, err := synth.ProtoBytes([]byte(protoSrc))
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages, got %d", len(msgs))
	}
	var customer *synth.ProtoMessage
	for _, m := range msgs {
		if m.Name() == "Customer" {
			customer = m
		}
	}
	if customer == nil {
		t.Fatal("Customer message not parsed")
	}
	if got := customer.Columns(); len(got) != 9 || got[0] != "id" {
		t.Fatalf("columns %v", got)
	}

	rows, err := customer.Generate(200, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]bool{"PENDING": true, "PAID": true, "SHIPPED": true}
	for _, r := range rows {
		if !strings.Contains(r["email"].(string), "@") {
			t.Fatalf("email %v", r["email"])
		}
		if !strings.HasPrefix(r["phone"].(string), "+998") {
			t.Fatalf("phone %v", r["phone"])
		}
		if _, ok := r["age"].(int); !ok {
			t.Fatalf("age not int: %T", r["age"])
		}
		if _, ok := r["balance"].(float64); !ok {
			t.Fatalf("balance not float: %T", r["balance"])
		}
		if _, ok := r["active"].(bool); !ok {
			t.Fatalf("active not bool: %T", r["active"])
		}
		// enum resolved to its declared values
		if !statuses[r["status"].(string)] {
			t.Fatalf("status %v not a declared enum value", r["status"])
		}
		// nested message became an object with coherent city/postcode
		addr, ok := r["address"].(map[string]any)
		if !ok || addr["city"] == "" || addr["postcode"] == "" {
			t.Fatalf("address not generated: %v", r["address"])
		}
		// repeated field became an array
		tags, ok := r["tags"].([]any)
		if !ok || len(tags) == 0 {
			t.Fatalf("tags not an array: %v", r["tags"])
		}
	}
}
