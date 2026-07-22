package providers_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

func ids(t *testing.T, loc string) []string {
	t.Helper()
	y, err := synth.YAMLBytes([]byte("name: t\ncount: 200\nseed: 5\nfields:\n  id: { kind: ssn }\n"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate(synth.WithLocale(loc))
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r["id"].(string)
	}
	return out
}

// A US SSN on an Uzbek record is the detail that makes a fixture obviously
// fake to the person who has to trust it.
func TestNationalIDFollowsLocale(t *testing.T) {
	for _, tc := range []struct {
		locale string
		re     string
	}{
		{"uz_UZ", `^\d{14}$`},
		{"en_US", `^\d{3}-\d{2}-\d{4}$`},
		{"tr_TR", `^\d{11}$`},
		{"ru_RU", `^\d{4} \d{6}$`},
		{"es_ES", `^\d{8}[A-Z]$`},
		{"pl_PL", `^\d{11}$`},
		{"zh_CN", `^\d{17}[0-9X]$`},
		{"ko_KR", `^\d{6}-\d{7}$`},
	} {
		re := regexp.MustCompile(tc.re)
		for _, v := range ids(t, tc.locale) {
			if !re.MatchString(v) {
				t.Fatalf("%s: %q does not match %s", tc.locale, v, tc.re)
			}
		}
	}
}

// A generated identifier has to pass the validator it will be fed to, or the
// fixture tests nothing.
func TestPINFLCheckDigit(t *testing.T) {
	weights := []int{7, 3, 1, 7, 3, 1, 7, 3, 1, 7, 3, 1, 7}
	for _, v := range ids(t, "uz_UZ") {
		sum := 0
		for i, w := range weights {
			sum += int(v[i]-'0') * w
		}
		if sum%10 != int(v[13]-'0') {
			t.Fatalf("PINFL %q has a wrong check digit", v)
		}
	}
}

func TestTCKimlikCheckDigits(t *testing.T) {
	for _, v := range ids(t, "tr_TR") {
		if v[0] == '0' {
			t.Fatalf("TC Kimlik %q starts with zero", v)
		}
		odd, even, sum := 0, 0, 0
		for i := 0; i < 9; i++ {
			if i%2 == 0 {
				odd += int(v[i] - '0')
			} else {
				even += int(v[i] - '0')
			}
		}
		if want := ((odd*7-even)%10 + 10) % 10; want != int(v[9]-'0') {
			t.Fatalf("TC Kimlik %q: 10th digit is wrong", v)
		}
		for i := 0; i < 10; i++ {
			sum += int(v[i] - '0')
		}
		if sum%10 != int(v[10]-'0') {
			t.Fatalf("TC Kimlik %q: 11th digit is wrong", v)
		}
	}
}

func TestSpanishDNILetter(t *testing.T) {
	const table = "TRWAGMYFPDXBNJZSQVHLCKE"
	for _, v := range ids(t, "es_ES") {
		n := 0
		for i := 0; i < 8; i++ {
			n = n*10 + int(v[i]-'0')
		}
		if table[n%23] != v[8] {
			t.Fatalf("DNI %q has the wrong letter", v)
		}
	}
}

func TestChineseResidentIDCheckCharacter(t *testing.T) {
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	for _, v := range ids(t, "zh_CN") {
		sum := 0
		for i, w := range weights {
			sum += int(v[i]-'0') * w
		}
		if "10X98765432"[sum%11] != v[17] {
			t.Fatalf("resident id %q has the wrong check character", v)
		}
	}
}

// A US SSN must avoid the ranges the SSA never issues.
func TestUSSSNAvoidsUnissuedRanges(t *testing.T) {
	for _, v := range ids(t, "en_US") {
		area := v[:3]
		if area == "000" || area == "666" || v[0] == '9' {
			t.Fatalf("SSN %q uses an unissued area", v)
		}
		if v[4:6] == "00" || v[7:] == "0000" {
			t.Fatalf("SSN %q uses an unissued group or serial", v)
		}
	}
}

// An unknown locale must still produce something, not an empty string.
func TestUnknownLocaleFallsBack(t *testing.T) {
	for _, v := range ids(t, "en_GB") {
		if strings.TrimSpace(v) == "" {
			t.Fatal("an unmapped locale produced an empty identifier")
		}
	}
}

// pinfl, nationalid and taxid are the names people search for. Someone building
// an Uzbek schema looks for "pinfl", not "ssn", and finding nothing they invent
// a text column instead — so the alias is the difference between the feature
// being found and not.
func TestNationalIDAliases(t *testing.T) {
	for _, alias := range []string{"pinfl", "nationalid", "taxid"} {
		y, err := synth.YAMLBytes([]byte(
			"name: t\ncount: 50\nseed: 5\nfields:\n  a: { kind: " + alias + " }\n  b: { kind: ssn }\n"))
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		rows, err := y.Generate(synth.WithLocale("uz_UZ"))
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		for i, r := range rows {
			v, _ := r["a"].(string)
			if len(v) != 14 {
				t.Fatalf("%s row %d: %q is not a 14-digit PINFL", alias, i, v)
			}
		}
	}
}

// An alias must still follow the locale. Pinning "pinfl" to Uzbekistan would be
// a different, worse behaviour hidden behind a familiar name.
func TestAliasesFollowTheLocale(t *testing.T) {
	y, _ := synth.YAMLBytes([]byte("name: t\ncount: 20\nseed: 5\nfields:\n  a: { kind: pinfl }\n"))
	de, err := y.Generate(synth.WithLocale("de_DE"))
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range de {
		if v := r["a"].(string); len(v) != 11 {
			t.Fatalf("row %d: pinfl in de_DE gave %q, want an 11-digit Steuer-ID", i, v)
		}
	}
}
