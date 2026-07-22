// Package profile learns a schema from a SAMPLE FILE of real data (a CSV or
// JSONL export) and produces a Synth schema that reproduces its shape: column
// types, null rates, numeric ranges, and — for low-cardinality columns — the
// observed value set with its real frequencies.
//
// Synth never connects to a database, so profiling reads an export you produce
// yourself (e.g. `psql \copy users TO 'users.csv' CSV HEADER`). The result is
// synthetic data that behaves like production without ever containing it.
package profile

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bakhodir/synth/infer"
	"github.com/bakhodir/synth/schema"
)

// Result is a learned schema plus the statistics behind it.
type Result struct {
	Schema *schema.Schema
	Order  []string
	Rows   int
	Stats  map[string]*ColumnStats
}

// ColumnStats is what profiling observed for one column.
type ColumnStats struct {
	Name     string
	NonNull  int
	Nulls    int
	Distinct int
	// Min/Max are set for numeric columns.
	Min, Max float64
	Numeric  bool
	Integral bool
	// Values holds observed value counts, kept only while the column looks
	// low-cardinality (a categorical).
	Values map[string]int
	// Categorical is true when the column is treated as an enum.
	Categorical bool
}

// maxCategoricalDistinct bounds how many distinct values a column may have and
// still be reproduced as a weighted enum rather than a generated type.
const maxCategoricalDistinct = 50

// Load profiles a CSV or JSONL file, choosing the reader by extension.
func Load(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl", ".ndjson", ".json":
		return FromJSONL(f)
	default:
		return FromCSV(f)
	}
}

// FromCSV profiles CSV data whose first row is a header.
func FromCSV(r io.Reader) (*Result, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("profile: reading header: %w", err)
	}
	stats := newStats(header)
	rows := 0
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("profile: reading row %d: %w", rows+1, err)
		}
		rows++
		for i, col := range header {
			if i < len(rec) {
				observe(stats[col], rec[i])
			}
		}
	}
	return build(header, stats, rows), nil
}

// FromJSONL profiles newline-delimited JSON objects.
func FromJSONL(r io.Reader) (*Result, error) {
	dec := json.NewDecoder(r)
	var order []string
	stats := map[string]*ColumnStats{}
	rows := 0
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("profile: reading row %d: %w", rows+1, err)
		}
		rows++
		for k, v := range obj {
			if _, ok := stats[k]; !ok {
				stats[k] = &ColumnStats{Name: k, Values: map[string]int{}}
				order = append(order, k)
			}
			observe(stats[k], fmt.Sprint(valueOrEmpty(v)))
		}
	}
	return build(order, stats, rows), nil
}

func valueOrEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

func newStats(header []string) map[string]*ColumnStats {
	m := make(map[string]*ColumnStats, len(header))
	for _, h := range header {
		m[h] = &ColumnStats{Name: h, Values: map[string]int{}}
	}
	return m
}

// observe folds one raw cell into a column's statistics.
func observe(c *ColumnStats, raw string) {
	if c == nil {
		return
	}
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, "null") {
		c.Nulls++
		return
	}
	c.NonNull++
	if c.Values != nil {
		c.Values[v]++
		if len(c.Values) > maxCategoricalDistinct {
			c.Values = nil // too many distinct values to treat as an enum
		}
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		if c.NonNull == 1 || !c.Numeric {
			c.Min, c.Max = f, f
			c.Numeric = true
			c.Integral = true
		}
		if f < c.Min {
			c.Min = f
		}
		if f > c.Max {
			c.Max = f
		}
		if strings.ContainsAny(v, ".eE") {
			c.Integral = false
		}
	} else {
		c.Numeric = false
	}
}

var (
	reEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reUUID  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	rePhone = regexp.MustCompile(`^\+?\d[\d\s\-()]{6,}$`)
	reTime  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}([T ]\d{2}:\d{2})?`)
	reURL   = regexp.MustCompile(`^https?://`)
)

