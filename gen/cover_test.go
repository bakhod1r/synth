package gen

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

func mkField(name string, k schema.Kind, params map[string]string) schema.Field {
	if params == nil {
		params = map[string]string{}
	}
	return schema.Field{Name: name, Kind: k, Params: params}
}

func TestCompileUnknownKind(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{mkField("x", schema.Kind("nope"), nil)}}
	if _, err := Compile(s, "en"); err == nil {
		t.Fatal("expected unknown-kind error")
	}
}

func TestCompileAxisNonTimeErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		mkField("amt", schema.KindInt, nil),
		mkField("ts", schema.KindTimeSeries, map[string]string{"axis": "amt"}),
	}}
	if _, err := Compile(s, "en"); err == nil {
		t.Fatal("expected axis-not-time error")
	}
}

func TestCompileNestedAndArrayObjects(t *testing.T) {
	nested := &schema.Schema{Fields: []schema.Field{mkField("City", schema.KindCity, nil)}}
	elem := &schema.Field{Name: "Item", Kind: schema.KindObject, Nested: &schema.Schema{
		Fields: []schema.Field{mkField("Name", schema.KindName, nil)},
	}}
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Home", Kind: schema.KindObject, Nested: nested, Params: map[string]string{}},
		{Name: "Items", Kind: schema.KindArray, Elem: elem, Params: map[string]string{}, ArrMin: 1, ArrMax: 2},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Record(rng.New(1), 0)
	if _, ok := rec["Home"].(map[string]any); !ok {
		t.Fatalf("Home not a sub-record: %T", rec["Home"])
	}
	if _, ok := rec["Items"].([]any); !ok {
		t.Fatalf("Items not a slice: %T", rec["Items"])
	}
}

func TestCompileNestedErrorPropagates(t *testing.T) {
	nested := &schema.Schema{Fields: []schema.Field{mkField("bad", schema.Kind("nope"), nil)}}
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Home", Kind: schema.KindObject, Nested: nested, Params: map[string]string{}},
	}}
	if _, err := Compile(s, "en"); err == nil {
		t.Fatal("nested compile error should propagate")
	}
}

func TestUniqueTrackingAndHasUnique(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "N", Kind: schema.KindInt, Unique: true, Params: map[string]string{"min": "0", "max": "1000000"}},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !e.HasUnique() {
		t.Fatal("HasUnique should be true")
	}
	if e.Schema() != s {
		t.Fatal("Schema() should expose the schema")
	}
	seen := map[any]bool{}
	base := rng.New(9)
	for i := 0; i < 200; i++ {
		v := e.Record(base, i)["N"]
		if seen[v] {
			t.Fatalf("duplicate unique value %v", v)
		}
		seen[v] = true
	}
}

func TestChaosProducesEdgeCases(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{mkField("Name", schema.KindName, nil)}}
	e, _ := Compile(s, "en")
	e.Chaos = 1 // always chaos
	rec := e.Record(rng.New(3), 0)
	// Name is a string; chaos replaces with a nasty string (possibly empty).
	if _, ok := rec["Name"].(string); !ok {
		t.Fatalf("Name should stay string, got %T", rec["Name"])
	}
}

func TestFieldBlankAndFromRefAndUnknown(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Maybe", Kind: schema.KindName, Params: map[string]string{"blank": "100%"}},
		{Name: "FK", Kind: schema.KindInt, FromRef: []any{1, 2, 3}, Params: map[string]string{}},
		{Name: "Huh", Kind: schema.KindUnknown, Params: map[string]string{}},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Record(rng.New(2), 0)
	if rec["Maybe"] != nil {
		t.Fatalf("blank=100%% should be nil, got %v", rec["Maybe"])
	}
	if rec["FK"] == nil {
		t.Fatal("FromRef should draw a parent value")
	}
	if rec["Huh"] != nil {
		t.Fatal("unknown kind should be nil")
	}
}

func TestMaskValueModes(t *testing.T) {
	part := mkField("c", schema.KindLorem, map[string]string{"mask": "partial"})
	if got := maskValue(&part, "abcdef"); got != "**cdef" {
		t.Fatalf("partial = %q", got)
	}
	card := mkField("c", schema.KindCard, map[string]string{"mask": "partial"})
	if got := maskValue(&card, "4539578763621486"); !strings.HasPrefix(got.(string), "4539") ||
		!strings.HasSuffix(got.(string), "1486") || !strings.Contains(got.(string), "*") {
		t.Fatalf("card partial = %q", got)
	}
	hash := mkField("c", schema.KindLorem, map[string]string{"mask": "hash", "digest": "20"})
	if h := maskValue(&hash, "x").(string); len(h) != 20 {
		t.Fatalf("hash digest len = %d", len(h))
	}
	red := mkField("c", schema.KindLorem, map[string]string{"mask": "redact"})
	if maskValue(&red, "x") != "[REDACTED]" {
		t.Fatal("redact wrong")
	}
	tok := mkField("c", schema.KindLorem, map[string]string{"mask": "token"})
	if s := maskValue(&tok, "x").(string); !strings.HasPrefix(s, "tok_") || len(s) != len("tok_")+24 {
		t.Fatalf("token = %q", s)
	}
	none := mkField("c", schema.KindLorem, map[string]string{})
	if maskValue(&none, "keep") != "keep" {
		t.Fatal("no mask should pass through")
	}
	bogus := mkField("c", schema.KindLorem, map[string]string{"mask": "bogus"})
	if maskValue(&bogus, "keep") != "keep" {
		t.Fatal("unknown mask should return value")
	}
	if maskValue(&red, nil) != nil {
		t.Fatal("nil value should stay nil")
	}
}

