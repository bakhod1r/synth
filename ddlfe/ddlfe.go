// Package ddlfe is a frontend that parses SQL CREATE TABLE statements (from a
// .sql file or migration) into Synth schemas. It reads DDL as TEXT — Synth
// never connects to a database. Column names drive inference (an "email"
// column becomes an email), SQL types set the kind, and NOT NULL / PRIMARY KEY
// are honored.
package ddlfe

import (
	"regexp"
	"strings"

	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/schema"
)

// Table is one parsed CREATE TABLE.
type Table struct {
	Name   string
	Schema *schema.Schema
	Order  []string
}

var (
	// The name may be quoted four different ways and prefixed by a schema —
	// pg_dump writes `public.users`, SQL Server writes `[users]`. Only the last
	// segment is kept: the schema qualifier is where the table lives, not what
	// it is called, and carrying it into a generated CSV header helps nobody.
	ident    = `(?:"[^"]+"|` + "`[^`]+`" + `|\[[^\]]+\]|\w+)`
	reCreate = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(` +
		ident + `(?:\s*\.\s*` + ident + `)*)\s*\((.*?)\)\s*;`)

	// PRIMARY KEY (a, b) as a table-level constraint. Dropping it would leave
	// the key column looking ordinary and silently produce duplicates.
	rePKConstraint     = regexp.MustCompile(`(?is)^\s*(?:CONSTRAINT\s+` + ident + `\s+)?PRIMARY\s+KEY\s*\((.*)\)\s*$`)
	reUniqueConstraint = regexp.MustCompile(`(?is)^\s*(?:CONSTRAINT\s+` + ident + `\s+)?UNIQUE\s*\((.*)\)\s*$`)
)

// unquote strips the quoting styles the four major databases use, and reduces a
// qualified name to its last segment.
func unquote(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = strings.TrimSpace(name[i+1:])
	}
	if len(name) >= 2 {
		switch {
		case name[0] == '"' && name[len(name)-1] == '"',
			name[0] == '`' && name[len(name)-1] == '`':
			return name[1 : len(name)-1]
		case name[0] == '[' && name[len(name)-1] == ']':
			return name[1 : len(name)-1]
		}
	}
	return name
}

// Parse reads all CREATE TABLE statements in the given SQL text.
func Parse(sql string) ([]*Table, error) {
	var tables []*Table
	for _, m := range reCreate.FindAllStringSubmatch(sql, -1) {
		t := &Table{Name: unquote(m[1]), Schema: &schema.Schema{}}
		var pk, unique []string
		for _, col := range splitColumns(m[2]) {
			if cols, ok := constraintColumns(col, rePKConstraint); ok {
				pk = append(pk, cols...)
				continue
			}
			if cols, ok := constraintColumns(col, reUniqueConstraint); ok {
				unique = append(unique, cols...)
				continue
			}
			f, ok := parseColumn(col)
			if !ok {
				continue // FOREIGN KEY, CHECK and the rest
			}
			t.Schema.Fields = append(t.Schema.Fields, f)
			t.Order = append(t.Order, f.Name)
		}
		// A table-level constraint is applied after the columns exist, because
		// it names them.
		for _, name := range pk {
			if f := t.Schema.FieldByName(name); f != nil {
				f.PK, f.Unique = true, true
			}
		}
		for _, name := range unique {
			if f := t.Schema.FieldByName(name); f != nil {
				f.Unique = true
			}
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// constraintColumns returns the column names a table-level constraint covers.
func constraintColumns(clause string, re *regexp.Regexp) ([]string, bool) {
	m := re.FindStringSubmatch(clause)
	if m == nil {
		return nil, false
	}
	var out []string
	for _, part := range strings.Split(m[1], ",") {
		if name := unquote(part); name != "" {
			out = append(out, name)
		}
	}
	return out, true
}

// splitColumns splits the body on top-level commas (ignoring commas inside
// parentheses like NUMERIC(10,2) or PRIMARY KEY(a,b)).
func splitColumns(body string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range body {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(body[start:i]))
				start = i + 1
			}
		}
	}
	if s := strings.TrimSpace(body[start:]); s != "" {
		out = append(out, s)
	}
	return out
}

var constraintKeywords = map[string]bool{
	"primary": true, "foreign": true, "unique": true, "constraint": true,
	"check": true, "key": true, "index": true,
}

// parseColumn turns one column definition into a schema.Field. Returns ok=false
// for table-level constraints.
func parseColumn(col string) (schema.Field, bool) {
	fields := strings.Fields(col)
	if len(fields) < 2 {
		return schema.Field{}, false
	}
	name := strings.Trim(fields[0], "\"`[]")
	if constraintKeywords[strings.ToLower(name)] {
		return schema.Field{}, false
	}
	sqlType := strings.ToLower(fields[1])
	rest := strings.ToLower(col)

	f := schema.Field{Name: name, Params: map[string]string{}}
	f.Kind = kindForColumn(name, sqlType)
	if strings.Contains(rest, "primary key") {
		f.PK = true
		f.Unique = true
	} else if strings.Contains(rest, "unique") {
		f.Unique = true
	}
	// Wide range so integer/serial PKs stay unique across large datasets.
	if f.PK && f.Kind == schema.KindInt {
		f.Params["min"] = "1"
		f.Params["max"] = "2000000000"
	}
	return f, true
}

// kindForColumn resolves a column's kind. A concrete SQL type (numeric, uuid,
// bool, timestamp) wins — "id SERIAL" must stay an int, not become a UUID from
// the name. For string-like types (varchar/text), the column NAME drives
// inference so "email VARCHAR" becomes an email.
func kindForColumn(name, sqlType string) schema.Kind {
	typeKind := kindForSQLType(sqlType)
	if typeKind != schema.KindLorem {
		return typeKind // concrete non-string type
	}
	if k, matched := infer.Kind(name, ""); matched {
		return k
	}
	return schema.KindLorem
}

// kindForSQLType maps a SQL column type to a scalar Synth kind.
func kindForSQLType(t string) schema.Kind {
	base := t
	if i := strings.IndexByte(base, '('); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "uuid":
		return schema.KindUUID
	case "int", "integer", "smallint", "bigint", "serial", "bigserial", "int4", "int8":
		return schema.KindInt
	case "decimal", "numeric", "real", "double", "float", "money":
		return schema.KindFloat
	case "bool", "boolean":
		return schema.KindBool
	case "timestamp", "timestamptz", "date", "datetime", "time":
		return schema.KindTime
	default:
		return schema.KindLorem // varchar/text/char/etc.
	}
}
