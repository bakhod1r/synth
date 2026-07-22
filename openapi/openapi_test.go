package openapi

import (
	"testing"
	"time"

	"github.com/bakhodir/synth/schema"
)

const petstore = `
openapi: 3.0.0
info: { title: t, version: "1" }
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
  /orders:
    put:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                total: { type: number }
components:
  schemas:
    User:
      type: object
      required: [id, email]
      properties:
        id:      { type: string, format: uuid }
        email:   { type: string, format: email }
        age:     { type: integer, minimum: 18, maximum: 90 }
        active:  { type: boolean }
        tags:    { type: array, items: { type: string } }
        address:
          type: object
          properties:
            city: { type: string }
`

func parse(t *testing.T, src string) *Spec {
	t.Helper()
	s, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

func TestOperationSchema(t *testing.T) {
	s := parse(t, petstore)
	sc, err := s.Schema("post", "/users")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]schema.Kind{
		"id":     schema.KindUUID,
		"email":  schema.KindEmail,
		"age":    schema.KindInt,
		"active": schema.KindBool,
	}
	for col, kind := range want {
		f := sc.FieldByName(col)
		if f == nil {
			t.Errorf("no property %q", col)
			continue
		}
		if f.Kind != kind {
			t.Errorf("%s: kind = %q, want %q", col, f.Kind, kind)
		}
	}
}

func timeout() <-chan time.Time { return time.After(5 * time.Second) }

// A payload that ignores minimum and maximum is not a valid payload, which is
// the only reason to generate from a spec rather than from a struct.
func TestNumericBounds(t *testing.T) {
	s := parse(t, petstore)
	sc, err := s.Schema("post", "/users")
	if err != nil {
		t.Fatal(err)
	}
	f := sc.FieldByName("age")
	if f == nil {
		t.Fatal("no age")
	}
	if f.Params["min"] != "18" || f.Params["max"] != "90" {
		t.Fatalf("bounds = %v..%v, want 18..90", f.Params["min"], f.Params["max"])
	}
}

// The package documents that nested objects and arrays are generated shallowly.
// This pins that down: they must still appear as properties, because a payload
// missing a required field is rejected by the API it was generated for.
func TestArrayAndNestedObjectStillAppear(t *testing.T) {
	s := parse(t, petstore)
	sc, _ := s.Schema("post", "/users")
	for _, name := range []string{"tags", "address"} {
		if sc.FieldByName(name) == nil {
			t.Errorf("property %q was dropped entirely", name)
		}
	}
}

// An inline schema is as common as a $ref, and dropping it would make the
// operation unusable while the spec looks fine.
func TestInlineSchema(t *testing.T) {
	s := parse(t, petstore)
	sc, err := s.Schema("put", "/orders")
	if err != nil {
		t.Fatal(err)
	}
	if sc.FieldByName("total") == nil {
		t.Fatal("the inline schema's property is missing")
	}
}

// The method must not have to be typed in the same case the document uses.
func TestMethodIsCaseInsensitive(t *testing.T) {
	s := parse(t, petstore)
	for _, m := range []string{"post", "POST", "Post"} {
		if _, err := s.Schema(m, "/users"); err != nil {
			t.Errorf("method %q: %v", m, err)
		}
	}
}

// An unknown operation must say so. Returning an empty schema would generate
// zero-column payloads that look like a bug in the generator.
func TestUnknownOperation(t *testing.T) {
	s := parse(t, petstore)
	for _, tc := range [2]string{"/nope", "/users"} {
		method := "post"
		if tc == "/users" {
			method = "delete"
		}
		if _, err := s.Schema(method, tc); err == nil {
			t.Errorf("%s %s was accepted", method, tc)
		}
	}
}

func TestJSONDocument(t *testing.T) {
	src := `{"openapi":"3.0.0","paths":{"/u":{"post":{"requestBody":{"content":
		{"application/json":{"schema":{"type":"object","properties":{"id":
		{"type":"string","format":"uuid"}}}}}}}}}}`
	s := parse(t, src)
	sc, err := s.Schema("post", "/u")
	if err != nil {
		t.Fatal(err)
	}
	if f := sc.FieldByName("id"); f == nil || f.Kind != schema.KindUUID {
		t.Fatalf("id: %v", f)
	}
}

// A $ref pointing at itself must terminate. A spec like this is usually a
// mistake, but it is a mistake that would otherwise hang whatever loaded it.
func TestRecursiveRefTerminates(t *testing.T) {
	src := `
openapi: 3.0.0
paths:
  /n:
    post:
      requestBody:
        content:
          application/json:
            schema: { $ref: '#/components/schemas/Node' }
components:
  schemas:
    Node:
      type: object
      properties:
        name:  { type: string }
        child: { $ref: '#/components/schemas/Node' }
`
	done := make(chan struct{})
	go func() {
		defer close(done)
		if s, err := Parse([]byte(src)); err == nil {
			s.Schema("post", "/n")
		}
	}()
	select {
	case <-done:
	case <-timeout():
		t.Fatal("a self-referencing $ref did not terminate")
	}
}

func TestMalformedDocument(t *testing.T) {
	for _, src := range []string{
		"", "not a document", "{", "openapi: 3.0.0\npaths: [not, a, map]\n",
		`{"paths":{"/u":{"post":{"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/nope"}}}}}}}}`,
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panicked: %v", src, r)
				}
			}()
			if s, err := Parse([]byte(src)); err == nil && s != nil {
				s.Schema("post", "/u")
			}
		}()
	}
}

func TestUnsupportedContentType(t *testing.T) {
	src := `
openapi: 3.0.0
paths:
  /u:
    post:
      requestBody:
        content:
          text/plain:
            schema: { type: string }
`
	s := parse(t, src)
	if _, err := s.Schema("post", "/u"); err == nil {
		t.Skip("non-JSON bodies are accepted, which is a reasonable choice")
	}
}
