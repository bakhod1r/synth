# Synth

![Synth](assets/banner.png)

**Fakers give you random strings. Synth gives you a dataset that holds together.**

A user's email matches their name. A transaction points at a real account, in that account's currency, with a timestamp after the account was opened. Every card passes Luhn, every IBAN passes its checksum. That's the difference: fakers generate *fields*, Synth generates *records that reference each other* — at millions of rows per run, streamed, in constant memory.

Synth generates *realistic*, locale-aware data — users, payments, transactions, business records — instead of random fake values. It streams millions of records to CSV, JSONL, SQL `INSERT` files, or Parquet with minimal memory usage, and can produce valid request payloads from OpenAPI schemas.

## How it differs from a faker

| | Faker libraries | Synth |
| --- | --- | --- |
| Scope | one field at a time | whole records, with relations between them |
| Consistency | `Name()` and `Email()` are unrelated | email derives from the name, city matches the postcode |
| Validity | random digits | Luhn-valid cards, checksum-valid IBANs, real BIN ranges |
| Volume | build a slice in memory | streamed, constant memory at 100M+ rows |
| Output | strings you wire up yourself | CSV, JSONL, SQL, Parquet, CDC files |
| Schemas | none | generates valid payloads from your OpenAPI spec |

## Features

### Referential integrity
Records are generated as a graph, not a list. Declare a relation once and every child row points at a parent that actually exists.

```go
users := synth.Users(10_000)
synth.Orders(500_000, synth.BelongsTo(users, "user_id"))
```

Foreign keys resolve. Cardinality is controllable (`OneToMany`, `Weighted`). Load the exported parent table into Postgres with your own loader and the child table's FK constraints pass on the first try.

### Temporal causality
Timestamps aren't random points in a range — they respect the order events can happen in. An order is `created → paid → shipped → delivered`, each strictly after the last, with realistic gaps. Accounts are never used before they're opened, and refunds never precede their charge.

```go
synth.Orders(1_000, synth.Timeline("2026-01-01", "2026-07-01"), synth.Lifecycle(synth.OrderFlow))
```

### Format-valid values
Every generated identifier passes the check a real system would run on it:

- Credit cards — Luhn-valid, issued from real BIN ranges per brand
- IBANs — mod-97 checksum, correct per-country length and BBAN layout
- National IDs, VAT numbers, tax IDs — country-specific check digits
- Emails, URLs, phone numbers — RFC / E.164 conformant

### Locale coherence
Locale isn't just a name list. Pick `uz_UZ` and you get Uzbek names, `+998` phone numbers, Tashkent districts, UZS amounts, and postcodes that match the city they're attached to — consistently across every field of the record.

### Statistical shape
Real data isn't uniform. Synth draws from distributions so your test data stresses the same paths production does.

Via tags:

```go
type Txn struct {
    Amount   float64 `synth:"amount,dist=lognormal,mu=10,sigma=1"` // long tail
    Status   string  `synth:"enum,choices=settled|pending|failed,weights=0.94|0.05|0.01"`
    Category string  `synth:"enum,choices=a|b|c|d,dist=zipf,s=1.2"` // hot keys
}
```

Or in code:

```go
synth.Make[Txn](1_000_000, synth.Weighted("Status", map[string]float64{
    "settled": 0.94, "pending": 0.05, "failed": 0.01,
}))
```

Distributions: `normal`, `lognormal`, `exp` (numeric fields) and `zipf` /
explicit `weights` (enums).

Long tails, hot keys, and skew are what break partitioning and query planners — uniform fakers never surface those bugs.

### Constant-memory streaming
Records are pushed through a pipeline, never accumulated. Generating 100M rows uses the same memory as generating 1K. Generation is sharded across cores, and sinks batch and backpressure independently.

### Deterministic and reproducible
A seed fully determines the output. The same seed produces byte-identical data across runs, machines, and Go versions — so a failing CI run is reproducible locally, and golden-file tests stay stable.

```go
synth.Make[User](1000, synth.WithSeed(42))
```