func TestDigestSecretVsPlain(t *testing.T) {
	plain := mkField("c", schema.KindLorem, map[string]string{})
	secret := mkField("c", schema.KindLorem, map[string]string{"secret": "k"})
	if digest(&plain, "v") == digest(&secret, "v") {
		t.Fatal("secret digest should differ from plain")
	}
}

func TestDigestAlgorithm(t *testing.T) {
	def := mkField("c", schema.KindLorem, map[string]string{})
	s256 := mkField("c", schema.KindLorem, map[string]string{"algo": "sha256"})
	s512 := mkField("c", schema.KindLorem, map[string]string{"algo": "sha512"})

	// Default is SHA-256: 64 hex characters, and unchanged by naming it.
	if got := digest(&def, "v"); len(got) != 64 {
		t.Fatalf("default digest len = %d, want 64", len(got))
	}
	if digest(&def, "v") != digest(&s256, "v") {
		t.Fatal("algo=sha256 must equal the default, so old specs stay byte-identical")
	}
	// SHA-512 is 128 hex characters and a different value.
	if got := digest(&s512, "v"); len(got) != 128 {
		t.Fatalf("sha512 digest len = %d, want 128", len(got))
	}
	if digest(&s512, "v")[:64] == digest(&def, "v") {
		t.Fatal("sha512 should not share a prefix with sha256 by construction")
	}
	// The algorithm carries through the HMAC (secret=) path too.
	sec := mkField("c", schema.KindLorem, map[string]string{"algo": "sha512", "secret": "k"})
	if got := digest(&sec, "v"); len(got) != 128 {
		t.Fatalf("hmac-sha512 digest len = %d, want 128", len(got))
	}
	// Unknown algo falls back to the default rather than failing.
	bad := mkField("c", schema.KindLorem, map[string]string{"algo": "crc32"})
	if digest(&bad, "v") != digest(&def, "v") {
		t.Fatal("unknown algo should fall back to sha256")
	}
}

func TestTruncate(t *testing.T) {
	long := strings.Repeat("a", 64)
	f8 := mkField("c", schema.KindLorem, map[string]string{"digest": "8"}) // floors to 16
	if got := truncate(&f8, long); len(got) != 16 {
		t.Fatalf("floor to 16, got %d", len(got))
	}
	fbad := mkField("c", schema.KindLorem, map[string]string{"digest": "x"})
	if truncate(&fbad, long) != long {
		t.Fatal("invalid digest should not truncate")
	}
	fbig := mkField("c", schema.KindLorem, map[string]string{"digest": "999"})
	if truncate(&fbig, long) != long {
		t.Fatal("digest >= len should not truncate")
	}
}

func TestPartialMaskShort(t *testing.T) {
	if partialMask("ab") != "**" {
		t.Fatal("<=4 chars fully starred")
	}
	if partialMask("12-345678") == "" {
		t.Fatal("empty result")
	}
}

func TestCardMaskShortFallback(t *testing.T) {
	// Fewer than 12 digits falls back to partialMask.
	if got := cardMask("1234"); got != "****" {
		t.Fatalf("short card = %q", got)
	}
	long := cardMask("4539-5787-6362-1486")
	if !strings.HasPrefix(long, "4539") || !strings.HasSuffix(long, "1486") {
		t.Fatalf("long card = %q", long)
	}
}

func TestArrayDefaults(t *testing.T) {
	// ArrMin/ArrMax zero -> default 1..3; scalar element.
	f := schema.Field{Name: "xs", Kind: schema.KindArray, Params: map[string]string{},
		Elem: &schema.Field{Name: "xs", Kind: schema.KindInt, Params: map[string]string{}}}
	s := &schema.Schema{Fields: []schema.Field{f}}
	e, _ := Compile(s, "en")
	got := e.array(rng.New(1), &e.schema.Fields[0], &e.loc.Places[0], "male", map[string]any{})
	if len(got) < 1 || len(got) > 3 {
		t.Fatalf("array len = %d", len(got))
	}
}

