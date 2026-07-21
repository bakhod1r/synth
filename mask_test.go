package synth_test

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

const dump = `id,full_name,email,phone,card,city,order_total,notes
1,John Smith,john.smith@acme.com,+1 415 555 0101,4539578763621486,Boston,120.50,regular customer
2,Mary Jones,mary@corp.io,+1 415 555 0102,5425233430109903,Boston,45.00,contact bob@corp.io
3,John Smith,john.smith@acme.com,+1 415 555 0101,4539578763621486,Austin,99.99,vip
`

func writeDump(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "dump.csv")
	out := filepath.Join(dir, "masked.csv")
	if err := os.WriteFile(in, []byte(dump), 0o644); err != nil {
		t.Fatal(err)
	}
	return in, out
}

func readCSV(t *testing.T, path string) ([]string, [][]string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return recs[0], recs[1:]
}

func TestMaskRemovesPII(t *testing.T) {
	in, out := writeDump(t)
	m := synth.NewMasker("secret-key", "en_US")
	rep, err := m.File(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rows != 3 {
		t.Fatalf("rows %d", rep.Rows)
	}

	header, rows := readCSV(t, out)
	col := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		t.Fatalf("no column %q", name)
		return -1
	}
	joined := strings.Join(header, ",")
	for _, r := range rows {
		joined += "\n" + strings.Join(r, ",")
	}
	// No original personal value may survive anywhere in the output.
	for _, secret := range []string{
		"John Smith", "Mary Jones", "john.smith@acme.com", "mary@corp.io",
		"4539578763621486", "5425233430109903", "555 0101", "bob@corp.io",
	} {
		if strings.Contains(joined, secret) {
			t.Fatalf("PII leaked into output: %q", secret)
		}
	}
	// Formats must be preserved so the data still exercises the same code.
	for _, r := range rows {
		if !strings.Contains(r[col("email")], "@") {
			t.Fatalf("email lost its format: %q", r[col("email")])
		}
		if len(r[col("card")]) != 16 {
			t.Fatalf("card length changed: %q", r[col("card")])
		}
	}
	// Non-personal columns are untouched.
	for i, r := range rows {
		want := []string{"120.50", "45.00", "99.99"}[i]
		if r[col("order_total")] != want {
			t.Fatalf("non-PII column changed: %q != %q", r[col("order_total")], want)
		}
	}
}

// The same input value must map to the same replacement, so joins survive.
func TestMaskIsConsistent(t *testing.T) {
	in, out := writeDump(t)
	m := synth.NewMasker("secret-key", "en_US")
	if _, err := m.File(in, out); err != nil {
		t.Fatal(err)
	}
	header, rows := readCSV(t, out)
	idx := 0
	for i, h := range header {
		if h == "email" {
			idx = i
		}
	}
	// Rows 1 and 3 shared an email in the source; they must still match.
	if rows[0][idx] != rows[2][idx] {
		t.Fatalf("identical inputs produced different outputs: %q vs %q", rows[0][idx], rows[2][idx])
	}
	// And differ from the other customer's.
	if rows[0][idx] == rows[1][idx] {
		t.Fatal("different inputs collapsed to the same output")
	}
}

// A different key must produce different (unlinkable) output.
func TestMaskKeyChangesOutput(t *testing.T) {
	in, out1 := writeDump(t)
	out2 := out1 + ".2"
	if _, err := synth.NewMasker("key-a", "en_US").File(in, out1); err != nil {
		t.Fatal(err)
	}
	if _, err := synth.NewMasker("key-b", "en_US").File(in, out2); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(out1)
	b, _ := os.ReadFile(out2)
	if string(a) == string(b) {
		t.Fatal("different keys produced identical output")
	}
}

func TestMaskExplicitRules(t *testing.T) {
	in, out := writeDump(t)
	m := synth.NewMasker("k", "en_US")
	m.Rule(synth.MaskRule{Column: "notes", Strategy: synth.MaskRedact})
	m.Rule(synth.MaskRule{Column: "city", Strategy: synth.MaskDrop})
	if _, err := m.File(in, out); err != nil {
		t.Fatal(err)
	}
	header, rows := readCSV(t, out)
	col := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		return -1
	}
	if rows[0][col("city")] != "" {
		t.Fatalf("Drop did not blank the column: %q", rows[0][col("city")])
	}
	notes := rows[0][col("notes")]
	if strings.ContainsAny(notes, "abcdefghijklmnopqrstuvwxyz") {
		t.Fatalf("Redact left readable text: %q", notes)
	}
	if !strings.Contains(notes, " ") {
		t.Fatalf("Redact did not preserve shape: %q", notes)
	}
}
