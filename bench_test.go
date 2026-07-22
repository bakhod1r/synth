package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
)

func BenchmarkFluentName(b *testing.B) {
	g := synth.New(synth.Config{Seed: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Name()
	}
}

func BenchmarkMake(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = synth.Make[User](1000, synth.WithSeed(1))
	}
}

func BenchmarkMakeParallel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = synth.MakeParallel[User](1000, 0, synth.WithSeed(1))
	}
}

// Parallel output must match serial output for the same seed.
func TestParallelMatchesSerial(t *testing.T) {
	serial := synth.Make[User](200, synth.WithSeed(77))
	par, err := synth.MakeParallel[User](200, 4, synth.WithSeed(77))
	if err != nil {
		t.Fatal(err)
	}
	for i := range serial {
		if serial[i] != par[i] {
			t.Fatalf("record %d: parallel != serial", i)
		}
	}
}
