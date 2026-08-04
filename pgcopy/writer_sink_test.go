package pgcopy

import (
	"errors"
	"strings"
	"testing"
)

// failSink fails every write. bufio absorbs the first 4096 bytes, so a test
// only reaches these paths by writing more than that.
type failSink struct{}

func (failSink) Write(p []byte) (int, error) { return 0, errSink }

// COPY output goes down a pipe into psql, which can die mid-load. Every write
// in WriteRow has to report that, because the alternative is telling the caller
// a partial COPY succeeded — and a partial COPY is a partially loaded table.
//
// bufio only reaches the sink when its 4096-byte buffer fills, so each case
// sizes its rows to make a specific write the one that overflows.
func TestTextWriteRowReportsSinkErrors(t *testing.T) {
	t.Run("separator", func(t *testing.T) {
		// 8 rows of two 240-byte values leave the buffer at 3856 bytes; the
		// next value fills it to exactly 4096, so the tab after it is the write
		// that flushes and fails.
		w := NewText(failSink{}, []string{"a", "b"})
		val := strings.Repeat("x", 240)
		row := map[string]any{"a": val, "b": val}
		var err error
		for i := 0; i < 9 && err == nil; i++ {
			err = w.WriteRow(row)
		}
		if !errors.Is(err, errSink) {
			t.Fatalf("WriteRow = %v, want the sink's error from the separator write", err)
		}
	})

	t.Run("null", func(t *testing.T) {
		// A missing column writes the two-byte \N marker; enough of them
		// overflow the buffer on their own.
		cols := make([]string, 4096)
		for i := range cols {
			cols[i] = "missing"
		}
		w := NewText(failSink{}, cols)
		if err := w.WriteRow(nil); !errors.Is(err, errSink) {
			t.Fatalf("WriteRow = %v, want the sink's error from a null write", err)
		}
	})
}

// A binary writer that has already failed stays failed, and Close must report
// that rather than the trailer's success — otherwise a truncated file passes
// for a complete one.
func TestBinaryCloseReportsStickyError(t *testing.T) {
	b := NewBinary(failSink{}, []string{"a"})
	// An unsupported value type sets the sticky error without touching the sink.
	if err := b.WriteRow(map[string]any{"a": struct{ X int }{1}}); err == nil {
		t.Fatal("WriteRow = nil error, want one for a value with no binary encoding")
	}
	first := b.WriteRow(map[string]any{"a": "later"})
	if err := b.Close(); err == nil || err.Error() != first.Error() {
		t.Fatalf("Close = %v, want the sticky error %v", err, first)
	}
}

// The trailer is itself a write, and it can be the one that overflows the
// buffer. Close reports that too: a file missing its trailer is exactly what
// the trailer exists to distinguish.
func TestBinaryCloseReportsTrailerWriteError(t *testing.T) {
	b := NewBinary(failSink{}, []string{"a"})
	// The 19-byte header plus a 6-byte field prefix and a 4070-byte value fill
	// the buffer to 4095, leaving the two-byte trailer to flush and fail.
	if err := b.WriteRow(map[string]any{"a": strings.Repeat("x", 4070)}); err != nil {
		t.Fatalf("WriteRow: %v", err)
	}
	if err := b.Close(); !errors.Is(err, errSink) {
		t.Fatalf("Close = %v, want the sink's error from the trailer write", err)
	}
}
