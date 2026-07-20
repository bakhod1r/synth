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
synth.New(synth.WithSeed(42))
```

Sub-streams are seeded independently, so adding a new field doesn't shift the values of existing ones.

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
go get github.com/bakhod1r/synth
```

## Quick start

```go
package main

import (
	"context"
	"os"

	"github.com/bakhod1r/synth"
)

func main() {
	gen := synth.New(
		synth.WithLocale("uz_UZ"),
		synth.WithSeed(42),
	)

	gen.Stream(context.Background(), synth.Users(1_000_000), synth.JSONL(os.Stdout))
}
```

## Data domains

| Domain | Examples |
| --- | --- |
| People | name, email, phone, address, national ID |
| Payments | card, IBAN, merchant, currency, amount |
| Transactions | ledger entries, timestamps, statuses |
| Business | companies, invoices, orders, inventory |

## Sinks

```go
synth.Kafka(brokers, topic)   // stream to Kafka
synth.Postgres(dsn, table)    // batched COPY into Postgres
synth.CSV(w)                  // CSV
synth.JSONL(w)                // newline-delimited JSON
```

## OpenAPI payloads

```go
spec, _ := synth.LoadOpenAPI("openapi.yaml")
payload, _ := spec.Payload("POST", "/v1/payments")
```

## License

MIT — see [LICENSE](LICENSE).
