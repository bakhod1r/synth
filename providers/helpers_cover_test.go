package providers

import (
	"testing"
	"time"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

func baseCtx() Ctx {
	l := locale.Get("en_US")
	p := l.Places[0]
	return Ctx{Rand: rng.New(1), Locale: l, Params: map[string]string{}, Place: &p, Gender: "male",
		Field: &schema.Field{Name: "f", Params: map[string]string{}}}
}

func TestEnumBranches(t *testing.T) {
	if enum(Ctx{Field: nil}) != "" {
		t.Fatal("nil field -> empty")
	}
	if enum(Ctx{Field: &schema.Field{}}) != "" {
		t.Fatal("no choices -> empty")
	}
	c := baseCtx()
	c.Field.Choices = []string{"a", "b", "c", "d"}
	c.Params["dist"] = "zipf"
	c.Field.Params = c.Params
	got := enum(c)
	if got == "" {
		t.Fatal("zipf enum produced empty")
	}
}

func TestWeightedPickFallback(t *testing.T) {
	// All-zero weights: u<acc never holds, so the loop returns the last index.
	if got := weightedPick(rng.New(1), []float64{0, 0, 0}); got != 2 {
		t.Fatalf("all-zero weights should return last index, got %d", got)
	}
}

func TestSampleDist(t *testing.T) {
	for _, name := range []string{"normal", "lognormal", "exp", "exponential"} {
		c := baseCtx()
		c.Params["dist"] = name
		if _, ok := sampleDist(c); !ok {
			t.Fatalf("dist %q not sampled", name)
		}
	}
	c := baseCtx()
	if _, ok := sampleDist(c); ok {
		t.Fatal("no dist -> false")
	}
	c.Params["dist"] = "bogus"
	if _, ok := sampleDist(c); ok {
		t.Fatal("unknown dist -> false")
	}
}

func TestAmountWithDist(t *testing.T) {
	c := baseCtx()
	c.Params["dist"] = "normal"
	c.Params["min"] = "10"
	c.Params["max"] = "20"
	v := amount(c).(float64)
	if v < 10 || v > 20 {
		t.Fatalf("amount out of clamp: %v", v)
	}
}

func TestRegisterPickStringHas(t *testing.T) {
	k := schema.Kind("customtestkind")
	Register(k, func(c Ctx) any { return "custom" })
	if !Has(k) || Get(k) == nil {
		t.Fatal("Register/Has failed")
	}
	c := baseCtx()
	if PickString(c, []string{"only"}) != "only" {
		t.Fatal("PickString wrong")
	}
	if pick(rng.New(1), nil) != "" {
		t.Fatal("pick empty -> empty")
	}
}

func TestEmailSafeAndInitial(t *testing.T) {
	if emailSafe("O'Br-ien.") != "OBrien" {
		t.Fatalf("emailSafe = %q", emailSafe("O'Br-ien."))
	}
	// Non-ASCII is transliterated, not kept: a local part outside ASCII needs
	// SMTPUTF8, which most of the mail path does not speak.
	if got := emailSafe("Иван"); got != "Ivan" {
		t.Fatalf("emailSafe(Cyrillic) = %q, want %q", got, "Ivan")
	}
	// A script with no transliteration table folds away entirely, and the
	// caller substitutes a handle.
	if got := emailSafe("蕭哲瑋"); got != "" {
		t.Fatalf("emailSafe(Han) = %q, want empty", got)
	}
	if initial("") != "" {
		t.Fatal("initial of empty")
	}
	if initial("Xyz") != "X" {
		t.Fatal("initial wrong")
	}
}

func TestEmailLocalShapes(t *testing.T) {
	// Sweep seeds to hit all 8 switch arms.
	seen := map[string]bool{}
	for s := uint64(0); s < 200; s++ {
		seen[emailLocal(rng.New(s), "jane", "doe")] = true
	}
	if len(seen) < 8 {
		t.Fatalf("expected varied local-parts, got %d", len(seen))
	}
}

func TestIntFloatProviderDerivedAndDist(t *testing.T) {
	mkc := func(params map[string]string, sib any) Ctx {
		c := baseCtx()
		c.Params = params
		c.Field.Params = params
		c.Sibling = func(string) any { return sib }
		return c
	}
	// derived path
	di := mkc(map[string]string{"derive": "x", "slope": "2", "min": "0", "max": "1000000", "noise": "0.1"}, 100)
	if intProvider(di) == nil {
		t.Fatal("int derived nil")
	}
	if floatProvider(di) == nil {
		t.Fatal("float derived nil")
	}
	// dist path
	dd := mkc(map[string]string{"dist": "normal", "min": "0", "max": "100"}, nil)
	_ = intProvider(dd)
	_ = floatProvider(dd)
}

func TestDerivedAndToFloat(t *testing.T) {
	c := baseCtx()
	c.Params = map[string]string{}
	if _, ok := derived(c); ok {
		t.Fatal("no derive -> false")
	}
	c.Params["derive"] = "x"
	c.Sibling = func(string) any { return "notnumber" }
	if _, ok := derived(c); ok {
		t.Fatal("non-numeric sibling -> false")
	}
	for _, v := range []any{float64(1), float32(1), int(1), int32(1), int64(1)} {
		if _, ok := toFloat(v); !ok {
			t.Fatalf("toFloat(%T) failed", v)
		}
	}
	if _, ok := toFloat("x"); ok {
		t.Fatal("toFloat(string) should fail")
	}
}

func TestClampAndStrDefault(t *testing.T) {
	if clampInt(-5, 0, 10) != 0 || clampInt(50, 0, 10) != 10 || clampInt(5, 0, 10) != 5 {
		t.Fatal("clampInt wrong")
	}
	if clampFloat(-1, 0, 1) != 0 || clampFloat(2, 0, 1) != 1 || clampFloat(0.5, 0, 1) != 0.5 {
		t.Fatal("clampFloat wrong")
	}
	if strDefault("", "d") != "d" || strDefault("x", "d") != "x" {
		t.Fatal("strDefault wrong")
	}
}

func TestParamHelpers(t *testing.T) {
	if paramFloat(nil, "k", 3) != 3 {
		t.Fatal("nil params")
	}
	if paramFloat(map[string]string{"k": "bad"}, "k", 3) != 3 {
		t.Fatal("bad value default")
	}
	if paramFloat(map[string]string{"k": "2.5"}, "k", 3) != 2.5 {
		t.Fatal("good value")
	}
	if v, ok := floatParam(map[string]string{"k": "1.5"}, "k"); !ok || v != 1.5 {
		t.Fatal("floatParam good")
	}
	if _, ok := floatParam(map[string]string{}, "k"); ok {
		t.Fatal("floatParam missing")
	}
	if _, ok := floatParam(map[string]string{"k": "x"}, "k"); ok {
		t.Fatal("floatParam bad")
	}
	if paramInt(nil, "k", 7) != 7 {
		t.Fatal("paramInt nil")
	}
	if paramInt(map[string]string{"k": "bad"}, "k", 7) != 7 {
		t.Fatal("paramInt bad")
	}
}

func TestTimeSeriesProvider(t *testing.T) {
	ax := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	c := baseCtx()
	c.Params = map[string]string{
		"axis": "ts", "base": "10", "trend": "1", "amplitude": "5",
		"period": "24h", "noise": "0.5", "start": "2026-01-01", "min": "0", "max": "1000",
	}
	c.Sibling = func(n string) any {
		if n == "ts" {
			return ax
		}
		return nil
	}
	if v := timeSeriesProvider(c).(float64); v < 0 || v > 1000 {
		t.Fatalf("ts clamp: %v", v)
	}
	// no axis -> base
	c2 := baseCtx()
	c2.Params = map[string]string{"base": "42"}
	if timeSeriesProvider(c2).(float64) != 42 {
		t.Fatal("no axis should return base")
	}
	// axis blanked (sibling not a time) -> base
	c3 := baseCtx()
	c3.Params = map[string]string{"axis": "ts", "base": "7"}
	c3.Sibling = func(string) any { return "nope" }
	if timeSeriesProvider(c3).(float64) != 7 {
		t.Fatal("blanked axis should return base")
	}
}

func TestTimeProviderBranches(t *testing.T) {
	// after= predecessor
	prev := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := baseCtx()
	c.Field = &schema.Field{Name: "t", From: "created", Params: map[string]string{"gap": "1h..48h"}}
	c.Params = c.Field.Params
	c.Sibling = func(string) any { return prev }
	if got := timeProvider(c).(time.Time); !got.After(prev) {
		t.Fatal("after gap should follow predecessor")
	}
	// min & max window
	win := baseCtx()
	win.Params = map[string]string{"min": "2020-01-01", "max": "2020-12-31"}
	win.Field.Params = win.Params
	_ = timeProvider(win)
	// min only
	mn := baseCtx()
	mn.Params = map[string]string{"min": "2020-01-01"}
	_ = timeProvider(mn)
	// max only
	mx := baseCtx()
	mx.Params = map[string]string{"max": "2020-01-01"}
	_ = timeProvider(mx)
	// inverted window -> from
	inv := baseCtx()
	inv.Params = map[string]string{"min": "2020-12-31", "max": "2020-01-01"}
	_ = timeProvider(inv)
	// default (no params)
	_ = timeProvider(baseCtx())
}

func TestParseGap(t *testing.T) {
	if mn, mx := parseGap(""); mn != time.Minute || mx != 72*time.Hour {
		t.Fatal("empty default")
	}
	if mn, mx := parseGap("2h"); mn != 2*time.Hour || mx != 2*time.Hour {
		t.Fatal("single duration")
	}
	if _, mx := parseGap("1h..5h"); mx != 5*time.Hour {
		t.Fatal("range parse")
	}
	if mn, mx := parseGap("bad"); mn != time.Minute || mx != 72*time.Hour {
		t.Fatal("bad -> default")
	}
	if mn, mx := parseGap("5h..1h"); mx < mn {
		t.Fatal("inverted should clamp max to min")
	}
}

func TestBirthAndAge(t *testing.T) {
	c := baseCtx()
	c.Params = map[string]string{"min": "18", "max": "80"}
	if _, ok := birthDate(c).(time.Time); !ok {
		t.Fatal("birthDate type")
	}
	// span<=0 branch: min==max very tight
	c2 := baseCtx()
	c2.Params = map[string]string{"min": "-5", "max": "-5"} // clamps min to 0, max to 0
	_ = birthDate(c2)

	// age from birthdate sibling
	born := time.Date(1990, 6, 1, 0, 0, 0, 0, time.UTC)
	ca := baseCtx()
	ca.Field = &schema.Field{Name: "age", From: "dob", Params: map[string]string{}}
	ca.Sibling = func(string) any { return born }
	if age(ca).(int) <= 0 {
		t.Fatal("age from dob should be positive")
	}
	// yearsBetween birthday-not-yet path
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(1990, 12, 31, 0, 0, 0, 0, time.UTC)
	if yearsBetween(late, now) != 35 {
		t.Fatalf("yearsBetween = %d", yearsBetween(late, now))
	}
}
