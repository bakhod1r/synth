package diff

import (
	"testing"

	"github.com/bakhod1r/synth/profile"
)

func TestCompareCoversBranches(t *testing.T) {
	// Column removed (in a, not in b) and added (in b, not a).
	a := res([]string{"x", "gone"}, num("x", 0, 10, 100, 0), num("gone", 0, 1, 100, 0))
	b := res([]string{"x", "new"}, num("x", 0, 10, 100, 0), num("new", 0, 1, 100, 0))
	fs := Compare(a, b, Options{})
	if Errors(fs) < 2 {
		t.Fatalf("expected removed+added errors, got %v", fs)
	}

	// Type flip: numeric -> non-numeric.
	an := res([]string{"c"}, num("c", 0, 10, 100, 0))
	bn := res([]string{"c"}, &profile.ColumnStats{Name: "c", Numeric: false, NonNull: 100})
	if Errors(Compare(an, bn, Options{})) == 0 {
		t.Fatal("type flip should error")
	}

	// Categorical flip.
	ac := res([]string{"c"}, cat("c", map[string]int{"a": 50}))
	bc := res([]string{"c"}, &profile.ColumnStats{Name: "c", Categorical: false, NonNull: 50})
	if Warns(Compare(ac, bc, Options{})) == 0 {
		t.Fatal("categorical flip should warn")
	}

	// relDrift x==0 (min baseline 0, so no drift finding) and sign negative
	// (max decreases past tolerance).
	az := res([]string{"c"}, num("c", 0, 100, 100, 0))
	bz := res([]string{"c"}, num("c", 0, 40, 100, 0))
	Compare(az, bz, Options{Tolerance: 0.1})

	// Min drift past tolerance (baseline min non-zero so relDrift is defined).
	amin := res([]string{"c"}, num("c", 100, 200, 100, 0))
	bmin := res([]string{"c"}, num("c", 10, 200, 100, 0))
	if Warns(Compare(amin, bmin, Options{Tolerance: 0.1})) == 0 {
		t.Fatal("min drift should warn")
	}

	// Differing row counts produce an info finding.
	ar := res([]string{"c"}, num("c", 0, 1, 100, 0))
	br := res([]string{"c"}, num("c", 0, 1, 100, 0))
	br.Rows = 250
	if len(Compare(ar, br, Options{})) == 0 {
		t.Fatal("row count change should report")
	}

	// Category set changed both directions.
	ad := res([]string{"c"}, cat("c", map[string]int{"a": 10, "b": 10}))
	bd := res([]string{"c"}, cat("c", map[string]int{"b": 10, "z": 10}))
	if Warns(Compare(ad, bd, Options{})) == 0 {
		t.Fatal("category delta should warn")
	}
}

func TestNullRateEmptyAndTolerance(t *testing.T) {
	// A column with no observations has a zero null rate (total==0 branch).
	a := res([]string{"c"}, &profile.ColumnStats{Name: "c", Numeric: true})
	b := res([]string{"c"}, &profile.ColumnStats{Name: "c", Numeric: true})
	if len(Compare(a, b, Options{NullRateTolerance: 0.2})) != 0 {
		t.Fatal("empty columns should not drift")
	}
	if (Options{NullRateTolerance: 0.3}).nullTolerance() != 0.3 {
		t.Fatal("explicit null tolerance ignored")
	}
}
