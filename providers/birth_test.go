package providers_test

import (
	"testing"
	"time"

	"github.com/bakhod1r/synth"
)

// anchor is the fixed "today" the birthdate provider measures ages from.
var anchor = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func birthRows(t *testing.T, fields string) []map[string]any {
	t.Helper()
	y, err := synth.YAMLBytes([]byte("name: t\ncount: 300\nseed: 17\nfields:\n" + fields))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func ageOn(born time.Time) int {
	years := anchor.Year() - born.Year()
	if anchor.YearDay() < born.YearDay() {
		years--
	}
	return years
}

// The default is adults, because that is what a user table almost always holds.
func TestBirthDateDefaultsToAdults(t *testing.T) {
	for i, r := range birthRows(t, "  dob: { kind: birthdate }\n") {
		born, ok := r["dob"].(time.Time)
		if !ok {
			t.Fatalf("row %d: dob is %T, not a time", i, r["dob"])
		}
		if got := ageOn(born); got < 18 || got > 80 {
			t.Fatalf("row %d: %s is age %d, outside the 18..80 default",
				i, born.Format("2006-01-02"), got)
		}
	}
}

// min and max are ages, not dates. Reading them as dates would invert their
// meaning — the "minimum" date is the oldest person — and nobody would notice
// until the data was wrong in production.
func TestBirthDateBoundsAreAges(t *testing.T) {
	for _, tc := range []struct {
		field    string
		lo, hi   int
		describe string
	}{
		{"  d: { kind: birthdate, min: 0, max: 17 }\n", 0, 17, "minors"},
		{"  d: { kind: birthdate, min: 65 }\n", 65, 80, "pensioners"},
		{"  d: { kind: birthdate, min: 30, max: 30 }\n", 30, 30, "exactly thirty"},
	} {
		for i, r := range birthRows(t, tc.field) {
			born := r["d"].(time.Time)
			if got := ageOn(born); got < tc.lo || got > tc.hi {
				t.Fatalf("%s row %d: %s is age %d, outside %d..%d",
					tc.describe, i, born.Format("2006-01-02"), got, tc.lo, tc.hi)
			}
		}
	}
}

// A row saying 34 next to a date of birth in 1990 makes a fixture untrustworthy
// the moment anyone checks.
func TestAgeAgreesWithItsBirthDate(t *testing.T) {
	rows := birthRows(t, "  dob: { kind: birthdate }\n  age: { kind: age, from: dob }\n")
	for i, r := range rows {
		born := r["dob"].(time.Time)
		got, ok := r["age"].(int)
		if !ok {
			t.Fatalf("row %d: age is %T, not an int", i, r["age"])
		}
		if want := ageOn(born); got != want {
			t.Fatalf("row %d: age %d does not match a birth date of %s (want %d)",
				i, got, born.Format("2006-01-02"), want)
		}
	}
}

// Without from=, age is a plain number in range — useful on its own.
func TestStandaloneAge(t *testing.T) {
	for i, r := range birthRows(t, "  a: { kind: age, min: 21, max: 35 }\n") {
		got := r["a"].(int)
		if got < 21 || got > 35 {
			t.Fatalf("row %d: age %d is outside 21..35", i, got)
		}
	}
}

// The anchor is fixed rather than time.Now(), so a golden file recorded today
// still matches next year. A moving anchor makes every dated fixture expire.
func TestBirthDateIsReproducible(t *testing.T) {
	a := birthRows(t, "  d: { kind: birthdate }\n")
	b := birthRows(t, "  d: { kind: birthdate }\n")
	for i := range a {
		if !a[i]["d"].(time.Time).Equal(b[i]["d"].(time.Time)) {
			t.Fatalf("row %d differs between runs with the same seed", i)
		}
	}
}

// A reversed range must not produce an empty or panicking result.
func TestBirthDateHandlesAReversedRange(t *testing.T) {
	for i, r := range birthRows(t, "  d: { kind: birthdate, min: 60, max: 20 }\n") {
		if got := ageOn(r["d"].(time.Time)); got != 60 {
			t.Fatalf("row %d: a reversed range gave age %d, want the min held at 60", i, got)
		}
	}
}

// A date of birth has no clock time. Storing 1975-02-23T14:37:09 in a DOB
// column is the kind of detail that quietly breaks a date comparison.
func TestBirthDateHasNoTimeOfDay(t *testing.T) {
	for i, r := range birthRows(t, "  d: { kind: birthdate }\n") {
		born := r["d"].(time.Time)
		if h, m, s := born.Clock(); h != 0 || m != 0 || s != 0 {
			t.Fatalf("row %d: %s carries a time of day", i, born.Format(time.RFC3339))
		}
	}
}
