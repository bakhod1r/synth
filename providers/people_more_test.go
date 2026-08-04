package providers

import (
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

func ctxWith(seed uint64, params map[string]string) Ctx {
	l := locale.Get("en_US")
	p := l.Places[0]
	if params == nil {
		params = map[string]string{}
	}
	return Ctx{Rand: rng.New(seed), Locale: l, Params: params, Place: &p, Gender: "male"}
}

func ageOf(t *testing.T, v any) int {
	t.Helper()
	d, ok := v.(time.Time)
	if !ok {
		t.Fatalf("birthdate = %#v, want a time.Time", v)
	}
	years := birthAnchor.Year() - d.Year()
	if birthAnchor.YearDay() < d.YearDay() {
		years--
	}
	return years
}

// min and max are ages, not dates: a spec that says "adults" must keep meaning
// adults, whichever direction the dates run.
func TestBirthDateHonorsAgeBounds(t *testing.T) {
	cases := []struct {
		name   string
		params map[string]string
		lo, hi int
	}{
		{"default is adults", nil, defaultMinAge, defaultMaxAge},
		{"minors", map[string]string{"min": "0", "max": "17"}, 0, 17},
		{"pensioners", map[string]string{"min": "65"}, 65, defaultMaxAge},
		{"a single year", map[string]string{"min": "30", "max": "30"}, 30, 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := ctxWith(7, c.params)
			for i := 0; i < 300; i++ {
				got := ageOf(t, birthDate(ctx))
				if got < c.lo || got > c.hi {
					t.Fatalf("age %d is outside %d..%d", got, c.lo, c.hi)
				}
			}
		})
	}
}

// A reversed range is a caller slip, not a contradiction: it must still produce
// a date rather than an empty span.
func TestBirthDateWithAReversedRange(t *testing.T) {
	ctx := ctxWith(3, map[string]string{"min": "60", "max": "20"})
	for i := 0; i < 50; i++ {
		if _, ok := birthDate(ctx).(time.Time); !ok {
			t.Fatal("no date produced for a reversed range")
		}
	}
}

// The anchor is fixed on purpose: a golden file recorded today must not fail
// next month.
func TestBirthDateAnchorIsFixed(t *testing.T) {
	a := birthDate(ctxWith(11, nil))
	b := birthDate(ctxWith(11, nil))
	if a != b {
		t.Fatalf("the same seed gave two dates: %v vs %v", a, b)
	}
}

// A password hash column must be self-describing: without the salt and the cost
// nothing can verify it later.
func TestPasswordHashIsVerifiableInForm(t *testing.T) {
	got, ok := passwordHashProvider(ctxWith(5, nil)).(string)
	if !ok {
		t.Fatal("passwordhash did not produce a string")
	}
	parts := strings.Split(got, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		t.Fatalf("hash = %q, want pbkdf2-sha256$iterations$salt$digest", got)
	}
	if n, err := strconv.Atoi(parts[1]); err != nil || n <= 0 {
		t.Fatalf("iterations = %q", parts[1])
	}
	for i, seg := range parts[2:] {
		if _, err := base64.RawStdEncoding.DecodeString(seg); err != nil {
			t.Fatalf("segment %d is not base64: %q", i+2, seg)
		}
	}
}

func TestPasswordHashHonorsAnIterationCount(t *testing.T) {
	got := passwordHashProvider(ctxWith(6, map[string]string{"iterations": "1000"})).(string)
	if !strings.HasPrefix(got, "pbkdf2-sha256$1000$") {
		t.Fatalf("hash = %q, want the requested cost", got)
	}
	// A nonsense cost falls back to the fixture default rather than failing.
	got = passwordHashProvider(ctxWith(6, map[string]string{"iterations": "0"})).(string)
	if !strings.HasPrefix(got, "pbkdf2-sha256$"+strconv.Itoa(fixtureIterations)+"$") {
		t.Fatalf("hash = %q, want the default cost", got)
	}
}

// Two rows must not share a hash even when the generated password repeats: the
// salt is what makes that true.
func TestPasswordHashesDifferPerRow(t *testing.T) {
	ctx := ctxWith(8, nil)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		h := passwordHashProvider(ctx).(string)
		if seen[h] {
			t.Fatalf("a hash repeated: %q", h)
		}
		seen[h] = true
	}
}

func TestPassphraseShape(t *testing.T) {
	ctx := ctxWith(9, map[string]string{"words": "5", "sep": "_"})
	got := passphrase(ctx).(string)
	if n := len(strings.Split(got, "_")); n != 5 {
		t.Fatalf("passphrase %q has %d words, want 5", got, n)
	}

	// Fewer than two words is not a passphrase; the default is used instead.
	got = passphrase(ctxWith(9, map[string]string{"words": "1"})).(string)
	if n := len(strings.Split(got, "-")); n != 4 {
		t.Fatalf("passphrase %q has %d words, want the default 4", got, n)
	}

	got = passphrase(ctxWith(9, map[string]string{"capitalize": "true", "number": "true"})).(string)
	parts := strings.Split(got, "-")
	if len(parts) != 5 {
		t.Fatalf("with number=true, %q should carry a trailing group", got)
	}
	if _, err := strconv.Atoi(parts[len(parts)-1]); err != nil {
		t.Fatalf("the trailing group %q is not digits", parts[len(parts)-1])
	}
	for _, w := range parts[:len(parts)-1] {
		if w == "" || w[0] < 'A' || w[0] > 'Z' {
			t.Fatalf("capitalize=true left %q lowercase", w)
		}
	}
}

// Every locale with its own passphrase bank must be reachable through the
// catalog, or the option silently falls back to English.
func TestPassphraseBanksAreRegistered(t *testing.T) {
	for code := range passphraseBanks {
		words := localeCatalog[code][schema.KindPassphrase]
		if len(words) == 0 {
			t.Errorf("locale %q has a passphrase bank that was never registered", code)
		}
	}
}

// A title generator that repeats itself is useless for testing search,
// truncation or layout — which is the reason it exists.
func TestArticleTitlesVary(t *testing.T) {
	ctx := ctxWith(12, nil)
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		got, ok := articleTitle(ctx).(string)
		if !ok || got == "" {
			t.Fatalf("articleTitle = %#v", got)
		}
		if strings.Contains(got, "{") || strings.Contains(got, "}") {
			t.Fatalf("an unresolved placeholder survived: %q", got)
		}
		seen[got] = true
	}
	if len(seen) < 100 {
		t.Fatalf("only %d distinct titles in 500 draws", len(seen))
	}
}
