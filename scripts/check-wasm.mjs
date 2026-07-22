// Exercises the WebAssembly module directly, without a browser.
//
//   scripts/build-pages.sh dist/pages
//   node scripts/check-wasm.mjs dist/pages
//
// This is the Go side of the static build. The browser glue is thin, but a
// route that answers wrongly here answers wrongly everywhere — and the hosted
// page runs the same generator the library does, so a broken module is a broken
// release rather than only a broken demo.
//
// It asserts rather than prints, so CI fails instead of logging something
// nobody reads.
import fs from 'fs';
import path from 'path';

const dir = process.argv[2];
globalThis.require = (await import('module')).createRequire(import.meta.url);
globalThis.fs = fs;
globalThis.path = path;
globalThis.TextEncoder = TextEncoder;
globalThis.TextDecoder = TextDecoder;
globalThis.performance = performance;


await import(path.resolve(dir, 'wasm_exec.js'));

const go = new globalThis.Go();
const bytes = fs.readFileSync(path.join(dir, 'synth.wasm'));
const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
go.run(instance);
for (let i = 0; i < 200 && !globalThis.synthWasm; i++) {
  await new Promise((r) => setTimeout(r, 10));
}
if (!globalThis.synthWasm) { console.log('FAIL: module did not export'); process.exit(1); }

const call = (m, p, b) => globalThis.synthWasm.handle(m, p, b || '');
const json = (m, p, b) => { const r = call(m, p, b); return { status: r.status, data: JSON.parse(r.body) }; };

let failed = 0;
const check = (name, ok, detail) => {
  console.log(`${ok ? '  ok  ' : '  FAIL'} ${name}${detail ? '  ' + detail : ''}`);
  if (!ok) failed++;
};

console.log('module version:', globalThis.synthWasm.version);

const types = json('GET', '/api/types').data;
check('catalog', types.length > 200, `${types.length} types`);

const locales = json('GET', '/api/locales').data;
check('locales', locales.length > 40, `${locales.length}`);
check('locales sorted', JSON.stringify(locales) === JSON.stringify([...locales].sort()));

const presets = json('GET', '/api/presets').data;
check('presets', presets.length > 0 && presets.every((p) => p.yaml), `${presets.length}`);

const spec = JSON.stringify({
  name: 'users', count: 3, locale: 'uz_UZ', seed: 42,
  fields: { ism: { kind: 'name' }, pinfl: { kind: 'pinfl' }, karta: { kind: 'card', mask: 'partial' } },
  order: ['ism', 'pinfl', 'karta'],
});
const prev = json('POST', '/api/preview', spec);
check('preview', prev.status === 200 && prev.data.length === 3);

// Locale coherence is the claim the whole library rests on, so the demo has to
// demonstrate it rather than merely load.
const row = prev.data[0];
check('uz_UZ card is a local scheme',
  /^(8600|9860)/.test(row.karta), row.karta);
check('uz_UZ national id is a 14-digit PINFL',
  /^\d{14}$/.test(row.pinfl), row.pinfl);
check('mask applied', row.karta.includes('*'), row.karta);

const csv = call('POST', '/api/generate', JSON.stringify({ ...JSON.parse(spec), count: 5, format: 'csv' }));
const lines = csv.body.trim().split('\n');
check('csv', csv.status === 200 && lines.length === 6 && lines[0] === 'ism,pinfl,karta');

const preset = json('GET', '/api/preview?preset=transaction&rows=2&seed=9');
check('preset masks by default',
  preset.data[0].card_number.includes('*'), preset.data[0].card_number);

// A seed must give the same rows, or the demo undersells the one property that
// distinguishes Synth from a hosted generator.
const a = json('POST', '/api/preview', spec).data;
const b = json('POST', '/api/preview', spec).data;
check('reproducible', JSON.stringify(a) === JSON.stringify(b));

// Bad input must be an error, never a crash: a panic inside wasm kills the
// module and every later request fails with an unrelated message.
check('unknown kind rejected',
  call('POST', '/api/preview', '{"count":1,"fields":{"a":{"kind":"nope"}},"order":["a"]}').status === 400);
check('unknown route rejected', call('GET', '/api/nope').status === 404);
check('junk body rejected', call('POST', '/api/preview', 'not json').status === 400);
check('empty body rejected', call('POST', '/api/preview', '').status === 400);
check('module still alive after errors',
  json('GET', '/api/types').data.length === types.length);

check('preview capped',
  json('POST', '/api/preview', JSON.stringify({ ...JSON.parse(spec), count: 100000 })).data.length === 100);

console.log(failed ? `\n${failed} check(s) failed` : '\nall checks passed');
process.exit(failed ? 1 : 0);
