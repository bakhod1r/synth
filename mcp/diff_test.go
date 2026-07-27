package mcp

import "testing"

func TestDiffIdentical(t *testing.T) {
	csv := "id,amount\n1,100\n2,200\n"
	out, err := handleDiff(diffArgs{A: csv, B: csv, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	r := out.(diffResult)
	if len(r.Findings) != 0 || r.Errors != 0 {
		t.Errorf("identical datasets differ: %+v", r)
	}
}

func TestDiffFindsAddedColumn(t *testing.T) {
	a := "id\n1\n2\n"
	b := "id,tier\n1,gold\n2,silver\n"
	out, err := handleDiff(diffArgs{A: a, B: b, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	r := out.(diffResult)
	if r.Errors == 0 {
		t.Errorf("an added column should be an error, got %+v", r)
	}
}

func TestDiffEmptyDatasetErrors(t *testing.T) {
	if _, err := handleDiff(diffArgs{A: "", B: "id\n1\n", Format: "csv"}); err == nil {
		t.Error("an empty dataset should error")
	}
}
