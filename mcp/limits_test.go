package mcp

import (
	"strings"
	"testing"
)

func TestRowsWithin(t *testing.T) {
	for _, tc := range []struct {
		in      int
		want    int
		wantErr bool
	}{
		{0, defaultRows, false},
		{1, 1, false},
		{maxRows, maxRows, false},
		{maxRows + 1, 0, true},
		{-5, 0, true},
	} {
		got, err := rowsWithin(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("rowsWithin(%d) error = %v, wantErr %v", tc.in, err, tc.wantErr)
		}
		if err == nil && got != tc.want {
			t.Fatalf("rowsWithin(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// The error has to name the limit and what to do instead. A model told only
// "too many" retries with another number it cannot justify either.
func TestRowLimitErrorIsActionable(t *testing.T) {
	_, err := rowsWithin(maxRows + 1)
	if err == nil {
		t.Fatal("no error")
	}
	for _, want := range []string{"1000", "synth gen"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestInputWithin(t *testing.T) {
	if err := inputWithin("small"); err != nil {
		t.Fatalf("a small input was rejected: %v", err)
	}
	if err := inputWithin(strings.Repeat("x", maxInputBytes+1)); err == nil {
		t.Fatal("an oversized input was accepted")
	}
}
