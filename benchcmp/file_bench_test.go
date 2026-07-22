package benchcmp_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bakhodir/synth"
	gofaker "github.com/go-faker/faker/v4"
	jaswdr "github.com/jaswdr/faker/v2"
)

// Writing a fixture to a file is what these libraries are actually used for,
// and it is where the comparison changes shape: go-faker and jaswdr generate
// values and stop, so anyone using them writes the file by hand. These
// benchmarks include that hand-written loop, because leaving it out would
// compare Synth's whole job against half of theirs.
//
// The file goes to b.TempDir(), which the test framework removes. Nothing here
// writes outside it.

const fileRows = 10_000

// BenchmarkSynth_WriteCSV is generate-then-write with the library's own writer.
func BenchmarkSynth_WriteCSV(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		users := synth.Make[BenchUser](fileRows, synth.WithSeed(1))
		if err := synth.WriteCSV(filepath.Join(dir, "s.csv"), users); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSynth_WriteJSONL(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		users := synth.Make[BenchUser](fileRows, synth.WithSeed(1))
		if err := synth.WriteJSONL(filepath.Join(dir, "s.jsonl"), users); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSynth_WriteSQL(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		users := synth.Make[BenchUser](fileRows, synth.WithSeed(1))
		if err := synth.WriteSQL(filepath.Join(dir, "s.sql"), "users", users); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSynth_StreamCSV never holds the rows in memory. At ten thousand rows
// the difference is small; the point is that this one does not grow with the
// row count, so the same code writes ten million.
func BenchmarkSynth_StreamCSV(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	for b.Loop() {
		s := synth.Stream[BenchUser](fileRows, synth.WithSeed(1))
		if err := s.ToCSV(filepath.Join(dir, "stream.csv")); err != nil {
			b.Fatal(err)
		}
	}
}

// go-faker has no file output, so this is what its users write themselves.
func BenchmarkGoFaker_WriteCSV(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	for b.Loop() {
		f, err := os.Create(filepath.Join(dir, "gf.csv"))
		if err != nil {
			b.Fatal(err)
		}
		w := csv.NewWriter(f)
		if err := w.Write([]string{"Name", "Email", "Phone", "City"}); err != nil {
			b.Fatal(err)
		}
		for i := 0; i < fileRows; i++ {
			var u goFakerUser
			if err := gofaker.FakeData(&u); err != nil {
				b.Fatal(err)
			}
			if err := w.Write([]string{u.Name, u.Email, u.Phone, u.City}); err != nil {
				b.Fatal(err)
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}

func BenchmarkJaswdr_WriteCSV(b *testing.B) {
	dir := b.TempDir()
	fk := jaswdr.New()
	b.ReportAllocs()
	for b.Loop() {
		f, err := os.Create(filepath.Join(dir, "jw.csv"))
		if err != nil {
			b.Fatal(err)
		}
		w := csv.NewWriter(f)
		if err := w.Write([]string{"Name", "Email", "Phone", "City"}); err != nil {
			b.Fatal(err)
		}
		for i := 0; i < fileRows; i++ {
			if err := w.Write([]string{
				fk.Person().Name(), fk.Internet().Email(),
				fk.Phone().Number(), fk.Address().City(),
			}); err != nil {
				b.Fatal(err)
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}

func BenchmarkJaswdr_WriteJSONL(b *testing.B) {
	dir := b.TempDir()
	fk := jaswdr.New()
	b.ReportAllocs()
	for b.Loop() {
		f, err := os.Create(filepath.Join(dir, "jw.jsonl"))
		if err != nil {
			b.Fatal(err)
		}
		enc := json.NewEncoder(f)
		for i := 0; i < fileRows; i++ {
			if err := enc.Encode(BenchUser{
				Name:  fk.Person().Name(),
				Email: fk.Internet().Email(),
				Phone: fk.Phone().Number(),
				City:  fk.Address().City(),
			}); err != nil {
				b.Fatal(err)
			}
		}
		f.Close()
	}
}

// How the row count affects the write, so the cost is known to be linear rather
// than assumed to be.
func BenchmarkSynth_WriteCSVScaling(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			dir := b.TempDir()
			b.ReportAllocs()
			for b.Loop() {
				users := synth.Make[BenchUser](n, synth.WithSeed(1))
				if err := synth.WriteCSV(filepath.Join(dir, "s.csv"), users); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The written file has to be correct, not merely fast. A benchmark that
// produced an empty or truncated file would look excellent.
func TestBenchmarkedWritersProduceCompleteFiles(t *testing.T) {
	dir := t.TempDir()
	users := synth.Make[BenchUser](100, synth.WithSeed(1))

	csvPath := filepath.Join(dir, "s.csv")
	if err := synth.WriteCSV(csvPath, users); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 101 {
		t.Fatalf("the CSV has %d lines, want a header and 100 rows", len(records))
	}
	for i, rec := range records[1:] {
		for j, field := range rec {
			if field == "" {
				t.Fatalf("row %d column %d is empty", i, j)
			}
		}
	}
}
