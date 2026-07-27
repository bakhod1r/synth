package gen

import (
	"math"
	"testing"
	"time"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// A time-series field with a fixed axis timestamp evaluates
// base + trend*t_days + amplitude*sin(2π*t_sec/period) + noise. With trend and
// noise off it is a pure sine, checked at t=0, quarter and half period.
func tsSchema(axisTime time.Time, params map[string]string) *schema.Schema {
	iso := axisTime.Format(time.RFC3339)
	p := map[string]string{"axis": "ts", "base": "40", "trend": "0", "amplitude": "20",
		"period": "24h", "noise": "0", "start": "2026-01-01T00:00:00Z", "min": "-1000", "max": "1000"}
	for k, v := range params {
		p[k] = v
	}
	return &schema.Schema{Fields: []schema.Field{
		// pin ts to a constant by min==max as an RFC3339 range is awkward; instead
		// use a fixed 'at' param the time provider reads via from is not available,
		// so we set the axis through a from-less time with equal min/max dates.
		{Name: "ts", Kind: schema.KindTime, GoType: "time.Time",
			Params: map[string]string{"min": iso, "max": iso}},
		{Name: "cpu", Kind: schema.KindTimeSeries, GoType: "float64", Params: p},
	}}
}

func tsValue(t *testing.T, axis time.Time, params map[string]string) float64 {
	t.Helper()
	e, err := Compile(tsSchema(axis, params), "en_US")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Record(rng.New(1), 0)
	got, ok := rec["cpu"].(float64)
	if !ok {
		t.Fatalf("cpu is %T, want float64", rec["cpu"])
	}
	return got
}

func TestTimeSeriesPureSine(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		off  time.Duration
		want float64
	}{
		{0, 40},              // sin(0)=0
		{6 * time.Hour, 60},  // quarter period, sin(π/2)=1 → 40+20
		{12 * time.Hour, 40}, // half period, sin(π)=0
		{18 * time.Hour, 20}, // three-quarter, sin(3π/2)=-1 → 40-20
	}
	for _, c := range cases {
		got := tsValue(t, start.Add(c.off), nil)
		if math.Abs(got-c.want) > 1e-6 {
			t.Errorf("at +%s: got %v, want %v", c.off, got, c.want)
		}
	}
}

// Trend alone is linear in days from start.
func TestTimeSeriesTrend(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := tsValue(t, start.Add(48*time.Hour),
		map[string]string{"amplitude": "0", "trend": "1.5"})
	if want := 40.0 + 1.5*2; math.Abs(got-want) > 1e-6 {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A non-time axis is a compile-time error.
func TestTimeSeriesNonTimeAxisErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "ts", Kind: schema.KindName, GoType: "string", Params: map[string]string{}},
		{Name: "cpu", Kind: schema.KindTimeSeries, GoType: "float64",
			Params: map[string]string{"axis": "ts", "base": "1"}},
	}}
	if _, err := Compile(s, "en_US"); err == nil {
		t.Error("expected an error for a non-time axis")
	}
}

// min/max clamp the series (a CPU percentage stays in range).
func TestTimeSeriesClamps(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := tsValue(t, start.Add(6*time.Hour),
		map[string]string{"amplitude": "1000", "max": "100"})
	if got != 100 {
		t.Errorf("got %v, want clamped to 100", got)
	}
}
