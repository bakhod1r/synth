package synth_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

type Address struct {
	Street   string
	City     string
	Postcode string
}

type LineItem struct {
	Product  string
	Quantity int
	Price    float64 `synth:"amount,min=1,max=1000"`
}

type Customer struct {
	ID       uuid.UUID `synth:"pk"`
	FullName string
	Email    string `synth:"email,from=FullName"`
	Address  Address
	Tags     []string
	Items    []LineItem
}

func TestNestedStruct(t *testing.T) {
	custs := synth.Make[Customer](100, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	for _, c := range custs {
		if c.Address.City == "" || c.Address.Street == "" || c.Address.Postcode == "" {
			t.Fatalf("nested address not filled: %+v", c.Address)
		}
	}
}

func TestNestedSliceOfStructs(t *testing.T) {
	custs := synth.Make[Customer](100, synth.WithSeed(2))
	sawItems := false
	for _, c := range custs {
		if len(c.Items) > 0 {
			sawItems = true
			for _, it := range c.Items {
				if it.Product == "" {
					t.Fatal("line item product empty")
				}
				if it.Price < 1 || it.Price > 1000 {
					t.Fatalf("line item price out of range: %v", it.Price)
				}
			}
		}
	}
	if !sawItems {
		t.Fatal("no line items generated")
	}
}

func TestNestedSliceOfScalars(t *testing.T) {
	custs := synth.Make[Customer](50, synth.WithSeed(3))
	saw := false
	for _, c := range custs {
		if len(c.Tags) > 0 {
			saw = true
		}
	}
	if !saw {
		t.Fatal("no string tags generated")
	}
}

// Nested output must marshal to valid JSON (API/DB payload use case).
func TestNestedJSON(t *testing.T) {
	custs := synth.Make[Customer](5, synth.WithSeed(4))
	b, err := json.Marshal(custs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Address") || !strings.Contains(string(b), "Items") {
		t.Fatalf("nested JSON missing fields: %s", b)
	}
}
