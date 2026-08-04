# Changelog

Notable changes to Synth. Versions follow [semantic versioning](https://semver.org).

The public API is frozen at `v1`. A breaking change means `v2`, and `v2` means a
new import path — so anything that would break your build cannot reach you by
accident.

## [Unreleased]

## [1.4.3] — 2026-08-04

### Fixed

- Build: `TestEncodeCSVHeaderWriteError` was declared twice, so `go vet` and
  `go test` failed to compile the root package — the 1.4.2 release build never
  produced binaries.
- `Streamer.ToCSV` and `Streamer.ToJSONL` compile the schema before creating
  the output file, so a schema error no longer leaves an empty file behind.

## [1.4.2] — 2026-08-04

### Changed

- The workbench labels the per-column locale picker `localize`, the name the
  setting had before `locale=` replaced it.

## [1.4.1] — 2026-08-04

### Changed

- The workbench no longer shows a `localize` dropdown per column. Per-field
  `locale=` says everything `localize=false` could — `locale: en_US` is the
  same opt-out — so two controls for one decision only invited setting both
  and wondering which won. `localize=` is untouched in struct tags, YAML and
  the API; only the workbench control is gone.

## [1.4.0] — 2026-08-03

### Added

- Per-field locale: `locale=ja_JP` on a struct tag, or `locale: ja_JP` in a YAML
  field, generates that one column as if the dataset locale were that locale.
  `localize=false` could only say "English instead"; a record that mixes voices
  for any other reason — a Japanese phone number on an Uzbek customer, a German
  shipping city on a Turkish order — now has a way to say so. It wins over
  `localize=` when both are set, and an unknown locale name is a compile error
  rather than a silent fall back to English.
- The workbench offers the same setting per column, next to `localize`.
- `locale.Has` reports whether a locale name is registered — `locale.Get` falls
  back to `en_US`, which is right when generating and wrong when validating.

### Fixed

- The hosted WebAssembly workbench marked a type as locale-following only when
  it had a per-locale word list, so structurally localized types — `name`,
  `email`, `phone`, `country`, `city` and the rest — showed an unlit dot and
  hid their `localize` setting. Both builds now answer `/api/types` through one
  shared rule, so the palette means the same thing in either.

## [1.3.2] — 2026-08-03

### Fixed

- Release workflow: the CLI binary is now built from inside the `cmd/synth`
  module (it stopped being part of the root module in 1.3.1), so the
  cross-platform release build succeeds again. No library or CLI behaviour
  changes.

## [1.3.1] — 2026-08-03

### Changed

- The `synth` CLI is now its own module (`cmd/synth`). Its output-format
  dependencies — the Parquet writer and the zstd/gzip compressor — no longer
  sit in the core library's module graph, so `go get github.com/bakhod1r/synth`
  is back to exactly `google/uuid` and `yaml.v3`. This reverts the v1.2.0
  regression that pulled the Parquet dependency tree into the core, and the CI
  dependency-budget gate passes again. Installing the binary
  (`go install github.com/bakhod1r/synth/cmd/synth@latest`) is unchanged and
  still produces Parquet and compressed output.

## [1.3.0] — 2026-08-03

### Added

- Hash and token masks can pick their digest algorithm with `algo=`: `sha256`
  (the default, unchanged) or `sha512`. Both come from the standard library, so
  no dependency is added, and the choice carries through the `secret=` HMAC path.
- Workbench: a localizable column can be opted out of the dataset locale from
  its options dialog (`localize=false`, generated as `en_US`), shown only where
  the type actually follows the locale. The hash/token mask gains an algorithm
  select. The static build's banner now links to the docs.

### Fixed

- Workbench: with the palette hidden, the work area collapsed to its content
  width and left the right of the page empty — the shell grid had a dead third
  track and `main` fell back into the auto-sized first one. `main` is now pinned
  to the flexible track, and the dead `#tools`/`.controls` rules are gone.

## [1.2.0] — 2026-08-03

### Added

- Parquet is now a first-class CLI output: `synth gen -f parquet` or a
  `.parquet` extension writes the file directly. It needs a real path — a
  Parquet footer cannot stream to stdout or through the gzip/zstd sink, and
  `--append` does not apply. The `sink/parquet` writer stays importable from Go.

### Changed

- The core module now requires `sink/parquet`, so an import of `synth` pulls the
  Parquet dependency graph. The previous "core needs only `google/uuid` and
  `yaml.v3`" guarantee no longer holds; docs updated to match. The `mcp` module
  still stays out of the core graph.

### Fixed

- `reflectfe`: a `uuid.UUID` struct field (a `[16]byte` array) was treated as a
  byte array instead of a scalar UUID; named scalars are now recognised before
  the array path.
- `synth.Ref`: an empty parent slice panicked later at `IntN(0)`; the ref is now
  skipped so the foreign-key field generates normally, matching `RefValues`.
- `yamlfe`: `mu`/`sigma`/`s`/`rate` were rendered as quoted strings, which YAML
  would not unmarshal back into their `*float64` fields; they now round-trip as
  bare numbers.

## [1.1.0] — 2026-07-27

### Added

- Postgres `COPY` output: `--format pgcopy` (text) and `--format pgcopy-binary`,
  or the `.pgcopy` / `.pgbin` extensions. An `INSERT` per row is the slowest way
  to load Postgres; `COPY` is what the server wants for bulk data.
- A matching `CREATE TABLE` is written alongside as `<out>.sql`. Binary `COPY`
  carries no type names — the table's column types are what the bytes are
  decoded as — so both come from one type table and cannot disagree.
- gzip and zstd output, chosen by the filename: `-o users.jsonl.gz`,
  `-o users.csv.zst`. Applies to `gen`, `cdc` and `snapshot`, with the format
  still read from the extension underneath.
- String length limits from the source schema are honoured. `varchar(n)`,
  `char(n)` and JSON Schema / OpenAPI `maxLength` now reach the generator as
  `maxlen`, which truncates to them in runes; previously the length was parsed
  and discarded.
- Cross-run foreign keys: `gen --fk col=parent.csv:key` fills a child column
  from a key column in a parent file written by an earlier run, so tables
  generated separately still join. `synth.RefValues` is the library equivalent.
- `gen --append` extends an existing file instead of overwriting it, using a
  `<out>.synthstate` sidecar to continue without repeating rows. `synth.Offset`
  is the underlying option.
- `cdc --soft-delete` emits a delete as an `op=u` update that stamps a
  `deleted_at` column, rather than an `op=d`, so a consumer can be tested
  against both delete workloads from one spec.
- Correlated numerics: `derive: other` makes a numeric field a linear function
  of another field in the same row (`slope`, `intercept`, `noise`), so related
  columns like income and age come out correlated instead of independent.
- `kind: timeseries`: a numeric column that follows
  `base + trend + seasonality + noise` over a named timestamp `axis`, for
  metrics and IoT data.
- `synth diff a.csv b.csv` compares two datasets by shape — columns, types,
  numeric ranges, null rates, category sets — and exits non-zero on a
  structural break, for CI regression guards. `--tolerance` and `--format json`
  included. An MCP `diff` tool exposes the same over inline datasets.
- Workbench **Share** button: encodes the schema in the URL fragment so a link
  reopens it exactly, with nothing uploaded.
- k-anonymity check: `verify --k N --qi age,zip,gender` fails when any
  quasi-identifier combination is shared by fewer than N rows, the measure of
  whether "anonymized" data can still be re-identified.
- Differential-privacy masking: `mask --dp col:epsilon:sensitivity` adds Laplace
  noise to a numeric column, bounding how much one record shows through.
- Cascade deletes in CDC: `cdc -s parent.yaml --child child.yaml --child-fk col`
  produces a two-table change stream where deleting a parent deletes its
  children first, then the parent — the order a foreign key requires.
- Per-field `localize=` opt-out: a field can be forced to the neutral locale
  while the rest of the record stays localized, for columns that should not vary
  by region.
- Wider email generation: more provider domains and safe-character handling, so
  addresses stay valid across the expanded name banks.

### Dependencies

- `github.com/klauspost/compress` for zstd.

## [1.0.0] — 2026-07-22

First release. Everything below already existed; the tag is what makes it
depend-able, and the commitment is that it will keep working.

### Generation

- Records rather than fields: referential integrity (`Ref`, `OneToMany`),
  temporal causality (`after=`, `gap=`), unique constraints and primary keys,
  nested structs and slices.
- Locale coherence across 52 locales — one `Place` is drawn per record, so the
  city matches the postcode instead of merely both being Uzbek. Gendered name
  banks keep first name, surname and the gender column consistent.
- 260 column types, including `birthdate`/`age`, the national identifier under
  the names people search for (`pinfl`, `nationalid`, `taxid`), and the card
  security code under all six of its network names (`cvv`, `cvc`, `cvv2`,
  `cvc2`, `csc`, `cid`).
- Format-valid values: Luhn cards with real BIN ranges, mod-97 IBANs, ISIN, LEI,
  CUSIP, EAN-13, and a national identifier per locale with its real check digit
  — PINFL, TC Kimlik, IIN, PESEL, DNI, NIF, BSN, Aadhaar, the Chinese MOD 11-2.
- Statistical distributions: uniform, normal, log-normal, exponential, Zipf.
- Per-record RNG, so the same seed gives byte-identical output at any worker
  count.

### Frontends

Go structs, YAML, OpenAPI 3, SQL DDL, JSON Schema, Avro, Protobuf, and profiling
a real CSV/JSONL export. All collapse into one schema, so every feature works
regardless of where the schema came from.

### Products built on the same engine

- `synth verify` — audit an existing dataset for broken checksums, malformed
  formats and time anomalies.
- `synth mask` — replace personal data in a real export, keeping foreign keys
  matched across related dumps.
- `synth snapshot` — the table at an instant, or the change events between two.
  Replaying the events onto the earlier state reproduces the later one.
- Constraint mining — learn invariants from a sample and hold them while
  generating.
- `synth ui` — a local browser workbench, loopback only.
- MCP server (`mcp/`) — seven tools for an assistant, stdio only, no files.

### Output

CSV, JSONL, SQL `INSERT`, Parquet (`sink/parquet/`), CDC events. Streaming keeps
memory constant at any row count.

### Boundaries

These are deliberate and enforced by tests, not conventions:

- **No database.** Synth supplies data; a loader writes it.
- **No network.** The workbench binds loopback and refuses anything else. The
  MCP server speaks stdio and cannot import `net/http`.
- **No files from MCP.** Every tool takes its input as an argument, so a
  prompt-injected model cannot turn a data generator into a file reader.
- **Two dependencies** in the core library. Anything heavier lives in a nested
  module, and CI fails if that slips.

### Fixed before the tag

Found while preparing this release, all of them the quiet kind:

- An unquoted date bound in YAML was parsed as a timestamp and then formatted
  into something no provider could read. An unparseable bound is ignored rather
  than rejected, so the `user` preset shipped generating dates of birth in 2025.
- `mask=hash` produced the same digest for the same value in different columns,
  so anyone holding two masked tables could join on the masked value and re-link
  the rows the mask was meant to separate.
- Profiling could emit a spec Synth itself could not parse, for a column name
  containing a control character or long enough to exceed YAML's key limit.
- Profiling turned a short sample of a UUID or email column into an enum whose
  choices are the real values — copying identifiers into a file meant for
  version control.
- A true/false column profiled as an enum of the strings `"true"` and `"false"`,
  handing a JSON consumer a string where the source had a boolean.
- `ddlfe` skipped table-level `PRIMARY KEY (id)` — the form pg_dump writes — so
  the key column looked ordinary and generation could produce duplicates. It
  also failed entirely on `public.users` and `[users]`.
- `locale.Names()` ranged over a map, so the locale list came out in a different
  order on every run.

[Unreleased]: https://github.com/bakhod1r/synth/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/bakhod1r/synth/compare/v1.3.2...v1.4.0
[1.1.0]: https://github.com/bakhod1r/synth/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/bakhod1r/synth/releases/tag/v1.0.0
