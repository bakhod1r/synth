package providers

import (
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
)

func TestAllNationalIDFormats(t *testing.T) {
	for lang, f := range nationalIDFormats {
		for seed := uint64(0); seed < 50; seed++ {
			out := f(rng.New(seed))
			if out == "" {
				t.Fatalf("format %q produced empty at seed %d", lang, seed)
			}
		}
	}
}

func TestLangOfNoSeparator(t *testing.T) {
	if langOf("en") != "en" {
		t.Fatal("langOf without separator should return input")
	}
	if langOf("uz_UZ") != "uz" {
		t.Fatal("langOf should strip region")
	}
}

func TestNationalIDFallbackAndNilLocale(t *testing.T) {
	// nil locale -> usSSN fallback.
	if nationalID(Ctx{Rand: rng.New(1)}) == nil {
		t.Fatal("nil locale should still produce an id")
	}
}

func TestVerhoeffReturnsDigit(t *testing.T) {
	for _, s := range []string{"", "1", "123456789", "0000"} {
		if d := verhoeff(s); d < 0 || d > 9 {
			t.Fatalf("verhoeff(%q) = %d out of range", s, d)
		}
	}
}
