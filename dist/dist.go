// Package dist provides statistical distributions so generated data has the
// shape production data does — skew, hot keys, long tails — instead of the
// uniform noise most fakers produce. This is what stresses query planners,
// partitioners and caches the way real traffic does.
package dist

import (
	"math"

	"github.com/bakhod1r/synth/internal/rng"
)

// Dist samples a float64 from an underlying distribution.
type Dist interface {
	Sample(r *rng.Rand) float64
}

// Uniform draws uniformly from [Min,Max].
type Uniform struct{ Min, Max float64 }

func (d Uniform) Sample(r *rng.Rand) float64 {
	return d.Min + r.Float64()*(d.Max-d.Min)
}

// Normal is a Gaussian with mean Mu and standard deviation Sigma.
type Normal struct{ Mu, Sigma float64 }

func (d Normal) Sample(r *rng.Rand) float64 {
	return d.Mu + d.Sigma*r.NormFloat64()
}

// LogNormal produces positive, right-skewed values — the classic shape of
// transaction amounts, session durations, file sizes.
type LogNormal struct{ Mu, Sigma float64 }

func (d LogNormal) Sample(r *rng.Rand) float64 {
	return math.Exp(d.Mu + d.Sigma*r.NormFloat64())
}

// Exponential models inter-arrival times / waiting times. Rate is lambda.
type Exponential struct{ Rate float64 }

func (d Exponential) Sample(r *rng.Rand) float64 {
	rate := d.Rate
	if rate <= 0 {
		rate = 1
	}
	u := 1 - r.Float64() // (0,1]
	return -math.Log(u) / rate
}

// Zipf ranks N items so a few dominate (hot keys). Sample returns a rank in
// [0,N). S>1 controls skew; higher S = sharper concentration.
type Zipf struct {
	N int
	S float64
	// cdf is the precomputed cumulative distribution; build with NewZipf.
	cdf []float64
}

// NewZipf builds a Zipf over n items with skew s (s>0, typically ~1.07).
func NewZipf(n int, s float64) *Zipf {
	if n <= 0 {
		n = 1
	}
	if s <= 0 {
		s = 1.07
	}
	cdf := make([]float64, n)
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += 1.0 / math.Pow(float64(i+1), s)
		cdf[i] = sum
	}
	for i := range cdf {
		cdf[i] /= sum
	}
	return &Zipf{N: n, S: s, cdf: cdf}
}

// Rank returns a Zipf-distributed index in [0,N).
func (z *Zipf) Rank(r *rng.Rand) int {
	u := r.Float64()
	// Binary search the CDF.
	lo, hi := 0, len(z.cdf)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if z.cdf[mid] < u {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (z *Zipf) Sample(r *rng.Rand) float64 { return float64(z.Rank(r)) }
