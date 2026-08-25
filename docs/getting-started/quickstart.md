# Quick start

Synth is a **pure data provider**: it never connects to a database, never runs
`INSERT`, never reads DDL from a server. You hand it a plain Go struct; it hands
you coherent records — in memory, to a file, or streamed. Loading is a separate
tool's job.

```go
package main

import (
	"time"

	"github.com/bakhod1r/synth"
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

## Referential integrity

```go
users  := synth.Make[User](10_000, synth.WithSeed(1))
orders := synth.Make[Order](500_000, synth.Ref(users, "UserID")) // every FK is real
```

More in [Referential integrity](../guide/referential-integrity.md).

## Temporal causality

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

More in [Temporal causality](../guide/temporal.md).

## Fluent single values

For a one-off value rather than a dataset:

```go
g := synth.New(synth.Config{Seed: 42, Locale: "uz_UZ"})
g.Name()      // "Azizbek Karimov"
g.Phone()     // "+998901234567"
g.Card()      // Luhn-valid
g.Amount(1000, 500000)
```

## Without Go

Describe the data in YAML and run the CLI — see
[CLI and YAML specs](cli.md).

```bash
synth gen -s users.yaml -o users.csv -n 100000 --seed 42
```
