// Exercises the package the way a consumer does, through its public exports.
//
// It asserts rather than prints, so a broken publish fails instead of logging
// something nobody reads.
import assert from 'node:assert/strict';
import {
  generate, generateAs, listTypes, listLocales, listPresets, version,
} from './index.js';

let failed = 0;
async function check(name, fn) {
  try {
    await fn();
    console.log(`  ok    ${name}`);
  } catch (err) {
    failed++;
    console.log(`  FAIL  ${name}\n        ${err.message}`);
  }
}

console.log('synth', await version());

await check('catalog', async () => {
  const types = await listTypes();
  assert.ok(types.length > 200, `${types.length} types`);
  assert.ok(types.every((t) => t.kind && t.category));
});

await check('locales are sorted', async () => {
  const locales = await listLocales();
  assert.ok(locales.length > 40);
  assert.deepEqual(locales, [...locales].sort());
});

await check('presets carry their yaml', async () => {
  const presets = await listPresets();
  assert.ok(presets.length > 0);
  assert.ok(presets.every((p) => p.name && p.yaml));
});

await check('generate from a preset', async () => {
  const rows = await generate({ preset: 'user', rows: 5, seed: 1 });
  assert.equal(rows.length, 5);
  assert.ok(rows[0].email.includes('@'));
});

await check('generate from a schema', async () => {
  const rows = await generate({
    rows: 3,
    schema: { name: { kind: 'name' }, city: { kind: 'city' } },
  });
  assert.equal(rows.length, 3);
  assert.deepEqual(Object.keys(rows[0]).sort(), ['city', 'name']);
});

// Coherence is the claim the library rests on. A package that generated
// unrelated fields would be a faker with extra steps.
await check('a record holds together', async () => {
  const [row] = await generate({
    rows: 1, locale: 'uz_UZ', seed: 7,
    schema: {
      name: { kind: 'name' }, phone: { kind: 'phone' },
      card: { kind: 'card' }, pinfl: { kind: 'pinfl' },
    },
  });
  assert.match(row.phone, /^\+998/, `phone ${row.phone} is not Uzbek`);
  assert.match(row.card, /^(8600|9860)/, `card ${row.card} is not a local scheme`);
  assert.match(row.pinfl, /^\d{14}$/, `pinfl ${row.pinfl} is not 14 digits`);
});

await check('the same seed reproduces', async () => {
  const a = await generate({ preset: 'order', rows: 20, seed: 42 });
  const b = await generate({ preset: 'order', rows: 20, seed: 42 });
  assert.deepEqual(a, b);
});

await check('different seeds differ', async () => {
  const a = await generate({ preset: 'order', rows: 5, seed: 1 });
  const b = await generate({ preset: 'order', rows: 5, seed: 2 });
  assert.notDeepEqual(a, b);
});

// The masking default has to survive the trip through JavaScript. A card number
// pasted into a log is exactly the accident it prevents.
await check('sensitive columns are masked by default', async () => {
  const rows = await generate({ preset: 'transaction', rows: 10, seed: 3 });
  for (const r of rows) {
    assert.ok(r.card_number.includes('*'), `unmasked card ${r.card_number}`);
  }
});

await check('unmasked is opt-in', async () => {
  const rows = await generate({ preset: 'transaction', rows: 5, seed: 3, unmasked: true });
  assert.ok(!rows[0].card_number.includes('*'));
});

await check('csv output', async () => {
  const csv = await generateAs('csv', {
    rows: 4, name: 'people',
    schema: { name: { kind: 'name' }, email: { kind: 'email' } },
  });
  const lines = csv.trim().split('\n');
  assert.equal(lines[0], 'name,email');
  assert.equal(lines.length, 5);
});

await check('jsonl output', async () => {
  const jsonl = await generateAs('jsonl', { preset: 'user', rows: 3, seed: 1 });
  const lines = jsonl.trim().split('\n');
  assert.equal(lines.length, 3);
  assert.ok(JSON.parse(lines[0]).email);
});

await check('sql output', async () => {
  const sql = await generateAs('sql', {
    rows: 2, name: 'people', schema: { name: { kind: 'name' } },
  });
  assert.match(sql, /INSERT INTO/i);
  assert.match(sql, /people/);
});

// Ambiguous input is a mistake worth naming. Preferring one silently would
// generate from a schema the caller did not ask for.
await check('preset and schema together are refused', async () => {
  await assert.rejects(
    () => generate({ preset: 'user', schema: { a: { kind: 'city' } } }),
    /exactly one/);
  await assert.rejects(() => generate({}), /exactly one/);
});

await check('an unknown kind is an error, not a crash', async () => {
  await assert.rejects(() => generate({ schema: { a: { kind: 'nope' } } }), /nope/);
  // The module must survive it: a panic inside wasm kills it, and every later
  // call then fails with an unrelated message.
  const rows = await generate({ preset: 'user', rows: 1 });
  assert.equal(rows.length, 1);
});

await check('an unknown preset is an error', async () => {
  await assert.rejects(() => generate({ preset: 'nope' }));
});

console.log(failed ? `\n${failed} failed` : '\nall passed');
process.exit(failed ? 1 : 0);
