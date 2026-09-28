package synth_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

type plainRow struct{ Name string }

func TestNegativeCountIsAnError(t *testing.T) {
	if _, err := synth.TryMake[plainRow](-1); err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("TryMake(-1) err = %v, want a negative-count error", err)
	}
	if _, err := synth.MakeParallel[plainRow](-1, 2); err == nil {
		t.Fatal("MakeParallel(-1) accepted")
	}
}

type sqlRow struct {
	Ratio float64
	Path  string
	Note  *string
}

// NaN and Inf are not SQL literals.
func TestWriteSQLNonFiniteAndBackslash(t *testing.T) {
	path := t.TempDir() + "/o.sql"
	rows := []sqlRow{{Ratio: math.NaN(), Path: `C:\tmp`}, {Ratio: math.Inf(1), Path: "x"}}
	if err := synth.WriteSQL(path, "t", rows); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b := string(raw)
	if strings.Contains(b, "NaN") || strings.Contains(b, "Inf") {
		t.Fatalf("non-finite float written as a bare literal:\n%s", b)
	}
	if strings.Count(b, "NULL") < 2 {
		t.Fatalf("non-finite floats and a nil pointer should be NULL:\n%s", b)
	}
}
