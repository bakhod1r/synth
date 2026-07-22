package providers_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

func pw(t *testing.T, field string) []string {
	t.Helper()
	y, err := synth.YAMLBytes([]byte("name: t\ncount: 200\nseed: 8\nfields:\n  p: " + field + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r["p"].(string)
	}
	return out
}

func hasAny(s, set string) bool { return strings.ContainsAny(s, set) }

// Every option the workbench offers has to reach the generator. A control that
// silently does nothing is worse than no control.
func TestPasswordOptions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		check func(t *testing.T, v string)
	}{
		{"exact length", `{ kind: password, length: 18 }`, func(t *testing.T, v string) {
			if len(v) != 18 {
				t.Fatalf("%q is %d characters, want 18", v, len(v))
			}
		}},
		{"length range", `{ kind: password, min: 20, max: 24 }`, func(t *testing.T, v string) {
			if len(v) < 20 || len(v) > 24 {
				t.Fatalf("%q is %d characters, want 20..24", v, len(v))
			}
		}},
		{"no symbols", `{ kind: password, symbols: false }`, func(t *testing.T, v string) {
			if hasAny(v, "!@#$%^&*()-_=+[]{};:,.?") {
				t.Fatalf("%q contains a symbol", v)
			}
		}},
		{"no uppercase", `{ kind: password, upper: false }`, func(t *testing.T, v string) {
			if v != strings.ToLower(v) {
				t.Fatalf("%q contains an uppercase letter", v)
			}
		}},
		{"no digits", `{ kind: password, digits: false }`, func(t *testing.T, v string) {
			if hasAny(v, "0123456789") {
				t.Fatalf("%q contains a digit", v)
			}
		}},
		{"pin is digits only", `{ kind: password, strength: pin }`, func(t *testing.T, v string) {
			if strings.Trim(v, "0123456789") != "" {
				t.Fatalf("pin %q is not all digits", v)
			}
			if len(v) < 4 || len(v) > 6 {
				t.Fatalf("pin %q is %d characters, want 4..6", v, len(v))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range pw(t, tc.field) {
				tc.check(t, v)
			}
		})
	}
}

// An explicit option must beat the named strength: the policy is a starting
// point, not a cage.
func TestExplicitOptionBeatsStrength(t *testing.T) {
	for _, v := range pw(t, `{ kind: password, strength: pin, length: 30 }`) {
		if len(v) != 30 {
			t.Fatalf("length: 30 lost to strength: pin — got %d characters in %q", len(v), v)
		}
	}
}

// Ambiguous look-alikes are excluded by default, because a fixture containing
// them gets transcribed wrongly during manual testing.
func TestAmbiguousCharactersAreOptIn(t *testing.T) {
	for _, v := range pw(t, `{ kind: password, length: 24 }`) {
		if hasAny(v, "0O1lI") {
			t.Fatalf("%q contains an ambiguous character by default", v)
		}
	}
	found := false
	for _, v := range pw(t, `{ kind: password, ambiguous: true, length: 24 }`) {
		if hasAny(v, "0O1lI") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ambiguous: true never produced an ambiguous character")
	}
}

// Every enabled class must actually appear, or a policy test built on the
// fixture fails at random and gets muted.
func TestEnabledClassesAlwaysAppear(t *testing.T) {
	for _, v := range pw(t, `{ kind: password, length: 12 }`) {
		for _, c := range []struct {
			label, set string
		}{
			{"lowercase", "abcdefghijkmnpqrstuvwxyz"},
			{"uppercase", "ABCDEFGHJKLMNPQRSTUVWXYZ"},
			{"digit", "23456789"},
			{"symbol", "!@#$%^&*()-_=+[]{};:,.?"},
		} {
			if !hasAny(v, c.set) {
				t.Fatalf("%q has no %s", v, c.label)
			}
		}
	}
}
