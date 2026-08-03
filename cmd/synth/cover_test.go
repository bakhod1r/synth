package main

import (
	"os"
	"path/filepath"
	"testing"
)

const coverSpec = `
name: users
count: 30
fields:
  id: { kind: uuid, pk: true }
  name: { kind: name }
  email: { kind: email, from: name }
  age: { kind: int, min: 18, max: 80 }
  amount: { kind: amount, min: 1, max: 1000, mask: partial }
  created_at: { kind: time }
  updated_at: { kind: time }
`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunGenFormats(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	for _, f := range []string{"csv", "jsonl", "sql"} {
		out := filepath.Join(dir, "out."+f)
		if err := runGen([]string{"-s", spec, "-o", out, "-f", f, "--seed", "1", "-l", "uz_UZ"}); err != nil {
			t.Fatalf("gen %s: %v", f, err)
		}
	}
	// Format inferred from extension, plus chaos and unmask.
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "x.jsonl"),
		"--seed", "2", "--chaos", "0.1", "--unmasked"}); err != nil {
		t.Fatalf("gen inferred: %v", err)
	}
	// From a preset.
	if err := runGen([]string{"--preset", "user", "-o", filepath.Join(dir, "p.csv"), "-n", "5"}); err != nil {
		t.Fatalf("gen preset: %v", err)
	}
	// Parquet: written to a real path, and the file carries the PAR1 magic.
	pq := filepath.Join(dir, "out.parquet")
	if err := runGen([]string{"-s", spec, "-o", pq, "--seed", "1", "-n", "20"}); err != nil {
		t.Fatalf("gen parquet: %v", err)
	}
	b, err := os.ReadFile(pq)
	if err != nil || len(b) < 8 || string(b[:4]) != "PAR1" || string(b[len(b)-4:]) != "PAR1" {
		t.Fatalf("parquet magic missing: err=%v len=%d", err, len(b))
	}
}

func TestRunGenErrors(t *testing.T) {
	if err := runGen([]string{"-o", "/tmp/x"}); err == nil {
		t.Fatal("gen without spec/preset should error")
	}
	if err := runGen([]string{"--preset", "nope"}); err == nil {
		t.Fatal("unknown preset should error")
	}
	if err := runGen([]string{"-s", "/nonexistent.yaml"}); err == nil {
		t.Fatal("missing spec should error")
	}
	// Parquet needs a real path; it cannot stream to stdout.
	if err := runGen([]string{"-f", "parquet", "--preset", "user"}); err == nil {
		t.Fatal("parquet to stdout should error")
	}
}

func TestRunGenAppend(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	out := filepath.Join(dir, "a.csv")
	if err := runGen([]string{"-s", spec, "-o", out, "--seed", "5", "-n", "3"}); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", out, "--seed", "5", "-n", "3", "--append"}); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestRunProfileMaskCDC(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	csv := filepath.Join(dir, "data.csv")
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}

	// profile
	prof := filepath.Join(dir, "prof.yaml")
	if err := runProfile([]string{"-i", csv, "-o", prof}); err != nil {
		t.Fatalf("profile: %v", err)
	}
	if err := runProfile(nil); err == nil {
		t.Fatal("profile without -i should error")
	}

	// mask with a DP rule
	masked := filepath.Join(dir, "masked.csv")
	if err := runMask([]string{"-i", csv, "-o", masked, "--key", "k", "--dp", "age:1.0:10"}); err != nil {
		t.Fatalf("mask: %v", err)
	}
	if err := runMask([]string{"-i", csv, "-o", masked}); err == nil {
		t.Fatal("mask without key should error")
	}
	if err := runMask([]string{"-i", csv, "-o", csv, "--key", "k"}); err == nil {
		t.Fatal("mask over input should error")
	}

	// cdc single-table
	cdcOut := filepath.Join(dir, "cdc.jsonl")
	if err := runCDC([]string{"-s", spec, "-o", cdcOut, "-n", "20", "--update-rate", "0.2", "--delete-rate", "0.1"}); err != nil {
		t.Fatalf("cdc: %v", err)
	}
	if err := runCDC(nil); err == nil {
		t.Fatal("cdc without spec should error")
	}
}

