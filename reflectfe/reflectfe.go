// Package reflectfe is the struct frontend: it turns a Go type + `synth:` tags
// into a schema.Schema. Tag wins; when absent, it defers to the infer package.
// Results are cached per reflect.Type so reflection runs once per type.
package reflectfe

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/schema"
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
		} else {
			k, _ := infer.Kind(sf.Name, f.GoType)
			f.Kind = k
			if k == schema.KindUnknown {
				warns = append(warns, schema.Warning{Field: sf.Name, Reason: "no synonym or type match; left as zero value"})
			}
		}
		s.Fields = append(s.Fields, f)
	}
	infer.LinkDependencies(s)
	return &built{schema: s, warnings: warns}
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