// build turns statistics into a schema, choosing for each column between a
// weighted enum (low cardinality), a detected format, a numeric range, or
// name-based inference.
func build(order []string, stats map[string]*ColumnStats, rows int) *Result {
	s := &schema.Schema{}
	for _, name := range order {
		c := stats[name]
		if c == nil {
			continue
		}
		c.Distinct = len(c.Values)
		f := schema.Field{Name: name, Params: map[string]string{}}

		switch {
		// A two-value true/false column becomes a bool, not an enum of the
		// strings "true" and "false". An enum would reproduce the ratio but
		// emit a string where the source had a boolean, and a consumer reading
		// the JSON gets "true" instead of true. The ratio is carried in a param
		// instead, so nothing is lost.
		case isBoolean(c):
			f.Kind = schema.KindBool
			f.Params["true"] = strconv.FormatFloat(trueShare(c), 'g', 4, 64)
		// Numeric columns stay numeric (a learned range), so generated values
		// keep their type instead of becoming enum strings.
		case c.Numeric:
			if c.Integral {
				f.Kind = schema.KindInt
				f.Params["min"] = strconv.Itoa(int(c.Min))
				f.Params["max"] = strconv.Itoa(int(c.Max))
			} else {
				f.Kind = schema.KindFloat
				f.Params["min"] = strconv.Itoa(int(c.Min))
				f.Params["max"] = strconv.Itoa(int(c.Max) + 1)
			}
		// Low-cardinality text columns are reproduced as a weighted enum, so the
		// observed category frequencies carry over exactly.
		//
		// A column whose values are a recognizable format is never a category,
		// however few distinct values were seen. This matters beyond neatness:
		// an enum spec lists its choices verbatim, so treating a short sample of
		// a UUID or email column as categorical would copy real identifiers into
		// a file meant to be committed — and profiling promises the opposite.
		// isIdentifierLike needs twenty rows to be sure; a format match does not.
		case c.Values != nil && len(c.Values) > 0 && !isIdentifierLike(c) && !hasRecognizableFormat(c):
			c.Categorical = true
			f.Kind = schema.KindEnum
			for v, n := range c.Values {
				f.Choices = append(f.Choices, v)
				f.Weights = append(f.Weights, float64(n))
			}
			sortChoices(&f)
		default:
			f.Kind = detectFormat(c)
		}
		s.Fields = append(s.Fields, f)
	}
	return &Result{Schema: s, Order: order, Rows: rows, Stats: stats}
}

// isIdentifierLike reports whether a column looks like a unique key rather than
// a category (every observed value distinct across a decent sample).
// isBoolean reports whether every value in the column is a boolean spelling.
// The spellings are the ones real exports use: SQL writes t/f, CSV from a
// spreadsheet writes TRUE/FALSE, JSON writes true/false, and some tools write
// 1/0 — but 1/0 is left alone, because an integer column that happens to hold
// only zeroes and ones is far more common than a boolean written that way.
func isBoolean(c *ColumnStats) bool {
	if len(c.Values) == 0 || len(c.Values) > 2 {
		return false
	}
	for v := range c.Values {
		if boolValue(v) == "" {
			return false
		}
	}
	return true
}

// boolValue normalises a boolean spelling, returning "" for anything else.
func boolValue(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "t", "yes", "y":
		return "true"
	case "false", "f", "no", "n":
		return "false"
	}
	return ""
}

// trueShare is the observed fraction of true values.
func trueShare(c *ColumnStats) float64 {
	var yes, total int
	for v, n := range c.Values {
		total += n
		if boolValue(v) == "true" {
			yes += n
		}
	}
	if total == 0 {
		return 0.5
	}
	return float64(yes) / float64(total)
}

// hasRecognizableFormat reports whether the column's values parse as a known
// format. Such a column is generated from its format rather than from a list of
// the values that were seen.
func hasRecognizableFormat(c *ColumnStats) bool {
	switch detectByValue(c) {
	case schema.KindUUID, schema.KindEmail, schema.KindURL, schema.KindTime, schema.KindPhone:
		return true
	}
	return false
}

func isIdentifierLike(c *ColumnStats) bool {
	return c.NonNull >= 20 && len(c.Values) == c.NonNull
}

// detectFormat inspects sample values for a recognizable format, then falls
// back to the column name, then to free text.
func detectFormat(c *ColumnStats) schema.Kind {
	if k := detectByValue(c); k != "" {
		return k
	}
	if k, matched := infer.Kind(c.Name, ""); matched {
		return k
	}
	return schema.KindLorem
}

// detectByValue reads the format from the data itself, returning "" when
// nothing matches. It is separate from detectFormat because the column-name
// fallback is a guess, and a guess must not stop a column from being treated as
// categorical.
func detectByValue(c *ColumnStats) schema.Kind {
	var sample string
	for v := range c.Values {
		sample = v
		break
	}
	switch {
	case sample == "":
		return ""
	case reUUID.MatchString(sample):
		return schema.KindUUID
	case reEmail.MatchString(sample):
		return schema.KindEmail
	case reURL.MatchString(sample):
		return schema.KindURL
	case reTime.MatchString(sample):
		return schema.KindTime
	case rePhone.MatchString(sample):
		return schema.KindPhone
	}
	return ""
}

// sortChoices keeps enum output stable across runs (map iteration is random).
func sortChoices(f *schema.Field) {
	type pair struct {
		v string
		w float64
	}
	ps := make([]pair, len(f.Choices))
	for i := range f.Choices {
		ps[i] = pair{f.Choices[i], f.Weights[i]}
	}
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && ps[j-1].v > ps[j].v; j-- {
			ps[j-1], ps[j] = ps[j], ps[j-1]
		}
	}
	for i, p := range ps {
		f.Choices[i], f.Weights[i] = p.v, p.w
	}
}
