package constraint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const csvSample = "id,name\n1,Ann\n2,Bo\n"
const jsonlSample = "{\"id\":1,\"name\":\"Ann\"}\n{\"id\":2,\"name\":\"Bo\"}\n"

func TestReadSampleFormats(t *testing.T) {
	for _, tc := range []struct{ format, data string }{
		{"csv", csvSample},
		{"", csvSample}, // empty means CSV
		{".csv", csvSample},
		{"jsonl", jsonlSample},
		{"ndjson", jsonlSample},
		{".jsonl", jsonlSample},
	} {
		rows, err := ReadSample(strings.NewReader(tc.data), tc.format, 0)
		if err != nil {
			t.Fatalf("format %q: %v", tc.format, err)
		}
		if len(rows) != 2 {
			t.Fatalf("format %q: got %d rows, want 2", tc.format, len(rows))
		}
		if rows[0]["name"] != "Ann" {
			t.Fatalf("format %q: got %v", tc.format, rows[0])
		}
	}
}

func TestReadSampleRejectsAnUnknownFormat(t *testing.T) {
	if _, err := ReadSample(strings.NewReader(csvSample), "parquet", 0); err == nil {
		t.Fatal("an unknown format was accepted")
	}
}

func TestReadSampleHonoursMax(t *testing.T) {
	data := "id\n" + strings.Repeat("1\n", 100)
	rows, err := ReadSample(strings.NewReader(data), "csv", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want the 5 asked for", len(rows))
	}
}

// LoadSample and ReadSample must agree, or the file path and the reader path
// drift and only one of them gets fixed when something is wrong.
func TestLoadSampleMatchesReadSample(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ name, data string }{
		{"a.csv", csvSample},
		{"b.jsonl", jsonlSample},
		{"c.dat", csvSample}, // an unknown extension must still read as CSV
	} {
		path := filepath.Join(dir, tc.name)
		if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := LoadSample(path, 0)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		want, err := ReadSample(strings.NewReader(tc.data), formatOf(path), 0)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got) != len(want) || got[0]["name"] != want[0]["name"] {
			t.Fatalf("%s: LoadSample and ReadSample disagree: %v vs %v", tc.name, got[0], want[0])
		}
	}
}
