package schema

import "testing"

func TestFieldByName(t *testing.T) {
	s := &Schema{Fields: []Field{
		{Name: "id", Kind: KindUUID},
		{Name: "email", Kind: KindEmail},
	}}
	f := s.FieldByName("email")
	if f == nil || f.Kind != KindEmail {
		t.Fatalf("FieldByName(email) = %+v", f)
	}
	// Returned pointer aliases the slice element (mutation visible).
	f.Unique = true
	if !s.Fields[1].Unique {
		t.Fatal("returned field is not aliased to slice element")
	}
	if got := s.FieldByName("missing"); got != nil {
		t.Fatalf("FieldByName(missing) = %+v, want nil", got)
	}
}
