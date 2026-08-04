package synth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type sinkRow struct {
	ID   int
	Name string
}

// A streaming run writes for as long as it generates, so a sink that dies part
// way through — a full disk, a closed pipe, a broken network mount — has to
// stop the run. Silently finishing would hand back a truncated file that looks
// complete.
func TestStreamSinkErrors(t *testing.T) {
	tests := []struct {
		name string
		run  func(w *failWriter) error
		// rows is chosen per format: CSV buffers through csv.Writer's 4096-byte
		// bufio, so it needs enough rows to force a real write; JSONL's encoder
		// writes straight through.
		rows  int
		after int
	}{
		{"csv header", func(w *failWriter) error { return Stream[wideHeaderRow](1).csvTo(w) }, 1, 0},
		{"csv rows", func(w *failWriter) error { return Stream[sinkRow](4096).csvTo(w) }, 4096, 0},
		{"jsonl rows", func(w *failWriter) error { return Stream[sinkRow](4).jsonlTo(w) }, 4, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(&failWriter{after: tc.after}); err == nil {
				t.Fatal("streaming to a failing sink returned nil, want the sink's error")
			}
		})
	}
}

// Both file entry points compile the schema before creating the file: a type
// that cannot stream should not leave an empty file behind for the caller to
// clean up.
func TestStreamFileEntryPointsRejectNonStructs(t *testing.T) {
	path := t.TempDir() + "/out"
	if err := Stream[int](3).ToCSV(path); err == nil {
		t.Error("ToCSV = nil error, want one for a non-struct type")
	}
	if err := Stream[int](3).ToJSONL(path); err == nil {
		t.Error("ToJSONL = nil error, want one for a non-struct type")
	}
}

// Run honors cancellation between bursts on both of its unpaced routes: with no
// rate configured, and when it is already behind schedule and would otherwise
// resync and continue. Without these checks a cancelled context would only stop
// the stream once it next slept.
func TestRateRunCancelsBetweenBursts(t *testing.T) {
	tests := []struct {
		name string
		cfg  RateConfig
	}{
		{"unpaced", RateConfig{Burst: 2}},
		// A rate this high makes every interval shorter than the time a burst
		// takes, so the stream is permanently behind schedule.
		{"behind schedule", RateConfig{Burst: 2, PerSecond: 1e9}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			n := 0
			err := Rate[sinkRow](tc.cfg).Run(ctx, func(sinkRow) error {
				n++
				if n == 2 {
					cancel() // cancel mid-burst; the check happens after it
				}
				return nil
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Run = %v, want context.Canceled", err)
			}
			if n < 2 {
				t.Fatalf("emitted %d records, want the burst to finish before the check", n)
			}
		})
	}
}

// A stream whose type cannot compile fails before pacing starts.
func TestRateRunReportsCompileError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := Rate[int](RateConfig{Total: 1}).Run(ctx, func(int) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "struct") {
		t.Fatalf("Run = %v, want a compile error naming the struct requirement", err)
	}
}

// The writer-level helpers are reachable on their own, and they carry the same
// type check as the file entry points rather than assuming a caller went
// through them.
func TestStreamWriterHelpersRejectNonStructs(t *testing.T) {
	if err := Stream[int](1).csvTo(&failWriter{after: 99}); err == nil {
		t.Error("csvTo = nil error, want one for a non-struct type")
	}
	if err := Stream[int](1).jsonlTo(&failWriter{after: 99}); err == nil {
		t.Error("jsonlTo = nil error, want one for a non-struct type")
	}
}
