package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

// Generation must satisfy the spec's invariants. Without enforcement an
// unconstrained generator produces a total that disagrees with its parts —
// which is exactly the failure a per-column faker cannot see.
func TestGeneratedRecordsSatisfySpecConstraints(t *testing.T) {
	spec := `name: orders
count: 500
seed: 7
fields:
  subtotal: { kind: amount, min: 1, max: 1000 }
  tax:      { kind: amount, min: 0, max: 200 }
  total:    { kind: amount, min: 0, max: 5 }
  opened:   { kind: time }
  closed:   { kind: time }
constraints:
  - {kind: sum, parts: [subtotal, tax], whole: total}
  - {kind: ordering, left: opened, right: closed}
`
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 500 {
		t.Fatalf("got %d rows, want 500", len(rows))
	}
	for i, r := range rows {
		for _, c := range y.Constraints() {
			if !c.Holds(r) {
				t.Fatalf("row %d violates %q: %v", i, c, r)
			}
		}
	}
	// The total's declared max of 5 is deliberately impossible; the constraint
	// must win, proving enforcement actually ran.
	if total, ok := rows[0]["total"].(float64); ok && total <= 5 {
		t.Fatalf("total %v looks like the raw generated value, not the derived sum", total)
	}
}

// Contradictory constraints must be reported. Silently emitting rows that
// violate the spec would be the worst outcome: the data looks authoritative
// and is not.
func TestContradictoryConstraintsAreReported(t *testing.T) {
	spec := `name: t
count: 10
seed: 1
fields:
  a: { kind: int, min: 0, max: 100 }
  b: { kind: int, min: 0, max: 100 }
constraints:
  - {kind: range, left: a, lo: 90, hi: 100}
  - {kind: range, left: b, lo: 0, hi: 10}
  - {kind: ordering, left: a, right: b}
`
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.Generate()
	if err == nil {
		t.Fatal("expected an error for constraints that cannot all hold")
	}
	if !strings.Contains(err.Error(), "contradict") {
		t.Fatalf("error does not explain the contradiction: %v", err)
	}
}

// A spec with no constraints must behave exactly as before.
func TestNoConstraintsIsUnchanged(t *testing.T) {
	spec := `name: t
count: 20
seed: 3
fields:
  name:  { kind: name }
  email: { kind: email }
`
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 20 {
		t.Fatalf("got %d rows, want 20", len(rows))
	}
	if !strings.Contains(rows[0]["email"].(string), "@") {
		t.Fatalf("generation changed: %v", rows[0])
	}
}
