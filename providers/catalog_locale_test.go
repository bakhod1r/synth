package providers_test

import (
	"testing"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/providers"
	"github.com/bakhodir/synth/schema"
)

type localeRow struct {
	Food      string `synth:"food"`
	Color     string `synth:"color"`
	Weekday   string `synth:"weekday"`
	Month     string `synth:"month"`
	Fruit     string `synth:"fruit"`
	Vegetable string `synth:"vegetable"`
	Drink     string `synth:"drink"`
	Animal    string `synth:"animal"`
	Season    string `synth:"season"`
	Weather   string `synth:"weather"`
}

// A locale with its own datasets must actually return them, not English with
// a different name attached.
func TestCatalogFollowsLocale(t *testing.T) {
	for _, code := range []string{"uz_UZ", "ru_RU", "tr_TR", "de_DE", "ja_JP", "pl_PL"} {
		t.Run(code, func(t *testing.T) {
			local := synth.Make[localeRow](300, synth.WithSeed(1), synth.WithLocale(code))
			english := synth.Make[localeRow](300, synth.WithSeed(1), synth.WithLocale("en_US"))

			same := 0
			for i := range local {
				if local[i].Food == english[i].Food {
					same++
				}
			}
			if same > len(local)/10 {
				t.Errorf("%s food matched en_US in %d of %d rows — not localized",
					code, same, len(local))
			}
			// Every localized field must be non-empty; an empty dataset would
			// silently produce blank columns.
			r := local[0]
			for name, v := range map[string]string{
				"food": r.Food, "color": r.Color, "weekday": r.Weekday, "month": r.Month,
				"fruit": r.Fruit, "vegetable": r.Vegetable, "drink": r.Drink,
				"animal": r.Animal, "season": r.Season, "weather": r.Weather,
			} {
				if v == "" {
					t.Errorf("%s: %s came back empty", code, name)
				}
			}
		})
	}
}

// A locale with no dataset for a kind must still generate — falling back to
// English is the documented behaviour, not an error.
func TestUncoveredLocaleFallsBack(t *testing.T) {
	rows := synth.Make[localeRow](50, synth.WithSeed(2), synth.WithLocale("fi_FI"))
	if rows[0].Food == "" {
		t.Fatal("an uncovered locale produced an empty food value instead of falling back")
	}
}

// Coverage must be introspectable, so the UI and the docs can state it rather
// than implying that choosing a locale translates everything.
func TestCoverageIsReportedHonestly(t *testing.T) {
	localized := providers.LocalizedKinds()
	if len(localized) == 0 {
		t.Fatal("no kinds report locale coverage")
	}
	has := map[schema.Kind]bool{}
	for _, k := range localized {
		has[k] = true
	}
	for _, k := range []schema.Kind{
		schema.KindFood, schema.KindColor, schema.KindWeekday, schema.KindMonth,
		schema.KindFruit, schema.KindVegetable, schema.KindDrink, schema.KindAnimal,
		schema.KindSeason, schema.KindWeather,
	} {
		if !has[k] {
			t.Errorf("%s has locale data but is not reported as localized", k)
		}
	}
	// A type that is the same word everywhere must not claim coverage.
	if has[schema.KindSuperhero] {
		t.Error("superhero returns English values in every locale; claiming " +
			"otherwise would mislead the user")
	}

	if got := providers.LocalesFor(schema.KindFood); len(got) < 5 {
		t.Errorf("food reports only %d locales: %v", len(got), got)
	}
	if got := providers.LocalesFor(schema.KindSuperhero); len(got) != 0 {
		t.Errorf("superhero should report no locale coverage, got %v", got)
	}
}

// A weekday list that is not exactly seven entries is a data-entry mistake,
// and the same goes for months and seasons.
func TestFixedSizeSetsAreComplete(t *testing.T) {
	sizes := map[schema.Kind]int{
		schema.KindWeekday: 7,
		schema.KindMonth:   12,
		schema.KindSeason:  4,
	}
	for kind, want := range sizes {
		for _, code := range providers.LocalesFor(kind) {
			if got := len(providers.LocaleValues(code, kind)); got != want {
				t.Errorf("%s %s has %d entries, want %d", code, kind, got, want)
			}
		}
	}
}

// A duplicate inside a locale list quietly skews the distribution.
func TestLocaleListsHaveNoDuplicates(t *testing.T) {
	for _, kind := range providers.LocalizedKinds() {
		for _, code := range providers.LocalesFor(kind) {
			seen := map[string]bool{}
			for _, v := range providers.LocaleValues(code, kind) {
				if v == "" {
					t.Errorf("%s %s: empty value", code, kind)
				}
				if seen[v] {
					t.Errorf("%s %s: %q appears twice", code, kind, v)
				}
				seen[v] = true
			}
		}
	}
}
