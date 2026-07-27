# Plan: cross-run FK, append mode, soft deletes

Spec: `docs/superpowers/specs/2026-07-26-cross-run-fk-and-soft-delete-design.md`

TDD per phase, each committed on its own.

## Phase 1 — FK source (`--fk`)

1. `RefValues(field, values)` option in synth.go, setting `Field.FromRef`.
2. `YAMLSpec.Generate` applies refs from the config (today only `Make` does).
3. CLI: parse `--fk col=parent.csv:key`, load with `constraint.LoadSample`,
   pull the key column into `[]any`, pass as `RefValues`.
4. Errors: missing file, missing key column, empty parent.

Tests: every generated child value is in the parent key set; the three error
cases fail before generation.

## Phase 2 — append mode (`--append`)

1. `state.go` in cmd/synth: read/write `<out>.synthstate` JSON
   (`rows`, `pk_high`, `seed`, `pk_column`).
2. `Generate` gains a record-index offset so row seeding starts above the prior
   run; integer PKs continue above `pk_high`.
3. CLI: on `--append`, read state, set offset, append to the file without a new
   CSV header, rewrite state.
4. Seed-mismatch error.

Tests: 2×n runs give 2n distinct rows and PKs; state round-trips; seed mismatch
errors; append with no state is a clean first run.

## Phase 3 — soft delete (`--soft-delete`)

1. `Config.SoftDeleteColumn` (default `deleted_at`).
2. In `Stream.Next`, a drawn delete under soft mode stamps the column, emits
   `op=u`, and removes the row from the mutable pool so it is not touched again.
3. CLI flag `--soft-delete`.

Tests: a delete emits `op=u` with `deleted_at` and no `op=d`; the row is not
mutated again; default behaviour unchanged.

## Phase 4 — docs

README (a cross-run FK example, an append example), CHANGELOG under Unreleased,
CLI usage text and flag help.
