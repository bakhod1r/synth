package synth

import (
	"reflect"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

type namedString string

// A provider's value and the struct field it lands in do not have to agree on
// type: a float provider can feed an int column, an int one a float column, and
// a non-string value can land in a string field. assign coerces rather than
// leaving the field at its zero value, because a silently empty column is the
// failure mode that is hardest to notice.
func TestAssignCoercesAcrossNumericKinds(t *testing.T) {
	tests := []struct {
		name string
		dst  any // pointer to the destination
		val  any
		want any
	}{
		{"float into int", new(int64), 3.9, int64(3)},
		{"int into int", new(int64), 5, int64(5)},
		{"int into float", new(float64), 7, float64(7)},
		{"float into float", new(float64), float32(1.5), float64(1.5)},
		{"int into uint", new(uint32), 5, uint32(5)},
		// A named string type is not assignable from string, so it takes the
		// coercion path a plain string field would skip.
		{"string into named string", new(namedString), "already", namedString("already")},
		{"non-string into string", new(namedString), 42, namedString("42")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fv := reflect.ValueOf(tc.dst).Elem()
			assign(fv, tc.val)
			if got := fv.Interface(); got != tc.want {
				t.Errorf("assign(%T, %v) = %v, want %v", tc.dst, tc.val, got, tc.want)
			}
		})
	}
}

// Unmasked has to leave every other parameter alone. The Params map is shared
// with the cached spec, so stripping the mask in place would unmask every later
// call as well — the copy is what keeps one unmasked call from leaking.
func TestStripMasksKeepsOtherParamsAndDoesNotShare(t *testing.T) {
	shared := map[string]string{"mask": "redact", "min": "1"}
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "card", Params: shared},
		{Name: "plain", Params: map[string]string{"min": "2"}},
	}}
	stripMasks(s)

	if _, ok := s.Fields[0].Params["mask"]; ok {
		t.Error("mask= survived stripMasks")
	}
	if got := s.Fields[0].Params["min"]; got != "1" {
		t.Errorf("min = %q, want %q: stripping a mask must not drop other params", got, "1")
	}
	if _, ok := shared["mask"]; !ok {
		t.Error("stripMasks edited the shared Params map in place; a later masked call would come back unmasked")
	}
	if s.Fields[1].Params["min"] != "2" {
		t.Error("a field with no mask= was rewritten")
	}
}
