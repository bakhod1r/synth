package synth

import "testing"

// RefValues fills a foreign-key field from values the caller already holds —
// the cross-run case, where the parent was generated in an earlier run and its
// keys were read back from a file.
func TestRefValuesFromSpec(t *testing.T) {
	spec := `
name: orders
count: 200
fields:
  id: {kind: uuid}
  user_id: {kind: int}
`
	y, err := YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	parents := []any{int64(10), int64(20), int64(30)}
	rows, err := y.GenerateN(200, RefValues("user_id", parents))
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[any]bool{int64(10): true, int64(20): true, int64(30): true}
	for i, r := range rows {
		if !allowed[r["user_id"]] {
			t.Fatalf("row %d: user_id %v (%T) not drawn from the parent set",
				i, r["user_id"], r["user_id"])
		}
	}
}

// A field named by RefValues that the spec does not have is a user error worth
// reporting, not a silent no-op that leaves the column randomly generated.
func TestRefValuesUnknownField(t *testing.T) {
	y, err := YAMLBytes([]byte("name: t\ncount: 1\nfields:\n  id: {kind: uuid}\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.GenerateN(1, RefValues("missing", []any{1}))
	if err == nil {
		t.Fatal("expected an error for a field the spec does not have")
	}
}
