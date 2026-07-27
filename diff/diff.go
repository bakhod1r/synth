// Package diff compares the shape of two datasets — their columns, types,
// numeric ranges, null rates and category sets — rather than their rows. It is
// built on the profile package: both sides are profiled, and the two summaries
// are compared. This is what answers "is this file shaped like that one?", the
// question a CI regression guard asks after a generator changes.
package diff

import (
	"fmt"
	"math"

	"github.com/bakhod1r/synth/profile"
)

// Severity ranks a finding. Only Error fails a CI check.
type Severity string

const (
	// Error is a structural break — a column that appeared, vanished, or
	// changed type. A consumer of the old shape breaks on the new one.
	Error Severity = "error"
	// Warn is drift within the same structure — a range, null rate or category
	// set that moved past tolerance. Worth a look, not a hard failure.
	Warn Severity = "warn"
	// Info is drift within tolerance, or the row-count difference.
	Info Severity = "info"
)

// Finding is one difference between the two datasets.
type Finding struct {
	Column   string   `json:"column,omitempty"`
	Severity Severity `json:"severity"`
	Detail   string   `json:"detail"`
}

// Options tunes the comparison.
type Options struct {
	// Tolerance is the fraction a numeric bound may move before it warns.
	// Defaults to 0.10.
	Tolerance float64
	// NullRateTolerance is the change in null fraction that warns. Defaults to
	// 0.05 (five percentage points).
	NullRateTolerance float64
}

func (o Options) tolerance() float64 {
	if o.Tolerance > 0 {
		return o.Tolerance
	}
	return 0.10
}

func (o Options) nullTolerance() float64 {
	if o.NullRateTolerance > 0 {
		return o.NullRateTolerance
	}
	return 0.05
}

// Compare reports how b differs from a, most-structural findings first (a is
// the baseline, b the candidate). An empty result means the shapes match.
func Compare(a, b *profile.Result, opts Options) []Finding {
	var out []Finding

	// Structural: which columns exist on each side, in a's order then b's
	// additions, so the report reads in the baseline's column order.
	seen := map[string]bool{}
	for _, name := range a.Order {
		seen[name] = true
		ca := a.Stats[name]
		cb, ok := b.Stats[name]
		if !ok {
			out = append(out, Finding{name, Error, "column removed"})
			continue
		}
		out = append(out, compareColumn(name, ca, cb, opts)...)
	}
	for _, name := range b.Order {
		if !seen[name] {
			out = append(out, Finding{name, Error, "column added"})
		}
	}

	if a.Rows != b.Rows {
		out = append(out, Finding{"", Info, fmt.Sprintf("row count %d → %d", a.Rows, b.Rows)})
	}
	return out
}

// compareColumn compares one column present on both sides.
func compareColumn(name string, a, b *profile.ColumnStats, opts Options) []Finding {
	// A type flip is the worst case: the same column now carries a different
	// kind of value, which every downstream cast gets wrong.
	if a.Numeric != b.Numeric {
		return []Finding{{name, Error, fmt.Sprintf("type changed: numeric %v → %v", a.Numeric, b.Numeric)}}
	}
	if a.Categorical != b.Categorical {
		return []Finding{{name, Warn, fmt.Sprintf("categorical %v → %v", a.Categorical, b.Categorical)}}
	}

	var out []Finding
	if a.Numeric {
		if d, ok := relDrift(a.Min, b.Min); ok && d > opts.tolerance() {
			out = append(out, Finding{name, Warn, fmt.Sprintf("min %g → %g (%+.0f%%)", a.Min, b.Min, d*100*sign(b.Min-a.Min))})
		}
		if d, ok := relDrift(a.Max, b.Max); ok && d > opts.tolerance() {
			out = append(out, Finding{name, Warn, fmt.Sprintf("max %g → %g (%+.0f%%)", a.Max, b.Max, d*100*sign(b.Max-a.Max))})
		}
	}
	if a.Categorical && b.Categorical {
		if added, removed := categoryDelta(a.Values, b.Values); len(added)+len(removed) > 0 {
			out = append(out, Finding{name, Warn, fmt.Sprintf("category set changed: +%v -%v", added, removed)})
		}
	}

	na, nb := nullRate(a), nullRate(b)
	if math.Abs(na-nb) > opts.nullTolerance() {
		out = append(out, Finding{name, Warn, fmt.Sprintf("null rate %.0f%% → %.0f%%", na*100, nb*100)})
	}
	return out
}

// relDrift is the relative change from x to y, and ok=false when x is zero (a
// relative change from zero is undefined; a bound that moves off zero shows up
// as a category or type change if it matters).
func relDrift(x, y float64) (float64, bool) {
	if x == 0 {
		return 0, false
	}
	return math.Abs(y-x) / math.Abs(x), true
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

// nullRate is the fraction of observed values that were null.
func nullRate(c *profile.ColumnStats) float64 {
	total := c.NonNull + c.Nulls
	if total == 0 {
		return 0
	}
	return float64(c.Nulls) / float64(total)
}

// categoryDelta returns the values gained and lost between two category sets.
func categoryDelta(a, b map[string]int) (added, removed []string) {
	for k := range b {
		if _, ok := a[k]; !ok {
			added = append(added, k)
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			removed = append(removed, k)
		}
	}
	return added, removed
}

// Errors counts the error-level findings — what a CI check keys its exit code
// on.
func Errors(fs []Finding) int {
	n := 0
	for _, f := range fs {
		if f.Severity == Error {
			n++
		}
	}
	return n
}

// Warns counts the warn-level findings.
func Warns(fs []Finding) int {
	n := 0
	for _, f := range fs {
		if f.Severity == Warn {
			n++
		}
	}
	return n
}