Each record is seeded independently from the base seed, so parallel generation is byte-identical to serial output.

### Schema-driven generation
Point Synth at a schema instead of hand-writing generators:

- **OpenAPI** — valid request bodies for every endpoint, respecting `format`, `pattern`, `enum`, `minimum`, and `required`
- **SQL DDL** — read `CREATE TABLE`, infer types, honor `NOT NULL`, `UNIQUE`, `CHECK`, and FK constraints
- **Go structs** — generate from your existing domain types via tags

### Test-ready outputs
CSV, JSONL, SQL `INSERT` files, Parquet, and Debezium-shaped CDC events — every
one of them a file. Synth never opens a database or network connection; handing
the file to your loader is the last step, and it is yours.

### Edge-case injection
Testing the happy path is the easy part. Ask for the values that break parsers: unicode names, emoji, RTL text, empty strings, boundary numerics, nulls in nullable columns.

```go
synth.New(synth.WithChaos(0.02))   // 2% of records carry a nasty value
```

## Browser workbench

```bash
synth ui            # then open http://127.0.0.1:8080
```

A local page with the type palette on the left, the schema in the middle, and a
live preview on the right that regenerates as you type. The seed is shown and
editable, because reproducibility is the thing a hosted generator cannot give
you — the UI teaches it rather than hiding it.

The palette is read from the provider registry itself, so it cannot drift from
what the engine actually supports, and each type is marked with whether its
values really follow the locale. Most do not, and saying so beats letting you
assume otherwise.

**The server binds `127.0.0.1` and refuses anything else** — `Serve("0.0.0.0:8080")`
returns an error, and a test enforces it. The page is embedded in the binary
with inline CSS and JavaScript: no CDN, no fonts, no telemetry, no outbound
request of any kind, and a test asserts the page contains no external origin.
The browser connects in; Synth never connects out.

## Install

```bash
go get github.com/bakhodir/synth
```

## Quick start

Synth is a **pure data provider**: it never connects to a database, never runs
`INSERT`, never reads DDL. You hand it a plain Go struct; it hands you coherent
records — in memory, to a file, or streamed. Loading is a separate tool's job.

```go
package main

import (
	"time"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `synth:"pk"`
	FirstName string
	Email     string `synth:"email,from=FirstName"` // derived from the name
	Phone     string
	Country   string
	Region    string
	City      string
	Postcode  string // stays coherent with Country/Region/City
	Card      string `synth:"card"` // Luhn-valid HUMO/UZCARD
	CreatedAt time.Time
}

func main() {
	// Tags are optional — untagged fields are inferred from name and type.
	users := synth.Make[User](10_000, synth.WithSeed(42), synth.WithLocale("uz_UZ"))

	synth.WriteCSV("users.csv", users)                       // to a file
	synth.Stream[User](1_000_000).ToJSONL("users.jsonl")     // constant memory
}
```

### Referential integrity

```go
users  := synth.Make[User](10_000, synth.WithSeed(1))
orders := synth.Make[Order](500_000, synth.Ref(users, "UserID")) // every FK is real
```

### Temporal causality

Timestamps respect the order events can happen in — a record's lifecycle stays
consistent instead of scattering random points in a range.

```go
type Order struct {
    CreatedAt   time.Time
    PaidAt      time.Time `synth:"time,after=CreatedAt,gap=1h..48h"`
    ShippedAt   time.Time `synth:"time,after=PaidAt,gap=1h..72h"`
    DeliveredAt time.Time `synth:"time,after=ShippedAt,gap=1h..120h"`
}
// CreatedAt < PaidAt < ShippedAt < DeliveredAt, always.
```

### Fluent single values

```go
g := synth.New(synth.Config{Seed: 42, Locale: "uz_UZ"})
g.Name()      // "Azizbek Karimov"
g.Phone()     // "+998901234567"
g.Card()      // Luhn-valid
g.Amount(1000, 500000)
```

## Benchmarks

