package pgcopy

import (
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/schema"
)

// allTypesRow exercises every branch of the type switches.
var allTypesRow = map[string]any{
	"s": "text", "bytes": []byte("raw"), "bt": true, "bf": false,
	"i": int(1), "i32": int32(2), "i64": int64(3),
	"f64": float64(1.5), "f32": float32(2.5), "t": time.Unix(0, 0).UTC(),
}

var allCols = []string{"s", "bytes", "bt", "bf", "i", "i32", "i64", "f64", "f32", "t"}

func TestBinaryAllTypes(t *testing.T) {
	var b strings.Builder
	w := NewBinary(&b, allCols)
	if err := w.WriteRow(allTypesRow); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTextAllTypes(t *testing.T) {
	var b strings.Builder
	w := NewText(&b, allCols)
	if err := w.WriteRow(allTypesRow); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

type failWriter struct{ n, at int }

func (f *failWriter) Write(p []byte) (int, error) {
	f.n++
	if f.n >= f.at {
		return 0, errFail
	}
	return len(p), nil
}

type errFailT struct{}

func (errFailT) Error() string { return "boom" }

var errFail = errFailT{}

func TestBinaryErrorPropagation(t *testing.T) {
	// A row that fails to encode sticks the error; later calls short-circuit.
	var b strings.Builder
	w := NewBinary(&b, []string{"bad"})
	if err := w.WriteRow(map[string]any{"bad": struct{}{}}); err == nil {
		t.Fatal("unsupported type should error")
	}
	// err is now set: subsequent WriteRow and Close return it.
	if err := w.WriteRow(map[string]any{"bad": "x"}); err == nil {
		t.Fatal("sticky error on WriteRow")
	}
	if err := w.Close(); err == nil {
		t.Fatal("sticky error on Close")
	}
}

func TestTextWriterFlushErrors(t *testing.T) {
	big := strings.Repeat("x", 6000)
	// Value write overflows bufio into the failing writer.
	if err := NewText(&failWriter{at: 1}, []string{"v"}).WriteRow(map[string]any{"v": big}); err == nil {
		t.Fatal("value write error should surface")
	}
	// A big first column flushes; the tab before the second column then fails.
	if err := NewText(&failWriter{at: 1}, []string{"a", "b"}).
		WriteRow(map[string]any{"a": big, "b": "y"}); err == nil {
		t.Fatal("separator/second-column write error should surface")
	}
	// A big first column flushes; the \N for a missing second column fails.
	if err := NewText(&failWriter{at: 1}, []string{"a", "n"}).
		WriteRow(map[string]any{"a": big}); err == nil {
		t.Fatal("null write error should surface")
	}
}

func TestBinaryCloseTrailerError(t *testing.T) {
	big := strings.Repeat("x", 6000)
	w := NewBinary(&failWriter{at: 2}, []string{"v"})
	_ = w.WriteRow(map[string]any{"v": big})
	if err := w.Close(); err == nil {
		t.Fatal("trailer/flush error should surface")
	}
}

func TestGoTypeForMappings(t *testing.T) {
	cases := map[string]string{
		"int": "int64", "int32": "int64", "float32": "float64", "uuid.UUID": "string",
	}
	for goType, want := range cases {
		f := schema.Field{Name: "c", GoType: goType, Kind: schema.KindInt}
		if got := goTypeFor(f); got != want {
			t.Fatalf("goTypeFor(%q) = %q, want %q", goType, got, want)
		}
	}
}
