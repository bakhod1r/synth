package ddlfe

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth/schema"
)

func parse(t *testing.T, sql string) []*Table {
	t.Helper()
	got, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse(%q): %v", sql, err)
	}
	return got
}

func TestParseColumnTypes(t *testing.T) {
	tables := parse(t, `
		CREATE TABLE users (
			id           UUID PRIMARY KEY,
			email        VARCHAR(255) NOT NULL UNIQUE,
			age          INTEGER,
			balance      NUMERIC(12,2),
			is_active    BOOLEAN DEFAULT TRUE,
			created_at   TIMESTAMP,
			metadata     JSONB
		);`)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	tb := tables[0]
	s := tb.Schema
	if tb.Name != "users" {
		t.Errorf("table name = %q", tb.Name)
	}
	want := map[string]schema.Kind{
		"id":        schema.KindUUID,
		"email":     schema.KindEmail,
		"age":       schema.KindInt,
		"balance":   schema.KindFloat,
		"is_active": schema.KindBool,
	}
	for _, f := range s.Fields {
		if k, ok := want[f.Name]; ok && f.Kind != k {
			t.Errorf("%s: kind = %q, want %q", f.Name, f.Kind, k)
		}
	}
}

// A primary key and a unique constraint change what the generator must
// guarantee, so losing them in the parse is worse than losing a type.
func TestPrimaryKeyAndUnique(t *testing.T) {
	s := parse(t, `CREATE TABLE t (id UUID PRIMARY KEY, email TEXT UNIQUE, name TEXT);`)[0].Schema
	// A primary key is unique by definition, so the parser sets both. Asserting
	// otherwise would be asserting a lie about SQL.
	for _, tc := range []struct {
		col        string
		pk, unique bool
	}{
		{"id", true, true},
		{"email", false, true},
		{"name", false, false},
	} {
		f := s.FieldByName(tc.col)
		if f == nil {
			t.Fatalf("no column %q", tc.col)
		}
		if f.PK != tc.pk {
			t.Errorf("%s: PK = %v, want %v", tc.col, f.PK, tc.pk)
		}
		if f.Unique != tc.unique {
			t.Errorf("%s: Unique = %v, want %v", tc.col, f.Unique, tc.unique)
		}
	}
}

// A table-level PRIMARY KEY (id) is as common as an inline one, and dropping it
// would silently produce duplicate keys.
func TestTableLevelPrimaryKey(t *testing.T) {
	s := parse(t, `CREATE TABLE t (id UUID NOT NULL, name TEXT, PRIMARY KEY (id));`)[0].Schema
	if f := s.FieldByName("id"); f == nil || !f.PK {
		t.Fatal("a table-level PRIMARY KEY was not applied")
	}
}

func TestMultipleTables(t *testing.T) {
	tables := parse(t, `
		CREATE TABLE users (id UUID PRIMARY KEY);
		CREATE TABLE orders (id UUID PRIMARY KEY, user_id UUID);
	`)
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}
	if tables[0].Name != "users" || tables[1].Name != "orders" {
		t.Fatalf("got %q and %q", tables[0].Name, tables[1].Name)
	}
}

// Real DDL arrives with schema prefixes, quoting and mixed case.
func TestQuotedAndQualifiedNames(t *testing.T) {
	for _, sql := range []string{
		`CREATE TABLE public.users (id UUID);`,
		`CREATE TABLE "users" ("id" UUID);`,
		`create table if not exists users (id uuid);`,
		"CREATE TABLE `users` (`id` UUID);",
		`CREATE TABLE [users] ([id] UUID);`,
	} {
		tables, err := Parse(sql)
		if err != nil {
			t.Errorf("%s: %v", sql, err)
			continue
		}
		if len(tables) != 1 {
			t.Errorf("%s: got %d tables", sql, len(tables))
			continue
		}
		if tables[0].Name != "users" {
			t.Errorf("%s: name = %q, want users", sql, tables[0].Name)
		}
	}
}

// A column name suggests a type when the SQL type does not. TEXT is TEXT, but a
// column called email holding TEXT should still produce an email.
func TestNameInfluencesKind(t *testing.T) {
	s := parse(t, `CREATE TABLE t (email TEXT, phone TEXT, city TEXT, note TEXT);`)[0].Schema
	for _, tc := range []struct {
		col  string
		kind schema.Kind
	}{
		{"email", schema.KindEmail},
		{"phone", schema.KindPhone},
		{"city", schema.KindCity},
	} {
		if f := s.FieldByName(tc.col); f == nil || f.Kind != tc.kind {
			got := schema.Kind("")
			if f != nil {
				got = f.Kind
			}
			t.Errorf("%s: kind = %q, want %q", tc.col, got, tc.kind)
		}
	}
}

// Input that is not DDL must be refused, not turned into an empty schema that
// generates zero columns and looks like it worked.
func TestNonDDLProducesNoTables(t *testing.T) {
	for _, sql := range []string{
		"", "   ", "SELECT * FROM users;", "not sql at all", "CREATE INDEX i ON t (a);",
	} {
		tables, err := Parse(sql)
		if err == nil && len(tables) > 0 {
			t.Errorf("%q produced %d tables", sql, len(tables))
		}
	}
}

// A malformed statement must not panic. DDL is often pasted by hand.
func TestMalformedInputDoesNotPanic(t *testing.T) {
	for _, sql := range []string{
		"CREATE TABLE (", "CREATE TABLE t (", "CREATE TABLE t (a INT,,,);",
		"CREATE TABLE t ()", "CREATE TABLE t (a VARCHAR());",
		"CREATE TABLE t (a NUMERIC(,));", strings.Repeat("CREATE TABLE t (a INT);", 200),
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panicked: %v", sql, r)
				}
			}()
			Parse(sql)
		}()
	}
}
