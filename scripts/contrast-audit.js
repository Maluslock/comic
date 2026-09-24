(() => {
  const IGNORE_TAGS = /UNI-TABBAR|UNI-TAB-BAR|UNI-SWIPER-DOT|UNI-PAGE-HEAD|UNI-NAV-BAR|UNI-SWIPER-ITEM/;

  function ignored(el) {
    let n = el;
    while (n && n.nodeType === 1) {
      if (IGNORE_TAGS.test(n.tagName)) return true;
      n = n.parentElement;
    }
    return false;
  }

  function parseColor(c) {
    if (!c || c === 'transparent' || c === 'rgba(0, 0, 0, 0)') return null;
    const m = c.match(/rgba?\(([^)]+)\)/);
    if (!m) return null;
    const p = m[1].split(/[,\s/]+/).filter(Boolean).map(parseFloat);
    return { r: p[0], g: p[1], b: p[2], a: p.length > 3 ? p[3] : 1 };
  }

  function lin(c) { c /= 255; return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); }
  function lum(c) { return 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b); }
  function ratio(a, b) {
    const L1 = lum(a), L2 = lum(b);
    return (Math.max(L1, L2) + 0.05) / (Math.min(L1, L2) + 0.05);
  }
  function over(fg, bg) {
    const a = fg.a === undefined ? 1 : fg.a;
    return { r: fg.r * a + bg.r * (1 - a), g: fg.g * a + bg.g * (1 - a), b: fg.b * a + bg.b * (1 - a) };
  }

  // Cumulative opacity from the element up to the root. An ancestor with opacity < 1
  // blends the whole subtree toward the backdrop, so text contrast degrades even when
  // the element's OWN opacity is 1 — `getComputedStyle(el).opacity` is not inherited.
  function cumulativeOpacity(el) {
    let o = 1, n = el;
    while (n && n.nodeType === 1) {
      o *= parseFloat(getComputedStyle(n).opacity);
      if (o === 0) return 0;
      n = n.parentElement;
    }
    return o;
  }
  function extractColors(img) {
    const out = [];
    const re = /rgba?\(([^)]+)\)/g;
    let m;
    while ((m = re.exec(img))) {
      const p = m[1].split(/[,\s/]+/).filter(Boolean).map(parseFloat);
      if (p.length >= 3) out.push({ r: p[0], g: p[1], b: p[2], a: p.length > 3 ? p[3] : 1 });
    }
    return out;
  }

  // A gradient is an opaque paint that OCCLUDES the base beneath it, so when one is
  // present the plain base is NOT a valid background colour. The converse holds too:
  // an OPAQUE solid occludes anything beneath it, including an outer gradient.
  // Innermost paint wins. Only fall back to the composited solid base when nothing
  // covers the text.
  function effectiveBackgrounds(el) {
    const stack = [];
    let n = el;
    while (n && n.nodeType === 1) {
      const cs = getComputedStyle(n);
      stack.push({ bg: parseColor(cs.backgroundColor), img: cs.backgroundImage });
      n = n.parentElement;
    }
    let base = { r: 10, g: 10, b: 26 };
    let cands = null;
    for (let i = stack.length - 1; i >= 0; i--) {
      const s = stack[i];
      if (s.bg && s.bg.a > 0) {
        base = over(s.bg, base);
        // Opaque solid: clear any gradient stops inherited from an outer layer, or
        // text on an opaque card inside a gradient region scores against stale stops.
        if (s.bg.a >= 1) cands = null;
      }
      if (s.img && s.img !== 'none') {
        const stops = extractColors(s.img);
        if (stops.length) cands = stops.map((st) => over(st, base));
      }
    }
    return cands || [base];
  }

  const results = [];
  for (const el of document.querySelectorAll('*')) {
    if (ignored(el)) continue;
    let text = '';
    for (const node of el.childNodes) if (node.nodeType === 3) text += node.nodeValue;
    text = text.replace(/\s+/g, ' ').trim();
    if (!text) continue;

    const r = el.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) continue;

    const cs = getComputedStyle(el);
    if (cs.visibility === 'hidden' || cs.display === 'none') continue;
    const alpha = cumulativeOpacity(el);
    if (alpha <= 0.001) continue;
    if (parseColor(cs.webkitTextFillColor || cs.color) === null) continue; // gradient-clipped text

    const fg = parseColor(cs.color);
    if (!fg) continue;
    // Semi-transparent text composites toward its background, which always REDUCES
    // contrast. Ignoring fg alpha therefore OVERESTIMATES contrast and hides real
    // violations, so composite the effective foreground onto each candidate first.
    const fgEff = { r: fg.r, g: fg.g, b: fg.b, a: fg.a * alpha };

    let worst = Infinity, worstBg = null;
    for (const bg of effectiveBackgrounds(el)) {
      const c = ratio(over(fgEff, bg), bg);
      if (c < worst) { worst = c; worstBg = bg; }
    }

    const size = parseFloat(cs.fontSize);
    const weight = parseInt(cs.fontWeight, 10) || 400;
    const large = size >= 24 || (size >= 18.66 && weight >= 700);
    const need = large ? 3 : 4.5;
    if (worst >= need) continue;

    results.push({
      sel: el.tagName.toLowerCase() + (el.className && typeof el.className === 'string' ? '.' + el.className.trim().split(/\s+/).join('.') : ''),
      text: text.slice(0, 34),
      color: cs.color,
      bg: worstBg ? 'rgb(' + Math.round(worstBg.r) + ',' + Math.round(worstBg.g) + ',' + Math.round(worstBg.b) + ')' : '?',
      ratio: Math.round(worst * 100) / 100,
      need: need,
      px: Math.round(size * 10) / 10
    });
  }

  results.sort((a, b) => a.ratio - b.ratio);
  return JSON.stringify({ url: location.hash || location.pathname, total: results.length, findings: results.slice(0, 24) });
})()
