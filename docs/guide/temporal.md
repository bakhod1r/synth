# Temporal causality

Timestamps aren't random points in a range — they respect the order events can
happen in. An order is `created → paid → shipped → delivered`, each strictly
after the last, with realistic gaps. Accounts are never used before they're
opened, and refunds never precede their charge.

## Lifecycles

```go
synth.Orders(1_000, synth.Timeline("2026-01-01", "2026-07-01"), synth.Lifecycle(synth.OrderFlow))
```

## Per-field ordering

```go
type Order struct {
    CreatedAt   time.Time
    PaidAt      time.Time `synth:"time,after=CreatedAt,gap=1h..48h"`
    ShippedAt   time.Time `synth:"time,after=PaidAt,gap=1h..72h"`
    DeliveredAt time.Time `synth:"time,after=ShippedAt,gap=1h..120h"`
}
// CreatedAt < PaidAt < ShippedAt < DeliveredAt, always.
```

| Tag option | Meaning |
| --- | --- |
| `after=<field>` | this timestamp is strictly later than that one |
| `gap=<lo>..<hi>` | how much later, as a duration range |

## Time as another axis

One seed fixes the dataset; [snapshots](../tools/snapshot.md) make *time* another
axis of that determinism — the table as it stood at any instant, or what changed
between two. [CDC](../tools/cdc.md) emits the same history as a change stream.
