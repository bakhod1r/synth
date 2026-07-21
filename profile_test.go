package synth_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

// Build a "real" export, profile it, and check the synthetic data reproduces
// its shape: category frequencies, numeric range, and detected formats.
func TestProfileLearnsShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.csv")

	var b strings.Builder
	b.WriteString("id,email,age,status,country\n")
	for i := 0; i < 1000; i++ {
		status := "active"
		switch {
		case i%100 < 5: // 5% banned
			status = "banned"
		case i%100 < 20: // 15% inactive
			status = "inactive"
		}
		fmt.Fprintf(&b, "%08d-1111-4222-8333-444444444444,user%d@example.com,%d,%s,%s\n",
			i, i, 20+i%40, status, []string{"US", "DE", "UZ"}[i%3])
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := synth.Profile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.SampleRows() != 1000 {
		t.Fatalf("profiled %d rows, want 1000", p.SampleRows())
	}
	if got := p.Columns(); len(got) != 5 || got[0] != "id" {
		t.Fatalf("columns: %v", got)
	}

	rows, err := p.Generate(5000, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	countries := map[string]int{}
	for _, r := range rows {
		// Numeric range learned from the sample (20..59).
		age, ok := r["age"].(int)
		if !ok || age < 20 || age > 59 {
			t.Fatalf("age outside learned range: %v", r["age"])
		}
		// Email format detected from values, not the column name alone.
		if !strings.Contains(fmt.Sprint(r["email"]), "@") {
			t.Fatalf("email not generated: %v", r["email"])
		}
		counts[fmt.Sprint(r["status"])]++
		countries[fmt.Sprint(r["country"])]++
	}

	// Category frequencies must approximate the sample (80/15/5).
	active := float64(counts["active"]) / float64(len(rows))
	banned := float64(counts["banned"]) / float64(len(rows))
	if active < 0.75 || active > 0.85 {
		t.Fatalf("active share %.3f, want ~0.80", active)
	}
	if banned < 0.02 || banned > 0.08 {
		t.Fatalf("banned share %.3f, want ~0.05", banned)
	}
	if len(countries) != 3 {
		t.Fatalf("expected 3 learned countries, got %d", len(countries))
	}
}

// High-cardinality identifier columns must not be reproduced verbatim as an
// enum — they get a generated type instead (no leaking of real values).
func TestProfileDoesNotEchoIdentifiers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ids.csv")
	var b strings.Builder
	b.WriteString("id\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "%08d-1111-4222-8333-444444444444\n", i)
	}
	os.WriteFile(path, []byte(b.String()), 0o644)

	p, _ := synth.Profile(path)
	rows, err := p.Generate(100, synth.WithSeed(2))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if strings.HasPrefix(fmt.Sprint(r["id"]), "0000") {
			t.Fatalf("profiler echoed a real identifier: %v", r["id"])
		}
	}
}
