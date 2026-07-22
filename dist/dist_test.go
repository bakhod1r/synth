package dist

import (
	"math"
	"testing"

	"github.com/bakhodir/synth/internal/rng"
)

// A wrong distribution is the quietest bug in a data generator: the values look
// plausible one at a time, and only an aggregate over thousands of rows shows
// that the shape is wrong. So these tests check the shape, not the values.

const samples = 200_000

func draw(d interface{ Sample(*rng.Rand) float64 }) []float64 {
	r := rng.New(1)
	out := make([]float64, samples)
	for i := range out {
		out[i] = d.Sample(r)
	}
	return out
}

func meanStd(xs []float64) (float64, float64) {
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		sq += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(sq / float64(len(xs)))
}

func TestUniformRange(t *testing.T) {
	xs := draw(Uniform{Min: 10, Max: 20})
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, x := range xs {
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	if lo < 10 || hi > 20 {
		t.Fatalf("range = %v..%v, want inside 10..20", lo, hi)
	}
	// The whole range must be used, or "uniform" is a lie about a narrower one.
	if lo > 10.1 || hi < 19.9 {
		t.Fatalf("range = %v..%v, which does not cover 10..20", lo, hi)
	}
	if mean, _ := meanStd(xs); math.Abs(mean-15) > 0.1 {
		t.Fatalf("mean = %v, want ~15", mean)
	}
}

func TestNormalMeanAndSpread(t *testing.T) {
	xs := draw(Normal{Mu: 100, Sigma: 15})
	mean, std := meanStd(xs)
	if math.Abs(mean-100) > 0.5 {
		t.Errorf("mean = %v, want ~100", mean)
	}
	if math.Abs(std-15) > 0.5 {
		t.Errorf("std = %v, want ~15", std)
	}

	// Roughly two thirds within one sigma is what makes it a normal rather than
	// something else with the same mean and spread.
	var within int
	for _, x := range xs {
		if math.Abs(x-100) <= 15 {
			within++
		}
	}
	if share := float64(within) / float64(len(xs)); math.Abs(share-0.682) > 0.02 {
		t.Errorf("%.3f of samples within one sigma, want ~0.682", share)
	}
}

// A log-normal is the shape money and durations actually have: never negative,
// with a long right tail. Both properties matter — a negative salary is an
// invalid row, and no tail makes the distribution pointless.
func TestLogNormalIsPositiveAndSkewed(t *testing.T) {
	xs := draw(LogNormal{Mu: 3, Sigma: 1})
	for i, x := range xs {
		if x <= 0 {
			t.Fatalf("sample %d = %v, log-normal values are always positive", i, x)
		}
	}
	mean, _ := meanStd(xs)
	var above int
	for _, x := range xs {
		if x > mean {
			above++
		}
	}
	// A symmetric distribution has half its mass above the mean; a right-skewed
	// one has clearly less.
	if share := float64(above) / float64(len(xs)); share > 0.45 {
		t.Errorf("%.3f of samples above the mean — the distribution is not skewed", share)
	}
}

func TestExponential(t *testing.T) {
	const rate = 0.5
	xs := draw(Exponential{Rate: rate})
	for i, x := range xs {
		if x < 0 {
			t.Fatalf("sample %d = %v, an exponential is never negative", i, x)
		}
	}
	// Mean and standard deviation both equal 1/rate; that equality is what
	// distinguishes an exponential from any other positive distribution.
	mean, std := meanStd(xs)
	want := 1 / rate
	if math.Abs(mean-want) > 0.05 {
		t.Errorf("mean = %v, want ~%v", mean, want)
	}
	if math.Abs(std-want) > 0.05 {
		t.Errorf("std = %v, want ~%v", std, want)
	}
}

// Zipf is what makes a generated dataset look real: a few values appear
// constantly and most appear once. A uniform draw would defeat every cache and
// index test the data was made for.
func TestZipfIsHeavyHeaded(t *testing.T) {
	z := NewZipf(1000, 1.1)
	r := rng.New(1)
	counts := make([]int, 1000)
	for i := 0; i < samples; i++ {
		rank := z.Rank(r)
		if rank < 0 || rank >= 1000 {
			t.Fatalf("rank %d is outside 0..999", rank)
		}
		counts[rank]++
	}
	// The most common value must dominate, and the top ten must carry a large
	// share. Uniform would give 0.1% and 1%.
	top := float64(counts[0]) / float64(samples)
	if top < 0.05 {
		t.Errorf("the most common rank is %.3f of draws, want a heavy head", top)
	}
	var topTen int
	for _, c := range counts[:10] {
		topTen += c
	}
	if share := float64(topTen) / float64(samples); share < 0.2 {
		t.Errorf("the top ten ranks are %.3f of draws, want a heavy head", share)
	}
	// The ordering must hold: rank 0 is the most common by construction.
	if counts[0] <= counts[1] || counts[1] <= counts[50] {
		t.Errorf("ranks are not ordered by frequency: %d, %d, %d",
			counts[0], counts[1], counts[50])
	}
}

// Every distribution must be reproducible from a seed, or a dataset built with
// one is not reproducible either.
func TestSamplingIsDeterministic(t *testing.T) {
	for name, d := range map[string]interface {
		Sample(*rng.Rand) float64
	}{
		"uniform":     Uniform{Min: 0, Max: 1},
		"normal":      Normal{Mu: 0, Sigma: 1},
		"lognormal":   LogNormal{Mu: 0, Sigma: 1},
		"exponential": Exponential{Rate: 1},
		"zipf":        NewZipf(100, 1.2),
	} {
		a, b := rng.New(7), rng.New(7)
		for i := 0; i < 100; i++ {
			if x, y := d.Sample(a), d.Sample(b); x != y {
				t.Fatalf("%s: draw %d differs between two runs of the same seed: %v vs %v",
					name, i, x, y)
			}
		}
	}
}

// Degenerate parameters must produce a usable number rather than NaN, which
// would reach a CSV as the literal text "NaN".
func TestDegenerateParameters(t *testing.T) {
	r := rng.New(1)
	for name, x := range map[string]float64{
		"zero sigma":     Normal{Mu: 5, Sigma: 0}.Sample(r),
		"inverted range": Uniform{Min: 10, Max: 1}.Sample(r),
		"zero rate":      Exponential{Rate: 0}.Sample(r),
	} {
		if math.IsNaN(x) {
			t.Errorf("%s produced NaN", name)
		}
	}
	if z := NewZipf(0, 1.1); z != nil {
		if rank := z.Rank(r); rank < 0 {
			t.Errorf("an empty Zipf produced rank %d", rank)
		}
	}
}