func TestRunVerifyAndSnapshotAndDiff(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	csv := filepath.Join(dir, "data.csv")
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}

	if err := runVerify([]string{"-i", csv}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := runVerify(nil); err == nil {
		t.Fatal("verify without -i should error")
	}

	// snapshot --at and --from/--to
	if err := runSnapshot([]string{"-s", spec, "--at", "2026-06-01", "-o", filepath.Join(dir, "snap.csv")}); err != nil {
		t.Fatalf("snapshot at: %v", err)
	}
	if err := runSnapshot([]string{"-s", spec, "--from", "2026-01-01", "--to", "2026-12-01",
		"-o", filepath.Join(dir, "snap2.jsonl")}); err != nil {
		t.Fatalf("snapshot range: %v", err)
	}
	if err := runSnapshot([]string{"-s", spec}); err == nil {
		t.Fatal("snapshot without at/range should error")
	}
	if err := runSnapshot([]string{"-s", spec, "--at", "x", "--from", "y"}); err == nil {
		t.Fatal("snapshot with both at and range should error")
	}

	// diff two CSVs
	csv2 := filepath.Join(dir, "data2.csv")
	if err := runGen([]string{"-s", spec, "-o", csv2, "-f", "csv", "--seed", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := runDiff([]string{csv, csv2}); err != nil {
		t.Fatalf("diff: %v", err)
	}
	if err := runDiff([]string{csv}); err == nil {
		t.Fatal("diff needs two paths")
	}
}

func TestRunVerifyWithRefAndParseRef(t *testing.T) {
	dir := t.TempDir()
	parent := writeFile(t, dir, "p.csv", "id\n1\n2\n3\n")
	child := writeFile(t, dir, "c.csv", "user_id\n1\n2\n")
	if err := runVerify([]string{"-i", child, "--ref", "user_id=" + parent + ":id"}); err != nil {
		t.Fatalf("verify with ref: %v", err)
	}
	// Malformed ref specs.
	if _, err := parseRef("noequals"); err == nil {
		t.Fatal("ref without = should error")
	}
	if _, err := parseRef("col=filewithoutcolon"); err == nil {
		t.Fatal("ref without :key should error")
	}
	if _, err := parseRef("col=/nonexistent.csv:id"); err == nil {
		t.Fatal("ref to missing file should error")
	}
}

func TestRunUIFailsToBindInvalidPort(t *testing.T) {
	// An invalid port makes ui.Serve return a bind error rather than block.
	if err := runUI([]string{"--port", "-1"}); err == nil {
		t.Fatal("invalid port should surface a bind error")
	}
}

func TestRunDispatch(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	csv := filepath.Join(dir, "d.csv")
	csv2 := filepath.Join(dir, "d2.csv")

	ok := func(args ...string) {
		if code := run(args); code != 0 {
			t.Fatalf("run(%v) = %d, want 0", args, code)
		}
	}
	// Flag/help/version dispatch (exit 0).
	ok("version")
	ok("-v")
	ok("help")
	ok("-h")
	// Every subcommand through the real dispatch, all success paths.
	ok("gen", "-s", spec, "-o", csv, "-f", "csv", "--seed", "1")
	ok("gen", "-s", spec, "-o", csv2, "-f", "csv", "--seed", "2")
	ok("profile", "-i", csv, "-o", filepath.Join(dir, "p.yaml"))
	ok("mask", "-i", csv, "-o", filepath.Join(dir, "m.csv"), "--key", "k")
	ok("cdc", "-s", spec, "-o", filepath.Join(dir, "c.jsonl"), "-n", "10")
	ok("verify", "-i", csv)
	ok("diff", csv, csv2)
	ok("snapshot", "-s", spec, "--at", "2026-06-01", "-o", filepath.Join(dir, "sn.csv"))

	// Exit-code paths.
	if run(nil) != 2 {
		t.Fatal("no args should exit 2")
	}
	if run([]string{"bogus-subcommand"}) != 2 {
		t.Fatal("unknown subcommand should exit 2")
	}
	if run([]string{"gen"}) != 1 {
		t.Fatal("a command that errors should exit 1")
	}
}

const childSpec = `
name: orders
count: 40
fields:
  id: { kind: uuid, pk: true }
  user_id: { kind: uuid }
  amount: { kind: amount, min: 1, max: 100 }
`

func TestRunCDCCascade(t *testing.T) {
	dir := t.TempDir()
	parent := writeFile(t, dir, "p.yaml", coverSpec)
	child := writeFile(t, dir, "c.yaml", childSpec)
	out := filepath.Join(dir, "cdc.jsonl")
	if err := runCDC([]string{"-s", parent, "--child", child, "--child-fk", "user_id",
		"-o", out, "-n", "20", "--snapshot", "3"}); err != nil {
		t.Fatalf("cdc cascade: %v", err)
	}
	// --child without --child-fk is an error.
	if err := runCDC([]string{"-s", parent, "--child", child, "-o", out}); err == nil {
		t.Fatal("cascade without child-fk should error")
	}
}

func TestRunDiffJSONAndErrors(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	a := filepath.Join(dir, "a.csv")
	b := filepath.Join(dir, "b.csv")
	if err := runGen([]string{"-s", spec, "-o", a, "-f", "csv", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", b, "-f", "csv", "--seed", "9"}); err != nil {
		t.Fatal(err)
	}
	if err := runDiff([]string{"-f", "json", a, b}); err != nil {
		t.Fatalf("diff json: %v", err)
	}
	if err := runDiff([]string{"--tolerance", "notnum", a, b}); err == nil {
		t.Fatal("bad tolerance should error")
	}
	if err := runDiff([]string{"--tolerance"}); err == nil {
		t.Fatal("tolerance without value should error")
	}
	if err := runDiff([]string{"-f"}); err == nil {
		t.Fatal("format without value should error")
	}
}

func TestRunVerifyConstraintsAndKAnonError(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	csv := filepath.Join(dir, "d.csv")
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}
	// A spec supplies constraints to re-check; JSON output to a file.
	if err := runVerify([]string{"-i", csv, "-s", spec, "-f", "json",
		"-o", filepath.Join(dir, "rep.json")}); err != nil {
		t.Fatalf("verify constraints json: %v", err)
	}
	// --k without --qi is an error.
	if err := runVerify([]string{"-i", csv, "--k", "2"}); err == nil {
		t.Fatal("k-anonymity without --qi should error")
	}
	// A dangling foreign key makes verify report the check-failed signal.
	child := writeFile(t, dir, "child.csv", "user_id\nZZZ\n")
	parent := writeFile(t, dir, "parent.csv", "id\naaa\n")
	if err := runVerify([]string{"-i", child, "--ref", "user_id=" + parent + ":id"}); err != errCheckFailed {
		t.Fatalf("dangling ref = %v, want errCheckFailed", err)
	}
	// Through run(), a failed check exits 1.
	if code := run([]string{"verify", "-i", child, "--ref", "user_id=" + parent + ":id"}); code != 1 {
		t.Fatalf("run verify fail exit = %d, want 1", code)
	}
}

func TestParseFlagErrors(t *testing.T) {
	bad := [][]string{
		{"-s"},                  // missing value
		{"--bogus"},             // unknown flag
		{"-n", "abc"},           // bad int
		{"--seed", "xx"},        // bad uint
		{"--chaos", "xx"},       // bad float
		{"--update-rate", "xx"}, // bad float
		{"--delete-rate", "xx"}, // bad float
		{"--snapshot", "xx"},    // bad int
		{"--churn", "xx"},       // bad float
		{"--k", "xx"},           // bad int
	}
	for _, args := range bad {
		if _, err := parseFlags(args); err == nil {
			t.Fatalf("parseFlags(%v) should error", args)
		}
	}
}

func TestSQLValueTypes(t *testing.T) {
	cases := map[string]any{
		"NULL":    nil,
		"42":      42,
		"3.5":     3.5,
		"TRUE":    true,
		"FALSE":   false,
		"'O''Br'": "O'Br",
	}
	for want, in := range cases {
		if got := sqlValue(in); got != want {
			t.Fatalf("sqlValue(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestOutputBadPath(t *testing.T) {
	if _, _, err := output(filepath.Join(t.TempDir(), "nodir", "f.csv")); err == nil {
		t.Fatal("output to a nonexistent dir should error")
	}
	// Empty path returns stdout with a no-op closer.
	f, closeOut, err := output("")
	if err != nil || f == nil {
		t.Fatal("output(\"\") should return stdout")
	}
	closeOut()
}

func TestRunCommandsFileErrors(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	bad := filepath.Join(dir, "nodir", "out")
	// Each command's output-open error path.
	if err := runGen([]string{"-s", spec, "-o", bad, "-f", "csv"}); err == nil {
		t.Fatal("gen bad out")
	}
	if err := runProfile([]string{"-i", "/nonexistent.csv", "-o", bad}); err == nil {
		t.Fatal("profile missing input")
	}
	if err := runCDC([]string{"-s", "/nonexistent.yaml"}); err == nil {
		t.Fatal("cdc missing spec")
	}
	if err := runSnapshot([]string{"-s", "/nonexistent.yaml", "--at", "2026-01-01"}); err == nil {
		t.Fatal("snapshot missing spec")
	}
	if err := runDiff([]string{"/nonexistent1.csv", "/nonexistent2.csv"}); err == nil {
		t.Fatal("diff missing inputs")
	}
}

func TestRunGenPgCopyAndSinks(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	for _, f := range []string{"pgcopy", "pgcopy-binary"} {
		if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "p."+f), "-f", f, "-n", "5"}); err != nil {
			t.Fatalf("gen %s: %v", f, err)
		}
	}
	// Gzip sink via .gz extension.
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "g.csv.gz"), "-n", "5"}); err != nil {
		t.Fatalf("gen gz: %v", err)
	}
	// Unknown format.
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "x"), "-f", "bogus"}); err == nil {
		t.Fatal("unknown format should error")
	}
	// Append to an unsupported format is rejected.
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "a.pgcopy"), "-f", "pgcopy", "--append"}); err == nil {
		t.Fatal("append to pgcopy should be rejected")
	}
}

