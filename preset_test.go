package synth_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bakhodir/synth"
)

// "Give me 100 fake transactions" must be one call.
func TestTransactionPreset(t *testing.T) {
	rows, err := synth.Transactions(100, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 100 {
		t.Fatalf("got %d rows, want 100", len(rows))
	}
	for _, col := range []string{"id", "amount", "currency", "direction", "created_at"} {
		if _, ok := rows[0][col]; !ok {
			t.Errorf("transaction has no %q column", col)
		}
	}
}

// Sensitive columns must be masked by default. A card number in a fixture
// reaches a ticket or a screenshot sooner or later.
func TestSensitiveColumnsAreMaskedByDefault(t *testing.T) {
	rows, err := synth.Transactions(200, synth.WithSeed(2))
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		card, _ := r["card_number"].(string)
		if !strings.Contains(card, "*") {
			t.Fatalf("row %d: card number %q is not masked", i, card)
		}
		if len(strings.TrimLeft(strings.TrimRight(card, "0123456789"), "0123456789")) == 0 {
			t.Fatalf("row %d: %q hides nothing", i, card)
		}
		id, _ := r["national_id"].(string)
		if len(id) != 64 || strings.Contains(id, "-") {
			t.Fatalf("row %d: national id %q is not hashed", i, id)
		}
	}
}

// Unmasked() must be an explicit decision, and must actually work.
func TestUnmaskedReturnsRawValues(t *testing.T) {
	rows, err := synth.Transactions(50, synth.WithSeed(3), synth.Unmasked())
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		card, _ := r["card_number"].(string)
		if strings.Contains(card, "*") {
			t.Fatalf("row %d: Unmasked() still returned a masked card", i)
		}
		if len(card) < 14 {
			t.Fatalf("row %d: %q is not a full card number", i, card)
		}
	}
}

// One unmasked call must not unmask later ones: the preset spec is cached and
// shared, so the mask must be stripped from a copy.
func TestUnmaskedDoesNotLeakIntoLaterCalls(t *testing.T) {
	if _, err := synth.Transactions(10, synth.WithSeed(4), synth.Unmasked()); err != nil {
		t.Fatal(err)
	}
	rows, err := synth.Transactions(10, synth.WithSeed(4))
	if err != nil {
		t.Fatal(err)
	}
	card, _ := rows[0]["card_number"].(string)
	if !strings.Contains(card, "*") {
		t.Fatalf("a previous Unmasked() call unmasked this one: %q", card)
	}
}

// Every preset must generate, and its spec must be readable and re-parseable
// so it works as a worked example.
func TestEveryPresetGeneratesAndRoundTrips(t *testing.T) {
	for _, p := range synth.Presets() {
		rows, err := synth.Generate(p, 20, synth.WithSeed(5))
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if len(rows) != 20 {
			t.Errorf("%s: got %d rows", p, len(rows))
		}
		spec, ok := synth.PresetSpec(p)
		if !ok || spec == "" {
			t.Errorf("%s has no readable spec", p)
			continue
		}
		if _, err := synth.YAMLBytes([]byte(spec)); err != nil {
			t.Errorf("%s spec does not parse: %v", p, err)
		}
	}
}

// A preset must be a starting point, not a cage.
func TestPresetSpecIsEditable(t *testing.T) {
	spec, err := synth.Spec(synth.PresetUser)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := spec.GenerateN(5, synth.WithLocale("uz_UZ"), synth.WithSeed(6))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows", len(rows))
	}
}

// The mask modes must each do what their name says.
func TestMaskModes(t *testing.T) {
	spec := `name: t
count: 100
seed: 9
fields:
  raw:     { kind: card }
  partial: { kind: card, mask: partial }
  hashed:  { kind: card, mask: hash }
  redact:  { kind: card, mask: redact }
  token:   { kind: card, mask: token }
`
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		if strings.Contains(r["raw"].(string), "*") {
			t.Fatalf("row %d: an unmasked column was masked", i)
		}
		if !strings.Contains(r["partial"].(string), "*") {
			t.Fatalf("row %d: partial mask did not apply", i)
		}
		if len(r["hashed"].(string)) != 64 {
			t.Fatalf("row %d: hash is not SHA-256 hex", i)
		}
		if r["redact"] != "[REDACTED]" {
			t.Fatalf("row %d: redact gave %q", i, r["redact"])
		}
		if !strings.HasPrefix(r["token"].(string), "tok_") {
			t.Fatalf("row %d: token gave %q", i, r["token"])
		}
	}
}

// A masked card keeps its first four digits (the BIN identifies the issuer,
// not the cardholder) and its last four, and stars everything between.
func TestCardMaskKeepsBothEnds(t *testing.T) {
	raw, err := synth.Transactions(50, synth.WithSeed(11), synth.Unmasked())
	if err != nil {
		t.Fatal(err)
	}
	masked, err := synth.Transactions(50, synth.WithSeed(11))
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		full := raw[i]["card_number"].(string)
		got := masked[i]["card_number"].(string)
		if len(got) != len(full) {
			t.Fatalf("row %d: mask changed the length: %q vs %q", i, got, full)
		}
		if got[:4] != full[:4] {
			t.Fatalf("row %d: %q does not keep the leading four of %q", i, got, full)
		}
		if got[len(got)-4:] != full[len(full)-4:] {
			t.Fatalf("row %d: %q does not keep the trailing four of %q", i, got, full)
		}
		middle := got[4 : len(got)-4]
		if strings.Trim(middle, "*") != "" {
			t.Fatalf("row %d: %q leaks digits in the middle", i, got)
		}
	}
}

