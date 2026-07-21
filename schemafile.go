package synth

import (
	"time"

	"github.com/bakhodir/synth/gen"
	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/schema"
	"github.com/bakhodir/synth/schemafe"
)

// SchemaFile is a record definition parsed from a JSON Schema or Avro schema.
type SchemaFile struct {
	tbl *schemafe.Table
}

// LoadSchema parses a JSON Schema or Avro schema file (detected by content).
func LoadSchema(path string) (*SchemaFile, error) {
	t, err := schemafe.Load(path)
	if err != nil {
		return nil, err
	}
	return &SchemaFile{tbl: t}, nil
}

// JSONSchemaBytes parses a JSON Schema document.
func JSONSchemaBytes(data []byte) (*SchemaFile, error) {
	t, err := schemafe.ParseJSONSchema(data)
	if err != nil {
		return nil, err
	}
	return &SchemaFile{tbl: t}, nil
}

// AvroBytes parses an Avro record schema.
func AvroBytes(data []byte) (*SchemaFile, error) {
	t, err := schemafe.ParseAvro(data)
	if err != nil {
		return nil, err
	}
	return &SchemaFile{tbl: t}, nil
}

// Name returns the record name (JSON Schema title / Avro record name).
func (s *SchemaFile) Name() string { return s.tbl.Name }

// Columns returns field names in declaration order.
func (s *SchemaFile) Columns() []string { return s.tbl.Order }

// Generate produces n records matching the schema.
func (s *SchemaFile) Generate(n int, opts ...Option) ([]map[string]any, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	sc := &schema.Schema{Fields: append([]schema.Field(nil), s.tbl.Schema.Fields...)}
	applyWeighted(sc, cfg.weighted)
	eng, err := gen.Compile(sc, cfg.locale)
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
