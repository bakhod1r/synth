# Postgres COPY output and compressed sinks

Date: 2026-07-22

## Problem

Synth writes CSV, JSONL, SQL `INSERT` and Parquet. Two gaps hurt at the volumes
Synth targets:

1. **Loading into Postgres is slow.** `INSERT` statements are the slowest path
   Postgres offers. A 100M-row file takes hours to load.
2. **Files are large.** 100M rows of JSONL is tens of gigabytes on disk, and
   every output format is written uncompressed.

## Scope

In scope:

- `gzip` and `zstd` compression for every existing text output.
- `--format pgcopy` — Postgres `COPY` text format.
- `--format pgcopy-binary` — Postgres `COPY` binary format, plus a matching
  `CREATE TABLE` file.
- `maxlen` as a schema-level field property, required to emit correct
  `varchar(n)` DDL.

Explicitly out of scope (considered and dropped):

- **Avro** — dropped; Parquet already covers the data-lake case.
- **ORC** — no maintained Go writer exists. Reaching it through
  `apache/arrow-go` is a large dependency for a format Parquet substitutes for.
- **Connecting to Postgres.** Synth writes files. Loading them is the user's
  job, as with every other output.

## Dependencies

`github.com/klauspost/compress` is added to the core module, making three direct
dependencies (`uuid`, `yaml.v3`, `compress`). This is deliberate: compression
exists for large files, and `zstd` is roughly 3-5x faster than `gzip` at a
better ratio, which is the whole reason to compress at all. Putting it in a
submodule would mean `-o users.jsonl.zst` silently fails in the core binary.

`gzip` is stdlib. Postgres `COPY` encoding is stdlib — no new dependency.

No new Go module. Everything lands in the core module.

## Components

```
sinkopen.go        new  openSink(path) (io.WriteCloser, error)
pgcopy/pgcopy.go   new  TextWriter, BinaryWriter
pgcopy/ddl.go      new  CREATE TABLE from schema.Schema
encode.go          unchanged
```

### openSink

One function decides compression, so every format gets it for free. The
filename carries both facts: format from the inner extension, compression from
the outer one.

| Path | Format | Compression |
| --- | --- | --- |
| `users.csv` | csv | none |
| `users.csv.gz` | csv | gzip |
| `users.jsonl.zst` | jsonl | zstd |
| `users.pgbin` | pgcopy-binary | none |

An explicit `--format` overrides the inferred format; compression is always
taken from the extension.

### pgcopy

Streaming, not buffered:

```go
w := pgcopy.NewText(dst, cols)
w := pgcopy.NewBinary(dst, cols, types)
w.WriteRow(map[string]any) error
w.Close() error
```

No row accumulates in memory, so the constant-memory guarantee holds at 100M
rows.

**Text format** is tab-separated with `\N` for NULL and backslash escapes for
tab, newline, carriage return and backslash. It loads with:

```sql
COPY t FROM '/path/users.pgcopy';
```

**Binary format** is the `PGCOPY` wire format: an 11-byte `PGCOPY\n\377\r\n\0`
signature, a 4-byte flags field, a 4-byte header-extension length (zero), then
per row an `int16` field count followed by, per field, an `int32` byte length
(`-1` for NULL) and the raw value bytes. The file ends with the `0xFFFF`
trailer.

## Type mapping

The 250+ `schema.Kind` values collapse to five Go value types, so the binary
encoder keys off the value type, not the kind.

| Go value | OID | Postgres type | Encoding |
| --- | --- | --- | --- |
| `string` | 1043 | `varchar` / `varchar(n)` | UTF-8 bytes |
| `int64` | 20 | `bigint` | big-endian int64 |
| `float64` | 701 | `double precision` | big-endian IEEE-754 |
| `bool` | 16 | `boolean` | one byte, 0 or 1 |
| `time.Time` | 1114 | `timestamp` | int64 microseconds since 2000-01-01 UTC |
| `nil` | — | — | length `-1` |

The DDL writer emits exactly these column types from the same table, so a
mismatch between the data file and the table is structurally impossible — which
is the only way binary `COPY` is usable in practice, since Postgres rejects the
whole file on the first type disagreement.

The generated `.sql` file contains the `CREATE TABLE` and the matching `COPY`
command, so the two files are used together.

## maxlen

`varchar(n)` needs `n`. Nothing currently retains it: `ddlfe` reads
`varchar(50)` and discards the `50` (every string-like column collapses to
`KindLorem` at `ddlfe/ddlfe.go:203`), and the OpenAPI frontend ignores
`maxLength`.

**New:** `Field.Params["maxlen"]`, a string field's maximum length in runes.

Producers:

- `ddlfe` — from `varchar(n)` and `char(n)`.
- `openapi` and the JSON Schema frontend — from `maxLength`.
- Go struct tags — `synth:"...,maxlen=50"`.

Consumers:

- `pgcopy/ddl.go` — emits `varchar(n)` when set, `varchar` otherwise.
- **The generator** — truncates generated strings to `n`, counted in runes so
  multi-byte names are not cut mid-character.

Truncating in the generator rather than only in the DDL is the point: a limit
that exists only in the table produces `value too long for type character
varying(50)` at load time. This also makes the DDL frontend honour a constraint
it currently reads and throws away, which is useful well beyond `pgcopy`.

## Testing

- Golden files: text and binary output are byte-stable for a fixed seed.
- Round-trip: a compressed file decompresses to exactly the uncompressed bytes.
- The binary signature, flags, header extension and trailer are asserted at the
  byte level.
- A test locks the DDL's column types to the OIDs the binary encoder emits, so
  the two cannot drift apart.
- Edge cases: NULL, empty string, unicode, and tab / newline / backslash inside
  a value (escaped in text format).
- `maxlen` truncation lands on a rune boundary.
- `ddlfe` recovers `n` from `varchar(n)`.
- No real Postgres is started — the project connects to nothing.

## Not doing

- `COPY ... WITH (FORMAT csv)`. The existing CSV output already loads that way.
- Per-column type overrides. The five-type mapping is derived from the schema;
  a user who needs `numeric` or `jsonb` can edit the generated DDL.
