package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/locale"
)

func TestNewFieldTypes(t *testing.T) {
	type Rec struct {
		ID       int
		Street   string
		Color    string
		Job      string
		Product  string
		Gender   string
		MAC      string `synth:"mac"`
		HexColor string `synth:"hexcolor"`
	}
	recs := synth.Make[Rec](50, synth.WithSeed(1))
	for _, r := range recs {
		if r.Street == "" || r.Color == "" || r.Job == "" || r.Product == "" || r.Gender == "" {
			t.Fatalf("empty breadth field: %+v", r)
		}
		if strings.Count(r.MAC, ":") != 5 {
			t.Fatalf("bad mac %q", r.MAC)
		}
		if !strings.HasPrefix(r.HexColor, "#") || len(r.HexColor) != 7 {
			t.Fatalf("bad hex %q", r.HexColor)
		}
	}
}

func TestCustomTypeRegisterSet(t *testing.T) {
	synth.RegisterSet("cinema", "Inception", "Interstellar", "Tenet", "Dune")
	type Ticket struct {
		ID     int
		Cinema string `synth:"cinema"`
	}
	valid := map[string]bool{"Inception": true, "Interstellar": true, "Tenet": true, "Dune": true}
	for _, tk := range synth.Make[Ticket](30, synth.WithSeed(2)) {
		if !valid[tk.Cinema] {
			t.Fatalf("unexpected cinema value %q", tk.Cinema)
		}
	}
}

func TestCustomTypeRegisterFunc(t *testing.T) {
	synth.Register("rating", func(r synth.R) any {
		return r.IntRange(1, 5)
	})
	type Review struct {
		ID     int
		Rating int `synth:"rating"`
	}
	for _, rv := range synth.Make[Review](40, synth.WithSeed(3)) {
		if rv.Rating < 1 || rv.Rating > 5 {
			t.Fatalf("rating out of range: %d", rv.Rating)
		}
	}
}

// A sample of the ~50 locales must resolve and stay phone-coherent.
func TestManyLocales(t *testing.T) {
	type P struct {
		ID    int
		Name  string
		Phone string
	}
	cases := map[string]string{
		"ja_JP": "+81", "de_DE": "+49", "ru_RU": "+7", "tr_TR": "+90",
		"ko_KR": "+82", "fr_FR": "+33", "ar_SA": "+966", "hi_IN": "+91",
	}
	for loc, code := range cases {
		recs := synth.Make[P](5, synth.WithSeed(9), synth.WithLocale(loc))
		for _, r := range recs {
			if r.Name == "" {
				t.Fatalf("%s: empty name", loc)
			}
			if !strings.HasPrefix(r.Phone, code) {
				t.Fatalf("%s: phone %q lacks %q", loc, r.Phone, code)
			}
		}
	}
}

func TestLocaleCount(t *testing.T) {
	if n := len(locale.Names()); n < 50 {
		t.Fatalf("expected >=50 locales, got %d", n)
	}
}
