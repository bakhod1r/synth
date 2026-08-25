# Learn from real data (profiling)

Point Synth at an **export** of a real table and it learns the shape — column
types, numeric ranges, null rates, and the real frequency of each category —
then generates synthetic rows that behave the same. Synth never connects to your
database; you produce the sample yourself:

```bash
psql -c "\copy (SELECT * FROM users LIMIT 10000) TO 'sample.csv' CSV HEADER"
```

```go
p, _ := synth.Profile("sample.csv")
rows, _ := p.Generate(1_000_000)   // same distribution, none of the real data
```

From the CLI, write the learned spec out and generate from it forever after:

```bash
synth profile -i prod_export.csv -o users.yaml --name users
synth gen -s users.yaml -n 1000000 -o fake_users.csv
```

## What is kept, and what is not

Low-cardinality columns keep their observed split (e.g. 80% active / 15%
inactive / 5% banned). Identifier-like columns are never echoed back, so real
values cannot leak into the output.

## What per-column profiling cannot see

That a `total` must agree with its line items, or that a refunded order must
carry a refund timestamp. Those are mined separately — see
[Constraint mining](constraints.md).
