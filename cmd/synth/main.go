// Command synth generates, profiles, masks and audits data from the command
// line. Synth stays a pure provider: every subcommand reads and writes files,
// none of them opens a database or network connection.
//
// Usage:
//
//	synth gen -s users.yaml -o users.csv            # format from extension
//	synth gen -s users.yaml -f sql -o users.sql -n 100000
//	synth gen -s users.yaml -o users.pgbin           # Postgres COPY + CREATE TABLE
//	synth gen -s users.yaml -o users.jsonl.zst       # compressed by extension
//	synth profile -i export.csv -o inferred.yaml    # learn a spec from real data
//	synth mask -i export.csv -o safe.csv --key K    # anonymize a real dump
//	synth cdc -s users.yaml -o changes.jsonl -n 1000
//	synth verify -i orders.csv --ref user_id=users.csv:id
//	synth snapshot -s users.yaml --at 2026-01-01 -o jan.csv
//	synth ui --port 8080                            # local browser workbench
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/constraint"
	parquet "github.com/bakhod1r/synth/sink/parquet"
	"github.com/bakhod1r/synth/ui"
	"github.com/bakhod1r/synth/verify"
)

// version is stamped at build time with -ldflags "-X main.version=v1.2.3".
//
// It falls back to the module version the binary was built from, so a copy
// installed with `go install` still reports something true rather than
// "unknown" — the answer matters most when someone is reporting a bug against a
// build nobody can identify.
var version = ""

func versionString() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

// errCheckFailed signals a data-quality failure — verify found defects, or diff
// found structural breaks — that must exit non-zero. It carries no message: the
// report itself has already been written, so run reports only the exit code.
var errCheckFailed = errors.New("check failed")

func main() { os.Exit(run(os.Args[1:])) }

// run dispatches a subcommand and returns the process exit code, so every
// path — including the error and usage exits — is reachable from a test without
// terminating the process.
func run(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	var err error
	switch args[0] {
	case "gen":
		err = runGen(args[1:])
	case "profile":
		err = runProfile(args[1:])
	case "mask":
		err = runMask(args[1:])
	case "cdc":
		err = runCDC(args[1:])
	case "verify":
		err = runVerify(args[1:])
	case "diff":
		err = runDiff(args[1:])
	case "snapshot":
		err = runSnapshot(args[1:])
	case "ui":
		err = runUI(args[1:])
	case "-h", "--help", "help":
		usage()
		return 0
	case "-v", "--version", "version":
		fmt.Printf("synth %s %s/%s %s\n",
			versionString(), runtime.GOOS, runtime.GOARCH, runtime.Version())
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", args[0])
		usage()
		return 2
	}
	if errors.Is(err, errCheckFailed) {
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "synth:", err)
		return 1
	}
	return 0
}

