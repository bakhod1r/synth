package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// capture runs fn with stdout redirected, returning what it printed. The list
// and import commands communicate through stdout, so that is what is checked.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		done <- b.String()
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

func sum(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func TestListSourcesShowsEveryColumn(t *testing.T) {
	sources := []Source{
		{Name: "beta", Licence: "CC0-1.0", SHA256: "x", Retrieved: "2026-01-02", Note: "a note"},
		{Name: "alpha", Licence: "MIT", SHA256: "y"},
	}
	out := capture(t, func() { listSources(sources) })
	for _, want := range []string{"NAME", "beta", "alpha", "CC0-1.0", "MIT", "2026-01-02", "a note"} {
		if !strings.Contains(out, want) {
			t.Errorf("listing does not mention %q:\n%s", want, out)
		}
	}
	// A missing retrieved date prints as a dash rather than as a blank column.
	if !strings.Contains(out, "—") {
		t.Errorf("a missing date was not shown as a dash:\n%s", out)
	}
	if !strings.Contains(out, "2 sources") {
		t.Errorf("the count is missing:\n%s", out)
	}
}

func TestOrDash(t *testing.T) {
	if got := orDash(""); got != "—" {
		t.Errorf("orDash(\"\") = %q", got)
	}
	if got := orDash("2026-01-01"); got != "2026-01-01" {
		t.Errorf("orDash passed through the wrong value: %q", got)
	}
}

func TestSelectSources(t *testing.T) {
	all := []Source{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	got, err := selectSources(all, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("no names should select everything, got %d", len(got))
	}

	got, err = selectSources(all, []string{"c", "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "c" || got[1].Name != "a" {
		t.Fatalf("selection did not follow the order asked for: %+v", got)
	}

	if _, err := selectSources(all, []string{"nope"}); err == nil {
		t.Error("an unknown name should be an error")
	} else if !strings.Contains(err.Error(), "synthdata list") {
		t.Errorf("the error should point at the list command: %v", err)
	}
}

// verify is the cheap pre-flight: it must pass when the upstream file matches
// and fail, per source, when it does not.
func TestVerifyReportsPerSource(t *testing.T) {
	const good = "col\n1\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "moved") {
			w.Write([]byte("something else entirely"))
			return
		}
		w.Write([]byte(good))
	}))
	defer srv.Close()

	cache := t.TempDir()
	sources := []Source{
		{Name: "stable", URL: srv.URL + "/stable.csv", Licence: "CC0-1.0", SHA256: sum(good)},
		{Name: "moved", URL: srv.URL + "/moved.csv", Licence: "CC0-1.0", SHA256: sum(good)},
	}

	out := capture(t, func() {
		if err := verify(sources[:1], cache, nil); err != nil {
			t.Errorf("a matching source failed verification: %v", err)
		}
	})
	if !strings.Contains(out, "ok   stable") {
		t.Errorf("output = %q", out)
	}

	err := verify(sources, cache, nil)
	if err == nil {
		t.Fatal("a changed upstream file passed verification")
	}
	if !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("err = %v, want a count of failures", err)
	}

	if err := verify(sources, cache, []string{"nope"}); err == nil {
		t.Error("verifying an unknown source should error")
	}
}

// Until a source has an importer, import must say so rather than report
// success for work it did not do.
func TestImportSkipsSourcesWithoutAnImporter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	// The NOTICE is redirected to a scratch file: changing the working
	// directory instead would leak into every test that runs after this one.
	notice := redirectNotice(t)

	sources := []Source{{
		Name: "unimported", URL: srv.URL, Licence: "CC0-1.0", SHA256: sum("data"),
	}}
	out := capture(t, func() {
		if err := runImport(sources, t.TempDir(), nil); err != nil {
			t.Errorf("import failed: %v", err)
		}
	})
	if !strings.Contains(out, "skip unimported") {
		t.Fatalf("output = %q, want a skip line", out)
	}
	// NOTICE is rewritten on every import so it cannot fall behind the data.
	written, err := os.ReadFile(notice)
	if err != nil {
		t.Fatalf("NOTICE was not written: %v", err)
	}
	if !strings.Contains(string(written), "unimported") {
		t.Fatalf("NOTICE does not list the source:\n%s", written)
	}
}

