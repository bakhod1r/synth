package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A unique quasi-identifier tuple fails k-anonymity, so verify exits 1.
func TestVerifyKAnonymityExitsOne(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	data := filepath.Join(dir, "d.csv")
	os.WriteFile(data, []byte("age,zip\n30,10001\n30,10001\n99,55555\n"), 0o600)

	cmd := exec.Command(bin, "verify", "-i", data, "--k", "2", "--qi", "age,zip")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected exit 1 on a k-anonymity violation")
	} else if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v", err)
	}
}

// A satisfied dataset passes.
func TestVerifyKAnonymitySatisfied(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	data := filepath.Join(dir, "d.csv")
	os.WriteFile(data, []byte("age,zip\n30,10001\n30,10001\n40,20002\n40,20002\n"), 0o600)
	run(t, bin, "verify", "-i", data, "--k", "2", "--qi", "age,zip") // run fails on non-zero
}

// --k without --qi is a usage error.
func TestVerifyKWithoutQIErrors(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	data := filepath.Join(dir, "d.csv")
	os.WriteFile(data, []byte("age\n30\n"), 0o600)
	cmd := exec.Command(bin, "verify", "-i", data, "--k", "2")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for --k without --qi")
	}
}
