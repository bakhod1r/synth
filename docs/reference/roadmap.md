# Status and roadmap

## Implemented

- **Frontends** — Go structs with tagless inference, YAML specs and CLI
  (`synth gen`), OpenAPI-driven payloads, SQL DDL, JSON Schema, Avro, Protobuf,
  and real-data profiling.
- **Coherence** — referential integrity (`Ref`), temporal causality
  (`after=`/`gap=` lifecycle ordering), unique constraints (`unique` tag, PKs),
  `OneToMany` cardinality, nested structs and slices generated recursively, and
  locale coherence (country → region → city → postcode → phone).
- **Validity** — Luhn-valid cards (HUMO/UZCARD), mod-97 IBANs, and
  gender-coherent names.
- **Statistics** — Normal/LogNormal/Exponential/Zipf/Weighted distributions and
  chaos injection (`WithChaos`).
- **Coverage** — 264 field types across 52 locales (native names, dialing codes,
  currencies, capital regions), plus real-world datasets — recognizable values
  blended with combinatorial ones so repetition stays low across large datasets.
- **Custom types** — `Register`/`RegisterSet` for your own values.
- **Engine** — deterministic per-record RNG, parallel generation, and
  CSV/JSONL/SQL encoders with streaming.

## Roadmap

- Locale datasets for the culturally-specific catalog types beyond the ten
  locales currently covered.
- Protobuf `map<k,v>` fields.

## Out of scope

Network sinks (Kafka, Postgres) are intentionally **out of scope** — Synth stays
a pure provider; feed its output to your own loader. See
[Real-time pacing](../tools/rate.md) for the streaming-test shape that does not
require Synth to hold a connection.
