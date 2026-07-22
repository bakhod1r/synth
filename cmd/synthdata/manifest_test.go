package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A source with no licence must be rejected. Shipping data we cannot
// redistribute is worse than shipping less data — the second is a smaller
// dataset, the first is somebody else's problem becoming ours.
func TestManifestRejectsUnlicensedSource(t *testing.T) {
	_, err := LoadManifestBytes([]byte(`
sources:
  - name: mystery
    url: https://example.com/data.csv
    sha256: abc
`))
	if err == nil {
		t.Fatal("a source with no licence was accepted")
	}
	if !strings.Contains(err.Error(), "licence") {
		t.Fatalf("the error does not name the problem: %v", err)
	}
}

// The allow-list must hold in both directions: nothing unreviewed gets in, and
// nothing already reviewed gets locked out by a typo in the check.
func TestLicenceAllowList(t *testing.T) {
	const tmpl = "sources:\n  - name: x\n    url: u\n    sha256: a\n    attribution: someone\n    licence: "

	for _, bad := range []string{
		"ODbL-1.0", "GPL-3.0", "AGPL-3.0", "CC-BY-SA-4.0", "CC-BY-NC-4.0",
		"proprietary", "unknown", "",
	} {
		if _, err := LoadManifestBytes([]byte(tmpl + bad + "\n")); err == nil {
			t.Errorf("licence %q was accepted", bad)
		}
	}
	for _, ok := range []string{
		"CC0-1.0", "CC-BY-4.0", "Unicode-3.0", "public-domain", "MIT", "Apache-2.0",
	} {
		if _, err := LoadManifestBytes([]byte(tmpl + ok + "\n")); err != nil {
			t.Errorf("licence %q was rejected: %v", ok, err)
		}
	}
}

// The rejection has to explain itself. Someone adding a source will hit this,
// and "not allowed" sends them to read the code instead of the reasoning.
func TestLicenceRejectionExplainsItself(t *testing.T) {
	_, err := LoadManifestBytes([]byte(
		"sources:\n  - name: osm\n    url: u\n    sha256: a\n    licence: ODbL-1.0\n"))
	if err == nil {
		t.Fatal("ODbL was accepted")
	}
	for _, want := range []string{"Share-alike", "allow-list", "CC0-1.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

// A CC BY source with no attribution line is a licence violation waiting to
// happen, and the manifest is the only place it can be caught before release.
func TestAttributionRequiredWhereTheLicenceDemandsIt(t *testing.T) {
	_, err := LoadManifestBytes([]byte(
		"sources:\n  - name: x\n    url: u\n    sha256: a\n    licence: CC-BY-4.0\n"))
	if err == nil || !strings.Contains(err.Error(), "attribution") {
		t.Fatalf("CC BY without attribution was accepted: %v", err)
	}
	// Public domain does not require one, and demanding it anyway would be
	// friction with nothing behind it.
	if _, err := LoadManifestBytes([]byte(
		"sources:\n  - name: x\n    url: u\n    sha256: a\n    licence: public-domain\n")); err != nil {
		t.Fatalf("public domain was made to carry an attribution: %v", err)
	}
}

// Without a checksum an import cannot be reproduced, which is the difference
// between a build and a download.
func TestChecksumRequired(t *testing.T) {
	_, err := LoadManifestBytes([]byte(
		"sources:\n  - name: x\n    url: u\n    licence: CC0-1.0\n"))
	if err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("a source with no checksum was accepted: %v", err)
	}
}

func TestDuplicateNamesRejected(t *testing.T) {
	_, err := LoadManifestBytes([]byte(`
sources:
  - {name: x, url: u, sha256: a, licence: CC0-1.0}
  - {name: x, url: v, sha256: b, licence: CC0-1.0}
`))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("two sources with one name were accepted: %v", err)
	}
}

func TestRetrievedDateMustParse(t *testing.T) {
	_, err := LoadManifestBytes([]byte(
		"sources:\n  - name: x\n    url: u\n    sha256: a\n    licence: CC0-1.0\n    retrieved: last Tuesday\n"))
	if err == nil {
		t.Fatal("an unparseable retrieval date was accepted")
	}
}

