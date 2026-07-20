// Package rng provides a fast, per-instance random source. Unlike the
// stdlib global rand, every Generator owns its own *Rand, so concurrent
// goroutines never contend on a shared mutex — the classic faker bottleneck.
package rng

import "math/rand/v2"

// Rand is a lightweight, non-cryptographic PRNG. Not safe for concurrent
// use; give each goroutine its own via Fork or a pool.
type Rand struct {
	seed uint64
	src  *rand.Rand
}

// New returns a Rand seeded deterministically.
func New(seed uint64) *Rand {
	return &Rand{seed: seed, src: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// Fork derives a new independent Rand for index i. It is a PURE function of
// this Rand's seed and i — it does not consume this Rand's stream — so record i
// is identical no matter the order or goroutine it is produced in.
func (r *Rand) Fork(i uint64) *Rand {
	return New(r.seed ^ ((i + 1) * 0x2545f4914f6cdd1d))
}

// Intn returns a non-negative int in [0,n). Panics if n<=0.
func (r *Rand) Intn(n int) int { return r.src.IntN(n) }

// IntRange returns an int in [min,max].
func (r *Rand) IntRange(min, max int) int {
	if max <= min {
		return min
	}
	return min + r.src.IntN(max-min+1)
}

// Float64 returns a float in [0,1).
func (r *Rand) Float64() float64 { return r.src.Float64() }

// Bool returns true with the given probability.
func (r *Rand) Bool(p float64) bool { return r.src.Float64() < p }

// Pick returns a random element index for a slice of length n.
func (r *Rand) Pick(n int) int { return r.src.IntN(n) }

// Uint64 returns a random 64-bit value.
func (r *Rand) Uint64() uint64 { return r.src.Uint64() }

// Digits returns a string of n random decimal digits.
func (r *Rand) Digits(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + r.src.IntN(10))
	}
	return string(b)
}
