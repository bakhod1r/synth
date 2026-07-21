// Package schemafe parses JSON Schema and Avro schema documents into Synth
// schemas. Both are JSON, so they share one frontend. Files are read as text —
// Synth never contacts a schema registry or a database.
package schemafe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/schema"
)

// Table is a parsed record definition.
type Table struct {
	Name   string
	Schema *schema.Schema
	Order  []string
}

// --- JSON Schema ---

type jsonSchemaDoc struct {
	Title      string                   `json:"title"`
	Type       string                   `json:"type"`
	Required   []string                 `json:"required"`
	Properties map[string]jsonSchemaDoc `json:"properties"`
	Format     string                   `json:"format"`
	Enum       []any                    `json:"enum"`
	Minimum    *float64                 `json:"minimum"`
	Maximum    *float64                 `json:"maximum"`
	Items      *jsonSchemaDoc           `json:"items"`
	// propOrder preserves declaration order, filled by orderedKeys.
	propOrder []string
}

// ParseJSONSchema builds a schema from a JSON Schema document.
func ParseJSONSchema(data []byte) (*Table, error) {
	var doc jsonSchemaDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("schemafe: parse json schema: %w", err)
	}
	if len(doc.Properties) == 0 {
		return nil, fmt.Errorf("schemafe: json schema has no properties")
	}
	order, err := orderedKeys(data, "properties")
	if err != nil || len(order) == 0 {
		order = mapKeys(doc.Properties)
	}
	name := doc.Title
	if name == "" {
		name = "data"
	}
	t := &Table{Name: name, Schema: &schema.Schema{}}
	for _, key := range order {
		prop, ok := doc.Properties[key]
		if !ok {
			continue
		}
		f := schema.Field{Name: key, Params: map[string]string{}}
		f.Kind = jsonKind(key, prop)
		applyEnum(&f, prop.Enum)
		applyRange(&f, prop.Minimum, prop.Maximum)
		t.Schema.Fields = append(t.Schema.Fields, f)
		t.Order = append(t.Order, key)
	}
	return t, nil
}

func jsonKind(name string, p jsonSchemaDoc) schema.Kind {
	switch p.Format {
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
	case "ipv6":
		return schema.KindIPv6
	case "hostname":
		return schema.KindDomain
	}
	switch p.Type {
	case "integer":
		return schema.KindInt
	case "number":
		return schema.KindFloat
	case "boolean":
		return schema.KindBool
	case "string":
		if k, matched := infer.Kind(name, ""); matched {
			return k
		}
		return schema.KindLorem
	}
	if k, matched := infer.Kind(name, ""); matched {
		return k
	}
	return schema.KindLorem
}

// --- Avro ---

type avroSchema struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Fields []avroField `json:"fields"`
}

type avroField struct {
	Name string `json:"name"`
	Type any    `json:"type"` // string, []any (union), or object
}

// ParseAvro builds a schema from an Avro record schema.
func ParseAvro(data []byte) (*Table, error) {
	var doc avroSchema
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("schemafe: parse avro: %w", err)
	}
	if doc.Type != "record" || len(doc.Fields) == 0 {
		return nil, fmt.Errorf("schemafe: avro schema is not a record with fields")
	}
	name := doc.Name
	if name == "" {
		name = "data"
	}
	t := &Table{Name: name, Schema: &schema.Schema{}}
	for _, af := range doc.Fields {
		f := schema.Field{Name: af.Name, Params: map[string]string{}}
		f.Kind = avroKind(af.Name, af.Type)
		t.Schema.Fields = append(t.Schema.Fields, f)
		t.Order = append(t.Order, af.Name)
	}
	return t, nil
}

// avroKind resolves an Avro type, unwrapping unions like ["null","string"] and
// honoring logical types (uuid, timestamp-millis).
func avroKind(name string, typ any) schema.Kind {
	switch v := typ.(type) {
	case string:
		return avroPrimitive(name, v)
	case []any: // union: use the first non-null branch
		for _, b := range v {
			if s, ok := b.(string); ok && s == "null" {
				continue
			}
			return avroKind(name, b)
		}
	case map[string]any:
		if lt, ok := v["logicalType"].(string); ok {
			switch lt {
			case "uuid":
				return schema.KindUUID
			case "date", "timestamp-millis", "timestamp-micros":
				return schema.KindTime
			case "decimal":
				return schema.KindFloat
			}
		}
		if s, ok := v["type"].(string); ok {
			return avroPrimitive(name, s)
		}
	}
	return schema.KindLorem
}

func avroPrimitive(name, t string) schema.Kind {
	switch t {
	case "int", "long":
		return schema.KindInt
	case "float", "double":
		return schema.KindFloat
	case "boolean":
		return schema.KindBool
	case "string", "bytes":
		if k, matched := infer.Kind(name, ""); matched {
			return k
		}
		return schema.KindLorem
	}
	return schema.KindLorem
}

// --- shared helpers ---

// Load reads a schema file and picks the parser by content: an Avro record
// schema declares `"type":"record"`, otherwise it is treated as JSON Schema.
func Load(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err == nil && probe.Type == "record" {
		return ParseAvro(data)
	}
	return ParseJSONSchema(data)
}

func applyEnum(f *schema.Field, values []any) {
	if len(values) == 0 {
		return
	}
	f.Kind = schema.KindEnum
	for _, v := range values {
		f.Choices = append(f.Choices, fmt.Sprint(v))
	}
}

func applyRange(f *schema.Field, min, max *float64) {
	if min != nil {
		f.Params["min"] = fmt.Sprintf("%g", *min)
	}
	if max != nil {
		f.Params["max"] = fmt.Sprintf("%g", *max)
	}
}

// orderedKeys returns the keys of a top-level object field in document order,
// which encoding/json's map decoding would otherwise lose.
func orderedKeys(data []byte, field string) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	sub, ok := raw[field]
	if !ok {
		return nil, fmt.Errorf("schemafe: no %q object", field)
	}
	dec := json.NewDecoder(bytes.NewReader(sub))
	if _, err := dec.Token(); err != nil { // opening '{'
		return nil, err
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, fmt.Sprint(tok))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func mapKeys(m map[string]jsonSchemaDoc) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
