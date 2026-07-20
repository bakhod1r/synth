package synth_test

import (
	"math"
	"sort"
	"testing"

	"github.com/bakhodir/synth"
)

type Txn struct {
	ID       int
	Amount   float64 `synth:"amount,dist=lognormal,mu=10,sigma=1,min=0,max=100000000"`
	Status   string  `synth:"enum,choices=settled|pending|failed,weights=0.94|0.05|0.01"`
	Category string  `synth:"enum,choices=a|b|c|d|e|f|g|h,dist=zipf,s=1.2"`
}

// Weighted enum: "settled" should dominate (~94%), "failed" should be rare.
func TestWeightedEnumSkew(t *testing.T) {
	txns := synth.Make[Txn](20000, synth.WithSeed(1))
	counts := map[string]int{}
	for _, x := range txns {
		counts[x.Status]++
	}
	settled := float64(counts["settled"]) / float64(len(txns))
	failed := float64(counts["failed"]) / float64(len(txns))
	if settled < 0.90 || settled > 0.97 {
		t.Fatalf("settled share %.3f outside expected ~0.94", settled)
	}
	if failed > 0.03 {
		t.Fatalf("failed share %.3f too high", failed)
	}
}

// Zipf: the first category must dominate the last.
func TestZipfHotKey(t *testing.T) {
	txns := synth.Make[Txn](20000, synth.WithSeed(2))
	counts := map[string]int{}
	for _, x := range txns {
		counts[x.Category]++
	}
	if counts["a"] <= counts["h"]*3 {
		t.Fatalf("expected zipf hot key: a=%d h=%d", counts["a"], counts["h"])
	}
}

// LogNormal amounts must stay positive and right-skewed (mean > median).
func TestLogNormalShape(t *testing.T) {
	txns := synth.Make[Txn](20000, synth.WithSeed(3))
	var sum float64
	vals := make([]float64, 0, len(txns))
	for _, x := range txns {
		if x.Amount < 0 {
			t.Fatalf("negative amount %v", x.Amount)
		}
		sum += x.Amount
		vals = append(vals, x.Amount)
	}
	mean := sum / float64(len(vals))
	// crude median: nth smallest via selection is overkill; sort.
	sort.Float64s(vals)
	median := vals[len(vals)/2]
	if !(mean > median) {
		t.Fatalf("expected right skew: mean %.1f median %.1f", mean, median)
	}
	if math.IsNaN(mean) {
		t.Fatal("NaN mean")
	}
}

// Weighted via code option (no tag) should behave the same.
func TestWeightedOption(t *testing.T) {
	type Rec struct {
		ID     int
		Status string
	}
	recs := synth.Make[Rec](10000, synth.WithSeed(4),
		synth.Weighted("Status", map[string]float64{"ok": 0.9, "err": 0.1}))
	ok := 0
	for _, r := range recs {
		if r.Status == "ok" {
			ok++
		}
	}
	share := float64(ok) / float64(len(recs))
	if share < 0.86 || share > 0.94 {
		t.Fatalf("ok share %.3f outside ~0.9", share)
	}
}
