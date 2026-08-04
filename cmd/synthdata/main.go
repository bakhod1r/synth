// Command synthdata imports Synth's datasets from open sources.
//
// It is a developer tool, run by hand, whose output is committed Go source. The
// library itself never downloads anything: `go get github.com/bakhod1r/synth`
// works offline, and this module's HTTP and archive dependencies stay out of
// the library's graph entirely.
//
//	go run . list                  # sources, with licence and retrieval date
//	go run . import cldr geonames  # named sources, or all of them
//	go run . notice                # regenerate NOTICE
//	go run . verify                # check every checksum without importing
//
// # Why the rules are strict
//
// A generated dataset is only as trustworthy as its provenance. A value nobody
// can trace has to be assumed wrong, and a licence nobody recorded has to be
// assumed incompatible. So a source without a licence, a checksum, or — where
// the licence demands it — an attribution line, cannot enter the manifest at
// all. See manifest.go.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

// Both paths are variables rather than constants so a test can point them at
// scratch files. They are repository-relative, and a test that redirected them
// by changing the working directory would leave every other test — and the real
// NOTICE — at the mercy of the order they run in.
var (
	manifestPath = "sources.yaml"
	noticePath   = "../../NOTICE"
)

func main() { os.Exit(run(os.Args[1:])) }

// run is main without the exit, so the command's behaviour — including the
// exit codes, which are the part a script depends on — can be tested in
// process. It parses its own flag set rather than the global one so a second
// call does not see the first one's flags.
func run(argv []string) int {
	fs := flag.NewFlagSet("synthdata", flag.ContinueOnError)
	cache := fs.String("cache", defaultCache(), "where downloads are kept")
	fs.Usage = func() { usage(fs) }
	if err := fs.Parse(argv); err != nil {
		return 2
	}

	args := fs.Args()
	if len(args) == 0 {
		usage(fs)
		return 2
	}

	sources, err := LoadManifest(manifestPath)
	if err != nil {
		return fail(err)
	}

	switch args[0] {
	case "list":
		listSources(sources)
	case "notice":
		if err := writeNoticeFile(sources); err != nil {
			return fail(err)
		}
		fmt.Println("wrote", noticePath)
	case "verify":
		if err := verify(sources, *cache, args[1:]); err != nil {
			return fail(err)
		}
	case "import":
		if err := runImport(sources, *cache, args[1:]); err != nil {
			return fail(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		usage(fs)
		return 2
	}
	return 0
}

func usage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `synthdata — import Synth's datasets from open sources

This is a developer tool. Its output is committed Go source; the library never
downloads anything.

Usage:
  synthdata list                 sources, with licence and retrieval date
  synthdata verify [name...]     check checksums without importing
  synthdata import [name...]     import named sources, or all
  synthdata notice               regenerate NOTICE

Flags:
`)
	fs.PrintDefaults()
}

func listSources(sources []Source) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tLICENCE\tRETRIEVED\tNOTE")
	for _, s := range sources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Licence, orDash(s.Retrieved), s.Note)
	}
	w.Flush()
	fmt.Printf("\n%d sources. Every one carries a licence, a checksum and, where\n", len(sources))
	fmt.Println("the licence requires it, an attribution line — see NOTICE.")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// verify downloads each source and checks its checksum, without running any
// importer. It is the cheap way to find out that an upstream file moved before
// spending an afternoon on why the generated data changed.
func verify(sources []Source, cache string, names []string) error {
	selected, err := selectSources(sources, names)
	if err != nil {
		return err
	}
	var failures int
	for _, s := range selected {
		if _, err := s.Fetch(cache); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s\n     %v\n", s.Name, err)
			failures++
			continue
		}
		fmt.Printf("ok   %s\n", s.Name)
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d sources failed", failures, len(selected))
	}
	return nil
}

// runImport fetches each source and hands it to its importer.
//
// The importers arrive one per task; until a source has one, the command says
// so rather than reporting success for work it did not do.
func runImport(sources []Source, cache string, names []string) error {
	selected, err := selectSources(sources, names)
	if err != nil {
		return err
	}
	for _, s := range selected {
		imp, ok := importers[s.Name]
		if !ok {
			fmt.Printf("skip %s — no importer yet\n", s.Name)
			continue
		}
		data, err := s.Fetch(cache)
		if err != nil {
			return err
		}
		written, err := imp(s, data)
		if err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		fmt.Printf("ok   %s → %s\n", s.Name, written)
	}
	// NOTICE is rewritten on every import, so it cannot fall behind the data it
	// describes.
	return writeNoticeFile(sources)
}

// importers maps a source name to the function that turns it into Go source.
// Each is added by its own task; an empty map means the skeleton is in place
// and no data has been imported yet.
var importers = map[string]func(Source, []byte) (string, error){}

func selectSources(all []Source, names []string) ([]Source, error) {
	if len(names) == 0 {
		return all, nil
	}
	byName := map[string]Source{}
	for _, s := range all {
		byName[s.Name] = s
	}
	out := make([]Source, 0, len(names))
	for _, n := range names {
		s, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("no source named %q — run `synthdata list`", n)
		}
		out = append(out, s)
	}
	return out, nil
}

func writeNoticeFile(sources []Source) error {
	f, err := os.Create(noticePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteNotice(f, sources)
}

func defaultCache() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "synthdata")
	}
	return ".cache"
}

// fail reports an error and returns the exit code that goes with it, so every
// failing branch of run reads the same way.
func fail(err error) int {
	fmt.Fprintln(os.Stderr, "synthdata:", err)
	return 1
}
