// Command synth generates data from a declarative YAML spec and writes it to a
// file — no Go code required. Synth stays a pure provider: this writes files
// (CSV/JSONL/SQL), it never connects to a database.
//
// Usage:
//
//	synth gen -s users.yaml -o users.csv            # format from extension
//	synth gen -s users.yaml -f sql -o users.sql -n 100000
//	synth gen -s users.yaml -f jsonl                # stdout
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bakhodir/synth"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "gen" {
		usage()
		os.Exit(2)
	}
	fs := parseFlags(os.Args[2:])
	if fs.spec == "" {
		fmt.Fprintln(os.Stderr, "error: -s <spec.yaml> is required")
		usage()
		os.Exit(2)
	}

	spec, err := synth.LoadYAML(fs.spec)
	check(err)

	var opts []synth.Option
	if fs.seed != 0 {
		opts = append(opts, synth.WithSeed(fs.seed))
	}
	if fs.locale != "" {
		opts = append(opts, synth.WithLocale(fs.locale))
	}
	if fs.chaos > 0 {
		opts = append(opts, synth.WithChaos(fs.chaos))
	}
	recs, err := spec.Generate(opts...)
	check(err)
	if fs.n > 0 && fs.n < len(recs) {
		recs = recs[:fs.n]
	}

	out := os.Stdout
	if fs.out != "" {
		f, err := os.Create(fs.out)
		check(err)
		defer f.Close()
		out = f
	}
	format := fs.format
	if format == "" {
		format = formatFromExt(fs.out)
	}
	table := spec.Name()
	if table == "" {
		table = "data"
	}
	w := bufio.NewWriter(out)
	defer w.Flush()

	switch format {
	case "jsonl":
		writeJSONL(w, recs)
	case "sql":
		writeSQL(w, table, spec.Columns(), recs)
	default:
		writeCSV(w, spec.Columns(), recs)
	}
	if fs.out != "" {
		fmt.Fprintf(os.Stderr, "wrote %d rows to %s (%s)\n", len(recs), fs.out, format)
	}
}

type flags struct {
	spec, out, format, locale string
	seed                      uint64
	n                         int
	chaos                     float64
}

func parseFlags(args []string) flags {
	var f flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-s", "--spec":
			f.spec = next(args, &i)
		case "-o", "--out":
			f.out = next(args, &i)
		case "-f", "--format":
			f.format = next(args, &i)
		case "-l", "--locale":
			f.locale = next(args, &i)
		case "-n", "--rows":
			fmt.Sscanf(next(args, &i), "%d", &f.n)
		case "--seed":
			fmt.Sscanf(next(args, &i), "%d", &f.seed)
		case "--chaos":
			fmt.Sscanf(next(args, &i), "%g", &f.chaos)
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			os.Exit(2)
		}
	}
	return f
}

func next(args []string, i *int) string {
	*i++
	if *i >= len(args) {
		fmt.Fprintln(os.Stderr, "error: missing value for flag")
		os.Exit(2)
	}
	return args[*i]
}

func formatFromExt(path string) string {
	switch strings.TrimPrefix(filepath.Ext(path), ".") {
	case "jsonl", "json":
		return "jsonl"
	case "sql":
		return "sql"
	default:
		return "csv"
	}
}

func writeCSV(w *bufio.Writer, cols []string, recs []map[string]any) {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write(cols)
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

Usage:
  synth gen -s <spec.yaml> [-o out] [-f csv|jsonl|sql] [-n rows] [-l locale] [--seed N] [--chaos p]

Flags:
  -s, --spec     YAML data-definition file (required)
  -o, --out      output file (default: stdout); format inferred from extension
  -f, --format   csv | jsonl | sql (default: csv)
  -n, --rows     cap number of rows
  -l, --locale   override locale (e.g. uz_UZ)
      --seed     override deterministic seed
      --chaos    fraction of edge-case values (0..1)`)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