// runGen generates records from a YAML spec.
func runGen(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.spec == "" && fs.preset == "" {
		return fmt.Errorf("gen: give -s <spec.yaml> or --preset <name> (%s)", presetList())
	}
	var spec *synth.YAMLSpec
	if fs.preset != "" {
		spec, err = synth.Spec(synth.Preset(fs.preset))
		if err != nil {
			return fmt.Errorf("%w — available: %s", err, presetList())
		}
	} else {
		spec, err = synth.LoadYAML(fs.spec)
	}
	if err != nil {
		return err
	}
	opts := fs.options()
	if fs.unmask {
		opts = append(opts, synth.Unmasked())
	}
	fkOpts, err := fkOptions(fs.fks)
	if err != nil {
		return err
	}
	opts = append(opts, fkOpts...)

	format := fs.format
	if format == "" {
		format = formatFromExt(fs.out)
	}
	table := spec.Name()
	if table == "" {
		table = "data"
	}

	// Append: continue an existing file rather than overwriting it. The state
	// sidecar carries the offset so the new rows differ from the ones already
	// written, and the header is suppressed so it is not repeated mid-file.
	var state genState
	if fs.append {
		if err := checkAppendable(format, fs.out); err != nil {
			return err
		}
		state, err = readState(statePath(fs.out))
		if err != nil {
			return err
		}
		if state.Rows > 0 && state.Seed != fs.seed {
			return fmt.Errorf("gen: --append seed %d does not match the file's "+
				"seed %d; pass --seed %d to continue it", fs.seed, state.Seed, state.Seed)
		}
		opts = append(opts, synth.Offset(state.Rows))
	}

	recs, err := spec.GenerateN(fs.n, opts...)
	if err != nil {
		return err
	}

	// Parquet writes a self-describing columnar file with a footer, so it needs a
	// real seekable path and cannot stream through the gzip/zstd sink or stdout.
	// It also carries its own schema, so there is no companion DDL to emit.
	if format == "parquet" {
		if fs.out == "" {
			return fmt.Errorf("gen: parquet needs -o <file>, not stdout")
		}
		if err := parquet.WriteRows(fs.out, spec.Columns(), recs); err != nil {
			return fmt.Errorf("gen: parquet: %w", err)
		}
		fmt.Fprintf(os.Stderr, "wrote %d rows to %s (parquet)\n", len(recs), fs.out)
		return nil
	}

	out, err := openSinkMode(fs.out, fs.append)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	switch format {
	case "jsonl":
		writeJSONL(w, recs)
	case "sql":
		writeSQL(w, table, spec.Columns(), recs)
	case "csv":
		// The header goes in only when starting a fresh file. A first --append
		// against a file that does not exist yet (state.Rows == 0) still needs
		// it; a later append into an existing file must not repeat it.
		writeCSVRows(w, spec.Columns(), recs, !fs.append || state.Rows == 0)
	case "pgcopy", "pgcopy-binary":
		if err := writePgCopy(w, format, table, spec, recs, fs.out); err != nil {
			return err
		}
	default:
		return fmt.Errorf("gen: unknown format %q "+
			"(want csv, jsonl, sql, parquet, pgcopy or pgcopy-binary)", format)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	// Closed explicitly, not by the defer: with a compressor in the way, Close
	// is what writes the final block, and its error is the difference between
	// a complete file and a truncated one.
	if err := out.Close(); err != nil {
		return err
	}
	if fs.append {
		state.Rows += len(recs)
		state.Seed = fs.seed
		if err := writeState(statePath(fs.out), state); err != nil {
			return err
		}
	}
	if fs.out != "" {
		fmt.Fprintf(os.Stderr, "wrote %d rows to %s (%s)\n", len(recs), fs.out, format)
	}
	return nil
}

// checkAppendable rejects the format/target combinations append cannot handle.
// Appending concatenates rows, which is meaningful only for the row-per-line
// formats; a second pgcopy or parquet file embedded after the first would carry
// its own header and trailer and be unreadable, and stdout cannot be appended
// to a prior run.
func checkAppendable(format, out string) error {
	if out == "" {
		return fmt.Errorf("gen: --append needs -o <file>, not stdout")
	}
	switch format {
	case "csv", "jsonl", "sql":
		return nil
	default:
		return fmt.Errorf("gen: --append supports csv, jsonl and sql, not %q", format)
	}
}

// runProfile learns a spec from a real CSV/JSONL export. It reads the file;
// it never touches the system the export came from.
func runProfile(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.in == "" {
		return fmt.Errorf("profile: -i <export.csv> is required")
	}
	p, err := synth.Profile(fs.in)
	if err != nil {
		return err
	}
	name := fs.name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(fs.in), filepath.Ext(fs.in))
	}
	count := fs.n
	if count == 0 {
		count = p.SampleRows()
	}
	doc, err := p.YAML(name, count)
	if err != nil {
		return err
	}
	out, closeOut, err := output(fs.out)
	if err != nil {
		return err
	}
	defer closeOut()
	if _, err := out.Write(doc); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "profiled %d rows, %d columns\n", p.SampleRows(), len(p.Columns()))
	return nil
}

