package constraint

// maxPasses bounds the repair loop. Repairs interact — swapping two columns to
// fix one ordering can break another — so a single pass is not enough. Each
// pass is a sorting step over a partial order, which converges quickly; the cap
// exists so a pathological or contradictory constraint set cannot spin.
const maxPasses = 8

// Enforce repairs a record until it satisfies every constraint.
//
// Repair rather than reject: rejecting and resampling would make generation
// cost unbounded, and for these invariants there is always an obvious fix.
// The generated values are treated as the facts and the dependent value is
// recomputed from them — a total is derived from its parts, not the reverse.
//
// Repairs are applied in passes until the record is stable. Enforce returns
// true when every constraint holds. It returns false when the constraints
// contradict each other and no fixed point exists within maxPasses; the record
// is left in its best repaired state, and the caller can report the set as
// unsatisfiable rather than emit data that quietly violates it.
func Enforce(cs []Constraint, rec map[string]any) bool {
	for pass := 0; pass < maxPasses; pass++ {
		for _, c := range cs {
			switch c.Kind {
			case Ordering:
				enforceOrdering(c, rec)
			case SumEquals:
				enforceSum(c, rec)
			case Implication:
				enforceImplication(c, rec)
			case Range:
				enforceRange(c, rec)
			}
		}
		if AllHold(cs, rec) {
			return true
		}
	}
	return false
}

// AllHold reports whether rec satisfies every constraint.
func AllHold(cs []Constraint, rec map[string]any) bool {
	for _, c := range cs {
		if !c.Holds(rec) {
			return false
		}
	}
	return true
}

// enforceOrdering swaps the two values when they are the wrong way round.
// Swapping preserves both generated values and their distribution — clamping
// one to the other would pile up duplicates at the boundary.
func enforceOrdering(c Constraint, rec map[string]any) {
	l, okL := toNum(rec[c.Left])
	r, okR := toNum(rec[c.Right])
	if !okL || !okR || l <= r {
		return
	}
	rec[c.Left], rec[c.Right] = rec[c.Right], rec[c.Left]
}

// enforceSum recomputes the whole from its parts.
func enforceSum(c Constraint, rec map[string]any) {
	if _, ok := rec[c.Whole]; !ok {
		return
	}
	sum := 0.0
	for _, p := range c.Parts {
		v, ok := toNum(rec[p])
		if !ok {
			return // a missing part means there is nothing to derive from
		}
		sum += v
	}
	rec[c.Whole] = sum
}

// enforceImplication clears a column that should not be populated.
//
// Only the exclusive form can be repaired. When the sample showed the target
// populated *only* alongside the trigger, a non-matching row must have it
// empty, and emptying it is safe. The non-exclusive form says nothing about
// non-matching rows, so nothing is changed; and when the trigger does match,
// the generator has already produced a value, so there is nothing to fill.
func enforceImplication(c Constraint, rec map[string]any) {
	if !c.Exclusive {
		return
	}
	if _, ok := rec[c.Then]; !ok {
		return
	}
	if asString(rec[c.When]) != c.Equals {
		rec[c.Then] = nil
	}
}

// enforceRange clamps a value into the observed bounds.
func enforceRange(c Constraint, rec map[string]any) {
	v, ok := toNum(rec[c.Left])
	if !ok {
		return
	}
	switch {
	case v < c.Lo:
		rec[c.Left] = c.Lo
	case v > c.Hi:
		rec[c.Left] = c.Hi
	}
}
