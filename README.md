# Synth

![Synth](assets/banner.png)

**Fakers give you random strings. Synth gives you a dataset that holds together.**

A user's email matches their name. A transaction points at a real account, in that account's currency, with a timestamp after the account was opened. Every card passes Luhn, every IBAN passes its checksum. That's the difference: fakers generate *fields*, Synth generates *records that reference each other* — at millions of rows per run, streamed, in constant memory.

Synth generates *realistic*, locale-aware data — users, payments, transactions, business records — instead of random fake values. It streams millions of records straight to Kafka, Postgres, CSV, or JSONL with minimal memory usage, and can produce valid request payloads from OpenAPI schemas.

## How it differs from a faker

| | Faker libraries | Synth |
| --- | --- | --- |
| Scope | one field at a time | whole records, with relations between them |
| Consistency | `Name()` and `Email()` are unrelated | email derives from the name, city matches the postcode |
| Validity | random digits | Luhn-valid cards, checksum-valid IBANs, real BIN ranges |
| Volume | build a slice in memory | streamed, constant memory at 100M+ rows |
| Output | strings you wire up yourself | Kafka, Postgres, CSV, JSONL sinks |
| Schemas | none | generates valid payloads from your OpenAPI spec |

## Features

### Referential integrity
Records are generated as a graph, not a list. Declare a relation once and every child row points at a parent that actually exists.

```go
users := synth.Users(10_000)
synth.Orders(500_000, synth.BelongsTo(users, "user_id"))
```

Foreign keys resolve. Cardinality is controllable (`OneToMany`, `Weighted`). Load a parent table straight into Postgres and the child table's FK constraints pass on the first try.

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

```go
synth.Field("amount", synth.LogNormal(mu, sigma))
synth.Field("country", synth.Zipf())         // a few countries dominate
synth.Field("status", synth.Weighted(map[string]float64{
    "settled": 0.94, "pending": 0.05, "failed": 0.01,
}))
```

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

### Test-ready sinks
Kafka, Postgres (batched `COPY`), CSV, and JSONL out of the box, behind one interface — write your own in a few lines.

### Edge-case injection
Testing the happy path is the easy part. Ask for the values that break parsers: unicode names, emoji, RTL text, empty strings, boundary numerics, nulls in nullable columns.

```go
synth.New(synth.WithChaos(0.02))   // 2% of records carry a nasty value
```

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

### Fluent single values

```go
g := synth.New(synth.Config{Seed: 42, Locale: "uz_UZ"})
g.Name()      // "Azizbek Karimov"
g.Phone()     // "+998901234567"
g.Card()      // Luhn-valid
g.Amount(1000, 500000)
```

## Benchmarks

Measured on Apple Silicon (`go test -bench`), Go 1.25:

| Operation | Throughput | allocs/op |
| --- | --- | --- |
| Single value (`g.Name()`) | ~15.6M ops/sec (64 ns) | 2 |
| `Make[User]` (10 fields), serial | ~560K records/sec | — |
| `MakeParallel[User]`, 8 cores | ~1.28M records/sec | — |

Per-instance RNG means no global-`rand` mutex — parallel generation scales, and
same-seed output is byte-identical regardless of worker count.

## Data domains

| Domain | Examples |
| --- | --- |
| People | name, email, phone, address, national ID |
| Payments | card, IBAN, merchant, currency, amount |
| Transactions | ledger entries, timestamps, statuses |
| Business | companies, invoices, orders, inventory |

## Output

Synth writes **files** — it never opens a network or DB connection.

```go
synth.WriteCSV("users.csv", users)
synth.WriteJSONL("users.jsonl", users)
synth.WriteSQL("users.sql", "users", users) // INSERT statements you run yourself

synth.Stream[User](100_000_000).ToCSV("users.csv") // constant memory
```

## Status & roadmap

**Implemented:** struct frontend with tagless inference, referential integrity
(`Ref`), locale coherence (country → region → city → postcode → phone),
Luhn-valid cards (HUMO/UZCARD), mod-97 IBANs, deterministic per-record RNG,
parallel generation, CSV/JSONL/SQL encoders and streaming, `uz_UZ` + `en_US`.

**Roadmap (separate specs):** statistical distributions (LogNormal, Zipf,
Weighted), edge-case/chaos injection, OpenAPI-driven payloads, more locales.
Network sinks (Kafka, Postgres) are intentionally **out of scope** — Synth stays
a pure provider; feed its output to your own loader.

## License

MIT — see [LICENSE](LICENSE).
