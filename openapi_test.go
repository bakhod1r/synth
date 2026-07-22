package synth_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

const spec = `
openapi: 3.0.0
paths:
  /payments:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Payment'
components:
  schemas:
    Payment:
      type: object
      required: [amount, currency]
      properties:
        amount:
          type: integer
          minimum: 100
          maximum: 5000
        currency:
          type: string
          enum: [USD, EUR, UZS]
        customer_email:
          type: string
          format: email
        callback_url:
          type: string
          format: uri
`

func TestOpenAPIPayload(t *testing.T) {
	api, err := synth.OpenAPIBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	payloads, err := api.Payloads("POST", "/payments", 100, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	valcur := map[string]bool{"USD": true, "EUR": true, "UZS": true}
	for _, p := range payloads {
		amt, ok := p["amount"].(int)
		if !ok || amt < 100 || amt > 5000 {
			t.Fatalf("amount invalid: %v", p["amount"])
		}
		if !valcur[p["currency"].(string)] {
			t.Fatalf("currency not in enum: %v", p["currency"])
		}
		if !strings.Contains(p["customer_email"].(string), "@") {
			t.Fatalf("email invalid: %v", p["customer_email"])
		}
		if !strings.HasPrefix(p["callback_url"].(string), "http") {
			t.Fatalf("url invalid: %v", p["callback_url"])
		}
	}
}

func TestOpenAPIPayloadJSON(t *testing.T) {
	api, _ := synth.OpenAPIBytes([]byte(spec))
	b, err := api.PayloadJSON("POST", "/payments", synth.WithSeed(2))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := m["amount"]; !ok {
		t.Fatal("missing amount in JSON payload")
	}
}

func TestOpenAPIUnknownPath(t *testing.T) {
	api, _ := synth.OpenAPIBytes([]byte(spec))
	if _, err := api.Payload("POST", "/nope"); err == nil {
		t.Fatal("expected error for unknown path")
	}
}
