package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/synth"
)

type contact struct {
	Phone    string `synth:"phone_e164"`
	Display  string `synth:"phone_national,from=Phone"`
	Intl     string `synth:"phone_international,from=Phone"`
	LineType string `synth:"phone_type,from=Phone"`
}

// The whole point: every number a numbering plan would accept.
func TestPhoneNumbersAreValid(t *testing.T) {
	for _, loc := range []string{"uz_UZ", "en_US", "de_DE", "ru_RU", "fr_FR", "ja_JP"} {
		for _, c := range synth.Make[contact](100, synth.WithSeed(1), synth.WithLocale(loc)) {
			p, err := phonex.Parse(c.Phone)
			if err != nil {
				t.Fatalf("%s: %q does not parse: %v", loc, c.Phone, err)
			}
			if !p.IsValid() {
				t.Errorf("%s: %q parses but is not valid", loc, c.Phone)
			}
		}
	}
}

// The formatted fields must describe the number in Phone, not another one.
func TestPhoneFormatsAgreeWithinRecord(t *testing.T) {
	for _, c := range synth.Make[contact](100, synth.WithSeed(2), synth.WithLocale("uz_UZ")) {
		p, err := phonex.Parse(c.Phone)
		if err != nil {
			t.Fatalf("%q does not parse: %v", c.Phone, err)
		}
		if c.Display != p.National() {
			t.Errorf("%q: national = %q, want %q", c.Phone, c.Display, p.National())
		}
		if c.Intl != p.International() {
			t.Errorf("%q: international = %q, want %q", c.Phone, c.Intl, p.International())
		}
	}
}

// The number must belong to the record's locale, not to whichever region the
// metadata happened to list first.
func TestPhoneMatchesLocale(t *testing.T) {
	for _, tc := range []struct{ locale, dial string }{
		{"uz_UZ", "+998"},
		{"de_DE", "+49"},
		{"ja_JP", "+81"},
	} {
		for _, c := range synth.Make[contact](20, synth.WithSeed(3), synth.WithLocale(tc.locale)) {
			if !strings.HasPrefix(c.Phone, tc.dial) {
				t.Errorf("%s: %q is not a %s number", tc.locale, c.Phone, tc.dial)
			}
		}
	}
}

// A record's number should be issued where the record lives: the locale gives
// each place an operator or area code, and the number is built on it.
func TestPhoneMatchesPlace(t *testing.T) {
	type located struct {
		City  string `synth:"city"`
		Phone string `synth:"phone_e164"`
	}
	for _, tc := range []struct {
		locale string
		dial   string
		code   map[string]string
	}{
		{"uz_UZ", "+998", map[string]string{
			"Toshkent": "90", "Samarqand": "91", "Buxoro": "93",
			"Andijon": "94", "Farg'ona": "95",
		}},
		{"en_US", "+1", map[string]string{
			"Los Angeles": "213", "San Diego": "619", "New York": "212",
			"Houston": "713", "Chicago": "312",
		}},
		{"ru_RU", "+7", map[string]string{
			"Москва": "495", "Санкт-Петербург": "812", "Екатеринбург": "343",
		}},
		// The region cannot come from the dialling code: +1 is Canada as well
		// as the United States, and +7 is Kazakhstan as well as Russia.
		{"en_CA", "+1", map[string]string{"Toronto": "416"}},
		{"kk_KZ", "+7", map[string]string{"Астана": "7172"}},
		// Plans that count the trunk digit as part of the national number: the
		// place code an atlas gives is a digit short of what they want.
		{"en_GB", "+44", map[string]string{"London": "20"}},
		{"it_IT", "+39", map[string]string{"Roma": "06"}},
		{"ja_JP", "+81", map[string]string{"東京": "3"}},
	} {
		for _, r := range synth.Make[located](60, synth.WithSeed(8), synth.WithLocale(tc.locale)) {
			code, known := tc.code[r.City]
			if !known {
				continue // the locale has places this test does not pin down
			}
			if want := tc.dial + code; !strings.HasPrefix(r.Phone, want) {
				t.Errorf("%s: %s has %s, want a %s number", tc.locale, r.City, r.Phone, want)
			}
		}
	}
}

// Locality is a preference, not a guarantee: where the plan will not take the
// place's code, the number must still be valid, and every record must still get
// a different one.
func TestPhoneNumbersStayValidAndVaried(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range synth.Make[contact](100, synth.WithSeed(10), synth.WithLocale("ja_JP")) {
		p, err := phonex.Parse(c.Phone)
		if err != nil || !p.IsValid() {
			t.Fatalf("%q is not a valid number: %v", c.Phone, err)
		}
		seen[c.Phone] = true
	}
	if len(seen) < 90 {
		t.Errorf("100 records produced only %d distinct numbers", len(seen))
	}
}

func TestPhoneSameSeedSameNumbers(t *testing.T) {
	a := synth.Make[contact](50, synth.WithSeed(9), synth.WithLocale("uz_UZ"))
	b := synth.Make[contact](50, synth.WithSeed(9), synth.WithLocale("uz_UZ"))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("record %d differs between runs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// A region whose plan does not separate mobile from fixed line (North America)
// once left every record holding the metadata's example number.
func TestPhoneNumbersVaryWithinLocale(t *testing.T) {
	for _, loc := range []string{"uz_UZ", "en_US", "de_DE"} {
		seen := map[string]bool{}
		for _, c := range synth.Make[contact](50, synth.WithSeed(4), synth.WithLocale(loc)) {
			seen[c.Phone] = true
		}
		if len(seen) < 25 {
			t.Errorf("%s: 50 records produced only %d distinct numbers", loc, len(seen))
		}
	}
}

// Where the plan does not separate the two, say so rather than guess.
func TestPhoneLineTypeIsHonest(t *testing.T) {
	for _, tc := range []struct{ locale, want string }{
		{"uz_UZ", "mobile"},
		{"en_US", "fixed_line_or_mobile"},
	} {
		for _, c := range synth.Make[contact](20, synth.WithSeed(5), synth.WithLocale(tc.locale)) {
			if c.LineType != tc.want {
				t.Errorf("%s: line type = %q, want %q (number %q)",
					tc.locale, c.LineType, tc.want, c.Phone)
			}
		}
	}
}

// A from= value the metadata rejects is the user's own field. Reformatting is
// impossible; a type column gets "unknown" rather than the digits.
func TestPhoneUnparseableFromIsNotInvented(t *testing.T) {
	type odd struct {
		Phone    string `synth:"lorem"`
		Display  string `synth:"phone_national,from=Phone"`
		LineType string `synth:"phone_type,from=Phone"`
	}
	for _, r := range synth.Make[odd](10, synth.WithSeed(6)) {
		if r.Display != r.Phone {
			t.Errorf("national = %q, want the input %q back", r.Display, r.Phone)
		}
		if r.LineType != "unknown" {
			t.Errorf("line type = %q, want %q", r.LineType, "unknown")
		}
	}
}

func BenchmarkPhoneRecord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		synth.Make[contact](1, synth.WithSeed(uint64(i)), synth.WithLocale("uz_UZ"))
	}
}
