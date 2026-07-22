// Routes the page's /api/ calls into the WebAssembly build instead of a server.
//
// The workbench's app.js is byte-identical between the local server and this
// page. It calls fetch('/api/preview') either way; here that call lands in Go
// compiled to wasm, running in the tab. Keeping the frontend unaware of which
// backend it has is what stops the hosted demo drifting from the real tool.
//
// Loaded only by the static build. `synth ui` never sees this file.
'use strict';

(function () {
  const realFetch = window.fetch.bind(window);
  let ready = null;

  // boot loads the runtime and the module, and resolves once Go has exported
  // its handler. Every /api/ call waits on the same promise, so a click during
  // the download queues rather than failing.
  function boot() {
    if (ready) return ready;
    ready = (async () => {
      const go = new Go();
      const result = await WebAssembly.instantiateStreaming(
        realFetch('synth.wasm'), go.importObject);
      go.run(result.instance); // never resolves by design; the module parks
      // go.run schedules the Go main on a microtask, so the export is not
      // visible on the very next line.
      for (let i = 0; i < 200 && !window.synthWasm; i++) {
        await new Promise((r) => setTimeout(r, 10));
      }
      if (!window.synthWasm) throw new Error('the generator did not start');
      return window.synthWasm;
    })();
    return ready;
  }

  window.fetch = async function (input, init) {
    const url = typeof input === 'string' ? input : input.url;
    if (!url.startsWith('/api/') && !url.includes('/api/')) {
      return realFetch(input, init);
    }
    const path = url.slice(url.indexOf('/api/'));
    const method = (init && init.method) || 'GET';
    const body = (init && init.body) || '';

    let wasm;
    try {
      wasm = await boot();
    } catch (err) {
      return new Response(String(err), { status: 503 });
    }

    // The call is synchronous inside wasm. Yielding first lets the browser
    // paint the "Generating…" state, so a large run does not look frozen.
    await new Promise((r) => setTimeout(r, 0));
    const res = wasm.handle(method, path, typeof body === 'string' ? body : '');
    // The page reads a preview with res.json() and a download with res.blob().
    // A blob does not care about the type, and the requested format lives in
    // the POST body rather than the path, so guessing from the path would be
    // wrong more often than not.
    const type = path.startsWith('/api/generate')
      ? 'application/octet-stream'
      : 'application/json';
    return new Response(res.body, { status: res.status, headers: { 'Content-Type': type } });
  };
})();
