package providers_test

import (
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// The two "language" kinds mean different things, and the clear aliases must
// return the same values as the names they replace.
func TestLanguageAliasesMatchTheirOriginals(t *testing.T) {
	type Row struct {
		Prog  string `synth:"programminglanguage"`
		Human string `synth:"humanlanguage"`
	}
	type Old struct {
		Prog  string `synth:"language"`
		Human string `synth:"languagename"`
	}
	a := synth.Make[Row](200, synth.WithSeed(4))
	b := synth.Make[Old](200, synth.WithSeed(4))
	for i := range a {
		if a[i].Prog != b[i].Prog || a[i].Human != b[i].Human {
			t.Fatalf("alias diverged at row %d: %+v vs %+v", i, a[i], b[i])
		}
	}
	// They must not be the same pool: one is Go, the other is Uzbek.
	progs, humans := map[string]bool{}, map[string]bool{}
	for _, r := range a {
		progs[r.Prog] = true
		humans[r.Human] = true
	}
	for p := range progs {
		if humans[p] {
			t.Fatalf("%q appears as both a programming and a spoken language", p)
		}
	}
}

// Body parts must be generated, and must follow the locale.
func TestBodyPartFollowsLocale(t *testing.T) {
	type Row struct {
		Part string `synth:"bodypart"`
	}
	uz := synth.Make[Row](200, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	en := synth.Make[Row](200, synth.WithSeed(1), synth.WithLocale("en_US"))
	if uz[0].Part == "" {
		t.Fatal("bodypart came back empty")
	}
	same := 0
	for i := range uz {
		if uz[i].Part == en[i].Part {
			same++
		}
	}
	if same > len(uz)/10 {
		t.Fatalf("uz_UZ body parts matched English in %d of %d rows", same, len(uz))
	}
	if got := providers.LocalesFor(schema.KindBodyPart); len(got) < 5 {
		t.Fatalf("bodypart reports only %d locales: %v", len(got), got)
	}
}
