package mcp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/cdc"
	"github.com/bakhodir/synth/snapshot"
)

// snapshotArgs asks one of two questions: what did the table look like at an
// instant (at=), or what changed between two (from=/to=). They are different
// questions, so setting both is an error rather than a silent preference.
type snapshotArgs struct {
	Spec   string `json:"spec"`
	At     string `json:"at"`
	From   string `json:"from"`
	To     string `json:"to"`
	Locale string `json:"locale"`
	Rows   int    `json:"rows"`
	Seed   uint64 `json:"seed"`
	// Start is when the table came into existence, and Window the span over
	// which rows are born and change. They matter more than they look: an
	// instant outside [Start, Start+Window] returns nothing, which reads as a
	// bug and is not one.
	Start  string `json:"start"`
	Window string `json:"window"`
}

type snapshotResult struct {
	// Start and Window are echoed back so a caller who got an empty result can
	// see whether they simply asked outside the table's lifetime.
	Start  string           `json:"start"`
	Window string           `json:"window"`
	Rows   []map[string]any `json:"rows,omitempty"`
	Events []cdc.Event      `json:"events,omitempty"`
}

func handleSnapshot(a snapshotArgs) (any, error) {
	hasAt := a.At != ""
	hasRange := a.From != "" || a.To != ""
	if hasAt == hasRange {
		return nil, fmt.Errorf("set either at= for the state at one instant, " +
			"or from= and to= for the changes between two — not both")
	}
	if hasRange && (a.From == "" || a.To == "") {
		return nil, fmt.Errorf("from= and to= go together; set both")
	}
	n, err := rowsWithin(a.Rows)
	if err != nil {
		return nil, err
	}
	if err := inputWithin(a.Spec); err != nil {
		return nil, err
	}
	spec, err := synth.YAMLBytes([]byte(a.Spec))
	if err != nil {
		return nil, fmt.Errorf("the spec does not parse: %w", err)
	}

	cfg := snapshot.Config{
		Table:  spec.Name(),
		Rows:   n,
		Seed:   a.Seed,
		Locale: a.Locale,
	}
	if a.Start != "" {
		if cfg.Start, err = parseInstant(a.Start); err != nil {
			return nil, err
		}
	}
	if a.Window != "" {
		if cfg.Window, err = parseWindow(a.Window); err != nil {
			return nil, err
		}
	}

	tl, err := snapshot.New(spec.Schema(), cfg)
	if err != nil {
		return nil, err
	}
	out := snapshotResult{Start: a.Start, Window: a.Window}

	if hasAt {
		when, err := parseInstant(a.At)
		if err != nil {
			return nil, err
		}
		out.Rows = tl.At(when)
		return out, nil
	}
	from, err := parseInstant(a.From)
	if err != nil {
		return nil, err
	}
	to, err := parseInstant(a.To)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, fmt.Errorf("to= (%s) is before from= (%s)", a.To, a.From)
	}
	out.Events = tl.Between(from, to)
	return out, nil
}

// parseInstant accepts a date or a full RFC 3339 timestamp.
//
// It rejects anything else rather than guessing: a misread date produces a
// plausible-looking snapshot of the wrong moment, and nothing downstream would
// ever reveal the mistake.
func parseInstant(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot read %q as a date — use 2026-07-01 "+
		"or 2026-07-01T12:00:00Z", s)
}

// parseWindow accepts Go durations plus a day suffix, because a table's
// lifetime is naturally written in days and time.ParseDuration stops at hours.
func parseWindow(s string) (time.Duration, error) {
	if days, ok := strings.CutSuffix(strings.TrimSpace(s), "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("cannot read %q as a window — use 180d or 720h", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("cannot read %q as a window — use 180d or 720h", s)
	}
	return d, nil
}
