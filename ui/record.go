package ui

import (
	"sync"
	"time"
)

// Usage counters for the workbench.
//
// The counters live behind an interface so the core module can show them
// without taking on a database driver. The default implementation counts in
// memory and forgets everything when the process exits; the ui/stats module
// implements the same interface over SQLite, so the numbers survive restarts
// for anyone who wants that. See ui/stats/README.md.
//
// Nothing here is reported anywhere. The counters exist so you can see how much
// data you have generated, and they are as local as the rest of the workbench.

// Run describes one completed generation.
type Run struct {
	At      time.Time `json:"at"`
	Name    string    `json:"name"`
	Rows    int       `json:"rows"`
	Columns int       `json:"columns"`
	Format  string    `json:"format"`
	Bytes   int64     `json:"bytes"`
	Millis  int64     `json:"ms"`
}

// Cells is rows × columns — the count that actually says how much data was
// produced. Ten thousand rows of two columns and ten thousand of two hundred
// are the same number of rows and nothing alike.
func (r Run) Cells() int64 { return int64(r.Rows) * int64(r.Columns) }

// Totals is what the page shows.
type Totals struct {
	Files int   `json:"files"`
	Rows  int64 `json:"rows"`
	Cells int64 `json:"cells"`
	Bytes int64 `json:"bytes"`
	// Persistent reports whether these survive a restart. The page says so
	// rather than letting a number that silently resets look like data loss.
	Persistent bool `json:"persistent"`
}

// Recorder stores completed runs and reports the totals.
//
// An implementation must never fail a generation: Record's error is logged and
// dropped by the caller. Counting is a convenience, and losing a count is not
// worth losing the data someone asked for.
type Recorder interface {
	Record(Run) error
	Totals() (Totals, error)
}

// memoryRecorder is the default: counts for this session only.
type memoryRecorder struct {
	mu     sync.Mutex
	totals Totals
}

func newMemoryRecorder() *memoryRecorder { return &memoryRecorder{} }

func (m *memoryRecorder) Record(r Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totals.Files++
	m.totals.Rows += int64(r.Rows)
	m.totals.Cells += r.Cells()
	m.totals.Bytes += r.Bytes
	return nil
}

func (m *memoryRecorder) Totals() (Totals, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.totals, nil // Persistent stays false: this forgets on exit.
}
