package providers

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestLocalizable(t *testing.T) {
	// Locale-driven: the provider reads the locale itself.
	for _, k := range []schema.Kind{schema.KindFirstName, schema.KindCity, schema.KindPhone, schema.KindCurrency} {
		if !Localizable(k) {
			t.Errorf("%s reads the locale but Localizable says otherwise", k)
		}
	}
	// Catalog-driven: at least one locale ships its own dataset.
	for _, k := range LocalizedKinds() {
		if !Localizable(k) {
			t.Errorf("%s has per-locale data but Localizable says otherwise", k)
		}
	}
	// The same everywhere: a UUID or an HTTP status has no language.
	for _, k := range []schema.Kind{schema.KindUUID, schema.KindHTTPStatus, schema.KindMD5, schema.KindPort} {
		if Localizable(k) {
			t.Errorf("%s is locale-independent but Localizable claims it is not", k)
		}
	}
	if Localizable(schema.KindUnknown) {
		t.Error("the unknown kind must not be reported as localizable")
	}
}

func TestLocalizableKindsCoversBothSources(t *testing.T) {
	kinds := LocalizableKinds()
	set := map[schema.Kind]bool{}
	for _, k := range kinds {
		set[k] = true
	}
	for i := 1; i < len(kinds); i++ {
		if kinds[i-1] >= kinds[i] {
			t.Fatalf("LocalizableKinds is not sorted: %q before %q", kinds[i-1], kinds[i])
		}
	}
	for k := range localeDriven {
		if !set[k] {
			t.Errorf("locale-driven kind %s missing from LocalizableKinds", k)
		}
	}
	for _, k := range LocalizedKinds() {
		if !set[k] {
			t.Errorf("catalog-localized kind %s missing from LocalizableKinds", k)
		}
	}
}

// Every kind named as locale-driven must actually be a registered provider —
// otherwise the list rots silently as kinds are renamed.
func TestLocaleDrivenKindsExist(t *testing.T) {
	for k := range localeDriven {
		if Get(k) == nil {
			t.Errorf("localeDriven names %q, which is not a registered kind", k)
		}
	}
}
