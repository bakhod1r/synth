// Package verify audits an existing dataset.
//
// It is the inverse of generation, built from the same knowledge: the rules
// that make Synth's output coherent — checksum-valid identifiers, resolvable
// foreign keys, timestamps in a sensible order, columns that actually vary —
// are exactly the rules worth checking on data somebody else produced.
//
// Like every other part of Synth, this reads files. It never connects to the
// system the data came from.
//
// The design rule is that a clean dataset produces an empty report. A check
// that fires on correct data is a false positive and a bug in this package,
// not something for the caller to filter out.
package verify

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Severity separates "this data is wrong" from "this data looks suspicious".
type Severity string

const (
	// SevError marks a definite defect: a failed checksum, a dangling key.
	SevError Severity = "error"
	// SevWarn marks something worth a human's attention that may be intended.
	SevWarn Severity = "warn"
)

// Finding is one problem found in the dataset.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Column   string   `json:"column,omitempty"`
	// Row is the zero-based row index, or -1 for a dataset-wide finding.
	Row    int      `json:"row"`
	Detail string   `json:"detail"`
	Sample []string `json:"sample,omitempty"`
}

// Ref describes a foreign key: Column in the audited rows must appear as
// ParentKey somewhere in Parent.
type Ref struct {
	Column    string
	Parent    []map[string]any
	ParentKey string
}

// Options selects which checks run and supplies what they need.
type Options struct {
	// Refs are foreign keys to resolve. Parent rows are read from their own
	// file; nothing is queried.
	Refs []Ref
	// Constraints are invariants to check, usually mined by the constraint
	// package from a trusted sample.
	Constraints []Constraint
	// MaxFindingsPerCheck caps how many rows a single check reports before it
	// collapses into one summary finding. A million broken rows should not
	// produce a million lines. Zero means the default.
	MaxFindingsPerCheck int
	// KAnonymity, when > 1, requires every combination of QuasiIdentifiers to be
	// shared by at least this many rows. A rarer combination re-identifies an
	// individual even with direct identifiers removed.
	KAnonymity int
	// QuasiIdentifiers are the columns whose combination could single someone
	// out — age, ZIP, gender. The k-anonymity check runs only when these are
	// set.
	QuasiIdentifiers []string
}

// Constraint is the subset of constraint.Constraint verify needs. It is
// declared here so the verify package does not force a dependency on callers
// that only want format checks.
type Constraint interface {
	Holds(rec map[string]any) bool
	String() string
}

// DefaultMaxFindings bounds per-check output.
const DefaultMaxFindings = 20

// Report is the result of an audit.
type Report struct {
	Rows     int       `json:"rows"`
	Findings []Finding `json:"findings"`
}

// Errors reports how many findings are definite defects.
func (r Report) Errors() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SevError {
			n++
		}
	}
	return n
}

// OK reports whether the dataset is free of definite defects. Warnings alone
// do not make a dataset invalid, so this is what a CI gate should use.
func (r Report) OK() bool { return r.Errors() == 0 }

// Run audits rows and returns everything it found.
func Run(rows []map[string]any, opts Options) Report {
	rep := Report{Rows: len(rows)}
	if len(rows) == 0 {
		return rep
	}
	if opts.MaxFindingsPerCheck <= 0 {
		opts.MaxFindingsPerCheck = DefaultMaxFindings
	}
	cols := columns(rows)

	rep.Findings = append(rep.Findings, checkFormats(rows, cols, opts)...)
	rep.Findings = append(rep.Findings, checkRefs(rows, opts)...)
	rep.Findings = append(rep.Findings, checkTemporal(rows, cols, opts)...)
	rep.Findings = append(rep.Findings, checkDistribution(rows, cols)...)
	rep.Findings = append(rep.Findings, checkConstraints(rows, opts)...)
	rep.Findings = append(rep.Findings, checkKAnonymity(rows, cols, opts)...)

	// Errors first, then row order — but a dataset-wide finding sorts after the
	// individual rows it summarizes, so a report reads as examples followed by
	// the total rather than the other way round.
	sort.SliceStable(rep.Findings, func(i, j int) bool {
		a, b := rep.Findings[i], rep.Findings[j]
		if a.Severity != b.Severity {
			return a.Severity == SevError
		}
		if (a.Row < 0) != (b.Row < 0) {
			return b.Row < 0
		}
		return a.Row < b.Row
	})
	return rep
}

// Text writes a human-readable report.
func (r Report) Text(w io.Writer) error {
	if len(r.Findings) == 0 {
		_, err := fmt.Fprintf(w, "%d rows checked, no problems found\n", r.Rows)
		return err
	}
	if _, err := fmt.Fprintf(w, "%d rows checked, %d findings (%d errors)\n\n",
		r.Rows, len(r.Findings), r.Errors()); err != nil {
		return err
	}
	for _, f := range r.Findings {
		loc := "dataset"
		if f.Row >= 0 {
			loc = fmt.Sprintf("row %d", f.Row)
		}
		if f.Column != "" {
			loc += ", column " + f.Column
		}
		if _, err := fmt.Fprintf(w, "  %-5s %-12s %s: %s\n",
			f.Severity, f.Check, loc, f.Detail); err != nil {
			return err
		}
		if len(f.Sample) > 0 {
			if _, err := fmt.Fprintf(w, "        e.g. %s\n",
				strings.Join(f.Sample, ", ")); err != nil {
				return err
			}
		}
	}
	return nil
}

// JSON writes the report as JSON, for a CI job that wants to parse it.
func (r Report) JSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// columns returns every column name, sorted so a report is deterministic.
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

// collector accumulates per-row findings for one check and collapses them once
// the cap is reached, so a systematically broken column reports once.
type collector struct {
	check    string
	severity Severity
	column   string
	max      int

	findings []Finding
	total    int
	samples  []string
}

func (c *collector) add(row int, detail, value string) {
	c.total++
	if len(c.samples) < 3 && value != "" {
		c.samples = append(c.samples, value)
	}
	if len(c.findings) < c.max {
		c.findings = append(c.findings, Finding{
			Check: c.check, Severity: c.severity, Column: c.column,
			Row: row, Detail: detail,
		})
	}
}

func (c *collector) result() []Finding {
	if c.total > c.max {
		return append(c.findings, Finding{
			Check: c.check, Severity: c.severity, Column: c.column, Row: -1,
			Detail: fmt.Sprintf("%d rows affected in total (showing the first %d)",
				c.total, c.max),
			Sample: c.samples,
		})
	}
	for i := range c.findings {
		if i == 0 {
			c.findings[i].Sample = c.samples
		}
	}
	return c.findings
}
