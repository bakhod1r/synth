# Unique columns

A column marked `unique` gets distinct values. Two modes, with a real trade-off
between them.

## Tracked (the default)

Synth resamples until it finds a fresh value, tracking everything generated so
far. Values stay natural, but memory grows with the row count, and a column with
fewer possible values than rows is an error rather than a silent duplicate:

```
synth: field "status" ran out of unique values after 3 rows; its value space is
too small for the row count (use unique=counter, or widen the field)
```

## `unique=counter`

Derives distinctness from the record index instead. It costs a visible suffix
(`ivanov.dilnoza41293@gmail.com`) and buys constant memory, any row count, and
parallel generation — `MakeParallel` accepts it, where tracked unique fields
must go through `Make`.

```go
type Row struct {
	ID    uuid.UUID `synth:"pk"`                    // unique by construction
	Email string    `synth:"email,unique"`          // tracked, natural values
	Slug  string    `synth:"username,unique=counter"` // constant memory at 1B rows
}
```

```yaml
fields:
  slug: { kind: username, unique: true, unique_mode: counter }
```

## Choosing

| | tracked | `counter` |
| --- | --- | --- |
| Values | natural | natural plus a numeric suffix |
| Memory | grows with rows | constant |
| Row ceiling | the column's value space | none |
| `MakeParallel` | no — use `Make` | yes |
| Too-small value space | error | not possible |
