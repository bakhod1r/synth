package mcp

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/verify"
)

// Real Luhn-valid numbers.
const goodCSV = "id,card\n1,4539578763621486\n2,4556737586899855\n"

// A mostly-valid column with one broken number. It has to be mostly valid on
// purpose: verify treats a column that fails nearly everywhere as misidentified
// rather than broken, which is the right call — a "card" column holding order
// references should not produce one finding per row.
const badCSV = "id,card\n1,4539578763621486\n2,4556737586899855\n" +
	"3,4532015112830366\n4,4539578763621480\n"

func TestVerifyAcceptsValidData(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: goodCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if rep := out.(verify.Report); !rep.OK() {
		t.Fatalf("valid data reported findings: %+v", rep.Findings)
	}
}

func TestVerifyFindsBrokenChecksums(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: badCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if out.(verify.Report).OK() {
		t.Fatal("invalid card numbers were reported as clean")
	}
}

func TestVerifyReadsJSONL(t *testing.T) {
	data := `{"id":1,"card":"4539578763621486"}` + "\n"
	out, err := handleVerify(verifyArgs{Data: data, Format: "jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.(verify.Report).OK() {
		t.Fatal("valid JSONL reported findings")
	}
}

func TestVerifyDefaultsToCSV(t *testing.T) {
	if _, err := handleVerify(verifyArgs{Data: goodCSV}); err != nil {
		t.Fatalf("an unset format was not treated as CSV: %v", err)
	}
}

func TestVerifyRejectsOversizedInput(t *testing.T) {
	_, err := handleVerify(verifyArgs{Data: strings.Repeat("x", maxInputBytes+1), Format: "csv"})
	if err == nil {
		t.Fatal("an oversized dataset was accepted")
	}
}

func TestVerifyRejectsEmptyInput(t *testing.T) {
	if _, err := handleVerify(verifyArgs{Data: "   ", Format: "csv"}); err == nil {
		t.Fatal("empty data was accepted")
	}
}

// A path is not data. Accepting one would turn a data generator into a
// file-reading primitive for a model reading untrusted text.
func TestVerifyDoesNotReadAPath(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: "/etc/passwd", Format: "csv"})
	if err != nil {
		return // rejecting it outright is fine
	}
	if rep := out.(verify.Report); rep.Rows > 1 {
		t.Fatalf("the tool appears to have read a file: %d rows", rep.Rows)
	}
}

// The report must round-trip through the transport, or the caller sees an empty
// object where the findings should be.
func TestVerifyReportSurvivesEncoding(t *testing.T) {
	out, _ := handleVerify(verifyArgs{Data: badCSV, Format: "csv"})
	res, err := result(out, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "card") {
		t.Fatalf("the encoded report does not name the broken column:\n%s", text)
	}
}
