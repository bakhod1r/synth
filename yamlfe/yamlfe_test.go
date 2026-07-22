package yamlfe

import (
	"testing"
	"time"
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
