// Package stats stores the workbench's usage counters in SQLite, so they
// survive a restart.
//
// # Why this is a separate module
//
// A SQLite driver brings ten transitive dependencies. The core library has two,
// and someone importing synth to generate a fixture should not pay for a
// database they never touch. So this implements ui.Recorder from outside and is
// wired in at the command, exactly as sink/parquet and mcp are.
//
// # What it is not
//
// This is not Synth reading or writing your data. The database holds counts —
// how many rows, how many columns, how many bytes, when — and never a generated
// value. It lives on your own machine and nothing is sent anywhere.
package stats

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bakhod1r/synth/ui"
	_ "modernc.org/sqlite" // pure Go, so no cgo and no C toolchain
)

// DB records runs in a SQLite file.
type DB struct {
	db   *sql.DB
	path string
}

// DefaultPath is where the counters live: ~/.synth/stats.db, or the OS config
// directory's equivalent.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot find a place to keep the counters: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "synth", "stats.db"), nil
}

// Open prepares the database, creating the file and its directory if needed.
// An empty path uses DefaultPath.
func Open(path string) (*DB, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("cannot create %s: %w", dir, err)
		}
	}
	// WAL keeps a reader from blocking the writer, so opening a second
	// workbench does not make the first one hang on its counters.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db: db, path: path}, nil
}

// Path is where the counters are stored, for a command that wants to say so.
func (d *DB) Path() string { return d.path }

func (d *DB) Close() error { return d.db.Close() }

// migrate creates the schema. Columns are stored rather than only cells so a
// later question — "which runs were wide?" — can still be answered; a stored
// total that threw away its parts cannot be re-derived.
func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS runs (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			at       INTEGER NOT NULL,   -- unix seconds
			name     TEXT    NOT NULL,
			rows     INTEGER NOT NULL,
			columns  INTEGER NOT NULL,
			format   TEXT    NOT NULL,
			bytes    INTEGER NOT NULL,
			millis   INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS runs_at ON runs (at);
	`)
	if err != nil {
		return fmt.Errorf("cannot prepare the counters table: %w", err)
	}
	return nil
}

// Record stores one completed run.
func (d *DB) Record(r ui.Run) error {
	_, err := d.db.Exec(
		`INSERT INTO runs (at, name, rows, columns, format, bytes, millis)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.At.Unix(), r.Name, r.Rows, r.Columns, r.Format, r.Bytes, r.Millis)
	return err
}

// Totals sums every run ever recorded.
//
// Cells is summed from rows × columns per run rather than from the totals,
// because multiplying two grand totals would count a narrow run's rows against
// a wide run's columns and give a number that means nothing.
func (d *DB) Totals() (ui.Totals, error) {
	var t ui.Totals
	t.Persistent = true
	err := d.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(rows), 0),
		       COALESCE(SUM(rows * columns), 0),
		       COALESCE(SUM(bytes), 0)
		FROM runs
	`).Scan(&t.Files, &t.Rows, &t.Cells, &t.Bytes)
	if err != nil {
		return ui.Totals{}, err
	}
	return t, nil
}

// Recent returns the last n runs, newest first — for a command that wants to
// print a history rather than a single total.
func (d *DB) Recent(n int) ([]ui.Run, error) {
	if n <= 0 {
		n = 20
	}
	rows, err := d.db.Query(
		`SELECT at, name, rows, columns, format, bytes, millis
		 FROM runs ORDER BY at DESC, id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ui.Run{}
	for rows.Next() {
		var r ui.Run
		var at int64
		if err := rows.Scan(&at, &r.Name, &r.Rows, &r.Columns, &r.Format, &r.Bytes, &r.Millis); err != nil {
			return nil, err
		}
		r.At = time.Unix(at, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

var _ ui.Recorder = (*DB)(nil)
