package yamlfe

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bakhod1r/synth/constraint"
	"github.com/bakhod1r/synth/schema"
)

// Render writes a schema back out as a YAML spec in the dialect Parse reads.
// Profiling a real export and rendering the result gives a spec you can commit
// to version control and generate from later without the original data.
//
// Round-tripping is the contract: Parse(Render(s)) must reproduce s.
func Render(s *schema.Schema, order []string, name string, count int, cs []constraint.Constraint) ([]byte, error) {
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
		fmt.Fprintf(&b, "name: %s\n", yamlString(name))
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
		writeEntry(&b, n, renderField(f))
	}
	b.WriteString(renderConstraints(cs))
	return b.Bytes(), nil
}

// maxSimpleKey is YAML's limit on a simple mapping key: 1024 characters, and it
// counts the escaped form. A column name long enough to exceed it after
// escaping cannot be written as `key: value` at all — the parser reports
// "mapping values are not allowed in this context", which says nothing about
// the real cause.
const maxSimpleKey = 1024

// writeEntry writes one field, falling back to YAML's explicit-key form when
// the key is too long to be a simple one.
//
// The fallback exists because the alternative is refusing to profile a file
// over a column name, and a name that long is somebody's real export however
// unlikely it looks.
func writeEntry(b *bytes.Buffer, name, body string) {
	key := yamlKey(name)
	if len(key) <= maxSimpleKey {
		fmt.Fprintf(b, "  %s: {%s}\n", key, body)
		return
	}
	fmt.Fprintf(b, "  ? %s\n  : {%s}\n", key, body)
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
	if f.UniqueMode != "" {
		// The mode is a fixed keyword, not user text, so it needs no quoting.
		parts = append(parts, "unique_mode: "+f.UniqueMode)
	}
	// from= and match= name other columns, so they carry the same hazards as a
	// key does.
	if f.From != "" {
		parts = append(parts, "from: "+yamlString(f.From))
	}
	if f.Match != "" {
		parts = append(parts, "match: "+yamlString(f.Match))
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
		// mu/sigma/s/rate are typed *float64 on the parse side; quoting them
		// ("mu: \"8\"") makes YAML refuse to unmarshal them back into a float,
		// so they must round-trip as bare numbers.
		if numericParam[k] {
			parts = append(parts, k+": "+f.Params[k])
			continue
		}
		// A param value profiled from real data is arbitrary text — a date
		// bound, a separator, a category name — so it is quoted like any other
		// value that did not originate here.
		parts = append(parts, k+": "+yamlString(f.Params[k]))
	}
	return strings.Join(parts, ", ")
}

// numericParam lists params that Parse reads into a typed *float64. They must
// render as bare numbers so a re-parse can unmarshal them. min/max are excluded
// because they may legitimately be date strings.
var numericParam = map[string]bool{"mu": true, "sigma": true, "s": true, "rate": true}

// quoteAll renders enum choices for an inline sequence.
func quoteAll(vals []string) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = yamlString(v)
	}
	return out
}

// yamlString renders any Go string as a YAML double-quoted scalar on one line.
//
// Everything this file emits ends up inside a flow mapping or sequence —
// `{kind: x, from: y}`, `[a, b]` — where a plain scalar containing a comma or a
// brace silently becomes two items or a syntax error. Double-quoting everything
// that came from data removes the question.
//
// yaml.Marshal is not used here for two reasons: it leaves a comma unquoted,
// which is exactly the flow-context hazard, and it folds a long string across
// lines, which cannot appear inside a flow mapping.
//
// Control characters are the case that made this necessary. A column named
// "\x10" — which a real, badly-exported CSV can produce — used to be written
// out raw, and the resulting spec would not parse: profiling a file made the
// tool emit something it could not read back.
func yamlString(v string) string {
	var b strings.Builder
	b.Grow(len(v) + 2)
	b.WriteByte('"')
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case 0x2028:
			b.WriteString(`\L`) // line separator
		case 0x2029:
			b.WriteString(`\P`) // paragraph separator
		default:
			switch {
			// C0, DEL and the whole C1 block. YAML's printable set excludes
			// them, and an unescaped one makes the document unreadable — which
			// is how a column named "\u0089" produced a spec Synth could not
			// parse back.
			case r < 0x20 || (r >= 0x7f && r <= 0x9f):
				fmt.Fprintf(&b, `\x%02x`, r)
			case r == utf8.RuneError:
				// Invalid UTF-8 reaches here as U+FFFD. Emitting it as an
				// escape keeps the document valid; emitting the raw bytes
				// would not.
				b.WriteString(`\uFFFD`)
			default:
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// yamlKey renders a column name as a mapping key.
//
// Every name is quoted rather than only the ones that look dangerous. The old
// version kept a list of characters to watch for, and the list was wrong: it
// covered "no" and "on" and a leading digit, and missed control characters
// entirely. A list of exceptions is the wrong shape for this problem.
func yamlKey(n string) string { return yamlString(n) }

// renderConstraints emits the mined-invariant block. Each line carries the
// support count as a comment so a reader can judge the evidence behind it.
func renderConstraints(cs []constraint.Constraint) string {
	if len(cs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("constraints:\n")
	for _, c := range cs {
		switch c.Kind {
		case constraint.Ordering:
			fmt.Fprintf(&b, "  - {kind: ordering, left: %s, right: %s}",
				yamlString(c.Left), yamlString(c.Right))
		case constraint.SumEquals:
			fmt.Fprintf(&b, "  - {kind: sum, parts: [%s], whole: %s}",
				strings.Join(quoteAll(c.Parts), ", "), yamlString(c.Whole))
		case constraint.Implication:
			fmt.Fprintf(&b, "  - {kind: implication, when: %s, equals: %s, then: %s, exclusive: %t}",
				yamlString(c.When), yamlString(c.Equals), yamlString(c.Then), c.Exclusive)
		case constraint.Range:
			fmt.Fprintf(&b, "  - {kind: range, left: %s, lo: %g, hi: %g}",
				yamlString(c.Left), c.Lo, c.Hi)
		default:
			continue
		}
		fmt.Fprintf(&b, "  # held over %d rows\n", c.Support)
	}
	return b.String()
}
