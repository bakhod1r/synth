# Outputs

Synth writes **files** — it never opens a network or DB connection.

```go
synth.WriteCSV("users.csv", users)
synth.WriteJSONL("users.jsonl", users)
synth.WriteSQL("users.sql", "users", users) // INSERT statements you run yourself

synth.Stream[User](100_000_000).ToCSV("users.csv") // constant memory
```

CSV, JSONL, SQL `INSERT` files, Parquet, Postgres `COPY` files, and
Debezium-shaped [CDC events](../tools/cdc.md) — every one of them a file.
Handing it to your loader is the last step, and it is yours.

## Postgres COPY

For bulk loading, `COPY` is what Postgres wants — an `INSERT` per row is the
slowest path the server offers:

```sh
synth gen -s users.yaml -n 100000000 -o users.pgbin   # binary COPY
synth gen -s users.yaml -n 100000000 -o users.pgcopy  # text COPY
```

Both write a matching `CREATE TABLE` next to the data as `users.pgbin.sql`:

```sh
psql -f users.pgbin.sql
```

That pairing is not a convenience. Binary `COPY` carries no type names, so the
table's column types are what the server decodes the bytes as, and a table built
by hand that differs in one column means a rejected file. Synth generates the
DDL and the encoding from the same type table, so they cannot disagree.

## Parquet

A first-class output — pick it by extension or `-f parquet`:

```sh
synth gen -s users.yaml -n 100000 -o users.parquet
synth gen --preset user -n 50 -f parquet -o users.parquet
```

The same writer is available from Go:

```go
parquet.WriteStructs("users.parquet", users)              // from Go structs
parquet.WriteRows("users.parquet", spec.Columns(), rows)  // from YAML/DDL/profiling
```

Column types are inferred (int64, double, boolean, string), so query engines see
real types rather than everything-as-string. Uploading the file to S3, MinIO or
a warehouse is your loader's job.

!!! warning "Parquet needs a real path"
    A Parquet file carries a footer, so it does not stream to stdout or through
    the gzip/zstd sink, and `--append` does not apply to it.

## Compression

Large outputs compress on the way out — name the file and Synth does the rest:

```sh
synth gen -s users.yaml -n 100000000 -o users.jsonl.zst
synth gen -s users.yaml -n 100000000 -o users.csv.gz
```

## Format summary

| Format | Pick it with | Streams | `--append` |
| --- | --- | --- | --- |
| CSV | `.csv`, `-f csv` | yes | yes |
| JSONL | `.jsonl`, `-f jsonl` | yes | yes |
| SQL `INSERT` | `.sql`, `-f sql` | yes | yes |
| Postgres text `COPY` | `.pgcopy`, `-f pgcopy` | yes | no |
| Postgres binary `COPY` | `.pgbin`, `-f pgcopy-binary` | yes | no |
| Parquet | `.parquet`, `-f parquet` | no | no |
| CDC events | `synth cdc` | yes | — |
