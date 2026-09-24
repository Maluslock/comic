# C 端霓虹配色对比度收敛 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 C 端全站实测不达 WCAG AA 的四族霓虹配色收敛到共享 SCSS mixin 并全站迁移，使 15 个页面 77 处对比度缺陷归零且不再复发。

**Architecture:** 在已全局注入的 `src/styles/variables.scss` 中新增两个 mixin（`on-neon-fill` 压霓虹实底用深字、`neon-pill` 紫底标签用亮紫字）与一个提亮红 token，然后把散落各页 scoped 样式中的缺陷声明替换为 `@include`。范围以**实测**为准，不以 grep 为准。

**Tech Stack:** Vue 3 `<script setup>` + SCSS（Vite `additionalData` 全局注入 variables.scss）+ uni-app H5/mp-weixin；验证工具为 `agent-browser` CLI + 注入式对比度脚本。

**Spec:** `docs/superpowers/specs/2026-09-24-cend-contrast-aa-design.md`

## Global Constraints

- 分支：`fix/cend-contrast-aa`（已建，stack 在 `feat/photographer-pricing` 之上）
- SCSS 只允许用 token/mixin，禁止新增硬编码色值（`$dark-bg-primary` 等已有 token 除外）
- 尺寸单位一律 `rpx`；**禁止用 `transform` / `zoom` 缩放交互元素**（那会同步收缩命中区，见 Task 7）
- mixin **只管颜色**：不得把 `padding` / `font-size` / `border-radius` / `border` 塞进 mixin
- 实测达标的**非紫**同色系（青 ~5.5 / 绿 ~5.8 / 金 ~5.9）**一律不动**（含 `activate.vue` 的 `.btn-works`、`.btn-services`、`.is-cert`，以及 `services.vue` 的 `.btn-tpl` / `.price-tag.is-fixed` / `.is-mutual` / `.is-active`）
- **紫**同色系一律迁移以保持族内一致：`.tag-item` 等实测 3.80 确属不达标；`activate.vue .btn-cert` 压深页底实测为**边际达标 4.54**，落在测量噪声边缘，一并迁移到 5.7 —— 这是**刻意的族内统一**，不是"顺手改达标项"。Task 4 Step 7 必须核对除 `.btn-cert` 外没有别的边际项被一并改动
- 每个 Task 结束必须 `git commit`
- 门禁命令（Task 7 全跑；改动后至少跑 `npx vue-tsc --noEmit`）：
  - `cd server && export GOROOT=/home/user/go-sdk/go1.22 && export PATH=$GOROOT/bin:$PATH && go build ./... && go vet ./... && go test ./...`
  - `npx vue-tsc --noEmit`
  - `bash server/scripts/smoke.sh`（dev，期望 EXIT=0）

## 环境前提（Task 1 前必须就绪）

| 依赖 | 启动方式 |
|---|---|
| dev API `:8088` | 二进制**不能**放 `/tmp`（每次 bash 调用是独立 tmpfs）。编译到 `server/build/comic-api`（已 gitignore），再以托管后台任务运行：`cd server && go build -o build/comic-api ./cmd/api`，然后 `./build/comic-api`（CWD 必须是 `server/`，它读 `.env`） |
| H5 dev `:5173` | 已在跑；若停，`npm run dev:h5` |
| `agent-browser` | 不在默认 PATH，需 `export PATH=/home/user/.nvm/versions/node/v22.16.0/bin:$PATH`（v0.27.0） |
| 测试账号 | coser `13800138000`、摄影师 `10000000001`，验证码任意 `1234` 前缀（如 `123456`） |

## 文件结构

| 文件 | 职责 |
|---|---|
| `scripts/contrast-audit.js` | **新建**。注入浏览器的对比度审计 payload（IIFE，返回 JSON）。单一职责：给定当前页面 DOM，算出所有不达标文本节点 |
| `scripts/contrast-audit.sh` | **新建**。运行器：登录取 token → 按角色遍历页面 → `open`+`reload` → 注入 payload → 汇总输出 |
| `src/styles/variables.scss` | **修改**。新增 `$error-bright` 与 `on-neon-fill` / `neon-pill` 两个 mixin（唯一事实源） |
| `src/pages/**/*.vue` | **修改**。把 31 处 scoped 规则中的缺陷声明替换为 `@include` |

---

### Task 1: 审计工具入库并可重复运行

把一次性脚本提升为入库 QA 工具，使后续每个 Task 都有客观、可重复的判据。

**Files:**
- Create: `scripts/contrast-audit.js`
- Create: `scripts/contrast-audit.sh`

**Interfaces:**
- Consumes: 运行中的 H5 dev（`BASE_URL`）与 API（`API_URL`）、已登录测试账号
- Produces: `<OUT_DIR>/<page>.json`（审计结果）、`<OUT_DIR>/<page>.meta`（元素数与文本长度，用于区分真通过与空态）；stdout 汇总表 `page total worst textLen`

- [ ] **Step 1: 创建 `scripts/contrast-audit.js`**

内容为注入浏览器的 IIFE，返回 JSON 字符串。关键实现（含**渐变遮挡**修正——存在渐变时其下的兜底底色不是有效背景色）：

