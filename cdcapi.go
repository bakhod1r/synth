package synth

import (
	"os"
	"reflect"

	"github.com/bakhod1r/synth/cdc"
	"github.com/bakhod1r/synth/reflectfe"
	"github.com/bakhod1r/synth/schema"
)

// CDCConfig controls a generated change-event history.
type CDCConfig = cdc.Config

// CDCEvent is one Debezium-shaped change event.
type CDCEvent = cdc.Event

// CDCStream produces insert/update/delete events over a schema.
type CDCStream = cdc.Stream

// CascadeConfig controls a two-table change stream with cascade deletes.
type CascadeConfig = cdc.CascadeConfig

// CascadeStream produces an interleaved change stream over a parent and a child
// table, where deleting a parent deletes its children too.
type CascadeStream = cdc.CascadeStream

// CDC builds a deterministic change-event stream for type T. The history is
// coherent: a row is inserted before it is updated, updates carry the true
// `before` image, and deleted rows are never touched again.
//
//	s, _ := synth.CDC[User](synth.CDCConfig{Table: "users", UpdateRate: 0.3, DeleteRate: 0.1})
//	s.WriteJSONL(os.Stdout, 1000)
func CDC[T any](cfg CDCConfig) (*CDCStream, error) {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, errNotStruct(zero)
	}
	cached, _ := reflectfe.Build(rt)
	s := &schema.Schema{Fields: append([]schema.Field(nil), cached.Fields...)}
	return cdc.New(s, cfg)
}

// WriteCDC writes n change events for type T to a JSONL file.
func WriteCDC[T any](path string, n int, cfg CDCConfig) error {
	s, err := CDC[T](cfg)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.WriteJSONL(f, n)
}

// CDCFromSpec builds a change-event stream from a YAML spec rather than a Go
// type, so the CLI can generate a history without compiled structs.
func (y *YAMLSpec) CDC(cfg CDCConfig) (*CDCStream, error) {
	s := &schema.Schema{Fields: append([]schema.Field(nil), y.spec.Schema.Fields...)}
	return cdc.New(s, cfg)
}

// Cascade builds a two-table change stream from this spec (the parent) and a
// child spec, where deleting a parent cascades to its children. The schemas are
// copied so the streams do not mutate the parsed specs.
func (y *YAMLSpec) Cascade(child *YAMLSpec, cfg CascadeConfig) (*CascadeStream, error) {
	p := &schema.Schema{Fields: append([]schema.Field(nil), y.spec.Schema.Fields...)}
	c := &schema.Schema{Fields: append([]schema.Field(nil), child.spec.Schema.Fields...)}
	return cdc.Cascade(p, c, cfg)
}