func TestApplyDPAndFKErrors(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	csv := filepath.Join(dir, "d.csv")
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "1"}); err != nil {
		t.Fatal(err)
	}
	// Malformed DP rules through runMask.
	for _, dp := range []string{"badformat", "col:notnum:1", "col:1:notnum", "col:-1:1", "col:1:-1"} {
		if err := runMask([]string{"-i", csv, "-o", filepath.Join(dir, "m.csv"), "--key", "k", "--dp", dp}); err == nil {
			t.Fatalf("dp %q should error", dp)
		}
	}
	// Malformed --fk forms through runGen.
	for _, fk := range []string{"noequals", "col=filewithoutcolon", "col=/nonexistent.csv:id"} {
		if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "o.csv"), "--fk", fk}); err == nil {
			t.Fatalf("fk %q should error", fk)
		}
	}
	// A valid --fk from a parent file.
	parent := writeFile(t, dir, "parent.csv", "id\n"+"aaa\nbbb\nccc\n")
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "fk.csv"),
		"--fk", "email=" + parent + ":id", "-n", "5"}); err != nil {
		t.Fatalf("valid fk: %v", err)
	}
	// --fk with a missing key column.
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "fk2.csv"),
		"--fk", "email=" + parent + ":nosuch"}); err == nil {
		t.Fatal("fk missing key column should error")
	}
}