```js
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
```

- [ ] **Step 2: 创建 `scripts/contrast-audit.sh`**

```bash
#!/usr/bin/env bash
# C 端对比度审计（可重复）。用法：bash scripts/contrast-audit.sh [OUT_DIR]
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:5173}"
API_URL="${API_URL:-http://127.0.0.1:8088}"
OUT_DIR="${1:-.audit/out}"
AB="${AGENT_BROWSER:-agent-browser}"

command -v "$AB" >/dev/null 2>&1 || export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PAYLOAD="$(cat "$SCRIPT_DIR/contrast-audit.js")"
mkdir -p "$OUT_DIR"

set_role() {  # $1=phone
  local resp token uid pid
  resp="$(curl -s -X POST "$API_URL/api/v1/login" -H 'Content-Type: application/json' \
    -d "{\"phone\":\"$1\",\"code\":\"123456\"}" --max-time 8)"
  token="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])' 2>/dev/null || true)"
  # 登录失败必须硬停：否则 token 为空、页面全部按**未登录态**测量，
  # 会产出一张看起来合理但完全错误的表（正是本项目两次栽过的假通过）。
  [ -n "$token" ] || {
    echo "FATAL: 账号 $1 登录失败（API 挂了或账号不对）—— 拒绝在未登录态下测量" >&2
    exit 1
  }
  uid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"]["id"])')"
  pid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"].get("photographerId") or 0)')"
  # open 建立 origin，再用 reload 触发整页重载，否则 Pinia store 读不到注入的 token
  "$AB" open "$BASE_URL/#/pages/index/index" >/dev/null 2>&1
  "$AB" eval "(function(){uni.setStorageSync('token','$token');uni.setStorageSync('user',JSON.stringify({id:$uid,name:'audit',phone:'$1',avatar:'',bio:'',photographerId:$pid}));return 1;})()" >/dev/null 2>&1
}

FAILED=0

audit_page() {  # $1=route
  local n; n="$(printf '%s' "$1" | tr '/?=' '___')"
  "$AB" open "$BASE_URL/#/$1" >/dev/null 2>&1
  "$AB" reload >/dev/null 2>&1
  sleep 3.2
  "$AB" eval "JSON.stringify({n:document.querySelectorAll('*').length,len:document.body.innerText.replace(/\s+/g,' ').trim().length})" > "$OUT_DIR/$n.meta" 2>&1
  "$AB" eval "$PAYLOAD" > "$OUT_DIR/$n.json" 2>&1
  printf '%-34s ' "$1"
  if ! python3 - "$OUT_DIR/$n.json" "$OUT_DIR/$n.meta" <<'PY'
import json, sys
ok = True
try:
    d = json.loads(json.loads(open(sys.argv[1]).read().strip()))
    print('total=%-3d exempt=%-2d worst=%-6s' % (
        d['total'], d.get('exempt', 0),
        d['findings'][0]['ratio'] if d['findings'] else '-'), end=' ')
except Exception:
    print('PARSE-FAIL', end=' '); ok = False
try:
    print('textLen=%d' % json.loads(json.loads(open(sys.argv[2]).read().strip()))['len'])
except Exception:
    print('(no meta)'); ok = False
sys.exit(0 if ok else 1)
PY
  then
    FAILED=$((FAILED + 1))
  fi
}

COSER_PAGES="pages/index/index pages/event/detail?id=1 pages/event/list pages/photographer/list \
pages/photographer/detail?id=1 pages/search/search pages/order/list pages/order/detail?id=1 \
pages/order/detail?id=2 pages/message/index pages/profile/index pages/favorite/list pages/follow/list \
pages/comment/index pages/booking/index pages/portfolio/index pages/calendar/index pages/settings/index \
pages/login/index"

PHOTOGRAPHER_PAGES="pages/photographer/services pages/photographer/works pages/photographer/orders \
pages/photographer/activate pages/photographer/profile-edit pages/photographer/cert-apply"

echo "== role=coser =="
set_role 13800138000
for p in $COSER_PAGES; do audit_page "$p"; done

echo "== role=photographer =="
set_role 10000000001
for p in $PHOTOGRAPHER_PAGES; do audit_page "$p"; done

# 退出码必须是有意义的通过/失败信号：任何页面测量失败 → 本次运行无效。
if [ "$FAILED" -ne 0 ]; then
  echo "FATAL: $FAILED 页测量失败 —— 本次运行不是有效结论，勿据此判定通过" >&2
  exit 1
fi
echo "OK: 全部页面测量成功"
```

- [ ] **Step 3: 让脚本可执行并验证能复现已知基线**

```bash
chmod +x scripts/contrast-audit.sh
bash scripts/contrast-audit.sh .audit/baseline
```

Expected（数字为**累积透明度修正后**的真实绘制值，允许 ±0.05）：

