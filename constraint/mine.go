package constraint

import "sort"

// Mining limits. Candidate generation is combinatorial, so it is bounded:
// beyond these sizes the search cost outgrows the value of what it finds.
const (
	maxNumericCols = 12 // for sum candidates
	maxDistinct    = 20 // a column with more values is not a category
	minGroupRows   = 20 // an implication needs real evidence, not one row
	minGroupFrac   = 0.05
)

// Mine finds invariants that hold across at least minSupport of the sample
// (0..1). It reads rows only — the same shape the profiler already loads from
// a CSV or JSONL export.
//
// Every candidate must survive falsification against the whole sample, and
// every reported constraint carries the row count it held over.
func Mine(rows []map[string]any, minSupport float64) []Constraint {
	if len(rows) == 0 {
		return nil
	}
	if minSupport <= 0 || minSupport > 1 {
		minSupport = 1
	}
	cols := columns(rows)
	num := numericColumns(rows, cols)

	var out []Constraint
	out = append(out, mineOrdering(rows, num, minSupport)...)
	out = append(out, mineSums(rows, num, minSupport)...)
	out = append(out, mineImplications(rows, cols, minSupport)...)
	out = append(out, mineRanges(rows, num)...)
	return out
}

// columns returns every column name seen, in sorted order so mining is
// deterministic regardless of Go's map iteration.
func columns(rows []map[string]any) []string {
	seen := map[string]bool{}
	for _, r := range rows {
		for k := range r {
			seen[k] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// numericColumns keeps the columns that parse as numbers (or timestamps) in
// the overwhelming majority of non-null rows. A column that is numeric only
// sometimes is a string column with digits in it, not a measurement.
func numericColumns(rows []map[string]any, cols []string) []string {
	var out []string
	for _, c := range cols {
		present, numeric := 0, 0
		for _, r := range rows {
			if isNull(r[c]) {
				continue
			}
			present++
			if _, ok := toNum(r[c]); ok {
				numeric++
			}
		}
		if present > 0 && float64(numeric)/float64(present) >= 0.99 {
			out = append(out, c)
		}
	}
	return out
}

// mineOrdering tests A <= B for every ordered pair. A pair where the two
// columns are always equal is skipped: that is a duplication, not an ordering,
// and reporting both directions would be noise.
func mineOrdering(rows []map[string]any, num []string, minSupport float64) []Constraint {
	var out []Constraint
	for _, a := range num {
		for _, b := range num {
			if a == b {
				continue
			}
			held, checked, strict := 0, 0, 0
			for _, r := range rows {
				x, okX := toNum(r[a])
				y, okY := toNum(r[b])
				if !okX || !okY {
					continue
				}
				checked++
				if x <= y {
					held++
				}
				if x < y {
					strict++
				}
			}
			if checked == 0 || strict == 0 {
				continue
			}
			if float64(held)/float64(checked) >= minSupport {
				out = append(out, Constraint{
					Kind: Ordering, Left: a, Right: b, Support: held,
				})
			}
		}
	}
	return out
}

// mineSums tests whether some pair or triple of numeric columns adds up to
// another. Only exact-arithmetic relationships survive, within a relative
// epsilon for floating-point error.
func mineSums(rows []map[string]any, num []string, minSupport float64) []Constraint {
	if len(num) > maxNumericCols {
		num = num[:maxNumericCols]
	}
	var out []Constraint
	for _, whole := range num {
		for i := 0; i < len(num); i++ {
			if num[i] == whole {
				continue
			}
			for j := i + 1; j < len(num); j++ {
				if num[j] == whole {
					continue
				}
				parts := []string{num[i], num[j]}
				if c, ok := checkSum(rows, parts, whole, minSupport); ok {
					out = append(out, c)
					continue // a pair explains it; no need for a triple
				}
				for k := j + 1; k < len(num); k++ {
					if num[k] == whole {
						continue
					}
					triple := []string{num[i], num[j], num[k]}
					if c, ok := checkSum(rows, triple, whole, minSupport); ok {
						out = append(out, c)
					}
				}
			}
		}
	}
	return out
}

func checkSum(rows []map[string]any, parts []string, whole string, minSupport float64) (Constraint, bool) {
	held, checked := 0, 0
	for _, r := range rows {
		w, ok := toNum(r[whole])
		if !ok {
			continue
		}
		sum, complete := 0.0, true
		for _, p := range parts {
			v, ok := toNum(r[p])
			if !ok {
				complete = false
				break
			}
			sum += v
		}
		if !complete {
			continue
		}
		checked++
		if nearlyEqual(sum, w) {
			held++
		}
	}
	if checked == 0 || float64(held)/float64(checked) < minSupport {
		return Constraint{}, false
	}
	return Constraint{
		Kind: SumEquals, Parts: append([]string(nil), parts...),
		Whole: whole, Support: held,
	}, true
}

// mineImplications finds rules of the form "when this category has this value,
// that column is populated".
//
// Two guards keep this from inventing rules. The trigger group must be large
// enough to be evidence, and the target column must be null somewhere outside
// the group — otherwise the column is simply always populated and the trigger
// has nothing to do with it.
func mineImplications(rows []map[string]any, cols []string, minSupport float64) []Constraint {
	minGroup := minGroupRows
	if n := int(float64(len(rows)) * minGroupFrac); n > minGroup {
		minGroup = n
	}

	var out []Constraint
	for _, when := range cols {
		values := distinctValues(rows, when)
		if len(values) == 0 {
			continue
		}
		for _, val := range values {
			for _, then := range cols {
				if then == when {
					continue
				}
				if c, ok := checkImplication(rows, when, val, then, minGroup, minSupport); ok {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

func checkImplication(rows []map[string]any, when, val, then string, minGroup int, minSupport float64) (Constraint, bool) {
	group, populated := 0, 0
	outsideNull, outsidePopulated := 0, 0
	for _, r := range rows {
		if asString(r[when]) == val {
			group++
			if !isNull(r[then]) {
				populated++
			}
			continue
		}
		if isNull(r[then]) {
			outsideNull++
		} else {
			outsidePopulated++
		}
	}
	if group < minGroup {
		return Constraint{}, false
	}
	if float64(populated)/float64(group) < minSupport {
		return Constraint{}, false
	}
	// The target must be missing somewhere else, or the rule is vacuous.
	if outsideNull == 0 {
		return Constraint{}, false
	}
	return Constraint{
		Kind: Implication, When: when, Equals: val, Then: then,
		Exclusive: outsidePopulated == 0,
		Support:   populated,
	}, true
}

// distinctValues returns a categorical column's values, or nil if the column
// has too many distinct values to be a category.
func distinctValues(rows []map[string]any, col string) []string {
	seen := map[string]int{}
	for _, r := range rows {
		v, ok := r[col]
		if !ok || isNull(v) {
			continue
		}
		if _, isStr := v.(string); !isStr {
			return nil // categories are strings; numbers are measurements
		}
		seen[asString(v)]++
		if len(seen) > maxDistinct {
			return nil
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// mineRanges records each numeric column's observed bounds.
func mineRanges(rows []map[string]any, num []string) []Constraint {
	var out []Constraint
	for _, c := range num {
		lo, hi, n := 0.0, 0.0, 0
		for _, r := range rows {
			v, ok := toNum(r[c])
			if !ok {
				continue
			}
			if n == 0 || v < lo {
				lo = v
			}
			if n == 0 || v > hi {
				hi = v
			}
			n++
		}
		if n == 0 || lo == hi {
			continue
		}
		out = append(out, Constraint{Kind: Range, Left: c, Lo: lo, Hi: hi, Support: n})
	}
	return out
}
