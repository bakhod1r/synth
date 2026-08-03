package infer

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestKindByName(t *testing.T) {
	k, matched := Kind("Full_Name", "string")
	if k != schema.KindName || !matched {
		t.Fatalf("Full_Name = %v,%v", k, matched)
	}
}

func TestKindByGoType(t *testing.T) {
	cases := []struct {
		goType string
		want   schema.Kind
	}{
		{"time.Time", schema.KindTime},
		{"uuid.UUID", schema.KindUUID},
		{"bool", schema.KindBool},
		{"int", schema.KindInt},
		{"uint64", schema.KindInt},
		{"float32", schema.KindFloat},
		{"float64", schema.KindFloat},
		{"string", schema.KindLorem},
		{"struct{}", schema.KindUnknown},
	}
	for _, c := range cases {
		// Use a name that is not in the synonym table so the type path runs.
		k, matched := Kind("xyzzy_unmatched", c.goType)
		if k != c.want || matched {
			t.Fatalf("goType %q = %v,%v want %v,false", c.goType, k, matched, c.want)
		}
	}
}

func TestAlias(t *testing.T) {
	Alias("ismi", schema.KindName)
	if k, ok := Kind("ismi", "string"); k != schema.KindName || !ok {
		t.Fatalf("alias ismi = %v,%v", k, ok)
	}
}

func TestLinkDependenciesEmailFromName(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Name", Kind: schema.KindName},
		{Name: "Email", Kind: schema.KindEmail},
	}}
	LinkDependencies(s)
	if s.FieldByName("Email").From != "Name" {
		t.Fatalf("email From = %q", s.FieldByName("Email").From)
	}
}

func TestLinkDependenciesCardAndAirport(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Card", Kind: schema.KindCard},
		{Name: "Brand", Kind: schema.KindCardBrand},
		{Name: "CVV", Kind: schema.KindCVV},
		{Name: "Apt", Kind: schema.KindAirport},
		{Name: "AptName", Kind: schema.KindAirportName},
	}}
	LinkDependencies(s)
	if s.FieldByName("Brand").From != "Card" {
		t.Fatal("card brand not linked")
	}
	if s.FieldByName("CVV").From != "Card" {
		t.Fatal("cvv not linked")
	}
	if s.FieldByName("AptName").From != "Apt" {
		t.Fatal("airport name not linked")
	}
}

func TestLinkDependenciesTimeOrdering(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "CreatedAt", Kind: schema.KindTime},
		{Name: "UpdatedAt", Kind: schema.KindTime},
		{Name: "DeletedAt", Kind: schema.KindTime},
		{Name: "Score", Kind: schema.KindInt},
	}}
	LinkDependencies(s)
	if s.FieldByName("UpdatedAt").From != "CreatedAt" {
		t.Fatalf("updated From = %q", s.FieldByName("UpdatedAt").From)
	}
	if s.FieldByName("DeletedAt").From != "CreatedAt" {
		t.Fatalf("deleted From = %q", s.FieldByName("DeletedAt").From)
	}
}

func TestLinkDependenciesNoAnchors(t *testing.T) {
	// No name, card, airport, or created field: nothing links, no panic.
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "Email", Kind: schema.KindEmail},
		{Name: "UpdatedAt", Kind: schema.KindTime},
	}}
	LinkDependencies(s)
	if s.FieldByName("Email").From != "" {
		t.Fatal("email should not link without a name field")
	}
}
