// Package reflectfe is the struct frontend: it turns a Go type + `synth:` tags
// into a schema.Schema. Tag wins; when absent, it defers to the infer package.
// Results are cached per reflect.Type so reflection runs once per type.
package reflectfe

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/bakhod1r/synth/infer"
	"github.com/bakhod1r/synth/schema"
)

var cache sync.Map // reflect.Type -> *built

type built struct {
	schema   *schema.Schema
	warnings []schema.Warning
}

// Build returns the schema and warnings for type t (must be a struct).
func Build(t reflect.Type) (*schema.Schema, []schema.Warning) {
	if v, ok := cache.Load(t); ok {
		b := v.(*built)
		return b.schema, b.warnings
	}
	b := build(t)
	cache.Store(t, b)
	return b.schema, b.warnings
}

func build(t reflect.Type) *built {
	s := &schema.Schema{}
	var warns []schema.Warning
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		f := schema.Field{Name: sf.Name, GoType: goTypeName(sf.Type), Params: map[string]string{}}
		tag := sf.Tag.Get("synth")
		if tag != "" && tag != "-" {
			parseTag(&f, tag)
		} else if isStructural(sf.Type) {
			// Nested struct or slice: structure wins over any name synonym
			// (a field named "Address" of struct type is an object, not a street).
			enrichStructural(&f, sf.Type)
		} else {
			k, _ := infer.Kind(sf.Name, f.GoType)
			f.Kind = k
		}
		if f.Kind == schema.KindUnknown {
			warns = append(warns, schema.Warning{Field: sf.Name, Reason: "no synonym or type match; left as zero value"})
		}
		s.Fields = append(s.Fields, f)
	}
	infer.LinkDependencies(s)
	return &built{schema: s, warnings: warns}
}

// isStructural reports whether t is a nested struct or slice that should be
// generated recursively (not a scalar, and not time.Time/uuid.UUID).
func isStructural(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// uuid.UUID is a [16]byte array; treat it (and time.Time) as a scalar, not
	// as a nested array, so the Array case below never claims it.
	if isScalarStruct(t) {
		return false
	}
	switch t.Kind() {
	case reflect.Struct:
		return !isScalarStruct(t)
	case reflect.Slice, reflect.Array:
		return true
	}
	return false
}

// enrichStructural handles nested structs and slices. Pointers are unwrapped.
// time.Time and uuid.UUID are treated as scalars, not structs.
func enrichStructural(f *schema.Field, t reflect.Type) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if isScalarStruct(t) {
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		f.Kind = schema.KindObject
		f.Nested = build(t).schema
	case reflect.Slice, reflect.Array:
		elem := t.Elem()
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		ef := &schema.Field{Name: f.Name, GoType: goTypeName(elem), Params: map[string]string{}}
		if elem.Kind() == reflect.Struct && !isScalarStruct(elem) {
			ef.Kind = schema.KindObject
			ef.Nested = build(elem).schema
		} else {
			k, _ := infer.Kind(f.Name, goTypeName(elem))
			if k == schema.KindUnknown {
				k = scalarKind(elem)
			}
			ef.Kind = k
		}
		if ef.Kind == schema.KindUnknown {
			return
		}
		f.Kind = schema.KindArray
		f.Elem = ef
		f.ArrMin, f.ArrMax = 1, 3
	}
}

// isScalarStruct reports types we generate as single values, not sub-objects.
func isScalarStruct(t reflect.Type) bool {
	s := t.String()
	return s == "time.Time" || s == "uuid.UUID"
}

// scalarKind maps a slice element's Go kind to a default scalar Synth kind.
func scalarKind(t reflect.Type) schema.Kind {
	switch t.Kind() {
	case reflect.String:
		return schema.KindLorem
	case reflect.Bool:
		return schema.KindBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return schema.KindInt
	case reflect.Float32, reflect.Float64:
		return schema.KindFloat
	}
	return schema.KindUnknown
}

// parseTag reads `synth:"kind,opt=val,flag"`.
func parseTag(f *schema.Field, tag string) {
	parts := strings.Split(tag, ",")
	head := strings.TrimSpace(parts[0])
	switch head {
	case "pk":
		f.PK = true
		f.Unique = true
		if f.GoType == "uuid.UUID" || f.GoType == "string" {
			f.Kind = schema.KindUUID
		} else {
			f.Kind = schema.KindInt
			// Wide range so integer PKs stay unique across large datasets.
			f.Params["min"] = "1"
			f.Params["max"] = "2000000000"
		}
	default:
		f.Kind = schema.Kind(head)
	}
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if k, v, ok := strings.Cut(p, "="); ok {
			switch k {
			case "from", "after":
				f.From = v
			case "match":
				f.Match = v
			case "unique":
				// `unique=counter` picks the enforcement strategy; a bare
				// `unique` keeps the default (see schema.Field.UniqueMode).
				f.Unique, f.UniqueMode = true, v
			case "choices":
				f.Choices = strings.Split(v, "|")
			case "weights":
				for _, ws := range strings.Split(v, "|") {
					var w float64
					fmt.Sscanf(ws, "%g", &w)
					f.Weights = append(f.Weights, w)
				}
			default:
				f.Params[k] = v
			}
		} else {
			switch p {
			case "unique":
				f.Unique = true
			case "pk":
				f.PK = true
			default:
				f.Params[p] = "true"
			}
		}
	}
}

func goTypeName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return goTypeName(t.Elem())
	}
	switch t.String() {
	case "time.Time":
		return "time.Time"
	case "uuid.UUID":
		return "uuid.UUID"
	}
	if t.PkgPath() != "" {
		return t.Name()
	}
	return t.String()
}