- `pages/login/index` → `worst=1.31` —— 该按钮的祖先 `uni-view.login-btn.disabled` 带 `opacity: 0.45`（未输入手机号时的 idle 态）。**该值状态相关**：按钮可用态为 1.97。门控取 idle 态的 **1.31**
- `pages/index/index` → 具名发现 `'近期热门漫展'`（白字压渐变青端）**仍精确为 2.43**；但 summary 的 `worst` 现为 **1.84**（装饰性 `◆`，祖先 `uni-text.cta-marker` opacity 0.7）
- `pages/photographer/services` → `total=0`（batch 1 已修正确，**不应**出现 1.00 的假阳性）

> **重要**：修正累积透明度后，一批**半透明 / 降透明度**元素的数字会变大（更差）—— 那是**修正本身**，不是回归。首轮基线里的 1.97 / 2.43 / 3.96 等值中，凡涉及 `rgba(255,255,255,0.8/0.9)` 或祖先 `opacity<1` 的，都是**忽略透明度导致的高估值**（例如登录页 1.97 实为 1.31）。Task 2–7 必须以修正后的数字为准。

- [ ] **Step 4: Commit**

```bash
git add scripts/contrast-audit.js scripts/contrast-audit.sh
git commit -m "test(qa): 对比度审计工具入库（含渐变遮挡修正 + 整页重载纪律）"
```

---

### Task 2: WCAG 未激活豁免分区 + mixin 层 + 登录页修复

> **本任务含一处由 controller 裁决带入的工具改动**（ledger `Ruling (T2-1)`）：Task 1 的门控把**禁用态**元素当作缺陷，而 WCAG 2.1 SC 1.4.3 明确豁免「未激活的用户界面组件」的对比度要求。实测证据：登录页主按钮在未输入手机号时带 `.disabled`（`opacity: 0.45`），**改深字后仍只有 2.20:1** —— 即**只改字色修不好登录页**。故先给门控加 `exempt` 分区，再修字色。

Mixin 与首个使用点放在同一 Task，使这一步有可观测的验收。

**Files:**
- Modify: `scripts/contrast-audit.js`（加 `isInactive()` + `exempt` 分区）
- Modify: `scripts/contrast-audit.sh`（summary 打印 `exempt=`）
- Modify: `src/styles/variables.scss`
- Modify: `src/pages/login/index.vue:325`

**Interfaces:**
- Produces: `@mixin on-neon-fill`（无参，输出 `color: $dark-bg-primary`）、`@mixin neon-pill`（无参，输出 `color: $neon-purple-bright; background: $neon-purple-dim`）、`$error-bright: #f87171` —— Task 3/4/5 全部依赖这些名字
- Produces: 审计 payload 新增返回字段 `exempt`（数量）与 `exemptSample`（前 6 条）；**`total` 从此只计可执行缺陷**，Task 3–7 的通过判据均基于它

- [ ] **Step 1: 给门控加 WCAG 未激活豁免分区**

`scripts/contrast-audit.js`：加入 `isInactive(el)`（自身或祖先 class 命中 `disabled`/`is-disabled`，或 `aria-disabled="true"`），在 `results.push` 之前分流：

```js
    // 未激活组件（禁用态）按 WCAG 1.4.3 豁免：不计入 total，但保留在 exempt 里可见，
    // 不静默丢弃 —— 豁免必须是可审计的。
    if (isInactive(el)) { exempt.push(record); continue; }
    results.push(record);
```

并让返回体带上 `exempt` 与 `exemptSample`。`scripts/contrast-audit.sh` 的 summary 加印 `exempt=%-2d`。

（完整代码见 brief，逐字采用。）

- [ ] **Step 2: 验证豁免分区生效，并记录它影响了哪些页**

Run: `bash scripts/contrast-audit.sh .audit/task2-tool`

Expected：
- `pages/login/index` → `total=0 exempt>=1`（禁用态登录按钮进入 exempt）
- 其他含禁用态的页面（如 `pages/photographer/cert-apply`，其 `.btn-submit.disabled`）**也会**把这些元素移入 exempt → 其 `total` 下降属**预期**，不是回归
- **必须逐页记录哪些页的 `total` 因豁免而下降**，确保没有可执行缺陷被误判为豁免

- [ ] **Step 3: 在 `variables.scss` 加入提亮红 token**

在 `$error-color: #ef4444;` 之后新增一行（按 token 名定位，不按行号）：

```scss
$error-bright: #f87171;   // 压 rgba(239,68,68,.1) 底 → 5.86:1（$error-color 实测仅 4.32:1）
```

- [ ] **Step 4: 在 `variables.scss` 末尾加入两个 mixin**

```scss
// === 对比度 mixin（唯一事实源）===
// 压霓虹实底的文字：紫端 4.95:1 / 青端 8.07:1 / 粉端 5.56:1
@mixin on-neon-fill {
  color: $dark-bg-primary;
}

// 同色系低对比 pill：亮紫字压紫底 → 实测 ≥5.7:1（原 3.8:1）
@mixin neon-pill {
  color: $neon-purple-bright;
  background: $neon-purple-dim;
}
```

- [ ] **Step 5: 迁移登录页按钮文字**

`src/pages/login/index.vue` 的 `.login-btn-text` 规则，把：

```scss
  color: $dark-text-primary;
```

改为：

```scss
  @include on-neon-fill;
```

