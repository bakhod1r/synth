// Package openapi is a frontend that turns an OpenAPI 3 spec into Synth
// schemas, so you can generate valid request payloads for an endpoint without
// hand-writing a struct. It maps JSON Schema (type/format/enum/min/max) onto
// schema.Kind, then the normal engine produces records.
//
// Scope: top-level object properties of a request body's application/json
// schema (including a local $ref into components/schemas). Nested objects and
// arrays are generated shallowly (as inferred scalars) in this version.
package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bakhodir/synth/schema"
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
	Required   []string              `yaml:"required" json:"required"`
	Properties map[string]jsonSchema `yaml:"properties" json:"properties"`
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
	root := s.resolve(body.Schema)
	if root.Type != "object" && root.Properties == nil {
		return nil, fmt.Errorf("openapi: request body is not an object")
	}
	required := map[string]bool{}
	for _, r := range root.Required {
		required[r] = true
	}
	out := &schema.Schema{}
	for name, prop := range root.Properties {
		p := s.resolve(prop)
		f := schema.Field{Name: name, Params: map[string]string{}, Kind: mapKind(p)}
		if len(p.Enum) > 0 {
			f.Kind = schema.KindEnum
			f.Choices = p.Enum
		}
		if p.Minimum != nil {
			f.Params["min"] = fmt.Sprintf("%g", *p.Minimum)
		}
		if p.Maximum != nil {
			f.Params["max"] = fmt.Sprintf("%g", *p.Maximum)
		}
		out.Fields = append(out.Fields, f)
	}
	return out, nil
}

// resolve follows a local $ref into components/schemas (one level).
func (s *Spec) resolve(js jsonSchema) jsonSchema {
	if js.Ref == "" {
		return js
	}
	name := js.Ref[strings.LastIndex(js.Ref, "/")+1:]
	if target, ok := s.doc.Components.Schemas[name]; ok {
		return target
	}
	return js
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