// The same value masked in two columns must not produce the same digest.
// Otherwise anyone holding both tables joins on the masked value and re-links
// exactly the rows the mask was meant to separate.
func TestHashMaskIsScopedToItsColumn(t *testing.T) {
	spec := `name: t
count: 100
seed: 21
fields:
  raw:  { kind: ssn }
  a:    { kind: ssn, from: raw, mask: hash }
  b:    { kind: ssn, from: raw, mask: hash }
`
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rows {
		if r["a"] == r["b"] {
			t.Fatalf("row %d: the same value hashed identically in two columns (%v)", i, r["a"])
		}
	}
}

// Within one column the mask must stay stable, or the column cannot be joined
// on and a golden test cannot pin it.
func TestHashMaskIsStableWithinItsColumn(t *testing.T) {
	a, _ := synth.Transactions(50, synth.WithSeed(22))
	b, _ := synth.Transactions(50, synth.WithSeed(22))
	for i := range a {
		if a[i]["national_id"] != b[i]["national_id"] {
			t.Fatalf("row %d: the masked value changed between runs", i)
		}
	}
}

// A salt must change the digest, so two datasets built from the same values do
// not share masked identifiers.
func TestSaltChangesTheDigest(t *testing.T) {
	rows := digestRows(t, `  raw: { kind: ssn }
  plain:  { kind: ssn, from: raw, mask: hash }
  salted: { kind: ssn, from: raw, mask: hash, salt: pepper }
`)
	for i, r := range rows {
		if r["plain"] == r["salted"] {
			t.Fatalf("row %d: the salt did not change the digest", i)
		}
	}
}

// A secret key must produce an HMAC, not the plain hash. Without a key, a short
// value like a national identifier can be enumerated until the digest matches.
func TestSecretKeyChangesTheDigest(t *testing.T) {
	rows := digestRows(t, `  raw: { kind: ssn }
  plain: { kind: ssn, from: raw, mask: hash }
  keyed: { kind: ssn, from: raw, mask: hash, secret: k1 }
  other: { kind: ssn, from: raw, mask: hash, secret: k2 }
`)
	for i, r := range rows {
		if r["keyed"] == r["plain"] {
			t.Fatalf("row %d: secret= produced the unkeyed hash", i)
		}
		if r["keyed"] == r["other"] {
			t.Fatalf("row %d: two different keys produced the same digest", i)
		}
	}
}

// digest= must shorten the value, but never below the point where collisions
// stop being theoretical.
func TestDigestLength(t *testing.T) {
	rows := digestRows(t, `  short: { kind: ssn, mask: hash, digest: 20 }
  floor: { kind: ssn, mask: hash, digest: 4 }
  over:  { kind: ssn, mask: hash, digest: 999 }
`)
	for i, r := range rows {
		if got := len(r["short"].(string)); got != 20 {
			t.Fatalf("row %d: digest 20 gave %d characters", i, got)
		}
		if got := len(r["floor"].(string)); got != 16 {
			t.Fatalf("row %d: digest 4 gave %d characters, want the 16 floor", i, got)
		}
		if got := len(r["over"].(string)); got != 64 {
			t.Fatalf("row %d: an oversized digest truncated to %d", i, got)
		}
	}
}

func digestRows(t *testing.T, fields string) []map[string]any {
	t.Helper()
	y, err := synth.YAMLBytes([]byte("name: t\ncount: 50\nseed: 31\nfields:\n" + fields))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// Every date bound in every preset must actually bind.
//
// The user preset shipped with dates of birth in 2025 despite asking for
// 1960..2006, because an unparseable bound is ignored rather than rejected.
// Counting rows and checking columns exist could not catch that; only reading
// the values against the spec can.
func TestPresetDateBoundsHold(t *testing.T) {
	bound := regexp.MustCompile(`(\w+):\s*\{[^}]*kind:\s*time[^}]*min:\s*(\S+?),\s*max:\s*(\S+?)\s*\}`)
	checked := 0
	for _, p := range synth.Presets() {
		spec, _ := synth.PresetSpec(p)
		rows, err := synth.Generate(p, 200, synth.WithSeed(13))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, m := range bound.FindAllStringSubmatch(spec, -1) {
			col, lo, hi := m[1], mustDate(t, m[2]), mustDate(t, m[3])
			checked++
			for i, r := range rows {
				got, ok := r[col].(time.Time)
				if !ok {
					t.Fatalf("%s.%s is %T, not a time", p, col, r[col])
				}
				if got.Before(lo) || got.After(hi) {
					t.Fatalf("%s.%s row %d: %s is outside %s..%s",
						p, col, i, got.Format("2006-01-02"),
						lo.Format("2006-01-02"), hi.Format("2006-01-02"))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no preset date bounds were checked — the pattern stopped matching")
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", strings.Trim(s, `"'`))
	if err != nil {
		t.Fatalf("cannot read %q as a date: %v", s, err)
	}
	return v
}

// A short value must be starred completely: revealing four of five characters
// is not masking.
func TestPartialMaskDoesNotRevealShortValues(t *testing.T) {
	spec := `name: t
count: 50
seed: 3
fields:
  code: { kind: cvv, mask: partial }
`
	y, _ := synth.YAMLBytes([]byte(spec))
	rows, _ := y.Generate()
	for i, r := range rows {
		v := r["code"].(string)
		if strings.Trim(v, "*") != "" {
			t.Fatalf("row %d: short value %q leaked characters", i, v)
		}
	}
}

// The same input must mask to the same output, or a masked column cannot be
// joined on.
func TestMaskIsDeterministic(t *testing.T) {
	a, _ := synth.Transactions(50, synth.WithSeed(7))
	b, _ := synth.Transactions(50, synth.WithSeed(7))
	for i := range a {
		if a[i]["national_id"] != b[i]["national_id"] {
			t.Fatalf("row %d: masked value differs between runs", i)
		}
	}
}