- [ ] **Step 6: 编译验证**

Run: `npx vue-tsc --noEmit`
Expected: 退出码 0，无输出错误

- [ ] **Step 7: 实测验证登录页可执行缺陷归零**

Run: `bash scripts/contrast-audit.sh .audit/task2`

Expected: `pages/login/index` 行 `total=0`，且 `exempt>=1`（禁用态按钮被豁免）。修复前该页为 `total=1 worst=1.31`（禁用态）。

**另外必须单独验证启用态是真的达标**（豁免不能成为掩盖手段）：在页面里去掉 `.disabled` 后重测，启用态应为 `total=0` 且该元素**不**出现在 findings 里。

- [ ] **Step 8: Commit**

```bash
git add scripts/contrast-audit.js scripts/contrast-audit.sh src/styles/variables.scss src/pages/login/index.vue
git commit -m "fix(qa,cend): WCAG 未激活豁免分区 + on-neon-fill/neon-pill mixin + 登录页按钮改深字"
```

---

### Task 3: F1 霓虹底浅字全站迁移

把 19 处 scoped 规则中的浅字声明换成 `@include on-neon-fill`。

**Files:**
- Modify（每处只改一行，见下）：
  `src/pages/booking/index.vue:367`、`src/pages/chat/index.vue:415`、`src/pages/comment/index.vue:380`、
  `src/pages/event/detail.vue:659`、`src/pages/favorite/list.vue:95`、`src/pages/follow/list.vue:112`、
  `src/pages/index/index.vue:400`、`src/pages/index/index.vue:476`、`src/pages/index/index.vue:754`、
  `src/pages/message/index.vue:355`、`src/pages/order/detail.vue:607`、
  `src/pages/photographer/activate.vue:186`、`src/pages/photographer/activate.vue:206`、
  `src/pages/photographer/cert-apply.vue:235`、`src/pages/photographer/cert-apply.vue:259`、
  `src/pages/photographer/profile-edit.vue:258`、`src/pages/photographer/works.vue:274`、
  `src/pages/photographer/works.vue:321`、`src/pages/profile/edit.vue:218`

**Interfaces:**
- Consumes: Task 2 的 `@mixin on-neon-fill`

- [ ] **Step 1: 迁移 `.btn-primary` / `.send-btn` / `.btn-retry` / `.btn-go` / `.btn-submit` / `.btn-add` / `.btn-reapply` / `.cta-primary`（`$neon-gradient` 底）**

以下 17 处，把各自的 `color: #fff;` 一行替换为 `@include on-neon-fill;`：

| 文件 | 选择器 |
|---|---|
| `src/pages/booking/index.vue` | `.btn-primary` |
| `src/pages/chat/index.vue` | `.send-btn` |
| `src/pages/comment/index.vue` | `.btn-primary` |
| `src/pages/event/detail.vue` | `.btn-retry` |
| `src/pages/favorite/list.vue` | `.btn-go` |
| `src/pages/follow/list.vue` | `.btn-go` |
| `src/pages/index/index.vue` | `.cta-primary` |
| `src/pages/index/index.vue` | `.btn-retry` |
| `src/pages/order/detail.vue` | `.btn-primary` |
| `src/pages/photographer/activate.vue` | `.btn-go` |
| `src/pages/photographer/activate.vue` | `.btn-submit` |
| `src/pages/photographer/cert-apply.vue` | `.btn-reapply` |
| `src/pages/photographer/cert-apply.vue` | `.btn-submit` |
| `src/pages/photographer/profile-edit.vue` | `.btn-submit` |
| `src/pages/photographer/works.vue` | `.btn-add` |
| `src/pages/photographer/works.vue` | `.btn-submit` |
| `src/pages/profile/edit.vue` | `.btn-submit` |

替换的原文与结果（以 `booking/index.vue` 为例，其余同构）：

```scss
// 前
.btn-primary { ... background: $neon-gradient; color: #fff; border-radius: $border-radius-lg; }
// 后
.btn-primary { ... background: $neon-gradient; @include on-neon-fill; border-radius: $border-radius-lg; }
```

- [ ] **Step 2: 迁移两个粉底角标**

`src/pages/index/index.vue:400` 的 `.badge`：

```scss
// 前
  color: #fff;
  background: $neon-pink;
// 后
  @include on-neon-fill;
  background: $neon-pink;
```

`src/pages/message/index.vue:355` 的 `.unread-count`：

```scss
// 前
  background: $neon-pink;
  color: #fff;
// 后
  background: $neon-pink;
  @include on-neon-fill;
```

- [ ] **Step 2b: `order/detail` —— 按状态变体分别取色（见 ledger `Ruling (T3-3)`）**

**关键事实**：`.status-card` 在不同状态下底色不同 —— active/pending/confirmed/completed 是亮底（青 / `$neon-gradient`），而 **`.cancelled` 是 `$dark-bg-secondary`（rgb(18,18,42)）**。深字压后者只有 **1.07:1**，白字压后者是 **18.3:1 ✅**。所以**不能**对这三个子元素无差别改深字。

用**变体作用域选择器**（只改选择器与颜色，不碰几何）：

