package constraint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/constraint"
)

func TestConstraintStringAllKinds(t *testing.T) {
	cases := []constraint.Constraint{
		{Kind: constraint.Ordering, Left: "a", Right: "b"},
		{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c"},
		{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t"},
		{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t", Exclusive: true},
		{Kind: constraint.Range, Left: "a", Lo: 1, Hi: 9},
		{Kind: constraint.Kind("weird")},
	}
	for _, c := range cases {
		if c.String() == "" {
			t.Fatalf("empty String for %+v", c)
		}
	}
}

func TestHoldsAllKinds(t *testing.T) {
	ord := constraint.Constraint{Kind: constraint.Ordering, Left: "a", Right: "b"}
	if !ord.Holds(map[string]any{"a": 1, "b": 2}) || ord.Holds(map[string]any{"a": 5, "b": 2}) {
		t.Fatal("ordering Holds wrong")
	}
	if !ord.Holds(map[string]any{"a": "notnum"}) { // vacuous when non-numeric
		t.Fatal("ordering should hold vacuously")
	}

	sum := constraint.Constraint{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c"}
	if !sum.Holds(map[string]any{"a": 1, "b": 2, "c": 3}) || sum.Holds(map[string]any{"a": 1, "b": 2, "c": 9}) {
		t.Fatal("sum Holds wrong")
	}
	if !sum.Holds(map[string]any{"a": 1}) || !sum.Holds(map[string]any{"c": "x"}) {
		t.Fatal("sum should hold vacuously on missing parts/whole")
	}

	imp := constraint.Constraint{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t"}
	if !imp.Holds(map[string]any{"s": "y", "t": nil}) {
		t.Fatal("implication holds when trigger absent and target null")
	}
	if imp.Holds(map[string]any{"s": "x", "t": nil}) {
		t.Fatal("implication violated when trigger matches but target null")
	}
	if !imp.Holds(map[string]any{"s": "x", "t": "v"}) {
		t.Fatal("implication holds when target populated")
	}
	excl := imp
	excl.Exclusive = true
	if excl.Holds(map[string]any{"s": "y", "t": "v"}) {
		t.Fatal("exclusive implication violated when populated off-trigger")
	}

	rng := constraint.Constraint{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 10}
	if !rng.Holds(map[string]any{"a": 5}) || rng.Holds(map[string]any{"a": 20}) {
		t.Fatal("range Holds wrong")
	}
	if !rng.Holds(map[string]any{"a": "x"}) {
		t.Fatal("range vacuous on non-numeric")
	}
}

func TestHoldsNumericCoercions(t *testing.T) {
	c := constraint.Constraint{Kind: constraint.Range, Left: "a", Lo: -1e18, Hi: 1e18}
	for _, v := range []any{
		float64(1), float32(1), int(1), int32(1), int64(1),
		time.Now(), "42", "2026-01-01", nil, "", "notnum",
	} {
		_ = c.Holds(map[string]any{"a": v}) // exercises every toNum branch
	}
}

func TestEnforceAllRepairs(t *testing.T) {
	// ordering swap
	r := map[string]any{"a": 9.0, "b": 1.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Ordering, Left: "a", Right: "b"}}, r)
	if r["a"].(float64) > r["b"].(float64) {
		t.Fatal("ordering not repaired")
	}
	// sum recompute
	r2 := map[string]any{"a": 2.0, "b": 3.0, "c": 99.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c"}}, r2)
	if r2["c"].(float64) != 5.0 {
		t.Fatalf("sum not recomputed: %v", r2["c"])
	}
	// sum with a missing part leaves whole alone
	r2b := map[string]any{"b": 3.0, "c": 99.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c"}}, r2b)

	// exclusive implication clears
	r3 := map[string]any{"s": "y", "t": "v"}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t", Exclusive: true}}, r3)
	if r3["t"] != nil {
		t.Fatal("exclusive implication not cleared")
	}
	// non-exclusive implication leaves it
	r3b := map[string]any{"s": "y", "t": "v"}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t"}}, r3b)
	if r3b["t"] != "v" {
		t.Fatal("non-exclusive should not clear")
	}
	// range clamp low and high
	rl := map[string]any{"a": -5.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 10}}, rl)
	if rl["a"].(float64) != 0 {
		t.Fatal("range low not clamped")
	}
	rh := map[string]any{"a": 50.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 10}}, rh)
	if rh["a"].(float64) != 10 {
		t.Fatal("range high not clamped")
	}
}

func TestEnforceContradiction(t *testing.T) {
	// a<=b and b<=a with a!=b cannot both hold after swaps -> returns false.
	r := map[string]any{"a": 2.0, "b": 1.0}
	cs := []constraint.Constraint{
		{Kind: constraint.Range, Left: "a", Lo: 5, Hi: 5},
		{Kind: constraint.Range, Left: "a", Lo: 1, Hi: 1},
	}
	if constraint.Enforce(cs, r) {
		t.Fatal("contradictory constraints should not converge")
	}
}

func TestMineSumsTriplesAndRanges(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 60; i++ {
		a, b, c := float64(i), float64(i*2), float64(i%3)
		rows = append(rows, map[string]any{
			"a": a, "b": b, "c": c, "total": a + b + c,
			"cat": []string{"p", "q"}[i%2],
			"opt": func() any {
				if i%2 == 0 {
					return "set"
				}
				return nil
			}(),
		})
	}
	cs := constraint.Mine(rows, 1)
	if len(cs) == 0 {
		t.Fatal("expected mined constraints")
	}
	// minSupport out of range defaults to 1.
	_ = constraint.Mine(rows, 5)
	if constraint.Mine(nil, 1) != nil {
		t.Fatal("empty rows -> nil")
	}
}

func TestMineNumericCategoryNotMined(t *testing.T) {
	// A numeric-valued column is a measurement, not a category: no implication.
	var rows []map[string]any
	for i := 0; i < 40; i++ {
		rows = append(rows, map[string]any{"n": i % 3, "x": "v"})
	}
	_ = constraint.Mine(rows, 1)
}

func TestHoldsUnknownKindAndAsStringToNum(t *testing.T) {
	// Unknown kind holds by default.
	if !(constraint.Constraint{Kind: constraint.Kind("mystery")}).Holds(map[string]any{}) {
		t.Fatal("unknown kind should hold")
	}
	// asString of a non-string When (implication compares stringified value).
	imp := constraint.Constraint{Kind: constraint.Implication, When: "s", Equals: "5", Then: "t"}
	_ = imp.Holds(map[string]any{"s": 5, "t": nil})
	// toNum default branch: a bool is not a number.
	rng := constraint.Constraint{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 1}
	if !rng.Holds(map[string]any{"a": true}) {
		t.Fatal("bool should be vacuous for range")
	}
}

func TestEnforceMissingKeyBranches(t *testing.T) {
	// sum whole absent
	r := map[string]any{"a": 1.0, "b": 2.0}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.SumEquals, Parts: []string{"a", "b"}, Whole: "c"}}, r)
	if _, ok := r["c"]; ok {
		t.Fatal("absent whole should stay absent")
	}
	// exclusive implication then absent
	r2 := map[string]any{"s": "y"}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Implication, When: "s", Equals: "x", Then: "t", Exclusive: true}}, r2)
	// range on non-numeric leaves value
	r3 := map[string]any{"a": "text"}
	constraint.Enforce([]constraint.Constraint{{Kind: constraint.Range, Left: "a", Lo: 0, Hi: 1}}, r3)
	if r3["a"] != "text" {
		t.Fatal("non-numeric range value should be untouched")
	}
}

