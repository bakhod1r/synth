package yamlfe

import (
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth/schema"
)

// An unquoted date in YAML is parsed by yaml.v3 as a time.Time, not a string.
// Formatting it back with fmt.Sprint gives "1960-01-01 00:00:00 +0000 UTC",
// which no provider parses — and since an unparseable bound is ignored rather
// than rejected, the field generated outside its range with nothing to show for
// it. The user preset shipped that way: dates of birth in 2025.
func TestUnquotedDateBoundsReachTheProvider(t *testing.T) {
	src := []byte("name: t\ncount: 1\nfields:\n  dob: { kind: time, min: 1960-01-01, max: 2006-12-31 }\n")
	spec, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	f := spec.Schema.FieldByName("dob")
	if f == nil {
		t.Fatal("no dob field")
	}
	for _, key := range []string{"min", "max"} {
		got := f.Params[key]
		if _, err := time.Parse(time.RFC3339, got); err != nil {
			t.Errorf("%s = %q, which no provider can parse: %v", key, got, err)
		}
	}
}

// The same bound written as a quoted string must give the same result, or the
// two spellings of one spec behave differently.
func TestQuotedAndUnquotedDatesAgree(t *testing.T) {
	unquoted, err := Parse([]byte("name: t\nfields:\n  d: { kind: time, min: 1960-01-01 }\n"))
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := Parse([]byte("name: t\nfields:\n  d: { kind: time, min: \"1960-01-01\" }\n"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := time.Parse(time.RFC3339, unquoted.Schema.FieldByName("d").Params["min"])
	b := quoted.Schema.FieldByName("d").Params["min"]
	if bt, err := time.Parse("2006-01-02", b); err != nil || !a.Equal(bt) {
		t.Fatalf("quoted %q and unquoted %v disagree", b, a)
	}
}

// Render must produce something Parse accepts, whatever the column is called.
//
// This is the property that broke: a column named "\x10" — which a badly
// exported CSV really can produce — was written out raw, and the spec would not
// parse. Profiling a file made the tool emit something it could not read back.
func TestRenderRoundTripsAnyColumnName(t *testing.T) {
	names := []string{
		"email", "no", "on", "yes", "true", "null", "~", "123", "",
		"a: b", "a,b", "a{b}", "a[b]", "#comment", "-dash", "with space",
		"\x10", "\x00", "\x7f", "tab\there", "new\nline", "quote\"d", `back\slash`,
		"емейл", "電子メール", "بريد", "🙂",
		strings.Repeat("very_long_", 40),
		// Long enough that its escaped form passes YAML's 1024-character limit
		// on a simple key, which needs the explicit `? key` form instead.
		strings.Repeat("\x01", 600),
		strings.Repeat("ключ_", 300),
	}
	fields := make([]schema.Field, 0, len(names))
	order := make([]string, 0, len(names))
	for _, n := range names {
		fields = append(fields, schema.Field{Name: n, Kind: schema.KindLorem})
		order = append(order, n)
	}

	out, err := Render(&schema.Schema{Fields: fields}, order, "tricky name: with, punctuation", 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(out)
	if err != nil {
		t.Fatalf("Render produced a spec Parse rejects: %v\n%s", err, out)
	}
	if len(spec.Schema.Fields) != len(names) {
		t.Fatalf("round trip kept %d of %d columns", len(spec.Schema.Fields), len(names))
	}
	// The names must survive intact, not merely parse.
	got := map[string]bool{}
	for _, f := range spec.Schema.Fields {
		got[f.Name] = true
	}
	for _, n := range names {
		if !got[n] {
			t.Errorf("column %q did not survive the round trip", n)
		}
	}
}

// Enum choices and param values come from real data too, and land inside a flow
// sequence where a comma splits an item in two.
func TestRenderRoundTripsAwkwardValues(t *testing.T) {
	f := schema.Field{
		Name:    "status",
		Kind:    schema.KindEnum,
		Choices: []string{"a,b", "c: d", "e}f", "", "\x01", "plain"},
		Params:  map[string]string{"sep": ", ", "min": "2026-01-01"},
	}
	out, err := Render(&schema.Schema{Fields: []schema.Field{f}}, []string{"status"}, "t", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse rejected: %v\n%s", err, out)
	}
	got := spec.Schema.FieldByName("status")
	if got == nil {
		t.Fatal("the field vanished")
	}
	if len(got.Choices) != len(f.Choices) {
		t.Fatalf("choices: got %q, want %q", got.Choices, f.Choices)
	}
	for i := range f.Choices {
		if got.Choices[i] != f.Choices[i] {
			t.Errorf("choice %d: got %q, want %q", i, got.Choices[i], f.Choices[i])
		}
	}
	if got.Params["sep"] != ", " {
		t.Errorf("sep: got %q, want %q", got.Params["sep"], ", ")
	}
}