// runMask rewrites a real export with synthetic values of the same shape.
func runMask(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.in == "" {
		return fmt.Errorf("mask: -i <export.csv> is required")
	}
	if fs.out == "" {
		return fmt.Errorf("mask: -o <output> is required (masking never writes over the input)")
	}
	if fs.key == "" {
		return fmt.Errorf("mask: --key is required — it makes replacements " +
			"deterministic so foreign keys still join across files")
	}
	if fs.in == fs.out {
		return fmt.Errorf("mask: -i and -o are the same file; refusing to " +
			"overwrite the original data")
	}
	m := synth.NewMasker(fs.key, fs.locale)
	if err := applyDPRules(m, fs.dps); err != nil {
		return err
	}
	rep, err := m.File(fs.in, fs.out)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "masked %d rows: %s -> %s\n", rep.Rows, fs.in, fs.out)
	for _, col := range rep.Columns {
		if n := rep.Masked[col]; n > 0 {
			fmt.Fprintf(os.Stderr, "  %s: %d values replaced\n", col, n)
		}
	}
	if len(rep.Untouched) > 0 {
		fmt.Fprintf(os.Stderr, "  left as-is: %s\n", strings.Join(rep.Untouched, ", "))
	}
	return nil
}

// runCDC writes a coherent insert/update/delete history as JSONL.
func runCDC(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.spec == "" {
		return fmt.Errorf("cdc: -s <spec.yaml> is required")
	}
	spec, err := synth.LoadYAML(fs.spec)
	if err != nil {
		return err
	}
	n := fs.n
	if n == 0 {
		n = spec.Count()
	}
	table := spec.Name()
	if table == "" {
		table = "data"
	}

	// A change stream over one table, or — with --child — a two-table stream
	// where deleting a parent cascades to its children. Both write JSONL the
	// same way.
	type jsonlStream interface {
		WriteJSONL(io.Writer, int) error
	}
	var stream jsonlStream
	if fs.child != "" {
		if fs.childFK == "" {
			return fmt.Errorf("cdc: --child needs --child-fk (the child column " +
				"that references the parent)")
		}
		childSpec, err := synth.LoadYAML(fs.child)
		if err != nil {
			return err
		}
		childTable := childSpec.Name()
		if childTable == "" {
			childTable = "child"
		}
		stream, err = spec.Cascade(childSpec, synth.CascadeConfig{
			ParentTable: table, ChildTable: childTable, ChildFK: fs.childFK,
			UpdateRate: fs.updateRate, DeleteRate: fs.deleteRate,
			Snapshot: fs.snapshot, Seed: fs.seed, Locale: fs.locale,
		})
		if err != nil {
			return err
		}
	} else {
		stream, err = spec.CDC(synth.CDCConfig{
			Table:      table,
			UpdateRate: fs.updateRate,
			DeleteRate: fs.deleteRate,
			Snapshot:   fs.snapshot,
			SoftDelete: fs.softDelete,
		})
		if err != nil {
			return err
		}
	}
	out, err := openSink(fs.out)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	if err := stream.WriteJSONL(w, n); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if fs.out != "" {
		fmt.Fprintf(os.Stderr, "wrote %d change events to %s\n", n, fs.out)
	}
	return nil
}

type flags struct {
	spec, in, out, format, locale, key, name string
	refs, fks, dps                           []string
	qi, child, childFK                       string
	at, from, to, port, preset               string
	unmask, append, softDelete               bool
	seed                                     uint64
	n, snapshot, k                           int
	chaos, updateRate, deleteRate, churn     float64
}

// options turns the shared generation flags into synth options.
func (f flags) options() []synth.Option {
	var opts []synth.Option
	if f.seed != 0 {
		opts = append(opts, synth.WithSeed(f.seed))
	}
	if f.locale != "" {
		opts = append(opts, synth.WithLocale(f.locale))
	}
	if f.chaos > 0 {
		opts = append(opts, synth.WithChaos(f.chaos))
	}
	return opts
}

