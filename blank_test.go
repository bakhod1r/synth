package synth_test

import (
	"math"
	"testing"
	"time"

	"github.com/bakhod1r/synth"
)

// A blank share must land close to what was asked for. Real tables have
// missing values, and code that has only ever seen complete rows breaks the
// first time it meets a null.
func TestBlankShareIsHonored(t *testing.T) {
	type Row struct {
		Always string `synth:"name"`
		Some   string `synth:"name,blank=25%"`
		Never  string `synth:"name"`
	}
	const n = 20000
	rows := synth.Make[Row](n, synth.WithSeed(1))

	blanks := 0
	for _, r := range rows {
		if r.Always == "" {
			t.Fatal("a field with no blank share came back empty")
		}
		if r.Never == "" {
			t.Fatal("a field with no blank share came back empty")
		}
		if r.Some == "" {
			blanks++
		}
	}
	got := float64(blanks) / n
	if math.Abs(got-0.25) > 0.02 {
		t.Fatalf("blank share is %.3f, want about 0.25", got)
	}
}

// "15", "15%" and "0.15" must all mean fifteen percent. Guessing wrong by a
// factor of a hundred is an unpleasant surprise.
func TestBlankAcceptsPercentAndFraction(t *testing.T) {
	type Percent struct {
		V string `synth:"name,blank=15%"`
	}
	type Bare struct {
		V string `synth:"name,blank=15"`
	}
	type Fraction struct {
		V string `synth:"name,blank=0.15"`
	}
	const n = 20000

	share := func(count int) float64 { return float64(count) / n }
	countBlank := func(vals []string) int {
		c := 0
		for _, v := range vals {
			if v == "" {
				c++
			}
		}
		return c
	}

	var a, b, c []string
	for _, r := range synth.Make[Percent](n, synth.WithSeed(2)) {
		a = append(a, r.V)
	}
	for _, r := range synth.Make[Bare](n, synth.WithSeed(2)) {
		b = append(b, r.V)
	}
	for _, r := range synth.Make[Fraction](n, synth.WithSeed(2)) {
		c = append(c, r.V)
	}
	for name, vals := range map[string][]string{"15%": a, "15": b, "0.15": c} {
		if got := share(countBlank(vals)); math.Abs(got-0.15) > 0.02 {
			t.Errorf("blank=%s gave %.3f, want about 0.15", name, got)
		}
	}
}

// A primary key must never be blanked: a row without an identity is not
// missing data, it is a broken row.
func TestPrimaryKeyIsNeverBlanked(t *testing.T) {
	type Row struct {
		ID   string `synth:"uuid,pk,blank=90%"`
		Name string `synth:"name,blank=90%"`
	}
	rows := synth.Make[Row](2000, synth.WithSeed(3))
	blankNames := 0
	for i, r := range rows {
		if r.ID == "" {
			t.Fatalf("row %d has no primary key", i)
		}
		if r.Name == "" {
			blankNames++
		}
	}
	if blankNames == 0 {
		t.Fatal("nothing was blanked — the test proves nothing")
	}
}

// A nonsensical blank share must be ignored rather than silently blanking
// everything or nothing at random.
func TestInvalidBlankIsIgnored(t *testing.T) {
	type Row struct {
		V string `synth:"name,blank=abc"`
	}
	for _, r := range synth.Make[Row](200, synth.WithSeed(4)) {
		if r.V == "" {
			t.Fatal("an unparseable blank share blanked a value")
		}
	}
}

// min/max bound a date column directly, which is what a date filter means.
// They are named after the numeric bounds rather than from/to, because from=
// already means "this timestamp follows that column".
func TestTimeRespectsItsWindow(t *testing.T) {
	type Row struct {
		When time.Time `synth:"time,min=2026-01-01,max=2026-03-01"`
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i, r := range synth.Make[Row](2000, synth.WithSeed(5)) {
		if r.When.Before(from) || r.When.After(to) {
			t.Fatalf("row %d: %s is outside the requested window", i, r.When)
		}
	}
}

// An unparseable bound must not silently become the zero time, which would put
// every row in the year 1.
func TestInvalidDateBoundIsIgnored(t *testing.T) {
	type Row struct {
		When time.Time `synth:"time,min=last-tuesday"`
	}
	for i, r := range synth.Make[Row](200, synth.WithSeed(6)) {
		if r.When.Year() < 2000 {
			t.Fatalf("row %d landed in %d — a bad bound became the zero time",
				i, r.When.Year())
		}
	}
}

// Blanking must not disturb the rest of the record: the same seed with the
// same schema still has to be reproducible.
func TestBlankIsDeterministic(t *testing.T) {
	type Row struct {
		A string `synth:"name,blank=30%"`
		B string `synth:"email"`
	}
	x := synth.Make[Row](500, synth.WithSeed(7))
	y := synth.Make[Row](500, synth.WithSeed(7))
	for i := range x {
		if x[i] != y[i] {
			t.Fatalf("row %d differs between runs: %+v vs %+v", i, x[i], y[i])
		}
	}
}
