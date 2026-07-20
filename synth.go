// Package synth generates realistic, coherent, referentially-consistent
// records from plain Go structs. It is a pure data provider: it never touches
// the network, a database, or DDL. A struct goes in; records come out — in
// memory, to a file, or streamed.
package synth

import (
	"fmt"
	"reflect"
	"time"

	"github.com/bakhodir/synth/gen"
	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/reflectfe"
	"github.com/bakhodir/synth/schema"
)

func errNotStruct(v any) error {
	return fmt.Errorf("synth: requires a struct type, got %T", v)
}

// Option configures a generation call.
type Option func(*config)

type config struct {
	seed   uint64
	locale string
	refs   []refSpec
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

// Ref links a foreign-key field on the child to a parent slice, so every
// child row points at a real parent. Pass OneToMany to control cardinality.
func Ref[P any](parents []P, fkField string, opts ...RefOption) Option {
	pkValues := extractPK(parents)
	rs := refSpec{fkField: fkField, values: pkValues}
	for _, o := range opts {
		o(&rs)
	}
	return func(c *config) { c.refs = append(c.refs, rs) }
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

	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	base := rng.New(cfg.seed)
	out := make([]T, n)
	for i := 0; i < n; i++ {
		rec := eng.Record(base, i)
		scatter(&out[i], rec)
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
			f.FromRef = rs.values
			f.RefMin, f.RefMax = rs.min, rs.max
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
	if vv.Type().AssignableTo(ft) {
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
