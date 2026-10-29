// Build a minimal iconify collection: only the icons this console actually uses.
// Importing @iconify/json/json/mdi.json wholesale ships ~3MB of unused SVGs.
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const names = readFileSync(resolve(here, 'icons.txt'), 'utf8')
  .split('\n')
  .map(s => s.trim())
  .filter(Boolean);

const full = JSON.parse(readFileSync(resolve(here, '../node_modules/@iconify/json/json/mdi.json'), 'utf8'));

const icons = {};
const missing = [];
for (const name of names) {
  if (full.icons[name]) icons[name] = full.icons[name];
  else if (full.aliases?.[name]) icons[name] = full.aliases[name];
  else missing.push(name);
}

writeFileSync(
  resolve(here, '../src/plugins/icons.json'),
  JSON.stringify({ prefix: 'mdi', width: full.width, height: full.height, icons }, null, 0)
);

console.log(`[icons] wrote ${Object.keys(icons).length} icons${missing.length ? `, missing: ${missing.join(',')}` : ''}`);
