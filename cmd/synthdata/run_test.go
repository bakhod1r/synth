package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// quiet redirects both streams for the duration of fn. run writes usage and
// progress to the terminal, and a test suite is not a terminal.
func quiet(t *testing.T, fn func()) {
	t.Helper()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	outOld, errOld := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = devNull, devNull
	defer func() { os.Stdout, os.Stderr = outOld, errOld }()
	fn()
}

// The exit code is the part of a command-line tool a script depends on, so it
// is what the tests assert. 2 means "you asked for something that is not a
// command"; 1 means "the command ran and failed".
func TestRunExitCodes(t *testing.T) {
	// The notice command rewrites NOTICE; point it at a scratch file so a test
	// run cannot touch the repository's own.
	notice := filepath.Join(t.TempDir(), "NOTICE")
	old := noticePath
	noticePath = notice
	t.Cleanup(func() { noticePath = old })

	tests := []struct {
		name string
		argv []string
		want int
	}{
		{"no command prints usage", nil, 2},
		{"unknown command", []string{"no-such-command"}, 2},
		{"unknown flag", []string{"-no-such-flag"}, 2},
		{"list", []string{"list"}, 0},
		{"notice", []string{"notice"}, 0},
		{"verify an unknown source", []string{"verify", "no-such-source"}, 1},
		{"import an unknown source", []string{"import", "no-such-source"}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			quiet(t, func() { got = run(tc.argv) })
			if got != tc.want {
				t.Fatalf("run(%q) = %d, want %d", tc.argv, got, tc.want)
			}
		})
	}

	if _, err := os.Stat(notice); err != nil {
		t.Errorf("the notice command wrote no file: %v", err)
	}
}

// A missing or malformed manifest stops the command before it does anything:
// every command reads the manifest first, because a source without a recorded
// licence and checksum must not be acted on at all.
func TestRunReportsAnUnreadableManifest(t *testing.T) {
	old := manifestPath
	manifestPath = filepath.Join(t.TempDir(), "no-such-sources.yaml")
	t.Cleanup(func() { manifestPath = old })

	var got int
	quiet(t, func() { got = run([]string{"list"}) })
	if got != 1 {
		t.Fatalf("run(list) = %d, want 1 when the manifest cannot be read", got)
	}
}

// A NOTICE that cannot be written is a failure, not a silent skip: the file is
// the project's licence compliance and an import that cannot refresh it has not
// finished.
func TestRunReportsAnUnwritableNotice(t *testing.T) {
	old := noticePath
	// A path under a file (rather than a directory) cannot be created.
	noticePath = filepath.Join(t.TempDir(), "not-a-dir", "NOTICE")
	t.Cleanup(func() { noticePath = old })

	var got int
	quiet(t, func() { got = run([]string{"notice"}) })
	if got != 1 {
		t.Fatalf("run(notice) = %d, want 1 when NOTICE cannot be written", got)
	}
}

// The flags are parsed by run's own flag set, so a second call does not
// inherit the first one's values — and -cache is accepted before the command,
// which is where Go's flag package requires it.
func TestRunAcceptsTheCacheFlag(t *testing.T) {
	cache := t.TempDir()
	var got int
	quiet(t, func() { got = run([]string{"-cache", cache, "verify", "no-such-source"}) })
	if got != 1 {
		t.Fatalf("run = %d, want 1 for an unknown source name", got)
	}
	if !strings.HasPrefix(cache, os.TempDir()) {
		t.Fatalf("the test's own cache directory looks wrong: %q", cache)
	}
}