func parseFlags(args []string) (flags, error) {
	var f flags
	for i := 0; i < len(args); i++ {
		var err error
		switch args[i] {
		case "-s", "--spec":
			f.spec, err = next(args, &i)
		case "-i", "--in":
			f.in, err = next(args, &i)
		case "-o", "--out":
			f.out, err = next(args, &i)
		case "-f", "--format":
			f.format, err = next(args, &i)
		case "-l", "--locale":
			f.locale, err = next(args, &i)
		case "--key":
			f.key, err = next(args, &i)
		case "--name":
			f.name, err = next(args, &i)
		case "--at":
			f.at, err = next(args, &i)
		case "--from":
			f.from, err = next(args, &i)
		case "--to":
			f.to, err = next(args, &i)
		case "--preset":
			f.preset, err = next(args, &i)
		case "--unmasked":
			f.unmask = true
		case "--port":
			f.port, err = next(args, &i)
		case "--churn":
			err = scanFloat(args, &i, &f.churn)
		case "--ref":
			var v string
			v, err = next(args, &i)
			f.refs = append(f.refs, v)
		case "--fk":
			var v string
			v, err = next(args, &i)
			f.fks = append(f.fks, v)
		case "--append":
			f.append = true
		case "--soft-delete":
			f.softDelete = true
		case "--k":
			err = scanInto(args, &i, &f.k)
		case "--qi":
			f.qi, err = next(args, &i)
		case "--dp":
			var v string
			v, err = next(args, &i)
			f.dps = append(f.dps, v)
		case "--child":
			f.child, err = next(args, &i)
		case "--child-fk":
			f.childFK, err = next(args, &i)
		case "-n", "--rows":
			err = scanInto(args, &i, &f.n)
		case "--snapshot":
			err = scanInto(args, &i, &f.snapshot)
		case "--seed":
			err = scanUint(args, &i, &f.seed)
		case "--chaos":
			err = scanFloat(args, &i, &f.chaos)
		case "--update-rate":
			err = scanFloat(args, &i, &f.updateRate)
		case "--delete-rate":
			err = scanFloat(args, &i, &f.deleteRate)
		default:
			return f, fmt.Errorf("unknown flag: %s", args[i])
		}
		if err != nil {
			return f, err
		}
	}
	return f, nil
}

func next(args []string, i *int) (string, error) {
	flag := args[*i]
	*i++
	if *i >= len(args) {
		return "", fmt.Errorf("missing value for %s", flag)
	}
	return args[*i], nil
}

// scanInto parses a numeric flag value and reports a bad one instead of
// silently leaving the field at zero.
func scanInto(args []string, i *int, dst *int) error {
	flag := args[*i]
	v, err := next(args, i)
	if err != nil {
		return err
	}
	if _, err := fmt.Sscanf(v, "%d", dst); err != nil {
		return fmt.Errorf("%s: %q is not a number", flag, v)
	}
	return nil
}

func scanUint(args []string, i *int, dst *uint64) error {
	flag := args[*i]
	v, err := next(args, i)
	if err != nil {
		return err
	}
	if _, err := fmt.Sscanf(v, "%d", dst); err != nil {
		return fmt.Errorf("%s: %q is not a number", flag, v)
	}
	return nil
}

func scanFloat(args []string, i *int, dst *float64) error {
	flag := args[*i]
	v, err := next(args, i)
	if err != nil {
		return err
	}
	if _, err := fmt.Sscanf(v, "%g", dst); err != nil {
		return fmt.Errorf("%s: %q is not a number", flag, v)
	}
	return nil
}

