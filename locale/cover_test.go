package locale

import "testing"

func TestFirstLastNamesForFallback(t *testing.T) {
	// A synthetic locale with no gendered banks: every gender falls back to the
	// mixed lists, and an unknown gender does too.
	l := &Locale{
		FirstNames: []string{"a", "b"},
		LastNames:  []string{"x", "y"},
	}
	for _, g := range []string{"male", "female", "other", ""} {
		if got := l.FirstNamesFor(g); len(got) != 2 {
			t.Fatalf("FirstNamesFor(%q) = %v", g, got)
		}
		if got := l.LastNamesFor(g); len(got) != 2 {
			t.Fatalf("LastNamesFor(%q) = %v", g, got)
		}
	}
}

func TestMkIBANDefaults(t *testing.T) {
	// A seed with no IBAN info gets the fallback country and length.
	l := mk(seed{key: "xx_XX", region: "R", city: "C", postcode: "P", prefix: "1"})
	if l.IBANLength != 22 || l.IBANCountry != "XX" {
		t.Fatalf("IBAN defaults = %d,%q", l.IBANLength, l.IBANCountry)
	}
}

func TestGetFallbackAndDefaults(t *testing.T) {
	if Get("no_SUCH") != enUS {
		t.Fatal("unknown locale should fall back to en_US")
	}
	// Every registered locale resolves and carries IBAN defaults.
	for _, name := range Names() {
		l := Get(name)
		if l == nil {
			t.Fatalf("Get(%q) nil", name)
		}
		if l.IBANLength == 0 || l.IBANCountry == "" {
			t.Fatalf("locale %q missing IBAN defaults", name)
		}
	}
}
