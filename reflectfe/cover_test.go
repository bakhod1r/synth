package reflectfe

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bakhod1r/synth/schema"
)

type inner struct {
	City string
	Zip  string
}

// Named scalar types: goTypeName yields the short name, so infer.Kind returns
// Unknown and scalarKind is exercised per reflect.Kind.
type myStr string
type myBool bool
type myInt int32
type myFloat float64

type coverAll struct {
	unexported int //nolint
	Skip       string `synth:"-"`
	ID         string `synth:"pk"`
	Seq        int64  `synth:"pk"`
	Uniq       string `synth:"email,unique"`
	Explicit   int    `synth:"int,min=1,max=9,flag"`
	Linked     string `synth:"email,from=Name,match=City"`
	Choice     string `synth:"enum,choices=a|b|c,weights=1|2|3"`
	Trailing   int    `synth:"int,"`   // empty trailing part
	BarePK     int    `synth:"int,pk"` // bare pk flag in parts[1:]
	Name       string
	UID        uuid.UUID // goTypeName uuid.UUID branch
	Home       inner     // nested struct -> object
	PtrHome    *inner    // pointer nested
	People     []inner   // slice of struct
	Ptrs       []*inner
	Strs       []myStr   // scalarKind String
	Bools      []myBool  // scalarKind Bool
	Ints       []myInt   // scalarKind Int
	Floats     []myFloat // scalarKind Float
	Created    time.Time // scalar struct
	Weird      []func()  // unknown element -> array dropped
	Blobs      []complex128
}

func TestBuildCoversEverything(t *testing.T) {
	s, warns := Build(reflect.TypeOf(coverAll{}))

	if s.FieldByName("unexported") != nil {
		t.Fatal("unexported field leaked")
	}
	// synth:"-" means "ignore the tag and infer", not "skip the field".
	if s.FieldByName("Skip") == nil {
		t.Fatal(`synth:"-" field should still be inferred`)
	}
	if f := s.FieldByName("BarePK"); !f.PK {
		t.Fatalf("bare pk flag not applied: %+v", f)
	}
	if s.FieldByName("Trailing") == nil {
		t.Fatal("Trailing field missing")
	}
	// uuid.UUID is [16]byte, so as a bare struct field it is treated as an
	// array by isStructural; GoType still records "uuid.UUID".
	if f := s.FieldByName("UID"); f.GoType != "uuid.UUID" {
		t.Fatalf("uuid GoType = %+v", f)
	}
	for _, n := range []string{"Strs", "Bools", "Ints", "Floats"} {
		if f := s.FieldByName(n); f.Kind != schema.KindArray || f.Elem.Kind == schema.KindUnknown {
			t.Fatalf("named-scalar slice %s = %+v", n, f)
		}
	}
	if f := s.FieldByName("ID"); f.Kind != schema.KindUUID || !f.PK || !f.Unique {
		t.Fatalf("string pk = %+v", f)
	}
	if f := s.FieldByName("Seq"); f.Kind != schema.KindInt || f.Params["max"] != "2000000000" {
		t.Fatalf("int pk = %+v", f)
	}
	if f := s.FieldByName("Uniq"); !f.Unique || f.Kind != schema.KindEmail {
		t.Fatalf("unique flag = %+v", f)
	}
	if f := s.FieldByName("Explicit"); f.Params["min"] != "1" || f.Params["flag"] != "true" {
		t.Fatalf("params/flag = %+v", f)
	}
	if f := s.FieldByName("Linked"); f.From != "Name" || f.Match != "City" {
		t.Fatalf("from/match = %+v", f)
	}
	if f := s.FieldByName("Choice"); len(f.Choices) != 3 || len(f.Weights) != 3 {
		t.Fatalf("choices/weights = %+v", f)
	}
	if f := s.FieldByName("Home"); f.Kind != schema.KindObject || f.Nested == nil {
		t.Fatalf("nested struct = %+v", f)
	}
	if f := s.FieldByName("PtrHome"); f.Kind != schema.KindObject {
		t.Fatalf("ptr nested = %+v", f)
	}
	if f := s.FieldByName("People"); f.Kind != schema.KindArray || f.Elem.Nested == nil {
		t.Fatalf("struct slice = %+v", f)
	}
	if f := s.FieldByName("Ptrs"); f.Kind != schema.KindArray || f.Elem.Nested == nil {
		t.Fatalf("ptr struct slice = %+v", f)
	}
	if f := s.FieldByName("Created"); f.Kind == schema.KindObject {
		t.Fatalf("time.Time must be scalar, got %+v", f)
	}
	// Weird ([]func()) and Blobs ([]complex128): unknown element -> not an array.
	if f := s.FieldByName("Weird"); f.Kind == schema.KindArray {
		t.Fatalf("func slice should not become array: %+v", f)
	}
	if len(warns) == 0 {
		t.Fatal("expected warnings for uninferable fields")
	}
}

func TestBuildCacheHit(t *testing.T) {
	type cachedType struct{ Name string }
	rt := reflect.TypeOf(cachedType{})
	a, _ := Build(rt)
	b, _ := Build(rt) // second call hits cache branch
	if a != b {
		t.Fatal("cache should return same schema pointer")
	}
}

func TestEnrichStructuralScalarStruct(t *testing.T) {
	// Called directly with time.Time: the isScalarStruct guard returns early,
	// leaving the field unchanged (not an object).
	f := &schema.Field{Name: "T"}
	enrichStructural(f, reflect.TypeOf(time.Time{}))
	if f.Kind == schema.KindObject {
		t.Fatalf("time.Time enriched as object: %+v", f)
	}
}

func TestGoTypeNameNamed(t *testing.T) {
	type myStr string
	// A named type with a PkgPath returns its short name.
	if got := goTypeName(reflect.TypeOf(myStr(""))); got != "myStr" {
		t.Fatalf("named type = %q", got)
	}
}
