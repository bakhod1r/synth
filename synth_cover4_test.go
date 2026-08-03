package synth_test

import (
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth"
)

func TestMakePanicsOnNonStruct(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Make on non-struct should panic")
		}
	}()
	_ = synth.Make[int](1)
}

type WarnRec struct {
	Weird chan int // uninferable
	Name  string
}

func TestWarnings(t *testing.T) {
	if len(synth.Warnings[WarnRec]()) == 0 {
		t.Fatal("expected warnings for uninferable field")
	}
	if synth.Warnings[int]() != nil {
		t.Fatal("non-struct warnings should be nil")
	}
}

func TestMakeParallel(t *testing.T) {
	rows, err := synth.MakeParallel[CoverRow](20, 4, synth.WithSeed(1))
	if err != nil || len(rows) != 20 {
		t.Fatalf("MakeParallel: %v len=%d", err, len(rows))
	}
	if _, err := synth.MakeParallel[int](5, 2); err == nil {
		t.Fatal("non-struct MakeParallel should error")
	}
	// A unique field is rejected in the parallel path.
	type Uniq struct {
		ID string `synth:"pk"`
		N  int    `synth:"int,unique"`
	}
	if _, err := synth.MakeParallel[Uniq](5, 2); err == nil {
		t.Fatal("unique field should be rejected in MakeParallel")
	}
}

func TestStreamEngineErrors(t *testing.T) {
	dir := t.TempDir()
	if err := synth.Stream[int](2).ToCSV(filepath.Join(dir, "a")); err == nil {
		t.Fatal("non-struct ToCSV should error")
	}
	if err := synth.Stream[int](2).ToJSONL(filepath.Join(dir, "b")); err == nil {
		t.Fatal("non-struct ToJSONL should error")
	}
	if err := synth.Stream[int](2).Each(func(int) error { return nil }); err == nil {
		t.Fatal("non-struct Each should error")
	}
}

func TestCDCNonStruct(t *testing.T) {
	if _, err := synth.CDC[int](synth.CDCConfig{}); err == nil {
		t.Fatal("non-struct CDC should error")
	}
	if err := synth.WriteCDC[int]("x", 1, synth.CDCConfig{}); err == nil {
		t.Fatal("non-struct WriteCDC should error")
	}
}

func TestPresetGenerateUnknown(t *testing.T) {
	if _, err := synth.Generate(synth.Preset("nope"), 1); err == nil {
		t.Fatal("unknown preset should error")
	}
}

func TestProfileBytesAndProfiledMethods(t *testing.T) {
	csv := []byte("a,b\n1,x\n2,y\n3,z\n")
	p, err := synth.ProfileBytes(csv, "csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Columns()) != 2 || p.SampleRows() != 3 {
		t.Fatalf("columns/rows: %v %d", p.Columns(), p.SampleRows())
	}
	rows, err := p.Generate(3, synth.WithSeed(1))
	if err != nil || len(rows) != 3 {
		t.Fatalf("profiled generate: %v", err)
	}
	if _, err := p.YAML("t", 5); err != nil {
		t.Fatal(err)
	}
	_ = p.Constraints()

	// JSONL format and an unknown format.
	if _, err := synth.ProfileBytes([]byte(`{"a":1}`+"\n"), "jsonl"); err != nil {
		t.Fatal(err)
	}
	if _, err := synth.ProfileBytes(csv, "xml"); err == nil {
		t.Fatal("unknown format should error")
	}
}

func TestBytesParsersErrors(t *testing.T) {
	if _, err := synth.YAMLBytes([]byte(":\n\t- bad")); err == nil {
		t.Fatal("bad yaml")
	}
	if _, err := synth.ProtoBytes([]byte("no message here")); err == nil {
		t.Fatal("bad proto")
	}
	if _, err := synth.JSONSchemaBytes([]byte("{bad")); err == nil {
		t.Fatal("bad json schema")
	}
	if _, err := synth.AvroBytes([]byte("{bad")); err == nil {
		t.Fatal("bad avro")
	}
	if _, err := synth.OpenAPIBytes([]byte("{bad")); err == nil {
		t.Fatal("bad openapi")
	}
}

func TestNewMaskerDefaultLocale(t *testing.T) {
	if synth.NewMasker("k", "") == nil {
		t.Fatal("NewMasker returned nil")
	}
}

