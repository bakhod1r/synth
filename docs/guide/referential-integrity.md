# Referential integrity

Records are generated as a graph, not a list. Declare a relation once and every
child row points at a parent that actually exists.

```go
users := synth.Users(10_000)
synth.Orders(500_000, synth.BelongsTo(users, "user_id"))
```

Or from your own types:

```go
users  := synth.Make[User](10_000, synth.WithSeed(1))
orders := synth.Make[Order](500_000, synth.Ref(users, "UserID")) // every FK is real
```

Foreign keys resolve. Cardinality is controllable (`OneToMany`, `Weighted`).
Load the exported parent table into Postgres with your own loader and the child
table's FK constraints pass on the first try.

## Across runs

Keys resolve across runs too, not only within one process. Generate the parent
today, the child next week, and point the child at the keys already on disk:

```sh
synth gen -s users.yaml  -o users.csv  -n 10000
synth gen -s orders.yaml -o orders.csv -n 500000 --fk user_id=users.csv:id
```

## Appending

`--append` extends a dataset without regenerating it — a sidecar tracks how much
exists so the new rows never repeat the old ones or their primary keys:

```sh
synth gen -s users.yaml -o users.csv -n 1000000 --append   # a million more, no collisions
```

!!! note
    `--append` supports CSV, JSONL and SQL, and needs `-o <file>` rather than
    stdout. A format with its own header and trailer — Parquet, binary `COPY` —
    cannot simply be extended, and Synth rejects the combination instead of
    writing an unreadable file. See [Outputs](outputs.md).

## Checking it afterwards

`synth verify` runs the same rule backwards over data somebody else produced,
reading parent tables from their own files:

```sh
synth verify -i orders.csv --ref user_id=users.csv:id
```

See [Verify](../tools/verify.md).
