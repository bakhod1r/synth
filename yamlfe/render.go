package yamlfe

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/bakhodir/synth/schema"
)

// Render writes a schema back out as a YAML spec in the dialect Parse reads.
// Profiling a real export and rendering the result gives a spec you can commit
// to version control and generate from later without the original data.
//
// Round-tripping is the contract: Parse(Render(s)) must reproduce s.
func Render(s *schema.Schema, order []string, name string, count int) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("yamlfe: nil schema")
	}
	byName := make(map[string]schema.Field, len(s.Fields))
	for _, f := range s.Fields {
		byName[f.Name] = f
	}
	if len(order) == 0 {
		for _, f := range s.Fields {
			order = append(order, f.Name)
		}
	}

	var b bytes.Buffer
	if name != "" {
		fmt.Fprintf(&b, "name: %s\n", name)
	}
	if count > 0 {
		fmt.Fprintf(&b, "count: %d\n", count)
	}
	b.WriteString("fields:\n")
	for _, n := range order {
		f, ok := byName[n]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  %s: {%s}\n", yamlKey(n), renderField(f))
	}
	return b.Bytes(), nil
}

// renderField emits the inline mapping body for one field.
func renderField(f schema.Field) string {
	var parts []string
	kind := string(f.Kind)
	if kind == "" {
		kind = "lorem" // an uninferred column still needs a generator
	}
	parts = append(parts, "kind: "+kind)
	if f.PK {
		parts = append(parts, "pk: true")
	} else if f.Unique {
		parts = append(parts, "unique: true")
	}
	if f.From != "" {
		parts = append(parts, "from: "+f.From)
	}
	if f.Match != "" {
		parts = append(parts, "match: "+f.Match)
	}
	if len(f.Choices) > 0 {
		parts = append(parts, "choices: ["+strings.Join(quoteAll(f.Choices), ", ")+"]")
	}
	if len(f.Weights) > 0 {
		w := make([]string, len(f.Weights))
		for i, x := range f.Weights {
			w[i] = fmt.Sprintf("%g", x)
		}
		parts = append(parts, "weights: ["+strings.Join(w, ", ")+"]")
	}
	// Params carry min/max/dist/mu/sigma/s/rate/gap. Sort so output is stable.
	keys := make([]string, 0, len(f.Params))
	for k := range f.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, k+": "+f.Params[k])
	}
	return strings.Join(parts, ", ")
}

// quoteAll quotes enum choices so values containing a comma, a colon or a
// leading digit survive the inline-sequence syntax.
func quoteAll(vals []string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
	}
	return out
}

// yamlKey quotes a column name that would otherwise be misread as a different
// YAML type — real exports have columns called "no", "on" and "yes".
func yamlKey(n string) string {
	if n == "" {
		return `""`
	}
	switch strings.ToLower(n) {
	case "y", "n", "yes", "no", "on", "off", "true", "false", "null", "~":
		return `"` + n + `"`
	}
	if strings.ContainsAny(n, ":{}[],&*#?|-<>=!%@`\"' ") || (n[0] >= '0' && n[0] <= '9') {
		return `"` + strings.ReplaceAll(n, `"`, `\"`) + `"`
	}
	return n
}