```scss
// .status-card:not(.cancelled) 的三个子元素改用深字
.status-card:not(.cancelled) {
  .status-icon { @include on-neon-fill; }   // 原 color: rgba(255,255,255,0.9)
  .status-text { @include on-neon-fill; }   // 原 color: #fff
  .order-id    { @include on-neon-fill; }   // 原 color: rgba(255,255,255,0.8)
}
```

`.cancelled` 保持原有浅字不动。具体写法以能命中 `src/pages/order/detail.vue` 现有 DOM 为准（先确认 `.status-card` 的变体类名实际是怎么挂的），**不得凭猜**。

- [ ] **Step 2c: 补上门控漏测的状态变体（见 ledger `Ruling (T3-3)`）**

25 页清单只跑了 `order/detail?id=1`(pending) 与 `?id=2`(confirmed)，所以**它看不见 `.cancelled` 被改坏**。给 `scripts/contrast-audit.sh` 的 `COSER_PAGES` 增补两个真实存在的状态页：

```
pages/order/detail?id=3      # completed
pages/order/detail?id=4      # cancelled
```

（这两个 id 属 coser `13800138000`，已核实存在。）

- [ ] **Step 2d: 让门控对渐变按「元素实际位置」取样（见 ledger `Ruling (T3-4)`）**

**背景**：`.header` 是 `linear-gradient(135deg, #a855f7 40%, #12122a 100%)`。审计目前把**所有渐变停靠点**当候选并取最差（紫端 → 3.09/3.96），但统计行实际投影在 **t≈0.56**，真实背景 ≈ `rgb(128,67,192)`，**白字在其上是 6.02:1，本来就达标**。这是**假阳性**，且整段渐变上不存在能同时满足两端的单一文字色（全灰暴力搜最大 3.96）—— 所以按最差停靠点判定会导致"无解"，进而逼迫改品牌渐变。**不这么修。**

改 `scripts/contrast-audit.js` 的 `effectiveBackgrounds`：遇到渐变时，不再取全部停靠点，而是**取元素自身包围盒的 5 个取样点**（四角 + 中心）投影到该渐变的渐变轴上，插值求出各点颜色，作为候选集。

要点：
- 需要解析 `linear-gradient(<angle>deg, <color> [pct]%, <color> [pct]%)`，并按 CSS 规则由渐变角度与**承载渐变的那个祖先盒子**的尺寸算出渐变线长度；
- 未标注百分比的停靠点按 CSS 规则均匀分布；
- 投影点落在首/末停靠点之外时钳制到端点色；
- **取样基准 = 被测元素自身的包围盒**，不是承载渐变的祖先盒。门控测的是承载文本的那个节点（uni-app 常把文本渲染成内层 `SPAN`），其真实背景就是该节点所在处的渐变颜色 —— 这正是我们要的模型。
- **不得**为了让"某个跨整段渐变的按钮，其内层文字必须失败"而弯折取样器（ledger `Ruling (T3-7)`：该预期基于"被测元素 = 按钮盒"的错误假设，已作废）。
- 本步的验收不是"某个特定元素必须失败"，而是：① 已知真缺陷仍被捕捉（profile 的 2 处、`order/detail` 亮底状态）；② **无任何页面 `total` 上升**。

- [ ] **Step 2e: `profile/index` —— 统计行给不透明底 + 角色标签改深字（见 ledger `Ruling (T3-6)`）**

Step 2d 生效后重测。实现者的**像素采样**证明这里有 **2 处真缺陷**（不是取样假象）：

| 元素 | 实压背景 | 白字 | 深字 |
|---|---|---|---|
| 最左 `.stat-label`「收藏」 | `rgb(176,102,247)` | 3.45 | 5.68 |
| `.role-tag.coser` | `rgb(185,119,248)` | 2.98 | 6.59 |
| 最右 `.stat-label` | `rgb(88,62,127)` | 8.71 | 2.25 |

同一条 `.stat-label` 规则**横跨亮紫与近黑** → 全灰暴力搜最优 worst-of-two 仅 **3.45 < 4.5** → **单一字色无解**。

处置：
1. 给 `.stats-row` 一个**完全不透明**的底色（`a=1`；半透明无效 —— 取样器仍会看到外层渐变），让三个统计项共享同一底面 → 标签保持浅字即达标。可按需加 `border-radius` / `padding`，使其呈现为一张统计卡。
2. `.role-tag.coser` → `@include on-neon-fill`（深字；实压亮紫约 6.59 ✅）。

**仍然不得修改 `.header` 的渐变色**（T3-5）。改完留 `.audit/shots/` 前后截图以便评估观感。

- [ ] **Step 3: 编译验证**

Run: `npx vue-tsc --noEmit`
Expected: 退出码 0

- [ ] **Step 4: 实测验证 F1 完全归零**

Run: `bash scripts/contrast-audit.sh .audit/task3`

Expected：以下签名**全部不再出现**（括号内为修复前实测值）：

