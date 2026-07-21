package synth_test

import (
	"testing"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/locale"
)

// set turns a name list into a lookup.
func set(vals []string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// A record's first name, surname and gender field must agree. The expected
// names are read from the locale itself rather than copied into the test:
// a hardcoded copy goes stale the moment the name banks are enriched, and
// then the test fails for a reason that has nothing to do with coherence.
func TestGenderCoherence(t *testing.T) {
	type Person struct {
		ID        int
		FirstName string
		LastName  string
		Gender    string
	}

	for _, code := range []string{"uz_UZ", "en_US", "ru_RU", "cs_CZ", "pl_PL", "lv_LV"} {
		t.Run(code, func(t *testing.T) {
			l := locale.Get(code)
			if l == nil {
				t.Fatalf("%s is not registered", code)
			}
			maleFirst := set(l.FirstNamesFor("male"))
			femaleFirst := set(l.FirstNamesFor("female"))
			maleLast := set(l.LastNamesFor("male"))
			femaleLast := set(l.LastNamesFor("female"))

			// Only meaningful where the two surname lists actually differ.
			gendered := false
			for s := range maleLast {
				if !femaleLast[s] {
					gendered = true
					break
				}
			}

			seenMale, seenFemale := 0, 0
			for _, p := range synth.Make[Person](2000, synth.WithSeed(1), synth.WithLocale(code)) {
				switch p.Gender {
				case "male":
					seenMale++
					if !maleFirst[p.FirstName] {
						t.Fatalf("%s: %q is not a male first name", code, p.FirstName)
					}
					if gendered && !maleLast[p.LastName] {
						t.Fatalf("%s: male %q got the surname %q", code, p.FirstName, p.LastName)
					}
				case "female":
					seenFemale++
					if !femaleFirst[p.FirstName] {
						t.Fatalf("%s: %q is not a female first name", code, p.FirstName)
					}
					if gendered && !femaleLast[p.LastName] {
						t.Fatalf("%s: female %q got the surname %q", code, p.FirstName, p.LastName)
					}
				default:
					t.Fatalf("%s: unexpected gender %q", code, p.Gender)
				}
			}
			if seenMale == 0 || seenFemale == 0 {
				t.Fatalf("%s: got %d male and %d female records; one gender never appeared",
					code, seenMale, seenFemale)
			}
		})
	}
}

// Uzbek surnames inflect, so a woman must get the -a form. This is the
// concrete case the generic test above generalizes, kept because a regression
// here is the kind that reads as obviously wrong to a native speaker.
func TestUzbekFemaleSurnameIsInflected(t *testing.T) {
	type Person struct {
		FirstName string
		LastName  string
		Gender    string
	}
	sawFemale := false
	for _, p := range synth.Make[Person](500, synth.WithSeed(3), synth.WithLocale("uz_UZ")) {
		if p.Gender != "female" {
			continue
		}
		sawFemale = true
		if len(p.LastName) == 0 || p.LastName[len(p.LastName)-1] != 'a' {
			t.Fatalf("female %s got the masculine surname %q", p.FirstName, p.LastName)
		}
	}
	if !sawFemale {
		t.Fatal("no female records were generated")
	}
}
