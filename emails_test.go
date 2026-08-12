package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/emailx"
	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/locale"
)

// The claim the README makes about generated addresses, checked against the
// validator rather than against a regexp written here. Every locale, because
// the ones that fail are the ones whose names are not written in Latin.
func TestEmailsAreValidInEveryLocale(t *testing.T) {
	type row struct {
		Email string `synth:"email"`
	}
	for _, loc := range locale.Names() {
		for _, r := range synth.Make[row](50, synth.WithSeed(1), synth.WithLocale(loc)) {
			e, err := emailx.Parse(r.Email)
			if err != nil {
				t.Errorf("%s: %q does not parse: %v", loc, r.Email, err)
				continue
			}
			if !e.IsValid() {
				t.Errorf("%s: %q parses but is not a valid address", loc, r.Email)
			}
		}
	}
}

// A name that does not transliterate must still yield a usable mailbox, and a
// different one per record.
func TestEmailsVaryWhereNamesDoNotTransliterate(t *testing.T) {
	type row struct {
		Email string `synth:"email"`
	}
	for _, loc := range []string{"zh_CN", "ja_JP", "ko_KR", "th_TH", "ar_EG", "he_IL", "hi_IN"} {
		seen := map[string]bool{}
		for _, r := range synth.Make[row](200, synth.WithSeed(3), synth.WithLocale(loc)) {
			if strings.HasPrefix(r.Email, "@") {
				t.Fatalf("%s: %q has no mailbox", loc, r.Email)
			}
			seen[r.Email] = true
		}
		if len(seen) < 150 {
			t.Errorf("%s: 200 records produced only %d distinct addresses", loc, len(seen))
		}
	}
}

func TestEmailDisposableIsRecognisedAsDisposable(t *testing.T) {
	type row struct {
		Email string `synth:"email_disposable"`
	}
	for _, r := range synth.Make[row](100, synth.WithSeed(2)) {
		e, err := emailx.Parse(r.Email)
		if err != nil {
			t.Fatalf("%q does not parse: %v", r.Email, err)
		}
		if !e.IsDisposable() {
			t.Errorf("%q is not on the disposable list", r.Email)
		}
	}
}

// The provider must be the one that actually owns the address's domain.
func TestEmailProviderAgreesWithAddress(t *testing.T) {
	type account struct {
		Email    string `synth:"email"`
		Provider string `synth:"email_provider,from=Email"`
	}
	checked := 0
	for _, r := range synth.Make[account](200, synth.WithSeed(4), synth.WithLocale("en_US")) {
		e, err := emailx.Parse(r.Email)
		if err != nil {
			t.Fatalf("%q does not parse: %v", r.Email, err)
		}
		if r.Provider != e.ProviderID() {
			t.Errorf("%q: provider = %q, want %q", r.Email, r.Provider, e.ProviderID())
		}
		if r.Provider != "" {
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no record had a recognised provider, so nothing was really compared")
	}
}

// A domain no public provider owns gets an empty provider rather than a guess.
func TestEmailProviderIsEmptyForUnknownDomain(t *testing.T) {
	type odd struct {
		Email    string `synth:"lorem"`
		Provider string `synth:"email_provider,from=Email"`
	}
	for _, r := range synth.Make[odd](10, synth.WithSeed(5)) {
		if r.Provider != "" {
			t.Errorf("provider = %q for %q, want empty", r.Provider, r.Email)
		}
	}
}

// Normalization must fold the address to the form its own provider treats as
// one mailbox, and must be stable: normalizing twice changes nothing.
func TestEmailNormalizedIsCanonical(t *testing.T) {
	type account struct {
		Email     string `synth:"email"`
		Canonical string `synth:"email_normalized,from=Email"`
	}
	for _, r := range synth.Make[account](100, synth.WithSeed(6), synth.WithLocale("en_US")) {
		want, err := emailx.Normalize(r.Email)
		if err != nil {
			t.Fatalf("%q does not normalize: %v", r.Email, err)
		}
		if r.Canonical != want {
			t.Errorf("%q: normalized = %q, want %q", r.Email, r.Canonical, want)
		}
		again, err := emailx.Normalize(r.Canonical)
		if err != nil || again != r.Canonical {
			t.Errorf("%q is not stable under a second normalize: %q (%v)", r.Canonical, again, err)
		}
	}
}

func TestEmailSameSeedSameAddresses(t *testing.T) {
	type account struct {
		Email      string `synth:"email"`
		Provider   string `synth:"email_provider,from=Email"`
		Disposable string `synth:"email_disposable"`
	}
	a := synth.Make[account](50, synth.WithSeed(7))
	b := synth.Make[account](50, synth.WithSeed(7))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("record %d differs between runs: %+v vs %+v", i, a[i], b[i])
		}
	}
}
