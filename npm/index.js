// Synth for JavaScript: the Go generator compiled to WebAssembly.
//
// The point is that it runs where you call it. There is no service behind this
// package, so a schema describing real columns never leaves the process — which
// is a different guarantee from a hosted generator promising not to look.
//
// Works in Node and in a browser. The module loads on first use and is reused
// afterwards, so the ~1.6 MB download happens once.

let ready = null;

/**
 * boot loads the WebAssembly module. Every call shares one promise, so
 * concurrent calls during the initial load queue instead of loading twice.
 */
async function boot() {
  if (ready) return ready;
  ready = (async () => {
    const isNode =
      typeof process !== 'undefined' && process.versions && process.versions.node;

    if (isNode) {
      const { readFileSync } = await import('node:fs');
      const { fileURLToPath } = await import('node:url');
      const { dirname, join } = await import('node:path');
      const here = dirname(fileURLToPath(import.meta.url));

      // wasm_exec.js is a script, not a module: it assigns globalThis.Go.
      const shim = readFileSync(join(here, 'wasm_exec.js'), 'utf8');
      // eslint-disable-next-line no-new-func
      new Function(shim).call(globalThis);

      const go = new globalThis.Go();
      const bytes = readFileSync(join(here, 'synth.wasm'));
      const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
      go.run(instance);
    } else {
      const base = new URL('.', import.meta.url);
      await import(new URL('wasm_exec.js', base).href);
      const go = new globalThis.Go();
      const result = await WebAssembly.instantiateStreaming(
        fetch(new URL('synth.wasm', base)), go.importObject);
      go.run(result.instance);
    }

    // go.run schedules the Go main on a microtask, so the export is not visible
    // on the next line.
    for (let i = 0; i < 500 && !globalThis.synthWasm; i++) {
      await new Promise((r) => setTimeout(r, 5));
    }
    if (!globalThis.synthWasm) {
      throw new Error('synth: the generator did not start');
    }
    return globalThis.synthWasm;
  })();
  return ready;
}

async function call(method, path, body) {
  const wasm = await boot();
  const res = wasm.handle(method, path, body || '');
  if (res.status >= 400) {
    throw new Error(`synth: ${res.body}`);
  }
  return res.body;
}

function query(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}

/**
 * Generate records.
 *
 * Card numbers and national identifiers come back masked. Pass
 * `unmasked: true` when a test genuinely needs the raw value — to check that a
 * validator accepts it, for instance.
 *
 * @param {object} options
 * @param {string} [options.preset]  A built-in schema; see listPresets().
 * @param {object} [options.schema]  Columns as {name: {kind, ...}}. Use this or preset.
 * @param {number} [options.rows=10]
 * @param {string} [options.locale]  e.g. "uz_UZ". Default en_US.
 * @param {number} [options.seed]    The same seed always gives the same rows.
 * @param {boolean} [options.unmasked=false]
 * @returns {Promise<object[]>}
 */
export async function generate({
  preset, schema, rows = 10, locale, seed, unmasked = false, name = 'data',
} = {}) {
  if ((preset === undefined) === (schema === undefined)) {
    throw new Error('synth: pass exactly one of preset or schema');
  }
  if (preset !== undefined) {
    return JSON.parse(await call('GET', '/api/preview' + query({
      preset, rows, locale, seed, unmasked: unmasked || undefined,
    })));
  }
  const body = JSON.stringify({
    name, count: rows, locale, seed: seed ?? 0,
    fields: schema, order: Object.keys(schema),
  });
  return JSON.parse(await call('POST', '/api/preview' + query({
    unmasked: unmasked || undefined,
  }), body));
}

/**
 * Generate and serialise in one step, for writing a fixture straight to disk.
 *
 * @param {'csv'|'jsonl'|'sql'|'json'} format
 * @returns {Promise<string>}
 */
export async function generateAs(format, options = {}) {
  const { preset, schema, rows = 10, locale, seed, unmasked = false, name = 'data' } = options;
  if ((preset === undefined) === (schema === undefined)) {
    throw new Error('synth: pass exactly one of preset or schema');
  }
  if (preset !== undefined) {
    return call('GET', '/api/generate' + query({
      preset, rows, locale, seed, format, unmasked: unmasked || undefined,
    }));
  }
  const body = JSON.stringify({
    name, count: rows, locale, seed: seed ?? 0, format,
    fields: schema, order: Object.keys(schema),
  });
  return call('POST', '/api/generate' + query({ unmasked: unmasked || undefined }), body);
}

/** The generatable column types, each saying whether it follows the locale. */
export async function listTypes() {
  return JSON.parse(await call('GET', '/api/types'));
}

/** The available data locales, sorted. */
export async function listLocales() {
  return JSON.parse(await call('GET', '/api/locales'));
}

/** The built-in schemas, each with its YAML — a starting point to edit. */
export async function listPresets() {
  return JSON.parse(await call('GET', '/api/presets'));
}

/** The version of the compiled generator, for a bug report. */
export async function version() {
  const wasm = await boot();
  return wasm.version;
}
