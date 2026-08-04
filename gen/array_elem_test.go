package gen

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// An array field with no element type used to nil-dereference on the first
// generated record. It must fail at Compile, where the mistake is.
func TestCompileRejectsArrayWithoutElement(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "tags", Kind: schema.KindArray, Params: map[string]string{}},
	}}
	_, err := Compile(s, "en_US")
	if err == nil || !strings.Contains(err.Error(), "no element type") {
		t.Fatalf("err = %v, want an array-without-element error", err)
	}
}

func TestCompileAcceptsArrayWithElement(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{{
		Name:   "tags",
		Kind:   schema.KindArray,
		Params: map[string]string{},
		Elem:   &schema.Field{Name: "tags", Kind: schema.KindWord, Params: map[string]string{}},
		ArrMin: 2, ArrMax: 2,
	}}}
	eng, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	rec := eng.Record(rng.New(1), 0)
	vals, ok := rec["tags"].([]any)
	if !ok || len(vals) != 2 {
		t.Fatalf("tags = %#v, want 2 generated elements", rec["tags"])
	}
	for _, v := range vals {
		if s, _ := v.(string); s == "" {
			t.Fatalf("empty element in %v", vals)
		}
	}
}
