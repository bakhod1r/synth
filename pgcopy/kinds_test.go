package pgcopy

import (
	"fmt"
	"testing"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// Every registered kind is generated and its actual Go type compared with what
// goTypeFor claims. Binary COPY decodes each field as whatever the DDL said the
// column is, so a kind whose type is guessed wrong here does not fail loudly —
// Postgres reads eight bytes of a float as a bigint and stores a number nobody
// questions.
//
// This is why the map lists exceptions rather than being maintained by hand: a
// kind added later that returns a float is caught here, not in production data.
func TestGoTypeForMatchesGeneratedValues(t *testing.T) {
	for _, k := range providers.Kinds() {
		k := k
		t.Run(string(k), func(t *testing.T) {
			s := &schema.Schema{Fields: []schema.Field{
				{Name: "v", Kind: k, Params: map[string]string{}},
			}}
			e, err := gen.Compile(s, "en_US")
			if err != nil {
				t.Skipf("kind %s does not compile standalone: %v", k, err)
			}
			rec := e.Record(rng.New(1), 0)
			v, ok := rec["v"]
			if !ok || v == nil {
				t.Skipf("kind %s generated no value", k)
			}
			got := actualGoType(v)
			if got == "" {
				t.Skipf("kind %s returns %T, which pgcopy does not model", k, v)
			}
			if want := goTypeFor(s.Fields[0]); got != want {
				t.Errorf("kind %s generates %s, but goTypeFor says %s "+
					"— add it to kindGoType", k, got, want)
			}
		})
	}
}

// actualGoType names the value's type the way pgTypes does, or "" for a type
// pgcopy has no column for (nested objects and arrays, which the CLI rejects
// before reaching here).
func actualGoType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64:
		return "int64"
	case float32, float64:
		return "float64"
	default:
		if fmt.Sprintf("%T", v) == "time.Time" {
			return "time.Time"
		}
		return ""
	}
}
