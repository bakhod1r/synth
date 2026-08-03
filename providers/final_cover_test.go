package providers

import (
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

func TestLocaleValues(t *testing.T) {
	if v := LocaleValues("uz_UZ", schema.KindBodyPart); len(v) == 0 {
		t.Fatal("expected uz body parts")
	}
	if LocaleValues("zz_ZZ", schema.KindBodyPart) != nil {
		t.Fatal("unknown locale -> nil")
	}
	if LocaleValues("uz_UZ", schema.Kind("nosuchkind")) != nil {
		t.Fatal("unknown kind -> nil")
	}
}

func TestAirportNameFrom(t *testing.T) {
	c := baseCtx()
	c.Field = &schema.Field{Name: "apname", From: "code", Params: map[string]string{}}
	c.Sibling = func(string) any { return "JFK" }
	if got := airportName(c); !strings.Contains(got.(string), "Kennedy") {
		t.Fatalf("JFK -> %v", got)
	}
	// Unknown code falls through to a random name.
	c.Sibling = func(string) any { return "ZZZ" }
	if airportName(c) == nil {
		t.Fatal("unknown code should still return a name")
	}
}

func TestTitleWordTokens(t *testing.T) {
	c := baseCtx()
	for _, tok := range []string{"n", "year", "topic", "verb", "adj", "noun", "unknowntoken"} {
		if titleWord(c, tok) == "" {
			t.Fatalf("token %q produced empty", tok)
		}
	}
	if titleWord(c, "unknowntoken") != "unknowntoken" {
		t.Fatal("unknown token should echo")
	}
}

func TestGenerateCardExplicitBrand(t *testing.T) {
	num := generateCard(rng.New(1), "visa")
	if !luhnValid(num) || !strings.HasPrefix(num, "4") {
		t.Fatalf("visa card = %q", num)
	}
}

func TestCVVAmexAndDigits(t *testing.T) {
	c := baseCtx()
	c.Field = &schema.Field{Name: "cvv", From: "card", Params: map[string]string{}}
	c.Sibling = func(string) any { return "378282246310005" } // amex prefix
	if len(cvv(c).(string)) != 4 {
		t.Fatal("amex cvv should be 4 digits")
	}
	c2 := baseCtx()
	c2.Params = map[string]string{"digits": "4"}
	if len(cvv(c2).(string)) != 4 {
		t.Fatal("digits=4 override")
	}
}

func TestBalanceEdges(t *testing.T) {
	// Explicit negative floor sets negShare to 0; hi<lo clamps.
	c := baseCtx()
	c.Params = map[string]string{"min": "-100", "max": "-200"}
	if _, ok := balance(c).(float64); !ok {
		t.Fatal("balance type")
	}
}

func TestPasswordLengthAndHash(t *testing.T) {
	if n := passwordLength(rng.New(1), map[string]string{"length": "20"}, 3); n != 20 {
		t.Fatalf("explicit length = %d", n)
	}
	if n := passwordLength(rng.New(1), map[string]string{"min": "30", "max": "10"}, 2); n < 2 {
		t.Fatalf("min>max length = %d", n)
	}
	c := baseCtx()
	c.Params = map[string]string{"iterations": "1000"}
	if h := passwordHashProvider(c).(string); !strings.HasPrefix(h, "pbkdf2-sha256$1000$") {
		t.Fatalf("hash = %q", h)
	}
}

func TestPassphraseOptions(t *testing.T) {
	c := baseCtx()
	c.Params = map[string]string{"words": "3", "sep": ".", "capitalize": "true", "number": "true"}
	out := passphrase(c).(string)
	if !strings.Contains(out, ".") {
		t.Fatalf("passphrase = %q", out)
	}
}

func TestBoolParam(t *testing.T) {
	if boolParam(map[string]string{}, "k", true) != true {
		t.Fatal("missing -> fallback")
	}
	if boolParam(map[string]string{"k": "notabool"}, "k", true) != true {
		t.Fatal("bad -> fallback")
	}
	if boolParam(map[string]string{"k": "false"}, "k", true) != false {
		t.Fatal("good -> parsed")
	}
}

func TestBirthSpanAndYearsNegative(t *testing.T) {
	// span<=0: minAge huge so newest<=oldest is impossible normally; use age 0..0.
	c := baseCtx()
	c.Params = map[string]string{"min": "0", "max": "0"}
	if _, ok := birthDate(c).(time.Time); !ok {
		t.Fatal("birthDate type")
	}
	// yearsBetween where born is after now -> clamps to 0.
	now := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	if yearsBetween(future, now) != 0 {
		t.Fatal("negative years should clamp to 0")
	}
}

func TestWeightedChecksShortString(t *testing.T) {
	// weights longer than the string exercise the i>=len break.
	if weightedCheck("12", []int{1, 2, 3, 4, 5}) < 0 {
		t.Fatal("weightedCheck")
	}
	if weightedSumMod11("12", []int{1, 2, 3, 4, 5}) < 0 {
		t.Fatal("weightedSumMod11")
	}
}

func TestReachableParamBranches(t *testing.T) {
	// bool with true= share.
	bc := baseCtx()
	bc.Params = map[string]string{"true": "0.9"}
	_ = Get(schema.KindBool)(bc)

	// gender fallback when the record has no gender.
	gc := baseCtx()
	gc.Gender = ""
	if Get(schema.KindGender)(gc) == nil {
		t.Fatal("gender fallback nil")
	}

	// enum uniform (no weights, no zipf).
	ec := baseCtx()
	ec.Field.Choices = []string{"a", "b"}
	ec.Field.Weights = nil
	if enum(ec) == "" {
		t.Fatal("uniform enum empty")
	}

	// email derived from a full "First Last" sibling.
	mc := baseCtx()
	mc.Sibling = func(n string) any {
		if n == "__from__" {
			return "Jane Q Doe"
		}
		return nil
	}
	if _, ok := email(mc).(string); !ok {
		t.Fatal("email from full name")
	}

	// email whose __from__ sibling yields no usable name falls back to picks.
	fc := baseCtx()
	fc.Sibling = func(string) any { return "" }
	if _, ok := email(fc).(string); !ok {
		t.Fatal("email pick fallback")
	}

	// timeSeries without min/max returns the raw value.
	ts := baseCtx()
	axis := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ts.Params = map[string]string{"axis": "ts", "base": "5", "trend": "1"}
	ts.Sibling = func(string) any { return axis }
	if _, ok := timeSeriesProvider(ts).(float64); !ok {
		t.Fatal("ts no-clamp")
	}

	// generateCard with an unknown brand takes the random fallback.
	if !luhnValid(generateCard(rng.New(1), "nosuchbrand")) {
		t.Fatal("unknown brand fallback invalid")
	}

	// card() with a locale that has no CardBINs uses the global generator.
	nc := baseCtx()
	l := *nc.Locale
	l.CardBINs = nil
	nc.Locale = &l
	if !luhnValid(card(nc).(string)) {
		t.Fatal("no-BIN card invalid")
	}

	// generatePassword with every class disabled falls back to lowercase.
	pw := generatePassword(rng.New(1), map[string]string{
		"lower": "false", "upper": "false", "digits": "false", "symbols": "false",
	})
	if pw == "" {
		t.Fatal("all-classes-off password empty")
	}
}

func TestIINKZManySeeds(t *testing.T) {
	for s := uint64(0); s < 500; s++ {
		if len(iinKZ(rng.New(s))) != 13 {
			t.Fatalf("iinKZ wrong length at seed %d", s)
		}
	}
}
