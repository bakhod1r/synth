# Synth Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Sof data provayder Go kutubxonasi — struct kiradi, realistik o'zaro bog'liq yozuvlar chiqadi (kod/fayl/stream).

**Architecture:** Uch qatlam — `reflectfe`/`infer` (frontend) → `schema` (IR) → `gen`+`providers`+`locale` (engine) → `encode` (chiqish). Engine struct bilmaydi, faqat IR.

**Tech Stack:** Go 1.25, standart kutubxona + `github.com/google/uuid`. Testlar `testing` bilan.

## Global Constraints

- Modul: `github.com/bakhod1r/synth`, Go 1.25.
- Tarmoqqa chiqmaydi, DB ulanmaydi, INSERT/DDL yo'q.
- Sof determinizm: seed + schema → bir xil natija.
- Har paket mustaqil test qilinadi.

---

### Task 1: Modul + schema IR
`schema` paketi: `Field{Name, Kind, Params, From, Match}`, `Schema{Fields}`, `Kind` enum.

### Task 2: providers — atomik generatorlar
`Provider` interfeys: `Generate(rng *rand.Rand) any`. Registry (nom→provider). Boshlang'ich: name, firstname, lastname, email, phone, city, postcode, uuid, int, float, bool, time, lorem, iban, card.

### Task 3: locale ma'lumotlari
`en_US`, `uz_UZ` — ismlar, shahar↔postcode juftliklari, tel prefiks. embed JSON.

### Task 4: infer — tagsiz xulosa
Sinonim jadvali + tur fallback + bog'liqlik topish (name→email, city→postcode, time tartibi).

### Task 5: reflectfe — struct→schema
Reflection + tag parse, tur bo'yicha kesh. Tag bo'lsa aynan, bo'lmasa infer.

### Task 6: gen — engine
Topologik tartib, `from`/`match`/`Ref` hal qilish, determinist rng per-record.

### Task 7: public API — Make/TryMake/Ref/WithSeed/Warnings
`synth` paketi: generics `Make[T]`.

### Task 8: encode — CSV/JSONL/SQL + Stream
`WriteCSV/WriteJSONL/WriteSQL`, `Stream[T].ToCSV`.

---

## Differentiator'lar (positioning — "yana bitta faker" emas)

Bular yadro ustiga quriladi, GitHub'da ajralib turish uchun:

- **Per-instance RNG** — global `rand` mutex bottleneck yo'q. Har `Generator`/goroutine o'z `rng.Rand`i (Task 1'da qilindi). Maqsad: 10M+ rows/min.
- **Worker pool** — `synth.Pool(n)` parallel generatsiya, har worker o'z RNG'si bilan, determinizm saqlanadi.
- **Correlated data** — region→city→postcode→phone bitta `locale.Place`dan (providers'da qilindi). Bu profesional ko'rinish beradi.
- **UZ fintech** — HUMO (9860) / UZCARD (8600) Luhn-valid kartalar, UZ IBAN (28), passport AA1234567 (qilindi).
- **Zero-alloc / benchmark** — README'ga `go-faker` / `jaswdr` bilan taqqoslash jadvali, `testing.B` benchmark'lar. Hotpath'da allocatsiyani kamaytirish.
- **Streaming sink** — `Stream[User](n).ToCSV/ToJSONL` doimiy xotirada (Task 8). Postgres/Kafka sink — **keyingi spec** (yadro sof provayder qoladi; sink alohida paket, ixtiyoriy).
- **OpenAPI-driven** — `schema.Schema`ni OpenAPI'dan chiqarish — keyingi spec (yadro IR arzon qiladi).
- **Fluent + Fill API** — `synth.New(Config{Seed, Locale})`, `g.Person().Name()`, va `synth.Fill(&u)` struct to'ldirish.

**MVP ketma-ketligi:** yadro (Task 1–8) → pool + benchmark → README taqqoslash → OpenAPI/CLI (alohida spec).
