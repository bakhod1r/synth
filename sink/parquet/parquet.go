// Package parquet writes generated data as Apache Parquet files — the format
// analytics and data-lake tooling expects.
//
// This lives in its own module so its dependency stays optional: the core
// synth library still needs only google/uuid and yaml.v3.
//
//	go get github.com/bakhod1r/synth/sink/parquet
//
// Like every Synth output, this writes a FILE. Uploading it to S3, MinIO or a
// warehouse is your loader's job.
package parquet

import (
	"fmt"
	"os"
	"time"

	pq "github.com/parquet-go/parquet-go"
)

// WriteStructs writes a slice of Go structs to a Parquet file. The Parquet
// schema is derived from T's fields.
func WriteStructs[T any](path string, records []T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := pq.NewGenericWriter[T](f)
	if _, err := w.Write(records); err != nil {
		return fmt.Errorf("parquet: writing rows: %w", err)
	}
	return w.Close()
}

// WriteRows writes map-shaped records (what the YAML, DDL, OpenAPI, Protobuf
// and profiling frontends produce). Column order is taken from cols, and the
// Parquet schema is inferred from the first row's value types.
func WriteRows(path string, cols []string, rows []map[string]any) error {
	if len(rows) == 0 {
		return fmt.Errorf("parquet: no rows to write")
	}
	group := pq.Group{}
	for _, c := range cols {
		group[c] = nodeFor(rows[0][c])
	}
	schema := pq.NewSchema("synth", group)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := pq.NewGenericWriter[map[string]any](f, schema)
	batch := make([]map[string]any, len(rows))
	for i, r := range rows {
		norm := make(map[string]any, len(cols))
		for _, c := range cols {
			norm[c] = normalize(r[c])
		}
		batch[i] = norm
	}
	if _, err := w.Write(batch); err != nil {
		return fmt.Errorf("parquet: writing rows: %w", err)
	}
	return w.Close()
}

// nodeFor maps a Go value to a Parquet leaf type. Everything Synth generates
// reduces to string, int64, float64 or bool.
func nodeFor(v any) pq.Node {
	switch normalize(v).(type) {
	case int64:
		return pq.Leaf(pq.Int64Type)
	case float64:
		return pq.Leaf(pq.DoubleType)
	case bool:
		return pq.Leaf(pq.BooleanType)
	default:
		return pq.String()
	}
}

// normalize converts a generated value into one of the four types the Parquet
// schema uses. Times become RFC 3339 strings so they stay human-readable in
// query engines without a timezone-handling surprise.
func normalize(v any) any {
	switch x := v.(type) {
	case nil:
		return ""
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float32:
		return float64(x)
	case float64:
		return x
	case bool:
		return x
	case string:
		return x
	case time.Time:
		return x.Format(time.RFC3339)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}
