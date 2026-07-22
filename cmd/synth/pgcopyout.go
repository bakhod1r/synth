package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/pgcopy"
)

// writePgCopy writes rows in one of the two COPY formats and, alongside the
// data file, the CREATE TABLE that receives it.
//
// The DDL is not optional. Binary COPY sends no type names, so the table's
// column types are what the server decodes the bytes as; a table built by hand
// that differs anywhere means a rejected file or a silently misread value. The
// text format is more forgiving, but the table still has to exist, and writing
// it costs nothing.
func writePgCopy(w io.Writer, format, table string, spec *synth.YAMLSpec,
	recs []map[string]any, outPath string) error {

	binary := format == "pgcopy-binary"
	if binary && outPath == "" {
		return fmt.Errorf("gen: pgcopy-binary writes binary data, so it needs " +
			"-o <file> rather than stdout")
	}

	var cw pgcopy.Writer
	if binary {
		cw = pgcopy.NewBinary(w, spec.Columns())
	} else {
		cw = pgcopy.NewText(w, spec.Columns())
	}
	for _, r := range recs {
		if err := cw.WriteRow(r); err != nil {
			return err
		}
	}
	if err := cw.Close(); err != nil {
		return err
	}
	if outPath == "" {
		return nil
	}

	ddl := pgcopy.DDL(table, spec.Schema(), absPath(outPath), binary)
	ddlPath := outPath + ".sql"
	if err := os.WriteFile(ddlPath, []byte(ddl), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s — load with: psql -f %s\n", ddlPath, ddlPath)
	return nil
}

// absPath makes the path in the COPY command absolute, since the server reads
// it relative to its own working directory and not the shell's.
func absPath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return p
	}
	return wd + "/" + p
}
