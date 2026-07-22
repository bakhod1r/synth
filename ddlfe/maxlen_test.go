package ddlfe

import "testing"

// A varchar(n) column carries a length the DDL author cared about. Dropping it
// means the generator can emit a value the source table would have rejected.
func TestVarcharLengthBecomesMaxlen(t *testing.T) {
	ddl := `CREATE TABLE users (
		id UUID PRIMARY KEY,
		nickname VARCHAR(12),
		code CHAR(3),
		bio TEXT,
		age INT
	);`
	tables, err := Parse(ddl)
	if err != nil {
		t.Fatal(err)
	}
	s := tables[0].Schema
	want := map[string]string{
		"nickname": "12",
		"code":     "3",
		"bio":      "", // unbounded
		"age":      "", // not a string type
		"id":       "",
	}
	for name, w := range want {
		f := s.FieldByName(name)
		if f == nil {
			t.Fatalf("field %q missing", name)
		}
		if got := f.Params["maxlen"]; got != w {
			t.Errorf("%s: maxlen = %q, want %q", name, got, w)
		}
	}
}

func TestVarcharLengthIgnoresPrecisionOnNumerics(t *testing.T) {
	tables, err := Parse(`CREATE TABLE t (amount NUMERIC(10,2));`)
	if err != nil {
		t.Fatal(err)
	}
	s := tables[0].Schema
	if got := s.FieldByName("amount").Params["maxlen"]; got != "" {
		t.Errorf("numeric precision leaked into maxlen: %q", got)
	}
}
