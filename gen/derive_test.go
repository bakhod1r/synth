package gen

import (
	"math"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

func deriveSchema(params map[string]string) *schema.Schema {
	p := map[string]string{"derive": "age", "slope": "1200", "intercept": "20000", "max": "1000000"}
	for k, v := range params {
		p[k] = v
	}
	return &schema.Schema{Fields: []schema.Field{
		{Name: "age", Kind: schema.KindInt, GoType: "int",
			Params: map[string]string{"min": "30", "max": "30"}}, // pin age=30
		{Name: "income", Kind: schema.KindFloat, GoType: "float64", Params: p},
	}}
}

// With no noise the derived field is an exact line: slope*age + intercept.
func TestDeriveExactLine(t *testing.T) {
	e, err := Compile(deriveSchema(map[string]string{"noise": "0"}), "en_US")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Record(rng.New(1), 0)
	if rec["age"] != int64(30) && rec["age"] != 30 {
		t.Fatalf("age pin failed: %v", rec["age"])
	}
	got := rec["income"].(float64)
	if want := 1200.0*30 + 20000; math.Abs(got-want) > 1e-9 {
		t.Errorf("income = %v, want %v", got, want)
	}
}

// Noise keeps the mean near the line and scales the spread with magnitude.
func TestDeriveNoiseCentersOnLine(t *testing.T) {
	e, err := Compile(deriveSchema(map[string]string{"noise": "0.1"}), "en_US")
	if err != nil {
		t.Fatal(err)
	}
	line := 1200.0*30 + 20000
	var sum float64
	const n = 4000
	for i := 0; i < n; i++ {
		sum += e.Record(rng.New(uint64(i)), i)["income"].(float64)
	}
	mean := sum / n
	if math.Abs(mean-line)/line > 0.02 {
		t.Errorf("mean %v drifts from the line %v by more than 2%%", mean, line)
	}
}

// min/max still clamp a derived value.
func TestDeriveClamps(t *testing.T) {
	e, err := Compile(deriveSchema(map[string]string{"noise": "0", "max": "50000"}), "en_US")
	if err != nil {
		t.Fatal(err)
	}
	got := e.Record(rng.New(1), 0)["income"].(float64)
	if got != 50000 {
		t.Errorf("income = %v, want clamped to 50000", got)
	}
}

// An int derived field rounds.
func TestDeriveIntRounds(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "age", Kind: schema.KindInt, GoType: "int", Params: map[string]string{"min": "10", "max": "10"}},
		{Name: "score", Kind: schema.KindInt, GoType: "int",
			Params: map[string]string{"derive": "age", "slope": "1.5", "intercept": "0", "noise": "0"}},
	}}
	e, err := Compile(s, "en_US")
	if err != nil {
		t.Fatal(err)
	}
	got := e.Record(rng.New(1), 0)["score"]
	if got != int64(15) && got != 15 {
		t.Errorf("score = %v (%T), want 15", got, got)
	}
}

// Deriving from a non-numeric field is a compile-time error, not a silent zero.
func TestDeriveNonNumericTargetErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "name", Kind: schema.KindName, GoType: "string", Params: map[string]string{}},
		{Name: "x", Kind: schema.KindFloat, GoType: "float64",
			Params: map[string]string{"derive": "name", "slope": "1", "intercept": "0"}},
	}}
	_, err := Compile(s, "en_US")
	if err == nil || !strings.Contains(err.Error(), "numeric") {
		t.Errorf("expected a non-numeric derive error, got %v", err)
	}
}
