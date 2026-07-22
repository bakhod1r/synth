package synth_test

import (
	"testing"

	"github.com/bakhod1r/synth"
)

const yamlSpec = `
name: users
count: 500
locale: uz_UZ
seed: 42
fields:
  id:       { kind: uuid, pk: true }
  fullname: { kind: name }
  email:    { kind: email, from: fullname }
  phone:    { kind: phone }
  status:   { kind: enum, choices: [active, inactive], weights: [0.9, 0.1] }
  amount:   { kind: amount, min: 100, max: 5000 }
`

func TestYAMLFrontend(t *testing.T) {
	spec, err := synth.YAMLBytes([]byte(yamlSpec))
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name() != "users" || spec.Count() != 500 {
		t.Fatalf("spec meta wrong: %s %d", spec.Name(), spec.Count())
	}
	if got := spec.Columns(); len(got) != 6 || got[0] != "id" {
		t.Fatalf("columns wrong: %v", got)
	}
	recs, err := spec.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 500 {
		t.Fatalf("want 500, got %d", len(recs))
	}
	active := 0
	ids := map[any]bool{}
	for _, r := range recs {
		if ids[r["id"]] {
			t.Fatal("duplicate pk from yaml unique")
		}
		ids[r["id"]] = true
		amt := r["amount"].(float64)
		if amt < 100 || amt > 5000 {
			t.Fatalf("amount out of range: %v", amt)
		}
		if r["status"] == "active" {
			active++
		}
	}
	share := float64(active) / float64(len(recs))
	if share < 0.85 || share > 0.95 {
		t.Fatalf("weighted status share off: %.3f", share)
	}
}
