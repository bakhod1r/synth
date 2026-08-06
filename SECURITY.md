# Security Policy

## Supported versions

Fixes land on the latest `v1` minor release. The public API is frozen at `v1`,
so upgrading within `v1` cannot break your build.

| Version | Supported |
| --- | --- |
| 1.5.x | yes |
| 1.4.3 | security fixes only |
| < 1.4.3 | no — 1.4.2 is retracted (it does not compile) |

## Reporting a vulnerability

Report privately, not in a public issue:

- GitHub → the repository's **Security** tab → **Report a vulnerability**, or
- email **bakhodiryashinmansur@gmail.com** with `SECURITY` in the subject.

Include the version, a schema and seed that reproduce it, and what an attacker
gets. Because generation is deterministic, a seed plus a schema is a complete
reproduction.

Expect an acknowledgement within 72 hours and an assessment within a week. If a
fix is warranted, the release notes will credit you unless you ask otherwise.

## What is in scope

Synth's threat model is narrow because the tool is narrow. In scope:

- **A boundary being crossed.** The library opens no network connection, no
  database connection, and writes no file you did not name. The MCP server
  reads no file at all. Anything that breaks one of those is a vulnerability,
  not a feature request.
- **Path handling.** Anything that lets a schema write outside the directory
  it was given — `dir=` on an image column, an output path, a sink.
- **A masking key leaking.** `synth mask` is keyed; the key must not appear in
  output, logs, error messages, or generated files.
- **Reversible masking.** `synth mask` claims pseudonymization. A practical way
  to invert it without the key is a vulnerability.
- **Crashes on hostile input.** The YAML, DDL, OpenAPI, JSON Schema, Avro and
  Protobuf parsers accept files you may not have written. A panic, an
  unbounded allocation, or a hang on a small input is in scope; the fuzz
  targets exist for exactly this.
- **Dependency vulnerabilities** in the two core dependencies or in a nested
  module's own set.

## What is not in scope

- **Generated data is not secret.** Card numbers pass Luhn, IBANs pass mod-97,
  passwords look like passwords. They are synthetic by construction. Do not
  report a generated value as a leak.
- **The PRNG is not cryptographic.** `internal/rng` is a PCG chosen for speed
  and reproducibility. Predicting its output from earlier output is its
  documented behaviour, not a flaw — determinism is the point. Never use
  generated values as keys, tokens, or anything an attacker must not guess.
- **Generated images and text are untrusted content by design.** An SVG from
  `imagegen` contains only rectangles, circles and polygons — no script, no
  external reference, no font — so it is safe to inline. But data that came
  from a user's own schema (enum choices, `Register` values) is theirs, and
  escaping it for your output format is your renderer's job.
- **Running a schema you did not write.** A schema is code-adjacent: it names
  output paths and, with `dir=`, directories to write into. Treat an untrusted
  schema the way you would treat an untrusted script.
- **Denial of service by asking for it.** `synth gen -n 10000000000` will take
  a long time and a lot of disk. That is the request being honoured.

## Hardening notes

- `synth mask` refuses to run without `--key`, because an unkeyed run is not
  reproducible, and refuses to overwrite its own input.
- The core library's dependency count is asserted in CI, so a supply-chain
  surface cannot grow quietly.
- Every generating surface is covered by race-enabled tests; report any data
  race you find, even without a demonstrated exploit.
