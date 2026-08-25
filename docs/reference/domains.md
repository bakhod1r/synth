# Data domains

264 field types across 52 locales.

| Domain | Examples |
| --- | --- |
| People | name, email, phone, address, national ID |
| Payments | card, IBAN, merchant, currency, amount |
| Transactions | ledger entries, timestamps, statuses |
| Business | companies, invoices, orders, inventory |

Alongside the combinatorial types there are **real-world datasets** — books,
movies, celebrities, brands, foods, animals, sports, universities, languages,
emoji — blended with combinatorial values so repetition stays low across large
datasets.

## Listing them at runtime

The type list is read from the provider registry, so it cannot drift from what
the engine supports:

- `synth ui` shows the whole palette, each type marked with whether its values
  actually follow the locale.
- The [MCP](../mcp.md) `list_types` tool returns the same list.
- `providers.LocalesFor(kind)` reports which locales a type has data for, and
  `providers.LocalizableKinds()` lists the kinds a locale reaches at all.

## Adding your own

See [Custom types](../guide/custom-types.md).
