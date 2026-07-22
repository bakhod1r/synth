package pgcopy

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func testSchema() *schema.Schema {
	return &schema.Schema{Fields: []schema.Field{
		{Name: "id", Kind: schema.KindUUID, GoType: "string", PK: true, Params: map[string]string{}},
		{Name: "nickname", Kind: schema.KindUsername, GoType: "string",
			Params: map[string]string{"maxlen": "12"}},
		{Name: "bio", Kind: schema.KindLorem, GoType: "string", Params: map[string]string{}},
		{Name: "age", Kind: schema.KindInt, GoType: "int", Params: map[string]string{}},
		{Name: "score", Kind: schema.KindFloat, GoType: "float64", Params: map[string]string{}},
		{Name: "active", Kind: schema.KindBool, GoType: "bool", Params: map[string]string{}},
		{Name: "created_at", Kind: schema.KindTime, GoType: "time.Time", Params: map[string]string{}},
	}}
}

func TestDDL(t *testing.T) {
	got := DDL("users", testSchema(), "/tmp/users.pgbin", true)
	for _, want := range []string{
		`CREATE TABLE "users" (`,
		`"id" varchar`,
		`"nickname" varchar(12)`,
		`"bio" varchar`,
		`"age" bigint`,
		`"score" double precision`,
		`"active" boolean`,
		`"created_at" timestamp`,
		`COPY "users" FROM '/tmp/users.pgbin' WITH (FORMAT binary);`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DDL missing %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "varchar(0)") {
		t.Error("an unbounded string got a length")
	}
}

func TestDDLTextFormatCopyCommand(t *testing.T) {
	got := DDL("users", testSchema(), "/tmp/users.pgcopy", false)
	if want := `COPY "users" FROM '/tmp/users.pgcopy';`; !strings.Contains(got, want) {
		t.Errorf("DDL missing %q, got:\n%s", want, got)
	}
	if strings.Contains(got, "FORMAT binary") {
		t.Error("text output claimed the binary format")
	}
}

// The binary encoder and the DDL writer must agree on every type. If they
// drift, Postgres either rejects the file or misreads it — the failure this
// whole pairing exists to prevent, so it is asserted rather than assumed.
func TestDDLTypesMatchBinaryEncoding(t *testing.T) {
	for goType, want := range map[string]int{
		"string":    -1, // variable length
		"int64":     8,
		"float64":   8,
		"bool":      1,
		"time.Time": 8,
	} {
		col, ok := pgTypes[goType]
		if !ok {
			t.Fatalf("%s has no Postgres type", goType)
		}
		if want > 0 && col.width != want {
			t.Errorf("%s: width %d, want %d", goType, col.width, want)
		}
		if col.name == "" {
			t.Errorf("%s: empty Postgres type name", goType)
		}
	}
}

func TestDDLQuotesReservedNames(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "order", GoType: "string", Params: map[string]string{}},
		{Name: "user", GoType: "string", Params: map[string]string{}},
	}}
	got := DDL("t", s, "/tmp/t.pgbin", true)
	if !strings.Contains(got, `"order"`) || !strings.Contains(got, `"user"`) {
		t.Errorf("reserved words were not quoted:\n%s", got)
	}
}