// output opens the destination, defaulting to stdout. The returned close
// function is a no-op for stdout so callers can defer it unconditionally.
func output(path string) (*os.File, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func formatFromExt(path string) string {
	switch strings.TrimPrefix(filepath.Ext(stripCompressionExt(path)), ".") {
	case "jsonl", "json":
		return "jsonl"
	case "sql":
		return "sql"
	case "parquet":
		return "parquet"
	case "pgcopy":
		return "pgcopy"
	case "pgbin":
		return "pgcopy-binary"
	default:
		return "csv"
	}
}

func writeCSV(w *bufio.Writer, cols []string, recs []map[string]any) {
	writeCSVRows(w, cols, recs, true)
}

// writeCSVRows writes rows, emitting the header only when asked. An --append
// run writes into an existing file that already has its header, so a second one
// mid-file would be read as a data row.
func writeCSVRows(w *bufio.Writer, cols []string, recs []map[string]any, header bool) {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if header {
		cw.Write(cols)
	}
	row := make([]string, len(cols))
	for _, r := range recs {
		for i, c := range cols {
			row[i] = fmt.Sprint(r[c])
		}
		cw.Write(row)
	}
}

func writeJSONL(w *bufio.Writer, recs []map[string]any) {
	enc := json.NewEncoder(w)
	for _, r := range recs {
		enc.Encode(r)
	}
}

func writeSQL(w *bufio.Writer, table string, cols []string, recs []map[string]any) {
	colList := strings.Join(cols, ", ")
	for _, r := range recs {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = sqlValue(r[c])
		}
		fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", "))
	}
}

func sqlValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(x)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `synth — realistic data generator

Every subcommand reads and writes files. Synth never connects to a database.

Usage:
  synth gen     -s <spec.yaml> [-o out] [-f csv|jsonl|sql] [-n rows] [-l locale] [--seed N] [--chaos p]
  synth gen     -s <child.yaml> -o child.csv --fk col=parent.csv:key   # FK from a parent file
  synth gen     -s <spec.yaml> -o out.csv --append                     # extend an existing file
  synth profile -i <export.csv> [-o spec.yaml] [--name table] [-n rows]
  synth mask    -i <export.csv> -o <safe.csv> --key <secret> [-l locale]
  synth snapshot -s <spec.yaml> --at <date> [-o out]        # the table at an instant
  synth snapshot -s <spec.yaml> --from <date> --to <date>   # what changed between two
  synth gen     --preset <name> -n 100 -o out.csv           # built-in schema
  synth --version
  synth ui      [--port 8080]                               # browser workbench (loopback only)
  synth verify  -i <data.csv> [--ref col=parent.csv:key] [-s spec.yaml] [-f text|json]
  synth diff    <a.csv> <b.csv> [--tolerance 0.1] [-f text|json]   # compare two datasets' shape
  synth cdc     -s <spec.yaml> [-o changes.jsonl] [-n events] [--update-rate p] [--delete-rate p] [--snapshot N] [--soft-delete]
  synth cdc     -s <parent.yaml> --child <child.yaml> --child-fk <col>   # cascade deletes across two tables

Flags:
  -s, --spec       YAML data-definition file
  -i, --in         input file to profile or mask
  -o, --out        output file (default: stdout)
  -f, --format     csv | jsonl | sql | parquet | pgcopy | pgcopy-binary
  -n, --rows       number of rows or events
  -l, --locale     locale (e.g. uz_UZ)
      --name       table name for a profiled spec
      --key        masking key; the same key keeps foreign keys joinable
      --seed       deterministic seed
      --chaos      fraction of edge-case values (0..1)
      --preset     built-in schema (see the list below)
      --unmasked   return raw card numbers and identifiers instead of masked
      --at, --from, --to   instants for snapshot (2026-01-01 or RFC 3339)
      --churn      mean updates per row over the window
      --ref        foreign key to resolve, as col=parent.csv:key (repeatable)
      --fk         foreign key to fill from a parent file, col=parent.csv:key (repeatable)
      --append     extend the output file instead of overwriting it
      --update-rate, --delete-rate, --snapshot   CDC history shape
      --soft-delete   emit a delete as an update stamping deleted_at
      --child, --child-fk   cascade CDC: child spec and its FK column to the parent
      --k, --qi    k-anonymity: require each --qi col,col combination k+ times
      --dp         Laplace-noise a numeric column while masking, col:epsilon:sensitivity (repeatable)

synth verify exits 1 when it finds an error, 0 when it finds only warnings,
so it drops into CI without a wrapper.`)
}

