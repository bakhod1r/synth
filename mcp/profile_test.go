package mcp

import (
	"strings"
	"testing"
)

const sampleCSV = "email,age,city\n" +
	"a@example.com,31,Tashkent\n" +
	"b@example.com,44,Samarkand\n" +
	"c@example.com,27,Bukhara\n"

func TestProfileInfersASpec(t *testing.T) {
	out, err := handleProfile(profileArgs{Data: sampleCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	res := out.(profileResult)
	if !strings.Contains(res.Spec, "email") {
		t.Fatalf("the inferred spec does not mention the email column:\n%s", res.Spec)
	}
	if res.Rows != 3 {
		t.Fatalf("profiled %d rows, want 3", res.Rows)
	}
	if len(res.Columns) != 3 {
		t.Fatalf("got stats for %d columns, want 3", len(res.Columns))
	}
}

// The inferred spec must be usable as-is, or profile is a report rather than a
// step in a workflow.
func TestProfileSpecGenerates(t *testing.T) {
	out, err := handleProfile(profileArgs{Data: sampleCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	spec := out.(profileResult).Spec
	rows, err := handleGenerate(generateArgs{Spec: spec, Rows: 3, Seed: 1})
	if err != nil {
		t.Fatalf("the inferred spec does not generate: %v\n%s", err, spec)
	}
	if len(rows.([]map[string]any)) != 3 {
		t.Fatal("wrong row count from the inferred spec")
	}
}

// The name ends up in a file someone reads months later; "data" is a poor one.
func TestProfileHonoursTheTableName(t *testing.T) {
	out, _ := handleProfile(profileArgs{Data: sampleCSV, Name: "customers"})
	if spec := out.(profileResult).Spec; !strings.Contains(spec, "customers") {
		t.Fatalf("the name was ignored:\n%s", spec)
	}
}

func TestProfileReadsJSONL(t *testing.T) {
	data := `{"email":"a@example.com","age":31}` + "\n" +
		`{"email":"b@example.com","age":44}` + "\n"
	out, err := handleProfile(profileArgs{Data: data, Format: "jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if out.(profileResult).Rows != 2 {
		t.Fatal("wrong row count from JSONL")
	}
}

func TestProfileRejectsAnUnknownFormat(t *testing.T) {
	if _, err := handleProfile(profileArgs{Data: sampleCSV, Format: "parquet"}); err == nil {
		t.Fatal("an unknown format was accepted")
	}
}

func TestProfileRejectsOversizedInput(t *testing.T) {
	if err := inputWithin(strings.Repeat("x", maxInputBytes+1)); err == nil {
		t.Fatal("the limit is not enforced")
	}
	if _, err := handleProfile(profileArgs{Data: strings.Repeat("x", maxInputBytes+1)}); err == nil {
		t.Fatal("an oversized dataset was accepted")
	}
}
