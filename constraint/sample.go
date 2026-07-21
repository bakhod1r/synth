package constraint

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultSample is how many rows mining reads. Invariants are structural, so
// they show up in the first few thousand rows; reading a whole multi-gigabyte
// export to confirm what the first 50,000 rows already say would trade a lot
// of memory for nothing.
const DefaultSample = 50_000

// LoadSample reads up to max rows from a CSV or JSONL export. Synth reads the
// file only — it never connects to whatever produced it.
func LoadSample(path string, max int) ([]map[string]any, error) {
	if max <= 0 {
		max = DefaultSample
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl", ".ndjson", ".json":
		return sampleJSONL(f, max)
	default:
		return sampleCSV(f, max)
	}
}

func sampleCSV(r io.Reader, max int) ([]map[string]any, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("constraint: reading header: %w", err)
	}
	var rows []map[string]any
	for len(rows) < max {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("constraint: reading row %d: %w", len(rows)+1, err)
		}
		m := make(map[string]any, len(header))
		for i, col := range header {
			if i < len(rec) {
				m[col] = rec[i]
			}
		}
		rows = append(rows, m)
	}
	return rows, nil
}

func sampleJSONL(r io.Reader, max int) ([]map[string]any, error) {
	dec := json.NewDecoder(r)
	var rows []map[string]any
	for len(rows) < max {
		var m map[string]any
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("constraint: reading row %d: %w", len(rows)+1, err)
		}
		rows = append(rows, m)
	}
	return rows, nil
}
