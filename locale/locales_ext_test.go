package locale

import (
	"slices"
	"testing"
)

// Enriched locales must carry native (non-English-fallback) datasets.
func TestNativeDatasets(t *testing.T) {
	cases := map[string]struct{ company, product string }{
		"de_DE": {"Siemens", "Bratwurst"},
		"fr_FR": {"Renault", "Baguette"},
		"ja_JP": {"トヨタ", "寿司"},
		"tr_TR": {"Arçelik", "Baklava"},
		"ko_KR": {"삼성", "김치"},
		"zh_CN": {"华为", "茶"},
	}
	for name, want := range cases {
		l := Get(name)
		if !slices.Contains(l.Companies, want.company) {
			t.Errorf("%s: missing native company %q (got %v)", name, want.company, l.Companies)
		}
		if !slices.Contains(l.Products, want.product) {
			t.Errorf("%s: missing native product %q", name, want.product)
		}
		// Must not fall back to English defaults.
		if slices.Contains(l.Companies, "Acme Corp") {
			t.Errorf("%s: still using English company fallback", name)
		}
	}
}

// Non-enriched locales still work via English fallback (no empty datasets).
func TestFallbackDatasets(t *testing.T) {
	l := Get("et_EE") // Estonia, not enriched
	if len(l.Companies) == 0 || len(l.Streets) == 0 || len(l.Jobs) == 0 || len(l.Products) == 0 {
		t.Fatal("fallback locale has empty datasets")
	}
}
