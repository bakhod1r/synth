// Package pgcopy writes generated rows in the two formats Postgres COPY
// accepts, which is the fast way to get bulk data into a table: an INSERT
// statement per row is the slowest path the server offers, and at the volumes
// Synth targets the difference is hours.
//
// Like every Synth output this writes a FILE. Synth never opens a database
// connection — handing the file to psql or your loader is your step:
//
//	COPY users FROM '/tmp/users.pgcopy';
//	COPY users FROM '/tmp/users.pgbin' WITH (FORMAT binary);
//
// Both writers stream. No row is retained, so 100M rows cost the same memory
// as one.
package pgcopy

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"time"
)

// Writer is what both formats implement, so callers pick a format once and
// then write rows without caring which.
type Writer interface {
	// WriteRow writes one record. Columns absent from the map are NULL.
	WriteRow(row map[string]any) error
	// Close flushes any buffered output and writes the format's trailer.
	Close() error
}

// TextWriter writes the default COPY text format: tab-separated, one row per
// line, \N for NULL.
type TextWriter struct {
	w    *bufio.Writer
	cols []string
}

// NewText returns a writer emitting the columns in the order given. That order
// is the column order the COPY command must use.
func NewText(w io.Writer, cols []string) *TextWriter {
	return &TextWriter{w: bufio.NewWriter(w), cols: cols}
}

func (t *TextWriter) WriteRow(row map[string]any) error {
	for i, c := range t.cols {
		if i > 0 {
			if err := t.w.WriteByte('\t'); err != nil {
				return err
			}
		}
		v, ok := row[c]
		if !ok || v == nil {
			if _, err := t.w.WriteString(`\N`); err != nil {
				return err
			}
			continue
		}
		if _, err := t.w.WriteString(escapeText(textValue(v))); err != nil {
			return err
		}
	}
	return t.w.WriteByte('\n')
}

// Close flushes. The text format has no trailer.
func (t *TextWriter) Close() error { return t.w.Flush() }

// textValue renders a value the way Postgres parses it back.
func textValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "t"
		}
		return "f"
	case time.Time:
		// RFC 3339 in UTC. Sending an offset would make the loaded value
		// depend on the server's TimeZone setting for a timestamp column.
		return x.UTC().Format(time.RFC3339Nano)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprint(v)
	}
}

// escapeText escapes the four characters that would otherwise be read as
// structure. A tab or a newline inside a value shifts every column after it,
// which is the kind of corruption that loads without complaint and is found
// much later.
func escapeText(s string) string {
	if !needsEscape(s) {
		return s
	}
	var b []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b = append(b, '\\', '\\')
		case '\t':
			b = append(b, '\\', 't')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		default:
			b = append(b, s[i])
		}
	}
	return string(b)
}

func needsEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '\t', '\n', '\r':
			return true
		}
	}
	return false
}

var (
	_ Writer = (*TextWriter)(nil)
	_ Writer = (*BinaryWriter)(nil)
)
