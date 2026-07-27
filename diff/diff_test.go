package diff

import (
	"testing"

	"github.com/bakhod1r/synth/profile"
	"github.com/bakhod1r/synth/schema"
)

func res(order []string, stats ...*profile.ColumnStats) *profile.Result {
	m := map[string]*profile.ColumnStats{}
	for _, s := range stats {
		m[s.Name] = s
	}
	return &profile.Result{Schema: &schema.Schema{}, Order: order, Rows: 100, Stats: m}
}

func num(name string, min, max float64, nonNull, nulls int) *profile.ColumnStats {
	return &profile.ColumnStats{Name: name, Numeric: true, Min: min, Max: max,
		NonNull: nonNull, Nulls: nulls}
}

func cat(name string, values map[string]int) *profile.ColumnStats {
	n := 0
	for _, c := range values {
		n += c
	}
	return &profile.ColumnStats{Name: name, Categorical: true, Values: values,
		NonNull: n, Distinct: len(values)}
}

func hasFinding(fs []Finding, col string, sev Severity) bool {
	for _, f := range fs {
		if f.Column == col && f.Severity == sev {
			return true
		}
	}
	return false
}

func TestIdenticalNoFindings(t *testing.T) {
	a := res([]string{"amount"}, num("amount", 0, 1000, 100, 0))
	b := res([]string{"amount"}, num("amount", 0, 1000, 100, 0))
	if fs := Compare(a, b, Options{}); len(fs) != 0 {
		t.Errorf("identical inputs produced findings: %+v", fs)
	}
}

func TestColumnRemovedAndAddedAreErrors(t *testing.T) {
	a := res([]string{"legacy_id"}, num("legacy_id", 1, 9, 100, 0))
	b := res([]string{"tier"}, cat("tier", map[string]int{"a": 100}))
	fs := Compare(a, b, Options{})
	if !hasFinding(fs, "legacy_id", Error) {
		t.Error("removed column should be an error")
	}
	if !hasFinding(fs, "tier", Error) {
		t.Error("added column should be an error")
	}
}

func TestTypeFlipIsError(t *testing.T) {
	a := res([]string{"x"}, num("x", 0, 10, 100, 0))
	b := res([]string{"x"}, cat("x", map[string]int{"hi": 100}))
	if !hasFinding(Compare(a, b, Options{}), "x", Error) {
		t.Error("numeric→categorical flip should be an error")
	}
}

func TestNumericDriftPastToleranceWarns(t *testing.T) {
	a := res([]string{"amount"}, num("amount", 0, 100, 100, 0))
	b := res([]string{"amount"}, num("amount", 0, 130, 100, 0)) // +30% max
	if !hasFinding(Compare(a, b, Options{}), "amount", Warn) {
		t.Error("30%% max drift should warn at the 10%% default")
	}
}

func TestNumericDriftWithinToleranceQuiet(t *testing.T) {
	a := res([]string{"amount"}, num("amount", 0, 100, 100, 0))
	b := res([]string{"amount"}, num("amount", 0, 105, 100, 0)) // +5%
	for _, f := range Compare(a, b, Options{}) {
		if f.Column == "amount" && f.Severity == Warn {
			t.Errorf("5%% drift should be within the 10%% tolerance: %+v", f)
		}
	}
}

func TestToleranceWidens(t *testing.T) {
	a := res([]string{"amount"}, num("amount", 0, 100, 100, 0))
	b := res([]string{"amount"}, num("amount", 0, 130, 100, 0))
	for _, f := range Compare(a, b, Options{Tolerance: 0.5}) {
		if f.Column == "amount" && f.Severity == Warn {
			t.Errorf("30%% drift should be quiet at tolerance 0.5: %+v", f)
		}
	}
}

func TestNewCategoryWarns(t *testing.T) {
	a := res([]string{"status"}, cat("status", map[string]int{"active": 50, "closed": 50}))
	b := res([]string{"status"}, cat("status", map[string]int{"active": 40, "closed": 40, "pending": 20}))
	if !hasFinding(Compare(a, b, Options{}), "status", Warn) {
		t.Error("a new category should warn")
	}
}

func TestNullRateDriftWarns(t *testing.T) {
	a := res([]string{"email"}, num("email", 0, 1, 100, 0)) // 0% null
	b := res([]string{"email"}, num("email", 0, 1, 80, 20)) // 20% null
	if !hasFinding(Compare(a, b, Options{}), "email", Warn) {
		t.Error("a 20-point null-rate jump should warn")
	}
}
