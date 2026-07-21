package providers

import (
	"strconv"
	"strings"
	"testing"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/locale"
	"github.com/bakhodir/synth/schema"
)

// Every generated card, for every brand, must have the brand's length, start
// with one of its prefixes, and pass Luhn.
func TestBrandCardsLuhnAndPrefix(t *testing.T) {
	for brand, spec := range creditCards {
		for seed := uint64(0); seed < 500; seed++ {
			num := generateCard(rng.New(seed), brand)
			if len(num) != spec.length {
				t.Fatalf("%s: length %d, want %d (%q)", brand, len(num), spec.length, num)
			}
			if !luhnValid(num) {
				t.Fatalf("%s: %q not Luhn-valid", brand, num)
			}
			ok := false
			for _, p := range spec.prefixes {
				if strings.HasPrefix(num, strconv.Itoa(p)) {
					ok = true
					break
				}
			}
			if !ok {
				t.Fatalf("%s: %q has no valid prefix", brand, num)
			}
		}
	}
}

// The card provider honors an explicit brand param.
func TestCardProviderBrandParam(t *testing.T) {
	c := Ctx{
		Rand:   rng.New(1),
		Locale: locale.Get("uz_UZ"),
		Params: map[string]string{"brand": "american express"},
		Field:  &schema.Field{},
	}
	num := Get(schema.KindCard)(c).(string)
	if len(num) != 15 {
		t.Fatalf("amex should be 15 digits, got %d (%q)", len(num), num)
	}
	if !strings.HasPrefix(num, "34") && !strings.HasPrefix(num, "37") {
		t.Fatalf("amex prefix wrong: %q", num)
	}
	if !luhnValid(num) {
		t.Fatalf("amex not Luhn-valid: %q", num)
	}
}
