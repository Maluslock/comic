# C 端霓虹配色对比度收敛 — 设计

- **日期**：2026-09-24
- **分支**：`fix/cend-contrast-aa`（stack 在 `feat/photographer-pricing` 之上，单独 PR）
- **上游依据**：`docs/qa/2026-09-21-C端多页UI审计-第二批.md`
- **背景 PR**：https://github.com/Maluslock/comic/pull/1

## 1. 背景与目标

第二批 UI 审计（17 页）发现跨页复现的配色对比度问题，指出应「在共享 SCSS / token 层收敛，避免逐页打补丁」——第一批只在「我的套餐管理页」单页打了补丁，本次证明是**全站配色模式**。

本次目标：把实测不达 WCAG AA 的配色**收敛到共享 mixin 层**并全站修正；**不再复发**。

### 与审计文档的一处校正

审计 §4 写「在共享 SCSS / token 层统一修 → 一次修好所有页」。**这在代码里不成立**：token 是共享的，但承载缺陷的**规则是逐页 scoped 复制**的（`global.scss` 的 `.tag` 甚至不是渲染用的那套）。因此本设计不是"改一个 token"，而是：

1. 新增共享 **mixin** 作为单一事实源；
2. 把散落的 ~30 处 scoped 规则**迁移**到 mixin。

## 2. 权威基线（实测量测，2026-09-24）

视口 390×844，dev（`localhost:5173`，API `:8088`）。方法：注入式脚本遍历文本节点，计算「实际前景色 vs 有效背景色」的对比度（含半透明叠加与**渐变停靠点取最差值**），按 WCAG AA 判定（正文 ≥4.5:1，大字 ≥3:1）。

### 缺陷族

| 族 | 形态 | 实测最差 | 处数 / 页数 | 说明 |
|---|---|---|---|---|
| **F1** | 浅字压霓虹底（青/渐变/紫实底） | **1.97:1** | 25 / 11 | 含**登录页主按钮**（全站最差） |
| **F2** | 紫字压紫底 pill | 3.80:1 | 40 / 4 | 首页/漫展详情/摄影师列表/摄影师详情 |
| **F3** | 红字压红底 | 4.32:1 | 2 / 1 | `photographer/works.vue` 删除按钮 |
| **F4** | 三级文本压卡片 | 4.49:1 | 10 / 3 | 与 4.5 差 0.01，属舍入边界 |
| **F4b** | `uni-switch` 触控高 36px | — | 1 / 1 | 触控目标，非对比度 |

> **两个口径不等价**：本表「处数/页数」是**实测发现数**（一个规则可在一个页面产生多处发现，如首页 9 个标签 = 9 处）；§4 的「块数/文件数」是**需改动的规则数**。两者都不可互相推算 —— 交互态（`:active` / `.active`）在默认渲染态测不到，只体现在块数里；而一处规则可能产生几十处发现。

### 关键实测数据

| 前景 | 有效背景 | 对比度 | 位置 |
|---|---|---|---|
| `#e2e8f0` (`$dark-text-primary`) | `rgb(6,182,212)` 青 | **1.97** | **登录页 `.login-btn-text`** |
| `#fff` | 青 | 2.43 | 首页渐变条、订单详情状态横幅、各「去逛逛」CTA、摄影师侧按钮 |
| `rgba(255,255,255,.8/.9)` | 青 | 2.43 | 订单详情横幅 |
| `#fff` | `#a855f7` 紫 | 3.96 | 摄影师详情「选择」、个人中心「收藏」 |
| `#a855f7` | `rgb(44,32,69)` 紫底 | 3.80 | 标签 pill ×36 |
| `#a855f7` | `rgb(41,28,73)` 紫底 | 3.92 | 摄影师列表 `▼` |
| `#ef4444` | `rgb(44,27,41)` 红底 | 4.32 | 删除按钮 ×2 |
| `#7b8aa3` | `rgb(34,34,48)` 卡片 | 4.49 | 预约/评价/摄影师详情 元信息 ×10 |

### 复核结论：审计数字可信

审计报的 **2.43:1 与 3.8:1 完全复现**。此前基于 token 手算「3.8 对不上」的怀疑**作废** —— 实测有效背景是 `rgb(44,32,69)`（比手算的底更亮），手算低估了叠层。

