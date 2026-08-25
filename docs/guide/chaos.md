# Edge-case injection

Testing the happy path is the easy part. Ask for the values that break parsers:
unicode names, emoji, RTL text, empty strings, boundary numerics, nulls in
nullable columns.

```go
synth.New(synth.WithChaos(0.02))   // 2% of records carry a nasty value
```

From the CLI:

```sh
synth gen -s users.yaml -n 100000 --chaos 0.02 -o users.csv
```

The fraction is between 0 and 1. Chaos is seeded like everything else, so a
parser that fell over on run 4,231 falls over there again.
