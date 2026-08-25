# Compare two datasets' shape (`synth diff`)

After changing a generator, or to guard a real feed against drift, `synth diff`
answers whether two files are shaped alike — columns, types, numeric ranges,
null rates, category sets — without comparing rows:

```sh
synth diff baseline.csv candidate.csv
synth diff baseline.csv candidate.csv --tolerance 0.2 -f json   # for CI
```

## Errors versus warnings

A column added, removed or retyped is an **error**; a range, null rate or
category set that moved past tolerance is a **warning**. Exit code 1 on any
error, 0 on warnings only — the same contract as [verify](verify.md), so a
pipeline fails on a structural break and passes on ordinary drift.

The [MCP server](../mcp.md) exposes the same as a `diff` tool over inline
datasets.
