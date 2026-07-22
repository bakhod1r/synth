// Package protofe parses .proto files (proto2/proto3) into Synth schemas.
// The file is read as TEXT — no protoc, no code generation, no registry.
// Scalar fields map from their proto type; message-typed fields are resolved
// recursively; repeated fields become arrays; enums become weighted-free enums
// over their declared values.
package protofe

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bakhod1r/synth/infer"
	"github.com/bakhod1r/synth/schema"
)

// Message is one parsed protobuf message.
type Message struct {
	Name   string
	Schema *schema.Schema
	Order  []string
}

var (
	reMessage = regexp.MustCompile(`(?s)message\s+(\w+)\s*\{(.*?)\n\}`)
	reEnum    = regexp.MustCompile(`(?s)enum\s+(\w+)\s*\{(.*?)\n\}`)
	// field: [repeated|optional|required] type name = number;
	reField    = regexp.MustCompile(`^\s*(repeated\s+|optional\s+|required\s+)?([\w.]+)\s+(\w+)\s*=\s*\d+`)
	reEnumVal  = regexp.MustCompile(`^\s*(\w+)\s*=\s*\d+`)
	reComment  = regexp.MustCompile(`//.*`)
	reMapField = regexp.MustCompile(`^\s*map\s*<`)
)

// Parse reads all messages from .proto source.
func Parse(src string) ([]*Message, error) {
	src = reComment.ReplaceAllString(src, "")

	// Collect enum value sets so enum-typed fields become real choices.
	enums := map[string][]string{}
	for _, m := range reEnum.FindAllStringSubmatch(src, -1) {
		var vals []string
		for _, line := range strings.Split(m[2], "\n") {
			if v := reEnumVal.FindStringSubmatch(line); v != nil {
				vals = append(vals, v[1])
			}
		}
		if len(vals) > 0 {
			enums[m[1]] = vals
		}
	}

	// Parse message bodies, keeping raw bodies so nested messages resolve.
	bodies := map[string]string{}
	var order []string
	for _, m := range reMessage.FindAllStringSubmatch(src, -1) {
		bodies[m[1]] = m[2]
		order = append(order, m[1])
	}
	if len(bodies) == 0 {
		return nil, fmt.Errorf("protofe: no message found")
	}

	out := make([]*Message, 0, len(order))
	for _, name := range order {
		msg, err := buildMessage(name, bodies, enums, map[string]bool{})
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

// buildMessage converts one message body into a schema. seen guards against
// recursive message definitions (a self-referencing field is skipped).
func buildMessage(name string, bodies map[string]string, enums map[string][]string, seen map[string]bool) (*Message, error) {
	body, ok := bodies[name]
	if !ok {
		return nil, fmt.Errorf("protofe: unknown message %q", name)
	}
	seen[name] = true
	defer delete(seen, name)

	msg := &Message{Name: name, Schema: &schema.Schema{}}
	for _, line := range strings.Split(body, "\n") {
		if reMapField.MatchString(line) {
			continue // maps are not modeled in this version
		}
		m := reField.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		label, typ, fname := strings.TrimSpace(m[1]), m[2], m[3]
		f := schema.Field{Name: fname, Params: map[string]string{}}

		elemKind, isScalar := protoKind(fname, typ)
		switch {
		case isScalar:
			f.Kind = elemKind
		case enums[typ] != nil:
			f.Kind = schema.KindEnum
			f.Choices = append([]string(nil), enums[typ]...)
		case hasBody(bodies, typ) && !seen[typ]:
			sub, err := buildMessage(typ, bodies, enums, seen)
			if err != nil {
				return nil, err
			}
			f.Kind = schema.KindObject
			f.Nested = sub.Schema
		default:
			continue // unresolvable or recursive type
		}

		// `repeated` wraps whatever we resolved into an array field.
		if label == "repeated" {
			elem := f
			elem.Name = fname
			f = schema.Field{
				Name: fname, Params: map[string]string{},
				Kind: schema.KindArray, Elem: &elem,
				ArrMin: 1, ArrMax: 3,
			}
		}
		msg.Schema.Fields = append(msg.Schema.Fields, f)
		msg.Order = append(msg.Order, fname)
	}
	return msg, nil
}

// hasBody reports whether typ names a message defined in this file.
func hasBody(bodies map[string]string, typ string) bool {
	_, ok := bodies[typ]
	return ok
}

// Load parses a .proto file from disk.
func Load(path string) ([]*Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

// protoKind maps a protobuf scalar type to a Synth kind; unknown (message or
// enum) types return ok=false so the caller can resolve them.
func protoKind(fieldName, protoType string) (schema.Kind, bool) {
	switch protoType {
	case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
		"fixed32", "fixed64", "sfixed32", "sfixed64":
		return schema.KindInt, true
	case "float", "double":
		return schema.KindFloat, true
	case "bool":
		return schema.KindBool, true
	case "string", "bytes":
		if k, matched := infer.Kind(fieldName, ""); matched {
			return k, true
		}
		return schema.KindLorem, true
	case "google.protobuf.Timestamp":
		return schema.KindTime, true
	}
	return schema.KindUnknown, false
}
