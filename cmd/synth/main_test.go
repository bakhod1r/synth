package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// build compiles the CLI once and returns the binary path.
func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "synth")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func run(t *testing.T, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v failed: %v\nstderr: %s", args, err, stderr.String())
	}
	return stdout.String()
}

func mustExist(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	if len(b) == 0 {
		t.Fatalf("%s is empty", path)
	}
	return b
}

const spec = `name: users
count: 50
locale: uz_UZ
seed: 42
fields:
  id:     { kind: uuid, pk: true }
  name:   { kind: name }
  email:  { kind: email }
  age:    { kind: int, min: 18, max: 65 }
`

// Every subcommand must run end to end and produce its output file.
func TestSubcommands(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}

	csvOut := filepath.Join(dir, "users.csv")
	run(t, bin, "gen", "-s", specPath, "-o", csvOut, "-f", "csv")
	csvData := mustExist(t, csvOut)
	if got := bytes.Count(csvData, []byte("\n")); got != 51 { // header + 50 rows
		t.Fatalf("csv has %d lines, want 51", got)
	}

	// profile reads the exported file and infers a spec — no database.
	profOut := filepath.Join(dir, "inferred.yaml")
	run(t, bin, "profile", "-i", csvOut, "-o", profOut)
	inferred := mustExist(t, profOut)
	if !bytes.Contains(inferred, []byte("fields:")) {
		t.Fatalf("inferred spec has no fields block:\n%s", inferred)
	}
	// The inferred spec must itself be usable as input.
	regen := filepath.Join(dir, "regen.csv")
	run(t, bin, "gen", "-s", profOut, "-o", regen)
	mustExist(t, regen)

	maskOut := filepath.Join(dir, "masked.csv")
	run(t, bin, "mask", "-i", csvOut, "-o", maskOut, "--key", "test-key")
	masked := mustExist(t, maskOut)
	if bytes.Equal(csvData, masked) {
		t.Fatal("mask produced a file identical to the input")
	}

	cdcOut := filepath.Join(dir, "events.jsonl")
	run(t, bin, "cdc", "-s", specPath, "-o", cdcOut, "-n", "20")
	events := mustExist(t, cdcOut)
	if got := bytes.Count(events, []byte("\n")); got != 20 {
		t.Fatalf("wrote %d events, want 20", got)
	}
	if !bytes.Contains(events, []byte(`"op"`)) {
		t.Fatalf("events are not in Debezium envelope shape:\n%s", events[:200])
	}
}

// Masking without a key must fail: an unkeyed run would not be reproducible.
func TestMaskRequiresKey(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	os.WriteFile(in, []byte("email\na@b.com\n"), 0o644)

	cmd := exec.Command(bin, "mask", "-i", in, "-o", filepath.Join(dir, "out.csv"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected a non-zero exit without --key")
	}
	if !strings.Contains(stderr.String(), "--key") {
		t.Fatalf("error does not mention --key: %s", stderr.String())
	}
}

// Masking in place would destroy the original data, so it must be refused.
func TestMaskRefusesToOverwriteInput(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	os.WriteFile(in, []byte("email\na@b.com\n"), 0o644)

	cmd := exec.Command(bin, "mask", "-i", in, "-o", in, "--key", "k")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected a refusal to overwrite the input")
	}
	if !strings.Contains(stderr.String(), "overwrite") {
		t.Fatalf("error does not explain the refusal: %s", stderr.String())
	}
}

// Parquet is a submodule, so the core CLI must say so rather than silently
// writing some other format.
// Parquet is a first-class CLI output: the binary writes a real .parquet file
// (PAR1 magic at both ends), but refuses to stream it to stdout.
func TestParquetOutput(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(spec), 0o644)
	out := filepath.Join(dir, "out.parquet")

	if err := exec.Command(bin, "gen", "-s", specPath, "-o", out, "-n", "20").Run(); err != nil {
		t.Fatalf("gen parquet: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil || len(b) < 8 || string(b[:4]) != "PAR1" || string(b[len(b)-4:]) != "PAR1" {
		t.Fatalf("parquet magic missing: err=%v len=%d", err, len(b))
	}

	cmd := exec.Command(bin, "gen", "-s", specPath, "-f", "parquet")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("parquet to stdout should error")
	}
}

// An unknown subcommand must exit non-zero and print usage.
func TestUnknownSubcommand(t *testing.T) {
	bin := build(t)
	cmd := exec.Command(bin, "frobnicate")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected a non-zero exit")
	}
	for _, want := range []string{"frobnicate", "gen", "profile", "mask", "cdc"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("usage does not mention %q: %s", want, stderr.String())
		}
	}
}

// A bad numeric flag must be reported, not silently treated as zero.
func TestBadNumericFlagIsReported(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(spec), 0o644)

	cmd := exec.Command(bin, "gen", "-s", specPath, "-n", "lots")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for a non-numeric -n")
	}
	if !strings.Contains(stderr.String(), "lots") {
		t.Fatalf("error does not name the bad value: %s", stderr.String())
	}
}