### 不达标族的边界（重要）

「同色系文字压同色半透明底」**并非一律不达标**，取决于色相亮度：

| 色相 | 对比度 | 判定 |
|---|---|---|
| 青 `#06b6d4` | ~5.5 | ✅ 达标，**不动** |
| 绿 `#22c55e` | ~5.8 | ✅ 达标，**不动** |
| 金 `#f59e0b` | ~5.9 | ✅ 达标，**不动** |
| 紫 `#a855f7` | 3.80 | ❌ 修 |
| 粉 `#ec4899` | ~3.98 | ❌ 修（`activate.vue .btn-services`） |
| 红 `#ef4444` | 4.32 | ❌ 修（实测确认） |

因此**不能对 `color: $neon-*` 做全局替换** —— 必须按实测结果定点修。

## 3. 设计：共享 mixin 层

`src/styles/variables.scss` 已由 `vite.config.ts` 的 `additionalData` 注入**每个 SFC**，新增 mixin 天然全局可用。

```scss
// 压霓虹实底的文字：紫端 4.95:1 / 青端 8.07:1 / 粉端 5.56:1
@mixin on-neon-fill {
  color: $dark-bg-primary;
}

// 同色系低对比 pill：亮字压同色暗底 → 实测 ≥5.7:1（原 3.8:1）
@mixin neon-pill {
  color: $neon-purple-bright;
  background: $neon-purple-dim;
}
```

### 刻意的边界收窄

mixin **只管颜色**，不动 `padding` / `font-size` / `border-radius` / `border`：

- 各处 pill 几何差异大（`10rpx 28rpx` vs `6rpx 20rpx`，字号 `$font-size-sm` vs `22rpx`），把几何塞进 mixin 会**改动布局**——那是另一个风险类别；
- `border` 是纯装饰，不参与对比度计算。

### F3（红）的处理

红色 `#ef4444` 无对应的 bright token。方案：新增 `$error-bright`（建议 `#f87171`，需实测校准 ≥4.5:1），或就地改用 `$error-color` 的提亮值。**实施时必须先算后测**，不留"看起来更亮"的臆断。

### F4（三级文本）的处理

`$dark-text-tertiary` 当前 `#7b8aa3`（上个会话刚从 `#64748b` 提上来，影响 31 文件）。实测 4.49:1 差 0.01。**决策：微调 token**（约 +0.4%，如 `#7e8da6`）使其明确达标。**知情代价**：二次触动那 31 个文件（视觉上几乎不可见）。

### F4b（switch）

给设置页开关容器补 `min-height: 44px`。仅此一处。

## 4. 迁移范围

### 权威范围 = 实测 ∪ 静态扫描

**范围以实测为准**。静态扫描（brace-balanced，仅顶层声明）用于**定位**，已排除两类假阳性：

- 嵌套子块的声明误算进父块（如 `.section-title` 的 `::before` 渐变装饰条 —— 文字其实压深卡片，**不是**渐变）→ 已修正扫描器
- 渐变裁剪文字（`-webkit-text-fill-color: transparent`）→ 测量脚本已排除

**注意静态扫描会漏 parent/child 拆分写法**：登录页把 `background: $neon-gradient` 放在父 `.login-btn`、`color` 放在子 `.login-btn-text`，静态扫描（要求同块）抓不到，**只有实测才发现**。这正是不以 grep 定范围的实证理由。

### F1 — 霓虹底浅字（19 块 / 14 文件 + 登录页）

```
booking/index.vue        .btn-primary
chat/index.vue           .send-btn
comment/index.vue        .btn-primary
event/detail.vue         .btn-retry
favorite/list.vue        .btn-go
follow/list.vue          .btn-go
index/index.vue          .badge, .cta-primary, .btn-retry
message/index.vue        .unread-count
order/detail.vue         .btn-primary
photographer/activate.vue     .btn-go, .btn-submit
photographer/cert-apply.vue   .btn-reapply, .btn-submit
photographer/profile-edit.vue .btn-submit
photographer/works.vue   .btn-add, .btn-submit
profile/edit.vue         .btn-submit
login/index.vue          .login-btn / .login-btn-text   ← 实测发现，最差
```