func TestImportRunsAnImporter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("payload"))
	}))
	defer srv.Close()

	redirectNotice(t)

	var seen []byte
	importers["fixture"] = func(s Source, data []byte) (string, error) {
		seen = data
		return "locale/fixture_data.go", nil
	}
	defer delete(importers, "fixture")

	sources := []Source{{
		Name: "fixture", URL: srv.URL, Licence: "CC0-1.0", SHA256: sum("payload"),
	}}
	out := capture(t, func() {
		if err := runImport(sources, t.TempDir(), nil); err != nil {
			t.Errorf("import failed: %v", err)
		}
	})
	if string(seen) != "payload" {
		t.Fatalf("the importer received %q", seen)
	}
	if !strings.Contains(out, "ok   fixture") {
		t.Fatalf("output = %q", out)
	}
}

func TestImportSurfacesFetchAndImporterErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	importers["broken"] = func(Source, []byte) (string, error) { return "", os.ErrInvalid }
	defer delete(importers, "broken")

	bad := []Source{{Name: "broken", URL: srv.URL, Licence: "CC0-1.0", SHA256: sum("x")}}
	if err := runImport(bad, t.TempDir(), nil); err == nil {
		t.Fatal("a 404 upstream should fail the import")
	}
	if err := runImport(bad, t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("an unknown source name should fail the import")
	}
}

func TestDefaultCacheIsAPath(t *testing.T) {
	if defaultCache() == "" {
		t.Fatal("defaultCache returned nothing")
	}
}

func TestCacheExtKeepsAUsefulExtension(t *testing.T) {
	cases := map[string]string{
		"https://x.test/data.csv":            ".csv",
		"https://x.test/data.zip?token=abc":  ".zip",
		"https://x.test/data.json#frag":      ".json",
		"https://x.test/data":                "",
		"https://x.test/data.averylongthing": "",
	}
	for url, want := range cases {
		if got := cacheExt(url); got != want {
			t.Errorf("cacheExt(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestLoadManifestReportsAMissingFile(t *testing.T) {
	if _, err := LoadManifest(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("a missing manifest should error")
	}
}

func TestManifestRejectsANamelessSource(t *testing.T) {
	_, err := LoadManifestBytes([]byte("sources:\n  - url: u\n    licence: MIT\n    sha256: a\n"))
	if err == nil || !strings.Contains(err.Error(), "no name") {
		t.Fatalf("err = %v, want a missing-name error", err)
	}
}

func TestManifestRejectsASourceWithNoURL(t *testing.T) {
	_, err := LoadManifestBytes([]byte("sources:\n  - name: n\n    licence: MIT\n    sha256: a\n"))
	if err == nil || !strings.Contains(err.Error(), "no url") {
		t.Fatalf("err = %v, want a missing-url error", err)
	}
}

func TestManifestRejectsAnEmptyDocument(t *testing.T) {
	if _, err := LoadManifestBytes([]byte("sources: []\n")); err == nil {
		t.Error("an empty source list should error")
	}
	if _, err := LoadManifestBytes([]byte("::: not yaml :::")); err == nil {
		t.Error("invalid YAML should error")
	}
}

func TestFetchReportsAnUnreachableHost(t *testing.T) {
	s := Source{Name: "down", URL: "http://127.0.0.1:1/never", Licence: "MIT", SHA256: "x"}
	if _, err := s.Fetch(t.TempDir()); err == nil {
		t.Fatal("an unreachable host should error")
	}
}

// redirectNotice points writeNoticeFile at a scratch file for the duration of
// one test, and returns its path. The real NOTICE is never touched.
func redirectNotice(t *testing.T) string {
	t.Helper()
	prev := noticePath
	path := filepath.Join(t.TempDir(), "NOTICE")
	noticePath = path
	t.Cleanup(func() { noticePath = prev })
	return path
}
