package webspec

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

// IsLocalized answers /api/types for both the server and the WebAssembly
// build, so the palette's dot must mean the same thing in either.
func TestIsLocalized(t *testing.T) {
	var structural, plain schema.Kind
	for k := range structurallyLocalized {
		structural = k
		break
	}
	if structural == "" {
		t.Fatal("structurallyLocalized is empty; the test has nothing to check")
	}
	if !IsLocalized(structural, nil) {
		t.Errorf("IsLocalized(%q, nil) = false, want true: a structurally localized kind needs no word list", structural)
	}
	plain = schema.KindBool
	if structurallyLocalized[plain] {
		t.Fatalf("%q is structurally localized; pick another kind for the negative case", plain)
	}
	if IsLocalized(plain, nil) {
		t.Errorf("IsLocalized(%q, nil) = true, want false: no dataset means no coverage to claim", plain)
	}
	if !IsLocalized(plain, []string{"uz_UZ"}) {
		t.Errorf("IsLocalized(%q, [uz_UZ]) = false, want true: a real dataset exists", plain)
	}
}
