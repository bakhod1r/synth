package parquet_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
	synthparquet "github.com/bakhod1r/synth/sink/parquet"
	pq "github.com/parquet-go/parquet-go"
)

type User struct {
	ID    int64   `parquet:"id"`
	Name  string  `parquet:"name"`
	Email string  `parquet:"email"`
	Score float64 `parquet:"score"`
}

// Struct records must survive a round trip through a Parquet file.
func TestWriteStructsRoundTrip(t *testing.T) {
	users := synth.Make[User](500, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	path := filepath.Join(t.TempDir(), "users.parquet")
	if err := synthparquet.WriteStructs(path, users); err != nil {
		t.Fatal(err)
	}

	back, err := pq.ReadFile[User](path)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != len(users) {
		t.Fatalf("read %d rows, wrote %d", len(back), len(users))
	}
	for i := range users {
		if back[i] != users[i] {
			t.Fatalf("row %d changed: %+v != %+v", i, back[i], users[i])
		}
	}
	for _, u := range back {
		if !strings.Contains(u.Email, "@") {
			t.Fatalf("email lost: %q", u.Email)
		}
	}
}

// Map-shaped records (YAML/DDL/profiling output) must write with a correct
// inferred schema and read back with the right column types.
func TestWriteRowsFromSpec(t *testing.T) {
	const spec = `
name: users
count: 300
locale: uz_UZ
seed: 42
fields:
  id:      { kind: uuid, pk: true }
  name:    { kind: name }
  age:     { kind: int, min: 18, max: 65 }
  balance: { kind: amount, min: 0, max: 5000 }
  active:  { kind: bool }
`
	sp, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := sp.Generate()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "spec.parquet")
	if err := synthparquet.WriteRows(path, sp.Columns(), rows); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Fatalf("parquet file missing or empty: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, _ := f.Stat()
	pf, err := pq.OpenFile(f, fi.Size())
	if err != nil {
		t.Fatal(err)
	}
	if got := pf.NumRows(); got != int64(len(rows)) {
		t.Fatalf("parquet has %d rows, want %d", got, len(rows))
	}

	// Column types must be inferred, not all strings.
	types := map[string]string{}
	for _, fld := range pf.Schema().Fields() {
		types[fld.Name()] = fld.Type().String()
	}
	if !strings.Contains(types["age"], "INT(64") {
		t.Fatalf("age should be a 64-bit int, got %q", types["age"])
	}
	if !strings.Contains(types["balance"], "DOUBLE") {
		t.Fatalf("balance should be DOUBLE, got %q", types["balance"])
	}
	if !strings.Contains(types["active"], "BOOLEAN") {
		t.Fatalf("active should be BOOLEAN, got %q", types["active"])
	}
	if !strings.Contains(strings.ToUpper(types["name"]), "STRING") &&
		!strings.Contains(strings.ToUpper(types["name"]), "BYTE_ARRAY") {
		t.Fatalf("name should be a string type, got %q", types["name"])
	}
}

func TestWriteRowsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.parquet")
	if err := synthparquet.WriteRows(path, []string{"a"}, nil); err == nil {
		t.Fatal("expected an error for zero rows")
	}
}
