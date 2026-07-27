# k-anonymity checking and differential-privacy noise

Date: 2026-07-26

## Problem

Synth can mask a real export, but offers no way to measure or strengthen the
privacy of what comes out.

1. **No k-anonymity check.** After masking or generating, nothing verifies that
   each combination of quasi-identifiers (age, ZIP, gender) is shared by at
   least *k* people. A unique combination re-identifies an individual even when
   the direct identifiers are gone — the classic failure of "anonymized" data.
2. **Masking is deterministic per value.** Numeric columns are replaced with a
   stable fake, but two runs of the same value give the same output, and the
   fake carries the original's magnitude. There is no calibrated-noise option to
   bound how much any single record influences a released number.

## Scope

- k-anonymity as a `verify` check: `--k N --qi col,col` reports quasi-identifier
  groups smaller than N.
- A differential-privacy masking strategy: Laplace noise on a numeric column,
  calibrated to an epsilon budget.
- Both surfaced in the CLI; k-anonymity also exposed through the MCP `verify`
  path is out of scope for now (the MCP verify tool takes no options yet).

Out of scope: l-diversity, t-closeness (a later privacy spec); DP for aggregate
queries (Synth releases rows, not query answers).

## Phase 1 — k-anonymity check

A new check in the `verify` package, driven by two options:

```go
Options{ KAnonymity: 5, QuasiIdentifiers: []string{"age","zip","gender"} }
```

Group the rows by the tuple of QI column values; any group with fewer than `k`
members is a finding naming the group and its size. The check runs only when
`KAnonymity > 0` and at least one QI column is given. A QI column absent from
the data is an error (a typo must not pass as "no violations").

Severity: an under-k group is an **error** — the dataset fails its own privacy
claim. The finding is dataset-wide (row -1) with the offending value tuple in
`Sample`, capped like every other check.

CLI: `synth verify -i data.csv --k 5 --qi age,zip,gender`. Exit 1 on a
violation, consistent with the rest of verify.

## Phase 2 — differential-privacy noise in mask

A new masking strategy, `DP`, for numeric columns: replace the value `v` with
`v + Laplace(0, sensitivity/epsilon)`, the Laplace mechanism. The masker gains:

```go
m.Rule(mask.Rule{Column: "salary", Strategy: mask.DP, Epsilon: 1.0, Sensitivity: 10000})
```

- `Epsilon` is the per-column privacy budget; smaller means more noise.
- `Sensitivity` is how much one record can change the value — for a released
  numeric column, the column's plausible range. Required; without a bound the
  noise is undefined.
- The noise is drawn from the same HMAC-seeded RNG the masker already uses, so a
  masked file is reproducible under the key. (This is input perturbation, not
  query DP; the reproducibility is a deliberate trade for testability, and the
  docs state it.)

Non-numeric values under a `DP` rule are an error at masking time — Laplace
noise on a name is meaningless.

CLI: `synth mask ... --dp col:epsilon:sensitivity` (repeatable), e.g.
`--dp salary:1.0:10000`.

## Testing

- k-anonymity: a dataset where every QI tuple repeats ≥k times passes; a unique
  tuple is an error; a missing QI column errors; k=1 never fires.
- DP: the mean of many noised values stays near the original (unbiased); the
  spread scales as sensitivity/epsilon; the same key reproduces the same noise;
  a non-numeric column under DP errors.
- CLI: `--k`/`--qi` exit codes; `--dp` parsing and errors.
