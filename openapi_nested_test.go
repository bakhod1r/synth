package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
)

const nestedSpec = `
openapi: 3.0.0
paths:
  /orders:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Order'
components:
  schemas:
    Order:
      type: object
      properties:
        id: {type: integer, minimum: 1, maximum: 9}
        customer:
          $ref: '#/components/schemas/Customer'
        items:
          type: array
          minItems: 2
          maxItems: 4
          items:
            $ref: '#/components/schemas/Item'
        tags:
          type: array
          items: {type: string, enum: [a, b]}
    Customer:
      type: object
      properties:
        email: {type: string, format: email}
        referrer:
          $ref: '#/components/schemas/Customer'
    Item:
      type: object
      properties:
        sku: {type: string}
        qty: {type: integer, minimum: 1, maximum: 5}
`

// Request bodies nest: an object property must be an object, an array
// property an array of the declared items, and a self-referencing schema must
// terminate.
func TestOpenAPINestedPayload(t *testing.T) {
	api, err := synth.OpenAPIBytes([]byte(nestedSpec))
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Payloads("POST", "/orders", 10, synth.WithSeed(4))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range got {
		cust, ok := p["customer"].(map[string]any)
		if !ok {
			t.Fatalf("customer = %T %v, want an object", p["customer"], p["customer"])
		}
		if _, ok := cust["email"].(string); !ok {
			t.Fatalf("customer.email = %v", cust["email"])
		}
		items, ok := p["items"].([]any)
		if !ok || len(items) < 2 || len(items) > 4 {
			t.Fatalf("items = %T %v, want 2..4 objects", p["items"], p["items"])
		}
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				t.Fatalf("item = %T, want object", it)
			}
			if _, ok := m["sku"]; !ok {
				t.Fatalf("item missing sku: %v", m)
			}
		}
		tags, ok := p["tags"].([]any)
		if !ok {
			t.Fatalf("tags = %T, want array", p["tags"])
		}
		for _, tg := range tags {
			if tg != "a" && tg != "b" {
				t.Fatalf("tag %v not in enum", tg)
			}
		}
	}
}
