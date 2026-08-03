package ddlfe

import "testing"

func TestTableLevelUniqueConstraint(t *testing.T) {
	tables, err := Parse(`CREATE TABLE t (
		id INT PRIMARY KEY,
		email VARCHAR(255),
		UNIQUE (email)
	);`)
	if err != nil {
		t.Fatal(err)
	}
	f := tables[0].Schema.FieldByName("email")
	if f == nil || !f.Unique {
		t.Fatalf("table-level UNIQUE not applied: %+v", f)
	}
	// Integer PK gets the wide unique range.
	id := tables[0].Schema.FieldByName("id")
	if !id.PK || id.Params["max"] != "2000000000" {
		t.Fatalf("int PK params = %+v", id)
	}
}

func TestVarcharMaxIsNotMaxlen(t *testing.T) {
	tables, err := Parse(`CREATE TABLE t (body VARCHAR(max));`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tables[0].Schema.FieldByName("body").Params["maxlen"]; ok {
		t.Fatal("varchar(max) should not set a numeric maxlen")
	}
}

func TestSingleTokenColumnSkipped(t *testing.T) {
	// parseColumn directly: fewer than two tokens is not a column.
	if _, ok := parseColumn("lonely"); ok {
		t.Fatal("single-token should not parse as a column")
	}
	if _, ok := parseColumn("   "); ok {
		t.Fatal("blank should not parse as a column")
	}
	// A line whose first token is a constraint keyword is not a column.
	if _, ok := parseColumn("CHECK (age > 0)"); ok {
		t.Fatal("CHECK constraint should not parse as a column")
	}
	if _, ok := parseColumn("CONSTRAINT fk FOREIGN KEY (a) REFERENCES b(id)"); ok {
		t.Fatal("named constraint should not parse as a column")
	}
}
