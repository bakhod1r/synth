package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Source is one upstream dataset.
//
// Every field is required for a reason. A source with no licence cannot ship;
// one with no checksum cannot be reproduced; one with no retrieval date cannot
// be audited later when somebody asks where a value came from.
type Source struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Licence string `yaml:"licence"`
	// Attribution is the credit line the licence requires. CC BY demands one;
	// public domain does not.
	Attribution string `yaml:"attribution,omitempty"`
	SHA256      string `yaml:"sha256"`
	Retrieved   string `yaml:"retrieved"` // YYYY-MM-DD
	// Note explains what is taken from the source and why, so a reviewer can
	// judge the curation without re-reading the importer.
	Note string `yaml:"note,omitempty"`
}

// allowedLicences is an allow-list, not a deny-list.
//
// A licence nobody has reviewed must fail closed. The alternative — listing
// what to reject — means every licence invented after this file was written is
// silently accepted, and the one that matters is always the one not on the list.
//
// Share-alike is excluded on purpose. ODbL in particular propagates to derived
// databases, so importing OpenStreetMap would place an obligation on everyone
// who imports Synth. That is not ours to impose.
var allowedLicences = map[string]string{
	"CC0-1.0":       "no attribution required",
	"public-domain": "no attribution required",
	"CC-BY-4.0":     "attribution required",
	"CC-BY-3.0":     "attribution required",
	"Unicode-3.0":   "attribution required",
	"MIT":           "attribution required",
	"BSD-3-Clause":  "attribution required",
	"Apache-2.0":    "attribution required",
	"ODC-PDDL-1.0":  "no attribution required",
}

// requiresAttribution reports whether a licence obliges us to credit the source.
func requiresAttribution(licence string) bool {
	return allowedLicences[licence] == "attribution required"
}

type manifest struct {
	Sources []Source `yaml:"sources"`
}

// LoadManifest reads and validates the source list.
func LoadManifest(path string) ([]Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadManifestBytes(data)
}

// LoadManifestBytes validates a manifest already in memory.
//
// Validation is strict and happens once, here, rather than at each use. A
// dataset that reaches a generated file without a recorded licence is a problem
// discovered by a lawyer rather than by a build.
func LoadManifestBytes(data []byte) ([]Source, error) {
	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if len(m.Sources) == 0 {
		return nil, fmt.Errorf("manifest: no sources")
	}

	seen := map[string]bool{}
	for i, s := range m.Sources {
		where := s.Name
		if where == "" {
			where = fmt.Sprintf("source %d", i)
		}
		if s.Name == "" {
			return nil, fmt.Errorf("%s: no name", where)
		}
		if seen[s.Name] {
			return nil, fmt.Errorf("%s: duplicate name", where)
		}
		seen[s.Name] = true

		if s.URL == "" {
			return nil, fmt.Errorf("%s: no url", where)
		}
		if s.Licence == "" {
			return nil, fmt.Errorf("%s: no licence — a dataset we cannot "+
				"redistribute must not ship, and no licence means we cannot", where)
		}
		if _, ok := allowedLicences[s.Licence]; !ok {
			return nil, fmt.Errorf("%s: licence %q is not on the allow-list. "+
				"Share-alike licences are excluded on purpose: they would place "+
				"an obligation on everyone who imports Synth. Allowed: %s",
				where, s.Licence, strings.Join(licenceNames(), ", "))
		}
		if requiresAttribution(s.Licence) && strings.TrimSpace(s.Attribution) == "" {
			return nil, fmt.Errorf("%s: licence %s requires attribution, and "+
				"none is recorded", where, s.Licence)
		}
		if s.SHA256 == "" {
			return nil, fmt.Errorf("%s: no sha256 — without one an import "+
				"cannot be reproduced", where)
		}
		if s.Retrieved != "" {
			if _, err := time.Parse("2006-01-02", s.Retrieved); err != nil {
				return nil, fmt.Errorf("%s: retrieved %q is not YYYY-MM-DD", where, s.Retrieved)
			}
		}
	}
	return m.Sources, nil
}

func licenceNames() []string {
	out := make([]string, 0, len(allowedLicences))
	for k := range allowedLicences {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Fetch downloads the source, or reads it from the cache, and verifies its
// checksum before returning a single byte of it.
//
// A mismatch stops the import. Upstream files change — a government agency
// reissues a release, a project renames a column — and regenerating different
// data without noticing is the failure this exists to prevent. The error
// reports the hash that was found, so updating the manifest after a deliberate
// upgrade is one copy-paste.
func (s Source) Fetch(cacheDir string) ([]byte, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(cacheDir, s.Name+cacheExt(s.URL))

	if data, err := os.ReadFile(path); err == nil {
		if sum := checksum(data); sum == s.SHA256 {
			return data, nil
		}
		// A stale cache entry is not an error worth stopping for: the file is
		// re-fetched and checked again below.
	}

	resp, err := http.Get(s.URL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s returned %s", s.Name, s.URL, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.Name, err)
	}

	if sum := checksum(data); sum != s.SHA256 {
		return nil, fmt.Errorf("%s: checksum mismatch — the upstream file has "+
			"changed.\n  manifest: %s\n  actual:   %s\n"+
			"If the change is expected, update sha256 and retrieved in sources.yaml.",
			s.Name, s.SHA256, sum)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return data, nil
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// cacheExt keeps the cached file's extension, so an archive is still
// recognisable on disk while debugging an importer.
func cacheExt(url string) string {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		url = url[:i]
	}
	ext := filepath.Ext(url)
	if len(ext) > 8 {
		return ""
	}
	return ext
}

// WriteNotice renders the attribution file.
//
// It is generated rather than maintained by hand for the same reason the data
// is: a NOTICE that drifts from the manifest is worse than none, because it
// asserts something false about what is in the release.
func WriteNotice(w io.Writer, sources []Source) error {
	sorted := append([]Source(nil), sources...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	fmt.Fprint(w, `Synth includes data derived from the open datasets listed below.

This file is generated by cmd/synthdata from cmd/synthdata/sources.yaml.
Do not edit it by hand: a NOTICE that has drifted from the manifest asserts
something false about what is in the release.

Only licences permitting redistribution are used. Share-alike licences are
excluded on purpose — ODbL propagates to derived databases, and importing one
would place an obligation on everyone who imports Synth.

`)
	for _, s := range sorted {
		fmt.Fprintf(w, "%s\n", s.Name)
		fmt.Fprintf(w, "  Source:    %s\n", s.URL)
		fmt.Fprintf(w, "  Licence:   %s\n", s.Licence)
		if s.Attribution != "" {
			fmt.Fprintf(w, "  Credit:    %s\n", s.Attribution)
		}
		if s.Retrieved != "" {
			fmt.Fprintf(w, "  Retrieved: %s\n", s.Retrieved)
		}
		fmt.Fprintf(w, "  SHA-256:   %s\n", s.SHA256)
		if s.Note != "" {
			fmt.Fprintf(w, "  Note:      %s\n", s.Note)
		}
		fmt.Fprintln(w)
	}
	return nil
}

// writeFile plants a file in the cache directory. It lives beside the cache
// logic rather than in the test, so a change to the cache layout has one place
// to update.
func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}