func TestNumericFlagsMissingValue(t *testing.T) {
	for _, args := range [][]string{{"-n"}, {"--seed"}, {"--chaos"}, {"--snapshot"}, {"--churn"}} {
		if _, err := parseFlags(args); err == nil {
			t.Fatalf("parseFlags(%v) missing value should error", args)
		}
	}
}

func TestDiffColumnChangesAndIdentical(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.csv", "x,gone\n1,2\n3,4\n5,6\n")
	b := writeFile(t, dir, "b.csv", "x,new\n1,2\n3,4\n5,6\n")
	// A structural break (column removed/added) reports the check-failed signal.
	if err := runDiff([]string{a, b}); err != errCheckFailed {
		t.Fatalf("diff cols = %v, want errCheckFailed", err)
	}
	// Identical files: no shape differences.
	if err := runDiff([]string{a, a}); err != nil {
		t.Fatalf("diff identical: %v", err)
	}
}

func TestSnapshotFormatsAndRange(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	// --at with jsonl and sql formats, plus churn/delete config.
	if err := runSnapshot([]string{"-s", spec, "--at", "2026-06-01", "-f", "jsonl",
		"-o", filepath.Join(dir, "a.jsonl"), "--churn", "2", "--delete-rate", "0.1"}); err != nil {
		t.Fatalf("snapshot jsonl: %v", err)
	}
	if err := runSnapshot([]string{"-s", spec, "--at", "2026-06-01", "-f", "sql",
		"-o", filepath.Join(dir, "a.sql")}); err != nil {
		t.Fatalf("snapshot sql: %v", err)
	}
	// Range writes change events.
	if err := runSnapshot([]string{"-s", spec, "--from", "2026-01-01", "--to", "2026-12-01",
		"-o", filepath.Join(dir, "r.jsonl")}); err != nil {
		t.Fatalf("snapshot range: %v", err)
	}
	// Bad instants and inverted range.
	if err := runSnapshot([]string{"-s", spec, "--at", "not-a-date"}); err == nil {
		t.Fatal("bad --at should error")
	}
	if err := runSnapshot([]string{"-s", spec, "--from", "bad", "--to", "2026-12-01"}); err == nil {
		t.Fatal("bad --from should error")
	}
	if err := runSnapshot([]string{"-s", spec, "--from", "2026-12-01", "--to", "2026-01-01"}); err == nil {
		t.Fatal("--to before --from should error")
	}
}