- `rgb(255,255,255) → rgb(6,182,212)`（2.43）
- `rgba(255,255,255,0.8) → rgb(6,182,212)`（2.02）
- `rgba(255,255,255,0.9) → rgb(6,182,212)`（2.21）
- `rgb(255,255,255) → rgb(168,85,247)`（3.96）
- `rgba(255,255,255,0.8) → rgb(168,85,247)`（3.09）
- `rgb(255,255,255) → rgb(236,72,153)`（3.53）
- `rgb(10,10,26) → rgb(168,85,247)`（3.44，`.cta-marker`）

**剩余发现必须只属于 F2（紫底紫字）、F3（红底红字）、F4（三级文本）三族** —— 即留给 Task 4/5/6 的那批。出现任何其他 light-on-neon 或深字压霓虹的残留，都说明清单再次遗漏。

**本轮新增覆盖**（Step 2c 加的 `?id=3` / `?id=4`）也必须一并达标：
- `pages/order/detail?id=4`（cancelled，深底 + 浅字）→ `total=0`
- `pages/order/detail?id=3`（completed，亮底 + 深字）→ `total=0`
- `pages/profile/index` → `total=0`（Step 2e 后）

且**不得出现任何新的** `rgb(10,10,26) → rgb(10,10,26)` 型 1.00:1 报告（渐变遮挡假阳性，Task 1 已修，出现即为回归），**也不得出现 `rgb(10,10,26) → rgb(18,18,42)` 型 1.07:1**（深字压深底 —— 即 `cancelled` 变体被误改的信号）。

- [ ] **Step 5: Commit**

```bash
git add scripts/contrast-audit.js scripts/contrast-audit.sh src/pages
git commit -m "fix(qa,cend): F1 收口 —— 变体作用域取色 + 渐变按位置取样 + 补测 cancelled/completed 状态"
```

---

### Task 4: F2 紫底 pill 迁移

**Files:**
- Modify: `src/pages/event/detail.vue:459`、`src/pages/index/index.vue:711`、
  `src/pages/message/index.vue:241`、`src/pages/photographer/activate.vue:190`、
  `src/pages/photographer/activate.vue:203`、`src/pages/photographer/detail.vue:422`、
  `src/pages/photographer/profile-edit.vue:255`、`src/pages/search/search.vue:242`

**Interfaces:**
- Consumes: Task 2 的 `@mixin neon-pill`

- [ ] **Step 1: 迁移两个标签 pill**

`src/pages/event/detail.vue:459` 的 `.tag-item`：

```scss
// 前
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 1rpx solid rgba($neon-purple, 0.3);
// 后
  @include neon-pill;
  border: 1rpx solid rgba($neon-purple, 0.3);
```

`src/pages/index/index.vue:711` 的 `.tag-item`：

```scss
// 前
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 2rpx solid rgba($neon-purple, 0.4);
// 后
  @include neon-pill;
  border: 2rpx solid rgba($neon-purple, 0.4);
```

- [ ] **Step 2: 迁移摄影师详情标签**

`src/pages/photographer/detail.vue:422` 的 `.tag`：

```scss
// 前
  background: $neon-purple-dim;
  color: $neon-purple;
  border-radius: $border-radius-sm;
// 后
  @include neon-pill;
  border-radius: $border-radius-sm;
```

- [ ] **Step 3: 迁移激活页认证按钮**

`src/pages/photographer/activate.vue:190` 的 `.btn-cert`：

```scss
// 前
  background: $neon-purple-dim;
  border: 2rpx solid $neon-purple;
  color: $neon-purple;
// 后
  @include neon-pill;
  border: 2rpx solid $neon-purple;
```

- [ ] **Step 4: 迁移三处交互态子块**

`src/pages/message/index.vue:241` 的 `.notification-icon { &.success { ... } }`，把子块内的两行：

```scss
  &.success {
    @include neon-pill;
  }
```

`src/pages/photographer/activate.vue:203` 的 `.mode-item { &.active { ... } }`，把子块内的 `color: $neon-purple;` 与 `background: $neon-purple-dim;` 两行替换为 `@include neon-pill;`（保留 `border-color`、`box-shadow`、`font-weight`）。

`src/pages/photographer/profile-edit.vue:255` 的 `.mode-item { &.active { ... } }`，同上处理。

`src/pages/search/search.vue:242` 的 `.hot-item { &:active { ... } }`，把子块内的 `background: $neon-purple-dim;` 与 `color: $neon-purple;` 替换为 `@include neon-pill;`。

- [ ] **Step 5: 编译验证**

Run: `npx vue-tsc --noEmit`
Expected: 退出码 0

- [ ] **Step 6: 实测验证 F2 归零**

Run: `bash scripts/contrast-audit.sh .audit/task4`

Expected：以下签名**不再出现**：

- `rgb(168, 85, 247) → rgb(44,32,69)`（原 3.80，36 处）
- `rgb(168, 85, 247) → rgb(41,28,73)`（`▼`，原 3.92）
- `rgb(168, 85, 247) → rgb(37,29,58)`（原 4.05）

且 `pages/index/index`、`pages/event/detail?id=1`、`pages/photographer/list`、`pages/photographer/detail?id=1` 四页 `total=0`。

- [ ] **Step 7: 确认未改动的青/绿/金同类项仍达标（spec §4「需实测确认」项）**

