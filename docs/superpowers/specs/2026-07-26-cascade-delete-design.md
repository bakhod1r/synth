# Cascade deletes in a two-table CDC stream

Date: 2026-07-26

## Problem

The CDC stream (`cdc` package) runs over a single table. A real change feed
covers related tables, and the case that a single-table stream cannot represent
is the **referential cascade**: deleting a parent row also deletes the child
rows that point at it. A consumer that maintains a join, or tests `ON DELETE
CASCADE` handling, has nothing to exercise it against.

## Scope

- `cdc.Cascade(parent, child, cfg)` — one interleaved change stream over a
  parent table and one child table whose foreign key points at the parent.
- On a parent delete, every child row referencing that parent is deleted too,
  each as its own event, before the parent's own delete — the order a
  foreign-key constraint requires.
- Inserts keep referential integrity: a child is only ever created against a
  parent that already exists.

Out of scope: more than one child table, multi-level cascades (grandchildren),
soft-delete cascades. One parent, one child, hard cascade — the shape that
covers the common `orders → order_items` case. Deeper trees get their own spec.

## Model

`cdc.Cascade` returns a `*CascadeStream` whose `Next()` yields events across
both tables, sharing one monotonic LSN and timestamp so the interleaving is a
single coherent history.

```go
type CascadeConfig struct {
    ParentTable, ChildTable string
    ParentKey               string // parent PK column; default: the schema's PK
    ChildKey                string // child PK column; default: the schema's PK
    ChildFK                 string // child column holding the parent's key (required)
    ChildrenPerParent       int    // children created with each new parent (default 3)
    UpdateRate, DeleteRate  float64 // step probabilities, as in the single-table stream
    Seed                    uint64
    Locale                  string
    Snapshot                int
    Start                   time.Time
    Interval                time.Duration
}
```

Each `Next()` step draws an action:

- **insert** (default): create a parent row, then `ChildrenPerParent` child rows
  whose `ChildFK` is the parent's key. Emitted as one parent `c` event followed
  by the child `c` events. Referential integrity holds by construction.
- **update**: mutate a random live row — parent or child — and emit its `u`.
- **delete**: pick a random live parent, delete each of its live children (a `d`
  per child), then delete the parent (`d`). Children first, so at no point does
  a deleted parent still have children pointing at it. A parent with no children
  is just a single `d`.

Every event carries `Source.Table` set to the parent or child table, so a
consumer routes by table. The stream is deterministic under the seed, like the
single-table one.

## Implementation

A new `cdc/cascade.go`. It reuses the existing per-table pieces:

- two `gen.Engine`s (parent, child), compiled from the two schemas;
- the same `event`/LSN/timestamp machinery as `Stream` (extracted to a small
  shared helper so both use one clock);
- live state as parents plus, per parent key, its live child rows, so a cascade
  delete can find the children to remove.

The child FK is filled after generation: the child engine produces a row, then
its `ChildFK` column is overwritten with the chosen parent's key — the same
"reference an existing key" move `RefValues` makes, kept local to the stream.

## Testing

- Insert keeps integrity: every child's `ChildFK` is a parent key that was
  created (and not yet deleted) at the time.
- Cascade: after a parent `d`, every child that referenced it has its own `d`
  earlier in the stream, and none is touched afterward.
- Ordering: in a cascade, all child deletes precede the parent delete, and LSNs
  are strictly increasing across the whole stream.
- A childless parent deletes as a single event.
- Determinism: same seed, byte-identical event sequence.
- `ChildFK` is required; its absence errors at construction.
