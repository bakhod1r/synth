package mask

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// Laplace noise is unbiased: over many distinct inputs the noised mean stays
// close to the true mean. Distinct inputs matter — the masker seeds noise from
// the value, so equal inputs get equal noise (its reproducibility trade).
func TestDPUnbiasedOverDistinctValues(t *testing.T) {
	m := New("secret", "en_US")
	m.Rule(Rule{Column: "salary", Strategy: DP, Epsilon: 1.0, Sensitivity: 100})

	const n = 5000
	var sumIn, sumOut float64
	for i := 0; i < n; i++ {
		in := float64(50000 + i) // distinct
		got, err := strconv.ParseFloat(m.Value("salary", strconv.FormatFloat(in, 'f', -1, 64)), 64)
		if err != nil {
			t.Fatalf("DP output not numeric: %v", err)
		}
		sumIn += in
		sumOut += got
	}
	drift := math.Abs(sumOut-sumIn) / sumIn
	if drift > 0.001 {
		t.Errorf("noised mean drifted %.4f from the true mean — noise is biased", drift)
	}
}

// Smaller epsilon means more noise: the spread scales as sensitivity/epsilon.
func TestDPSpreadScalesWithEpsilon(t *testing.T) {
	spread := func(eps float64) float64 {
		m := New("secret", "en_US")
		m.Rule(Rule{Column: "x", Strategy: DP, Epsilon: eps, Sensitivity: 100})
		var sumSq float64
		const n = 4000
		for i := 0; i < n; i++ {
			in := float64(1000 + i)
			out, _ := strconv.ParseFloat(m.Value("x", strconv.FormatFloat(in, 'f', -1, 64)), 64)
			d := out - in
			sumSq += d * d
		}
		return math.Sqrt(sumSq / n)
	}
	tight := spread(2.0)
	loose := spread(0.5) // 4x smaller epsilon → ~4x the noise
	if loose < tight*2 {
		t.Errorf("smaller epsilon should add more noise: eps=2 spread %.1f, eps=0.5 spread %.1f", tight, loose)
	}
}

// The same key reproduces the same noised value.
func TestDPReproducibleUnderKey(t *testing.T) {
	a := New("k", "en_US")
	a.Rule(Rule{Column: "x", Strategy: DP, Epsilon: 1, Sensitivity: 10})
	b := New("k", "en_US")
	b.Rule(Rule{Column: "x", Strategy: DP, Epsilon: 1, Sensitivity: 10})
	if a.Value("x", "500") != b.Value("x", "500") {
		t.Error("same key produced different noise")
	}
}

// A DP rule on a non-numeric value is an error surfaced through the file layer.
func TestDPNonNumericErrors(t *testing.T) {
	m := New("k", "en_US")
	m.Rule(Rule{Column: "name", Strategy: DP, Epsilon: 1, Sensitivity: 10})
	var out strings.Builder
	_, err := m.CSV(strings.NewReader("name\nAlice\n"), &out)
	if err == nil {
		t.Fatal("Laplace noise on a non-numeric column should error")
	}
}
