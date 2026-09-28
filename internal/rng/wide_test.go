package rng

import (
	"math"
	"testing"
)

// max-min+1 overflowed int for wide ranges and IntN panicked; a spec handed
// to synth-mcp could crash the server with one.
func TestIntRangeWide(t *testing.T) {
	r := New(1)
	for _, c := range [][2]int{{math.MinInt64, math.MaxInt64}, {-5e18, 5e18}, {0, math.MaxInt64}, {math.MinInt64, 0}} {
		for i := 0; i < 100; i++ {
			v := r.IntRange(c[0], c[1])
			if v < c[0] || v > c[1] {
				t.Fatalf("IntRange(%d,%d) = %d out of range", c[0], c[1], v)
			}
		}
	}
}
