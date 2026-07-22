package reflectfe

import (
	"reflect"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

type cached struct {
	ID    int `synth:"pk"`
	Email string
	City  string
}

// Building the same type twice must return the identical cached schema pointer,
// proving reflection runs once per type rather than on every call.
func TestSchemaCachedPerType(t *testing.T) {
	rt := reflect.TypeOf(cached{})
	s1, w1 := Build(rt)
	s2, w2 := Build(rt)
	if s1 != s2 {
		t.Fatal("schema not cached: Build returned different pointers")
	}
	if len(w1) != len(w2) {
		t.Fatal("warnings not cached consistently")
	}
}

// Tags win over name inference; untagged fields fall back to inference.
func TestTagOverridesInference(t *testing.T) {
	type rec struct {
		City  string `synth:"country"` // tag overrides the "city" synonym
		Email string // inferred from the name
	}
	s, _ := Build(reflect.TypeOf(rec{}))
	if got := s.FieldByName("City").Kind; got != schema.KindCountry {
		t.Fatalf("tag ignored: City kind = %q, want %q", got, schema.KindCountry)
	}
	if got := s.FieldByName("Email").Kind; got != schema.KindEmail {
		t.Fatalf("inference failed: Email kind = %q", got)
	}
}

// Uninferable fields are reported as warnings rather than failing silently.
func TestWarningsForUninferableFields(t *testing.T) {
	type rec struct {
		Name string
		Junk chan int
	}
	_, warns := Build(reflect.TypeOf(rec{}))
	if len(warns) != 1 || warns[0].Field != "Junk" {
		t.Fatalf("expected one warning for Junk, got %+v", warns)
	}
}
