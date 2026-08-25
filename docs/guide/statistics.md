# Statistical shape

Real data isn't uniform. Synth draws from distributions so your test data
stresses the same paths production does.

Via tags:

```go
type Txn struct {
    Amount   float64 `synth:"amount,dist=lognormal,mu=10,sigma=1"` // long tail
    Status   string  `synth:"enum,choices=settled|pending|failed,weights=0.94|0.05|0.01"`
    Category string  `synth:"enum,choices=a|b|c|d,dist=zipf,s=1.2"` // hot keys
}
```

Or in code:

```go
synth.Make[Txn](1_000_000, synth.Weighted("Status", map[string]float64{
    "settled": 0.94, "pending": 0.05, "failed": 0.01,
}))
```

Distributions: `normal`, `lognormal`, `exp` (numeric fields) and `zipf` /
explicit `weights` (enums).

Long tails, hot keys, and skew are what break partitioning and query planners —
uniform fakers never surface those bugs.

## Correlated fields

Numeric columns don't have to be independent. `derive` makes one a linear
function of another in the same row, so a scatter plot has a shape instead of a
cloud:

```yaml
age:    { kind: int, min: 25, max: 65 }
income: { kind: float, derive: age, slope: 1200, intercept: 20000, noise: 0.1 }
```

## Time series

`kind: timeseries` makes a column follow a curve over time — trend plus a
seasonal cycle plus noise — for metrics and IoT data:

```yaml
ts:  { kind: time, min: 2026-01-01T00:00:00Z, max: 2026-02-01T00:00:00Z }
cpu: { kind: timeseries, axis: ts, base: 40, trend: 0.5, amplitude: 20, period: 24h, noise: 3, min: 0, max: 100 }
```

Both are pure functions of the same row, so they generate in the same streaming,
deterministic pass as everything else.

## Learning the shape instead of declaring it

[Profiling](../tools/profile.md) reads the distribution off a real export, and
[constraint mining](../tools/constraints.md) learns the cross-column invariants
a per-column distribution cannot see.
