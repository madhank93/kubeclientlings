// Parses every mermaid diagram in the generated chapters. A diagram that fails
// to parse renders as the words "Syntax error in text" on the site and nowhere
// else — nothing in the Astro build catches it.
//
// Run from web/: `node scripts/check-mermaid.mjs`. It lives here so Node's
// module resolution finds web/node_modules/mermaid.
//
// mermaid sanitizes label text through DOMPurify, which binds to whatever
// `window` exists when it is first imported. Under Node that is nothing, and
// every parse dies with "DOMPurify.addHook is not a function" — so a jsdom
// window is installed before mermaid is imported, not after.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { JSDOM } from 'jsdom';

const dom = new JSDOM('<!doctype html><html><body></body></html>');
globalThis.window = dom.window;
globalThis.document = dom.window.document;
globalThis.DOMParser = dom.window.DOMParser;
globalThis.Node = dom.window.Node;
globalThis.Element = dom.window.Element;
globalThis.HTMLElement = dom.window.HTMLElement;
globalThis.SVGElement = dom.window.SVGElement;
globalThis.NodeFilter = dom.window.NodeFilter;

const { default: mermaid } = await import('mermaid');

const dir = 'src/data/chapters';
const block = /<pre class="mermaid">([\s\S]*?)<\/pre>/g;
const unescape = (s) =>
  s.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');

let total = 0;
let bad = 0;

for (const file of readdirSync(dir).filter((f) => f.endsWith('.md'))) {
  const src = readFileSync(join(dir, file), 'utf8');
  for (const m of src.matchAll(block)) {
    total++;
    const line = src.slice(0, m.index).split('\n').length;
    try {
      await mermaid.parse(unescape(m[1]));
    } catch (e) {
      bad++;
      console.error(`✗ ${dir}/${file}:${line}\n  ${String(e.message).split('\n')[0]}`);
    }
  }
}

console.log(`${total - bad}/${total} diagrams parse`);
process.exit(bad === 0 ? 0 : 1);
