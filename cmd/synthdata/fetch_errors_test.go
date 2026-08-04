package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fetch is the only thing standing between an upstream change and a silently
// different dataset, so each way it can fail has to reach the caller. None of
// these are exotic: a cache directory that is not a directory, a connection
// that dies mid-body, a cache entry shadowed by a directory of the same name.
func TestFetchReportsIOFailures(t *testing.T) {
	t.Run("cache directory cannot be created", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "a-file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		s := Source{Name: "s", URL: "http://127.0.0.1:0/x"}
		if _, err := s.Fetch(filepath.Join(file, "cache")); err == nil {
			t.Fatal("Fetch = nil error, want one: the cache path is under a file")
		}
	})

	t.Run("the response body is truncated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Promising more than is sent makes the client's read fail rather
			// than returning a short body as if it were whole.
			w.Header().Set("Content-Length", "1024")
			w.Write([]byte("short"))
		}))
		defer srv.Close()

		s := Source{Name: "s", URL: srv.URL + "/data.txt"}
		if _, err := s.Fetch(t.TempDir()); err == nil {
			t.Fatal("Fetch = nil error, want one for a truncated body")
		}
	})

	t.Run("the cache entry cannot be written", func(t *testing.T) {
		body := "payload"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(body))
		}))
		defer srv.Close()

		cache := t.TempDir()
		s := Source{Name: "s", URL: srv.URL + "/data.txt", SHA256: sum(body)}
		// A directory where the cached file belongs makes the write fail after
		// the download has already been checked.
		if err := os.Mkdir(filepath.Join(cache, "s.txt"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := s.Fetch(cache)
		if err == nil {
			t.Fatal("Fetch = nil error, want one when the cache entry cannot be written")
		}
		if !strings.Contains(err.Error(), "s.txt") {
			t.Errorf("error = %q, want it to name the path it could not write", err)
		}
	})
}

// Until a source has an importer the command says so and moves on. Once it has
// one, the importer's output is what gets reported — and its failure is the
// import's failure, named after the source so the log says which one.
func TestRunImportUsesRegisteredImporters(t *testing.T) {
	body := "payload"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	notice := filepath.Join(t.TempDir(), "NOTICE")
	oldNotice := noticePath
	noticePath = notice
	t.Cleanup(func() { noticePath = oldNotice })

	src := Source{Name: "importable", URL: srv.URL + "/data.txt", SHA256: sum(body), Licence: "CC0-1.0"}

	t.Run("success reports what was written", func(t *testing.T) {
		importers["importable"] = func(Source, []byte) (string, error) { return "locale/generated.go", nil }
		t.Cleanup(func() { delete(importers, "importable") })

		out := capture(t, func() {
			if err := runImport([]Source{src}, t.TempDir(), nil); err != nil {
				t.Errorf("runImport: %v", err)
			}
		})
		if !strings.Contains(out, "locale/generated.go") {
			t.Errorf("output = %q, want the path the importer reported", out)
		}
	})

	t.Run("a failing importer fails the import", func(t *testing.T) {
		importers["importable"] = func(Source, []byte) (string, error) {
			return "", os.ErrInvalid
		}
		t.Cleanup(func() { delete(importers, "importable") })

		var err error
		capture(t, func() { err = runImport([]Source{src}, t.TempDir(), nil) })
		if err == nil {
			t.Fatal("runImport = nil error, want the importer's failure")
		}
		if !strings.Contains(err.Error(), "importable") {
			t.Errorf("error = %q, want it to name the source", err)
		}
	})
}

// defaultCache falls back to a relative directory when the OS cannot name a
// cache location — an unset HOME, which is how a bare CI container arrives.
func TestDefaultCacheFallsBackWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if got := defaultCache(); got != ".cache" {
		t.Errorf("defaultCache = %q, want %q when the OS cannot name a cache directory", got, ".cache")
	}
}
