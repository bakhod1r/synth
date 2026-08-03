package synth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bakhod1r/synth"
)

type CoverRow struct {
	ID        uuid.UUID `synth:"pk"`
	Name      string
	Email     string `synth:"email,from=Name"`
	Amount    float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CoverChild struct {
	ID     uuid.UUID `synth:"pk"`
	RowID  uuid.UUID
	Detail string
}

func TestGeneratorConvenience(t *testing.T) {
	g := synth.New()
	// Exercise every single-value accessor.
	_ = g.Name()
	_ = g.FirstName()
	_ = g.Email()
	_ = g.Phone()
	_ = g.City()
	_ = g.Region()
	_ = g.Country()
	_ = g.Postcode()
	_ = g.Company()
	_ = g.Username()
	_ = g.Card()
	_ = g.IBAN()
	_ = g.IPv4()
	_ = g.URL()
	_ = g.Currency()
	if g.Amount(1, 100) < 0 {
		t.Fatal("amount negative")
	}
}

func TestPresetShorthands(t *testing.T) {
	for _, fn := range []func(int, ...synth.Option) ([]map[string]any, error){
		synth.Users, synth.Payments, synth.Transactions, synth.Orders,
	} {
		rows, err := fn(3, synth.WithSeed(1))
		if err != nil || len(rows) != 3 {
			t.Fatalf("preset shorthand: %v rows=%d", err, len(rows))
		}
	}
	if _, err := synth.Spec(synth.PresetUser); err != nil {
		t.Fatal(err)
	}
	if _, err := synth.Spec(synth.Preset("nope")); err == nil {
		t.Fatal("unknown preset spec should error")
	}
}

func TestMakeFillOffsetAndNonStruct(t *testing.T) {
	rows := synth.Make[CoverRow](5, synth.WithSeed(2), synth.Offset(10))
	if len(rows) != 5 {
		t.Fatalf("Make len = %d", len(rows))
	}
	var one CoverRow
	if err := synth.Fill(&one, synth.WithSeed(3)); err != nil {
		t.Fatal(err)
	}
	// A non-struct type is rejected.
	if _, err := synth.TryMake[int](1); err == nil {
		t.Fatal("non-struct TryMake should error")
	}
	var n int
	if err := synth.Fill(&n); err == nil {
		t.Fatal("non-struct Fill should error")
	}
}

func TestStreamToCSVAndEach(t *testing.T) {
	dir := t.TempDir()
	if err := synth.Stream[CoverRow](4, synth.WithSeed(1)).ToCSV(filepath.Join(dir, "s.csv")); err != nil {
		t.Fatal(err)
	}
	count := 0
	if err := synth.Stream[CoverRow](4, synth.WithSeed(1)).Each(func(CoverRow) error {
		count++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("Each visited %d", count)
	}
}

func TestWriteJSONL(t *testing.T) {
	recs := synth.Make[CoverRow](3, synth.WithSeed(1))
	p := filepath.Join(t.TempDir(), "o.jsonl")
	if err := synth.WriteJSONL(p, recs); err != nil {
		t.Fatal(err)
	}
}

func TestCDCAndCascadeAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := synth.WriteCDC[CoverRow](filepath.Join(dir, "cdc.jsonl"), 10,
		synth.CDCConfig{Table: "rows", Seed: 1, Snapshot: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := synth.CDC[CoverRow](synth.CDCConfig{Seed: 1}); err != nil {
		t.Fatal(err)
	}
	tl, err := synth.Snapshot[CoverRow](synth.SnapshotConfig{Rows: 20, Seed: 1,
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	_ = tl.At(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	if _, err := synth.Snapshot[int](synth.SnapshotConfig{}); err == nil {
		t.Fatal("non-struct Snapshot should error")
	}

	when, err := synth.ParseInstant("2026-01-01")
	if err != nil || when.Year() != 2026 {
		t.Fatalf("ParseInstant = %v %v", when, err)
	}
	if _, err := synth.ParseInstant("garbage"); err == nil {
		t.Fatal("bad instant should error")
	}
}

func TestYAMLSpecSnapshotAndCascade(t *testing.T) {
	parent := `
name: parent
count: 5
fields:
  id: { kind: uuid, pk: true }
  order_id: { kind: uuid }
`
	child := `
name: child
fields:
  id: { kind: uuid, pk: true }
  order_id: { kind: uuid }
`
	py, err := synth.YAMLBytes([]byte(parent))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := py.Snapshot(synth.SnapshotConfig{}); err != nil {
		t.Fatal(err)
	}
	if _, err := py.CDC(synth.CDCConfig{}); err != nil {
		t.Fatal(err)
	}
	cy, _ := synth.YAMLBytes([]byte(child))
	if _, err := py.Cascade(cy, synth.CascadeConfig{ChildFK: "order_id"}); err != nil {
		t.Fatal(err)
	}
}

func TestLoaders(t *testing.T) {
	dir := t.TempDir()
	ddl := filepath.Join(dir, "s.sql")
	os.WriteFile(ddl, []byte("CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(20));"), 0o644)
	if _, err := synth.LoadDDL(ddl); err != nil {
		t.Fatal(err)
	}
	yamlp := filepath.Join(dir, "s.yaml")
	os.WriteFile(yamlp, []byte("name: t\nfields:\n  a: { kind: int }\n"), 0o644)
	if _, err := synth.LoadYAML(yamlp); err != nil {
		t.Fatal(err)
	}
	jsonp := filepath.Join(dir, "s.json")
	os.WriteFile(jsonp, []byte(`{"title":"t","properties":{"a":{"type":"integer"}}}`), 0o644)
	if _, err := synth.LoadSchema(jsonp); err != nil {
		t.Fatal(err)
	}
	protop := filepath.Join(dir, "s.proto")
	os.WriteFile(protop, []byte("message M {\n  int64 id = 1;\n  string name = 2;\n}\n"), 0o644)
	if _, err := synth.LoadProto(protop); err != nil {
		t.Fatal(err)
	}
}

func TestProfileAndOpenAPI(t *testing.T) {
	dir := t.TempDir()
	csvp := filepath.Join(dir, "d.csv")
	os.WriteFile(csvp, []byte("a,b\n1,x\n2,y\n3,z\n"), 0o644)
	p, err := synth.Profile(csvp)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stats() == nil {
		t.Fatal("nil stats")
	}
	_ = p.Constraints()

	spec := `
openapi: 3.0.0
info: { title: t, version: "1" }
paths:
  /u:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: { type: string, format: uuid }
`
	apip := filepath.Join(dir, "api.yaml")
	os.WriteFile(apip, []byte(spec), 0o644)
	if _, err := synth.OpenAPI(apip); err != nil {
		t.Fatal(err)
	}
}
