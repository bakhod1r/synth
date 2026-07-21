package locale_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/bakhodir/synth/locale"
)

// Every locale must reach at least 1000 distinct full-name combinations for
// each gender. Below that, a table of a few thousand rows visibly repeats.
func TestEveryLocaleReachesThousandNames(t *testing.T) {
	for _, code := range locale.Names() {
		l := locale.Get(code)
		for _, g := range []string{"male", "female"} {
			first, last := l.FirstNamesFor(g), l.LastNamesFor(g)
			if n := len(first) * len(last); n < 1000 {
				t.Errorf("%s %s: only %d combinations (%d first x %d last)",
					code, g, n, len(first), len(last))
			}
		}
	}
}

// A duplicate inside one list silently reduces the real cardinality and makes
// one name twice as likely as its neighbours.
func TestNameListsHaveNoDuplicates(t *testing.T) {
	for _, code := range locale.Names() {
		l := locale.Get(code)
		lists := map[string][]string{
			"maleFirst":   l.FirstNamesFor("male"),
			"femaleFirst": l.FirstNamesFor("female"),
			"maleLast":    l.LastNamesFor("male"),
			"femaleLast":  l.LastNamesFor("female"),
		}
		for name, list := range lists {
			seen := map[string]bool{}
			for _, v := range list {
				if seen[v] {
					t.Errorf("%s %s: %q appears twice", code, name, v)
				}
				seen[v] = true
			}
		}
	}
}

// Male and female first-name lists must not overlap, or gender coherence is
// only nominal. A few genuinely unisex names are tolerated.
func TestGenderListsAreMostlyDistinct(t *testing.T) {
	for _, code := range locale.Names() {
		l := locale.Get(code)
		male := l.FirstNamesFor("male")
		female := map[string]bool{}
		for _, f := range l.FirstNamesFor("female") {
			female[f] = true
		}
		overlap := 0
		for _, m := range male {
			if female[m] {
				overlap++
			}
		}
		if len(male) > 0 && float64(overlap)/float64(len(male)) > 0.25 {
			t.Errorf("%s: %d of %d male first names also appear as female names",
				code, overlap, len(male))
		}
	}
}

// No name may be empty or carry stray whitespace: both show up directly in
// generated output.
func TestNamesAreWellFormed(t *testing.T) {
	for _, code := range locale.Names() {
		l := locale.Get(code)
		for _, g := range []string{"male", "female"} {
			for _, list := range [][]string{l.FirstNamesFor(g), l.LastNamesFor(g)} {
				for _, v := range list {
					if v == "" {
						t.Errorf("%s: empty name in a %s list", code, g)
						continue
					}
					if v != strings.TrimSpace(v) {
						t.Errorf("%s: %q has leading or trailing whitespace", code, v)
					}
					for _, r := range v {
						if unicode.IsControl(r) || unicode.IsDigit(r) {
							t.Errorf("%s: %q contains %q, which is not part of a name", code, v, r)
							break
						}
					}
				}
			}
		}
	}
}

// A name written in the wrong script is the clearest sign of a copy-paste
// slip, and it is invisible to a reader who does not know the language. These
// locales write names in a non-Latin script, so a Latin or Cyrillic letter in
// their lists is a mistake — except where the locale genuinely mixes scripts.
func TestNonLatinLocalesUseTheirOwnScript(t *testing.T) {
	nonLatin := map[string]*unicode.RangeTable{
		"ru_RU": unicode.Cyrillic, "uk_UA": unicode.Cyrillic, "bg_BG": unicode.Cyrillic,
		"kk_KZ": unicode.Cyrillic,
		"zh_CN": unicode.Han, "zh_TW": unicode.Han,
		"ko_KR": unicode.Hangul,
		"th_TH": unicode.Thai,
		"el_GR": unicode.Greek,
		"he_IL": unicode.Hebrew,
		"ka_GE": unicode.Georgian,
		"ar_SA": unicode.Arabic, "ar_EG": unicode.Arabic, "fa_IR": unicode.Arabic,
		"hi_IN": unicode.Devanagari, "bn_BD": unicode.Bengali,
	}
	for code, script := range nonLatin {
		l := locale.Get(code)
		if l == nil {
			t.Errorf("%s is not registered", code)
			continue
		}
		for _, g := range []string{"male", "female"} {
			for _, list := range [][]string{l.FirstNamesFor(g), l.LastNamesFor(g)} {
				for _, name := range list {
					if !mostlyIn(name, script) {
						t.Errorf("%s: %q is not written in the locale's script", code, name)
					}
				}
			}
		}
	}
}

// mostlyIn reports whether a name's letters are predominantly in the given
// script. A stray combining mark or space must not fail an otherwise correct
// name, but a Latin word inside a Cyrillic list must.
func mostlyIn(name string, script *unicode.RangeTable) bool {
	letters, inScript := 0, 0
	for _, r := range name {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.Is(script, r) {
			inScript++
		}
	}
	if letters == 0 {
		return false
	}
	return float64(inScript)/float64(letters) >= 0.8
}

// Languages that inflect surnames for gender must actually supply both forms,
// or a woman gets a man's surname.
func TestGenderedSurnameLocalesDiffer(t *testing.T) {
	for _, code := range []string{"ru_RU", "pl_PL", "cs_CZ", "sk_SK", "lv_LV", "lt_LT", "bg_BG", "el_GR"} {
		l := locale.Get(code)
		if l == nil {
			t.Errorf("%s is not registered", code)
			continue
		}
		male, female := l.LastNamesFor("male"), l.LastNamesFor("female")
		if len(male) != len(female) {
			t.Errorf("%s: %d male surnames but %d female", code, len(male), len(female))
			continue
		}
		differing := 0
		for i := range male {
			if male[i] != female[i] {
				differing++
			}
		}
		// Some surnames are invariant even in these languages, but most inflect.
		if float64(differing)/float64(len(male)) < 0.5 {
			t.Errorf("%s: only %d of %d surnames differ by gender; the female "+
				"forms look like they were copied from the male list",
				code, differing, len(male))
		}
	}
}

// Generation must actually use the banks, not just store them.
func TestGeneratedNamesComeFromTheBank(t *testing.T) {
	l := locale.Get("cs_CZ")
	female := map[string]bool{}
	for _, s := range l.LastNamesFor("female") {
		female[s] = true
	}
	if !female["Nováková"] {
		t.Fatal("cs_CZ female surnames do not include the inflected form Nováková")
	}
	male := map[string]bool{}
	for _, s := range l.LastNamesFor("male") {
		male[s] = true
	}
	if male["Nováková"] {
		t.Fatal("cs_CZ male surnames include a female form")
	}
}
