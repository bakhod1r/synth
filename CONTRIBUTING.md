# Contributing to Synth

Thanks for taking the time. This file records what the project asks of a change
so you can find out before you write it, not in review.

## The short version

```sh
git clone https://github.com/bakhod1r/synth && cd synth
go test ./...           # everything, race-clean
gofmt -l .              # must print nothing
go vet ./...
```

If those three pass, CI will agree with you about almost everything.

## Ground rules the code is built on

These are not style preferences. Each one is enforced by a test or a CI job,
and a change that breaks one is a change to the project's contract.

**Two dependencies.** The core library depends on `google/uuid` and `yaml.v3`,
and nothing else, ever. Anything that needs more lives in its own nested module
(`cmd/synth`, `mcp`, `sink/parquet`, …). CI counts the dependencies and fails if
the number moves. If your feature seems to need a third, it needs a module, or
it needs to be written out — that is how `imagegen` came to rasterize its own
text instead of importing a font library.

**Determinism.** The same seed and the same schema produce the same bytes, on
every machine, in any order, on any number of goroutines. Record *i* must not
depend on record *i−1* having been generated first — that is why `rng.Fork` is
a pure function of (seed, index) and does not consume the parent stream. A
provider that reads a clock, a global, or a shared random source is a bug.

**No network, no database, no writing outside what was asked for.** Synth is a
pure provider: it produces values and writes the files you name. There is a
test for each of those boundaries. A feature that phones home does not belong
here regardless of how useful it is.

**Errors are values, not exits.** A provider that cannot do its job returns
something visible in the output; it does not panic and does not abort a
million-row generation. One malformed parameter should cost you one bad column,
not the run.

## Adding a field type

Most contributions are a new `schema.Kind`. The path is deliberately short:

1. Add the constant to `schema/schema.go`, with a comment saying what it means
   when it is not obvious from the name.
2. Register a provider in `providers/`. Use `set` for a plain list,
   `setLocalized` for one with locale variants, or a function for anything
   composed.
3. Add it to `infer/infer.go` if a column with that name should be recognized
   automatically.
4. Test it. A list needs a test that it is non-empty and reachable; a composed
   value needs a test of the property that makes it valid — Luhn, mod-97,
   correct length, agreement with `from=`.

Prefer composition to a long list. A fixed list of 200 values repeats visibly
within a few thousand rows; a composed value does not. Where the list is the
point (real book titles, real cities), say so in a comment so nobody
"optimizes" it later.

## Tests

- Unit tests live next to the code.
- `tests/` is for end-to-end checks that cross package boundaries.
- `benchmarks/` is for measurements that belong to no single package. They are
  informational: never assert a timing threshold, because a shared runner's
  numbers are too noisy and a test that fails randomly gets ignored.
- Fuzz targets cover the parsers. If you touch a parser, add the input that
  broke it to the seed corpus.

Write the test that would have caught the bug, and say in a comment what
behaviour it protects. A test named `TestFoo2` that asserts a value nobody can
explain is worse than no test.

## Commits and pull requests

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org):
`feat(providers): …`, `fix(locale): …`, `test: …`, `docs: …`, `chore: …`. The
subject line says what changed; the body says why, if why is not obvious.

A pull request should:

- pass `go test ./...` with `-race`
- be `gofmt`'d
- update `CHANGELOG.md` under `## [Unreleased]` if the change is user-visible
- update the README if it adds a capability someone would look for there

Small and focused beats large and comprehensive. If a change touches the public
API, say so explicitly in the description — the API is frozen at `v1`, so an
addition is cheap and a change is a `v2`.

## Reporting a bug

Include the schema (struct, YAML or DDL), the seed, and what you got versus what
you expected. Because generation is deterministic, a seed plus a schema is a
complete reproduction — that is the whole reason it works that way.

Security issues go to [SECURITY.md](SECURITY.md) instead, not to the issue
tracker.

## Conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).
