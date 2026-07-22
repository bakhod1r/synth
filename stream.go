package synth

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/reflectfe"
	"github.com/bakhod1r/synth/schema"
)

// Streamer generates n records lazily and writes them straight to a sink,
// never holding more than one record in memory — for 100M-row runs.
type Streamer[T any] struct {
	n    int
	opts []Option
}

// Stream prepares a lazy generation of n records of type T.
func Stream[T any](n int, opts ...Option) *Streamer[T] {
	return &Streamer[T]{n: n, opts: opts}
}

func (s *Streamer[T]) engine() (*gen.Engine, *rng.Rand, []string, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range s.opts {
		o(&cfg)
	}
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, nil, nil, fmt.Errorf("synth: Stream requires a struct type, got %T", zero)
	}
	cached, _ := reflectfe.Build(rt)
	sc := &schema.Schema{Fields: append([]schema.Field(nil), cached.Fields...)}
	applyRefs(sc, cfg.refs)
	applyWeighted(sc, cfg.weighted)
	eng, err := gen.Compile(sc, cfg.locale)
	if err != nil {
		return nil, nil, nil, err
	}
	eng.Chaos = cfg.chaos
	return eng, rng.New(cfg.seed), fieldNames(rt), nil
}

// ToCSV streams records into a CSV file in constant memory.
func (s *Streamer[T]) ToCSV(path string) error {
	eng, base, cols, err := s.engine()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	defer cw.Flush()
	if err := cw.Write(cols); err != nil {
		return err
	}
	row := make([]string, len(cols))
	for i := 0; i < s.n; i++ {
		rec := eng.Record(base, i)
		for j, c := range cols {
			row[j] = fmt.Sprint(rec[c])
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// ToJSONL streams records into a JSONL file in constant memory.
func (s *Streamer[T]) ToJSONL(path string) error {
	eng, base, cols, err := s.engine()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := 0; i < s.n; i++ {
		rec := eng.Record(base, i)
		obj := make(map[string]any, len(cols))
		for _, c := range cols {
			obj[c] = rec[c]
		}
		if err := enc.Encode(obj); err != nil {
			return err
		}
	}
	return nil
}

// Each calls fn for every generated record, one at a time (constant memory).
func (s *Streamer[T]) Each(fn func(T) error) error {
	eng, base, _, err := s.engine()
	if err != nil {
		return err
	}
	for i := 0; i < s.n; i++ {
		var rec T
		scatter(&rec, eng.Record(base, i))
		if err := fn(rec); err != nil {
			return err
		}
	}
	return nil
}
