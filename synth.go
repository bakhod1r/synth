// Package synth generates realistic, coherent, referentially-consistent
// records from plain Go structs. It is a pure data provider: it never touches
// the network, a database, or DDL. A struct goes in; records come out — in
// memory, to a file, or streamed.
package synth

import (
	"fmt"
	"reflect"
	"time"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/reflectfe"
	"github.com/bakhod1r/synth/schema"
)

func errNotStruct(v any) error {
	return fmt.Errorf("synth: requires a struct type, got %T", v)
}

// Option configures a generation call.
type Option func(*config)

type config struct {
	seed     uint64
	locale   string
	refs     []refSpec
	weighted map[string]weightedSpec
	chaos    float64
	unmask   bool
	offset   int
}

type weightedSpec struct {
	choices []string
	weights []float64
}

type refSpec struct {
	fkField string
	values  []any
	min     int
	max     int
}

// WithSeed sets the base seed for deterministic output.
func WithSeed(seed uint64) Option { return func(c *config) { c.seed = seed } }

// WithLocale selects a locale ("uz_UZ", "en_US", ...).
func WithLocale(name string) Option { return func(c *config) { c.locale = name } }

// Weighted turns a field into a weighted enum in code (an alternative to the
// `synth:"enum,choices=...,weights=..."` tag). Weights need not sum to 1.
//
//	synth.Weighted("Status", map[string]float64{"settled":0.94,"pending":0.05,"failed":0.01})
func Weighted(field string, choices map[string]float64) Option {
	ws := weightedSpec{}
	for k, v := range choices {
		ws.choices = append(ws.choices, k)
		ws.weights = append(ws.weights, v)
	}
	return func(c *config) {
		if c.weighted == nil {
			c.weighted = map[string]weightedSpec{}
		}
		c.weighted[field] = ws
	}
}

// WithChaos makes a fraction p (0..1) of string/numeric fields carry an
// edge-case value — empty strings, emoji, RTL text, SQL/HTML fragments,
// pathologically long input, boundary numerics. Use it to test the paths the
// happy path never reaches. Referential-key fields are never corrupted.
func WithChaos(p float64) Option { return func(c *config) { c.chaos = p } }

// Ref links a foreign-key field on the child to a parent slice, so every
// child row points at a real parent. Pass OneToMany to control cardinality.
func Ref[P any](parents []P, fkField string, opts ...RefOption) Option {
	pkValues := extractPK(parents)
	// No parent keys means nothing to point at; skip the ref entirely so the FK
	// field generates normally instead of drawing from an empty slice (IntN(0)).
	if len(pkValues) == 0 {
		return func(c *config) {}
	}
	rs := refSpec{fkField: fkField, values: pkValues}
	for _, o := range opts {
		o(&rs)
	}
	return func(c *config) { c.refs = append(c.refs, rs) }
}

// Offset starts row generation at record index n instead of 0.
//
// Each row is seeded from its index, so the output is a deterministic function
// of the index. Offsetting the index is what lets a second run extend a first
// one: Offset(1000) produces rows 1000..1000+n, which differ from the first
// run's rows 0..999 yet stay reproducible. This is the mechanism behind the
// CLI's --append.
func Offset(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.offset = n
		}
	}
}

// RefValues links a foreign-key field to values the caller already holds,
// rather than to a parent slice generated in the same process. This is the
// cross-run case: the parent was written to a file in an earlier run, its key
// column read back, and passed here so the child points at rows that already
// exist on disk.
//
//	users, _ := synth.Users(10000)                      // run 1, written out
//	keys := readColumn("users.csv", "id")               // read back later
//	orders, _ := spec.GenerateN(500000, synth.RefValues("user_id", keys))
//
// A nil or empty values slice is a no-op: with no parent keys there is nothing
// to point at, and the field generates as it otherwise would.
func RefValues(fkField string, values []any) Option {
	return func(c *config) {
		if len(values) == 0 {
			return
		}
		c.refs = append(c.refs, refSpec{fkField: fkField, values: values})
	}
}

// RefOption tunes a Ref.
type RefOption func(*refSpec)

// OneToMany makes each parent own between min and max children (best-effort;
// applied by drawing FK values proportionally). Currently distributes
// uniformly at random within the range semantics.
func OneToMany(min, max int) RefOption {
	return func(rs *refSpec) { rs.min, rs.max = min, max }
}

// Make generates n records of type T. It panics on configuration errors
// (unknown tag, dependency cycle) — use TryMake in production code.
func Make[T any](n int, opts ...Option) []T {
	out, err := TryMake[T](n, opts...)
	if err != nil {
		panic(err)
	}
	return out
}

// TryMake is Make that returns configuration errors instead of panicking.
func TryMake[T any](n int, opts ...Option) ([]T, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("synth: Make requires a struct type, got %T", zero)
	}
	cached, _ := reflectfe.Build(rt)
	// Work on a copy: refs mutate fields and the cache must stay pristine.
	s := &schema.Schema{Fields: append([]schema.Field(nil), cached.Fields...)}
	applyRefs(s, cfg.refs)
	applyWeighted(s, cfg.weighted)

	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	eng.Chaos = cfg.chaos
	base := rng.New(cfg.seed)
	out := make([]T, n)
	for i := 0; i < n; i++ {
		rec := eng.Record(base, i)
		scatter(&out[i], rec)
	}
	if err := eng.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Fill populates a single struct pointer in place.
func Fill[T any](p *T, opts ...Option) error {
	recs, err := TryMake[T](1, opts...)
	if err != nil {
		return err
	}
	*p = recs[0]
	return nil
}

