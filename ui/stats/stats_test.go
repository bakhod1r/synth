package stats

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bakhod1r/synth/ui"
)

func open(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestTotals(t *testing.T) {
	db := open(t)
	now := time.Now()
	for _, r := range []ui.Run{
		{At: now, Name: "a", Rows: 100, Columns: 5, Format: "csv", Bytes: 1000},
		{At: now, Name: "b", Rows: 50, Columns: 20, Format: "jsonl", Bytes: 2000},
	} {
		if err := db.Record(r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.Totals()
	if err != nil {
		t.Fatal(err)
	}
	if got.Files != 2 || got.Rows != 150 || got.Bytes != 3000 {
		t.Fatalf("got %+v", got)
	}
	// 100×5 + 50×20 = 1500. Multiplying the grand totals instead would give
	// 150×25 = 3750 — a narrow run's rows counted against a wide run's columns.
	if got.Cells != 1500 {
		t.Fatalf("cells = %d, want 1500", got.Cells)
	}
	if !got.Persistent {
		t.Fatal("a database-backed recorder reports itself as not persistent")
	}
}

// The point of this module: the numbers outlive the process.
func TestTotalsSurviveReopening(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Record(ui.Run{At: time.Now(), Name: "a", Rows: 42, Columns: 3, Bytes: 99}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	got, err := again.Totals()
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows != 42 || got.Files != 1 {
		t.Fatalf("counters did not survive a restart: %+v", got)
	}
}

// An empty database must report zeroes, not an error. A fresh install opening
// the workbench for the first time is the common case.
func TestEmptyDatabaseReportsZero(t *testing.T) {
	got, err := open(t).Totals()
	if err != nil {
		t.Fatal(err)
	}
	if got.Files != 0 || got.Rows != 0 || got.Cells != 0 {
		t.Fatalf("a fresh database is not empty: %+v", got)
	}
}

// Opening twice must work: a second workbench on the same database is a normal
// thing to do, and WAL is configured so neither blocks.
func TestTwoConnectionsToOneFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	a, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	if err := a.Record(ui.Run{At: time.Now(), Rows: 10, Columns: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := b.Totals()
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows != 10 {
		t.Fatalf("the second connection does not see the first's write: %+v", got)
	}
}

func TestRecent(t *testing.T) {
	db := open(t)
	base := time.Now()
	for i := 0; i < 5; i++ {
		if err := db.Record(ui.Run{
			At: base.Add(time.Duration(i) * time.Minute), Name: "r", Rows: i, Columns: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.Recent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d runs, want 3", len(got))
	}
	// Newest first, so a history reads top-down.
	if got[0].Rows != 4 || got[2].Rows != 2 {
		t.Fatalf("wrong order: %d then %d", got[0].Rows, got[2].Rows)
	}
}

// Migrating an existing database must not fail or lose anything.
func TestMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	for i := 0; i < 3; i++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		if err := db.Record(ui.Run{At: time.Now(), Rows: 1, Columns: 1}); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
	db, _ := Open(path)
	defer db.Close()
	got, _ := db.Totals()
	if got.Files != 3 {
		t.Fatalf("reopening lost runs: %+v", got)
	}
}

// The database must hold counts and never a generated value. If a field of
// actual data ever appears in this schema, this test is where it gets caught.
func TestSchemaStoresNoGeneratedValues(t *testing.T) {
	db := open(t)
	rows, err := db.db.Query(`SELECT name FROM pragma_table_info('runs')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	allowed := map[string]bool{
		"id": true, "at": true, "name": true, "rows": true,
		"columns": true, "format": true, "bytes": true, "millis": true,
	}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatal(err)
		}
		if !allowed[col] {
			t.Errorf("column %q is not a counter — the database must never hold generated data", col)
		}
	}
}
