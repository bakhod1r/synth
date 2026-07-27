package main_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// End to end: a cascade run emits child deletes before the parent delete, and
// never a child pointing at an already-deleted parent.
func TestCDCCascadeCLI(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	parent := filepath.Join(dir, "orders.yaml")
	child := filepath.Join(dir, "items.yaml")
	os.WriteFile(parent, []byte(
		"name: orders\nfields:\n  id: {kind: uuid, pk: true}\n  total: {kind: amount}\n"), 0o600)
	os.WriteFile(child, []byte(
		"name: items\nfields:\n  id: {kind: uuid, pk: true}\n  order_id: {kind: uuid}\n  sku: {kind: word}\n"), 0o600)

	cmd := exec.Command(bin, "cdc", "-s", parent, "--child", child,
		"--child-fk", "order_id", "-n", "60", "--delete-rate", "0.4")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	type ev struct {
		Op     string         `json:"op"`
		Before map[string]any `json:"before"`
		After  map[string]any `json:"after"`
		Source struct {
			Table string `json:"table"`
			LSN   int64  `json:"lsn"`
		} `json:"source"`
	}
	deletedParents := map[any]bool{}
	var lastLSN int64
	sawCascade := false
	sc := bufio.NewScanner(&stdout)
	for sc.Scan() {
		var e ev
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		if e.Source.LSN <= lastLSN {
			t.Fatalf("LSN not increasing: %d after %d", e.Source.LSN, lastLSN)
		}
		lastLSN = e.Source.LSN
		if e.Op == "d" && e.Source.Table == "items" {
			if deletedParents[e.Before["order_id"]] {
				t.Fatal("child deleted after its parent was gone")
			}
		}
		if e.Op == "d" && e.Source.Table == "orders" {
			deletedParents[e.Before["id"]] = true
			sawCascade = true
		}
	}
	if !sawCascade {
		t.Fatal("no parent delete in the stream")
	}
}

// --child without --child-fk is a usage error.
func TestCDCCascadeNeedsChildFK(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	parent := filepath.Join(dir, "p.yaml")
	child := filepath.Join(dir, "c.yaml")
	os.WriteFile(parent, []byte("name: p\nfields:\n  id: {kind: uuid, pk: true}\n"), 0o600)
	os.WriteFile(child, []byte("name: c\nfields:\n  id: {kind: uuid, pk: true}\n  pid: {kind: uuid}\n"), 0o600)
	cmd := exec.Command(bin, "cdc", "-s", parent, "--child", child, "-n", "5")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for --child without --child-fk")
	} else if !strings.Contains(stderr.String(), "child-fk") {
		t.Errorf("error should mention --child-fk, got: %s", stderr.String())
	}
}