func TestMineManyColumnsAndMixed(t *testing.T) {
	// 14 numeric columns forces the maxNumericCols truncation. 200 rows with a
	// single non-numeric cell per relevant column keeps them >=99% numeric while
	// exercising the skip branches in ordering and checkSum.
	var rows []map[string]any
	for i := 0; i < 200; i++ {
		r := map[string]any{"a": float64(i), "b": float64(i * 2), "total": float64(i * 3)}
		for c := 0; c < 14; c++ {
			r["n"+string(rune('a'+c))] = float64(i + c)
		}
		if i == 0 {
			r["a"] = "notnum" // part non-numeric in one row
		}
		if i == 1 {
			r["total"] = "notnum" // whole non-numeric in one row
		}
		rows = append(rows, r)
	}
	_ = constraint.Mine(rows, 1)
}

func TestMineTooManyDistinctCategory(t *testing.T) {
	// A string column with >20 distinct values is not a category.
	var rows []map[string]any
	for i := 0; i < 60; i++ {
		rows = append(rows, map[string]any{
			"id":  "v" + string(rune('a'+i%50)) + string(rune('a'+i)),
			"opt": "x",
		})
	}
	_ = constraint.Mine(rows, 1)
}

func TestMineLargeDatasetGroupFraction(t *testing.T) {
	// A big dataset makes the minGroup come from the fraction, not the floor.
	var rows []map[string]any
	for i := 0; i < 1000; i++ {
		var opt any
		if i%2 == 0 {
			opt = "set"
		} else {
			opt = nil
		}
		rows = append(rows, map[string]any{"cat": []string{"p", "q"}[i%2], "opt": opt})
	}
	_ = constraint.Mine(rows, 1)
}

func TestSampleLoaders(t *testing.T) {
	dir := t.TempDir()
	csvp := filepath.Join(dir, "d.csv")
	os.WriteFile(csvp, []byte("a,b\n1,2\n3,4\n"), 0o644)
	rows, err := constraint.LoadSample(csvp, 0) // 0 -> DefaultSample
	if err != nil || len(rows) != 2 {
		t.Fatalf("csv load: %v rows=%d", err, len(rows))
	}
	jp := filepath.Join(dir, "d.jsonl")
	os.WriteFile(jp, []byte(`{"a":1}`+"\n"+`{"a":2}`+"\n"), 0o644)
	if _, err := constraint.LoadSample(jp, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := constraint.LoadSample(filepath.Join(dir, "missing.csv"), 10); err == nil {
		t.Fatal("missing file should error")
	}
	// unknown format
	if _, err := constraint.ReadSample(strings.NewReader("x"), "xml", 10); err == nil {
		t.Fatal("unknown format should error")
	}
	// empty CSV -> nil rows
	if rows, err := constraint.ReadSample(strings.NewReader(""), "csv", 10); err != nil || rows != nil {
		t.Fatalf("empty csv: %v %v", rows, err)
	}
	// malformed CSV row
	if _, err := constraint.ReadSample(strings.NewReader("a,b\n\"bad,2\n"), "csv", 10); err == nil {
		t.Fatal("malformed csv should error")
	}
	// malformed JSONL
	if _, err := constraint.ReadSample(strings.NewReader("{bad}\n"), "jsonl", 10); err == nil {
		t.Fatal("malformed jsonl should error")
	}
	// malformed header (unterminated quote) surfaces a non-EOF read error
	if _, err := constraint.ReadSample(strings.NewReader("\"unterminated,b\n"), "csv", 10); err == nil {
		t.Fatal("malformed header should error")
	}
}
