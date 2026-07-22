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

// LoadSample reads up to max rows from a CSV or JSONL export, picking the
// format from the file extension. Synth reads the file only — it never connects
// to whatever produced it.
func LoadSample(path string, max int) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSample(f, formatOf(path), max)
}

// formatOf reads the format from a file extension, falling back to CSV. The
// fallback is deliberate: an export named .dat or .txt is almost always CSV,
// and refusing it would break callers that worked before ReadSample existed.
func formatOf(path string) string {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".jsonl", ".ndjson", ".json":
		return "jsonl"
	default:
		return "csv"
	}
}

// ReadSample is LoadSample for an already-open reader, with the format named
// rather than inferred. Callers that must not touch the filesystem — the MCP
// server is one — use this; LoadSample delegates to it so the two cannot drift
// apart.
//
// format is "csv" (the default) or "jsonl"/"ndjson", with or without a leading
// dot, so a file extension can be passed straight through.
func ReadSample(r io.Reader, format string, max int) ([]map[string]any, error) {
	if max <= 0 {
		max = DefaultSample
	}
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "jsonl", "ndjson", "json":
		return sampleJSONL(r, max)
	case "", "csv":
		return sampleCSV(r, max)
	default:
		return nil, fmt.Errorf("unknown format %q — use csv or jsonl", format)
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
