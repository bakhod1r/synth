package constraint_test

import (
	"testing"

	"github.com/bakhod1r/synth/constraint"
)

// Mining must find an ordering that genuinely holds, and must not report one
// that only looks plausible over part of the sample.
func TestMineOrdering(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 200; i++ {
		rows = append(rows, map[string]any{
			"created_at": float64(i),
			"updated_at": float64(i + 5), // always later
			"noise":      float64(200 - i),
		})
	}
	cs := constraint.Mine(rows, 0.99)

	found := false
	for _, c := range cs {
		if c.Kind != constraint.Ordering {
			continue
		}
		if c.Left == "created_at" && c.Right == "updated_at" {
			found = true
			if c.Support != 200 {
				t.Errorf("support is %d, want 200", c.Support)
			}
		}
		if c.Left == "noise" || c.Right == "noise" {
			t.Errorf("mined a spurious ordering involving noise: %s", c)
		}
	}
	if !found {
		t.Fatalf("did not mine created_at <= updated_at; got %v", cs)
	}
}

// Two columns that are always equal are a duplication, not an ordering, and
// reporting both directions would be noise.
func TestMineIgnoresAlwaysEqualColumns(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 100; i++ {
		rows = append(rows, map[string]any{"a": float64(i), "b": float64(i)})
	}
	for _, c := range constraint.Mine(rows, 0.99) {
		if c.Kind == constraint.Ordering {
			t.Fatalf("mined an ordering between always-equal columns: %s", c)
		}
	}
}

// A sum invariant must survive floating-point representation error.
func TestMineSum(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 100; i++ {
		sub, tax := float64(i)*1.1, float64(i)*0.07
		rows = append(rows, map[string]any{
			"subtotal": sub, "tax": tax, "total": sub + tax,
		})
	}
	for _, c := range constraint.Mine(rows, 0.99) {
		if c.Kind == constraint.SumEquals && c.Whole == "total" && len(c.Parts) == 2 {
			return
		}
	}
	t.Fatalf("did not mine subtotal + tax = total; got %v", constraint.Mine(rows, 0.99))
}

// An implication must be mined when the trigger group is real evidence.
func TestMineImplication(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 300; i++ {
		r := map[string]any{"status": "paid", "refund_at": nil}
		if i%3 == 0 {
			r = map[string]any{"status": "refunded", "refund_at": "2026-01-01T00:00:00Z"}
		}
		rows = append(rows, r)
	}
	for _, c := range constraint.Mine(rows, 0.99) {
		if c.Kind == constraint.Implication && c.Equals == "refunded" && c.Then == "refund_at" {
			if !c.Exclusive {
				t.Error("refund_at appears only for refunded rows, so the rule is exclusive")
			}
			return
		}
	}
	t.Fatalf("did not mine status=refunded => refund_at non-null; got %v", constraint.Mine(rows, 0.99))
}

// A column that is always populated has nothing to do with any trigger, so no
// implication should be invented for it.
func TestMineRejectsVacuousImplication(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 300; i++ {
		status := "paid"
		if i%3 == 0 {
			status = "refunded"
		}
		rows = append(rows, map[string]any{"status": status, "created_at": "2026-01-01T00:00:00Z"})
	}
	for _, c := range constraint.Mine(rows, 0.99) {
		if c.Kind == constraint.Implication {
			t.Fatalf("invented a rule about an always-populated column: %s", c)
		}
	}
}

// A trigger seen only a handful of times is not evidence.
func TestMineRejectsTinyTriggerGroup(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 300; i++ {
		r := map[string]any{"status": "paid", "refund_at": nil}
		if i < 3 { // only three rows carry the trigger
			r = map[string]any{"status": "refunded", "refund_at": "2026-01-01T00:00:00Z"}
		}
		rows = append(rows, r)
	}
	for _, c := range constraint.Mine(rows, 0.99) {
		if c.Kind == constraint.Implication && c.Equals == "refunded" {
			t.Fatalf("mined a rule from %d rows: %s", 3, c)
		}
	}
}

// Enforcement must leave every constraint satisfied.
func TestEnforceRepairsRecords(t *testing.T) {
	cs := []constraint.Constraint{
		{Kind: constraint.Ordering, Left: "created_at", Right: "updated_at"},
		{Kind: constraint.SumEquals, Parts: []string{"subtotal", "tax"}, Whole: "total"},
		{Kind: constraint.Range, Left: "age", Lo: 18, Hi: 65},
	}
	rec := map[string]any{
		"created_at": 100.0, "updated_at": 20.0, // violates ordering
		"subtotal": 10.0, "tax": 1.0, "total": 999.0, // violates sum
		"age": 900.0, // violates range
	}
	constraint.Enforce(cs, rec)
	for _, c := range cs {
		if !c.Holds(rec) {
			t.Fatalf("constraint %q still violated after Enforce: %v", c, rec)
		}
	}
	// Repair must preserve the generated values, not flatten them.
	if rec["created_at"] != 20.0 || rec["updated_at"] != 100.0 {
		t.Fatalf("ordering repair should swap, not clamp: %v", rec)
	}
	if rec["total"] != 11.0 {
		t.Fatalf("total should be derived from its parts, got %v", rec["total"])
	}
}

