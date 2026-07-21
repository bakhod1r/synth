package synth

import (
	"context"
	"time"

	"github.com/bakhodir/synth/internal/rng"
)

// RateConfig paces a generated stream so it arrives over wall-clock time,
// the way a real event source would. Use it to drive streaming tests, load
// tests and consumer back-pressure experiments without a broker.
type RateConfig struct {
	// PerSecond is the target event rate. Values <= 0 mean "as fast as
	// possible" (no pacing).
	PerSecond float64
	// Burst is how many events are emitted per tick. Larger bursts mean fewer,
	// coarser wakeups; 0 defaults to 1.
	Burst int
	// Jitter (0..1) randomizes each interval by up to ±Jitter, so arrivals look
	// like real traffic instead of a metronome.
	Jitter float64
	// Total caps how many events are emitted. 0 means run until the context is
	// cancelled.
	Total int
}

// RateStream emits records at a wall-clock rate.
type RateStream[T any] struct {
	cfg  RateConfig
	opts []Option
}

// Rate prepares a paced stream of T. Nothing is generated until Run is called.
//
//	synth.Rate[Event](synth.RateConfig{PerSecond: 500, Total: 10_000}).
//	    Run(ctx, func(e Event) error { return producer.Send(e) })
func Rate[T any](cfg RateConfig, opts ...Option) *RateStream[T] {
	return &RateStream[T]{cfg: cfg, opts: opts}
}

// Run generates records and hands each to fn at the configured rate. It stops
// on the first error from fn, when Total is reached, or when ctx is cancelled
// (returning ctx.Err()).
func (r *RateStream[T]) Run(ctx context.Context, fn func(T) error) error {
	burst := r.cfg.Burst
	if burst < 1 {
		burst = 1
	}
	// Generation itself is lazy and constant-memory; pacing only decides when
	// the next batch is handed over.
	gen := Stream[T](batchCap(r.cfg.Total), r.opts...)
	eng, base, _, err := gen.engine()
	if err != nil {
		return err
	}

	var interval time.Duration
	if r.cfg.PerSecond > 0 {
		interval = time.Duration(float64(time.Second) * float64(burst) / r.cfg.PerSecond)
	}
	jitterRNG := newJitterSource(r.opts...)

	emitted := 0
	next := time.Now()
	for {
		if r.cfg.Total > 0 && emitted >= r.cfg.Total {
			return nil
		}
		for i := 0; i < burst; i++ {
			if r.cfg.Total > 0 && emitted >= r.cfg.Total {
				return nil
			}
			var rec T
			scatter(&rec, eng.Record(base, emitted))
			if err := fn(rec); err != nil {
				return err
			}
			emitted++
		}
		if interval <= 0 {
			// Unpaced: still honor cancellation between bursts.
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}
		step := interval
		if r.cfg.Jitter > 0 {
			// ±Jitter around the interval keeps arrivals irregular but keeps
			// the long-run average rate on target.
			f := 1 + r.cfg.Jitter*(2*jitterRNG.Float64()-1)
			step = time.Duration(float64(interval) * f)
		}
		next = next.Add(step)
		wait := time.Until(next)
		if wait <= 0 {
			// Behind schedule: don't try to catch up in a tight loop, just
			// resync so the stream degrades gracefully under load.
			next = time.Now()
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// newJitterSource derives the pacing RNG from the same options as the data, so
// a seeded run has a reproducible arrival pattern too.
func newJitterSource(opts ...Option) *rng.Rand {
	cfg := config{seed: uint64(time.Now().UnixNano())}
	for _, o := range opts {
		o(&cfg)
	}
	return rng.New(cfg.seed ^ 0x9a7e)
}

// batchCap gives the lazy generator a bound; an unbounded run still works
// because records are produced one at a time from the record index.
func batchCap(total int) int {
	if total > 0 {
		return total
	}
	return 1 << 30
}
