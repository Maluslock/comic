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

  // WCAG 2.1 SC 1.4.3 明确豁免「未激活的用户界面组件」的对比度要求。
  // 禁用态在本项目用 class 里的 `disabled` 表达（如 .login-btn.disabled、.btn-submit.disabled）。
  // 这类元素**不算缺陷**，但也不静默丢弃 —— 单独进 `exempt` 数组以便人看。
  function isInactive(el) {
    let n = el;
    while (n && n.nodeType === 1) {
      const cls = typeof n.className === 'string' ? n.className : '';
      if (/(^|\s)disabled(\s|$)/.test(cls) || /(^|\s|-)is-disabled(\s|$)/.test(cls)) return true;
      if (n.getAttribute && n.getAttribute('aria-disabled') === 'true') return true;
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

  // --- Position-aware gradient sampling ---------------------------------------
  // A gradient's colour depends on WHERE the text sits on it. Scoring every stop
  // and taking the worst end reports a colour the text never touches: on a header
  // that fades purple→dark the stats row sits at t≈0.56, yet was judged against the
  // dark end (false positive). So sample the ELEMENT's own border box (4 corners +
  // centre), project each point onto the gradient's axis — whose length comes from
  // the box of the ancestor that actually carries the gradient — and interpolate.
  // A large element still spans the whole axis, so both ends stay in its samples.

  function splitTopLevel(s) {
    const out = [];
    let depth = 0, cur = '';
    for (const ch of s) {
      if (ch === '(') depth++;
      else if (ch === ')') depth--;
      if (ch === ',' && depth === 0) { out.push(cur); cur = ''; }
      else cur += ch;
    }
    out.push(cur);
    return out;
  }

  // Parses a SINGLE plain linear-gradient(). Anything else (multiple layers,
  // repeating-/radial-/conic-, px or calc() stops, unknown colour syntax) → null
  // and the caller keeps the old "all stops" fallback.
  function parseLinearGradient(img) {
    const s = String(img).trim();
    if (!/^linear-gradient\(/i.test(s) || !/\)$/.test(s)) return null;
    const inner = s.slice(s.indexOf('(') + 1, -1);
    const parts = splitTopLevel(inner).map((p) => p.trim()).filter(Boolean);
    if (parts.length < 2) return null;

    let angleDeg = 180, i = 0; // CSS default: to bottom
    const head = parts[0];
    if (/^-?[\d.]+deg$/i.test(head)) { angleDeg = parseFloat(head); i = 1; }
    else if (/^to\s+/i.test(head)) {
      const kw = head.toLowerCase().replace(/^to\s+/, '').split(/\s+/).sort().join(' ');
      const map = { top: 0, right: 90, bottom: 180, left: 270, 'right top': 45,
        'bottom right': 135, 'bottom left': 225, 'left top': 315 };
      if (map[kw] === undefined) return null;
      angleDeg = map[kw]; i = 1;
    } else if (!/^(rgba?\(|transparent)/i.test(head)) return null;

    const stops = [];
    for (; i < parts.length; i++) {
      const m = /^(rgba?\([^)]*\)|transparent)\s*(.*)$/i.exec(parts[i]);
      if (!m) return null;
      const color = /^transparent$/i.test(m[1]) ? { r: 0, g: 0, b: 0, a: 0 } : parseColor(m[1]);
      if (!color) return null;
      const rest = m[2].trim();
      let pos = null;
      if (rest) {
        const pm = /^(-?[\d.]+)%$/.exec(rest);
        if (!pm) return null; // px / calc() stops unsupported → fall back
        pos = parseFloat(pm[1]) / 100;
      }
      stops.push({ color, pos });
    }
    if (stops.length < 2) return null;
    return { stops, angleDeg };
  }

  // Stops without a percentage are distributed evenly per CSS, then clamped monotonic.
  function resolveStopPositions(stops) {
    const n = stops.length;
    const pos = stops.map((s) => s.pos);
    if (pos[0] === null) pos[0] = 0;
    if (pos[n - 1] === null) pos[n - 1] = 1;
    let i = 0;
    while (i < n) {
      if (pos[i] === null) {
        let j = i;
        while (j < n && pos[j] === null) j++;
        const a = pos[i - 1], b = pos[j], k = j - i + 1;
        for (let m = i; m < j; m++) pos[m] = a + ((b - a) * (m - i + 1)) / k;
        i = j;
      } else i++;
    }
    for (let m = 1; m < n; m++) if (pos[m] < pos[m - 1]) pos[m] = pos[m - 1];
    return pos;
  }

  // Colour at projection t, clamped to the end stops; premultiplied interpolation.
  function gradientColorAt(stops, pos, t) {
    const n = stops.length;
    if (t <= pos[0]) return stops[0].color;
    if (t >= pos[n - 1]) return stops[n - 1].color;
    for (let i = 0; i < n - 1; i++) {
      if (t >= pos[i] && t <= pos[i + 1]) {
        const span = pos[i + 1] - pos[i];
        const f = span <= 0 ? 0 : (t - pos[i]) / span;
        const A = stops[i].color, B = stops[i + 1].color;
        const a = A.a + (B.a - A.a) * f;
        if (a <= 0) return { r: 0, g: 0, b: 0, a: 0 };
        return {
          r: (A.r * A.a + (B.r * B.a - A.r * A.a) * f) / a,
          g: (A.g * A.a + (B.g * B.a - A.g * A.a) * f) / a,
          b: (A.b * A.a + (B.b * B.a - A.b * A.a) * f) / a,
          a: a
        };
      }
    }
    return stops[n - 1].color;
  }

  // Gradient colours at the element's 5 sample points, or null to fall back.
  function gradientSamples(img, hostRect, pts) {
    const g = parseLinearGradient(img);
    if (!g || hostRect.width < 1 || hostRect.height < 1) return null;
    const th = (g.angleDeg * Math.PI) / 180;
    const dx = Math.sin(th), dy = -Math.cos(th);
    const L = Math.abs(hostRect.width * Math.sin(th)) + Math.abs(hostRect.height * Math.cos(th));
    if (!(L > 0)) return null;
    const cx = hostRect.left + hostRect.width / 2;
    const cy = hostRect.top + hostRect.height / 2;
    const pos = resolveStopPositions(g.stops);
    return pts.map(([px, py]) => {
      const t = 0.5 + ((px - cx) * dx + (py - cy) * dy) / L;
      return gradientColorAt(g.stops, pos, t);
    });
  }

  // A gradient is an opaque paint that OCCLUDES the base beneath it, so when one is
  // present the plain base is NOT a valid background colour. The converse holds too:
  // an OPAQUE solid occludes anything beneath it, including an outer gradient.
  // Innermost paint wins. Only fall back to the composited solid base when nothing
  // covers the text.
  function effectiveBackgrounds(el) {
    const r = el.getBoundingClientRect();
    const pts = [
      [r.left, r.top], [r.right, r.top], [r.left, r.bottom], [r.right, r.bottom],
      [r.left + r.width / 2, r.top + r.height / 2]
    ];
    const chain = [];
    let n = el;
    while (n && n.nodeType === 1) { chain.push(n); n = n.parentElement; }
    chain.reverse(); // outermost → innermost

    let base = { r: 10, g: 10, b: 26 };
    let cands = null;
    for (const node of chain) {
      const cs = getComputedStyle(node);
      const bg = parseColor(cs.backgroundColor);
      if (bg && bg.a > 0) {
        base = over(bg, base);
        // Opaque solid: clear any gradient inherited from an outer layer, or
        // text on an opaque card inside a gradient region scores against stale stops.
        if (bg.a >= 1) cands = null;
      }
      const img = cs.backgroundImage;
      if (img && img !== 'none') {
        const sampled = gradientSamples(img, node.getBoundingClientRect(), pts);
        if (sampled) {
          cands = sampled.map((c) => over(c, base));
        } else {
          const stops = extractColors(img);
          if (stops.length) cands = stops.map((st) => over(st, base));
        }
      }
    }
    return cands || [base];
  }

  const results = [];
  const exempt = [];
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

    const record = {
      sel: el.tagName.toLowerCase() + (el.className && typeof el.className === 'string' ? '.' + el.className.trim().split(/\s+/).join('.') : ''),
      text: text.slice(0, 34),
      color: cs.color,
      bg: worstBg ? 'rgb(' + Math.round(worstBg.r) + ',' + Math.round(worstBg.g) + ',' + Math.round(worstBg.b) + ')' : '?',
      ratio: Math.round(worst * 100) / 100,
      need: need,
      px: Math.round(size * 10) / 10
    };
    // 未激活组件（禁用态）按 WCAG 1.4.3 豁免：不计入 total，但保留在 exempt 里可见，
    // 不静默丢弃 —— 豁免必须是可审计的。
    if (isInactive(el)) { exempt.push(record); continue; }
    results.push(record);
  }

  results.sort((a, b) => a.ratio - b.ratio);
  exempt.sort((a, b) => a.ratio - b.ratio);
  return JSON.stringify({
    url: location.hash || location.pathname,
    total: results.length,
    findings: results.slice(0, 24),
    exempt: exempt.length,
    exemptSample: exempt.slice(0, 6)
  });
})()
