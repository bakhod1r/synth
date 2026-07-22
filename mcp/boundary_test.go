package mcp

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The MCP server must not import a package that reads the filesystem, opens a
// connection or runs a program.
//
// It runs with the user's own permissions on behalf of a model that may be
// reading attacker-controlled text, so a later change that adds os.ReadFile to
// a tool would hand that model a file-reading primitive. In review such a change
// looks like a small convenience; here it fails the build.
//
// parser.ParseDir does not recurse, so cmd/synth-mcp — which legitimately
// imports os to exit with a status — is outside this scan. That is the right
// line: the boundary applies to the tool handlers, while the command has to be
// able to report a startup failure.
func TestNoFilesystemOrNetworkImports(t *testing.T) {
	banned := map[string]string{
		"os":            "the server must not touch the filesystem",
		"os/exec":       "the server must not run a program",
		"io/ioutil":     "the server must not touch the filesystem",
		"path/filepath": "a path argument is the thing this boundary forbids",
		"net":           "the server must not open a connection",
		"net/http":      "the server must not open a connection",
		"net/url":       "the server must not fetch anything",
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			for _, imp := range file.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("%s: unreadable import %s", name, imp.Path.Value)
				}
				if why, bad := banned[path]; bad {
					t.Errorf("%s imports %q: %s", name, path, why)
				}
			}
		}
	}
}

// Every tool must be registered, described, and callable. The description is
// the only thing a model has to decide whether to call a tool at all.
func TestEveryToolIsRegisteredAndDescribed(t *testing.T) {
	want := []string{
		"generate", "list_types", "list_presets",
		"verify", "profile", "mask", "snapshot",
	}
	tools := listTools(t)
	for _, name := range want {
		tool, ok := tools[name]
		if !ok {
			t.Errorf("tool %q is not registered", name)
			continue
		}
		if len(tool.Description) < 40 {
			t.Errorf("tool %q has a description too short to be useful: %q", name, tool.Description)
		}
	}
	if len(tools) != len(want) {
		t.Errorf("got %d tools, want %d — a new tool needs a line in this test", len(tools), len(want))
	}
}

// Every tool that takes data must say it reads no files. That sentence is what
// stops a model from trying a path first and reporting the failure as ours.
func TestDataToolsSayTheyReadNoFiles(t *testing.T) {
	tools := listTools(t)
	for _, name := range []string{"generate", "verify", "profile", "mask", "snapshot"} {
		tool, ok := tools[name]
		if !ok {
			t.Fatalf("tool %q is not registered", name)
		}
		if !strings.Contains(strings.ToLower(tool.Description), "no file") {
			t.Errorf("tool %q does not say it reads no files: %q", name, tool.Description)
		}
	}
}
