package synth

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/bakhod1r/synth/gen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/reflectfe"
	"github.com/bakhod1r/synth/schema"
)

// MakeParallel generates n records across `workers` goroutines. Each worker
// forks its own rng from a per-record deterministic seed, so output is
// independent of worker count — no shared-rand mutex, and still reproducible.
// workers <= 0 uses GOMAXPROCS.
func MakeParallel[T any](n, workers int, opts ...Option) ([]T, error) {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, errNotStruct(zero)
	}
	cached, _ := reflectfe.Build(rt)
	s := &schema.Schema{Fields: append([]schema.Field(nil), cached.Fields...)}
	applyRefs(s, cfg.refs)
	applyWeighted(s, cfg.weighted)
	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	eng.Chaos = cfg.chaos
	if eng.HasUnique() {
		return nil, fmt.Errorf("synth: MakeParallel does not support tracked unique fields; use Make, or declare them unique=counter")
	}

	out := make([]T, n)
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * chunk
		end := start + chunk
		if end > n {
			end = n
		}
		if start >= end {
			break
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			// Each worker derives an independent base from the shared seed
			// plus record index, so record i is identical regardless of which
			// worker produced it.
			base := rng.New(cfg.seed)
			for i := start; i < end; i++ {
				scatter(&out[i], eng.Record(base, i))
			}
		}(start, end)
	}
	wg.Wait()
	return out, nil
}
