package providers

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

// A passphrase bank is often the first dataset a locale has, so registration
// creates the catalog entry rather than assuming one — and it must not wipe the
// datasets of a locale that already has some.
func TestRegisterPassphraseBanks(t *testing.T) {
	t.Run("creates the catalog for a locale with no datasets", func(t *testing.T) {
		const code = "zz_NEW"
		t.Cleanup(func() { delete(localeCatalog, code) })

		registerPassphraseBanks(map[string][]string{code: {"olma", "anor"}})
		if got := localeCatalog[code][schema.KindPassphrase]; len(got) != 2 {
			t.Fatalf("words = %v, want the bank filed under a freshly created catalog", got)
		}
	})

	t.Run("keeps the other datasets of a locale that has some", func(t *testing.T) {
		const code = "zz_EXISTING"
		localeCatalog[code] = map[schema.Kind][]string{schema.KindColor: {"qizil"}}
		t.Cleanup(func() { delete(localeCatalog, code) })

		registerPassphraseBanks(map[string][]string{code: {"bahor"}})
		if got := localeCatalog[code][schema.KindColor]; len(got) != 1 {
			t.Errorf("colors = %v, want the existing dataset untouched", got)
		}
		if got := localeCatalog[code][schema.KindPassphrase]; len(got) != 1 {
			t.Errorf("words = %v, want the bank added alongside", got)
		}
	})
}
