# Plan: synth diff, MCP diff tool, workbench share-link

Spec: `docs/superpowers/specs/2026-07-26-diff-and-sharelink-design.md`

TDD per phase, each committed on its own.

## Phase 1 — diff package

`diff.Compare(a, b *profile.Result, opts) []Finding` with severity
error/warn/info. Findings for: column added/removed, type flip, numeric
min/max/mean drift past tolerance, null-rate drift, categorical set change,
distinct/row-count info.

Tests: identical → none; removed/added → error; type flip → error; numeric drift
past/within tolerance; new category → warn.

## Phase 2 — `synth diff` CLI

`runDiff`: profile both files, call `diff.Compare`, print text or `--format
json`, `--tolerance` flag, exit 1 on any error finding. Usage text.

Tests: exit codes; json parses; tolerance widens.

## Phase 3 — MCP diff tool

Add a `diff` tool: two inline datasets (`a`, `b`), `format`, `tolerance`;
profile each, call `diff.Compare`, return findings + summary.

Tests: same findings as the CLI on the same data.

## Phase 4 — workbench share-link

`app.js`: a Share button encodes `currentSpec()` as base64url JSON into
`#s=`, copies the URL; boot decodes `#s=` and loads it, falling back to the
default starter on a malformed fragment. Go side asserts the page serves; the
encode/decode is one pure JS function.

## Phase 5 — docs

README (a `synth diff` CI example), CHANGELOG, CLI usage, MCP README.
