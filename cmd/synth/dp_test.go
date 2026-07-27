package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// --dp noises a numeric column: the values change but stay numeric.
func TestMaskDPNoisesNumericColumn(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	out := filepath.Join(dir, "out.csv")
	os.WriteFile(in, []byte("id,salary\n1,50000\n2,60000\n3,70000\n"), 0o600)

	run(t, bin, "mask", "-i", in, "-o", out, "--key", "k", "--dp", "salary:1.0:10000")

	body, _ := os.ReadFile(out)
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	changed := false
	for _, l := range lines[1:] {
		cols := strings.Split(l, ",")
		if _, err := strconv.ParseFloat(cols[1], 64); err != nil {
			t.Errorf("noised salary %q is not numeric", cols[1])
		}
		if cols[1] != "50000" && cols[1] != "60000" && cols[1] != "70000" {
			changed = true
		}
	}
	if !changed {
		t.Error("no salary was noised")
	}
}

// A malformed --dp spec is a usage error.
func TestMaskDPMalformedErrors(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	os.WriteFile(in, []byte("id,salary\n1,50000\n"), 0o600)
	cmd := exec.Command(bin, "mask", "-i", in, "-o", filepath.Join(dir, "o.csv"),
		"--key", "k", "--dp", "salary:oops")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for a malformed --dp spec")
	}
}

// DP on a non-numeric column fails rather than passing the value through.
func TestMaskDPNonNumericErrors(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	os.WriteFile(in, []byte("name\nAlice\n"), 0o600)
	cmd := exec.Command(bin, "mask", "-i", in, "-o", filepath.Join(dir, "o.csv"),
		"--key", "k", "--dp", "name:1.0:10")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for DP on a non-numeric column")
	}
}
