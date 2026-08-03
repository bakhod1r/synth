package mask

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestValueEmptyPassthrough(t *testing.T) {
	m := New("k", "en")
	if m.Value("email", "") != "" {
		t.Fatal("empty value must pass through")
	}
}

func TestValueStrategies(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "keepme", Strategy: Keep})
	m.Rule(Rule{Column: "dropme", Strategy: Drop})
	m.Rule(Rule{Column: "redactme", Strategy: Redact})
	m.Rule(Rule{Column: "fakeme", Strategy: Fake, Kind: schema.KindName})

	if got := m.Value("keepme", "abc"); got != "abc" {
		t.Fatalf("Keep = %q", got)
	}
	if got := m.Value("dropme", "abc"); got != "" {
		t.Fatalf("Drop = %q", got)
	}
	if got := m.Value("redactme", "ab-12"); got != "**-**" {
		t.Fatalf("Redact = %q", got)
	}
	if got := m.Value("fakeme", "Alice"); got == "" || got == "Alice" {
		t.Fatalf("Fake name = %q", got)
	}
}

func TestValueUnknownStrategyFallthrough(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "c", Strategy: Strategy("bogus")})
	if got := m.Value("c", "x"); got != "x" {
		t.Fatalf("unknown strategy should return value, got %q", got)
	}
}

func TestValueConsistency(t *testing.T) {
	m := New("k", "en")
	a := m.Value("email", "bob@corp.io")
	b := m.Value("email", "bob@corp.io")
	if a != b {
		t.Fatalf("same input mapped to %q and %q", a, b)
	}
	if strings.Contains(a, "bob@corp.io") {
		t.Fatal("output leaked original")
	}
}

func TestValueUnlistedPersonalFaked(t *testing.T) {
	m := New("k", "en")
	// Column name "email" is personal by synonym -> faked, not kept.
	out := m.Value("email", "jane@x.com")
	if out == "jane@x.com" || !strings.Contains(out, "@") {
		t.Fatalf("unlisted personal not faked properly: %q", out)
	}
}

func TestValueScrubEmbedded(t *testing.T) {
	m := New("k", "en")
	// "notes" is non-personal by name; embedded email must be scrubbed.
	out := m.Value("notes", "contact bob@corp.io now")
	if strings.Contains(out, "bob@corp.io") {
		t.Fatalf("embedded email not scrubbed: %q", out)
	}
	if !strings.HasPrefix(out, "contact ") {
		t.Fatalf("surrounding text not preserved: %q", out)
	}
}

func TestFakeShapePreservingFallback(t *testing.T) {
	m := New("k", "en")
	// No kind, non-personal value: shape-preserving replacement.
	// Mixed case + digits so all shapePreserving branches run.
	out := m.fake("code", "Ab-1x9", "")
	if len(out) != len("Ab-1x9") || out[2] != '-' {
		t.Fatalf("shape not preserved: %q", out)
	}
	if out == "Ab-1x9" {
		t.Fatalf("value unchanged: %q", out)
	}
}

func TestRedact(t *testing.T) {
	if got := redact("aB9 x!"); got != "*** *!" {
		t.Fatalf("redact = %q", got)
	}
}

func TestFormatKind(t *testing.T) {
	cases := []struct {
		v    string
		want schema.Kind
	}{
		{"a@b.com", schema.KindEmail},
		{"123-45-6789", schema.KindSSN},
		{"aa:bb:cc:dd:ee:ff", schema.KindMAC},
		{"10.0.0.1", schema.KindIPv4},
		{"GB82WEST12345698765432", schema.KindIBAN},
		{"4539578763621486", schema.KindCard}, // Luhn-valid
		{"+1 555 123 4567", schema.KindPhone},
	}
	for _, c := range cases {
		k, ok := formatKind(c.v)
		if !ok || k != c.want {
			t.Fatalf("formatKind(%q) = %v,%v want %v", c.v, k, ok, c.want)
		}
	}
	if _, ok := formatKind("just text"); ok {
		t.Fatal("plain text should not match")
	}
	// A non-Luhn 16-digit run is not a card; it falls through to phone shape.
	if k, ok := formatKind("1234567890123456"); !ok || k != schema.KindPhone {
		t.Fatalf("non-Luhn digit run = %v,%v want phone", k, ok)
	}
}

func TestPersonalKind(t *testing.T) {
	if k, ok := personalKind("email", "x"); !ok || k != schema.KindEmail {
		t.Fatal("email column not personal")
	}
	if k, ok := personalKind("notes", "a@b.com"); !ok || k != schema.KindEmail {
		t.Fatal("embedded-format not detected via value")
	}
	if k, ok := personalKind("salary_band", "x"); !ok || k != schema.KindLorem {
		t.Fatal("name-hint column not flagged")
	}
	if _, ok := personalKind("widget", "plain"); ok {
		t.Fatal("non-personal column wrongly flagged")
	}
}

func TestLuhnValid(t *testing.T) {
	if !luhnValid("4539578763621486") {
		t.Fatal("valid card rejected")
	}
	if luhnValid("4539578763621487") {
		t.Fatal("invalid card accepted")
	}
}

func TestLaplaceZeroScale(t *testing.T) {
	// scale<=0 returns 0 exactly (covered indirectly, asserted directly here).
	m := New("k", "en")
	m.Rule(Rule{Column: "n", Strategy: DP, Epsilon: 1, Sensitivity: 0})
	if got := m.Value("n", "42"); got != "42" {
		t.Fatalf("zero-sensitivity DP should not move value, got %q", got)
	}
}

func TestDPDefaultEpsilon(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "n", Strategy: DP, Sensitivity: 10}) // epsilon<=0 -> 1
	if got := m.Value("n", "100"); got == "" {
		t.Fatal("DP produced empty")
	}
}