这些块**刻意不迁移**，必须证明它们本来就不需要修。在 `pages/photographer/activate` 上量测：

```bash
export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"
agent-browser open "http://localhost:5173/#/pages/photographer/activate" >/dev/null 2>&1
agent-browser reload >/dev/null 2>&1; sleep 3.5
agent-browser eval "(function(){const out={};for(const c of ['btn-works','btn-services','btn-cert']){const e=document.querySelector('.'+c);if(e)out[c]=getComputedStyle(e).color;}return JSON.stringify(out);})()"
bash scripts/contrast-audit.sh .audit/task4
```

Expected：

- 这些按钮的 `color` 仍为 `$neon-cyan` / `$neon-pink` / `$neon-purple(-bright)` 对应的 rgb，**未被误改**
- `pages/photographer/activate` 行 `total=0`
- 若任一页数值**上升**，说明误伤了达标项 → 回退该块并重跑

- [ ] **Step 8: Commit**

```bash
git add src/pages
git commit -m "fix(cend): F2 紫底标签对比度 3.80→5.7:1（8 块，含 ▼ 与交互态）"
```

---

### Task 5: F3 红底红字

**Files:**
- Modify: `src/pages/photographer/works.vue:301`（`.btn-del`）、`src/pages/photographer/works.vue:316`（`.img-del`）、`src/pages/photographer/services.vue:404`（`.btn-del`）

**Interfaces:**
- Consumes: Task 2 的 `$error-bright`

- [ ] **Step 1: 迁移 `works.vue` 的 `.btn-del`**

```scss
// 前
  color: $error-color;
  background: rgba(239, 68, 68, 0.1);
// 后
  color: $error-bright;
  background: rgba(239, 68, 68, 0.1);
```

- [ ] **Step 2: 迁移 `works.vue` 的 `.img-del`**

```scss
// 前
  color: $error-color;
  background: rgba(239, 68, 68, 0.1);
// 后
  color: $error-bright;
  background: rgba(239, 68, 68, 0.1);
```

- [ ] **Step 3: 迁移 `services.vue` 的 `.btn-del`**

同 Step 1 的替换。

- [ ] **Step 4: 编译验证**

Run: `npx vue-tsc --noEmit`
Expected: 退出码 0

- [ ] **Step 5: 实测验证 F3 归零**

Run: `bash scripts/contrast-audit.sh .audit/task5`
Expected: `rgb(239, 68, 68) → rgb(44,27,41)`（原 4.32）不再出现；`pages/photographer/works` 变为 `total=0`

- [ ] **Step 6: Commit**

```bash
git add src/pages/photographer/works.vue src/pages/photographer/services.vue
git commit -m "fix(cend): F3 删除按钮红字对比度 4.32→5.86:1（新增 \$error-bright）"
```

---

### Task 6: F4 三级文本 token 微调

**Files:**
- Modify: `src/styles/variables.scss:57`

**Interfaces:**
- Produces: 更新后的 `$dark-text-tertiary`（影响 31 个引用文件，均为同一微调）

- [ ] **Step 1: 微调 token**

```scss
// 前
$dark-text-tertiary: #7b8aa3;
// 后
$dark-text-tertiary: #7e8da6;
```

（卡片底实测 4.49 → 4.67:1。仅为舍入边界修正，视觉几乎不可见。）

- [ ] **Step 2: 编译验证**

Run: `npx vue-tsc --noEmit`
Expected: 退出码 0

- [ ] **Step 3: 实测验证 F4 归零**

Run: `bash scripts/contrast-audit.sh .audit/task6`
Expected: `rgb(123, 138, 163) → rgb(34,34,48)`（原 4.49，10 处）不再出现

- [ ] **Step 4: Commit**

```bash
git add src/styles/variables.scss
git commit -m "fix(cend): F4 三级文本对比度 4.49→4.67:1（\$dark-text-tertiary #7b8aa3→#7e8da6）"
```

---

### Task 7: switch 触控高度 + 全量复测 + 门禁

**Files:**
- Modify: `src/pages/settings/index.vue:351-354`（`.menu-switch`）

> **根因已实测确认**：`.menu-switch` 的原生 `offsetHeight` 本来就是 **45px**，是 `transform: scale(0.8)` 把视觉与**命中区**压在 **36px**。因此审计文档建议的 `min-height: 44px` **无效**（元素本就有 45px 高，min-height 不会解除缩放的命中区收缩）。正确做法是去掉缩放。

- [ ] **Step 1: 去掉缩放，恢复原生 45px 命中区**

`src/pages/settings/index.vue` 中：

```scss
// 前
.menu-switch {
  transform: scale(0.8);
  transform-origin: right center;
}
// 后
.menu-switch {
  /* 原生高 45px ≥ 44px 触控下限；此前的 scale(0.8) 会把命中区压到 36px */
}
```

（若去掉缩放后该行视觉过挤，只允许调整**外层行**的 `padding`/`gap` 来腾空间，**不得**用 `transform`/`zoom` 缩放开关注区。）

- [ ] **Step 2: 实测验证触控高**

