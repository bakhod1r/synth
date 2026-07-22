package mcp

import (
	"fmt"
	"testing"
	"time"
)

const snapSpec = "name: t\nfields:\n" +
	"  id: { kind: uuid, pk: true }\n" +
	"  amount: { kind: amount }\n" +
	"  status: { kind: orderstatus }\n"

func TestSnapshotAtAnInstant(t *testing.T) {
	out, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 20, Seed: 1, At: "2026-07-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.(snapshotResult).Rows) == 0 {
		t.Fatal("the snapshot is empty")
	}
}

// The two forms are different questions and must not be mixed in one call.
func TestSnapshotRejectsBothForms(t *testing.T) {
	_, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5,
		At: "2026-01-01", From: "2026-01-01", To: "2026-07-01"})
	if err == nil {
		t.Fatal("a call with both at= and from=/to= was accepted")
	}
	if _, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5}); err == nil {
		t.Fatal("a call with neither form was accepted")
	}
	if _, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5, From: "2026-01-01"}); err == nil {
		t.Fatal("a half-specified range was accepted")
	}
}

func TestSnapshotBetweenReturnsEvents(t *testing.T) {
	out, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 50, Seed: 2,
		From: "2026-01-01", To: "2026-07-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.(snapshotResult).Events) == 0 {
		t.Fatal("no change events between two instants six months apart")
	}
}

// Two instants far apart must differ, or the tool is not worth calling.
func TestTwoInstantsDiffer(t *testing.T) {
	early, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 30, Seed: 3, At: "2026-02-01"})
	if err != nil {
		t.Fatal(err)
	}
	late, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 30, Seed: 3, At: "2026-11-01"})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(early) == fmt.Sprint(late) {
		t.Fatal("two instants nine months apart produced identical state")
	}
}

func TestSnapshotIsReproducible(t *testing.T) {
	a, _ := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 20, Seed: 4, At: "2026-06-01"})
	b, _ := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 20, Seed: 4, At: "2026-06-01"})
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Fatal("the same seed and instant gave different state")
	}
}

func TestSnapshotRejectsABadDate(t *testing.T) {
	for _, bad := range []string{"last Tuesday", "07/01/2026", ""} {
		args := snapshotArgs{Spec: snapSpec, Rows: 5, At: bad}
		if bad == "" {
			args.At, args.From, args.To = "", "nonsense", "2026-01-01"
		}
		if _, err := handleSnapshot(args); err == nil {
			t.Fatalf("an unparseable date %q was accepted", bad)
		}
	}
}

func TestSnapshotRejectsAReversedRange(t *testing.T) {
	_, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5,
		From: "2026-07-01", To: "2026-01-01"})
	if err == nil {
		t.Fatal("a range ending before it starts was accepted")
	}
}

func TestParseWindow(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want time.Duration
	}{
		{"180d", 180 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"720h", 720 * time.Hour},
		{"90m", 90 * time.Minute},
	} {
		got, err := parseWindow(tc.in)
		if err != nil {
			t.Fatalf("parseWindow(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseWindow(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"", "soon", "-5d", "0d", "d"} {
		if _, err := parseWindow(bad); err == nil {
			t.Fatalf("parseWindow(%q) was accepted", bad)
		}
	}
}

// An instant before the table existed returns nothing. That is correct, and the
// result echoes start= so a caller can tell it apart from a broken spec.
func TestSnapshotBeforeStartIsEmptyButExplained(t *testing.T) {
	out, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 20, Seed: 5,
		Start: "2026-06-01", Window: "90d", At: "2026-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	res := out.(snapshotResult)
	if len(res.Rows) != 0 {
		t.Fatalf("got %d rows before the table existed", len(res.Rows))
	}
	if res.Start != "2026-06-01" {
		t.Fatalf("the result does not echo start=, so an empty result looks like a bug")
	}
}

func TestSnapshotRejectsABadWindow(t *testing.T) {
	_, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5, At: "2026-07-01", Window: "soon"})
	if err == nil {
		t.Fatal("an unparseable window was accepted")
	}
}
