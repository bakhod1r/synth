package synth_test

import (
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth"
)

func TestRegisterCustomTypeUsesR(t *testing.T) {
	// A custom provider that exercises the R adapter's Intn/Float64/Digits/Pick.
	synth.Register("coverkind", func(r synth.R) any {
		_ = r.Intn(5)
		_ = r.IntRange(1, 3)
		_ = r.Float64()
		_ = r.Pick([]string{"a", "b"})
		return r.Digits(4)
	})
	type Rec struct {
		X string `synth:"coverkind"`
	}
	rows := synth.Make[Rec](3, synth.WithSeed(1))
	if len(rows) != 3 {
		t.Fatalf("custom kind rows = %d", len(rows))
	}
}

func TestYAMLSpecSchemaAccessor(t *testing.T) {
	y, err := synth.YAMLBytes([]byte("name: t\nfields:\n  a: { kind: int }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if y.Schema() == nil || len(y.Schema().Fields) != 1 {
		t.Fatal("Schema accessor wrong")
	}
}

func TestLoaderErrorPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.x")
	if _, err := synth.LoadDDL(missing); err == nil {
		t.Fatal("LoadDDL missing")
	}
	if _, err := synth.LoadYAML(missing); err == nil {
		t.Fatal("LoadYAML missing")
	}
	if _, err := synth.LoadSchema(missing); err == nil {
		t.Fatal("LoadSchema missing")
	}
	if _, err := synth.LoadProto(missing); err == nil {
		t.Fatal("LoadProto missing")
	}
	if _, err := synth.OpenAPI(missing); err == nil {
		t.Fatal("OpenAPI missing")
	}
	if _, err := synth.Profile(missing); err == nil {
		t.Fatal("Profile missing")
	}
}

func TestParseErrorPaths(t *testing.T) {
	if _, err := synth.DDLBytes([]byte("not ddl at all")); err != nil {
		_ = err // DDLBytes tolerates non-DDL; just exercise it
	}
	if _, err := synth.OpenAPIBytes([]byte("{not yaml")); err == nil {
		t.Fatal("bad openapi bytes should error")
	}
	if _, err := synth.ProfileBytes([]byte("{bad"), "jsonl"); err == nil {
		t.Fatal("bad profile bytes should error")
	}
}

func TestWriteErrorPaths(t *testing.T) {
	type Rec struct{ Name string }
	recs := synth.Make[Rec](2, synth.WithSeed(1))
	badDir := filepath.Join(t.TempDir(), "nodir", "f")
	if err := synth.WriteCSV(badDir, recs); err == nil {
		t.Fatal("WriteCSV bad path")
	}
	if err := synth.WriteJSONL(badDir, recs); err == nil {
		t.Fatal("WriteJSONL bad path")
	}
	if err := synth.WriteSQL(badDir, "t", recs); err == nil {
		t.Fatal("WriteSQL bad path")
	}
	if err := synth.Stream[Rec](2).ToCSV(badDir); err == nil {
		t.Fatal("ToCSV bad path")
	}
	if err := synth.WriteCDC[Rec](badDir, 3, synth.CDCConfig{Seed: 1}); err == nil {
		t.Fatal("WriteCDC bad path")
	}
}
