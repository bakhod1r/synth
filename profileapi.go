package synth

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/bakhod1r/synth/constraint"
	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/profile"
	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/yamlfe"
)

// Profiled is a schema learned from a real-data sample (see the profile
// package). Generating from it produces synthetic rows whose shape — types,
// ranges and category frequencies — matches the sample, without ever copying
// the original data.
type Profiled struct {
	res *profile.Result
	// cons are cross-column invariants mined from the same sample. They are
	// what a per-column profile cannot see: a total agreeing with its parts,
	// a timestamp ordering, a status that implies a populated column.
	cons []constraint.Constraint
}

// Profile learns a schema from a CSV or JSONL export of real data. Synth reads
// the file only; it never connects to a database.
func Profile(path string) (*Profiled, error) {
	r, err := profile.Load(path)
	if err != nil {
		return nil, err
	}
	p := &Profiled{res: r}
	// Mining re-reads a bounded sample: profiling streams and keeps only
	// statistics, but invariants are relationships between whole rows.
	sample, err := constraint.LoadSample(path, constraint.DefaultSample)
	if err != nil {
		return nil, err
	}
	p.cons = constraint.Mine(sample, 1.0)
	return p, nil
}

// ProfileBytes is Profile for data already in memory, with the format named
// rather than read from a file extension. Callers that must not touch the
// filesystem — the MCP server is one — use this.
//
// It takes bytes rather than an io.Reader because profiling and constraint
// mining each need their own pass over the data: profiling streams and keeps
// only statistics, while an invariant is a relationship between whole rows. A
// single reader cannot be consumed twice, and buffering it here would hide the
// memory cost from the caller who chose to hold the data in the first place.
//
// format is "csv" (the default) or "jsonl"/"ndjson".
func ProfileBytes(data []byte, format string) (*Profiled, error) {
	var (
		r   *profile.Result
		err error
	)
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "jsonl", "ndjson", "json":
		r, err = profile.FromJSONL(bytes.NewReader(data))
	case "", "csv":
		r, err = profile.FromCSV(bytes.NewReader(data))
	default:
		return nil, fmt.Errorf("unknown format %q — use csv or jsonl", format)
	}
	if err != nil {
		return nil, err
	}
	sample, err := constraint.ReadSample(bytes.NewReader(data), format, constraint.DefaultSample)
	if err != nil {
		return nil, err
	}
	return &Profiled{res: r, cons: constraint.Mine(sample, 1.0)}, nil
}

// Columns returns the profiled column names in file order.
func (p *Profiled) Columns() []string { return p.res.Order }

// SampleRows returns how many rows were profiled.
func (p *Profiled) SampleRows() int { return p.res.Rows }

// Stats exposes the observed per-column statistics (distinct counts, null
// counts, numeric ranges) so you can inspect what was learned.
func (p *Profiled) Stats() map[string]*profile.ColumnStats { return p.res.Stats }

// Generate produces n synthetic rows matching the profiled shape.
func (p *Profiled) Generate(n int, opts ...Option) ([]map[string]any, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	s := &schema.Schema{Fields: append([]schema.Field(nil), p.res.Schema.Fields...)}
	applyWeighted(s, cfg.weighted)
	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	eng.Chaos = cfg.chaos
	base := rng.New(cfg.seed)
	out := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		out[i] = eng.Record(base, i)
	}
	if err := eng.Err(); err != nil {
		return nil, err
	}
	if err := enforceAll(p.cons, out); err != nil {
		return nil, err
	}
	return out, nil
}

// YAML renders the profiled schema as a YAML spec, the same dialect yamlfe
// parses. This closes the loop: profile a real export once, keep the spec in
// version control, and generate from it forever after without the original
// data.
func (p *Profiled) YAML(name string, count int) ([]byte, error) {
	return yamlfe.Render(p.res.Schema, p.res.Order, name, count, p.cons)
}

// Constraints returns the cross-column invariants mined from the sample.
func (p *Profiled) Constraints() []constraint.Constraint { return p.cons }