// verify must audit a real file, resolve foreign keys across files, and use
// its exit code so CI can gate on it.
func TestVerifySubcommand(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()

	users := filepath.Join(dir, "users.csv")
	os.WriteFile(users, []byte("id,email\nu1,a@example.com\nu2,b@example.com\n"), 0o644)

	// A clean child table: exit 0, and the report says so.
	good := filepath.Join(dir, "good.csv")
	os.WriteFile(good, []byte("order_id,user_id\no1,u1\no2,u2\n"), 0o644)
	out := run(t, bin, "verify", "-i", good, "--ref", "user_id="+users+":id")
	if !strings.Contains(out, "no problems") {
		t.Fatalf("clean data did not report clean: %q", out)
	}

	// A dangling key: exit 1, and the bad value is named.
	bad := filepath.Join(dir, "bad.csv")
	os.WriteFile(bad, []byte("order_id,user_id\no1,u1\no2,ghost\n"), 0o644)
	cmd := exec.Command(bin, "verify", "-i", bad, "--ref", "user_id="+users+":id")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 for a dangling foreign key")
	}
	if !strings.Contains(stdout.String(), "ghost") {
		t.Fatalf("report does not name the dangling value:\n%s", stdout.String())
	}
}

// A malformed --ref must be explained, not silently ignored.
func TestVerifyRejectsMalformedRef(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	os.WriteFile(in, []byte("a\n1\n"), 0o644)

	cmd := exec.Command(bin, "verify", "-i", in, "--ref", "nonsense")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for a malformed --ref")
	}
	if !strings.Contains(stderr.String(), "col=parent.csv:key") {
		t.Fatalf("error does not show the expected form: %s", stderr.String())
	}
}

// snapshot must materialize a point in time, and its event log must line up
// with the snapshots on either side.
func TestSnapshotSubcommand(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(`name: orders
count: 200
seed: 42
fields:
  id:         { kind: uuid, pk: true }
  customer:   { kind: name }
  amount:     { kind: amount, min: 1, max: 500 }
  created_at: { kind: time }
  updated_at: { kind: time }
`), 0o644)

	jan := filepath.Join(dir, "jan.csv")
	run(t, bin, "snapshot", "-s", specPath, "--at", "2026-04-01", "-o", jan, "--churn", "2")
	janRows := mustExist(t, jan)

	jul := filepath.Join(dir, "jul.csv")
	run(t, bin, "snapshot", "-s", specPath, "--at", "2026-10-01", "-o", jul, "--churn", "2")
	julRows := mustExist(t, jul)

	// The table grows over time, so the later snapshot must have more rows.
	if bytes.Count(julRows, []byte("\n")) <= bytes.Count(janRows, []byte("\n")) {
		t.Fatal("the later snapshot is not larger than the earlier one")
	}

	changes := filepath.Join(dir, "changes.jsonl")
	run(t, bin, "snapshot", "-s", specPath, "--from", "2026-04-01",
		"--to", "2026-10-01", "-o", changes, "--churn", "2")
	events := mustExist(t, changes)
	if !bytes.Contains(events, []byte(`"op"`)) {
		t.Fatalf("events are not in Debezium envelope shape:\n%s", events[:200])
	}
}

// Asking for an instant and a range at once is a contradiction, not a
// preference to resolve silently.
func TestSnapshotRejectsAtWithRange(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(spec), 0o644)

	cmd := exec.Command(bin, "snapshot", "-s", specPath,
		"--at", "2026-01-01", "--from", "2026-01-01", "--to", "2026-02-01")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error when --at and --from/--to are combined")
	}
	if !strings.Contains(stderr.String(), "one or the other") {
		t.Fatalf("error does not explain the conflict: %s", stderr.String())
	}
}

// A backwards range is a mistake worth reporting.
func TestSnapshotRejectsBackwardsRange(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(spec), 0o644)

	cmd := exec.Command(bin, "snapshot", "-s", specPath, "--from", "2026-06-01", "--to", "2026-01-01")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for a backwards range")
	}
	if !strings.Contains(stderr.String(), "after") {
		t.Fatalf("error is unclear: %s", stderr.String())
	}
}

// An unparseable date must say what form it wanted.
func TestSnapshotRejectsBadDate(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	os.WriteFile(specPath, []byte(spec), 0o644)

	cmd := exec.Command(bin, "snapshot", "-s", specPath, "--at", "last tuesday")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error for an unparseable date")
	}
	if !strings.Contains(stderr.String(), "2026-01-01") {
		t.Fatalf("error does not show the expected form: %s", stderr.String())
	}
}

// A released binary must be able to say what it is. When someone reports a bug,
// the first question is which build they were running, and a binary that cannot
// answer ends the conversation.
func TestVersionFlag(t *testing.T) {
	bin := build(t)
	for _, flag := range []string{"--version", "-v", "version"} {
		out := run(t, bin, flag)
		if !strings.HasPrefix(out, "synth ") {
			t.Fatalf("%s printed %q", flag, out)
		}
		// The line must carry enough to identify a build: a version, the
		// platform it was built for, and the Go toolchain.
		for _, want := range []string{runtime.GOOS, runtime.GOARCH, "go1."} {
			if !strings.Contains(out, want) {
				t.Errorf("%s output %q does not mention %q", flag, strings.TrimSpace(out), want)
			}
		}
	}
}

// The version stamped by the release workflow must actually reach the binary.
// -ldflags pointing at a variable that does not exist fails silently, which is
// how a release ships reporting "(devel)".
func TestVersionIsStampable(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "synth")
	out, err := exec.Command("go", "build",
		"-ldflags", "-X main.version=v9.9.9-test", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if got := run(t, bin, "--version"); !strings.Contains(got, "v9.9.9-test") {
		t.Fatalf("the stamped version did not reach the binary: %q", strings.TrimSpace(got))
	}
}
