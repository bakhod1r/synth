package synth

import (
	"os"
	"time"

	"github.com/bakhod1r/synth/ddlfe"
	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// DDLTable is a table parsed from SQL DDL, ready to generate rows.
type DDLTable struct {
	name   string
	order  []string
	schema *schema.Schema
}

// LoadDDL parses CREATE TABLE statements from a .sql file. Synth reads the DDL
// as text — it never connects to a database.
func LoadDDL(path string) ([]*DDLTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DDLBytes(data)
}

// DDLBytes parses CREATE TABLE statements from SQL text.
func DDLBytes(sql []byte) ([]*DDLTable, error) {
	tables, err := ddlfe.Parse(string(sql))
	if err != nil {
		return nil, err
	}
	out := make([]*DDLTable, len(tables))
	for i, t := range tables {
		out[i] = &DDLTable{name: t.Name, order: t.Order, schema: t.Schema}
	}
	return out, nil
}

// Name returns the table name.
func (d *DDLTable) Name() string { return d.name }

// Columns returns column names in declaration order.
func (d *DDLTable) Columns() []string { return d.order }

// Generate produces n rows as field→value maps.
func (d *DDLTable) Generate(n int, opts ...Option) ([]map[string]any, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	s := &schema.Schema{Fields: append([]schema.Field(nil), d.schema.Fields...)}
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
	return out, nil
}
