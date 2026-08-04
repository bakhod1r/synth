package locale

import "testing"

// Has is the validating counterpart to Get: Get silently falls back to en_US,
// so a caller that must reject a typo in a spec has only this to go on.
func TestHas(t *testing.T) {
	for _, name := range Names() {
		if !Has(name) {
			t.Errorf("Has(%q) = false, want true for a registered locale", name)
		}
	}
	for _, name := range []string{"", "en-US", "xx_XX", "EN_US"} {
		if Has(name) {
			t.Errorf("Has(%q) = true, want false for an unregistered name", name)
		}
	}
}
