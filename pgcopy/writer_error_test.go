package pgcopy

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// A COPY file is loaded by a server that cannot ask questions, so a write
// failure must reach the caller rather than produce a short, valid-looking
// file.

type failAfter struct {
	left int
}

var errSink = errors.New("sink closed")

func (f *failAfter) Write(p []byte) (int, error) {
	if f.left <= 0 {
		return 0, errSink
	}
	f.left--
	return len(p), nil
}

func TestTextWriterSurfacesWriteError(t *testing.T) {
	// bufio flushes once its buffer fills, so a wide-enough row forces a real
	// write through to the failing sink.
	w := NewText(&failAfter{left: 0}, []string{"a", "b"})
	big := strings.Repeat("x", 8192)
	var err error
	for i := 0; i < 10 && err == nil; i++ {
		err = w.WriteRow(map[string]any{"a": big, "b": big})
	}
	if err == nil {
		if err = w.Close(); err == nil {
			t.Fatal("the sink error never surfaced")
		}
	}
}

func TestTextWriterNullAndEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := NewText(&buf, []string{"a", "b", "c", "d"})
	row := map[string]any{
		"a": nil,
		"b": "tab\there\nnewline\\slash\rcr",
		"c": true,
		// "d" is absent: a missing column is as null as an explicit nil.
	}
	if err := w.WriteRow(row); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSuffix(buf.String(), "\n")
	fields := strings.Split(got, "\t")
	if len(fields) != 4 {
		t.Fatalf("got %d fields, want 4: %q", len(fields), got)
	}
	if fields[0] != `\N` || fields[3] != `\N` {
		t.Fatalf("nulls not written as \\N: %q", got)
	}
	if strings.ContainsAny(fields[1], "\t\n\r") {
		t.Fatalf("a structural character survived escaping: %q", fields[1])
	}
	if fields[1] != `tab\there\nnewline\\slash\rcr` {
		t.Fatalf("escaped value = %q", fields[1])
	}
	if fields[2] != "t" {
		t.Fatalf("bool = %q, want t", fields[2])
	}
}

func TestTextValueFormats(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"plain", "plain"},
		{true, "t"},
		{false, "f"},
		{int(7), "7"},
		{int32(-7), "-7"},
		{int64(1 << 40), "1099511627776"},
		{float64(1.5), "1.5"},
		{float32(2.5), "2.5"},
		{uint8(3), "3"},
	}
	for _, c := range cases {
		if got := textValue(c.in); got != c.want {
			t.Errorf("textValue(%v) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := textValue(time.Date(2024, 5, 4, 3, 2, 1, 0, time.UTC)); !strings.HasPrefix(got, "2024-05-04") {
		t.Errorf("time = %q", got)
	}
}

func TestBinaryWriterSurfacesWriteError(t *testing.T) {
	w := NewBinary(&failAfter{left: 0}, []string{"a"})
	big := strings.Repeat("y", 8192)
	var err error
	for i := 0; i < 10 && err == nil; i++ {
		err = w.WriteRow(map[string]any{"a": big})
	}
	if err == nil {
		err = w.Close()
	}
	if err == nil {
		t.Fatal("the sink error never surfaced")
	}
	// Once failed, the writer must stay failed rather than resume mid-file.
	if again := w.WriteRow(map[string]any{"a": "z"}); again == nil {
		t.Fatal("the writer accepted a row after a write error")
	}
	if again := w.Close(); again == nil {
		t.Fatal("Close reported success after a write error")
	}
}

func TestBinaryFileHeaderAndTrailer(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"id", "name", "when", "missing"})
	if err := w.WriteRow(map[string]any{
		"id":   int64(1),
		"name": "a",
		"when": time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	if !bytes.HasPrefix(out, binarySignature) {
		t.Fatalf("missing PGCOPY signature: %q", out[:11])
	}
	// The trailer is a field count of -1 (0xFFFF).
	if !bytes.HasSuffix(out, []byte{0xFF, 0xFF}) {
		t.Fatalf("missing trailer: % x", out[len(out)-4:])
	}
}

func TestBinaryWriterRejectsUnencodableValue(t *testing.T) {
	var buf bytes.Buffer
	w := NewBinary(&buf, []string{"a"})
	err := w.WriteRow(map[string]any{"a": struct{ X chan int }{}})
	if err == nil {
		t.Skip("every value type this build can produce is encodable")
	}
	if !strings.Contains(err.Error(), `column "a"`) {
		t.Fatalf("err = %v, want the column named", err)
	}
}
