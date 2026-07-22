# @bakhod1r/synth

Realistic, coherent, referentially-consistent fake data for JavaScript. The Go
generator compiled to WebAssembly, so it runs where you call it.

```bash
npm install @bakhod1r/synth
```

```js
import { generate } from '@bakhod1r/synth';

const users = await generate({ preset: 'user', rows: 100, locale: 'uz_UZ' });
// [{ full_name: 'Sevara Valiyeva', email: 'sevara.valiyeva13@umail.uz',
//    phone: '+998957086305', city: "Farg'ona", ... }]
```

## Why not a faker

A faker gives you fields. This gives you records that hold together — the email
derives from the name, the city matches the postcode, the phone has the right
country code, and every card passes Luhn.

```js
const [row] = await generate({
  rows: 1, locale: 'uz_UZ',
  schema: {
    name:  { kind: 'name' },
    phone: { kind: 'phone' },
    card:  { kind: 'card' },
    pinfl: { kind: 'pinfl' },
  },
});
// name  "Umida Hakimova"
// phone "+998901234567"      the locale's country code
// card  "9860…"              HUMO, a real Uzbek scheme, Luhn-valid
// pinfl "32707505150568"     14 digits with a correct check digit
```

## Nothing is sent anywhere

There is no service behind this package. The generator runs in your process, so
a schema describing real columns never leaves it — a different guarantee from a
hosted generator promising not to look.

## API

```js
import {
  generate, generateAs, listTypes, listLocales, listPresets, version,
} from '@bakhod1r/synth';
```

### `generate(options)`

| option | |
| --- | --- |
| `preset` | A built-in schema. Use this **or** `schema`, not both. |
| `schema` | `{ column: { kind, ...params } }` |
| `rows` | Default 10 |
| `locale` | `"uz_UZ"`, `"de_DE"`, … Default `en_US` |
| `seed` | The same seed always gives the same rows |
| `unmasked` | Return raw card numbers and identifiers. Off by default |

### `generateAs(format, options)`

The same, serialised: `'csv'`, `'jsonl'`, `'sql'` or `'json'`.

```js
import { writeFile } from 'node:fs/promises';

await writeFile('users.csv',
  await generateAs('csv', { preset: 'user', rows: 10_000 }));
```

### The rest

`listTypes()` — the 260 column types, each saying whether it follows the locale.
Most do not, and the flag says so rather than letting you assume.

`listLocales()` — the 52 data locales, sorted.

`listPresets()` — the built-in schemas with their YAML, to read and edit rather
than guess at.

`version()` — the compiled generator's version, for a bug report.

## Masking

Card numbers and national identifiers come back masked:

```js
await generate({ preset: 'transaction', rows: 1 });
// card_number: "5425********7019"
// national_id: "75aad2c5c2b86b1a…"   SHA-256
```

A fixture reaches a ticket, a screenshot or a log sooner or later, so the safe
form is the default and nobody has to remember. `unmasked: true` when a test
genuinely needs the raw value — to check that a validator accepts it.

Per column: `mask: 'partial' | 'hash' | 'redact' | 'token'`. A hash is scoped to
its column, so the same value in two columns gives different digests and a
masked export cannot be re-joined. Add `secret` to make the digest impossible to
recompute — without a key, a short value like a national id can simply be
guessed and hashed until it matches.

## Size

About 1.6 MB gzipped, loaded once on the first call.

## Also available as

- A Go library — `go get github.com/bakhod1r/synth`
- A CLI — `synth gen --preset user -n 1000000 -o users.csv`
- An MCP server, for assistants
- [A page you can just open](https://bakhod1r.github.io/synth)

## Licence

MIT
