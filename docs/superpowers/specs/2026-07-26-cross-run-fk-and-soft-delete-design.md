# Cross-run foreign keys, append mode, and soft deletes

Date: 2026-07-26

## Problem

Three gaps, all about generating data across more than one run or with a
fuller mutation shape.

1. **Foreign keys only resolve within one process.** `synth.Ref` takes an
   in-memory parent slice, so a child table can point at a parent only when
   both are generated in the same program. Generating `users.csv` today and
   `orders.csv` next week — the normal case for a CLI — has no way to make the
   orders point at real users.
2. **There is no way to add rows to an existing dataset.** Re-running `gen`
   overwrites the file, and the same seed reproduces the same rows, so a second
   run cannot extend the first: identical primary keys, identical everything.
3. **CDC deletes are only hard deletes.** The change stream emits `op=d`
   (row removed). Real systems just as often soft-delete — set a `deleted_at`
   column and keep the row — and a consumer that has to handle both is not
   testable against Synth today.

## Scope

In scope:

- `synth gen --fk childcol=parent.csv:key` — draw a child field's values from a
  key column in an already-written parent file.
- `synth gen --append` — extend an existing output file instead of overwriting
  it, without repeating rows or primary keys, using a sidecar state file.
- `synth cdc --soft-delete` — emit a delete as an update that stamps a
  `deleted_at` column, rather than an `op=d` event.

Out of scope (considered, deferred to their own specs):

- **Referential / cascade deletes.** Deleting a parent and its children needs a
  multi-table CDC stream; today the stream is over one table. Its own project.
- **Reading parent keys from Parquet.** The FK source reuses the existing
  CSV/JSONL loader; Parquet parents wait until someone needs them.

## B1 — cross-run foreign keys

The CLI already parses `col=parent.csv:key` for `verify --ref` and loads the
file with `constraint.LoadSample`. `gen` gets the same flag, `--fk`, reusing
both.

```sh
synth gen -s users.yaml  -o users.csv  -n 10000
synth gen -s orders.yaml -o orders.csv -n 500000 --fk user_id=users.csv:id
```

`--fk user_id=users.csv:id` reads the `id` column from `users.csv`, and the
generator fills `user_id` by drawing from those values — the same mechanism
`Ref` already uses, via `Field.FromRef`. The engine path is unchanged; only the
source of the values is new (a file instead of an in-memory slice).

A new option, `RefValues(field string, values []any)`, sets `FromRef` from a
slice the caller already has. `YAMLSpec.Generate` learns to apply it (today
only `Make` applies refs). The flag is repeatable, one per FK column.

Errors are specific: a missing file, a key column absent from the parent, or a
parent with zero rows each fail before generation with a message naming the
flag.

## B2 — append mode

`--append` extends an existing file. Two things must hold: new rows must not
repeat old ones, and integer primary keys must not collide.

Both need one fact the output file does not carry — how much has been
generated. A **sidecar state file**, `<out>.synthstate`, records it as JSON:

```json
{"rows": 10000, "pk_high": 10000, "seed": 42, "pk_column": "id"}
```

- `rows` is the record index the next run starts from. The generator seeds each
  row from its index (`eng.Record(base, i)`), so starting at `rows` rather than
  `0` gives fresh rows that are still deterministic and still reproducible.
- `pk_high` is the largest integer primary key written so far. When the PK is an
  auto-increment integer, the next run continues above it. When the PK is a
  UUID, collision is not a real risk and `pk_high` is ignored.

On `--append`:

1. Read `<out>.synthstate`. Absent → treat as a first run (offset 0), then
   write it, so `--append` works whether or not the first run used it.
2. Generate `n` rows starting at record index `rows`, continuing integer PKs
   above `pk_high`.
3. Append the data to the file (no header re-emitted for CSV).
4. Rewrite the state file with the new totals.

The seed in the state file guards against an append with a different seed than
the original run silently producing an incoherent mix; a mismatch is an error
the user must resolve with `--seed`.

## D — soft delete

`cdc --soft-delete` changes what a drawn delete emits. Instead of removing the
row and emitting `op=d` with `after=null`, it:

- sets `deleted_at` on the row to the event timestamp,
- emits an `op=u` update carrying the true before/after,
- and stops touching the row afterwards, exactly as a hard delete does — a
  soft-deleted row is logically gone even though it stays in `live`.

The column name defaults to `deleted_at` and is configurable
(`Config.SoftDeleteColumn`). If the schema has no such column, it is added to
the event's `after` image; the DDL and other outputs are unaffected because CDC
events are JSON maps, not a fixed schema.

`DeleteRate` keeps its meaning — the probability a step deletes — so the same
spec produces a hard-delete or soft-delete history by the flag alone, which is
what makes a consumer testable against both.

## Testing

- FK: generated child values are all drawn from the parent key set; an empty
  parent, a missing key column, and a missing file each error before
  generation.
- Append: two runs of n produce 2n distinct rows with no repeated record and no
  duplicate integer PK; the state file round-trips; a seed mismatch errors;
  append with no prior state behaves as a first run.
- Soft delete: a drawn delete emits `op=u` with `deleted_at` set and no `op=d`;
  the row is never mutated again; hard-delete behaviour is unchanged without
  the flag.
- Determinism holds throughout: same seed and same inputs, byte-identical
  output.