// runVerify audits an existing dataset. Both the data and any parent tables
// are read from files; nothing is queried.
func runVerify(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.in == "" {
		return fmt.Errorf("verify: -i <data.csv> is required")
	}
	rows, err := constraint.LoadSample(fs.in, 0)
	if err != nil {
		return err
	}
	opts := verify.Options{}
	for _, spec := range fs.refs {
		ref, err := parseRef(spec)
		if err != nil {
			return err
		}
		opts.Refs = append(opts.Refs, ref)
	}
	// A spec supplies the invariants to re-check against this dataset.
	if fs.spec != "" {
		y, err := synth.LoadYAML(fs.spec)
		if err != nil {
			return err
		}
		for _, c := range y.Constraints() {
			opts.Constraints = append(opts.Constraints, c)
		}
	}
	if fs.k > 0 {
		opts.KAnonymity = fs.k
		for _, c := range strings.Split(fs.qi, ",") {
			if c = strings.TrimSpace(c); c != "" {
				opts.QuasiIdentifiers = append(opts.QuasiIdentifiers, c)
			}
		}
		if len(opts.QuasiIdentifiers) == 0 {
			return fmt.Errorf("verify: --k needs --qi col,col (the quasi-identifier columns)")
		}
	}

	rep := verify.Run(rows, opts)
	out, closeOut, err := output(fs.out)
	if err != nil {
		return err
	}
	defer closeOut()
	if fs.format == "json" {
		err = rep.JSON(out)
	} else {
		err = rep.Text(out)
	}
	if err != nil {
		return err
	}
	if !rep.OK() {
		return errCheckFailed
	}
	return nil
}

// fkOptions turns each --fk col=parent.csv:key flag into a RefValues option by
// reading the key column out of the parent file. This is what makes foreign
// keys work across runs: the parent was written in an earlier run, and the
// child now draws its FK from those written values.
func fkOptions(fks []string) ([]synth.Option, error) {
	var opts []synth.Option
	for _, spec := range fks {
		col, rest, ok := strings.Cut(spec, "=")
		if !ok {
			return nil, fmt.Errorf("--fk %q: want col=parent.csv:key", spec)
		}
		file, key, ok := strings.Cut(rest, ":")
		if !ok {
			return nil, fmt.Errorf("--fk %q: missing :key after the parent file", spec)
		}
		parent, err := constraint.LoadSample(file, 0)
		if err != nil {
			return nil, fmt.Errorf("--fk %q: %w", spec, err)
		}
		if len(parent) == 0 {
			return nil, fmt.Errorf("--fk %q: parent file %s has no rows", spec, file)
		}
		values := make([]any, 0, len(parent))
		for _, row := range parent {
			v, ok := row[key]
			if !ok {
				return nil, fmt.Errorf("--fk %q: parent file %s has no column %q",
					spec, file, key)
			}
			values = append(values, v)
		}
		opts = append(opts, synth.RefValues(col, values))
	}
	return opts, nil
}

// applyDPRules registers a Laplace-noise rule per --dp col:epsilon:sensitivity
// flag, so masking a numeric column can bound how much any one record shows
// through the released number.
func applyDPRules(m *synth.Masker, dps []string) error {
	for _, spec := range dps {
		parts := strings.Split(spec, ":")
		if len(parts) != 3 {
			return fmt.Errorf("--dp %q: want col:epsilon:sensitivity", spec)
		}
		eps, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || eps <= 0 {
			return fmt.Errorf("--dp %q: epsilon must be a positive number", spec)
		}
		sens, err := strconv.ParseFloat(parts[2], 64)
		if err != nil || sens <= 0 {
			return fmt.Errorf("--dp %q: sensitivity must be a positive number", spec)
		}
		m.Rule(synth.MaskRule{
			Column: parts[0], Strategy: synth.MaskDP, Epsilon: eps, Sensitivity: sens,
		})
	}
	return nil
}

