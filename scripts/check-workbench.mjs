// Drives the workbench page against a running server, so the generate flow is
// checked the way a person uses it rather than by reading the code.
//
//   go run ./cmd/synth ui --port 7777 &
//   cd scripts && npm install && npm run check-workbench
//
// What it asserts, and why each one is worth asserting:
//
//   - Download starts hidden. A button that regenerates on every click makes it
//     impossible to tell whether you are saving what you just looked at.
//   - A second click during a run is ignored, so one press is one dataset.
//   - Any edit retracts the download. Otherwise the file you save belongs to an
//     earlier schema and nothing on the page says so.
//   - Download does not re-request. The reported size and timing must describe
//     the same bytes that reach the disk.
//
// The "Not implemented: navigation" line from jsdom is expected: it is jsdom
// declining to follow the download anchor, not a fault in the page.
import { JSDOM, VirtualConsole } from 'jsdom';

const calls = {};

const vc = new VirtualConsole();
vc.on('jsdomError', (e) => console.log('PAGE ERROR:', e.message));

const ORIGIN = process.env.SYNTH_UI || 'http://127.0.0.1:7777';

const dom = await JSDOM.fromURL(ORIGIN + '/', {
  runScripts: 'dangerously',
  resources: 'usable',
  virtualConsole: vc,
  pretendToBeVisual: true,
  beforeParse(win) {
    // jsdom ships no fetch, Blob or URL.createObjectURL. Shim them onto the
    // window before the page's own scripts run, resolving relative paths
    // against the real server so the page talks to the actual API.
    win.fetch = (url, opts) => {
      const path = new URL(url, ORIGIN).pathname;
      calls[path] = (calls[path] || 0) + 1;
      return fetch(new URL(url, ORIGIN), opts);
    };
    win.Blob = Blob;
    win.URL.createObjectURL = () => 'blob:stub';
    win.URL.revokeObjectURL = () => {};
    win.performance = performance;
    win.HTMLElement.prototype.scrollIntoView = () => {};
  },
});
const { window } = dom;
const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const $ = (id) => window.document.getElementById(id);

await wait(2500);

const gen = $('generate'), dl = $('download'), status = $('genStatus');
console.log('generate button :', gen ? `"${$('generateLabel').textContent}"` : 'MISSING');
console.log('download hidden :', dl.hidden);
console.log('fields present  :', window.document.querySelectorAll('#fields tbody tr').length);

$('count').value = '5000';
$('count').dispatchEvent(new window.Event('input', { bubbles: true }));
await wait(300);

// Click twice in a row: the second must be ignored while the first is running.
gen.click();
const busyLabel = $('generateLabel').textContent;
const disabledNow = gen.disabled;
gen.click();
await wait(4000);

console.log('while running   : label', JSON.stringify(busyLabel), '| disabled', disabledNow);
console.log('after run       : label', JSON.stringify($('generateLabel').textContent), '| disabled', gen.disabled);
console.log('download shown  :', !dl.hidden);
console.log('status          :', JSON.stringify(status.textContent));

// Editing the schema must retract the download.
$('count').value = '10';
$('count').dispatchEvent(new window.Event('input', { bubbles: true }));
await wait(400);
console.log('after an edit   : download hidden again ->', dl.hidden, '| status', JSON.stringify(status.textContent));
// Download must save what Generate already produced. A second request would be
// wasted work and would make the reported size and timing a claim about
// different bytes than the ones written to disk.
$('count').value = '5000';
$('count').dispatchEvent(new window.Event('input', { bubbles: true }));
await wait(300);
gen.click();
await wait(4000);
const before = calls['/api/generate'];
dl.click();
await wait(500);
console.log('download re-ran generate ->', calls['/api/generate'] !== before,
            `(${before} then ${calls['/api/generate']})`);
console.log('requests        :', JSON.stringify(calls));
window.close();
