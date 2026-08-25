# Learned invariants (constraint mining)

Per-column [profiling](profile.md) learns what each column looks like. It cannot
see that a `total` must agree with its line items, or that a refunded order must
carry a refund timestamp. `synth profile` mines those cross-column invariants
from the sample and writes them into the spec:

```yaml
constraints:
  - {kind: sum, parts: [subtotal, tax], whole: total}    # held over 48,912 rows
  - {kind: ordering, left: created_at, right: updated_at}  # held over 50,000 rows
  - {kind: implication, when: status, equals: "refunded", then: refund_at, exclusive: true}
```

## Enforcement

Generation then enforces them. A total is **derived** from its parts rather than
generated independently; an out-of-order timestamp pair is swapped, which
preserves both values and their distribution instead of piling duplicates up at
a boundary.

## Why mining is conservative

A candidate is generated, then falsified against the whole sample; an
implication additionally needs a trigger group of real size, and needs its
target column to be empty *somewhere else* — otherwise the column is simply
always populated and the trigger has nothing to do with it. Each surviving rule
records how many rows it held over, so you can judge it rather than trust it.

## Contradictions fail loudly

If a spec's constraints contradict each other, `Generate` returns an error
naming the unsatisfiable one. Emitting rows that quietly violate the spec would
be worse than failing: the data would look authoritative and be wrong.

## Re-checking them later

`synth verify -s <spec.yaml>` runs the mined invariants back over a dataset —
see [Verify](verify.md).
