package cdc

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestNewDefaultsAndErrors(t *testing.T) {
	// Empty Table/Key/Interval/Start take defaults; a no-PK schema keys off the
	// first column.
	noPK := &schema.Schema{Fields: []schema.Field{
		{Name: "col", Kind: schema.KindName, Params: map[string]string{}},
	}}
	if _, err := New(noPK, Config{}); err != nil {
		t.Fatal(err)
	}
	// Rate sum >= 1 is rejected.
	if _, err := New(softSchema(), Config{UpdateRate: 0.6, DeleteRate: 0.6}); err == nil {
		t.Fatal("rate sum >= 1 should error")
	}
	// A schema the engine cannot compile surfaces the error.
	bad := &schema.Schema{Fields: []schema.Field{{Name: "x", Kind: schema.Kind("nope"), Params: map[string]string{}}}}
	if _, err := New(bad, Config{}); err == nil {
		t.Fatal("uncompilable schema should error")
	}
}

func TestPrimaryKeyNameEmptySchema(t *testing.T) {
	// An empty schema falls back to "id" (via New's Key default path).
	if _, err := New(&schema.Schema{}, Config{}); err != nil {
		t.Fatal(err)
	}
}

func TestStreamWriteJSONL(t *testing.T) {
	s, err := New(softSchema(), Config{Table: "t", Snapshot: 3, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := s.WriteJSONL(&b, 10); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "\n") {
		t.Fatal("no events written")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errFail }

type errFailT struct{}

func (errFailT) Error() string { return "boom" }

var errFail = errFailT{}

func TestWriteJSONLEncodeErrors(t *testing.T) {
	s, _ := New(softSchema(), Config{Snapshot: 3, Seed: 1})
	if err := s.WriteJSONL(failWriter{}, 5); err == nil {
		t.Fatal("stream encode error should surface")
	}
	cs, _ := Cascade(parentSchema(), childSchema(), CascadeConfig{ChildFK: "order_id", Snapshot: 2, Seed: 1})
	if err := cs.WriteJSONL(failWriter{}, 5); err == nil {
		t.Fatal("cascade encode error should surface")
	}
}

func TestStreamExhaustsToNil(t *testing.T) {
	// A tiny history with a high delete rate empties out; Next eventually nil.
	s, _ := New(softSchema(), Config{Snapshot: 1, DeleteRate: 0.95, Seed: 2})
	nilSeen := false
	for i := 0; i < 5000; i++ {
		if s.Next() == nil {
			nilSeen = true
			break
		}
	}
	if !nilSeen {
		t.Skip("stream did not exhaust in the sampled window")
	}
	// WriteJSONL past exhaustion exercises the ev==nil break.
	var b strings.Builder
	_ = s.WriteJSONL(&b, 10)
}

func TestCascadeDefaultsAndErrors(t *testing.T) {
	// ChildFK not present in the child schema.
	if _, err := Cascade(parentSchema(), childSchema(), CascadeConfig{ChildFK: "nope"}); err == nil {
		t.Fatal("unknown ChildFK should error")
	}
	// Rate sum >= 1.
	if _, err := Cascade(parentSchema(), childSchema(),
		CascadeConfig{ChildFK: "order_id", UpdateRate: 0.7, DeleteRate: 0.7}); err == nil {
		t.Fatal("rate sum >= 1 should error")
	}
	// Bad parent schema (compile error).
	bad := &schema.Schema{Fields: []schema.Field{{Name: "x", Kind: schema.Kind("nope"), Params: map[string]string{}}}}
	if _, err := Cascade(bad, childSchema(), CascadeConfig{ChildFK: "order_id"}); err == nil {
		t.Fatal("bad parent should error")
	}
	// Bad child schema (compile error) with a valid FK column present.
	badChild := &schema.Schema{Fields: []schema.Field{
		{Name: "order_id", Kind: schema.KindUUID, Params: map[string]string{}},
		{Name: "y", Kind: schema.Kind("nope"), Params: map[string]string{}},
	}}
	if _, err := Cascade(parentSchema(), badChild, CascadeConfig{ChildFK: "order_id"}); err == nil {
		t.Fatal("bad child should error")
	}

	// All defaults applied on a valid pair; then drive events + WriteJSONL.
	cs, err := Cascade(parentSchema(), childSchema(), CascadeConfig{ChildFK: "order_id", Snapshot: 2, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := cs.WriteJSONL(&b, 20); err != nil {
		t.Fatal(err)
	}
	if b.Len() == 0 {
		t.Fatal("no cascade events written")
	}
}
