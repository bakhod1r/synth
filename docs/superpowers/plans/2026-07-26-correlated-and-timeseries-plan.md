# Plan: correlated numerics and time-series fields

Spec: `docs/superpowers/specs/2026-07-26-correlated-and-timeseries-design.md`

TDD per phase, each committed on its own.

## Phase 1 — topo edges for derive/axis

`topoOrder` reads `Params["derive"]` and `Params["axis"]` as dependency edges,
alongside `From` and `Match`. Small, lands first because both features need it.

Tests: a field with `derive: other` is generated after `other`; a cycle errors.

## Phase 2 — `derive`

1. In the numeric providers (float, int), when `Params["derive"]` is set:
   read the sibling, compute `slope*sibling + intercept`, add gaussian noise
   with sd `noise*|value|`, clamp to min/max, round for int.
2. Compile-time check: `derive` target exists and is numeric.

Tests: exact line with `noise: 0`; clamp; int rounding; non-numeric target
errors; mean stays near the line with noise.

## Phase 3 — `kind: timeseries`

1. Register `KindTimeSeries`. Provider reads the `axis` sibling (time.Time),
   computes `base + trend*t_days + amplitude*sin(2π*t_sec/period) + noise`,
   clamps to min/max.
2. `start` param parsed as an instant (reuse the spec's date parsing),
   defaulting to 2026-01-01Z. `period` parsed as a Go duration.
3. Compile-time check: `axis` exists and resolves to time.Time.

Tests: pure sine at t=0/quarter/half period; linear trend in days; start shift;
non-time axis errors; determinism.

## Phase 4 — docs

README (a correlated example, a metrics/IoT time-series example), CHANGELOG,
and the YAML field reference in the spec docs.
