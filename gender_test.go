package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

// In gendered-surname locales, a female first name must get a female surname
// form and a Gender field of "female" (and vice versa).
func TestGenderCoherenceUz(t *testing.T) {
	type Person struct {
		ID        int
		FirstName string
		LastName  string
		Gender    string
	}
	femaleFirst := map[string]bool{
		"Dilnoza": true, "Malika": true, "Nilufar": true, "Gulnora": true, "Shahnoza": true,
		"Zuhra": true, "Kamola": true, "Feruza": true, "Sevara": true, "Madina": true,
		"Nodira": true, "Charos": true, "Dilfuza": true, "Muslima": true, "Gulbahor": true,
		"Zilola": true, "Nargiza": true, "Sabina": true, "Laylo": true, "Zarina": true,
		"Maftuna": true, "Dilorom": true, "Shahzoda": true, "Ozoda": true,
	}
	for _, p := range synth.Make[Person](2000, synth.WithSeed(1), synth.WithLocale("uz_UZ")) {
		isFemaleFirst := femaleFirst[p.FirstName]
		femaleSurname := strings.HasSuffix(p.LastName, "a") // -ova/-eva/-a
		if isFemaleFirst != femaleSurname {
			t.Fatalf("name gender mismatch: %s %s", p.FirstName, p.LastName)
		}
		wantGender := "male"
		if isFemaleFirst {
			wantGender = "female"
		}
		if p.Gender != wantGender {
			t.Fatalf("%s %s: gender %q, want %q", p.FirstName, p.LastName, p.Gender, wantGender)
		}
	}
}

// English: first name matches gender field; surnames are shared (not gendered).
func TestGenderCoherenceEn(t *testing.T) {
	type Person struct {
		ID        int
		FirstName string
		Gender    string
	}
	maleFirst := map[string]bool{
		"James": true, "John": true, "Robert": true, "Michael": true, "William": true,
		"David": true, "Richard": true, "Joseph": true, "Thomas": true, "Charles": true,
		"Daniel": true, "Matthew": true, "Anthony": true, "Mark": true, "Donald": true,
		"Steven": true, "Andrew": true, "Paul": true, "Joshua": true, "Kevin": true,
		"Brian": true, "George": true, "Edward": true, "Ronald": true,
	}
	for _, p := range synth.Make[Person](2000, synth.WithSeed(2), synth.WithLocale("en_US")) {
		want := "female"
		if maleFirst[p.FirstName] {
			want = "male"
		}
		if p.Gender != want {
			t.Fatalf("%s: gender %q, want %q", p.FirstName, p.Gender, want)
		}
	}
}
