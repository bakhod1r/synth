package synth_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/bakhod1r/synth"
)

type registerRow struct {
	Email string
	Name  string
}

// Register while other goroutines generate must not be a fatal concurrent map
// access. Run with -race.
func TestRegisterConcurrentWithGeneration(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = synth.Make[registerRow](5, synth.WithSeed(uint64(j)))
			}
		}()
	}
	for i := 0; i < 50; i++ {
		synth.RegisterSet(fmt.Sprintf("racekind%d", i), "a", "b")
	}
	wg.Wait()
}

type lateRow struct {
	Cinema string
}

// A type built before Register must pick up the new kind afterwards; the
// schema cache used to keep what was inferred first.
func TestRegisterInvalidatesInferredSchemas(t *testing.T) {
	_ = synth.Make[lateRow](1, synth.WithSeed(1)) // cached with Cinema unknown
	synth.RegisterSet("cinema", "Inception")
	got := synth.Make[lateRow](3, synth.WithSeed(1))
	for _, r := range got {
		if r.Cinema != "Inception" {
			t.Fatalf("Cinema = %q after Register, want Inception", r.Cinema)
		}
	}
}