// The exclusive form must clear a column that should not be populated.
func TestEnforceClearsExclusiveImplication(t *testing.T) {
	c := constraint.Constraint{
		Kind: constraint.Implication, When: "status", Equals: "refunded",
		Then: "refund_at", Exclusive: true,
	}
	paid := map[string]any{"status": "paid", "refund_at": "2026-01-01T00:00:00Z"}
	constraint.Enforce([]constraint.Constraint{c}, paid)
	if paid["refund_at"] != nil {
		t.Fatalf("refund_at should be cleared for a paid row: %v", paid)
	}

	refunded := map[string]any{"status": "refunded", "refund_at": "2026-01-01T00:00:00Z"}
	constraint.Enforce([]constraint.Constraint{c}, refunded)
	if refunded["refund_at"] == nil {
		t.Fatalf("refund_at must survive on a refunded row: %v", refunded)
	}
}

// A non-exclusive rule says nothing about non-matching rows, so enforcement
// must not touch them.
func TestEnforceLeavesNonExclusiveRowsAlone(t *testing.T) {
	c := constraint.Constraint{
		Kind: constraint.Implication, When: "status", Equals: "refunded",
		Then: "note", Exclusive: false,
	}
	rec := map[string]any{"status": "paid", "note": "keep me"}
	constraint.Enforce([]constraint.Constraint{c}, rec)
	if rec["note"] != "keep me" {
		t.Fatalf("non-exclusive rule modified a non-matching row: %v", rec)
	}
}

// Mining what has been enforced must find the same invariants: the round trip
// is what makes the feature trustworthy.
func TestMinedConstraintsSurviveEnforcement(t *testing.T) {
	var sample []map[string]any
	for i := 0; i < 200; i++ {
		sub, tax := float64(i)*2, float64(i)*0.2
		sample = append(sample, map[string]any{
			"created_at": float64(i), "updated_at": float64(i + 10),
			"subtotal": sub, "tax": tax, "total": sub + tax,
		})
	}
	mined := constraint.Mine(sample, 1.0)
	if len(mined) == 0 {
		t.Fatal("mined nothing from a sample full of invariants")
	}

	// Deliberately incoherent records, as an unconstrained generator makes.
	var generated []map[string]any
	for i := 0; i < 200; i++ {
		rec := map[string]any{
			"created_at": float64(500 - i), "updated_at": float64(i),
			"subtotal": float64(i), "tax": float64(i) * 3, "total": float64(i) * 77,
		}
		if !constraint.Enforce(mined, rec) {
			t.Fatalf("row %d could not be repaired: %v", i, rec)
		}
		generated = append(generated, rec)
	}
	for _, c := range mined {
		for i, rec := range generated {
			if !c.Holds(rec) {
				t.Fatalf("row %d violates %q after enforcement: %v", i, c, rec)
			}
		}
	}
}

// Timestamps are where ordering invariants matter most, so string dates must
// be understood, not skipped.
func TestMineOrderingOnStringTimestamps(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 100; i++ {
		rows = append(rows, map[string]any{
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-06-01T00:00:00Z",
		})
	}
	for _, c := range constraint.Mine(rows, 1.0) {
		if c.Kind == constraint.Ordering && c.Left == "created_at" && c.Right == "updated_at" {
			return
		}
	}
	t.Fatal("did not mine an ordering between RFC 3339 timestamp columns")
}

// An empty CSV cell and a SQL NULL mean the same thing to an export.
func TestNullAndEmptyStringAreEquivalent(t *testing.T) {
	c := constraint.Constraint{
		Kind: constraint.Implication, When: "status", Equals: "refunded", Then: "refund_at",
	}
	if !c.Holds(map[string]any{"status": "paid", "refund_at": ""}) {
		t.Fatal("an empty cell should count as null")
	}
	if c.Holds(map[string]any{"status": "refunded", "refund_at": ""}) {
		t.Fatal("an empty cell should violate a rule requiring a value")
	}
}

// Mining an empty sample must return nothing rather than panic.
func TestMineEmptySample(t *testing.T) {
	if got := constraint.Mine(nil, 1.0); got != nil {
		t.Fatalf("mined %v from an empty sample", got)
	}
}