// A checksum mismatch means the upstream file changed under us. The import must
// stop rather than quietly regenerate different data — a dataset that changes
// without anyone deciding to change it is the failure this guards against.
func TestFetchRejectsChangedContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "different content")
	}))
	defer srv.Close()

	s := Source{
		Name: "x", URL: srv.URL, Licence: "CC0-1.0",
		SHA256: strings.Repeat("0", 64),
	}
	_, err := s.Fetch(t.TempDir())
	if err == nil {
		t.Fatal("a changed file was accepted")
	}
	// The error must carry the hash that was found, or updating the manifest
	// after a deliberate upgrade means computing it by hand.
	if !strings.Contains(err.Error(), checksum([]byte("different content"))) {
		t.Fatalf("the error does not report the actual hash: %v", err)
	}
}

func TestFetchAcceptsMatchingContent(t *testing.T) {
	const body = "id,name\n1,Ann\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body)
	}))
	defer srv.Close()

	s := Source{Name: "x", URL: srv.URL, Licence: "CC0-1.0", SHA256: checksum([]byte(body))}
	got, err := s.Fetch(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("got %q", got)
	}
}

// The second fetch must come from disk. An importer re-downloading a hundred
// megabytes on every run is one nobody uses.
func TestFetchCaches(t *testing.T) {
	const body = "cached"
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		io.WriteString(w, body)
	}))
	defer srv.Close()

	s := Source{Name: "x", URL: srv.URL, Licence: "CC0-1.0", SHA256: checksum([]byte(body))}
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if _, err := s.Fetch(dir); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Fatalf("fetched %d times, want 1", hits)
	}
}

// A cache entry that no longer matches must be replaced rather than trusted.
func TestStaleCacheIsRefetched(t *testing.T) {
	const body = "fresh"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	s := Source{Name: "x", URL: srv.URL, Licence: "CC0-1.0", SHA256: checksum([]byte(body))}
	if err := writeFile(dir, "x", "stale bytes"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Fetch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("the stale cache entry was used: %q", got)
	}
}

// NOTICE is generated, so every credit the licences require reaches it without
// anyone remembering to add one.
func TestNoticeCarriesEveryAttribution(t *testing.T) {
	var buf bytes.Buffer
	err := WriteNotice(&buf, []Source{
		{Name: "geonames", URL: "https://geonames.org", Licence: "CC-BY-4.0",
			Attribution: "GeoNames, https://www.geonames.org/, CC BY 4.0",
			SHA256:      "abc", Retrieved: "2026-07-22"},
		{Name: "cldr", URL: "https://cldr.unicode.org", Licence: "Unicode-3.0",
			Attribution: "Unicode CLDR, Unicode-3.0", SHA256: "def"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"GeoNames", "CC BY 4.0", "Unicode CLDR", "geonames", "cldr",
		"https://geonames.org", "2026-07-22", "abc",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("NOTICE is missing %q:\n%s", want, out)
		}
	}
}

// Sorted output means a re-run produces the same file, so a diff shows what
// actually changed rather than that a map was walked in a new order.
func TestNoticeIsStable(t *testing.T) {
	sources := []Source{
		{Name: "zeta", URL: "u", Licence: "CC0-1.0", SHA256: "a"},
		{Name: "alpha", URL: "u", Licence: "CC0-1.0", SHA256: "b"},
	}
	var first, second bytes.Buffer
	WriteNotice(&first, sources)
	WriteNotice(&second, []Source{sources[1], sources[0]})
	if first.String() != second.String() {
		t.Fatal("NOTICE depends on the order the sources were listed in")
	}
	if strings.Index(first.String(), "alpha") > strings.Index(first.String(), "zeta") {
		t.Fatal("NOTICE is not sorted by name")
	}
}

// The manifest that ships must itself be valid, or the checks above test
// nothing that reaches a release.
func TestShippedManifestIsValid(t *testing.T) {
	sources, err := LoadManifest("sources.yaml")
	if err != nil {
		t.Fatalf("sources.yaml does not validate: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("sources.yaml lists nothing")
	}
	for _, s := range sources {
		if s.Note == "" {
			t.Errorf("%s: no note saying what is taken from it and why", s.Name)
		}
	}
}
