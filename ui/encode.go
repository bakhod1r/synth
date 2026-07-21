package ui

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
)

// buildYAML renders the posted schema as a YAML spec, so the UI and the CLI
// go through exactly the same parser. A preview that took a different path
// than `synth gen` would be a preview of the wrong thing.
func buildYAML(req *specRequest) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", nonEmpty(req.Name, "data"))
	fmt.Fprintf(&b, "count: %d\n", req.Count)
	if req.Locale != "" {
		fmt.Fprintf(&b, "locale: %s\n", req.Locale)
	}
	fmt.Fprintf(&b, "seed: %d\n", req.Seed)
	b.WriteString("fields:\n")

	order := req.Order
	if len(order) == 0 {
		for name := range req.Fields {
			order = append(order, name)
		}
		sort.Strings(order)
	}
	for _, name := range order {
		def, ok := req.Fields[name]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  %q: {%s}\n", name, renderDef(def))
	}
	return []byte(b.String()), nil
}

// renderDef emits one field's inline mapping. Keys are sorted so the same
// schema always produces the same document.
func renderDef(def map[string]any) string {
	keys := make([]string, 0, len(def))
	for k := range def {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		switch v := def[k].(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s: %q", k, v))
		case bool:
			parts = append(parts, fmt.Sprintf("%s: %t", k, v))
		case float64:
			parts = append(parts, fmt.Sprintf("%s: %g", k, v))
		case []any:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = fmt.Sprintf("%q", fmt.Sprint(item))
			}
			parts = append(parts, fmt.Sprintf("%s: [%s]", k, strings.Join(items, ", ")))
		default:
			parts = append(parts, fmt.Sprintf("%s: %q", k, fmt.Sprint(v)))
		}
	}
	return strings.Join(parts, ", ")
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func writeCSV(w io.Writer, cols []string, rows []map[string]any) {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write(cols)
	rec := make([]string, len(cols))
	for _, r := range rows {
		for i, c := range cols {
			rec[i] = fmt.Sprint(r[c])
		}
		cw.Write(rec)
	}
}

func writeSQL(w io.Writer, table string, cols []string, rows []map[string]any) {
	colList := strings.Join(cols, ", ")
	for _, r := range rows {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = sqlValue(r[c])
		}
		fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", "))
	}
}

func sqlValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(x)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}
