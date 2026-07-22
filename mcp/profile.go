package mcp

import (
	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/profile"
)

type profileArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
	// Name becomes the table name in the inferred spec. Without it the spec
	// says "data", which is a poor name to find in a repository later.
	Name string `json:"name"`
	// Rows is the count the inferred spec asks for. Zero means the number of
	// rows profiled, which keeps the generated set the same size as the sample.
	Rows int `json:"rows"`
}

// profileResult pairs the inferred spec with the statistics behind it.
//
// The spec alone hides the guesswork: a column typed as `email` because three
// of four values parsed is a very different claim from one where all ten
// thousand did, and only the statistics say which happened.
type profileResult struct {
	Spec    string                          `json:"spec"`
	Rows    int                             `json:"rows_profiled"`
	Columns map[string]*profile.ColumnStats `json:"columns"`
}

// handleProfile learns a schema from a real dataset, so a caller can generate
// more data shaped like it without keeping the original.
func handleProfile(a profileArgs) (any, error) {
	if err := inputWithin(a.Data); err != nil {
		return nil, err
	}
	p, err := synth.ProfileBytes([]byte(a.Data), a.Format)
	if err != nil {
		return nil, err
	}
	name := a.Name
	if name == "" {
		name = "data"
	}
	count := a.Rows
	if count == 0 {
		count = p.SampleRows()
	}
	spec, err := p.YAML(name, count)
	if err != nil {
		return nil, err
	}
	return profileResult{
		Spec:    string(spec),
		Rows:    p.SampleRows(),
		Columns: p.Stats(),
	}, nil
}
