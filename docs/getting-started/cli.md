# CLI and YAML specs

Describe data declaratively and generate it without writing Go. Every library
capability is reachable from the command line, and every subcommand reads and
writes **files** — none of them connects to anything.

```bash
go install github.com/bakhod1r/synth/cmd/synth@latest
```

## A spec

```yaml
# users.yaml
name: users
count: 1000
locale: uz_UZ
fields:
  id:      { kind: uuid, pk: true }
  name:    { kind: name }
  email:   { kind: email, from: name }
  status:  { kind: enum, choices: [active, inactive], weights: [0.9, 0.1] }
  balance: { kind: amount, min: 0, max: 1000000, dist: lognormal, mu: 9, sigma: 1.2 }
```

```bash
synth gen -s users.yaml -o users.csv          # or -f jsonl | sql
synth gen -s users.yaml -f sql -n 100000 --seed 42
```

## Subcommands

| Command | What it does |
| --- | --- |
| `synth gen` | generate rows from a spec or a preset |
| `synth profile` | [learn a spec](../tools/profile.md) from a real export |
| `synth mask` | [anonymize](../tools/mask.md) a real dump |
| `synth verify` | [audit](../tools/verify.md) an existing dataset |
| `synth diff` | [compare two datasets' shape](../tools/diff.md) |
| `synth cdc` | generate a [change-event history](../tools/cdc.md) |
| `synth snapshot` | the table [as it stood at an instant](../tools/snapshot.md) |
| `synth ui` | the [browser workbench](../workbench.md), loopback only |

## Flags

```
  -s, --spec       YAML data-definition file
  -i, --in         input file to profile or mask
  -o, --out        output file (default: stdout)
  -f, --format     csv | jsonl | sql | parquet | pgcopy | pgcopy-binary
  -n, --rows       number of rows or events
  -l, --locale     locale (e.g. uz_UZ)
      --name       table name for a profiled spec
      --key        masking key; the same key keeps foreign keys joinable
      --seed       deterministic seed
      --chaos      fraction of edge-case values (0..1)
      --preset     built-in schema
      --unmasked   return raw card numbers and identifiers instead of masked
      --at, --from, --to   instants for snapshot (2026-01-01 or RFC 3339)
      --churn      mean updates per row over the window
      --ref        foreign key to resolve, as col=parent.csv:key (repeatable)
      --fk         foreign key to fill from a parent file, col=parent.csv:key (repeatable)
      --append     extend the output file instead of overwriting it
      --update-rate, --delete-rate, --snapshot   CDC history shape
      --soft-delete   emit a delete as an update stamping deleted_at
      --child, --child-fk   cascade CDC: child spec and its FK column to the parent
      --k, --qi    k-anonymity: require each --qi col,col combination k+ times
      --dp         Laplace-noise a numeric column while masking, col:epsilon:sensitivity
```

## Presets

A built-in schema, for when you want rows now and a spec later:

```bash
synth gen --preset user -n 100 -o users.csv
```

`employee`, `event`, `order`, `patient`, `payment`, `product`, `transaction`,
`user`.

!!! note "`--preset user` is slower than it looks"
    It generates about 5,000 rows/sec, where the same shape without its
    `password_hash` column runs at roughly 500,000. That column runs PBKDF2 at
    1,000 iterations — a key derivation function, meant to be expensive. See
    [Benchmarks](../reference/benchmarks.md).

## Typical pipelines

```bash
# Learn a spec from a real export, then generate from the spec forever after.
synth profile -i prod_export.csv -o users.yaml
synth gen -s users.yaml -n 1000000 -o fake_users.csv

# Anonymize a real dump. The same --key across files keeps foreign keys joinable.
synth mask -i prod_export.csv -o safe.csv --key "$MASK_KEY"

# Generate a coherent insert/update/delete history in Debezium's envelope shape.
synth cdc -s users.yaml -o changes.jsonl -n 10000 --update-rate 0.3 --delete-rate 0.1
```

`synth mask` refuses to run without `--key` (an unkeyed run is not reproducible)
and refuses to write over its own input.
