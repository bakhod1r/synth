package schemafe

import "testing"

func TestJSONSchemaMaxLengthBecomesMaxlen(t *testing.T) {
	doc := []byte(`{
		"type": "object",
		"properties": {
			"nickname": {"type": "string", "maxLength": 12},
			"bio": {"type": "string"},
			"age": {"type": "integer", "maximum": 120}
		}
	}`)
	tbl, err := ParseJSONSchema(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"nickname": "12", "bio": "", "age": ""}
	for name, w := range want {
		f := tbl.Schema.FieldByName(name)
		if f == nil {
			t.Fatalf("field %q missing", name)
		}
		if got := f.Params["maxlen"]; got != w {
			t.Errorf("%s: maxlen = %q, want %q", name, got, w)
		}
	}
}
