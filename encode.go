package synth

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// WriteCSV writes records to a CSV file. Column order and header come from the
// struct's field order.
func WriteCSV[T any](path string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return encodeCSV(f, records)
}

func encodeCSV[T any](w io.Writer, records []T) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	var zero T
	cols := fieldNames(reflect.TypeOf(zero))
	if err := cw.Write(cols); err != nil {
		return err
	}
	for i := range records {
		rv := reflect.ValueOf(records[i])
		row := make([]string, len(cols))
		for j := range cols {
			row[j] = fmt.Sprint(rv.Field(j).Interface())
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteJSONL writes one JSON object per line.
func WriteJSONL[T any](path string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return encodeJSONL(f, records)
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
	defer f.Close()
	return encodeSQL(f, table, records)
}

func encodeSQL[T any](w io.Writer, table string, records []T) error {
	var zero T
	cols := fieldNames(reflect.TypeOf(zero))
	colList := strings.Join(cols, ", ")
	for i := range records {
		rv := reflect.ValueOf(records[i])
		vals := make([]string, len(cols))
		for j := range cols {
			vals[j] = sqlValue(rv.Field(j).Interface())
		}
		if _, err := fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", ")); err != nil {
			return err
		}
	}
	return nil
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
		s := fmt.Sprint(x)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}

func fieldNames(rt reflect.Type) []string {
	var cols []string
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).PkgPath != "" {
			continue
		}
		cols = append(cols, rt.Field(i).Name)
	}
	return cols
}