### F2 — 紫底 pill（8 块 / 7 文件，含 `▼`）

```
event/detail.vue              .tag-item
index/index.vue               .tag-item
message/index.vue             .notification-icon.success
photographer/activate.vue     .btn-cert, .mode-item.active
photographer/detail.vue       .tag
photographer/profile-edit.vue .mode-item.active
search/search.vue             .hot-item:active
```

### F3 — 红底（1 文件）

```
photographer/works.vue        .btn-del          （实测 4.32）
photographer/services.vue     .btn-del, .img-del（同模式，需实测确认）
photographer/works.vue        .img-del
```

### 顺带修正的同类项（需实测确认，非自动扩大）

`activate.vue .btn-services`（粉）、`.btn-works`（青，预期达标）、`.is-cert`（金，预期达标）、`services.vue .btn-edit`（青，预期达标）—— **预期达标者不动**，以复测数字为准。

## 5. 验收标准（可执行）

实施后用同一审计脚本对全部页面复测：

1. **F1/F2/F3/F4 四族全部消除**（脚本 `total` 中不再出现对应签名）
2. 登录页 `total = 0`
3. 设置页 switch 高度 ≥44px
4. **不得引入新发现**（对比例表逐页 diff，只降不升）
5. 门禁：`go build ./... && go vet ./... && go test ./...`、`npx vue-tsc --noEmit`、`smoke.sh`（dev + prod）
6. 视觉回归：对改动最大的页（登录、首页、订单详情、摄影师详情、我的套餐、我的作品）留前后截图

## 6. 验证方法与纪律（上一会话两度假通过的教训）

审计脚本从 `.audit/` 提升为入库工具 **`scripts/contrast-audit.js`**，使其可重复，而非一次性脚本。

**必须遵守的测量纪律**（每条都是实测踩出来的）：

1. **`agent-browser open` 只换 hash 时 SPA 不整页重载** → 注入 token 后必须 `reload`，否则测的是**未登录态**（本次首轮就中招，把登录页缺陷误当成多个摄影师页的缺陷）
2. 每页断言 `document.querySelectorAll('*').length > 20`
3. **区分「真通过」与「空态未覆盖」**：`favorite/list` 仅 25 字（空态），其 `total=0` 无意义，须标记为未覆盖
4. **渐变遮挡**：存在渐变时，其下的兜底底色**不是**有效背景色。脚本曾把兜底底色也当候选，导致「深字压渐变」被误报为 1.00:1 —— 修复后 `services.vue` 假阳性消失。**若不修，修复后会产生大量假失败**
5. 排除 uni-app 内置元素（TabBar / swiper-dot / page-head-btn）与渐变裁剪文字
6. `agent-browser` 需 `export PATH=/home/user/.nvm/versions/node/v22.16.0/bin:$PATH`（不在默认 PATH）
7. dev API 二进制 `/tmp/comic-api` 随 tmpfs 消失 → 用 `server/build/comic-api`（已 gitignore）并以托管后台任务运行

## 7. 风险与取舍

| 项 | 说明 |
|---|---|
| **全站主按钮观感变化** | 18 处 CTA 白字→深字。技术上深字在整条紫→青渐变上达标（4.95–8.07），但确会改变品牌观感。这是范围 A 的既定代价，非本设计可回避 |
| **F4 token 二次触达 31 文件** | 已确认接受 |
| **误伤达标项** | 青/绿/金的同色系 pill **实测达标**（5.5/5.8/5.9），mixin 迁移时**不得**顺手套用；只改实测不达标者 |
| **mp-weixin 平台** | mixin 在编译期展开，跨平台无运行时风险；但需构建验证 `npm run build:mp-weixin` |

## 8. 不在本次范围

- `order/list`、`message` 等页面**空态下的对比度**（无内容可测，标记未覆盖）
- 系统性 load-time 错误（每页 1 条 `{text:"Object"}`）—— 已单独立项建议，需 CDP 级抓栈
- 订单详情「拍摄时长」恒 0（DTO 缺字段）—— 属后端缺陷，非配色
- 触控目标的其他历史问题（除 switch 外）