func TestArrayObjectWithoutSubEngine(t *testing.T) {
	// Elem is an object whose Nested has no compiled sub-engine -> nil elements.
	f := schema.Field{Name: "xs", Kind: schema.KindArray, ArrMin: 2, ArrMax: 2, Params: map[string]string{},
		Elem: &schema.Field{Name: "xs", Kind: schema.KindObject, Nested: &schema.Schema{}, Params: map[string]string{}}}
	e := &Engine{schema: &schema.Schema{}, loc: locale.Get("en")}
	out := e.array(rng.New(1), &f, &e.loc.Places[0], "male", map[string]any{})
	for _, v := range out {
		if v != nil {
			t.Fatalf("expected nil element, got %v", v)
		}
	}
}

func TestBlankShare(t *testing.T) {
	f := func(v string) *schema.Field { return &schema.Field{Params: map[string]string{"blank": v}} }
	cases := map[string]float64{
		"0.15": 0.15,
		"15":   0.15,
		"15%":  0.15,
		"200":  1, // clamps to 1
		"0":    0,
		"bad":  0,
	}
	for in, want := range cases {
		if got := blankShare(f(in)); got != want {
			t.Fatalf("blankShare(%q) = %v want %v", in, got, want)
		}
	}
	if got := blankShare(&schema.Field{Params: map[string]string{}}); got != 0 {
		t.Fatalf("missing blank = %v", got)
	}
}

func TestBasePlaceFor(t *testing.T) {
	base := locale.Get("en_US")
	p := basePlaceFor(base, &locale.Place{City: "X", Region: "Y", Postcode: "Z"})
	if p == nil {
		t.Fatal("nil place")
	}
	if got := basePlaceFor(nil, p); got != p {
		t.Fatal("nil base should return the input place")
	}
	// place nil -> empty key path.
	if basePlaceFor(base, nil) == nil {
		t.Fatal("nil place should still map")
	}
}

func TestCompileTopoCycleErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "A", Kind: schema.KindName, From: "B", Params: map[string]string{}},
		{Name: "B", Kind: schema.KindName, From: "A", Params: map[string]string{}},
	}}
	if _, err := Compile(s, "en"); err == nil {
		t.Fatal("dependency cycle should error")
	}
}

func TestUniqueResampleOnSmallSpace(t *testing.T) {
	// Tiny value space forces the resample loop (185-187) to run.
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "N", Kind: schema.KindInt, Unique: true, Params: map[string]string{"min": "0", "max": "9"}},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	base := rng.New(1)
	seen := map[any]bool{}
	for i := 0; i < 10; i++ {
		v := e.Record(base, i)["N"]
		seen[v] = true
	}
	if len(seen) < 5 {
		t.Fatalf("expected several distinct values, got %d", len(seen))
	}
}

func TestFieldObjectWithoutSubEngine(t *testing.T) {
	f := schema.Field{Name: "Home", Kind: schema.KindObject, Nested: &schema.Schema{}, Params: map[string]string{}}
	e := &Engine{schema: &schema.Schema{}, loc: locale.Get("en")}
	if got := e.field(rng.New(1), &f, &e.loc.Places[0], "male", map[string]any{}); got != nil {
		t.Fatalf("object without sub-engine should be nil, got %v", got)
	}
}

func TestSiblingFromWiring(t *testing.T) {
	// email from=Name makes the provider read Sibling("__from__").
	s := &schema.Schema{Fields: []schema.Field{
		mkField("Name", schema.KindName, nil),
		{Name: "Email", Kind: schema.KindEmail, From: "Name", Params: map[string]string{}},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	rec := e.Record(rng.New(1), 0)
	if _, ok := rec["Email"].(string); !ok {
		t.Fatalf("email not generated: %v", rec["Email"])
	}
}

func TestSiblingFromEmptyReturnsNil(t *testing.T) {
	// email with no from= makes Sibling("__from__") take the nil branch.
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Email", Kind: schema.KindEmail, Params: map[string]string{}},
	}}
	e, err := Compile(s, "en")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.Record(rng.New(1), 0)["Email"].(string); !ok {
		t.Fatal("email without from should still generate")
	}
}

func TestArrayMaxBelowMin(t *testing.T) {
	f := schema.Field{Name: "xs", Kind: schema.KindArray, ArrMin: 3, ArrMax: 1, Params: map[string]string{},
		Elem: &schema.Field{Name: "xs", Kind: schema.KindInt, Params: map[string]string{}}}
	e := &Engine{schema: &schema.Schema{}, loc: locale.Get("en")}
	out := e.array(rng.New(1), &f, &e.loc.Places[0], "male", map[string]any{})
	if len(out) != 3 {
		t.Fatalf("max<min should clamp to min=3, got %d", len(out))
	}
}

func TestChaosValueTypes(t *testing.T) {
	r := rng.New(1)
	if _, ok := chaosValue(r, "s").(string); !ok {
		t.Fatal("string chaos")
	}
	if _, ok := chaosValue(r, 5).(int); !ok {
		t.Fatal("int chaos")
	}
	if _, ok := chaosValue(r, 1.5).(float64); !ok {
		t.Fatal("float chaos")
	}
	type x struct{}
	if _, ok := chaosValue(r, x{}).(x); !ok {
		t.Fatal("other type should pass through")
	}
}
