package providers

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
)

func TestFoldASCIITransliterates(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		// Latin with diacritics: the base letter survives.
		{"Zieliński", "Zielinski"},
		{"Pawłowski", "Pawlowski"},
		{"Şimşek", "Simsek"},
		{"Öztürk", "Ozturk"},
		{"Đoàn", "Doan"},
		{"Cecília", "Cecilia"},
		{"Zoë", "Zoe"},
		{"Straße", "Strasse"},
		{"Æther", "AEther"},
		// Cyrillic, Greek, Georgian.
		{"Иван", "Ivan"},
		{"Борисов", "Borisov"},
		{"Володимир", "Volodimir"},
		{"Мұқанов", "Muqanov"},
		{"Θεοδώρου", "Theodorou"},
		{"Παύλος", "Paulos"},
		{"Βασιλείου", "Vasileiou"},
		{"შენგელია", "shengelia"},
		// Punctuation that is legal in a name is not legal here.
		{"O'Br-ien.", "OBrien"},
		{"Anne Marie", "AnneMarie"},
		// No table: folds away entirely, and email() substitutes a handle.
		{"蕭哲瑋", ""},
		{"อาทิตย์", ""},
		{"مازن", ""},
	} {
		if got := foldASCII(tc.in); got != tc.want {
			t.Errorf("foldASCII(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Whatever comes out must be usable as a mailbox local part, which is the whole
// reason the folding exists.
func TestFoldASCIIOutputIsASCII(t *testing.T) {
	for _, in := range []string{
		"Владимир", "Φώτιος", "გურამ", "Sağyndyqov", "Đoàn", "Ćirković", "Þór",
	} {
		got := foldASCII(in)
		if got == "" {
			t.Errorf("foldASCII(%q) folded to nothing", in)
			continue
		}
		for _, r := range got {
			if r > 127 {
				t.Errorf("foldASCII(%q) = %q, which is not ASCII", in, got)
				break
			}
		}
	}
}

func TestLatinHandleIsPronounceableASCII(t *testing.T) {
	r := rng.New(1)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		h := latinHandle(r)
		if h == "" {
			t.Fatal("latinHandle produced an empty handle")
		}
		if strings.ToLower(h) != h {
			t.Errorf("latinHandle produced %q, want lower case", h)
		}
		for _, c := range h {
			if c < 'a' || c > 'z' {
				t.Fatalf("latinHandle produced %q, which is not plain lower-case ASCII", h)
			}
		}
		seen[h] = true
	}
	if len(seen) < 100 {
		t.Errorf("200 handles produced only %d distinct values", len(seen))
	}
}
