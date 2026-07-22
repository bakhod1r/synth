package synth_test

import (
	"strings"
	"testing"

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