Measured on Apple Silicon (M-series, 8 cores), Go 1.25, `go test -bench -benchmem`.
Each library fills the same four fields (name, email, phone, city).

**Struct filling** — one record from a struct definition:

| Library | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| `go-faker/faker` v4 | 10,848 | 8,778 | 116 |
| **Synth** | **2,494** | **2,001** | **34** |

**Per-field calls** — the fluent API:

| Library | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| `jaswdr/faker` v2 | 5,612 | 4,533 | 61 |
| **Synth** | **778** | **222** | **14** |

**Batch generation** — Synth's normal mode, where schema work is done once for
the whole run: 1,678 ns/record (~596K records/sec single-threaded), and
~1.28M records/sec across 8 cores with `MakeParallel`.

Reproduce with `cd benchcmp && go test -bench=. -benchmem -run=^$ .`
(the comparison lives in its own module, so the competing fakers never enter
this library's dependency graph — Synth depends only on `google/uuid` and
`yaml.v3`).

Per-instance RNG means no global-`rand` mutex — parallel generation scales, and
same-seed output is byte-identical regardless of worker count.

## Data domains

| Domain | Examples |
| --- | --- |
| People | name, email, phone, address, national ID |
| Payments | card, IBAN, merchant, currency, amount |
| Transactions | ledger entries, timestamps, statuses |
| Business | companies, invoices, orders, inventory |

## Learn from real data (profiling)

Point Synth at an **export** of a real table and it learns the shape — column
types, numeric ranges, null rates, and the real frequency of each category —
then generates synthetic rows that behave the same. Synth never connects to
your database; you produce the sample yourself:

```bash
psql -c "\copy (SELECT * FROM users LIMIT 10000) TO 'sample.csv' CSV HEADER"
```

```go
p, _ := synth.Profile("sample.csv")
rows, _ := p.Generate(1_000_000)   // same distribution, none of the real data
```

Low-cardinality columns keep their observed split (e.g. 80% active / 15%
inactive / 5% banned). Identifier-like columns are never echoed back, so real
values cannot leak into the output.

## Learned invariants (constraint mining)

Per-column profiling learns what each column looks like. It cannot see that a
`total` must agree with its line items, or that a refunded order must carry a
refund timestamp. `synth profile` mines those cross-column invariants from the
sample and writes them into the spec:

```yaml
constraints:
  - {kind: sum, parts: [subtotal, tax], whole: total}    # held over 48,912 rows
  - {kind: ordering, left: created_at, right: updated_at}  # held over 50,000 rows
  - {kind: implication, when: status, equals: "refunded", then: refund_at, exclusive: true}
```

Generation then enforces them. A total is **derived** from its parts rather
than generated independently; an out-of-order timestamp pair is swapped, which
preserves both values and their distribution instead of piling duplicates up at
a boundary.

Mining is conservative on purpose. A candidate is generated, then falsified
against the whole sample; an implication additionally needs a trigger group of
real size, and needs its target column to be empty *somewhere else* — otherwise
the column is simply always populated and the trigger has nothing to do with
it. Each surviving rule records how many rows it held over, so you can judge
it rather than trust it.

If a spec's constraints contradict each other, `Generate` returns an error
naming the unsatisfiable one. Emitting rows that quietly violate the spec would
be worse than failing: the data would look authoritative and be wrong.

## Audit an existing dataset (`synth verify`)

The rules that make Synth's output coherent are the rules worth checking on
data somebody else produced. `synth verify` is the generator run backwards:

```bash
synth verify -i orders.csv --ref user_id=users.csv:id
synth verify -i orders.csv -s orders.yaml -f json   # also re-check mined invariants
```

It reports failed check digits (Luhn, IBAN mod-97, EAN-13/UPC), unparseable
emails, URLs and IP addresses, dangling foreign keys, timestamp pairs in the
wrong order, and columns that carry no information. Parent tables are read from
their own files — nothing is queried.

Exit code 1 on any error, 0 when only warnings were found, so it drops into CI
without a wrapper. A degenerate column is a *warning*, not an error: real data
sometimes looks like that, and a tool that cries wolf gets muted.

