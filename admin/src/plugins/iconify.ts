import { addAPIProvider, addCollection } from '@iconify/vue';
import type { IconifyJSON } from '@iconify/types';

/** Setup the iconify offline */
export async function setupIconifyOffline() {
  const { VITE_ICONIFY_URL } = import.meta.env;

  if (VITE_ICONIFY_URL) {
    addAPIProvider('', { resources: [VITE_ICONIFY_URL] });
    return;
  }

  const mods = await Promise.all([
    import('@iconify/json/json/mdi.json'),
    import('@iconify/json/json/material-symbols.json'),
    import('@iconify/json/json/ant-design.json'),
    import('@iconify/json/json/line-md.json'),
    import('@iconify/json/json/majesticons.json'),
    import('@iconify/json/json/ph.json')
  ]);

  mods.forEach(mod => addCollection(mod.default as IconifyJSON));
}