// Warnings returns the fields Synth could not infer for type T (left as zero).
func Warnings[T any]() []schema.Warning {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil
	}
	_, w := reflectfe.Build(rt)
	return w
}

// applyRefs binds ref specs to their FK fields in the schema (per-call, so we
// clone the field's ref data without mutating the cached schema fields'
// providers). We copy the schema fields to avoid cross-call contamination.
func applyRefs(s *schema.Schema, refs []refSpec) {
	for _, rs := range refs {
		if f := s.FieldByName(rs.fkField); f != nil {
			f.FromRef = expandCardinality(rs.values, rs.min, rs.max)
			f.RefMin, f.RefMax = rs.min, rs.max
		}
	}
}

// applyRefsChecked binds ref specs like applyRefs, but reports a ref whose
// field the schema does not have instead of ignoring it. A misspelled FK column
// that silently leaves the field randomly generated is the kind of error that
// passes generation and fails only when someone tries to join the tables.
func applyRefsChecked(s *schema.Schema, refs []refSpec) error {
	for _, rs := range refs {
		f := s.FieldByName(rs.fkField)
		if f == nil {
			return fmt.Errorf("synth: ref field %q is not in the spec", rs.fkField)
		}
		f.FromRef = expandCardinality(rs.values, rs.min, rs.max)
		f.RefMin, f.RefMax = rs.min, rs.max
	}
	return nil
}

// expandCardinality implements OneToMany: each parent appears a deterministic
// count in [min,max] in the returned pool, so drawing foreign keys uniformly
// from the pool gives each parent roughly that many children. When min<=0 the
// values are returned unchanged (uniform, unbounded cardinality).
func expandCardinality(values []any, min, max int) []any {
	if min <= 0 || len(values) == 0 {
		return values
	}
	if max < min {
		max = min
	}
	span := max - min + 1
	out := make([]any, 0, len(values)*(min+max)/2)
	for i, v := range values {
		count := min + i%span // spread counts deterministically across the range
		for j := 0; j < count; j++ {
			out = append(out, v)
		}
	}
	return out
}

// applyWeighted turns named fields into weighted enums per the config.
func applyWeighted(s *schema.Schema, w map[string]weightedSpec) {
	for name, ws := range w {
		if f := s.FieldByName(name); f != nil {
			f.Kind = schema.KindEnum
			f.Choices = ws.choices
			f.Weights = ws.weights
		}
	}
}

// scatter writes generated values back into the struct fields by name.
func scatter[T any](dst *T, rec map[string]any) {
	rv := reflect.ValueOf(dst).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}
		val, ok := rec[rt.Field(i).Name]
		if !ok || val == nil {
			continue
		}
		assign(fv, val)
	}
}

func assign(fv reflect.Value, val any) {
	vv := reflect.ValueOf(val)
	ft := fv.Type()
	// Direct assignability (same type, incl. uuid.UUID, time.Time).
	if vv.IsValid() && vv.Type().AssignableTo(ft) {
		fv.Set(vv)
		return
	}
	// Pointer target: allocate and recurse.
	if ft.Kind() == reflect.Ptr {
		p := reflect.New(ft.Elem())
		assign(p.Elem(), val)
		fv.Set(p)
		return
	}
	// Nested object: a generated map[string]any scattered into a struct.
	if ft.Kind() == reflect.Struct {
		if m, ok := val.(map[string]any); ok {
			for i := 0; i < ft.NumField(); i++ {
				sub := fv.Field(i)
				if !sub.CanSet() {
					continue
				}
				if ev, ok := m[ft.Field(i).Name]; ok && ev != nil {
					assign(sub, ev)
				}
			}
			return
		}
	}
	// Array/slice: a generated []any scattered element-by-element.
	if ft.Kind() == reflect.Slice {
		if arr, ok := val.([]any); ok {
			s := reflect.MakeSlice(ft, len(arr), len(arr))
			for i, ev := range arr {
				assign(s.Index(i), ev)
			}
			fv.Set(s)
			return
		}
	}
	// Numeric coercion (int provider → int64 field, etc.).
	switch ft.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if vv.CanInt() {
			fv.SetInt(vv.Int())
		} else if vv.CanFloat() {
			fv.SetInt(int64(vv.Float()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if vv.CanInt() {
			fv.SetUint(uint64(vv.Int()))
		}
	case reflect.Float32, reflect.Float64:
		if vv.CanFloat() {
			fv.SetFloat(vv.Float())
		} else if vv.CanInt() {
			fv.SetFloat(float64(vv.Int()))
		}
	case reflect.String:
		if s, ok := val.(string); ok {
			fv.SetString(s)
		} else {
			fv.SetString(fmt.Sprint(val))
		}
	}
}

// extractPK reads the primary-key value from each parent for use as FK values.
// It uses the field tagged `synth:"pk"`, else a field named "ID", else the
// first field.
func extractPK[P any](parents []P) []any {
	out := make([]any, 0, len(parents))
	if len(parents) == 0 {
		return out
	}
	rt := reflect.TypeOf(parents[0])
	pkIdx := pkFieldIndex(rt)
	for i := range parents {
		rv := reflect.ValueOf(parents[i])
		out = append(out, rv.Field(pkIdx).Interface())
	}
	return out
}

func pkFieldIndex(rt reflect.Type) int {
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("synth")
		if tag == "pk" || tag == "pk,unique" {
			return i
		}
	}
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).Name == "ID" {
			return i
		}
	}
	return 0
}
