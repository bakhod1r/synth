package schemafe

import (
	"strings"
	"testing"
)

// orderedKeys recovers the document order that encoding/json's map decoding
// throws away. It is the only thing keeping a generated table's columns in the
// order the schema declares them, so its failure modes matter: every one of
// them falls back to map order rather than losing a column.
func TestOrderedKeysRejectsMalformedDocuments(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string // substring of the expected error
	}{
		{"not an object", `[1,2]`, "cannot unmarshal"},
		{"field missing", `{"other":{}}`, `no "properties" object`},
		{"field is not an object", `{"properties":[1]}`, "expected comma after array element"},
		{"truncated object", `{"properties":{"a":`, "unexpected end"},
		{"value is malformed", `{"properties":{"a":tru}}`, "invalid character"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keys, err := orderedKeys([]byte(tc.doc), "properties")
			if err == nil {
				t.Fatalf("orderedKeys = %v, want an error", keys)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestOrderedKeysPreservesDocumentOrder(t *testing.T) {
	keys, err := orderedKeys([]byte(`{"properties":{"z":{},"a":{},"m":{}}}`), "properties")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(keys, ","); got != "z,a,m" {
		t.Errorf("keys = %q, want %q", got, "z,a,m")
	}
}

// encoding/json matches field names case-insensitively, so a document written
// with "Properties" still decodes — but the raw scan that recovers order looks
// for the exact key and finds nothing. The parse falls back to map order rather
// than failing: an unordered table beats no table.
func TestParseJSONSchemaFallsBackWhenOrderIsUnrecoverable(t *testing.T) {
	tbl, err := ParseJSONSchema([]byte(`{"title":"t","Properties":{"a":{"type":"string"}}}`))
	if err != nil {
		t.Fatalf("ParseJSONSchema: %v", err)
	}
	if len(tbl.Order) != 1 || tbl.Order[0] != "a" {
		t.Fatalf("Order = %v, want the single property to survive the fallback", tbl.Order)
	}
}
