# synth diff, an MCP diff tool, and a workbench share-link

Date: 2026-07-26

## Problem

Three developer-experience gaps, all independent.

1. **No way to compare two datasets' shape.** After changing a generator, or to
   guard against a real dataset drifting, there is nothing that answers "is this
   file shaped like that one?" — column set, types, ranges, null rates,
   category mix. A CI job that wants to fail on drift has to hand-roll it.
2. **The MCP server cannot diff.** It exposes generate/profile/verify/mask/…,
   but not the comparison above, so an assistant holding two datasets cannot ask
   whether they match.
3. **A workbench schema cannot be shared.** The browser workbench holds the
   whole schema in memory; there is no URL that reproduces it, so a schema
   cannot be sent to a colleague or bookmarked.

## Scope

- `synth diff a.csv b.csv` — profile both and report per-column drift; exit
  non-zero when drift crosses a threshold, for CI.
- An MCP `diff` tool over two inline datasets.
- A workbench **Share** button that encodes the schema in the URL, and boot-time
  decoding of that URL.

Out of scope: diffing row-by-row (this is a *shape* diff, built on `profile`);
semantic column matching across renames (columns are matched by name).

## Phase 1 — `synth diff`

Profile both inputs with the existing `profile` package, then compare the two
`profile.Result`s column by column.

Findings, each with a severity:

- **error** — a column present in one and not the other; a column that changed
  between numeric and non-numeric, or categorical and free. These break a
  consumer, so they fail CI.
- **warn** — a numeric column whose min/max or mean moved by more than a
  tolerance (default 10%); a null rate that moved by more than a tolerance
  (default 5 percentage points); a categorical column whose value set changed.
- **info** — distinct-count drift within tolerance, row-count difference.

```
$ synth diff old.csv new.csv
~ amount   mean 512.30 → 640.10 (+25%)      warn: numeric drift
- legacy_id                                 error: column removed
+ tier                                      error: column added
~ status   {active,closed} → {active,closed,pending}  warn: new category
2 errors, 1 warning
```

`--tolerance` overrides the numeric fraction, `--format json` emits the findings
as JSON for a CI step to parse, and the exit code is 1 when any error-level
finding is present (2 warnings still exit 0, like `verify`).

A `diff` package holds the comparison so both the CLI and the MCP tool call it;
the CLI does file I/O, the package takes two `*profile.Result`.

## Phase 2 — MCP `diff` tool

```
diff(a: string, b: string, format?: "csv"|"jsonl", tolerance?: number)
```

Both datasets are passed as text, like the other MCP tools — it reads no files.
It profiles each with the same code path and returns the findings as structured
JSON plus a one-line summary. Reuses the Phase 1 `diff` package unchanged.

## Phase 3 — workbench share-link

The workbench state that matters is what `currentSpec()` already builds:
`{name, count, locale, seed, fields, order}`. A **Share** button:

1. serialises that object to JSON,
2. base64url-encodes it (no server round-trip, no storage),
3. puts it in the URL fragment (`#s=<encoded>`) — a fragment, so it is never
   sent to the server, keeping the "loopback only, touches nothing" promise,
4. copies the full URL to the clipboard.

On boot, if `location.hash` carries `#s=`, decode it and load those fields
instead of the default `name`+`email` starter. A malformed fragment is ignored
(fall back to the default), never thrown — a truncated paste should not break
the page.

The encoding is the spec object, not rendered YAML, so it round-trips exactly
back into the editor controls.

## Testing

- diff package: identical inputs yield no findings; a removed/added column is an
  error; numeric drift past tolerance is a warn and within tolerance is not; a
  new category is a warn; a type flip (numeric→string) is an error.
- CLI: exit 1 on an error finding, 0 on warnings only; `--format json` parses;
  `--tolerance` widens the numeric threshold.
- MCP: two inline datasets produce the same findings as the CLI on the same
  data.
- Share-link: `currentSpec()` → encode → decode round-trips to an equal object;
  a malformed fragment falls back to the default without error. (JS is covered
  by a small pure encode/decode function unit-tested in Go's httptest-served
  page only if a JS test harness exists; otherwise the encode/decode is written
  as one pure function and exercised manually, with the Go side asserting the
  page serves and the fragment format is stable.)
