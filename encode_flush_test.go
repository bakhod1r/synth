package synth

import (
	"errors"
	"strings"
	"testing"
)

type flushFailWriter struct{}

func (flushFailWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

type flushRow struct{ Name string }

// csv.Writer buffers, so a write error surfaces at the final Flush. A deferred
// Flush threw it away and a truncated file came back as success.
func TestEncodeCSVReportsFlushError(t *testing.T) {
	if err := encodeCSV(flushFailWriter{}, []flushRow{{"a"}}); err == nil {
		t.Fatal("encodeCSV swallowed the write error")
	}
}

type ptrRow struct {
	Name *string
	Age  *int
}

func TestSQLValueDereferencesPointers(t *testing.T) {
	name := "O'Neil"
	var b strings.Builder
	if err := encodeSQL(&b, "people", []ptrRow{{Name: &name}}); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "'O''Neil'") || !strings.Contains(got, "NULL") || strings.Contains(got, "0x") {
		t.Fatalf("sql = %s", got)
	}
}