// parseRef reads a foreign key written as col=parent.csv:key.
func parseRef(spec string) (verify.Ref, error) {
	col, rest, ok := strings.Cut(spec, "=")
	if !ok {
		return verify.Ref{}, fmt.Errorf("--ref %q: want col=parent.csv:key", spec)
	}
	file, key, ok := strings.Cut(rest, ":")
	if !ok {
		return verify.Ref{}, fmt.Errorf("--ref %q: missing :key after the parent file", spec)
	}
	parent, err := constraint.LoadSample(file, 0)
	if err != nil {
		return verify.Ref{}, fmt.Errorf("--ref %q: %w", spec, err)
	}
	return verify.Ref{Column: col, Parent: parent, ParentKey: key}, nil
}

// runSnapshot materializes the table at an instant, or the change events
// between two. Both come from the same derived per-row history, so replaying
// the events over the earlier snapshot reproduces the later one.
func runSnapshot(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	if fs.spec == "" {
		return fmt.Errorf("snapshot: -s <spec.yaml> is required")
	}
	if fs.at == "" && (fs.from == "" || fs.to == "") {
		return fmt.Errorf("snapshot: give --at <date>, or both --from and --to")
	}
	if fs.at != "" && (fs.from != "" || fs.to != "") {
		return fmt.Errorf("snapshot: --at asks for one instant and --from/--to " +
			"for a range; use one or the other")
	}
	spec, err := synth.LoadYAML(fs.spec)
	if err != nil {
		return err
	}
	cfg := synth.SnapshotConfig{Rows: fs.n, Churn: fs.churn, Seed: fs.seed, Locale: fs.locale}
	if fs.deleteRate > 0 {
		cfg.DeleteFrac = fs.deleteRate
	}
	tl, err := spec.Snapshot(cfg)
	if err != nil {
		return err
	}

	out, err := openSink(fs.out)
	if err != nil {
		return err
	}
	// LIFO: the buffer flushes, then the compressor writes its final block,
	// then the file closes.
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()

	if fs.at != "" {
		when, err := synth.ParseInstant(fs.at)
		if err != nil {
			return err
		}
		rows := tl.At(when)
		format := fs.format
		if format == "" {
			format = formatFromExt(fs.out)
		}
		switch format {
		case "jsonl":
			writeJSONL(w, rows)
		case "sql":
			writeSQL(w, spec.Name(), spec.Columns(), rows)
		default:
			writeCSV(w, spec.Columns(), rows)
		}
		fmt.Fprintf(os.Stderr, "%d rows as of %s\n", len(rows), when.Format(time.RFC3339))
		return nil
	}

	from, err := synth.ParseInstant(fs.from)
	if err != nil {
		return err
	}
	to, err := synth.ParseInstant(fs.to)
	if err != nil {
		return err
	}
	if !to.After(from) {
		return fmt.Errorf("snapshot: --to must be after --from")
	}
	events := tl.Between(from, to)
	enc := json.NewEncoder(w)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "%d change events between %s and %s\n",
		len(events), from.Format(time.RFC3339), to.Format(time.RFC3339))
	return nil
}

// runUI serves the browser workbench. It binds loopback only: the browser
// connects in, Synth never connects out.
func runUI(args []string) error {
	fs, err := parseFlags(args)
	if err != nil {
		return err
	}
	port := fs.port
	if port == "" {
		port = "8080"
	}
	return ui.Serve("127.0.0.1:" + port)
}

// presetList names the built-in schemas, for error messages and usage.
func presetList() string {
	names := make([]string, 0, len(synth.Presets()))
	for _, p := range synth.Presets() {
		names = append(names, string(p))
	}
	return strings.Join(names, ", ")
}