Two rules keep it honest. A column is audited only when its values already
mostly match what its name claims, so a column called `card` holding loyalty
tiers is skipped rather than reported a million times. And a clean dataset must
produce an **empty** report — a check that fires on correct data is a false
positive and a bug here, not something for you to filter out.

## Anonymize a production dump (GDPR)

Hand Synth a real export and get one that is safe to share: personal data is
replaced with synthetic values of the same format, everything else is left
alone.

```go
m := synth.NewMasker("team-key", "en_US")
m.Rule(synth.MaskRule{Column: "notes", Strategy: synth.MaskRedact})
report, _ := m.File("dump.csv", "safe.csv")
```

- **Consistent** — the same input value always maps to the same replacement, so
  joins and foreign keys still line up (use the same key across related dumps).
- **Format-preserving** — an email stays an email, a card stays 16 digits.
- **Irreversible** — replacements come from a keyed hash, not an encoding.
- **Thorough** — PII is caught by column name, by value format, and inside free
  text (an email buried in a `notes` field is scrubbed too).

## Change events (CDC)

Generate a coherent insert/update/delete history in Debezium's envelope shape —
no database, no Kafka, just a file:

```go
synth.WriteCDC[User]("changes.jsonl", 10_000, synth.CDCConfig{
    Table: "users", UpdateRate: 0.3, DeleteRate: 0.1, Snapshot: 100,
})
```

A row exists before it is updated, updates carry the true `before` image, and
deleted rows are never touched again. LSNs and timestamps advance monotonically.

## Time travel

One seed already fixes a whole dataset. Snapshots make *time* another axis of
that determinism: ask for the table as it stood at any instant, or for what
changed between two.

```bash
synth snapshot -s orders.yaml --at 2026-01-01 -o jan.csv
synth snapshot -s orders.yaml --at 2026-07-01 -o jul.csv
synth snapshot -s orders.yaml --from 2026-01-01 --to 2026-07-01 -o changes.jsonl
```

```go
tl, _ := synth.Snapshot[Order](synth.SnapshotConfig{Rows: 100_000, Churn: 2, DeleteFrac: 0.1})
jan := tl.At(jan1)
jul := tl.At(jul1)
events := tl.Between(jan1, jul1)

tl.Apply(jan, events)   // == jul, exactly
```

That last line is the contract, and it is the point: replaying the log over the
earlier snapshot reproduces the later one. Migration and incremental-ETL tests
need a source of truth on both ends *and* the diff between them, and here all
three come from one seed.

The equivalence holds by construction rather than by luck. Each row's whole
life — when it was born, when it changed, whether it was deleted — is derived
from its index, and `At` and `Between` read that same life. Nothing is
simulated forward, so a snapshot a century out costs no more than one an hour
out. Consecutive ranges tile exactly, so you can walk a timeline in steps and
land on the same state as one jump.

## Real-time pacing

Deliver records over wall-clock time, the way a real event source would — for
streaming tests and consumer back-pressure experiments:

```go
synth.Rate[Event](synth.RateConfig{PerSecond: 5000, Jitter: 0.2}).
    Run(ctx, func(e Event) error { return myProducer.Send(e) })
```

Synth paces the handoff; where the events go is your code's decision.

## Input formats

Synth builds its schema from whichever definition you already have:

| Source | API |
| --- | --- |
| Go structs (tags optional) | `synth.Make[T]` |
| YAML spec | `synth.LoadYAML` |
| OpenAPI 3 | `synth.OpenAPI` |
| SQL DDL (`CREATE TABLE`) | `synth.LoadDDL` |
| JSON Schema | `synth.LoadSchema` |
| Avro schema | `synth.LoadSchema` |
| Protobuf (`.proto`) | `synth.LoadProto` |
| Real-data sample (CSV/JSONL) | `synth.Profile` |

## CLI & YAML specs

Describe data declaratively and generate it without writing Go:

