package gen

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

// A tracked unique keys a map by the value; an array value is not hashable
// and panicked on the first record. Compile refuses it instead.
func TestCompileRefusesUniqueComposite(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{{
		Name: "Tags", Kind: schema.KindArray, Unique: true,
		Elem: &schema.Field{Name: "Tags", Kind: schema.KindLorem},
	}}}
	if _, err := Compile(s, "en_US"); err == nil {
		t.Fatal("unique on an array field compiled")
	}
}
