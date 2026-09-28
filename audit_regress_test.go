package synth_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

type weightedRow struct {
	Status string
}

// Same seed, same output: Weighted must not depend on map iteration order.
func TestWeightedIsDeterministic(t *testing.T) {
	choices := map[string]float64{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1, "f": 1}
	first := synth.Make[weightedRow](200, synth.WithSeed(7), synth.Weighted("Status", choices))
	for i := 0; i < 20; i++ {
		again := synth.Make[weightedRow](200, synth.WithSeed(7), synth.Weighted("Status", choices))
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("run %d differs from the first with the same seed", i)
		}
	}
}

const orderSpec = `
openapi: 3.0.0
paths:
  /things:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                a: {type: string}
                b: {type: string}
                c: {type: string}
                d: {type: string}
                e: {type: string}
                f: {type: string}
                g: {type: string}
                h: {type: string}
`

func TestOpenAPIPayloadIsDeterministic(t *testing.T) {
	var first []map[string]any
	for i := 0; i < 20; i++ {
		api, err := synth.OpenAPIBytes([]byte(orderSpec))
		if err != nil {
			t.Fatal(err)
		}
		got, err := api.Payloads("POST", "/things", 5, synth.WithSeed(3))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = got
			continue
		}
		if !reflect.DeepEqual(first, got) {
			t.Fatalf("run %d differs from the first with the same seed", i)
		}
	}
}

const boundsSpec = `
openapi: 3.0.0
paths:
  /n:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                big: {type: integer, minimum: 1000000, maximum: 2000000}
`

// %g writes 1e6 as "1e+06", which a %d scan reads back as 1.
func TestOpenAPILargeIntegerBounds(t *testing.T) {
	api, err := synth.OpenAPIBytes([]byte(boundsSpec))
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Payloads("POST", "/n", 50, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range got {
		var v float64
		switch n := p["big"].(type) {
		case int:
			v = float64(n)
		case int64:
			v = float64(n)
		case float64:
			v = n
		default:
			t.Fatalf("big = %T %v", p["big"], p["big"])
		}
		if v < 1_000_000 || v > 2_000_000 {
			t.Fatalf("big = %v, want within [1e6, 2e6]", v)
		}
	}
}

type withUnexported struct {
	id   int
	Name string
	Age  int
}

func TestWriteCSVSkipsUnexportedFields(t *testing.T) {
	rows := []withUnexported{{id: 1, Name: "Ann", Age: 30}}
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := synth.WriteCSV(path, rows); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || lines[1] != "Ann,30" {
		t.Fatalf("csv = %q, want header then Ann,30", b)
	}
}

func TestWriteSQLSkipsUnexportedFields(t *testing.T) {
	rows := []withUnexported{{id: 1, Name: "Ann", Age: 30}}
	path := filepath.Join(t.TempDir(), "out.sql")
	if err := synth.WriteSQL(path, "people", rows); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "'Ann'") || !strings.Contains(string(b), "30") {
		t.Fatalf("sql = %s", b)
	}
}

type node struct {
	Name     string
	Children []node
	Parent   *node
}

func TestSelfReferentialTypeDoesNotOverflow(t *testing.T) {
	got, err := synth.TryMake[node](3, synth.WithSeed(1))
	if err != nil {
		return // refusing the type is fine; crashing the process is not
	}
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
}

type inner struct {
	Email string `synth:"email,unique"`
}

type outer struct {
	Name  string
	Inner inner
}

// MakeParallel refuses tracked unique fields; one inside a nested struct must
// be refused the same way rather than written from several goroutines at once.
func TestMakeParallelWithNestedUnique(t *testing.T) {
	got, err := synth.MakeParallel[outer](2000, 8, synth.WithSeed(1))
	if err != nil {
		if !strings.Contains(err.Error(), "unique") {
			t.Fatalf("err = %v", err)
		}
		return
	}
	seen := map[string]bool{}
	for _, o := range got {
		if seen[o.Inner.Email] {
			t.Fatalf("duplicate nested unique email %q", o.Inner.Email)
		}
		seen[o.Inner.Email] = true
	}
}

const floatSpec = `
openapi: 3.0.0
paths:
  /f:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                ratio: {type: number, minimum: 0.01, maximum: 0.99}
`

func TestOpenAPIFractionalFloatBounds(t *testing.T) {
	api, err := synth.OpenAPIBytes([]byte(floatSpec))
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Payloads("POST", "/f", 50, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	distinct := map[float64]bool{}
	for _, p := range got {
		v, _ := p["ratio"].(float64)
		if v < 0.01 || v > 0.99 {
			t.Fatalf("ratio = %v, want within [0.01, 0.99]", v)
		}
		distinct[v] = true
	}
	if len(distinct) < 10 {
		t.Fatalf("only %d distinct ratios in 50: bounds collapsed", len(distinct))
	}
}
