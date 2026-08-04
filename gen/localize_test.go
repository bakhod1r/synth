package gen

import (
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

func genRows(t *testing.T, s *schema.Schema, loc string, n int) []map[string]any {
	t.Helper()
	e, err := Compile(s, loc)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	base := rng.New(42)
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = e.Record(base, i)
	}
	return out
}

func field(name string, k schema.Kind, params map[string]string) schema.Field {
	if params == nil {
		params = map[string]string{}
	}
	return schema.Field{Name: name, Kind: k, GoType: "string", Params: params}
}

// A de-localized field must leave the locale's alphabet behind while its
// localized neighbour in the same record keeps it.
func TestLocalizeFalseUsesBaseLocale(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("Local", schema.KindFirstName, nil),
		field("Base", schema.KindFirstName, map[string]string{"localize": "false"}),
	}}
	rows := genRows(t, s, "ru_RU", 40)

	cyrillic := func(v any) bool {
		for _, r := range v.(string) {
			if r >= 0x0400 && r <= 0x04FF {
				return true
			}
		}
		return false
	}
	var localCyr, baseCyr int
	for _, row := range rows {
		if cyrillic(row["Local"]) {
			localCyr++
		}
		if cyrillic(row["Base"]) {
			baseCyr++
		}
	}
	if localCyr == 0 {
		t.Fatalf("ru_RU first names were not Cyrillic at all: %v", rows[0]["Local"])
	}
	if baseCyr != 0 {
		t.Errorf("localize=false field stayed Russian in %d/%d rows", baseCyr, len(rows))
	}
}

// Default and localize=true must mean the same thing, and neither may be
// confused by a value the parser does not recognise.
func TestLocalizeDefaultsToOn(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params map[string]string
	}{
		{"absent", nil},
		{"true", map[string]string{"localize": "true"}},
		{"garbage", map[string]string{"localize": "maybe"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &schema.Schema{Fields: []schema.Field{field("Country", schema.KindCountry, tc.params)}}
			got := genRows(t, s, "uz_UZ", 1)[0]["Country"]
			if got == "United States" {
				t.Errorf("field was de-localized: got %q", got)
			}
		})
	}
}

// Two de-localized address fields must describe the same place, or the row is
// internally inconsistent — a city in one country with the postcode of another.
func TestLocalizeFalsePlaceStaysCoherent(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("City", schema.KindCity, map[string]string{"localize": "false"}),
		field("Postcode", schema.KindPostcode, map[string]string{"localize": "false"}),
		field("Region", schema.KindRegion, map[string]string{"localize": "false"}),
	}}
	base := locale.Get(baseLocale)
	for _, row := range genRows(t, s, "uz_UZ", 20) {
		city, region, postcode := row["City"], row["Region"], row["Postcode"]
		found := false
		for _, p := range base.Places {
			if p.City == city && p.Region == region && p.Postcode == postcode {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("de-localized place is not one en_US place: %v / %v / %v", city, region, postcode)
		}
	}
}

// Opting one field out must not shift the values of any other field: the rng
// stream is shared, so a de-localized column that consumed randomness would
// change every column after it.
func TestLocalizeFalseDoesNotDisturbOtherFields(t *testing.T) {
	with := &schema.Schema{Fields: []schema.Field{
		field("Opt", schema.KindCity, map[string]string{"localize": "false"}),
		field("Name", schema.KindFirstName, nil),
		field("Job", schema.KindJob, nil),
	}}
	without := &schema.Schema{Fields: []schema.Field{
		field("Opt", schema.KindCity, nil),
		field("Name", schema.KindFirstName, nil),
		field("Job", schema.KindJob, nil),
	}}
	a := genRows(t, with, "uz_UZ", 10)
	b := genRows(t, without, "uz_UZ", 10)
	for i := range a {
		for _, k := range []string{"Name", "Job"} {
			if a[i][k] != b[i][k] {
				t.Errorf("row %d field %s: %v with opt-out vs %v without", i, k, a[i][k], b[i][k])
			}
		}
	}
}

