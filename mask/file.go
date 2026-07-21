package mask

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Report summarizes what a masking run changed, so you can prove which columns
// were anonymized before sharing a dump.
type Report struct {
	Rows      int
	Columns   []string
	Masked    map[string]int // column -> values replaced
	Untouched []string
}

// File masks a CSV or JSONL export and writes the anonymized result. The format
// is chosen by the input extension.
func (m *Masker) File(in, out string) (*Report, error) {
	src, err := os.Open(in)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	dst, err := os.Create(out)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	switch strings.ToLower(filepath.Ext(in)) {
	case ".jsonl", ".ndjson", ".json":
		return m.jsonl(src, dst)
	default:
		return m.csv(src, dst)
	}
}

func (m *Masker) csv(r io.Reader, w io.Writer) (*Report, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("mask: reading header: %w", err)
	}
	if err := cw.Write(header); err != nil {
		return nil, err
	}
	rep := &Report{Columns: header, Masked: map[string]int{}}
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("mask: reading row %d: %w", rep.Rows+1, err)
		}
		rep.Rows++
		for i := range rec {
			if i >= len(header) {
				continue
			}
			masked := m.Value(header[i], rec[i])
			if masked != rec[i] {
				rep.Masked[header[i]]++
			}
			rec[i] = masked
		}
		if err := cw.Write(rec); err != nil {
			return nil, err
		}
	}
	rep.finish()
	return rep, nil
}

func (m *Masker) jsonl(r io.Reader, w io.Writer) (*Report, error) {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	rep := &Report{Masked: map[string]int{}}
	seen := map[string]bool{}
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("mask: reading row %d: %w", rep.Rows+1, err)
		}
		rep.Rows++
		for k, v := range obj {
			if !seen[k] {
				seen[k] = true
				rep.Columns = append(rep.Columns, k)
			}
			s, ok := v.(string)
			if !ok {
				continue // numbers/bools carry no direct identity
			}
			masked := m.Value(k, s)
			if masked != s {
				rep.Masked[k]++
			}
			obj[k] = masked
		}
		if err := enc.Encode(obj); err != nil {
			return nil, err
		}
	}
	rep.finish()
	return rep, nil
}

func (r *Report) finish() {
	for _, c := range r.Columns {
		if r.Masked[c] == 0 {
			r.Untouched = append(r.Untouched, c)
		}
	}
}
