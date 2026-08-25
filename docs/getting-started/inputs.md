# Input formats

Synth builds its schema from whichever definition you already have. Each one
parses into the same `schema.Schema`, so coherence, constraints and masking
behave identically no matter where the schema came from.

| Source | API |
| --- | --- |
| Go structs (tags optional) | `synth.Make[T]` |
| YAML spec | `synth.LoadYAML` |
| OpenAPI 3 | `synth.OpenAPI` |
| SQL DDL (`CREATE TABLE`) | `synth.LoadDDL` |
| JSON Schema | `synth.LoadSchema` |
| Avro schema | `synth.LoadSchema` |
| Protobuf (`.proto`) | `synth.LoadProto` |
| Real-data sample (CSV/JSONL) | `synth.Profile` |

## Schema-driven generation

Point Synth at a schema instead of hand-writing generators:

- **OpenAPI** — valid request bodies for every endpoint, respecting `format`,
  `pattern`, `enum`, `minimum`, and `required`.
- **SQL DDL** — read `CREATE TABLE`, infer types, honor `NOT NULL`, `UNIQUE`,
  `CHECK`, and FK constraints.
- **Go structs** — generate from your existing domain types via tags.

## Go structs

Tags are optional: an untagged field is inferred from its name and type, so an
existing domain type usually needs no annotation at all.

```go
type User struct {
	ID        uuid.UUID `synth:"pk"`
	FirstName string                        // inferred: a first name
	Email     string `synth:"email,from=FirstName"`
	CreatedAt time.Time                     // inferred: a timestamp
}
```

Nested structs and slices are generated recursively.

## Real data

`synth.Profile` learns a spec from an export instead of a schema — see
[Profiling](../tools/profile.md) and
[Constraint mining](../tools/constraints.md).
