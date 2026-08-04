package mcp

import (
	"strings"
	"testing"
)

// Every tool here takes the data inline. A caller passing a path instead of
// rows must be told, not handed an empty result that looks like success.

const csvA = "id,city,amount\n1,Tashkent,100\n2,Samarkand,120\n3,Bukhara,140\n"

func TestDiffFindsAChangedShape(t *testing.T) {
	changed := "id,city,amount\n1,Tashkent,100000\n2,Samarkand,120000\n3,Bukhara,140000\n"
	res, err := handleDiff(diffArgs{A: csvA, B: changed, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	d := res.(diffResult)
	if len(d.Findings) == 0 {
		t.Fatal("a tenfold range shift produced no findings")
	}
	if !strings.Contains(d.Summary, "warning") {
		t.Fatalf("summary = %q", d.Summary)
	}
}

func TestDiffIdenticalDatasetsAreQuiet(t *testing.T) {
	res, err := handleDiff(diffArgs{A: csvA, B: csvA, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	d := res.(diffResult)
	if len(d.Findings) != 0 || d.Errors != 0 || d.Warnings != 0 {
		t.Fatalf("identical data produced findings: %+v", d)
	}
	if d.Summary != "0 error(s), 0 warning(s)" {
		t.Fatalf("summary = %q", d.Summary)
	}
}

func TestDiffReportsWhichSideIsBad(t *testing.T) {
	_, err := handleDiff(diffArgs{A: "", B: csvA, Format: "csv"})
	if err == nil || !strings.Contains(err.Error(), "dataset a") {
		t.Errorf("err = %v, want the failing side named", err)
	}
	_, err = handleDiff(diffArgs{A: csvA, B: "   ", Format: "csv"})
	if err == nil || !strings.Contains(err.Error(), "dataset b") {
		t.Errorf("err = %v, want the failing side named", err)
	}
	_, err = handleDiff(diffArgs{A: strings.Repeat("x", maxInputBytes+1), B: csvA})
	if err == nil {
		t.Error("an oversized dataset should be refused")
	}
}

func TestDiffAcceptsJSONL(t *testing.T) {
	jsonl := "{\"a\":1}\n{\"a\":2}\n"
	res, err := handleDiff(diffArgs{A: jsonl, B: jsonl, Format: "jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.(diffResult).Findings) != 0 {
		t.Fatal("identical JSONL produced findings")
	}
}

// The masking key is required rather than defaulted: a default would silently
// keep two exports linkable, which is the property masking is bought for.
func TestMaskKeyRequirementIsExplained(t *testing.T) {
	_, err := handleMask(maskArgs{Data: "email\na@b.com\n", Format: "csv"})
	if err == nil || !strings.Contains(err.Error(), "key is required") {
		t.Fatalf("err = %v, want the key requirement explained", err)
	}
}

func TestMaskRejectsBadInput(t *testing.T) {
	if _, err := handleMask(maskArgs{Data: "   ", Key: "k"}); err == nil {
		t.Error("empty data should be refused")
	}
	if _, err := handleMask(maskArgs{Data: strings.Repeat("x", maxInputBytes+1), Key: "k"}); err == nil {
		t.Error("an oversized input should be refused")
	}
	if _, err := handleMask(maskArgs{Data: "a\n1\n", Key: "k", Format: "parquet"}); err == nil {
		t.Error("an unknown format should be refused")
	}
	if _, err := handleMask(maskArgs{Data: "{not json\n", Key: "k", Format: "jsonl"}); err == nil {
		t.Error("malformed JSONL should be refused")
	}
}

func TestMaskCSVAndJSONL(t *testing.T) {
	res, err := handleMask(maskArgs{
		Data: "email,city\nreal.person@example.com,Tashkent\n", Key: "k1", Format: "csv",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := res.(maskResult)
	if out.Rows != 1 {
		t.Fatalf("Rows = %d", out.Rows)
	}
	if strings.Contains(out.Data, "real.person") {
		t.Fatalf("the real address survived: %q", out.Data)
	}
	if out.Masked["email"] != 1 {
		t.Fatalf("Masked = %v", out.Masked)
	}

	res, err = handleMask(maskArgs{
		Data: `{"email":"real.person@example.com"}`, Key: "k1", Format: "jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.(maskResult).Data, "real.person") {
		t.Fatal("the real address survived the JSONL path")
	}
}

const snapSpecMore = `name: users
count: 10
fields:
  id:    { kind: uuid, pk: true }
  email: { kind: email }
`

func TestSnapshotAtAnInstantAndOverAWindow(t *testing.T) {
	res, err := handleSnapshot(snapshotArgs{
		Spec: snapSpecMore, Rows: 20, Seed: 1,
		Start: "2026-01-01", Window: "30d", At: "2026-01-15",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := res.(snapshotResult)
	if out.Start != "2026-01-01" || out.Window != "30d" {
		t.Fatalf("the request was not echoed back: %+v", out)
	}
	if out.Events != nil {
		t.Fatal("an at= query returned events")
	}

	res, err = handleSnapshot(snapshotArgs{
		Spec: snapSpecMore, Rows: 20, Seed: 1,
		Start: "2026-01-01", Window: "30d", From: "2026-01-05", To: "2026-01-20",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.(snapshotResult).Rows != nil {
		t.Fatal("a range query returned rows instead of events")
	}
}

func TestSnapshotArgumentErrors(t *testing.T) {
	cases := []struct {
		name string
		args snapshotArgs
		want string
	}{
		{"neither at nor a range", snapshotArgs{Spec: snapSpecMore}, "either at="},
		{"both at and a range", snapshotArgs{Spec: snapSpecMore, At: "2026-01-01", From: "2026-01-01", To: "2026-01-02"}, "either at="},
		{"half a range", snapshotArgs{Spec: snapSpecMore, From: "2026-01-01"}, "go together"},
		{"unparsable spec", snapshotArgs{Spec: "::: not yaml :::", At: "2026-01-01"}, "does not parse"},
		{"bad instant", snapshotArgs{Spec: snapSpecMore, At: "yesterday"}, ""},
		{"bad start", snapshotArgs{Spec: snapSpecMore, At: "2026-01-01", Start: "yesterday"}, ""},
		{"bad window", snapshotArgs{Spec: snapSpecMore, At: "2026-01-01", Window: "a while"}, ""},
		{"bad from", snapshotArgs{Spec: snapSpecMore, From: "nope", To: "2026-01-02"}, ""},
		{"bad to", snapshotArgs{Spec: snapSpecMore, From: "2026-01-01", To: "nope"}, ""},
		{"to before from", snapshotArgs{Spec: snapSpecMore, From: "2026-02-01", To: "2026-01-01"}, "before"},
		{"oversized spec", snapshotArgs{Spec: strings.Repeat("x", maxInputBytes+1), At: "2026-01-01"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := handleSnapshot(c.args)
			if err == nil {
				t.Fatal("expected an error")
			}
			if c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to mention %q", err, c.want)
			}
		})
	}
}
