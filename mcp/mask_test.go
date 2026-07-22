package mcp

import (
	"strings"
	"testing"
)

const personalCSV = "name,email,phone\n" +
	"John Smith,john@example.com,+1-212-555-0143\n" +
	"Jane Doe,jane@example.com,+1-212-555-0144\n"

func TestMaskReplacesPersonalData(t *testing.T) {
	out, err := handleMask(maskArgs{Data: personalCSV, Format: "csv", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	got := out.(maskResult).Data
	for _, secret := range []string{"John Smith", "john@example.com", "Jane Doe"} {
		if strings.Contains(got, secret) {
			t.Errorf("%q survived masking:\n%s", secret, got)
		}
	}
	if !strings.HasPrefix(got, "name,email,phone") {
		t.Fatalf("the header was not preserved:\n%s", got)
	}
}

// The same key must give the same replacement, or a masked export loses its
// joins and stops being usable as a fixture.
func TestMaskIsStableForTheSameKey(t *testing.T) {
	data := "email\njohn@example.com\njohn@example.com\n"
	out, err := handleMask(maskArgs{Data: data, Format: "csv", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.(maskResult).Data), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want a header and two rows", len(lines))
	}
	if lines[1] != lines[2] {
		t.Fatalf("the same value masked two different ways: %q vs %q", lines[1], lines[2])
	}
}

// A different key must make two exports unlinkable.
func TestDifferentKeysAreUnlinkable(t *testing.T) {
	data := "email\njohn@example.com\n"
	a, _ := handleMask(maskArgs{Data: data, Format: "csv", Key: "k1"})
	b, _ := handleMask(maskArgs{Data: data, Format: "csv", Key: "k2"})
	if a.(maskResult).Data == b.(maskResult).Data {
		t.Fatal("two different keys produced the same output")
	}
}

// Masking without a key would produce output that looks safe and is not
// reproducible; requiring one makes the choice explicit.
func TestMaskRequiresAKey(t *testing.T) {
	if _, err := handleMask(maskArgs{Data: personalCSV, Format: "csv"}); err == nil {
		t.Fatal("masking without a key was accepted")
	}
}

// The report must say which columns were left alone. A column of personal data
// under a name the detector does not know is exactly what would otherwise ship
// unmasked.
func TestMaskReportsUntouchedColumns(t *testing.T) {
	data := "email,internal_note\na@example.com,call back Tuesday\n"
	out, err := handleMask(maskArgs{Data: data, Format: "csv", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	res := out.(maskResult)
	if len(res.Untouched) == 0 {
		t.Fatal("no column was reported as untouched, though one was not recognized")
	}
}

func TestMaskReadsJSONL(t *testing.T) {
	data := `{"email":"john@example.com"}` + "\n"
	out, err := handleMask(maskArgs{Data: data, Format: "jsonl", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.(maskResult).Data, "john@example.com") {
		t.Fatal("the original email survived masking")
	}
}

func TestMaskRejectsAnUnknownFormat(t *testing.T) {
	if _, err := handleMask(maskArgs{Data: personalCSV, Format: "parquet", Key: "k"}); err == nil {
		t.Fatal("an unknown format was accepted")
	}
}

func TestMaskRejectsEmptyInput(t *testing.T) {
	if _, err := handleMask(maskArgs{Data: "  ", Key: "k"}); err == nil {
		t.Fatal("empty data was accepted")
	}
}
