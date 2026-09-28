package synth

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// WriteCSV writes records to a CSV file. Column order and header come from the
// struct's field order.
func WriteCSV[T any](path string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return closeAfter(f, encodeCSV(f, records))
}

// closeAfter closes f and returns the first error. A Close error is a lost
// write too: on some filesystems it is where a full disk is reported.
func closeAfter(f *os.File, err error) error {
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func encodeCSV[T any](w io.Writer, records []T) error {
	cw := csv.NewWriter(w)
	var zero T
	idx, cols := exportedFields(reflect.TypeOf(zero))
	if err := cw.Write(cols); err != nil {
		return err
	}
	for i := range records {
		rv := reflect.ValueOf(records[i])
		row := make([]string, len(cols))
		for j, fi := range idx {
			row[j] = fmt.Sprint(rv.Field(fi).Interface())
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	// csv.Writer buffers: the last block's write error only surfaces here.
	cw.Flush()
	return cw.Error()
}

// WriteJSONL writes one JSON object per line.
func WriteJSONL[T any](path string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return closeAfter(f, encodeJSONL(f, records))
}

func encodeJSONL[T any](w io.Writer, records []T) error {
	enc := json.NewEncoder(w)
	for i := range records {
		if err := enc.Encode(records[i]); err != nil {
			return err
		}
	}
	return nil
}

// WriteSQL writes INSERT statements for the given table. This produces a
// FILE — Synth never connects to a database; run the file with your own tool.
func WriteSQL[T any](path, table string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return closeAfter(f, encodeSQL(f, table, records))
}

func encodeSQL[T any](w io.Writer, table string, records []T) error {
	var zero T
	idx, cols := exportedFields(reflect.TypeOf(zero))
	colList := strings.Join(cols, ", ")
	for i := range records {
		rv := reflect.ValueOf(records[i])
		vals := make([]string, len(cols))
		for j, fi := range idx {
			vals[j] = sqlValue(rv.Field(fi).Interface())
		}
		if _, err := fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", ")); err != nil {
			return err
		}
	}
	return nil
}

func sqlValue(v any) string {
	// A pointer field stands for its target, and a nil one for NULL; printed
	// as-is it was a memory address.
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "NULL"
		}
		return sqlValue(rv.Elem().Interface())
	}
	switch x := v.(type) {
	case nil:
		return "NULL"
	case float32:
		return sqlFloat(float64(x))
	case float64:
		return sqlFloat(x)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(x)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	default:
		s := fmt.Sprint(x)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}

func fieldNames(rt reflect.Type) []string {
	_, cols := exportedFields(rt)
	return cols
}

// exportedFields returns the index and name of each exported field. Values
// must be read by these indices: reading field j for column j shifted every
// column after an unexported field and panicked on reaching it.
func exportedFields(rt reflect.Type) (idx []int, cols []string) {
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).PkgPath != "" {
			continue
		}
		idx = append(idx, i)
		cols = append(cols, rt.Field(i).Name)
	}
	return idx, cols
}

// sqlFloat writes a float literal. NaN and ±Inf have no SQL literal, so they
// are written as NULL rather than as a statement the database rejects.
func sqlFloat(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "NULL"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