// Same input, same output — the de-localized place is derived, not drawn.
func TestLocalizeFalseIsDeterministic(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("City", schema.KindCity, map[string]string{"localize": "false"}),
	}}
	a := genRows(t, s, "uz_UZ", 10)
	b := genRows(t, s, "uz_UZ", 10)
	for i := range a {
		if a[i]["City"] != b[i]["City"] {
			t.Fatalf("row %d differs between runs: %v vs %v", i, a[i]["City"], b[i]["City"])
		}
	}
}

func TestLocalizeField(t *testing.T) {
	for _, tc := range []struct {
		val  string
		want bool
	}{
		{"false", false}, {"FALSE", false}, {" no ", false}, {"0", false}, {"off", false},
		{"true", true}, {"yes", true}, {"", true},
	} {
		f := &schema.Field{Params: map[string]string{"localize": tc.val}}
		if got := localizeField(f); got != tc.want {
			t.Errorf("localize=%q: got %v want %v", tc.val, got, tc.want)
		}
	}
	if !localizeField(&schema.Field{Params: nil}) {
		t.Error("a field with no params must stay localized")
	}
}

// A named locale overrides the dataset's, per field, without disturbing its
// neighbours or the rng stream they draw from.
func TestFieldLocaleOverride(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("Local", schema.KindFirstName, nil),
		field("JP", schema.KindFirstName, map[string]string{"locale": "ja_JP"}),
	}}
	rows := genRows(t, s, "uz_UZ", 20)

	plain := &schema.Schema{Fields: []schema.Field{field("Local", schema.KindFirstName, nil)}}
	want := genRows(t, plain, "uz_UZ", 20)

	jpNames := map[string]bool{}
	for i, row := range rows {
		if row["Local"] != want[i]["Local"] {
			t.Fatalf("row %d: an override shifted a neighbouring field: %v vs %v",
				i, row["Local"], want[i]["Local"])
		}
		jpNames[row["JP"].(string)] = true
	}

	ja := map[string]bool{}
	for _, n := range locale.Get("ja_JP").MaleFirst {
		ja[n] = true
	}
	for _, n := range locale.Get("ja_JP").FemaleFirst {
		ja[n] = true
	}
	for name := range jpNames {
		if !ja[name] {
			t.Errorf("locale=ja_JP produced %q, which is not a Japanese first name", name)
		}
	}
}

// locale= wins over localize=false: naming a locale is the more specific
// instruction, and honouring the vaguer one would silently drop it.
func TestFieldLocaleBeatsLocalizeFalse(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("Name", schema.KindFirstName, map[string]string{"locale": "ja_JP", "localize": "false"}),
	}}
	en := map[string]bool{}
	for _, n := range locale.Get("en_US").MaleFirst {
		en[n] = true
	}
	for _, row := range genRows(t, s, "uz_UZ", 20) {
		if en[row["Name"].(string)] {
			t.Fatalf("localize=false won over locale=ja_JP: %v", row["Name"])
		}
	}
}

// A typo in a locale name is a compile error, not a column that quietly
// generates in English.
func TestFieldLocaleUnknownIsAnError(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("Name", schema.KindFirstName, map[string]string{"locale": "zz_ZZ"}),
	}}
	if _, err := Compile(s, "uz_UZ"); err == nil {
		t.Fatal("Compile accepted an unknown locale name")
	}
}

// Naming the dataset's own locale is a no-op rather than a reroute through the
// place mapping, which would otherwise pin every row to one city.
func TestFieldLocaleSameAsDataset(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		field("City", schema.KindCity, map[string]string{"locale": "uz_UZ"}),
	}}
	plain := &schema.Schema{Fields: []schema.Field{field("City", schema.KindCity, nil)}}
	got, want := genRows(t, s, "uz_UZ", 10), genRows(t, plain, "uz_UZ", 10)
	for i := range got {
		if got[i]["City"] != want[i]["City"] {
			t.Fatalf("row %d: locale=uz_UZ on a uz_UZ dataset changed the value: %v vs %v",
				i, got[i]["City"], want[i]["City"])
		}
	}
}