```bash
export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"
agent-browser open "http://localhost:5173/#/pages/settings/index" >/dev/null 2>&1
agent-browser reload >/dev/null 2>&1; sleep 3
agent-browser eval "JSON.stringify([...document.querySelectorAll('uni-switch')].map(e=>({h:Math.round(e.getBoundingClientRect().height),native:e.offsetHeight})))"
```

Expected: 每个元素的 `h >= 44`（修复前为 `h:36, native:45`）

- [ ] **Step 3: 全量复测并对基线做逐页 diff**

Run: `bash scripts/contrast-audit.sh .audit/final`

Expected：

- 全部页面 `total=0`（`total` 已是**可执行缺陷**口径，不含 WCAG 未激活豁免项），**除**明确标记为「空态未覆盖」者（`favorite/list` 等 textLen 极小页）
- 逐页核对 `exempt`：每个豁免项都必须能被指认为**未激活组件**（class 含 `disabled` / `aria-disabled`）。**任何拿 `exempt` 掩盖可执行缺陷的情况都视为失败** —— 必要时逐个列出 `exemptSample` 复核
- 与 `.audit/baseline` 对比：**无任何页面数值上升**

```bash
# 守卫：.audit/ 是 gitignore 的，基线可能不存在。缺基线时 diff 会拿空字典对比并
# 静默输出「无回归」—— 那正是本项目栽过的假通过。必须先确认基线非空。
test -n "$(ls -A .audit/baseline/*.json 2>/dev/null)" || {
  echo "基线缺失，先重建："; bash scripts/contrast-audit.sh .audit/baseline; }
python3 - <<'PY'
import json, glob, os, sys
def load(d):
    out={}
    for f in glob.glob(d+'/*.json'):
        try: out[os.path.basename(f)]=json.loads(json.loads(open(f).read().strip()))['total']
        except Exception: pass
    return out
b,a=load('.audit/baseline'),load('.audit/final')
if not b: sys.exit('FATAL: 基线为空，拒绝给出「无回归」结论')
missing=[k for k in b if k not in a]
if missing: sys.exit('FATAL: 复测缺页 %s（测量不完整，不得判定通过）' % missing)
for k in sorted(set(b)|set(a)):
    if a.get(k,0)>b.get(k,0): print('REGRESSION', k, b.get(k), '->', a.get(k))
print('所有页面缺陷数：', {k:a.get(k) for k in sorted(a) if a.get(k)})
PY
```

- [ ] **Step 4: 留前后截图（改动最大的 6 页）**

```bash
export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"
mkdir -p .audit/shots
for p in pages/login/index pages/index/index "pages/order/detail?id=1" "pages/photographer/detail?id=1" pages/photographer/services pages/photographer/works; do
  n=$(echo "$p" | tr '/?=' '___')
  agent-browser open "http://localhost:5173/#/$p" >/dev/null 2>&1
  agent-browser reload >/dev/null 2>&1; sleep 3
  agent-browser screenshot ".audit/shots/$n.png" >/dev/null 2>&1
done
ls -1 .audit/shots
```

- [ ] **Step 5: 跑门禁**

```bash
cd server && export GOROOT=/home/user/go-sdk/go1.22 && export PATH=$GOROOT/bin:$PATH && \
  go build ./... && go vet ./... && go test ./...
cd .. && npx vue-tsc --noEmit
bash server/scripts/smoke.sh
npm run build:h5
npm run build:mp-weixin
```

Expected: 全部退出码 0

- [ ] **Step 6: Commit**

```bash
git add src/pages/settings/index.vue
git commit -m "fix(cend): F4b 设置页开关去掉 scale(0.8)，命中区 36→45px"
```

---

## 完成标准（全部达成才算收工）

- [ ] 四族 + switch 全部达标，`scripts/contrast-audit.sh` 输出无对应签名，且所有页面 `total=0`（可执行口径）
- [ ] 逐页复核 `exempt` 清单：每项都确属 WCAG 未激活组件；无任何可执行缺陷被豁免掩盖
- [ ] 与基线逐页 diff 无回归
- [ ] `go build/vet/test` + `vue-tsc` + `smoke.sh` + `build:h5` + `build:mp-weixin` 全绿
- [ ] 前后截图留存
- [ ] 更新 `AGENTS.md`：在「Dark Theme Overrides」补 `on-neon-fill` / `neon-pill` / `$error-bright` 说明，避免后续再写 `#fff` 压霓虹底
- [ ] 开 PR（stack 在 PR #1 上），PR 描述含前后对比数字表

## 已知风险与回退

| 风险 | 处置 |
|---|---|
| 误伤实测达标的青/绿/金同色 pill | 每 Task 后跑实测；若某页数值上升即回退该块 |
| `@include` 在 scoped 样式中不可用 | 理论不会（`variables.scss` 经 `additionalData` 注入每个 SFC）；若报未定义，改为在 `global.scss` 定义 utility class |
| mp-weixin 编译差异 | Task 7 强制 `npm run build:mp-weixin` |
| 全站按钮观感变化引起不满 | 已在上游 spec §7 知情确认；如需回退，整分支 `git revert` 即可（独立 PR，不影响 #1） |
