# Plan: Postgres COPY output and compressed sinks

Spec: `docs/superpowers/specs/2026-07-22-pgcopy-and-compression-design.md`

Each phase is independently testable and committed on its own. TDD throughout:
test first, watch it fail, then implement.

## Phase 1 — `maxlen`

Independent of everything else; lands first because both the DDL writer and the
generator depend on it.

1. `schema`: document `Params["maxlen"]`. No struct change — `Params` already
   exists.
2. Tag parser: accept `maxlen=N`, reject non-positive values.
3. Generator: truncate string results to `maxlen` runes.
4. `ddlfe`: capture `n` from `varchar(n)` / `char(n)` into `Params["maxlen"]`.
5. OpenAPI / JSON Schema frontend: `maxLength` into `Params["maxlen"]`.

Tests: rune-boundary truncation on a multi-byte value; `varchar(50)` round-trips
to `maxlen=50`; a field with no limit is untouched.

## Phase 2 — compression

1. `sinkopen.go`: `openSink(path)` returning an `io.WriteCloser` that closes the
   compressor and then the file.
2. `formatFromExt`: strip a trailing `.gz` / `.zst` before inferring format.
3. Route the `gen` command's writers through `openSink`.
4. `go get github.com/klauspost/compress`.

Tests: `users.csv.gz` decompresses to exactly the `users.csv` bytes for the same
seed; same for `.zst` and for jsonl and sql; an unknown extension is not treated
as compression.

## Phase 3 — pgcopy text

1. `pgcopy.NewText(w, cols)`, `WriteRow`, `Close`.
2. Escaping: tab, newline, carriage return, backslash; `\N` for NULL.
3. Value formatting per Go type, timestamps as RFC 3339 in UTC.

Tests: golden file; a value containing a tab and a newline survives escaping;
NULL and empty string are distinguishable.

## Phase 4 — pgcopy binary

1. `pgcopy.NewBinary(w, cols, types)` — signature, flags, zero-length header
   extension, per-row framing, `0xFFFF` trailer.
2. Per-type encoders for the five Go types plus NULL.
3. Timestamp conversion to microseconds since 2000-01-01 UTC.

Tests: header and trailer asserted byte-for-byte; a known row encodes to a known
byte sequence; NULL emits length `-1`; timestamp conversion checked against a
hand-computed value.

## Phase 5 — DDL

1. `pgcopy/ddl.go`: `CREATE TABLE` plus the matching `COPY` command.
2. `varchar(n)` when `maxlen` is set, `varchar` otherwise.
3. A shared table maps Go type to (OID, Postgres type name) so the encoder and
   the DDL writer read from the same source.

Tests: the DDL's types and the encoder's OIDs come from the same table and a
test asserts they agree for every supported Go type.

## Phase 6 — CLI and docs

1. `--format pgcopy` and `--format pgcopy-binary`.
2. `pgcopy-binary` also writes `<out>.sql`.
3. `formatFromExt`: `.pgcopy` and `.pgbin`.
4. Update the usage text, `README.md` and `CHANGELOG.md`.

Tests: end-to-end CLI run produces both files; usage text lists the new formats.
