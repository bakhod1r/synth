package synth_test

import (
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

type streamRow struct {
	ID    int
	Email string
	City  string
}

func TestStreamToCSVWritesHeaderAndRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.csv")
	if err := synth.Stream[streamRow](50, synth.WithSeed(1)).ToCSV(path); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 51 {
		t.Fatalf("got %d rows incl. header, want 51", len(recs))
	}
	if strings.Join(recs[0], ",") != "ID,Email,City" {
		t.Fatalf("header = %v", recs[0])
	}
	for _, r := range recs[1:] {
		if r[1] == "" || !strings.Contains(r[1], "@") {
			t.Fatalf("Email not generated: %v", r)
		}
	}
}

func TestStreamToJSONLLineCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	if err := synth.Stream[streamRow](25, synth.WithSeed(2)).ToJSONL(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "\n"); n != 25 {
		t.Fatalf("got %d lines, want 25", n)
	}
}

// Streaming and in-memory generation are seeded identically, so the same seed
// must give the same rows through either path.
func TestStreamMatchesMakeForSameSeed(t *testing.T) {
	want := synth.Make[streamRow](10, synth.WithSeed(99))
	var got []streamRow
	err := synth.Stream[streamRow](10, synth.WithSeed(99)).Each(func(r streamRow) error {
		got = append(got, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d: stream %+v != make %+v", i, got[i], want[i])
		}
	}
}

func TestStreamEachPropagatesCallbackError(t *testing.T) {
	sentinel := errors.New("stop after three")
	seen := 0
	err := synth.Stream[streamRow](1000, synth.WithSeed(3)).Each(func(streamRow) error {
		seen++
		if seen == 3 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if seen != 3 {
		t.Fatalf("callback ran %d times after returning an error, want 3", seen)
	}
}

func TestStreamRejectsNonStruct(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.csv")
	if err := synth.Stream[int](5).ToCSV(path); err == nil {
		t.Error("ToCSV: expected a struct-type error")
	}
	if err := synth.Stream[int](5).ToJSONL(path); err == nil {
		t.Error("ToJSONL: expected a struct-type error")
	}
	if err := synth.Stream[int](5).Each(func(int) error { return nil }); err == nil {
		t.Error("Each: expected a struct-type error")
	}
}

func TestStreamUnwritablePath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "missing", "s.csv")
	if err := synth.Stream[streamRow](1).ToCSV(bad); err == nil {
		t.Error("ToCSV: expected a create error")
	}
	if err := synth.Stream[streamRow](1).ToJSONL(bad); err == nil {
		t.Error("ToJSONL: expected a create error")
	}
}

type badKindRow struct {
	X string `synth:"no-such-kind"`
}

func TestStreamReportsCompileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.csv")
	err := synth.Stream[badKindRow](1).ToCSV(path)
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("err = %v, want an unknown-kind error", err)
	}
	if err := synth.Stream[badKindRow](1).ToJSONL(path); err == nil {
		t.Error("ToJSONL: expected the same compile error")
	}
	if err := synth.Stream[badKindRow](1).Each(func(badKindRow) error { return nil }); err == nil {
		t.Error("Each: expected the same compile error")
	}
}
