package providers

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/locale"
	"github.com/bakhodir/synth/schema"
)

func ctx(seed uint64, loc string) Ctx {
	l := locale.Get(loc)
	p := l.Places[0]
	return Ctx{Rand: rng.New(seed), Locale: l, Params: map[string]string{}, Place: &p}
}

// Property: every generated card passes the Luhn check.
func TestCardLuhnValid(t *testing.T) {
	for seed := uint64(0); seed < 2000; seed++ {
		c := ctx(seed, "uz_UZ")
		num := Get(schema.KindCard)(c).(string)
		if !luhnValid(num) {
			t.Fatalf("card %q is not Luhn-valid", num)
		}
		// UZ cards must start with a HUMO/UZCARD BIN.
		if !strings.HasPrefix(num, "8600") && !strings.HasPrefix(num, "9860") {
			t.Fatalf("uz card %q has wrong BIN", num)
		}
	}
}

// Property: every IBAN passes the mod-97 checksum.
func TestIBANChecksumValid(t *testing.T) {
	for seed := uint64(0); seed < 2000; seed++ {
		c := ctx(seed, "uz_UZ")
		ib := Get(schema.KindIBAN)(c).(string)
		if len(ib) != c.Locale.IBANLength {
			t.Fatalf("iban %q wrong length %d", ib, len(ib))
		}
		if !ibanValid(ib) {
			t.Fatalf("iban %q failed mod-97", ib)
		}
	}
}

func TestPhoneHasCountryCode(t *testing.T) {
	c := ctx(1, "uz_UZ")
	ph := Get(schema.KindPhone)(c).(string)
	if !strings.HasPrefix(ph, "+998") {
		t.Fatalf("uz phone %q missing +998", ph)
	}
}

func luhnValid(s string) bool {
	sum, alt := 0, false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}

func ibanValid(s string) bool {
	rearranged := s[4:] + s[:4]
	var b strings.Builder
	for _, ch := range rearranged {
		if ch >= 'A' && ch <= 'Z' {
			b.WriteString(itoa(int(ch - 'A' + 10)))
		} else {
			b.WriteRune(ch)
		}
	}
	return mod97(b.String()) == 1
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
