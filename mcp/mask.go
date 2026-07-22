package mcp

import (
	"fmt"
	"strings"

	"github.com/bakhod1r/synth/mask"
)

type maskArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
	Key    string `json:"key"`
	Locale string `json:"locale"`
}

type maskResult struct {
	Data    string         `json:"data"`
	Rows    int            `json:"rows"`
	Masked  map[string]int `json:"masked"`
	Columns []string       `json:"columns"`
	// Untouched names the columns nothing was recognized in. It is the part
	// worth reading: a column of personal data under a name the detector does
	// not know is exactly what a caller would otherwise ship unmasked.
	Untouched []string `json:"untouched"`
}

// handleMask replaces personal data in a real export with generated values of
// the same shape, keeping the file usable as a fixture.
//
// The key is required rather than defaulted. With a default, two exports masked
// on different days would silently stay linkable — which is the one property
// the caller most likely believes they are buying by masking at all.
func handleMask(a maskArgs) (any, error) {
	if err := inputWithin(a.Data); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Data) == "" {
		return nil, fmt.Errorf("data is empty — pass the rows themselves, not a file path")
	}
	if a.Key == "" {
		return nil, fmt.Errorf("key is required: it makes replacements stable, so " +
			"foreign keys still match across related exports. Use the same key for " +
			"related dumps, and a fresh one to make two dumps unlinkable")
	}

	m := mask.New(a.Key, a.Locale)
	var out strings.Builder
	var (
		rep *mask.Report
		err error
	)
	switch strings.ToLower(strings.TrimPrefix(a.Format, ".")) {
	case "jsonl", "ndjson", "json":
		rep, err = m.JSONL(strings.NewReader(a.Data), &out)
	case "", "csv":
		rep, err = m.CSV(strings.NewReader(a.Data), &out)
	default:
		return nil, fmt.Errorf("unknown format %q — use csv or jsonl", a.Format)
	}
	if err != nil {
		return nil, err
	}
	return maskResult{
		Data:      out.String(),
		Rows:      rep.Rows,
		Masked:    rep.Masked,
		Columns:   rep.Columns,
		Untouched: rep.Untouched,
	}, nil
}
