package synth_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

// Exhausted is a struct whose unique column has two possible values, so any
// run longer than two rows must fail rather than repeat itself.
type exhausted struct {
	Status string `synth:"enum,choices=new|open,unique"`
}

func wantExhaustion(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an exhaustion error, got nil")
	}
	if !strings.Contains(err.Error(), "ran out of unique values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Every generating surface reports exhaustion; a duplicate must not reach the
// output file, the callback, or the returned slice.
func TestExhaustionReachesStreamCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	wantExhaustion(t, synth.Stream[exhausted](50, synth.WithSeed(1)).ToCSV(path))
}

func TestExhaustionReachesStreamJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.jsonl")
	wantExhaustion(t, synth.Stream[exhausted](50, synth.WithSeed(1)).ToJSONL(path))
}

func TestExhaustionReachesStreamEach(t *testing.T) {
	err := synth.Stream[exhausted](50, synth.WithSeed(1)).Each(func(exhausted) error { return nil })
	wantExhaustion(t, err)
}

// A rate stream has no end to check at, so it must stop at the failing record.
func TestExhaustionStopsRateStream(t *testing.T) {
	cfg := synth.RateConfig{PerSecond: 100000, Total: 50}
	seen := 0
	err := synth.Rate[exhausted](cfg, synth.WithSeed(1)).
		Run(context.Background(), func(exhausted) error { seen++; return nil })
	wantExhaustion(t, err)
	if seen >= 50 {
		t.Fatalf("stream should have stopped early, emitted %d", seen)
	}
}

// A BOOLEAN column has two values, so a UNIQUE constraint on it cannot hold
// past two rows.
func TestExhaustionReachesDDL(t *testing.T) {
	tables, err := synth.DDLBytes([]byte(`CREATE TABLE t (flag BOOLEAN UNIQUE);`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tables[0].Generate(50, synth.WithSeed(1))
	wantExhaustion(t, err)
}

func TestExhaustionReachesYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	spec := `count: 50
fields:
  status: { kind: enum, choices: [new, open], unique: true }
`
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	y, err := synth.LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.Generate(synth.WithSeed(1))
	wantExhaustion(t, err)
}

// The same YAML column in counter mode has no such ceiling.
func TestYAMLCounterMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	spec := `count: 500
fields:
  status: { kind: enum, choices: [new, open], unique: true, unique_mode: counter }
`
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	y, err := synth.LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate(synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[any]bool{}
	for _, r := range rows {
		if seen[r["status"]] {
			t.Fatalf("duplicate status %v", r["status"])
		}
		seen[r["status"]] = true
	}
}
