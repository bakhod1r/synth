package mcp

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bakhod1r/synth/diff"
	"github.com/bakhod1r/synth/profile"
)

// diffArgs takes both datasets inline, never a path — like every other tool
// here.
type diffArgs struct {
	A         string  `json:"a"`
	B         string  `json:"b"`
	Format    string  `json:"format"`
	Tolerance float64 `json:"tolerance"`
}

// diffResult is the shape comparison: the findings plus a one-line summary so a
// caller can decide without re-counting.
type diffResult struct {
	Findings []diff.Finding `json:"findings"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	Summary  string         `json:"summary"`
}

// handleDiff compares the shape of two inline datasets — columns, types,
// numeric ranges, null rates, category sets — so an assistant holding two
// datasets can ask whether they match, without either touching a file.
func handleDiff(a diffArgs) (any, error) {
	ra, err := profileInline(a.A, a.Format)
	if err != nil {
		return nil, fmt.Errorf("dataset a: %w", err)
	}
	rb, err := profileInline(a.B, a.Format)
	if err != nil {
		return nil, fmt.Errorf("dataset b: %w", err)
	}
	findings := diff.Compare(ra, rb, diff.Options{Tolerance: a.Tolerance})
	e, w := diff.Errors(findings), diff.Warns(findings)
	return diffResult{
		Findings: findings,
		Errors:   e,
		Warnings: w,
		Summary:  fmt.Sprintf("%d error(s), %d warning(s)", e, w),
	}, nil
}

// profileInline profiles an inline dataset into the shape summary diff needs.
func profileInline(data, format string) (*profile.Result, error) {
	if err := inputWithin(data); err != nil {
		return nil, err
	}
	if strings.TrimSpace(data) == "" {
		return nil, fmt.Errorf("empty — pass the rows themselves, not a file path")
	}
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "jsonl", "ndjson", "json":
		return profile.FromJSONL(bytes.NewReader([]byte(data)))
	default:
		return profile.FromCSV(bytes.NewReader([]byte(data)))
	}
}
