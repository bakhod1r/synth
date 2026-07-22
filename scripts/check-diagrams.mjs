// Renders every mermaid block in README.md and fails if any produces mermaid's
// error graphic. A broken diagram is not a build failure — it is a red box on
// the project's front page that nobody notices until someone else points at it.
//
//   npm install mermaid@11 jsdom@24
//   node scripts/check-diagrams.mjs
//
// Node and these two packages are a development convenience, not a dependency:
// nothing in the library or its tests needs them.
import fs from 'fs';
import { JSDOM } from 'jsdom';

const dom = new JSDOM('<!DOCTYPE html><body></body>', { pretendToBeVisual: true });
// Copy every DOM global mermaid's renderer reaches for. Missing one shows up as
// a render failure that looks like a broken diagram rather than a broken shim.
for (const k of Object.getOwnPropertyNames(dom.window)) {
  if (k in global) continue;
  try { global[k] = dom.window[k]; } catch {}
}
global.window = dom.window;
global.document = dom.window.document;

// jsdom has no SVG layout engine, so getBBox and getComputedTextLength are
// missing and every text-measuring diagram fails on the shim rather than on its
// own syntax. Approximating from the character count is enough: this checks
// that a diagram renders, not that it is laid out well.
const proto = dom.window.SVGElement.prototype;
proto.getBBox = function () {
  const chars = (this.textContent || '').length;
  return { x: 0, y: 0, width: chars * 8, height: 16 };
};
proto.getComputedTextLength = function () {
  return (this.textContent || '').length * 8;
};
proto.getScreenCTM = () => ({ a: 1, b: 0, c: 0, d: 1, e: 0, f: 0, inverse: () => ({}) });

const mermaid = (await import('mermaid')).default;
mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });

const readme = fs.readFileSync(process.argv[2] || 'README.md', 'utf8');
const blocks = [...readme.matchAll(/```mermaid\n([\s\S]*?)```/g)].map((m) => m[1]);
if (blocks.length === 0) {
  console.log('no mermaid blocks found — is the path right?');
  process.exit(1);
}
let bad = 0;
for (const [i, text] of blocks.entries()) {
  const kind = text.split('\n')[0].trim();
  try {
    const { svg } = await mermaid.render(`d${i}`, text);
    // Strip the stylesheet first: mermaid injects an .error-icon CSS rule into
    // every diagram, so searching the raw SVG for it flags healthy diagrams.
    const body = svg.replace(/<style[\s\S]*?<\/style>/g, '');
    const isError = /aria-roledescription="error"|class="error-icon"/.test(body);
    if (isError) { bad++; console.log(`  FAIL [${i}] ${kind}: renders as mermaid's error graphic`); }
    else console.log(`  ok   [${i}] ${kind} -> ${svg.length} bytes of SVG`);
  } catch (e) {
    bad++;
    console.log(`  FAIL [${i}] ${kind}: ${String(e.message || e).split('\n')[0]}`);
  }
}
console.log(bad ? `${bad} of ${blocks.length} would show as an error box` : `all ${blocks.length} diagrams render`);
process.exit(bad ? 1 : 0);
