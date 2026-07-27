package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/bakhod1r/synth/diff"
	"github.com/bakhod1r/synth/profile"
)

// runDiff compares the shape of two datasets and exits non-zero when they
// differ structurally, so a CI job can guard against a generator or a real feed
// drifting. It reads the two files; it changes nothing.
//
//	synth diff old.csv new.csv
//	synth diff old.csv new.csv --tolerance 0.2 --format json
func runDiff(args []string) error {
	var paths []string
	opts := diff.Options{}
	format := "text"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tolerance":
			i++
			if i >= len(args) {
				return fmt.Errorf("diff: --tolerance needs a value")
			}
			t, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				return fmt.Errorf("diff: --tolerance %q: %w", args[i], err)
			}
			opts.Tolerance = t
		case "-f", "--format":
			i++
			if i >= len(args) {
				return fmt.Errorf("diff: --format needs a value")
			}
			format = args[i]
		default:
			paths = append(paths, args[i])
		}
	}
	if len(paths) != 2 {
		return fmt.Errorf("diff: give two files to compare (synth diff a.csv b.csv)")
	}

	a, err := profile.Load(paths[0])
	if err != nil {
		return fmt.Errorf("diff: %s: %w", paths[0], err)
	}
	b, err := profile.Load(paths[1])
	if err != nil {
		return fmt.Errorf("diff: %s: %w", paths[1], err)
	}
	findings := diff.Compare(a, b, opts)

	if format == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
			return err
		}
	} else {
		printDiff(findings)
	}

	// Exit non-zero only on structural breaks, so a CI step fails on those and
	// passes on drift warnings — the same contract verify uses.
	if diff.Errors(findings) > 0 {
		os.Exit(1)
	}
	return nil
}

// printDiff writes the findings in reading order with a one-line summary. The
// leading glyph mirrors a unified diff: - removed, + added, ~ changed.
func printDiff(fs []diff.Finding) {
	for _, f := range fs {
		glyph := "~"
		switch {
		case f.Detail == "column removed":
			glyph = "-"
		case f.Detail == "column added":
			glyph = "+"
		}
		col := f.Column
		if col == "" {
			col = "(dataset)"
		}
		fmt.Printf("%s %-16s %s: %s\n", glyph, col, f.Severity, f.Detail)
	}
	e, w := diff.Errors(fs), diff.Warns(fs)
	if e == 0 && w == 0 {
		fmt.Fprintln(os.Stderr, "no shape differences")
		return
	}
	fmt.Fprintf(os.Stderr, "%d error(s), %d warning(s)\n", e, w)
}
