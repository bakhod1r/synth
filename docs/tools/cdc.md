# Change events (CDC)

Generate a coherent insert/update/delete history in Debezium's envelope shape —
no database, no Kafka, just a file:

```go
synth.WriteCDC[User]("changes.jsonl", 10_000, synth.CDCConfig{
    Table: "users", UpdateRate: 0.3, DeleteRate: 0.1, Snapshot: 100,
})
```

```sh
synth cdc -s users.yaml -o changes.jsonl -n 10000 --update-rate 0.3 --delete-rate 0.1
```

A row exists before it is updated, updates carry the true `before` image, and
deleted rows are never touched again. LSNs and timestamps advance monotonically.

## Soft deletes

Deletes are hard by default (`op=d`). Pass `--soft-delete` (or
`CDCConfig.SoftDelete`) to emit them as `op=u` updates that stamp a `deleted_at`
column instead, so a consumer can be tested against either workload from one
spec:

```sh
synth cdc -s users.yaml -n 10000 --delete-rate 0.1 --soft-delete
```

## Cascading across two tables

Give a child spec and the column that references the parent. Deleting a parent
then deletes its children first, then the parent — the order a foreign key
requires:

```sh
synth cdc -s orders.yaml --child items.yaml --child-fk order_id --delete-rate 0.2
```

Inserts keep integrity (a child only ever references a parent that exists), one
LSN and clock run across both tables, and the stream is deterministic under the
seed.

## Related

[Snapshots](snapshot.md) express the same history as *state* rather than as
events — and replaying one over the other is an enforced identity.