func TestCDCSoftDelete(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	if err := runCDC([]string{"-s", spec, "-o", filepath.Join(dir, "sd.jsonl"),
		"-n", "20", "--soft-delete", "--delete-rate", "0.2"}); err != nil {
		t.Fatalf("cdc soft-delete: %v", err)
	}
}

func TestAppendPathsAndState(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)

	// Append to jsonl (valid, header-less concatenation).
	jl := filepath.Join(dir, "a.jsonl")
	if err := runGen([]string{"-s", spec, "-o", jl, "-f", "jsonl", "--seed", "3", "-n", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", jl, "-f", "jsonl", "--seed", "3", "-n", "2", "--append"}); err != nil {
		t.Fatalf("append jsonl: %v", err)
	}
	// Append to stdout is rejected.
	if err := runGen([]string{"-s", spec, "--append", "-f", "csv"}); err == nil {
		t.Fatal("append to stdout should error")
	}
	// A corrupt state sidecar is an error, not a silent reset.
	csv := filepath.Join(dir, "b.csv")
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "5", "-n", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(csv+".synthstate", []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", csv, "-f", "csv", "--seed", "5", "--append"}); err == nil {
		t.Fatal("corrupt state should error")
	}
	// A seed that does not match the file's recorded seed is rejected. The seed
	// is only recorded once an append has written state, so append once first.
	csv2 := filepath.Join(dir, "c.csv")
	if err := runGen([]string{"-s", spec, "-o", csv2, "-f", "csv", "--seed", "7", "-n", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", csv2, "-f", "csv", "--seed", "7", "-n", "2", "--append"}); err != nil {
		t.Fatal(err)
	}
	if err := runGen([]string{"-s", spec, "-o", csv2, "-f", "csv", "--seed", "8", "-n", "2", "--append"}); err == nil {
		t.Fatal("seed mismatch on append should error")
	}
}

func TestZstdSink(t *testing.T) {
	dir := t.TempDir()
	spec := writeFile(t, dir, "s.yaml", coverSpec)
	if err := runGen([]string{"-s", spec, "-o", filepath.Join(dir, "z.csv.zst"), "-n", "5"}); err != nil {
		t.Fatalf("zstd sink: %v", err)
	}
}

func TestMiscHelpers(t *testing.T) {
	if versionString() == "" {
		t.Fatal("version empty")
	}
	if presetList() == "" {
		t.Fatal("preset list empty")
	}
	usage() // prints to stderr; just exercise it
}
