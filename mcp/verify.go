package mcp

import (
	"fmt"
	"strings"

	"github.com/bakhodir/synth/constraint"
	"github.com/bakhodir/synth/verify"
)

// verifyArgs takes the dataset itself, never a path. See the package comment
// for why.
type verifyArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

func handleVerify(a verifyArgs) (any, error) {
	rows, err := parseRows(a.Data, a.Format)
	if err != nil {
		return nil, err
	}
	return verify.Run(rows, verify.Options{}), nil
}

// parseRows reads an inline dataset. Shared with snapshot.
func parseRows(data, format string) ([]map[string]any, error) {
	if err := inputWithin(data); err != nil {
		return nil, err
	}
	if strings.TrimSpace(data) == "" {
		return nil, fmt.Errorf("data is empty — pass the rows themselves, not a file path")
	}
	rows, err := constraint.ReadSample(strings.NewReader(data), format, 0)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("the data parsed but produced no rows — " +
			"a CSV needs a header line, and JSONL needs one object per line")
	}
	return rows, nil
}
