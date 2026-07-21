package synth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bakhodir/synth"
)

type RateEvent struct {
	ID    int
	Name  string
	Email string
}

// The stream must deliver exactly Total records and honor the target rate.
func TestRateStreamTotalAndPacing(t *testing.T) {
	const total = 40
	const perSec = 400.0

	got := 0
	start := time.Now()
	err := synth.Rate[RateEvent](synth.RateConfig{
		PerSecond: perSec, Total: total, Burst: 4,
	}, synth.WithSeed(1)).Run(context.Background(), func(e RateEvent) error {
		if e.Email == "" {
			t.Fatal("record not generated")
		}
		got++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != total {
		t.Fatalf("emitted %d records, want %d", got, total)
	}
	// 40 events at 400/sec ≈ 100ms. Allow generous slack for CI timing, but it
	// must not complete instantly (that would mean pacing is ignored).
	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Fatalf("stream finished in %v — pacing was not applied", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("stream took %v — far slower than the target rate", elapsed)
	}
}

// Cancelling the context must stop the stream promptly.
func TestRateStreamContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	got := 0
	err := synth.Rate[RateEvent](synth.RateConfig{PerSecond: 50}).
		Run(ctx, func(RateEvent) error { got++; return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context deadline error, got %v", err)
	}
	if got == 0 {
		t.Fatal("no records emitted before cancellation")
	}
}

// An error from the callback stops the stream and is returned unchanged.
func TestRateStreamCallbackError(t *testing.T) {
	sentinel := errors.New("consumer failed")
	got := 0
	err := synth.Rate[RateEvent](synth.RateConfig{Total: 100}).
		Run(context.Background(), func(RateEvent) error {
			got++
			if got == 5 {
				return sentinel
			}
			return nil
		})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel error, got %v", err)
	}
	if got != 5 {
		t.Fatalf("stream continued after error: %d records", got)
	}
}

// With no rate set the stream runs unpaced and still delivers Total.
func TestRateStreamUnpaced(t *testing.T) {
	got := 0
	err := synth.Rate[RateEvent](synth.RateConfig{Total: 5000}, synth.WithSeed(2)).
		Run(context.Background(), func(RateEvent) error { got++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if got != 5000 {
		t.Fatalf("emitted %d, want 5000", got)
	}
}