```yaml
# users.yaml
name: users
count: 1000
locale: uz_UZ
fields:
  id:      { kind: uuid, pk: true }
  name:    { kind: name }
  email:   { kind: email, from: name }
  status:  { kind: enum, choices: [active, inactive], weights: [0.9, 0.1] }
  balance: { kind: amount, min: 0, max: 1000000, dist: lognormal, mu: 9, sigma: 1.2 }
```

```bash
go install github.com/bakhodir/synth/cmd/synth@latest
synth gen -s users.yaml -o users.csv          # or -f jsonl | sql
synth gen -s users.yaml -f sql -n 100000 --seed 42
```

Every library capability is reachable from the command line, and every
subcommand reads and writes **files** — none of them connects to anything.

```bash
# Learn a spec from a real export, then generate from the spec forever after.
synth profile -i prod_export.csv -o users.yaml
synth gen -s users.yaml -n 1000000 -o fake_users.csv

# Anonymize a real dump. The same --key across files keeps foreign keys joinable.
synth mask -i prod_export.csv -o safe.csv --key "$MASK_KEY"

# Generate a coherent insert/update/delete history in Debezium's envelope shape.
synth cdc -s users.yaml -o changes.jsonl -n 10000 --update-rate 0.3 --delete-rate 0.1
```

`synth mask` refuses to run without `--key` (an unkeyed run is not
reproducible) and refuses to write over its own input.

## Output

Synth writes **files** — it never opens a network or DB connection.

```go
synth.WriteCSV("users.csv", users)
synth.WriteJSONL("users.jsonl", users)
synth.WriteSQL("users.sql", "users", users) // INSERT statements you run yourself

synth.Stream[User](100_000_000).ToCSV("users.csv") // constant memory
```

### Parquet

Parquet lives in its own module, so its dependency stays optional — the core
library still needs only `google/uuid` and `yaml.v3`:

```bash
go get github.com/bakhodir/synth/sink/parquet
```

```go
parquet.WriteStructs("users.parquet", users)              // from Go structs
parquet.WriteRows("users.parquet", spec.Columns(), rows)  // from YAML/DDL/profiling
```

Column types are inferred (int64, double, boolean, string), so query engines
see real types rather than everything-as-string. Uploading the file to S3,
MinIO or a warehouse is your loader's job.

## Status & roadmap

**Implemented:** struct frontend with tagless inference, referential integrity
(`Ref`), temporal causality (`after=`/`gap=` lifecycle ordering), unique
constraints (`unique` tag, PKs), nested structs and slices (objects and arrays,
generated recursively), locale coherence (country → region → city → postcode → phone),
Luhn-valid cards (HUMO/UZCARD), mod-97 IBANs, deterministic per-record RNG,
parallel generation, CSV/JSONL/SQL encoders and streaming, `uz_UZ` + `en_US`.

statistical distributions (Normal/LogNormal/Exponential/Zipf/Weighted),
**50+ locales** (native names, dialing codes, currencies, capital regions), and
**custom types** via `Register`/`RegisterSet`, and **real-world datasets**
(books, movies, celebrities, brands, foods, animals, sports, universities,
languages, emoji) — recognizable values blended with combinatorial ones so
repetition stays low across large datasets.

```go
synth.RegisterSet("cinema", "Inception", "Interstellar", "Tenet", "Dune")
synth.Register("rating", func(r synth.R) any { return r.IntRange(1, 5) })
```

chaos injection (`WithChaos`), OpenAPI-driven payloads, a YAML frontend and CLI
(`synth gen`), nested structs/slices, and `OneToMany` cardinality.

a SQL-DDL frontend, JSON Schema and Avro frontends, real-data profiling,
gender-coherent names, and 173 field types across 52 locales.

**Roadmap:** gendered name banks for the remaining locales (14 of 52 currently
reach 1000+ name combinations); protobuf `map<k,v>` fields.
Network sinks (Kafka, Postgres) are intentionally **out of scope** — Synth stays
a pure provider; feed its output to your own loader.

## License

MIT — see [LICENSE](LICENSE).
