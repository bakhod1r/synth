# Custom types

When none of the 264 built-in types is the thing you need, register your own —
by value set or by function.

```go
synth.RegisterSet("cinema", "Inception", "Interstellar", "Tenet", "Dune")
synth.Register("rating", func(r synth.R) any { return r.IntRange(1, 5) })
```

Then use the name like any built-in kind:

```go
type Screening struct {
    Film   string `synth:"cinema"`
    Rating int    `synth:"rating"`
}
```

```yaml
fields:
  film:   { kind: cinema }
  rating: { kind: rating }
```

The `synth.R` handed to a generator function is the record's own RNG, so a
custom type stays deterministic under the seed and safe under parallel
generation — draw from it rather than from `math/rand`, or you give up
[reproducibility](streaming.md).
