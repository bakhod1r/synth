// Package constraint mines cross-column invariants from a real sample and
// enforces them during generation.
//
// Profiling learns what each column looks like on its own. That is not enough
// to make a dataset hold together: a table can pass every per-column check
// while its `total` disagrees with the sum of its line items, or while a
// refunded order has no refund timestamp. This package learns those
// relationships and keeps them true in generated data.
//
// Mining is deliberately conservative. A candidate invariant is generated,
// then falsified against the sample; only what survives is reported, along
// with the number of rows it held over, so a human can judge it.
package constraint

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Kind names the shape of an invariant.
type Kind string

const (
	// Ordering: Left <= Right, for numeric or timestamp columns.
	Ordering Kind = "ordering"
	// SumEquals: the sum of Parts equals Whole.
	SumEquals Kind = "sum"
	// Implication: when When == Equals, Then is non-null.
	Implication Kind = "implication"
	// Range: Lo <= Left <= Hi.
	Range Kind = "range"
)

// Constraint is one mined invariant. Only the fields relevant to its Kind are
// populated.
type Constraint struct {
	Kind Kind

	// Ordering and Range.
	Left  string
	Right string
	Lo    float64
	Hi    float64

	// SumEquals.
	Parts []string
	Whole string

	// Implication.
	When   string
	Equals string
	Then   string
	// Exclusive records that in the sample, Then was non-null *only* when the
	// trigger matched. That is a stronger fact than the implication alone, and
	// it is what lets enforcement null the column out for non-matching rows.
	Exclusive bool

	// Support is how many rows the invariant was checked against and held.
	Support int
}

// String renders a constraint the way it reads in a spec comment.
func (c Constraint) String() string {
	switch c.Kind {
	case Ordering:
		return fmt.Sprintf("%s <= %s", c.Left, c.Right)
	case SumEquals:
		return fmt.Sprintf("%s = %s", strings.Join(c.Parts, " + "), c.Whole)
	case Implication:
		s := fmt.Sprintf("%s = %q => %s is not null", c.When, c.Equals, c.Then)
		if c.Exclusive {
			s += " (and only then)"
		}
		return s
	case Range:
		return fmt.Sprintf("%g <= %s <= %g", c.Lo, c.Left, c.Hi)
	}
	return string(c.Kind)
}

// Holds reports whether rec satisfies the constraint. A row missing a column
// the constraint mentions satisfies it vacuously: the invariant says nothing
// about data that is not there.
func (c Constraint) Holds(rec map[string]any) bool {
	switch c.Kind {
	case Ordering:
		l, okL := toNum(rec[c.Left])
		r, okR := toNum(rec[c.Right])
		if !okL || !okR {
			return true
		}
		return l <= r

	case SumEquals:
		whole, ok := toNum(rec[c.Whole])
		if !ok {
			return true
		}
		sum := 0.0
		for _, p := range c.Parts {
			v, ok := toNum(rec[p])
			if !ok {
				return true
			}
			sum += v
		}
		return nearlyEqual(sum, whole)

	case Implication:
		if !isNull(rec[c.Then]) {
			// The exclusive form also forbids a value when the trigger is absent.
			if c.Exclusive {
				return asString(rec[c.When]) == c.Equals
			}
			return true
		}
		return asString(rec[c.When]) != c.Equals

	case Range:
		v, ok := toNum(rec[c.Left])
		if !ok {
			return true
		}
		return v >= c.Lo && v <= c.Hi
	}
	return true
}

// nearlyEqual compares with a relative tolerance, so a sum invariant is not
// hidden by floating-point representation error.
func nearlyEqual(a, b float64) bool {
	scale := math.Max(math.Max(math.Abs(a), math.Abs(b)), 1)
	return math.Abs(a-b) <= 1e-9*scale
}

// isNull treats SQL NULL, a missing key and an empty CSV cell alike: exports
// round-trip NULL as an empty field, and a rule learned from one form has to
// apply to the other.
func isNull(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// toNum converts a value to a comparable number. Timestamps become Unix
// seconds so ordering invariants work on dates as well as amounts — which is
// where they matter most.
func toNum(v any) (float64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case time.Time:
		return float64(x.UnixNano()) / 1e9, true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		for _, layout := range timeLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return float64(t.UnixNano()) / 1e9, true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}
