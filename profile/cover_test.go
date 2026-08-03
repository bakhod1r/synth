package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	csvp := filepath.Join(dir, "d.csv")
	os.WriteFile(csvp, []byte("a,b\n1,x\n"), 0o644)
	if _, err := Load(csvp); err != nil {
		t.Fatal(err)
	}
	jp := filepath.Join(dir, "d.jsonl")
	os.WriteFile(jp, []byte(`{"a":1}`+"\n"), 0o644)
	if _, err := Load(jp); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(dir, "missing.csv")); err == nil {
		t.Fatal("missing file should error")
	}
}

func TestFromCSVMalformed(t *testing.T) {
	if _, err := FromCSV(strings.NewReader("a,b\n\"unterminated,2\n")); err == nil {
		t.Fatal("malformed CSV row should error")
	}
}

func TestFromJSONLMalformed(t *testing.T) {
	if _, err := FromJSONL(strings.NewReader("{bad json}\n")); err == nil {
		t.Fatal("malformed JSON should error")
	}
}

func TestValueOrEmptyAndObserveNil(t *testing.T) {
	if valueOrEmpty(nil) != "" {
		t.Fatal("nil -> empty string")
	}
	if valueOrEmpty(5) != 5 {
		t.Fatal("non-nil passes through")
	}
	observe(nil, "x") // must not panic
}

func TestBuildSkipsMissingColumn(t *testing.T) {
	// order names a column with no stats -> skipped.
	res := build([]string{"ghost"}, map[string]*ColumnStats{}, 0)
	if res == nil {
		t.Fatal("build nil")
	}
}

func TestTrueShareEmpty(t *testing.T) {
	if got := trueShare(&ColumnStats{Values: map[string]int{}}); got != 0.5 {
		t.Fatalf("empty trueShare = %v", got)
	}
}

func TestDetectByValuePhone(t *testing.T) {
	c := &ColumnStats{Values: map[string]int{"+1 555 123 4567": 3}}
	if k := detectByValue(c); k != schema.KindPhone {
		t.Fatalf("phone detect = %v", k)
	}
}

func TestSortChoicesOrders(t *testing.T) {
	f := &schema.Field{Choices: []string{"c", "a", "b"}, Weights: []float64{1, 2, 3}}
	sortChoices(f)
	if f.Choices[0] != "a" || f.Choices[2] != "c" {
		t.Fatalf("sortChoices did not order: %v", f.Choices)
	}
	// Weights follow their value.
	if f.Weights[0] != 2 {
		t.Fatalf("weight not carried with value: %v", f.Weights)
	}
}
