package main_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/synth/diff"
)

func writeDiffCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Identical files: exit 0, no differences.
func TestDiffIdenticalExitsZero(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	csv := "id,amount\n1,100\n2,200\n3,300\n"
	a := writeDiffCSV(t, dir, "a.csv", csv)
	b := writeDiffCSV(t, dir, "b.csv", csv)
	out := run(t, bin, "diff", a, b) // run() fails the test on non-zero exit
	_ = out
}

// A removed column is a structural break: exit 1.
func TestDiffColumnRemovedExitsOne(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	a := writeDiffCSV(t, dir, "a.csv", "id,legacy\n1,x\n2,y\n")
	b := writeDiffCSV(t, dir, "b.csv", "id\n1\n2\n")
	cmd := exec.Command(bin, "diff", a, b)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 on a removed column")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
}

// --format json emits parseable findings.
func TestDiffJSONOutput(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	a := writeDiffCSV(t, dir, "a.csv", "amount\n0\n100\n")
	b := writeDiffCSV(t, dir, "b.csv", "amount\n0\n200\n")
	cmd := exec.Command(bin, "diff", a, b, "--format", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Run() // may exit 0 (warnings only)
	var findings []diff.Finding
	if err := json.Unmarshal(stdout.Bytes(), &findings); err != nil {
		t.Fatalf("json output did not parse: %v\n%s", err, stdout.String())
	}
	if len(findings) == 0 {
		t.Error("expected at least one drift finding")
	}
}
