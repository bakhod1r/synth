package mask

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestCSVMasking(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "name", Strategy: Fake, Kind: schema.KindName})
	m.Rule(Rule{Column: "note", Strategy: Keep})
	in := "id,name,note\n1,Alice,hello\n2,Bob,hi\n"
	var out strings.Builder
	rep, err := m.CSV(strings.NewReader(in), &out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rows != 2 {
		t.Fatalf("rows = %d", rep.Rows)
	}
	if rep.Masked["name"] != 2 {
		t.Fatalf("name masked = %d", rep.Masked["name"])
	}
	if strings.Contains(out.String(), "Alice") {
		t.Fatal("name leaked")
	}
	// "note" untouched.
	found := false
	for _, c := range rep.Untouched {
		if c == "note" {
			found = true
		}
	}
	if !found {
		t.Fatalf("note should be untouched: %v", rep.Untouched)
	}
}

// failWriter returns an error on the Nth write (1-indexed) and after.
type failWriter struct {
	n, at int
}

func (f *failWriter) Write(p []byte) (int, error) {
	f.n++
	if f.n >= f.at {
		return 0, errWrite
	}
	return len(p), nil
}

var errWrite = &writeErr{}

type writeErr struct{}

func (*writeErr) Error() string { return "write failed" }

func TestCSVHeaderWriteError(t *testing.T) {
	m := New("k", "en")
	// csv.Writer buffers via bufio (4096); a header larger than that overflows
	// and the failing writer makes cw.Write return an error on the header.
	big := strings.Repeat("x", 5000)
	if _, err := m.CSV(strings.NewReader(big+"\n1\n"), &failWriter{at: 1}); err == nil {
		t.Fatal("header write error should surface")
	}
}

func TestCSVRowWriteError(t *testing.T) {
	m := New("k", "en")
	// Small header stays buffered; a huge row field overflows the buffer so the
	// row cw.Write hits the failing underlying writer.
	bigRow := strings.Repeat("y", 5000)
	if _, err := m.CSV(strings.NewReader("a\n"+bigRow+"\n"), &failWriter{at: 1}); err == nil {
		t.Fatal("row write path error should surface")
	}
}

func TestCSVMalformedRow(t *testing.T) {
	m := New("k", "en")
	// Unclosed quote makes the row read fail.
	if _, err := m.CSV(strings.NewReader("a,b\n\"unclosed,2\n"), &strings.Builder{}); err == nil {
		t.Fatal("malformed row should error")
	}
}

func TestJSONLEncodeError(t *testing.T) {
	m := New("k", "en")
	if _, err := m.JSONL(strings.NewReader(`{"a":"b"}`+"\n"), &failWriter{at: 1}); err == nil {
		t.Fatal("encode error should surface")
	}
}

func TestCSVBadHeader(t *testing.T) {
	m := New("k", "en")
	if _, err := m.CSV(strings.NewReader(""), &strings.Builder{}); err == nil {
		t.Fatal("empty input should error on header read")
	}
}

func TestCSVRaggedRowExtraFieldsSkipped(t *testing.T) {
	m := New("k", "en")
	in := "a,b\n1,2,3\n" // extra field beyond header is skipped, not an error
	var out strings.Builder
	if _, err := m.CSV(strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
}

func TestCSVDPErrorSurfaces(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "amt", Strategy: DP, Epsilon: 1, Sensitivity: 1})
	in := "amt\nnotanumber\n"
	if _, err := m.CSV(strings.NewReader(in), &strings.Builder{}); err == nil {
		t.Fatal("DP on non-numeric should surface an error")
	}
}

func TestJSONLMasking(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "amt", Strategy: DP, Epsilon: 1, Sensitivity: 5})
	in := `{"name":"Alice","amt":100,"age":30}` + "\n" +
		`{"name":"Bob","amt":200,"age":25}` + "\n"
	var out strings.Builder
	rep, err := m.JSONL(strings.NewReader(in), &out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rows != 2 {
		t.Fatalf("rows = %d", rep.Rows)
	}
	if rep.Masked["amt"] != 2 {
		t.Fatalf("amt masked = %d", rep.Masked["amt"])
	}
	if strings.Contains(out.String(), "Alice") {
		t.Fatal("name leaked in JSONL")
	}
}

func TestJSONLBadLine(t *testing.T) {
	m := New("k", "en")
	if _, err := m.JSONL(strings.NewReader("{not json}\n"), &strings.Builder{}); err == nil {
		t.Fatal("malformed JSON should error")
	}
}

func TestJSONLDPNonNumericErrors(t *testing.T) {
	m := New("k", "en")
	m.Rule(Rule{Column: "amt", Strategy: DP, Epsilon: 1, Sensitivity: 1})
	// amt is a string that is not numeric under a DP rule.
	if _, err := m.JSONL(strings.NewReader(`{"amt":"xyz"}`+"\n"), &strings.Builder{}); err == nil {
		t.Fatal("expected DP non-numeric error")
	}
}

func TestNumberFromString(t *testing.T) {
	if v := numberFromString("3.5"); v != 3.5 {
		t.Fatalf("numberFromString(3.5) = %v", v)
	}
	if v := numberFromString("nope"); v != "nope" {
		t.Fatalf("non-numeric should pass through, got %v", v)
	}
}

func TestFileDispatch(t *testing.T) {
	dir := t.TempDir()
	csvIn := filepath.Join(dir, "in.csv")
	os.WriteFile(csvIn, []byte("name\nAlice\n"), 0o644)
	m := New("k", "en")
	m.Rule(Rule{Column: "name", Strategy: Fake, Kind: schema.KindName})
	if _, err := m.File(csvIn, filepath.Join(dir, "out.csv")); err != nil {
		t.Fatal(err)
	}

	jsonlIn := filepath.Join(dir, "in.jsonl")
	os.WriteFile(jsonlIn, []byte(`{"name":"Bob"}`+"\n"), 0o644)
	if _, err := m.File(jsonlIn, filepath.Join(dir, "out.jsonl")); err != nil {
		t.Fatal(err)
	}

	if _, err := m.File(filepath.Join(dir, "missing.csv"), filepath.Join(dir, "o")); err == nil {
		t.Fatal("missing input should error")
	}
	// Unwritable output path.
	if _, err := m.File(csvIn, filepath.Join(dir, "nodir", "o.csv")); err == nil {
		t.Fatal("bad output path should error")
	}
}
