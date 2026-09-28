// Package openapi is a frontend that turns an OpenAPI 3 spec into Synth
// schemas, so you can generate valid request payloads for an endpoint without
// hand-writing a struct. It maps JSON Schema (type/format/enum/min/max) onto
// schema.Kind, then the normal engine produces records.
//
// Scope: the application/json request body of an operation, including local
// $refs into components/schemas, nested objects, and arrays (items, minItems,
// maxItems). A property that refers back to a schema already being expanded
// is left out, so recursive schemas terminate. pattern is not supported.
package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/schema"
	"gopkg.in/yaml.v3"
)

// Spec is a parsed OpenAPI document.
type Spec struct {
	doc document
}

type document struct {
	Paths      map[string]map[string]operation `yaml:"paths" json:"paths"`
	Components struct {
		Schemas map[string]jsonSchema `yaml:"schemas" json:"schemas"`
	} `yaml:"components" json:"components"`
}

type operation struct {
	RequestBody struct {
		Content map[string]struct {
			Schema jsonSchema `yaml:"schema" json:"schema"`
		} `yaml:"content" json:"content"`
	} `yaml:"requestBody" json:"requestBody"`
}

type jsonSchema struct {
	Ref        string                `yaml:"$ref" json:"$ref"`
	Type       string                `yaml:"type" json:"type"`
	Format     string                `yaml:"format" json:"format"`
	Enum       []string              `yaml:"enum" json:"enum"`
	Minimum    *float64              `yaml:"minimum" json:"minimum"`
	Maximum    *float64              `yaml:"maximum" json:"maximum"`
	MaxLength  *int                  `yaml:"maxLength" json:"maxLength"`
	Required   []string              `yaml:"required" json:"required"`
	Properties map[string]jsonSchema `yaml:"properties" json:"properties"`
	Items      *jsonSchema           `yaml:"items" json:"items"`
	MinItems   *int                  `yaml:"minItems" json:"minItems"`
	MaxItems   *int                  `yaml:"maxItems" json:"maxItems"`
}

// Load parses an OpenAPI spec from a file (YAML or JSON).
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses an OpenAPI spec from bytes (YAML or JSON; YAML is a superset).
func Parse(data []byte) (*Spec, error) {
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("openapi: parse: %w", err)
	}
	return &Spec{doc: doc}, nil
}

// Schema builds a Synth schema for the request body of method+path.
func (s *Spec) Schema(method, path string) (*schema.Schema, error) {
	ops, ok := s.doc.Paths[path]
	if !ok {
		return nil, fmt.Errorf("openapi: path %q not found", path)
	}
	op, ok := ops[strings.ToLower(method)]
	if !ok {
		return nil, fmt.Errorf("openapi: method %s %s not found", method, path)
	}
	body, ok := op.RequestBody.Content["application/json"]
	if !ok {
		return nil, fmt.Errorf("openapi: %s %s has no application/json body", method, path)
	}
	root, rootRef := s.resolveNamed(body.Schema)
	if root.Type != "object" && root.Properties == nil {
		return nil, fmt.Errorf("openapi: request body is not an object")
	}
	stack := map[string]bool{}
	if rootRef != "" {
		stack[rootRef] = true
	}
	return s.object(root, stack), nil
}

// object builds the schema for an object's properties. stack holds the
// component names being expanded on this path: a schema that refers back to
// itself (Customer.referrer: Customer) has that property left out rather than
// expanded forever.
func (s *Spec) object(root jsonSchema, stack map[string]bool) *schema.Schema {
	// Sorted: map order is random, and the order fields are drawn in decides
	// the values, so an unsorted walk gives different payloads per run for
	// one seed.
	names := make([]string, 0, len(root.Properties))
	for name := range root.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	out := &schema.Schema{}
	for _, name := range names {
		if f, ok := s.field(name, root.Properties[name], stack); ok {
			out.Fields = append(out.Fields, f)
		}
	}
	return out
}

// field maps one property. It reports false for a property that recurses into
// a schema already being expanded.
func (s *Spec) field(name string, prop jsonSchema, stack map[string]bool) (schema.Field, bool) {
	p, ref := s.resolveNamed(prop)
	if ref != "" {
		if stack[ref] {
			return schema.Field{}, false
		}
		stack[ref] = true
		defer delete(stack, ref)
	}
	f := schema.Field{Name: name, Params: map[string]string{}, Kind: mapKind(p)}
	switch {
	case len(p.Enum) > 0:
		f.Kind = schema.KindEnum
		f.Choices = p.Enum
	case p.Type == "object" || (p.Type == "" && len(p.Properties) > 0):
		f.Kind = schema.KindObject
		f.Nested = s.object(p, stack)
		return f, true
	case p.Type == "array":
		if p.Items == nil {
			return f, true // mapKind's fallback: nothing said about elements
		}
		elem, ok := s.field(name, *p.Items, stack)
		if !ok {
			return schema.Field{}, false
		}
		f.Kind = schema.KindArray
		f.Elem = &elem
		f.ArrMin, f.ArrMax = 1, 3
		if p.MinItems != nil {
			f.ArrMin = *p.MinItems
		}
		if p.MaxItems != nil {
			f.ArrMax = *p.MaxItems
		} else if f.ArrMin > f.ArrMax {
			f.ArrMax = f.ArrMin
		}
		return f, true
	}
	if p.Minimum != nil {
		f.Params["min"] = strconv.FormatFloat(*p.Minimum, 'f', -1, 64)
	}
	if p.Maximum != nil {
		f.Params["max"] = strconv.FormatFloat(*p.Maximum, 'f', -1, 64)
	}
	// maxLength is a real constraint on the endpoint: a payload that
	// exceeds it is one the API would reject. The generator truncates to
	// it, so generated request bodies stay valid.
	if p.MaxLength != nil && *p.MaxLength > 0 {
		f.Params["maxlen"] = strconv.Itoa(*p.MaxLength)
	}
	return f, true
}

// resolveNamed is resolve that also names the component it followed, or ""
// when there was no $ref. Chains of refs are followed up to a small limit.
func (s *Spec) resolveNamed(js jsonSchema) (jsonSchema, string) {
	name := ""
	for i := 0; js.Ref != "" && i < 8; i++ {
		n := js.Ref[strings.LastIndex(js.Ref, "/")+1:]
		target, ok := s.doc.Components.Schemas[n]
		if !ok {
			break
		}
		name, js = n, target
	}
	return js, name
}

// mapKind maps a JSON Schema type+format to a Synth kind.
func mapKind(js jsonSchema) schema.Kind {
	switch js.Format {
	case "email":
		return schema.KindEmail
	case "uuid":
		return schema.KindUUID
	case "date-time", "date":
		return schema.KindTime
	case "uri", "url":
		return schema.KindURL
	case "ipv4":
		return schema.KindIPv4
	}
	switch js.Type {
	case "integer":
		return schema.KindInt
	case "number":
		return schema.KindFloat
	case "boolean":
		return schema.KindBool
	case "string":
		return schema.KindLorem
	}
	return schema.KindLorem
}

// PayloadJSON marshals a generated record map into indented JSON.
func PayloadJSON(rec map[string]any) ([]byte, error) {
	return json.MarshalIndent(rec, "", "  ")
}
