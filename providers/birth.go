package providers

import (
	"time"

	"github.com/bakhod1r/synth/schema"
)

// Birth dates and ages.
//
// A date of birth is one of the most common columns in real data, and until now
// it needed `kind: time` with hand-written min and max dates. That works, but it
// asks the author to do arithmetic — and the arithmetic goes stale, because a
// spec written with `min: 1960-01-01` describes a different population every
// year it is re-run.
//
// So the range is expressed in ages instead, which is what the author actually
// means: "adults" rather than "born before 2008".

// birthAnchor is the "today" ages are measured from.
//
// It is fixed rather than time.Now() for the same reason card expiry is: a
// generated dataset must be byte-identical from the same seed forever. With a
// moving anchor, a golden file recorded in July fails in August and the failure
// looks like a bug in whatever changed that week.
var birthAnchor = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// Default age range: adults, which is what a user table almost always holds.
const (
	defaultMinAge = 18
	defaultMaxAge = 80
)

func init() {
	registry[schema.KindBirthDate] = birthDate
	registry[schema.KindAge] = age
}

// birthDate generates a date of birth.
//
//	{ kind: birthdate }                     18..80 years old
//	{ kind: birthdate, min: 0, max: 17 }    minors
//	{ kind: birthdate, min: 65 }            pensioners
//
// min and max are ages in years, not dates. Reading them as dates would be the
// other reasonable choice, but then `min` would mean the *oldest* person and
// `max` the youngest, which inverts every time someone reads it.
func birthDate(c Ctx) any {
	minAge, maxAge := ageRange(c)
	// A younger age means a later date, so the age bounds swap when converted.
	newest := birthAnchor.AddDate(-minAge, 0, 0)
	oldest := birthAnchor.AddDate(-maxAge-1, 0, 1)
	span := newest.Sub(oldest)
	if span <= 0 {
		return newest
	}
	return oldest.Add(time.Duration(c.Rand.Float64() * float64(span))).Truncate(24 * time.Hour)
}

// age generates a whole number of years.
//
// With from= naming a birthdate column it is computed from that date rather
// than drawn independently, so the two agree. A row saying 34 next to a date of
// birth in 1990 is the kind of detail that makes a fixture untrustworthy the
// moment someone checks.
func age(c Ctx) any {
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		if born, ok := c.Sibling(c.Field.From).(time.Time); ok {
			return yearsBetween(born, birthAnchor)
		}
	}
	minAge, maxAge := ageRange(c)
	return c.Rand.IntRange(minAge, maxAge)
}

// yearsBetween counts whole years, decrementing when the birthday has not yet
// come round in the final year. Dividing the day count by 365.25 is close but
// wrong for anyone whose birthday is within a few days of the anchor.
func yearsBetween(born, now time.Time) int {
	years := now.Year() - born.Year()
	if now.YearDay() < born.YearDay() {
		years--
	}
	if years < 0 {
		years = 0
	}
	return years
}

// ageRange reads the min/max age params, keeping them ordered and non-negative.
func ageRange(c Ctx) (int, int) {
	minAge := paramInt(c.Params, "min", defaultMinAge)
	maxAge := paramInt(c.Params, "max", defaultMaxAge)
	if minAge < 0 {
		minAge = 0
	}
	if maxAge < minAge {
		maxAge = minAge
	}
	return minAge, maxAge
}
