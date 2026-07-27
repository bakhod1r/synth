package gen

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

// derive and axis are dependency edges: the referenced field must be generated
// before the field that reads it, exactly as from and match already are.
func TestTopoOrdersDeriveBeforeDependent(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "income", Kind: schema.KindFloat, Params: map[string]string{"derive": "age"}},
		{Name: "age", Kind: schema.KindInt, Params: map[string]string{}},
	}}
	order, err := topoOrder(s)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for rank, i := range order {
		pos[s.Fields[i].Name] = rank
	}
	if pos["age"] > pos["income"] {
		t.Errorf("age (%d) should come before income (%d)", pos["age"], pos["income"])
	}
}

func TestTopoOrdersAxisBeforeSeries(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "cpu", Kind: schema.KindFloat, Params: map[string]string{"axis": "ts"}},
		{Name: "ts", Kind: schema.KindTime, Params: map[string]string{}},
	}}
	order, err := topoOrder(s)
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != 1 {
		t.Errorf("ts should be generated first, order=%v", order)
	}
}

func TestTopoDeriveCycleErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "a", Kind: schema.KindFloat, Params: map[string]string{"derive": "b"}},
		{Name: "b", Kind: schema.KindFloat, Params: map[string]string{"derive": "a"}},
	}}
	_, err := topoOrder(s)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected a cycle error, got %v", err)
	}
}

func TestTopoDeriveUnknownTargetErrors(t *testing.T) {
	s := &schema.Schema{Fields: []schema.Field{
		{Name: "income", Kind: schema.KindFloat, Params: map[string]string{"derive": "nope"}},
	}}
	_, err := topoOrder(s)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("expected an unknown-field error, got %v", err)
	}
}
