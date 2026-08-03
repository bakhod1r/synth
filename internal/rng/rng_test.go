package rng

import (
	"math"
	"testing"
)

func TestNewDeterministic(t *testing.T) {
	a, b := New(42), New(42)
	for i := 0; i < 100; i++ {
		if a.Uint64() != b.Uint64() {
			t.Fatalf("same seed diverged at %d", i)
		}
	}
}

func TestForkPure(t *testing.T) {
	r := New(1)
	// Fork is a pure function of seed+i, independent of stream consumption.
	f1 := r.Fork(7)
	r.Uint64()
	r.Uint64()
	f2 := r.Fork(7)
	if f1.Uint64() != f2.Uint64() {
		t.Fatal("Fork not pure: consuming parent stream changed child")
	}
	// Different indices differ.
	if New(1).Fork(1).Uint64() == New(1).Fork(2).Uint64() {
		t.Fatal("distinct indices produced identical stream")
	}
}

func TestIntn(t *testing.T) {
	r := New(3)
	for i := 0; i < 1000; i++ {
		v := r.Intn(10)
		if v < 0 || v >= 10 {
			t.Fatalf("Intn out of range: %d", v)
		}
	}
}

func TestIntRange(t *testing.T) {
	r := New(4)
	if got := r.IntRange(5, 5); got != 5 {
		t.Fatalf("IntRange(5,5)=%d, want 5", got)
	}
	if got := r.IntRange(9, 2); got != 9 {
		t.Fatalf("IntRange(9,2)=%d, want min 9", got)
	}
	for i := 0; i < 1000; i++ {
		v := r.IntRange(3, 8)
		if v < 3 || v > 8 {
			t.Fatalf("IntRange out of bounds: %d", v)
		}
	}
}

func TestFloat64AndBool(t *testing.T) {
	r := New(5)
	for i := 0; i < 1000; i++ {
		f := r.Float64()
		if f < 0 || f >= 1 {
			t.Fatalf("Float64 out of range: %v", f)
		}
	}
	if r.Bool(0) {
		t.Fatal("Bool(0) must be false")
	}
	if !r.Bool(1) {
		t.Fatal("Bool(1) must be true")
	}
}

func TestPick(t *testing.T) {
	r := New(6)
	for i := 0; i < 100; i++ {
		if v := r.Pick(4); v < 0 || v >= 4 {
			t.Fatalf("Pick out of range: %d", v)
		}
	}
}

func TestNormFloat64(t *testing.T) {
	r := New(7)
	var sum float64
	const n = 20000
	for i := 0; i < n; i++ {
		v := r.NormFloat64()
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("NormFloat64 non-finite: %v", v)
		}
		sum += v
	}
	if mean := sum / n; math.Abs(mean) > 0.1 {
		t.Fatalf("NormFloat64 mean %v far from 0", mean)
	}
}

func TestDigits(t *testing.T) {
	r := New(8)
	s := r.Digits(12)
	if len(s) != 12 {
		t.Fatalf("Digits len=%d, want 12", len(s))
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("non-digit rune: %q", c)
		}
	}
	if r.Digits(0) != "" {
		t.Fatal("Digits(0) must be empty")
	}
}
