package verify

import "testing"

func rowsOf(maps ...map[string]any) []map[string]any { return maps }

// Every quasi-identifier tuple shared by at least k rows passes.
func TestKAnonymitySatisfied(t *testing.T) {
	rows := rowsOf(
		map[string]any{"age": "30", "zip": "10001"},
		map[string]any{"age": "30", "zip": "10001"},
		map[string]any{"age": "40", "zip": "20002"},
		map[string]any{"age": "40", "zip": "20002"},
	)
	rep := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"age", "zip"}})
	if !rep.OK() {
		t.Errorf("2-anonymous data reported violations: %+v", rep.Findings)
	}
}

// A tuple that appears fewer than k times is an error.
func TestKAnonymityViolation(t *testing.T) {
	rows := rowsOf(
		map[string]any{"age": "30", "zip": "10001"},
		map[string]any{"age": "30", "zip": "10001"},
		map[string]any{"age": "99", "zip": "55555"}, // unique → re-identifiable
	)
	rep := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"age", "zip"}})
	if rep.OK() {
		t.Fatal("a unique quasi-identifier tuple should fail k-anonymity")
	}
	found := false
	for _, f := range rep.Findings {
		if f.Check == "k-anonymity" && f.Severity == SevError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a k-anonymity error, got %+v", rep.Findings)
	}
}

// A quasi-identifier column that is not in the data is an error, not a silent
// pass — a typo must not read as "no violations".
func TestKAnonymityUnknownColumn(t *testing.T) {
	rows := rowsOf(map[string]any{"age": "30"})
	rep := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"age", "nope"}})
	if rep.OK() {
		t.Error("an unknown quasi-identifier column should error")
	}
}

// k=1 can never be violated, so the check stays quiet.
func TestKAnonymityKOneNeverFires(t *testing.T) {
	rows := rowsOf(
		map[string]any{"age": "1"},
		map[string]any{"age": "2"},
	)
	rep := Run(rows, Options{KAnonymity: 1, QuasiIdentifiers: []string{"age"}})
	if !rep.OK() {
		t.Errorf("k=1 should never fire: %+v", rep.Findings)
	}
}

// Without QI columns the check does not run at all.
func TestKAnonymityInactiveWithoutQI(t *testing.T) {
	rows := rowsOf(map[string]any{"age": "30"})
	rep := Run(rows, Options{KAnonymity: 5})
	for _, f := range rep.Findings {
		if f.Check == "k-anonymity" {
			t.Errorf("k-anonymity ran without quasi-identifiers: %+v", f)
		}
	}
}
