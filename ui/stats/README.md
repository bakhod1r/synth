# Counters that survive a restart

The workbench header shows how many times you have generated, how many rows, and
how many values. `synth ui` counts those in memory and forgets on exit. This
module keeps them in SQLite instead.

```bash
go install github.com/bakhodir/synth/ui/stats/cmd/synth-ui@latest
synth-ui                      # same page, counters at ~/.config/synth/stats.db
synth-ui --db ./project.db    # counters for one project
synth-ui --recent 20          # print the last runs and exit
```

## Why it is a separate module

`modernc.org/sqlite` brings ten transitive dependencies. The core library has
two, and someone importing `synth` to generate a fixture should not pay for a
database they never touch. So this implements `ui.Recorder` from outside and is
wired in at the command — the same shape as `sink/parquet` and `mcp`.

The driver is pure Go, so there is no cgo and no C toolchain to install.

## What is stored

Counts, and only counts:

| Column | |
|---|---|
| `at` | when the run finished |
| `name` | the table name from the spec |
| `rows`, `columns` | how much was produced |
| `format` | csv, jsonl, sql |
| `bytes`, `millis` | size and duration |

No generated value is ever written, and a test asserts the schema holds nothing
else. The file is on your own machine and nothing is sent anywhere.

`columns` is stored rather than only the product, because a total that threw
away its parts cannot answer a later question — "which runs were wide?" — and
re-deriving it is impossible.
