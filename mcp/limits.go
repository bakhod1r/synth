package mcp

import "fmt"

// maxRows caps a generate call.
//
// The rows come back inside the response, so an unbounded request would put a
// million rows into a model's context window: slow, expensive, and useless,
// because the model cannot read them. A thousand rows is enough to see the
// shape of a dataset; anything larger belongs in a file the CLI writes.
const maxRows = 1000

// maxInputBytes caps a dataset handed to verify, profile or mask.
const maxInputBytes = 8 << 20 // 8 MiB

// defaultRows is what an unset row count means. Small on purpose: a caller who
// wants more will say so, and one who forgot to say gets something they can
// read rather than something they have to scroll past.
const defaultRows = 10

// rowsWithin validates a requested row count, defaulting an unset one.
func rowsWithin(n int) (int, error) {
	if n == 0 {
		return defaultRows, nil
	}
	if n < 0 {
		return 0, fmt.Errorf("rows must be positive, got %d", n)
	}
	if n > maxRows {
		return 0, fmt.Errorf("rows is %d but this tool returns at most %d — "+
			"the rows travel back in the response, so a larger set belongs in a "+
			"file: run `synth gen -n %d -o data.csv` instead", n, maxRows, n)
	}
	return n, nil
}

// inputWithin rejects a dataset too large to process in a response.
func inputWithin(s string) error {
	if len(s) > maxInputBytes {
		return fmt.Errorf("input is %d bytes but this tool accepts at most %d — "+
			"pass a sample, or run the synth CLI on the full file", len(s), maxInputBytes)
	}
	return nil
}
