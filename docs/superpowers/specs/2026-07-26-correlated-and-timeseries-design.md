# Correlated numerics and time-series fields

Date: 2026-07-26

## Problem

Synth draws numeric fields from distributions, but every field is drawn
independently. Two shapes of real data that this cannot produce:

1. **Correlated numerics.** `income` rises with `age`; `weight` tracks `height`.
   Today each is an independent draw, so a scatter plot of the two is a
   featureless cloud — exactly the pattern real analytics code is written to
   find and test data never exercises.
2. **Time series.** A metric or sensor reading is a function of time: a daily
   cycle, a slow upward trend, noise on top. Synth has temporal *causality*
   (event ordering) but no way to make a numeric column follow a seasonal
   curve, so metrics and IoT test data cannot be generated at all.

## Constraint that shapes both

Rows are generated independently — each record index forks its own rng and
sees no other row. So neither feature may depend on aggregate state (the
dataset's min timestamp, the mean of a column). Both must be **pure functions
of values already in the same row**, read through the existing `Sibling`
mechanism. This is what keeps generation streaming and byte-identical under a
seed.

## Scope

In scope, one spec, sequential phases:

- `derive` — a numeric field as a linear function of another field in the same
  row, plus gaussian noise.
- `kind: timeseries` — a numeric field as `base + trend·t + amplitude·sin(2π·t/period) + noise`,
  where `t` comes from a named timestamp field in the same row.

Out of scope (dropped in brainstorming):

- Covariance matrices / multivariate joint sampling. The linear `derive` form
  covers the common `income~age` case at a fraction of the config.
- Correlation-coefficient joint draws — force both fields to be normal, which
  fights `min`/`max` bounds.
- Multiplicative seasonality, spikes, anomalies. The additive
  trend+seasonality+noise decomposition is enough for metrics/IoT; richer
  shapes wait for a need.

## Phase 1 — `derive`

A numeric field (`kind: float` or `kind: int`) gains three params:

```yaml
income:
  kind: float
  derive: age        # the field this one is a function of
  slope: 1200        # value = slope*age + intercept + noise
  intercept: 20000
  noise: 0.15        # gaussian sd as a fraction of the deterministic value
  min: 0             # existing bounds still clamp the result
```

`value = slope * sibling + intercept`, then a gaussian perturbation with
standard deviation `noise * |value|` is added (so noise scales with
magnitude), then the existing `min`/`max` clamp applies. `noise` defaults to 0
— an exact linear relation when omitted.

The referenced field must resolve to a number; a non-numeric or missing sibling
is a compile-time error. `derive` adds a dependency edge so the sibling is
generated first — the same topological ordering `from` and `match` already use.

`int` fields round the result.

## Phase 2 — `kind: timeseries`

```yaml
ts:
  kind: time                 # the axis: a real timestamp per row
cpu:
  kind: timeseries
  axis: ts                   # which field is the time axis
  base: 40                   # value at t = 0
  trend: 0.5                 # added per day since `start`
  period: 24h                # one full seasonal cycle
  amplitude: 20              # peak-to-mean swing
  noise: 3                   # gaussian sd, absolute
  start: 2026-01-01          # origin for t; default 2026-01-01T00:00:00Z
```

For a row whose `axis` timestamp is `T`:

```
t_days    = (T - start) / 24h
t_seconds = (T - start) in seconds
value = base + trend*t_days + amplitude*sin(2π * t_seconds / period_seconds) + noise
```

Every term is local to the row — `T` is the row's own axis value, `start` is a
fixed origin — so no cross-row pass is needed and the result is deterministic
under the seed. `period` is a Go duration (`24h`, `168h`, `8760h`). `axis` adds
a dependency edge so the timestamp is generated first, and must resolve to a
`time.Time`; anything else is a compile-time error.

`min`/`max`, if given, clamp — a CPU percentage stays in `[0,100]`.

## Wiring

Both `derive` and `axis` become dependency edges in `topoOrder` (today it reads
`From` and `Match`; it will also read `Params["derive"]` and `Params["axis"]`).
The YAML frontend already funnels unknown keys into `Params`, so `slope`,
`intercept`, `noise`, `base`, `trend`, `period`, `amplitude`, `start` need no
struct changes — only `timeseries` as a registered kind and the two providers.

## Testing

- `derive`: with `noise: 0`, values equal `slope*sibling + intercept` exactly;
  with noise, the mean over many rows stays near the line and the spread scales
  with magnitude; `min`/`max` clamp; a non-numeric or missing `derive` target
  errors at compile time; `int` rounds.
- `timeseries`: a pure sine (`trend: 0`, `noise: 0`) hits its known value at
  `t=0`, quarter period and half period; `trend` alone is linear in days;
  `start` shifts the origin; a non-time `axis` errors; determinism holds.
- Topo: `derive`/`axis` order the referenced field first; a cycle
  (`a derive b`, `b derive a`) is reported, not looped.
