# Streaming and determinism

## Constant-memory streaming

Records are pushed through a pipeline, never accumulated. Generating 100M rows
uses the same memory as generating 1K. Generation is sharded across cores, and
sinks batch and backpressure independently.

```go
synth.Stream[User](100_000_000).ToCSV("users.csv") // constant memory
```

## Deterministic and reproducible

A seed fully determines the output. The same seed produces byte-identical data
across runs, machines, and Go versions — so a failing CI run is reproducible
locally, and golden-file tests stay stable.

```go
synth.Make[User](1000, synth.WithSeed(42))
```

Each record is seeded independently from the base seed, so parallel generation
is byte-identical to serial output. Per-instance RNG also means no global-`rand`
mutex: parallel generation scales, and same-seed output does not depend on
worker count.

```go
synth.MakeParallel[User](10_000_000, synth.WithSeed(42))
```

!!! note "One exception"
    Tracked [unique](unique.md) columns must go through `Make`; `unique=counter`
    is the parallel-safe mode.
