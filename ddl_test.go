package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

const ddl = `
CREATE TABLE users (
    id          UUID PRIMARY KEY,
    full_name   VARCHAR(100) NOT NULL,
    email       VARCHAR(255) UNIQUE,
    age         INTEGER,
    balance     NUMERIC(12,2),
    is_active   BOOLEAN,
    created_at  TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS orders (
    id       BIGSERIAL PRIMARY KEY,
    user_id  UUID,
    total    DECIMAL(10,2)
);
`

func TestDDLFrontend(t *testing.T) {
	tables, err := synth.DDLBytes([]byte(ddl))
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("want 2 tables, got %d", len(tables))
	}
	users := tables[0]
	if users.Name() != "users" {
		t.Fatalf("table name %q", users.Name())
	}
	cols := users.Columns()
	if len(cols) != 7 || cols[0] != "id" || cols[2] != "email" {
		t.Fatalf("columns wrong: %v", cols)
	}

	rows, err := users.Generate(500, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	if err != nil {
		t.Fatal(err)
	}
	emails := map[string]bool{}
	for _, r := range rows {
		// email column inferred from name → contains "@"
		if !strings.Contains(r["email"].(string), "@") {
			t.Fatalf("email not generated: %v", r["email"])
		}
		if emails[r["email"].(string)] {
			t.Fatal("UNIQUE email violated")
		}
		emails[r["email"].(string)] = true
		// age is INTEGER
		if _, ok := r["age"].(int); !ok {
			t.Fatalf("age not int: %T", r["age"])
		}
		// is_active is BOOLEAN
		if _, ok := r["is_active"].(bool); !ok {
			t.Fatalf("is_active not bool: %T", r["is_active"])
		}
	}
}

// Integer serial PK from DDL must be unique.
func TestDDLIntPKUnique(t *testing.T) {
	tables, _ := synth.DDLBytes([]byte(ddl))
	orders := tables[1]
	rows, err := orders.Generate(5000, synth.WithSeed(2))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[any]bool{}
	for _, r := range rows {
		if seen[r["id"]] {
			t.Fatal("duplicate serial PK")
		}
		seen[r["id"]] = true
	}
}
