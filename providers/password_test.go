package providers_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

func classesOf(s string) (lower, upper, digit, symbol bool) {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			symbol = true
		}
	}
	return
}

// Every enabled class must appear in every password. A generator that only
// satisfies the policy most of the time makes policy tests flaky, and a flaky
// test gets muted rather than fixed.
func TestPasswordAlwaysSatisfiesItsPolicy(t *testing.T) {
	type Row struct {
		P string `synth:"password"`
	}
	for i, r := range synth.Make[Row](2000, synth.WithSeed(1)) {
		lower, upper, digit, symbol := classesOf(r.P)
		if !lower || !upper || !digit || !symbol {
			t.Fatalf("row %d: %q is missing a required class", i, r.P)
		}
		if len(r.P) < 12 || len(r.P) > 16 {
			t.Fatalf("row %d: %q is %d characters, want 12..16", i, r.P, len(r.P))
		}
	}
}

// The five named strengths must differ from each other in the ways their names
// promise, or the choice is decorative.
func TestPasswordStrengths(t *testing.T) {
	type pin struct {
		P string `synth:"password,strength=pin"`
	}
	type weak struct {
		P string `synth:"password,strength=weak"`
	}
	type medium struct {
		P string `synth:"password,strength=medium"`
	}
	type strong struct {
		P string `synth:"password,strength=strong"`
	}
	type veryStrong struct {
		P string `synth:"password,strength=very-strong"`
	}

	for i, r := range synth.Make[pin](300, synth.WithSeed(2)) {
		if strings.IndexFunc(r.P, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
			t.Fatalf("pin row %d: %q is not all digits", i, r.P)
		}
		if len(r.P) < 4 || len(r.P) > 6 {
			t.Fatalf("pin row %d: %q is %d digits", i, r.P, len(r.P))
		}
	}
	for i, r := range synth.Make[weak](300, synth.WithSeed(3)) {
		_, upper, digit, symbol := classesOf(r.P)
		if upper || digit || symbol {
			t.Fatalf("weak row %d: %q has more than lowercase", i, r.P)
		}
	}
	for i, r := range synth.Make[medium](300, synth.WithSeed(4)) {
		_, upper, digit, symbol := classesOf(r.P)
		if upper || symbol || !digit {
			t.Fatalf("medium row %d: %q should be lowercase plus digits", i, r.P)
		}
	}
	for i, r := range synth.Make[strong](300, synth.WithSeed(5)) {
		lower, upper, digit, symbol := classesOf(r.P)
		if !lower || !upper || !digit || symbol {
			t.Fatalf("strong row %d: %q should be letters and digits, no symbols", i, r.P)
		}
		if len(r.P) < 12 {
			t.Fatalf("strong row %d: %q is only %d characters", i, r.P, len(r.P))
		}
	}
	for i, r := range synth.Make[veryStrong](300, synth.WithSeed(6)) {
		lower, upper, digit, symbol := classesOf(r.P)
		if !lower || !upper || !digit || !symbol {
			t.Fatalf("very-strong row %d: %q is missing a class", i, r.P)
		}
		if len(r.P) < 16 {
			t.Fatalf("very-strong row %d: %q is only %d characters", i, r.P, len(r.P))
		}
	}
}

// Look-alike characters are excluded unless asked for: a fixture containing
// them gets transcribed wrongly during manual testing.
func TestPasswordAvoidsAmbiguousCharacters(t *testing.T) {
	type Row struct {
		P string `synth:"password"`
	}
	for i, r := range synth.Make[Row](1000, synth.WithSeed(7)) {
		if strings.ContainsAny(r.P, "0O1lI") {
			t.Fatalf("row %d: %q contains a look-alike character", i, r.P)
		}
	}
}

// An explicit param must override the named policy rather than be ignored.
func TestExplicitParamBeatsStrength(t *testing.T) {
	type Row struct {
		P string `synth:"password,strength=pin,length=20"`
	}
	for _, r := range synth.Make[Row](50, synth.WithSeed(8)) {
		if len(r.P) != 20 {
			t.Fatalf("length param ignored: %q is %d characters", r.P, len(r.P))
		}
	}
}

// A passphrase must be readable words in the record's own language.
func TestPassphraseUsesWords(t *testing.T) {
	type Row struct {
		P string `synth:"passphrase,words=4"`
	}
	for i, r := range synth.Make[Row](200, synth.WithSeed(9)) {
		parts := strings.Split(r.P, "-")
		if len(parts) != 4 {
			t.Fatalf("row %d: %q is not four words", i, r.P)
		}
		for _, w := range parts {
			if len(w) < 3 {
				t.Fatalf("row %d: %q contains a suspiciously short word", i, r.P)
			}
		}
	}
}

// A password hash must be self-describing and must not leak the plaintext.
func TestPasswordHashIsSelfDescribing(t *testing.T) {
	type Row struct {
		Password string `synth:"password"`
		Hash     string `synth:"passwordhash"`
	}
	for i, r := range synth.Make[Row](200, synth.WithSeed(10)) {
		parts := strings.Split(r.Hash, "$")
		if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
			t.Fatalf("row %d: %q is not a PHC-style hash", i, r.Hash)
		}
		if strings.Contains(r.Hash, r.Password) {
			t.Fatalf("row %d: the hash contains the plaintext", i)
		}
	}
}

// Two rows must not share a salt, or one rainbow table breaks every account.
func TestPasswordHashSaltsDiffer(t *testing.T) {
	type Row struct {
		Hash string `synth:"passwordhash"`
	}
	seen := map[string]bool{}
	rows := synth.Make[Row](500, synth.WithSeed(11))
	for _, r := range rows {
		salt := strings.Split(r.Hash, "$")[2]
		if seen[salt] {
			t.Fatalf("salt %q reused", salt)
		}
		seen[salt] = true
	}
}
