# PROJECT KNOWLEDGE BASE

**Generated:** 2026-07-06
**Updated:** 2026-09-07 (摄影师作品管理 + 摄影师主页管理 + 用户资料编辑 C 端模块 + AGENTS.md + Gitee push)
**Commit:** fd2e673
**Branch:** master

## OVERVIEW
米拉漫展 (Mira Comic-Con) — Uniapp mini-program connecting cosplayers with photographers at comic conventions. Vue 3 + TypeScript + Pinia + Vite, targeting WeChat MP (mp-weixin) and H5. Real Go backend (Gin + PostgreSQL + Redis on :8080) with legacy std-lib mock (:8081) for standalone demos. **Dark neon theme** (purple #a855f7 + cyan #06b6d4 on #0a0a1a base), event-driven UX, sitewide dark conversion in progress.

## STRUCTURE
```
comic/                         # 116 files, ~9k LOC
├── src/
│   ├── api/index.ts           # API client: real HTTP to :8080 + mock fallback
│   ├── components/            # 6 presentational components (all dark)
│   │   ├── PhotographerCard.vue   # Dark: circle avatar + left neon accent
│   │   ├── WorkCard.vue           # Dark: image + gradient overlay + neon border
│   │   ├── QuickActionBar.vue     # 4-grid shortcut (neon border + glow)
│   │   ├── HomeSkeleton.vue       # Shimmer skeleton for homepage
│   │   ├── ReviewCard.vue         # Dark: star rating + content + images
│   │   └── ServiceCard.vue        # Dark: checkbox + price + neon active state
│   ├── data/mock.ts           # Mock data (4 photographers, 5 events, 4 works)
│   ├── pages/                 # 11 page dirs, 13 .vue files
│   │   ├── index/             # Homepage (800 lines, 7-section, dark) ★ TabBar
│   │   ├── event/detail.vue   # Event detail (600 lines, cover+countdown, dark)
│   │   ├── photographer/      # list.vue (dark) + detail.vue (472 lines, dark)
│   │   ├── search/search.vue  # Search + tag filter (dark)
│   │   ├── booking/index.vue  # Booking form (dark)
│   │   ├── message/index.vue  # Notifications + real chat sessions (dark) ★ TabBar
│   │   ├── profile/index.vue  # User center (dark, gradient header) ★ TabBar
│   │   ├── favorite/list.vue  # My favorites (new, dark)
│   │   └── [4 light]          # comment, portfolio, chat, order/list, order/detail
│   ├── static/
│   │   ├── tab/               # TabBar icons (81×81 PNG, Material Symbols)
│   │   └── icons/             # SVG icons (search.svg, bell.svg)
│   ├── stores/                # Pinia (composition API)
│   │   ├── home.ts            # Home data + city filter (173 lines, used by index)
│   │   ├── user.ts            # Auth state (consumed by login + booking/order flows)
│   │   └── chat.ts            # Chat state (NOT consumed)
│   ├── styles/
│   │   ├── variables.scss     # Design tokens: light $ + dark $dark-* / $neon-*
│   │   └── global.scss        # Global utility classes
│   ├── types/index.ts         # All domain interfaces (149 lines)
│   ├── App.vue, main.ts, manifest.json, pages.json, uni.scss
├── server/
│   ├── mock_server.go         # Go std lib mock (:8081, 5 events + 4 photographers)
│   ├── cmd/api/main.go        # Gin+PostgreSQL backend (needs Go 1.22+)
│   └── internal/              # handler/ service/ repository/ db/ config/
├── vite.config.ts, tsconfig.json, index.html, package.json
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add a new page | `src/pages.json` → `src/pages/<name>/` | Create dir + index.vue, register in pages.json |
| Add API endpoint | `src/api/index.ts` | Add real HTTP client in `src/api/client.ts`; keep `src/data/mock.ts` fallback |
| Add data model | `src/types/index.ts` | All interfaces live here (User, ComicEvent, Booking, etc.) |
| Add Pinia store | `src/stores/` | Composition API (`defineStore` + `ref`/`computed`) |
| Add global style | `src/styles/global.scss` | Auto-imported by App.vue |
| Add SCSS variable | `src/styles/variables.scss` | Auto-injected into all SFCs via vite.config.ts |
| Change tab bar | `src/pages.json` → `tabBar` | 81×81 PNG icons in `src/static/tab/` |
| Platform config | `src/manifest.json` | h5 / mp-weixin (vueVersion: "3") |
| Build config | `vite.config.ts` | Plugin + SCSS preprocessor |
| TypeScript config | `tsconfig.json` | Path alias @/*, strict mode, includes .vue |
| uview-plus config | `src/uni.scss` + `src/main.ts` + `src/pages.json` easycom | Theme, plugin, auto-import |

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `createApp` | fn | `src/main.ts` | SSR bootstrap — Vue + Pinia + uviewPlus |
| `useUserStore` | store | `src/stores/user.ts` | Auth state; consumed by login + booking/order flows |
| `useChatStore` | store | `src/stores/chat.ts` | Chat state; NOT yet consumed by any page |
| `useHomeStore` | store | `src/stores/home.ts` | Home data (loads /api/v1/home→mock fallback); consumed by index page |
| `getHomeData` | api fn | `src/api/index.ts` | Home mock fallback（仅当 `/v1/home` 失败时用）；其余 C 端页面一律直连 `apiGet`/`apiPost`（2026-09-14 清理了 10 个仅返回 mock 的死函数，293→178 行） |
| `loginByPhone` | api fn | `src/api/index.ts` | Phone + code login → token |
| `getMe` | api fn | `src/api/index.ts` | Current user via Bearer token |
| `addFavorite` / `removeFavorite` / `getFavorites` | api fn | `src/api/index.ts` | Favorite create/delete/list (real HTTP + mock fallback) |
| `createChatSession` / `getChatMessages` / `sendChatMessage` / `getChatSessions` | api fn | `src/api/index.ts` | Chat session/messages/sessions (real HTTP + mock fallback) |
| `User` | interface | `src/types/index.ts` | Base user (role: photographer\|coser) |
| `Photographer` | interface | `src/types/index.ts` | Extends User with works/services/reviews/rating |
| `ComicEvent` | interface | `src/types/index.ts` | Comic convention (name, venue, dates, photographerCount) |
| `Work` | interface | `src/types/index.ts` | Portfolio work (images, tags) |
| `Booking` | interface | `src/types/index.ts` | Booking with status enum |
| `PhotographerCard` | component | `src/components/PhotographerCard.vue` | Dark card: 100rpx circle avatar, left 4rpx neon-purple border, neon tags |
| `WorkCard` | component | `src/components/WorkCard.vue` | Dark image card: 240rpx cover, gradient overlay, neon border ring, cyan-on-purple tags |
| `ServiceCard` | component | `src/components/ServiceCard.vue` | Dark service tier: checkbox + price + neon glow active state |
| `ReviewCard` | component | `src/components/ReviewCard.vue` | Dark review: avatar + star rating + content + image thumbnails |
| `QuickActionBar` | component | `src/components/QuickActionBar.vue` | 4-grid shortcut nav (neon border, dark bg) |
| `HomeSkeleton` | component | `src/components/HomeSkeleton.vue` | Shimmer skeleton for homepage loading |
| `HomeResponse` | interface | `src/types/index.ts:143` | Backend DTO: banners + upcomingEvents + hotTags + recommendedPhotographers + featuredWorks |
| `mockEvents` | data | `src/data/mock.ts` | 5 mock comic events (CP30, CD28, 萤火虫, IDO42, CJ) |
| `mockPhotographers` | data | `src/data/mock.ts` | 4 mock photographers |
| `mockServices` | data | `src/data/mock.ts` | 4 service tiers (¥399-¥1299) |

## AUTH & REAL DATA WIRING

Implemented in the full-pipeline integration (Tasks 1-14). The frontend now consumes the real Go backend (`:8080`) for the main user flows, with `src/data/mock.ts` retained as a fallback when the backend is unreachable.

### Backend auth stack

| Layer | Location | Purpose |
|-------|----------|---------|
| `users` / `user_tokens` tables | `server/db/migrations/000007_users.up.sql` | Persistent user identity and token storage |
| Token middleware | `server/internal/middleware/auth.go` | Validates `Authorization: Bearer <token>`, injects `userID` into context, returns 401 on missing/invalid tokens |
| Login handler | `server/internal/handler/auth.go` | `POST /api/v1/login` with phone + code; creates user + token, returns JWT-style token |
| Me handler | `server/internal/handler/auth.go` | `GET /api/v1/me` returns current user (id, name, phone, avatar) |

### Frontend auth client

| Symbol | Location | Role |
|--------|----------|------|
| `client.ts` | `src/api/client.ts` | `uni.request` wrapper that injects `Authorization: Bearer <token>` from `uni.getStorageSync('token')` and routes 401 responses to the login page |
| `loginByPhone` | `src/api/index.ts` | Calls `POST /api/v1/login`, stores token, updates `useUserStore` |
| `getMe` | `src/api/index.ts` | Calls `GET /api/v1/me` on app startup / profile refresh |
| `useUserStore` | `src/stores/user.ts` | Reactive auth state (token, userInfo, isLoggedIn) |

### Pages wired to real API

| Page | File | API used |
|------|------|----------|
| Login | `src/pages/login/index.vue` | `loginByPhone` |
| Profile | `src/pages/profile/index.vue` | `getMe` |
| Photographer list | `src/pages/photographer/list.vue` | `getPhotographers` |
| Photographer detail | `src/pages/photographer/detail.vue` | `getPhotographerDetail` |
| Search | `src/pages/search/search.vue` | `getTags`, `getPhotographers` |
| Portfolio | `src/pages/portfolio/index.vue` | `getWorks` |
| Booking | `src/pages/booking/index.vue` | `createBooking` (requires login) |
| Order list | `src/pages/order/list.vue` | `getOrders` |
| Order detail | `src/pages/order/detail.vue` | `getOrderDetail` |
| Comment | `src/pages/comment/index.vue` | `createReview` |
| Favorite list | `src/pages/favorite/list.vue` | `getFavorites` |
| Chat | `src/pages/chat/index.vue` | `createChatSession`, `getChatMessages`, `sendChatMessage` |
| Message | `src/pages/message/index.vue` | `getChatSessions` |

All endpoints fall back to mock data when `uni.request` fails, preserving standalone H5/WeChat DevTools demos without the backend running.

## CONVENTIONS

- **Vue 3 `<script setup lang="ts">`** — Composition API only, no Options API
- **SCSS variables auto-injected** — `variables.scss` auto-injected into every SFC via `vite.config.ts additionalData`
- **`uni.scss`** — uview-plus theme entry point; imported by uni-app build system
- **rpx units** — all spacing/sizing uses rpx (not px/rem)
- **Design tokens** — use SCSS variables only (`$primary-color`, `$spacing-md`, etc.), no hardcoded values
- **Path alias `@/`** — maps to `src/`; always `@/components/...`, never relative imports
- **TypeScript strict mode** — `"strict": true`, no `any` tolerated
- **Uniapp lifecycle hooks** — `@dcloudio/uni-app` hooks (`onLaunch`, `onShow`), not raw Vue lifecycle
- **Pages as dirs** — `src/pages/<name>/index.vue` with scoped `<style lang="scss" scoped>`
- **Pinia composition stores** — `defineStore('name', () => { ... })` pattern
- **Image strategy** — DiceBear avataaars for user avatars; picsum.photos for covers/works; PNG tabBar icons from Material Symbols
- **File operations** — use Write/Edit tools, NOT PowerShell file operations (PowerShell corrupts UTF-8)

### Dark Theme Conversion
When converting a light-themed page to dark neon theme:

| Light Variable | Dark Replacement | Usage |
|----------------|-----------------|-------|
| `$bg-page` | `$dark-bg-primary` (#0a0a1a) | Page background |
| `$bg-primary` | `$dark-bg-card` (rgba 5% white) | Card backgrounds |
| `$bg-secondary` | `$dark-bg-secondary` (#12122a) | Secondary panels |
| `$text-primary` | `$dark-text-primary` (#e2e8f0) | Headings, primary text |
| `$text-secondary` | `$dark-text-secondary` (#94a3b8) | Body text |
| `$text-tertiary` | `$dark-text-tertiary` (#64748b) | Meta, timestamps |
| `$primary-color` | `$neon-purple` (#a855f7) | Accent color |
| `$border-color` | `$dark-border` (rgba 8% white) | Dividers, borders |
| `$shadow-*` | `0 X Y rgba(0,0,0,0.2~0.4)` | Dark card depth |
| `#ccc` / `#f0f0f0` / `#ddd` | `$dark-bg-card-hover` / `$dark-text-tertiary` | Eliminate ALL hardcoded greys |

Additional rules:
- Add `1rpx solid $dark-border` + box-shadow to all card containers
- Section titles: neon gradient `::before` left bar (6rpx × 28rpx, `$neon-gradient`)
- Active states: `$neon-purple` + `box-shadow: 0 0 12rpx $neon-purple-glow`
- Emoji → SVG icon (`/static/icons/`) or geometric symbol (★ ☆ ◇ ◆ →)
- Preserve header gradients when converting profile pages

## ANTI-PATTERNS (THIS PROJECT)

- **Don't use relative imports across src/** — always `@/` alias
- **Don't add px-based styles** — use rpx or SCSS variables
- **Don't use Options API** — `<script setup>` only
- **Don't commit without `.gitignore`** — still missing at root
- **Don't call `uni.request` directly from pages** — go through `src/api/index.ts`
- **Don't edit files with PowerShell** — use Write/Edit tools to preserve UTF-8 encoding
- **Don't use uview-plus JS components on mp-weixin** — path issues on mini-program; use CSS layer only, fall back to native components

## UNIQUE STYLES

- Design token system in `src/styles/variables.scss` (semantic naming: `$primary-color`, `$shadow-md`, `$spacing-lg`)
- Global utility classes in `global.scss`: `.card`, `.btn-primary`, `.btn-outline`, `.tag`, `.flex`, `.ellipsis`
- Primary brand: `#6366f1` (indigo), shadows: `$shadow-sm`→`$shadow-md`→`$shadow-lg`→`$shadow-focus` depth scale
- Section titles: left colored bar accent (`::before` 6rpx×28rpx, `$primary-color`)
- Card depth: resting=`$shadow-md`, pressed=`$shadow-sm`, flat=none
- PhotographerCard: circle avatar (50%), left purple border accent (6rpx)
- Tags: purple outline pills (`border-radius-xl`, `rgba($primary, 0.08)` bg)

### Dark Theme Overrides
- **Dark neon theme**: `$dark-bg-primary: #0a0a1a` base with `$neon-purple: #a855f7` + `$neon-cyan: #06b6d4` dual-color accents
- **Dark cards**: semi-transparent `$dark-bg-card` (rgba white 5%) + `1rpx solid $dark-border` + dark shadow
- **Neon glow**: `$neon-purple-glow`, `$neon-cyan-glow` for active states
- **Gradient**: `$neon-gradient: linear-gradient(135deg, $neon-purple, $neon-cyan)` — hero title, section bars, CTAs
- **Cards press**: active → `scale(0.97-0.98)` + reduced shadow + enhanced border glow
- **PhotographerCard** (dark): 100rpx circle avatar, 3rpx neon-purple border ring, left 4rpx solid purple accent, neon tags
- **WorkCard** (dark): 240rpx cover, gradient overlay, neon border ring, cyan-on-purple tags

## COMMANDS
```bash
npm run dev:h5            # H5 dev server (port 5173)
npm run build:h5          # H5 production → dist/build/h5
npm run dev:mp-weixin     # WeChat MP dev → dist/dev/mp-weixin (watch mode)
npm run build:mp-weixin   # WeChat MP production → dist/build/mp-weixin

# Go backend (GOROOT=/home/Haxlock/go, GOPROXY=https://goproxy.cn,direct)
go run ./cmd/api                  # API server (:8080, Gin + PG :5433 + Redis :6379)
go run ./cmd/ingest               # One-shot nyato crawl (1 page)
go run ./cmd/ingest -pages 5      # Crawl 5 pages once
go run ./cmd/ingest -config config/cron.yaml   # YAML-driven cron daemon (recommended)
```

### Ingest Cron (YAML-driven)
`server/config/cron.yaml` defines tasks; `cmd/ingest -config` registers them via robfig/cron:

| Task | Schedule | Source | Action |
|------|----------|--------|--------|
| `nyato-daily-full` | `0 3 * * *` (03:00) | nyato, 5 pages | Full daily crawl + upsert |
| `nyato-afternoon-incremental` | `30 14 * * *` (14:30) | nyato, 2 pages | Daytime additions |
| `expire-cleanup` | `0 1 * * *` (01:00) | expire | Mark past events del_flag=true |
| `bilibili-sync` | disabled | bilibili | Best-effort Bilibili source (empty) |

- `enabled: false` tasks are skipped at startup
- Every crawl also marks expired events (`start_date < NOW()`) as deleted
- WAF note: nyato.com rejects Go HTTP/2 (`stream error: INTERNAL_ERROR`) — `internal/ingest/nyato.go` forces HTTP/1.1 + browser UA/Referer; do not revert
- Scraped text is UTF-8 sanitized before upsert (SQLSTATE 22021 guard)

WeChat DevTools → import `dist/build/mp-weixin`. No lint/test scripts yet. To type-check: `npx vue-tsc --noEmit`.

## 订单状态机

Booking order lifecycle and conflict rules implemented across Tasks 1-8.

### State transitions

| Method | Endpoint | Valid transition | Response |
|--------|----------|------------------|----------|
| `PUT` | `/api/v1/bookings/:id/status` | `pending` → `confirmed` | 200 |
| `PUT` | `/api/v1/bookings/:id/status` | `confirmed` → `completed` | 200 |
| `PUT` | `/api/v1/bookings/:id/status` | `pending`/`confirmed` → `cancelled` | 200 |
| `PUT` | `/api/v1/bookings/:id/status` | any other transition (e.g. `completed` → `pending`) | 409 Invalid transition |

### Conflict detection & timeslots

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/bookings` | 409 if the same photographer/date/time already has a non-cancelled booking |
| `GET` | `/api/v1/photographers/:id/timeslots?date=YYYY-MM-DD` | Returns `{ "occupied": ["14:00", ...] }` for the photographer on that date |

### Join fields

Order list/detail responses include joined photographer/service fields:

| Field | Source |
|-------|--------|
| `photographerName` | `photographers.name` joined via `photographer_id` |
| `photographerAvatar` | `photographers.avatar` joined via `photographer_id` |
| `serviceName` | `services.name` joined via `service_id` |

### Frontend

- `src/api/client.ts` exposes `apiPut(path, body)` used by order actions.
- `src/pages/order/list.vue` calls `apiPut(\`/v1/bookings/${id}/status\`, { status })` for confirm/complete/cancel and handles 409 with a toast.
- `src/pages/booking/index.vue` loads `GET /v1/photographers/:id/timeslots` and marks occupied slots as disabled.

## 完整性约束与核心不变量（2026-09-24 第二轮整体 review）

第二轮 review（业务逻辑 + 核心功能）发现并修复了 6 个缺陷（报告 B-10~B-15）。以下约束是**不变量**，
改动相关代码时不要绕过 —— 每一条都对应一次真实复现。

### 订单/预约（server/migrations/000029 + booking_service.go）

| 不变量 | 实现 | 为什么不能只在应用层做 |
|--------|------|------------------------|
| 同一摄影师同一 date+time 只能有一个未取消订单 | 部分唯一索引 `uniq_bookings_active_slot (photographer_id, date, time) WHERE status <> 'cancelled'` + `Create` 把 23505 映射成 `ErrConflict`(409) | `CountConflictBookings`(SELECT) → INSERT 是 TOCTOU。实测 12 个线程屏障对齐的并发请求曾 **7 个成功、7 条订单落同一档期**；加索引后复测 1 个 201 / 11 个 409。`WHERE status <> 'cancelled'` 是为了让取消释放时段 |
| 不能预约过去的档期 | `isPastSlot(date, time, now)`：按**服务器本地时区**，日期是硬约束，今天的已过时刻同样拒绝；时刻不可解析时退化为只比日期 | 前端 picker 的 `:start` 只是 UX，小程序端可绕 |
| 摄影师不能预约自己的套餐 | `Create` 比较下单人与 `GetPhotographerById().UserID` → `ErrSelfBooking`(400) | 自成交无业务含义，还会污染单量并给自评铺路 |

前端配套约定：**取本地日期一律用 `src/utils/date.ts` 的 `localDateKey`/`todayKey`，不要用
`new Date().toISOString().split('T')[0]`** —— 后者是 UTC，东八区 00:00–08:00 会得到「昨天」
（预约页默认日期曾因此落在过去的一天）。

### 评价（review_service.go + 迁移 000029）

- 评价必须对应**已完成**订单（`HasCompletedBookingWith`）→ 否则 403；不能自评；`uniq_reviews_user_photographer` 唯一索引兜重复 → 23505 映射 409。
- **评分与评价数每次评价后按 `reviews` 表重算**（`RecomputePhotographerRating`）。这两个字段此前是种子死数字（显示 4.9 分 / 234 条，真实只有 3 条评价），插入评价从不更新它们。重算是 best-effort（失败只记日志，下一条评价自我修正），不能让派生字段把已成功的评价变成「提交失败」。

### 摄影师列表排序（photographers.sql.go `SearchPhotographers` $7）

- 排序键：`all`（默认，评分优先）/`hot`（评价数）/`rating`/`order`（接单数）/`new`（入驻时间）。用 `CASE WHEN $7::text = ...` 参数绑定，**不要拼接字符串**。
- 每档末尾都有 `p.id DESC` 作为**稳定 tiebreaker**：排序键不唯一时 LIMIT/OFFSET 翻页会在页间重复或漏行（默认排序同样需要）。
- **`sort` 必须进列表缓存键**（`photographer_handler.go` 的 `cacheKey`），否则切筛选会命中另一档的缓存。
- 列表页筛选栏（热门/评分最高/接单最多/最新入驻）此前只改高亮、不传参数，四个筛选返回结果完全一致；现已接线。

### 聊天

- `MarkSessionRead` 现在也走 `assertParticipant`：非会话成员标记已读 → 403（与 `ListMessages`/`SendMessage` 口径一致）。此前返回 200 并给自己写一条无意义的 `chat_read_state` 行。

### 越权（IDOR）现状：已核查，基本干净

2026-09-24 做过一轮跨账号对抗测试：15 项「用 A 的 token 读写 B 的资源」（订单读取/改状态/报价/接受报价、
跨摄影师读接单、非本人会话读消息与发消息、收藏/关注/通知的越权写）**全部 403/404**。
模式是统一的：**路径/body 里的用户 id 不参与授权判断**，一律以 token 身份（`middleware.UserID(c)`）为准，
handler 用 `assertSelf` 或 `...ForUser` 后缀的 service 方法；`POST /favorites`、`POST /follows`、
`POST /notifications` 会直接**覆盖** body 里的 `userId`。新增接口请沿用这套模式。
（注意：这类探针容易误报 —— `POST /favorites {userId: 别人的}` 返回 201 看着像越权，实际落库是 token 用户；
判断越权要看**落库归属**，不能只看响应码。）

## 收藏 + 聊天模块

Favorites and real chat implemented across Tasks 1-9.

### Backend

| Table | Migration | Purpose |
|-------|-----------|---------|
| `photographer_favorites` | `server/migrations/000008_favorites_chat.up.sql` | User ↔ photographer many-to-many likes |
| `chat_sessions` | `server/migrations/000008_favorites_chat.up.sql` | 1:1 session between two users (ordered pair, `user1_id <= user2_id`) |
| `chat_messages` | `server/migrations/000008_favorites_chat.up.sql` | Messages within a session |

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/favorites` | Add favorite `{userId, photographerId}` → 201 `{ok:true}` |
| `GET` | `/api/v1/favorites/:userId` | List user's favorites with joined photographer fields |
| `DELETE` | `/api/v1/favorites/:userId/:photographerId` | Remove favorite → 204 |
| `POST` | `/api/v1/chat/sessions` | Get or create session `{userId, otherUserId}` → 201 `{id}` |
| `GET` | `/api/v1/chat/sessions/:userId` | List sessions with peer info + last message |
| `POST` | `/api/v1/chat/messages` | Send `{sessionId, senderId, content}` → 201 `{id}` |
| `GET` | `/api/v1/chat/messages/:sessionId` | List messages ascending |
| `POST` | `/api/v1/chat/sessions/:sessionId/read` | Auth required。标记该会话对调用者已读（`chat_read_state` upsert）→ 200 `{ok}` |
| `GET` | `/api/v1/chat/unread/:userId` | Auth required。调用者总未读数 → 200 `{count}`；path userId ≠ token → 403 |

### Frontend

| Page | File | API used |
|------|------|----------|
| Photographer detail | `src/pages/photographer/detail.vue` | `addFavorite`/`removeFavorite` (collect button), `createChatSession` → navigate to chat |
| My favorites | `src/pages/favorite/list.vue` | `getFavorites`, `removeFavorite` (取消) |
| Chat | `src/pages/chat/index.vue` | `getChatMessages`, `sendChatMessage` |
| Message | `src/pages/message/index.vue` | `getChatSessions` |

### Demo simplifications

- **No WebSocket** — messages are polled on page enter only.
- **未读数已真实**（2026-09-14）—— `chat_read_state(session_id,user_id,last_read_at)` 记录每人已读时间；`GET /chat/sessions/:userId` 返回每会话 `unreadCount`（`sender_id <> me AND created_at > last_read_at`）；进聊天页 `POST /chat/sessions/:id/read`；消息 TabBar 角标由 `src/utils/badge.ts` 设置。**仍无 WebSocket**（依赖微信域名），未读靠进页面刷新。
- **peer identity mapping** — `photographer.id` is treated as the peer `userId`; this holds for seed data where photographers have corresponding `users` rows.
- **Session ordering** — `chat_sessions.updated_at` is not bumped on new messages, so session list order may lag slightly behind latest activity.

## 关注漫展模块

Event follow (C-end) implemented in the event-follow module plan (4 tasks, commits `b6ff092`/`c3ff0ae`/`9fca693`). **Backend unchanged** — the pre-existing follows API is reused as-is; all changes are frontend-only.

### Frontend

| Piece | File | Role |
|-------|------|------|
| Shared follow utils | `src/utils/follow.ts` | `loadFollowedIds(userId)` → `Set<string>` from `GET /v1/follows/:userId`; `toggleFollow(userId, eventId, followed)` → POST/DELETE, returns new state |
| Event detail follow button | `src/pages/event/detail.vue` | `.follow-btn` on cover (`＋ 关注` ↔ `已关注`), state via `initFollowState()`/`onToggleFollow()`; not logged in → toast + redirect to login with `redirect` param |
| My follows page | `src/pages/follow/list.vue` | Registered in `pages.json` (custom nav, after `favorite/list`); refreshes on `onShow` (back-button safety); cards show cover/name/location·venue/date range/countdown; 取消 → `DELETE /v1/follows/:userId/:eventId` |
| Profile entry | `src/pages/profile/index.vue` | 关注的漫展 menu-item (◇ marker, after 我的收藏) → `uni.navigateTo('/pages/follow/list')` |
| Calendar binding | `src/pages/calendar/index.vue` | Follow state binds the logged-in user via `loadFollowedIds`/`toggleFollow` (was a hardcoded demo before); local handler renamed to `onToggleFollow` to avoid clashing with the shared util |

### Backend API (unchanged by this module)

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/follows` | `{userId, eventId}` → 200 `{ok:true}` |
| `GET` | `/api/v1/follows/:userId` | **Object response** `{list:[{id,eventId,name,location,venue,startDate,endDate,coverUrl,status}]}` — not a bare array |
| `DELETE` | `/api/v1/follows/:userId/:eventId` | Remove follow → 200 |

### Design notes

- **Shared Set source** — detail page, follows list and calendar all derive follow state from the same `/v1/follows/:userId` endpoint via `utils/follow.ts`, so state stays consistent across pages.
- **`onShow` refresh pattern** — follow/list refreshes on every show (same fix as the message page, commit `1ca894c`); keep this pattern for any page that shows follow state.
- **eventId is a string in the URL** — `id` route param is compared against `String(eventId)` from the API.

## 消息中心（P3 通知真实化）

Message-page notification area switched from hardcoded demo to real notifications (booking events → `notifications` table → message page), zero demo remnants sitewide. Implemented in the P3 notifications plan (5 tasks, commits `7214d4f`/`bc8a6c6`/`c10e5d0`/`515bf91` + Task 5 push), full-chain verified end-to-end in Task 5.

### Data model

`server/migrations/000011_notifications.{up,down}.sql` creates:

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGSERIAL | PK |
| `user_id` | BIGINT NOT NULL | Notification recipient |
| `type` | VARCHAR(20) default 'info' | `success` / `info` / `warning` |
| `title` | VARCHAR(100) NOT NULL | e.g. 预约成功 |
| `content` | TEXT NOT NULL | e.g. 您的摄影预约(#8)已提交，等待摄影师确认 |
| `read` | BOOLEAN default false | Reserved — no mark-read API yet |
| `created_at` | TIMESTAMPTZ default NOW() | Ordered by `idx_notifications_user(user_id, created_at DESC)` |

### Backend

| Piece | File | Role |
|-------|------|------|
| Repo | `server/internal/repository/notifications.sql.go` | `InsertNotification(ctx, userID, typ, title, content) (int64, error)` INSERT RETURNING id; `ListNotifications(ctx, userID)` — `WHERE user_id=$1 ORDER BY created_at DESC LIMIT 20`; `NotificationRow{ID,UserID,Type,Title,Content,Read,CreatedAt}` |
| Service | `server/internal/service/notification_service.go` | `NotificationService.Create/List`; `NotificationItem` JSON tags `id/type/title/content/read/createdAt` |
| Handler | `server/internal/handler/notification_handler.go` | `POST /api/v1/notifications` `{userId,type,title,content}` → 201 `{id}`; `GET /api/v1/notifications/:userId` → 200 **bare array** (not an object); both behind `AuthRequired` |

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/notifications` | Auth required. Insert → 201 `{id}` |
| `GET` | `/api/v1/notifications/:userId` | Auth required. Bare array, newest first, LIMIT 20 |
| `POST` | `/api/v1/notifications/:id/read` | Auth required. Mark the caller's own notification read → 200 `{ok}`；非本人/不存在 → 404（广播行 `user_id IS NULL` 不可标记，见 Known gaps） |

### Event triggers (booking_service)

`booking_service.go` `notify()` writes notifications after booking lifecycle transitions; write failure is logged and ignored (never blocks the main flow). **Both parties** are notified (2026-09-14: photographer-side added via `notifyPhotographer()`, which resolves `photographer_id → users.id` through `GetPhotographerById(...).UserID` and **skips silently when `UserID` is NULL** — legacy seed photographers):

| Trigger | Recipient | Type | Title | Content |
|---------|-----------|------|-------|---------|
| `Create` | coser | success | 预约成功 | `您的摄影预约(#%d)已提交，等待摄影师确认` |
| `Create` | **photographer** | info | 收到新预约 | `您收到一条新预约(#%d)，请及时确认接单` |
| `UpdateStatus` → confirmed | coser | info | 预约已确认 | 摄影师已确认接单，请按时赴约 |
| `UpdateStatus` → cancelled | coser | warning | 预约已取消 | 您的预约已被取消 |
| `UpdateStatus` → cancelled, actor=**coser** | **photographer** | warning | 预约已取消 | 对方取消了预约。 |
| `UpdateStatus` → completed | coser | success | 拍摄已完成 | 记得去评价本次拍摄哦 |
| `AdminUpdateStatus` → confirmed/cancelled/completed | coser + **photographer** | 同上 | 同上 | 管理员既非 coser 也非摄影师，双方都通知 |

- **Actor-awareness**：摄影师自己触发的流转不再通知摄影师（例如摄影师确认/完成时，摄影师不会收到「预约已确认」），避免自通知噪音。
- **隐藏约束**：`bookings.photographer_id` 是**摄影师档案 id**，不是 `users.id`；必须经 `GetPhotographerById(...).UserID`（`*int64`，可 NULL）解析后再 `notify`。

### Frontend

| Piece | File | Role |
|-------|------|------|
| Message page | `src/pages/message/index.vue` | 系统通知 section: `loadNotifications()` on `onShow` (alongside `loadSessions()`), `apiGet('/v1/notifications/:userId')` → map `{id,type,icon,title,content,read,time}`; not-logged-in/catch → `[]`; hardcoded demo array + demo banner block removed |
| Icon mapping | `notifyIcon(type)` in message page | success → `✓` (purple), warning → `⚠` (pink), info/default → `ℹ` (cyan) — geometric glyphs, no emoji |
| Time format | `formatNotifyTime(iso)` | `MM/DD HH:mm`, `''` fallback |

### Known gaps

- 广播通知（`user_id IS NULL`）无法标记已读 —— `MarkNotificationRead` 限定 `user_id = $2`，避免「一个用户把全员广播标记已读」的串扰；要支持需改为 per-user 已读表。个人通知（预约事件）点击后已读状态持久化。
- （已修 2026-09-14）通知端点不再信任客户端 `userId`：`POST /v1/notifications` 忽略 body 的 `userId`、写入 token 对应用户；`GET /v1/notifications/:userId` 与 token 不符 → 403。

## 双角色身份体系（P2a）

Every user can activate a photographer identity (闲鱼式双角色) — one-click activation + photographer orders panel. Implemented in the P2a dual-role plan (7 tasks, commits `9bbeaa9`/`4749b54`/`bad3685`/`39ba595`/`a0623b0`/`7bb99e0`), full-chain verified in Task 7.

### Data model

`server/migrations/000010_photographer_activation.up.sql` adds to `photographers`:

| Column | Type | Notes |
|--------|------|-------|
| `mode` | VARCHAR(10) default 'free' | 接单模式：`free` / `pay` / `both` |
| `mutual_intro` | TEXT | 互勉说明（activation intro） |
| `certified` | BOOLEAN default false | 认证摄影师 — 展示位，P2b 做申请设置 |
| `activated_at` | TIMESTAMPTZ | 开通时间 |

`photographers.user_id` (migration 000009) links the identity to a user; `login`/`me` responses carry `photographerId` when the user has activated, absent/null otherwise.

### Backend API

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/photographers/activate` | Auth required. `{name, mode, intro}` → 201 `{photographerId}`; **idempotent** — already-activated users get their existing id back (no field update) |
| `GET` | `/api/v1/photographers/by-user/:userId` | Auth required. Profile by user → 200, or 404 `not activated` |
| `GET` | `/api/v1/bookings/photographer/:photographerId` | Auth required。接单面板列表，含 coser join 字段（`coserName`/`coserAvatar`/`coserPhone`）。**`coserPhone` 按状态遮罩**（2026-09-24，B-06）：只有 `confirmed`/`completed` 才返回号码，`pending`/`cancelled` 返回空串 —— 摄影师确认前没有联系对方的正当必要，接单后要赴约必须联系得上。遮罩在 `bookings.sql.go` 的 SQL 里（`CASE WHEN b.status IN (...)`），**不是前端隐藏**，所以号码根本不出现在响应体里；前端据 `status` 显示「确认接单后可见」。**面议单报价阶段也不可见**（`price_status=quoted` 时 `status` 仍是 `pending`）—— 即「报价 ≠ 成交」，号码在 coser 接受报价、订单转 `confirmed` 的那一刻才出现（2026-09-24 真库实测）。改动这条 SQL 时别把 `CASE` 拿掉 |
| `PUT` | `/api/v1/bookings/:id/status` | Now takes `actorTag` (`coser` default / `photographer`). **Cross-actor guard**: `actorTag=photographer` requires the caller's own photographer id to equal the booking's `photographerId`, else **403 forbidden**; coser-tagged updates require `coserId == caller`. Actor check runs before transition validation |
| `POST` | `/api/v1/login`, `GET /api/v1/me` | Responses include `photographerId` |

### Frontend

| Piece | File | Role |
|-------|------|------|
| Activate page | `src/pages/photographer/activate.vue` | 开通摄影师 form (昵称 / 接单模式 互勉·收费·两者 / 简介); already-activated → 你已是摄影师 + 去接单管理; on success sets `userStore.user.photographerId` |
| Orders panel | `src/pages/photographer/orders.vue` | 接单管理: pending → 确认接单 / 拒绝, confirmed → 完成拍摄; sends `actorTag:'photographer'` |
| Profile badges | `src/pages/profile/index.vue` | Single role-tag: `photographerId` → 摄影师 (gold tint) else Coser; menu swaps 接单管理 ↔ 我是摄影师 |
| Detail mode/badge | `src/pages/photographer/detail.vue` | `.mode-tag` 互勉 / 收费 / 互勉 · 收费 + `.cert-badge` 认证摄影师 (certified only) |
| Coser order list | `src/pages/order/list.vue` | No 确认接单（演示） button — coser sees 取消预约 / 确认完成 / 联系摄影师 only (commit `7bb99e0`) |
| Router | `src/pages.json` | `pages/photographer/activate` + `pages/photographer/orders` registered |

### click-shot 借鉴说明

设计借鉴 click-shot（闲鱼式双角色），身份层与交易层分离：

- **身份层** — 单用户双身份：`photographers.user_id` 挂接用户；`me` 返回的 `photographerId` 决定前端的 badge、我是摄影师/接单管理 入口与接单面板可见性；`certified` 是身份层的信誉展示位。
- **交易层** — 摄影师接单面板（待确认 → 确认接单 → 完成拍摄）与 coser 订单列表共享同一 bookings 数据，用 `actorTag` + `actorUserID` 校验区分角色动作，未授权身份一律 403。

### Known gaps (Task 7 findings)

- `GET /photographers/by-user/:userId` omits `mode`/`certified` in its response (`GetByUser` builds `PhotographerItem` without them; `GET /photographers/:id` includes both). Follow-up: align GetByUser fields.
- `TestPhotographerHandler_Detail_Success` panics — its mock scan order predates the `user_id`/`mode`/`certified` columns added by P2a. Follow-up: update the mock to `GetPhotographerById` scan order.
- ~~UI login phone regex `^1[3-9]\d{9}$` blocks API test photographer account `10000000001`~~ **该说法已证伪（2026-09-24 实测）**：`src/pages/login/index.vue:102` 的 `canSend` 是 `^1[3-9]\d{9}$` **或** `^100\d{8}$`，`10000000001` 实际可正常 UI 登录；摄影师账号无需再靠 token 注入。另：登录按钮禁用时点击无反馈是另一个真缺陷（B-05，已修）。
- `certified` badge is display-only — application flow is P2b scope.

## 后台管理系统（P2c）

Soybean-admin 后台管理控制台（`admin/` 前端 + `server/` 管理 API），独立于 C 端（uniapp :5173 / :8080 C 端 API）运行。全链验证完成于 P2c Task 5（backend Task 1-3 + frontend Task 4 + Task 5 验证/推送，commits `9d24db5`/`ae68b5e`/`dbefacd`/`f0ae98e`/`4ef28de`/`ac9d464`）。

### 数据模型（admins 表）

`server/migrations/000012_admins.{up,down}.sql`:

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGSERIAL | PK |
| `username` | VARCHAR(64) UNIQUE | 登录名 |
| `password_hash` | VARCHAR(255) | bcrypt cost 10 |
| `role` | VARCHAR(20) default 'admin' | 角色（`admin` / `super` 预留，见「管理员管理」小节） |
| `token` / `token_expires_at` | VARCHAR/TIMESTAMPTZ | 管理端会话 token（与 C 端 `user_tokens` 完全隔离） |

- **种子账号 `admin` / `admin123`** — 仅用于本地演示；**生产部署前必须改密**（bcrypt 重新生成 `password_hash`）。
- **角色隔离** — `admins.token` 与 `user_tokens` 是两张表、两个 Bearer 中间件（`AdminAuthRequired` vs `AuthRequired`）：C 端 token 访问 admin 路由 401，反之亦然。
- 管理端 token 每次登录轮换（旧 token 即失效）。

### 管理 API（前缀 `/api/admin/v1`）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/login` | `{username,password}` → `{admin, token}`；bcrypt 校验，签发/轮换 token |
| `GET` | `/dashboard` | 指标：`totals`（users/photographers/certified/orders/pendingOrders/events）+ `trends`（7 天 date/newUsers/activeUsers/orders）+ `ordersByStatus` |
| `GET` | `/photographers` | 摄影师列表，`?certified=true|false` 可选过滤（空结果返回 null body，前端归一为 []） |
| `GET` | `/photographers/:id` | 摄影师详情（description/mutualIntro/userId + worksCount/reviewsCount 统计）；不存在 404 |
| `PUT` | `/photographers/:id/certified` | `{certified: bool}` → `{ok}` 认证/取消认证 |
| `GET` | `/orders` | 分页订单列表，`?status=` + `?page=` + `?pageSize=`；含摄影师/用户 join 字段 |
| `PUT` | `/orders/:id/status` | 状态机流转（同 C 端规则，`completed→pending` 等非法流转 409；管理员跳过身份校验但保留状态机） |
| `GET` | `/events` | 全部漫展（含 delFlag），前端客户端排序/过滤 |
| `PUT` | `/events/:id/status` | `{delFlag: bool}` → `{ok}` 上架/下架；**不存在 id → 404**（原误返 200） |
| `GET` | `/export/orders` | CSV 导出（UTF-8 BOM；`?status=` 可选） |
| `GET` | `/export/users` | CSV 导出（`?keyword=&role=&status=` 可选） |
| `GET` | `/export/photographers` | CSV 导出（`?certified=` 可选） |
| `GET` | `/audit-logs` | 操作日志（分页 `?page=&pageSize=`）→ `{list,total}`；由中间件 `AdminAuditLogger` 自动记录 **所有 admin 非 GET 请求（含 4xx 失败）**，只记 method/path/status/admin，**不记 body**（避免泄漏密码） |

登录除外，全部端点需要 `Authorization: Bearer <admin token>`。

### Dashboard 指标说明

- 6 统计卡：用户总量 / 今日新增用户 / 日活用户 / 摄影师（副题=已认证数）/ 订单总量（副题=待确认数）/ 漫展数量（仅统计未下架 delFlag=false）。
- 3 个 ECharts：用户增长趋势（折线，7 天 new/active users）、订单状态分布（饼）、每日订单量（柱）。
- 后端 `internal/repository/admin_stats.sql.go` 聚合；实测基线：users=16、photographers=12、orders=10、events≈49 活跃。

### admin/ 目录（soybean-admin 改造）

| Piece | Location | Role |
|-------|----------|------|
| 视图 | `admin/src/views/{dashboard,photographer,order,event,user,certification}/index.vue` | 工作台 / 摄影师审核 / 订单管理 / 漫展管理 / 用户管理 / 认证审核 |
| API 层 | `admin/src/service/api/admin.ts` + `request/admin.ts` | `VITE_ADMIN_API_BASE_URL=http://localhost:8080/api/admin/v1` |
| 登录 | `admin/src/views/_builtin/login/index.vue` | 简化版 admin 登录（显示种子账号提示） |
| 路由 | `admin/src/router/routes/` | elegant-router 自动路由：`/dashboard` `/photographer` `/order` `/event`；`VITE_ROUTE_HOME=dashboard` |
| 认证 store | `admin/src/store/modules/auth/` | token 存 localStorage（`SOY_token`，JSON 字符串格式） |

- **模板**：soybean-admin v4+（Vue3 + Vite8 + TS + Pinia + UnoCSS + NaiveUI），pnpm monorepo；原模板的示例页面/接口已替换为真实管理端。
- **开发端口 5174**（`vite.config.ts` 默认 5173 与 C 端 H5 冲突，以 `--port 5174` 启动）；**5173 是 C 端 H5，禁止占用/修改**。
- **i18n**：保留 i18n 框架但仅启用 zh-CN（`fallbackLocale: 'zh-CN'`），无语言切换器。
- **P2b 认证申请流已落地（admin P0）** — 摄影师自助提交 + 管理端审批见下节「账号管理 + 认证申请审核」；`PUT /photographers/:id/certified` 手动切换保留为人工调整手段。

## 账号管理 + 认证申请审核（admin P0）

用户管理（禁用/启用 + 详情统计）与摄影师认证申请审核（C 端自助提交 → 管理端审批 → 黄V 生效 + 通知），实现于 admin P0 计划（4 个任务，commits `d2e5fce`/`c10e554`/`8d66efb`/`2d6afcc` + Task 4 验证/推送）。全链验证（后端测试 + curl E2E + Playwright 浏览器 E2E + agent-browser 视觉复查）完成于 Task 4。

### 数据模型

| Migration | Table/Column | Notes |
|-----------|--------------|-------|
| `server/migrations/000013_users_status.{up,down}.sql` | `users.status` VARCHAR(10) | `active` / `disabled`，DEFAULT 'active'（历史用户全部 active，无需回填） |
| `server/migrations/000014_cert_applications.{up,down}.sql` | `photographer_cert_applications` | `user_id`/`photographer_id` FK、`evidence_images TEXT[]`、`evidence_desc`、`status`（pending/approved/rejected）、`review_reason`、`admin_id` FK admins、`reviewed_at`；索引 user/status；**部分唯一索引** `(photographer_id) WHERE status <> 'rejected'` |

- **一摄影师一活跃申请**：部分唯一索引是硬保证；并发重提映射 23505 → 409「already applied」，不用 Go 层先查后插的竞态方案。`rejected` 后可重新申请。
- 登录禁用守卫：`auth_service.Login` 在签发 token 前检查 `users.status`，disabled → 403 `{"error":"account disabled"}`。

### API（管理端前缀 `/api/admin/v1`，均需 admin token）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/users` | 用户列表，`?keyword=`（昵称/手机号模糊）+ `?role=` + `?status=` + `?page=` + `?pageSize=` |
| `GET` | `/users/:id` | 详情 + `stats`（bookingsCount/reviewsCount/favoritesCount/followsCount）+ `recentBookings`（含摄影师/服务 join） |
| `PUT` | `/users/:id/status` | `{status:"active"\|"disabled"}` → `{ok}` |
| `GET` | `/cert-applications` | 申请列表，`?status=pending\|approved\|rejected` + 分页 |
| `GET` | `/cert-applications/:id` | 申请详情（含证据图片、驳回理由、审核时间） |
| `PUT` | `/cert-applications/:id/review` | `{action:"approve"\|"reject", reason?}`；**reject 必须填 reason**；非 pending 复审 409 |

C 端新增：`POST /api/v1/photographers/cert-apply`（AuthRequired）`{evidenceImages[], evidenceDesc}` → 201 `{id}`；非摄影师 404、已有活跃申请 409。

### 审批事务说明

- **approve**：单个原子 repo 事务 `ReviewApproveTx`（申请标记 approved + `photographers.certified=true` + reviewed_at/admin_id），随后 best-effort 成功通知「认证通过（黄V 已生效）」。
- **reject**：单写（status=rejected + review_reason + admin_id）+ warning 通知「认证未通过」。
- 通知写入失败仅记日志、不阻塞审批结果（与 `booking_service.notify` 同模式）；通知**仅单用户**（复用 `notification_service.Create`），全员广播是 P1。
- 并发复审竞态：条件更新冲突 → 重读 → 映射 409「already reviewed」（避免误导性 404）。

### Frontend（admin/，浅色专业风）

| Piece | Location | Role |
|-------|----------|------|
| 用户管理 | `admin/src/views/user/index.vue` | 列表（keyword/role/status 筛选）+ 详情抽屉（4 统计卡 + 最近订单表）+ 禁用/启用（确认对话框） |
| 认证审核 | `admin/src/views/certification/index.vue` | 三 tab（待审核/已通过/已驳回）+ 详情抽屉（证据图片 NImageGroup + 驳回理由）+ 通过确认 + 驳回弹窗（理由必填，200 字） |
| API 层 | `admin/src/service/api/admin.ts` | `fetchAdminUsers` / `fetchAdminUserDetail` / `setUserStatus` / `fetchCertApplications` / `fetchCertApplicationDetail` / `reviewCertApplication` |
| 路由 | elegant-router auto | `/user`、`/certification` |

- **单根模板约束**：路由页必须单个根 `<div>`（`vite-plugin-vue-transition-root-validator` 会拦截多根）。
- C 端 `photographer/detail.vue` 显示 `.cert-badge` 认证摄影师（`certified` 字段）；**C 端尚无申请提交 UI**（cert-apply 仅 API，申请表单是 P2b 打磨范围）。

### 演示数据状态（Task 4 E2E 后）

- `photographers.certified=TRUE`：光影行者(1)、樱花落(2)、古风公子(4)、**暗夜骑士(3)**（Task 4 E2E 审批通过——演示可见黄V，属可接受演示数据）；测试摄影师(5) certified=false（申请 #5 驳回，可重新申请）。
- `photographer_cert_applications`：#1 approved（樱花落）、#2 rejected（暗夜骑士，后重提 #4）、#3 approved（古风公子）、#4 approved（暗夜骑士）、#5 rejected（测试摄影师，reason=样片数量不足）。
- user 142（13000000001，coser）E2E 中 禁用→403→启用 已恢复 active。
- **admin token 每次登录轮换**（curl 登录会使浏览器会话 401 跳登录；E2E 保持单一 admin 会话）。
- 测试账号：admin/admin123（本地演示，生产必须改密）；C 端摄影师 10000000001~04（验证码 `1234` 前缀即可，如 `123456`）。

### 后续（不在本次）

- **P1**：通知公告 / 内容管理 / 报表导出；审批通知全员广播。（管理员管理已落地，见「管理员管理」小节。）
- **P2**：操作日志 / 摄影师详情；C 端认证申请表单 UI（提交入口 + 状态展示）。

## 管理员管理（P2c Admin Accounts）

管理端管理员账号管理（列表/新建/禁用启用/重置密码），实现于 admin admin-management 计划（commits `307a195` backend + `49c1a41` frontend，Task 3 全链验证：后端 build/vet/test + curl E2E + Playwright 浏览器 E2E + agent-browser 视觉复查）。

### 数据模型

`server/migrations/000015_admins_status.{up,down}.sql` 给 `admins` 表加：

| Column | Type | Notes |
|--------|------|-------|
| `status` | VARCHAR(10) default 'active' | `active` / `disabled`，历史行默认 active 无需回填 |

- **种子保护**：`service.SetStatus` 拒绝禁用 id=1（主管理员）→ 400「cannot disable primary admin」；重置密码允许（生产改密必需）。若未来引入 super 角色，种子保护由 role=super 取代。
- **角色预留**：Create 校验 role 仅 `admin` / `super`（非法值 400「invalid role」）；当前 UI 只创建 admin，super 区分是后续。
- **disabled 即时失效**：`GetAdminByToken` 查询带 `AND status='active'` 过滤——禁用后已签发 token 立即 401；`Login` 签发前同样校验 status，禁用账号登录 401「invalid credentials」（与密码错误同响应，防枚举）。

### 管理 API（前缀 `/api/admin/v1`，均需 admin token）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/admins` | 全量列表；DTO 剥离 passwordHash（仅 id/username/role/status/createdAt） |
| `POST` | `/admins` | `{username,password,role}` → 201 `{id}`；重名 409「username exists」 |
| `PUT` | `/admins/:id/status` | `{status:"active"\|"disabled"}` → 200 `{ok}`；id=1 禁用 400「cannot disable primary admin」；不存在 404「admin not found」 |
| `PUT` | `/admins/:id/password` | `{password}` → 200 `{ok}`；重置后清 token——该账号所有已登录会话立即 401 强制登出 |

- 无 DELETE 端点（禁用即可，YAGNI）；测试清理走 SQL。
- 后端文件：`server/internal/handler/admin_admin_handler.go` / `service/admin_service.go` / `repository/admin_repo.go`（AdminService 已含登录/轮换逻辑，本次扩展 List/Create/SetStatus/ResetPassword）。

### Frontend（admin/，浅色专业风）

| Piece | Location | Role |
|-------|----------|------|
| 视图 | `admin/src/views/admin-accounts/index.vue` | 列表（ID/用户名/角色/状态/创建时间/操作）+ 新建管理员 modal（用户名/密码/角色 NSelect）+ 禁用/启用（确认对话框）+ 重置密码 modal |
| API 层 | `admin/src/service/api/admin.ts` | `fetchAdmins` / `createAdmin` / `setAdminStatus` / `resetAdminPassword` |
| 路由 | elegant-router auto | `/admin-accounts`（locale zh-cn「管理员管理」，侧边栏菜单） |

- 主管理员（id=1）active 时「禁用」按钮 disabled + hover tooltip「主管理员不可禁用」；「重置密码」始终可用。
- 角色 tag：admin=管理员（info 紫）、super=超级管理员（warning 黄）；状态 tag：active=启用（success 绿）、disabled=禁用（error 红）。
- 重置密码成功提示「密码已重置，该账号的登录状态已失效」。
- 目录名 `admin-accounts`（非 `admin`）避免 elegant-router 与既有 admin 语义冲突。

### 验证结论（Task 3）

- 后端 `go build`/`go vet`/`go test ./...` 全绿（handler/service 测试通过）。
- curl E2E 13 步全过：创建 testadmin → 禁用后登录 401 → 启用后登录恢复 → 重置密码后旧 token 401「invalid admin token」→ 新密码登录 200 → 主管理员禁用 400 → /admins/999 404 → SQL 清理。
- Playwright 浏览器 E2E（:5174 单 admin 会话）14 项断言：13 过 + 1 条预置模板告警（`VITE_OTHER_SERVICE_BASE_URL is not a valid json5 string`，soybean 模板启动告警，与本模块无关）。
- agent-browser 视觉复查：浅色专业风、表格/标签/按钮禁用态/中文渲染正常，无渲染缺陷（截图 `/tmp/opencode/admin-admin-page.png`）。
- **演示数据收尾**：测试后 `testadmin` 已 SQL 删除，`admins` 仅剩种子 admin（id=1, role=admin, status=active）。

### 后续（不在本次）

- **super 角色区分**：权限/菜单级差异（当前仅 role 值预留）。
- **操作日志**：管理员操作（禁用/重置密码）审计记录。
- **2FA**：管理端登录二次验证。

## 轮播图管理（P2c Banner）

管理端轮播图管理（列表/新增/编辑/上下线），实现于 P2c banner 计划（commits `0075b99` backend + `c2b3dc6` frontend，Task 3 全链验证）。

### 数据模型

`banners` 表**已存在（0 迁移）**：`id / image_url / title / link_type / link_id / sort_order / is_active / created_at / updated_at`。演示数据 3 条：ChinaJoy 2026(sort 0)、第40届萤火虫漫展(sort 1)、CP33 综合同人展(sort 2)，均 isActive=true。

### 管理 API（前缀 `/api/admin/v1`，均需 admin token）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/banners` | 全量列表，`ORDER BY sort_order ASC`（无分页，banners 少量） |
| `POST` | `/banners` | `{imageUrl,title,linkType,linkId,sortOrder}` → 201 `{id}`；title/imageUrl/linkType 必填 |
| `PUT` | `/banners/:id` | 全字段更新（含 `isActive`）→ 200 `{ok}`；404 not found |
| `PUT` | `/banners/:id/status` | `{isActive: bool}` 上下线 → 200 `{ok}` |

- `link_type` 仅支持 `event`（link_id 指向 comic_events.id）；其他值 → 400 invalid link type。
- 后端文件：`server/internal/repository/admin_banner_repo.go`（独立 pool 模式）/ `service/admin_banner_service.go` / `handler/admin_banner_handler.go`；路由注册在 `cmd/api/main.go` adminGroup。

### 前端（admin/）

| Piece | Location | Role |
|-------|----------|------|
| 视图 | `admin/src/views/banner/index.vue` | 轮播图管理：NDataTable（缩略图 NImage + 标题 + 类型标签 + 关联ID + 排序 + NSwitch 上线开关 + 编辑）+ 新增/编辑 NDrawer 表单 |
| API 层 | `admin/src/service/api/admin.ts` | `fetchBanners` / `createBanner` / `updateBanner` / `setBannerStatus` |
| 路由 | elegant-router auto | `/banner`（locale zh-cn「轮播图管理」） |

### C 端（打通后，commit `68796f7`）

C 端首页轮播已改为读取 banners 表——管理端维护的轮播图**实时驱动** C 端首页 carousel：

- `home_service.GetHomeData`（及未被引用的 `GetHomeDataParallel` 变体）通过 `mapBanners(data.Banners)` 映射首页 `banners` 字段；`mapBanners` 将 `repository.Banner` 映射为 `BannerItem`（`Title`/`LinkType` 经 `derefString` nil 安全处理，`LinkID` 为 `*int32`）。
- 数据源为 `home_repo.GetHomeData` 内的 `queries.GetActiveBanners(ctx, 5)`：`WHERE is_active=true ORDER BY sort_order ASC LIMIT 5`。
- 原 `eventsToBanners()` 事件派生函数**已删除**（commit `68796f7`，server 内 grep = 0）。

行为转变：轮播图内容从「由近期漫展自动派生（按天去重、最多 3 张）」改为「管理端经 banners 表控制（最多 5 张，按 sort_order 排序）」——C 端管理端 create/edit/上下线即改首页轮播。注意轮播内容现由管理端负责维护（首页「近期漫展」区块仍由 upcoming events 驱动，两者不再自动一致）。当前 banners 演示数据：ChinaJoy 2026(sort 0)、第40届萤火虫漫展(sort 1)、CP33 综合同人展(sort 2)，均 isActive=true。

### 后续（不在本次）

- 图片上传（当前仅 imageUrl 文本框）；排期（生效时间窗口）；link_type 扩展（页面/外链）。
- 无 DELETE 端点——下线即停用；硬删走 SQL。

## 通知公告管理（P2c Notification Broadcast）

管理端通知公告广播（全员 + 指定用户 + 历史），实现于 P2c admin notification 计划（commits `d228821` backend + `1d0bc2c` frontend，Task 3 全链验证：后端 build/vet/test + curl E2E + Playwright 双端 E2E + agent-browser 视觉复查）。

### 数据模型

`server/migrations/000016_notification_broadcast.{up,down}.sql`：

- **up**：`ALTER TABLE notifications ALTER COLUMN user_id DROP NOT NULL` —— 广播行 user_id=NULL 复用 notifications 表，**不建新表**。
- **down**：先 `DELETE FROM notifications WHERE user_id IS NULL`（否则 SET NOT NULL 失败），再恢复 NOT NULL。

### 管理 API（前缀 `/api/admin/v1`，均需 admin token）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/notifications` | 发布 `{type,title,content,targetType,userId?}` → 201 `{id}`；type 仅 success/info/warning；targetType=all 写广播行（user_id NULL），targetType=single 需 userId 存在（否则 404）后写单用户行 |
| `GET` | `/notifications` | 历史 `?page=&pageSize=` → `{list:[{id,type,title,content,userId,targetType,createdAt}], total}`；targetType 由 userId 推导（NULL → `all`，非空 → `single`） |
| `DELETE` | `/notifications/:id` | **撤回**（2026-09-15）：删除该通知行（广播/单用户皆可）→ 200 `{ok}`；不存在 → 404 |

- 后端文件：`server/internal/service/notification_admin_service.go`（`NotificationAdminService.Publish/ListHistory/Delete`）/ `handler/admin_notification_handler.go`；`repository.DeleteNotification`（RowsAffected=0 → `pgx.ErrNoRows` → `ErrNotificationNotFound` → 404）；路由注册在 `cmd/api/main.go` adminGroup。
- admin UI：通知历史表新增「操作」列（`撤回` 按钮 + `window.$dialog` 确认 → `deleteNotification(id)` → 刷新）。

### C 端（唯一 C 端后端改动，前端零改动）

- `repository.ListNotifications` SQL 改为 `WHERE (user_id = $1 OR user_id IS NULL)` —— **广播行对所有用户可见**；`NotificationRow.UserID *int64`（nil 安全，broadcast 行扫描不报错）。
- P3 消息页前端无改动：仍调 `GET /api/v1/notifications/:userId`，mapping 只用 id/type/title/content/read/time，不读 userId 字段。

### 前端（admin/，浅色专业风）

| Piece | Location | Role |
|-------|----------|------|
| 视图 | `admin/src/views/notification/index.vue` | 发布通知 NCard（类型/标题/内容/目标 全员·指定 + 用户 NSelect）+ 通知历史 NDataTable（目标列 全员/指定用户 #id 标签） |
| API 层 | `admin/src/service/api/admin.ts` | `fetchNotifications` / `publishNotification` |
| 路由 | elegant-router auto | `/notification`（locale zh-cn「通知公告」） |

### 后续（不在本次）

- 撤回广播、定时发布、富文本内容、已读率统计。
- 广播行无 user_id——管理端暂无 DELETE 端点，清理走 SQL。

## 内容管理（P2c Content）

管理端内容管理（作品上/下架、评论列表/删除、标签 CRUD），实现于 admin content 计划（commits `52ab3e9` backend + `8ef65ca` frontend，Task 3 全链验证：后端 build/vet/test + curl E2E + Playwright 双端 E2E + agent-browser 视觉复查）。

### 数据模型

`server/migrations/000017_works_status.{up,down}.sql` 给 `works` 表加 `status` VARCHAR(10) DEFAULT 'active' NOT NULL（`active` / `down`）——新作品自动上架（「先发后审」现状），**无 DROP NOT NULL 的 soft-delete 语义**；历史作品回填默认 active。

### 管理 API（前缀 `/api/admin/v1`，均需 admin token）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/works` | 作品分页列表 `?photographerId=&status=&page=&pageSize=` → `{list, total}` |
| `PUT` | `/works/:id/status` | `{status:"active"\|"down"}` → 200 `{ok}`；非法状态 400；**同时失效 home 缓存**（下架立即从首页 featured 消失，不等 5min TTL） |
| `GET` | `/reviews` | 评论分页 `?keyword=&photographerId=` → `{list, total}`；keyword 模糊匹配内容/用户名/摄影师名 |
| `DELETE` | `/reviews/:id` | **硬删**（无软删/恢复）→ 200 `{ok}`；不存在 404 |
| `GET` | `/tags` | 全量标签列表（bare array），`usageCount` 为 photographer_tags 关联数的**真实子查询统计** |
| `POST` | `/tags` | `{name}` → 201 `{id}`；重名 409 |
| `PUT` | `/tags/:id` | `{name}` → 200 `{ok}`；重名 409、不存在 404 |
| `DELETE` | `/tags/:id` | 删除 → 200 `{ok}`；**引用检查**：photographer_tags count>0 → 400 `tag in use`；不存在 404 |
| `POST` | `/tags/merge` | **合并标签**（2026-09-15）：`{fromId,toId}` → 200 `{ok}`；单事务：`photographer_tags` 中 `fromId` 的引用复制到 `toId`（`ON CONFLICT DO NOTHING`）→ **删除 `fromId` 旧引用** → 删除源标签；`fromId==toId` → 400 `cannot merge a tag into itself`；任一标签不存在 → 404 |

- admin UI：标签 tab 操作列新增「合并」按钮 → 合并模态（源标签只读 + 目标 `NSelect filterable`，选项为其余标签）。

- 后端文件：`server/internal/repository/admin_content_repo.go` / `service/admin_content_service.go` / `handler/admin_content_handler.go`；路由注册在 `cmd/api/main.go` adminGroup。

### C 端过滤（下架全链路不可见）

| 位置 | 改动 |
|------|------|
| `repository.GetWorksByPhotographer` | SQL 加 `AND status='active'` —— 摄影师详情页 works 过滤下架 |
| `repository.GetFeaturedWorks`（home featured） | 同样加 `AND status='active'` —— 首页精选过滤下架 |
| 缓存一致性 | 管理端 `SetWorkStatus` 成功后 `cache.Delete(ctx, "home")` —— 下架/恢复即时反映到 C 端首页 |

- 演示数据：works 4 条（id 1 原神-雷电将军 / 2 鬼灭之刃 / 3 魔卡少女樱 / 4 古风仙侠，均 active）；reviews 3 条（小狐狸 5★ / 月华 5★ / 用户5213 1★，均光影行者）；tags 13 个 usageCount 真实 0-2。

### 前端（admin/，浅色专业风）

| Piece | Location | Role |
|-------|----------|------|
| 视图 | `admin/src/views/content/index.vue` | 内容管理：NTabs 作品/评论/标签；作品 tab NImage 缩略图（`images[0]` + fallbackSrc）+ 状态 NTag（上架绿/下架红）+ 下架|恢复（NDialog 确认）+ 状态筛选 + remote 分页；评论 tab NAvatar + 关键字搜索（placeholder 搜索用户名/内容）+ 删除（NDialog 二次确认，文案注明「删除后不可恢复」）；标签 tab ID/名称/真实使用量 + 新增/编辑 NModal + 删除确认 |
| API 层 | `admin/src/service/api/admin.ts` | `fetchAdminWorks` / `fetchReviews` / `fetchTags` / `setWorkStatus` / `deleteReview` / `createTag` / `updateTag` / `deleteTag`（空列表归一 `[]`） |
| 类型 | `admin/src/typings/api/admin.d.ts` | `WorkStatus` / `AdminWork` / `AdminReview` / `AdminTag` 等 |
| 路由/图标 | elegant-router auto + `admin/build/plugins/router.ts` | `/content`，icon `mdi:book-open-page-variant-outline`，locale zh-cn「内容管理」 |

### 后续（不在本次）

- 审核流（作品先发后审 → 管理员审核后上架）；作品软删（当前仅上下架，无恢复手段的是评论硬删——YAGNI 决定）。
- 作品编辑（当前管理端只能改 status，不能改标题/图片/摄影师）。
- 标签合并（重复标签迁移引用再删）；评论管理扩展（回复/隐藏/申诉）。
- works 无 cover 列（orig = images TEXT[]）——缩略图约定取 `images[0]`。

## C 端认证申请（P2b C 端补全）

C 端摄影师自助认证申请（提交 → 审核中 → 已通过/已驳回 → 重新申请），实现于 cend cert-apply 计划（backend `28ec620` + frontend `0c57b1a`，Task 3 全链验证：后端 build/vet/test + curl API + Playwright 浏览器 E2E + agent-browser 暗色视觉复查）。管理端审批流见「账号管理 + 认证申请审核（admin P0）」小节。

### 后端（仅 GET 新增，POST 为 admin P0 已建）

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/api/v1/photographers/cert-application` | Auth required。当前用户**最新一条**申请的精简 DTO → 200 `{application:{id,status,reviewReason,createdAt}}`；无申请记录 → 200 `{application:null}`；非摄影师 → 403 `not a photographer` |
| `GET` | `/api/v1/photographers/cert-applications` | Auth required。当前用户**全部**申请历史（新→旧）精简 DTO → 200 `{list:[{id,status,reviewReason,createdAt}]}`（空为 `[]`）；非摄影师 → 403 |

- 精简 DTO 原则：**不暴露** userId/photographerId/evidenceImages/evidenceDesc 到 C 端（最小暴露；repository 层 json tags 完整但 handler 构造精简 gin.H）。
- `POST /api/v1/photographers/cert-apply`（admin P0 已建）：非摄影师 404、已有活跃申请 409「already applied」（部分唯一索引硬保证）；`rejected` 后可重新申请（新行）。
- 后端文件：`server/internal/handler/cert_application_handler.go`（`Submit`/`MyApplication`）+ `service.CertApplicationService.GetMyApplication`；路由注册在 `cmd/api/main.go:235`。

### 前端（C 端，暗色霓虹）

| Piece | File | Role |
|-------|------|------|
| 激活页入口 | `src/pages/photographer/activate.vue` | `.btn-cert` 按钮：onShow 调 `apiGet('/v1/photographers/:id')` 读 `certified`——true 显示「✓ 已认证摄影师」、false 显示「申请认证」→ `navigateTo('/pages/photographer/cert-apply')`；读取失败降级显示「申请认证」（不碰后端，复用既有详情 API） |
| 认证申请页 | `src/pages/photographer/cert-apply.vue` | 4 状态 + 表单：`pending`（◇ 审核中+提交时间）/`approved`（★ 已通过·黄V已生效）/`rejected`（⚠ 已驳回+驳回理由+重新申请按钮）/`not-applied` 或 reapplying → 表单（样片链接 1-5 个 + 作品说明 ≤300 字）→ 提交成功 toast「已提交，等待审核」并回显「审核中」卡；403 → toast「仅摄影师可申请认证」并返回 |
| API 层 | `src/api/index.ts` | `getMyCertApplication()` → `apiGet('/v1/photographers/cert-application')`；`applyCertification(evidenceImages[], evidenceDesc)` → `apiPost('/v1/photographers/cert-apply')`；`interface CertApplication {id,status,reviewReason,createdAt}` |
| 路由 | `src/pages.json` | `pages/photographer/cert-apply`（custom nav） |

- **暗色约束**：cert-apply 页严格用 `$dark-*`/`$neon-*` 变量（#0a0a1a 底、状态卡 $dark-bg-card、驳回红 `$error-color`、CTA `$neon-gradient` 紫→青）——不引入 admin 浅色风格。
- 状态流转：rejected → 重新申请 → 表单提交 → pending；approve/reject 由管理端「认证审核」操作（admin P0 小节），C 端只读状态。

### 演示数据状态（Task 3 E2E 后）

- `photographer_cert_applications`：#1 approved(樱花落2)、#2 rejected(暗夜骑士3)、#3 approved(古风公子4)、#4 approved(暗夜骑士3)、#5 rejected(测试摄影师5)、#6 rejected(测试摄影师5, Task 2 测试恢复)、#7 rejected(测试摄影师5, Task 3 E2E 提交后恢复)。
- `photographers.certified`：光影行者/樱花落/暗夜骑士/古风公子 = true；测试摄影师 = false（E2E 全程未改 certified）。
- 测试摄影师（13700000005，photographer_id=5；**不是** coser 的 13800138000）处于 rejected → 可直接演示「已驳回 + 重新申请」完整流程。2026-09-24 实测 DB：测试摄影师名下仅剩 #5 rejected（#6/#7 已在 E2E 收尾时清理）。

### 后续（不在本次）

- 图片上传（当前仅样片 URL 文本框）；多历史申请展示（当前仅返回最新一条）；作品上传。
- C 端 approved 摄影师主页黄V 徽章已有（detail.vue `.cert-badge`）；申请表单提交入口 UI 打磨。

## 摄影师作品管理（C 端）

摄影师自助发布/删除作品（作品集管理），实现于 cend photographer-works 计划（backend `3c57880` + frontend `52a4888` + DTO/过滤修正 `adb42dd`/`60d03f6`）。后端独立 handler + `worksStore` 测试 seam，无新迁移——复用既有 `works` 表（status 列由 内容管理 000017 引入）。

### 后端 API

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/photographers/works` | Auth required。`{title 必填, images 必填且 ≥1, description?}` → 201 `{id}`；绑定失败（缺 title/images）400「title and at least one image are required」；非摄影师 → 403「only photographers can publish works」；InsertWork 不写 status → DB default `'active'` 自动上架 |
| `GET` | `/api/v1/photographers/works/mine` | Auth required。自己的作品 → 200 bare array camelCase `WorkItem[]` `{id,title,images,description,status,createdAt(RFC3339)}`；非摄影师 → 403「not a photographer」；**unfiltered**——走 `GetAllWorksByPhotographer`，下架作品（status:'down'）自己仍可见 |
| `DELETE` | `/api/v1/photographers/works/:id` | Auth required。硬删 → 200 `{ok:true}`；**归属校验**：`works.photographer_id != 调用者摄影师 id` → 403「cannot delete another photographer's work」；不存在 → 404「work not found」；非摄影师 → 403「not a photographer」 |
| `PUT` | `/api/v1/photographers/works/:id` | Auth required。`{title 必填, images 必填且 ≥1, description?}` 全量更新 → 200 `{ok:true}`；**scoped UPDATE**（`WHERE id AND photographer_id`，消除 TOCTOU）；归属不符 403「cannot edit another photographer's work」；不存在 404；非摄影师 403 |

- **查询区分**：`AllWorksByPhotographer`（works.sql.go，`WHERE photographer_id=$1` 无 status 过滤）仅用于本人管理列表；公开路径 `GetWorksByPhotographer`（摄影师详情 works）/ `GetFeaturedWorks`（首页精选）仍 `AND status='active'`——下架作品 C 端公共不可见、管理列表可见可删。
- 后端文件：`server/internal/handler/photographer_work_handler.go`（`CreateWork`/`MyWorks`/`DeleteWork`）+ `service/photographer_service.go`（`PhotographerService.CreateWork/MyWorks/DeleteWork`，`worksStore` 接口注入 fake 供测试）+ `repository/works.sql.go`（`GetAllWorksByPhotographer`/`GetWorkPhotographerID`/`DeleteWorkByID`/`InsertWork`）；路由注册 `cmd/api/main.go:137-139`。
- 测试：`photographer_service_test.go` `TestCreateWork_*` / `TestDeleteWork_*`（NotOwner→ErrWorkForbidden、NotFound）/ `TestMyWorks_IncludesDowned` / `TestMyWorks_MapsCamelCaseDTO`（断言 status + RFC3339 `createdAt` camelCase）/ `TestMyWorks_Empty`。

### 前端（C 端，暗色霓虹）

| Piece | File | Role |
|-------|------|------|
| 作品管理页 | `src/pages/photographer/works.vue` | 暗色。list 模式：工具栏「共 N 件 + ＋ 新增作品」、卡片（缩略图 `images[0]` + 标题 + 上架/下架 tag（`status||'active'`）+ 说明 + `formatTime(created_at\|\|createdAt)` + 图片数）+ 删除（uni.showModal 确认「删除后不可恢复」）；form 模式：标题 ≤50 + 图片 URL 输入 1-5（`images.length<5` 才显示 + 添加图片）+ 说明 ≤300；403 → toast「仅摄影师可管理作品」并返回；空态「暂无作品」；后端空 slice 序列化为 null → 前端归一 `[]` |
| 激活页入口 | `src/pages/photographer/activate.vue` | 已激活卡 `.btn-works` 作品管理 → `navigateTo('/pages/photographer/works')` |
| API 层 | `src/api/index.ts` | `uploadWork({title,images,description?})` → apiPost `/v1/photographers/works`；`getMyWorks()` → apiGet `/v1/photographers/works/mine`；`deleteWork(id)` → apiDelete；`interface MyWork {id,title,images,description?,status?,created_at?,createdAt?}`（字段可选 = 兼容前端双读法；后端实际返回 status+camelCase createdAt） |
| 路由 | `src/pages.json` | `pages/photographer/works`（在 activate/cert-apply 之后注册） |

### Notes

- **图片上传已接入**——作品图支持「从相册上传」（`POST /api/v1/upload` → `/static/uploads/...`），同时保留手填外链。
- 本人可见下架作品（管理列表 unfiltered）；公共可见性由 `GetWorksByPhotographer`/`GetFeaturedWorks` 的 `status='active'` 过滤保证，管理端下架（内容管理）即时影响本人列表 tag 与公共首页。

### 后续（不在本次）

- 封面缩略图上传；作品列表分页（当前全量拉取）。

## 摄影师主页管理（C 端）

摄影师自助编辑主页资料（昵称/简介/城市/接单模式/互勉说明），实现于 cend photographer-profile 计划（backend `8b07732` + `2d6f79d` + frontend `f4aaa4d`）。无新迁移——复用 `photographers` 既有列（description/location/mode/mutual_intro 均 P2a 及更早已有）。

### 后端 API

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `PUT` | `/api/v1/photographers/profile` | Auth required。**全量更新** `{name,description,location,mode,mutualIntro,avatar?}` → 200 `{ok:true}`；`avatar` 为空字符串则**保留原头像**；`mode` 非法（∉ {free,pay,both}）→ 400「invalid mode」；非摄影师 → 403「not a photographer」；UPDATE 影响 0 行 → 404「photographer not found」。SQL 同时 `updated_at=NOW()` |
| `GET` | `/api/v1/photographers/profile/mine` | Auth required。自己主页 → 200 完整 `PhotographerItem`（含 `description`/`mode`/`mutualIntro`/`certified` + id/name/avatar/location/rating/reviewCount/orderCount/userId/tags）；非摄影师 → 403「not a photographer」 |

- **P2a 补齐**（fix `2d6f79d`）：共享 DTO `PhotographerItem`（home_service.go）补 `description` 字段（原仅 `PhotographerDetail` 顶层有）；`GetByUser`（`GET /photographers/by-user/:userId`，activate 页读 certified 用）经 `photographerItemFromRow` 现在返回 `mode`/`mutualIntro`/`certified`——**修复 P2a Known gap**（原 GetByUser 漏 mode/certified）。`GetPhotographerByUserID` SELECT 已含 `mutual_intro`。
- 后端文件：`server/internal/handler/photographer_profile_handler.go`（`PhotographerHandler.UpdateProfile/MyProfile`）+ `service/photographer_service.go`（`ProfileUpdate` 结构体、`UpdateProfile`（先查身份再验 mode 再 UPDATE）、`MyProfile`）+ `repository/photographers.sql.go`（`GetPhotographerByUserID`/`UpdatePhotographerProfile`/`ErrProfileNotFound`）；路由注册 `cmd/api/main.go:132-133`。
- 测试：`photographer_service_test.go` `TestUpdateProfile_NotPhotographer/InvalidMode/OK`、`TestMyProfile_IncludesModeCertified`（by-user 补齐回归）/`TestMyProfile_NotPhotographer`。

### 前端（C 端，暗色霓虹）

| Piece | File | Role |
|-------|------|------|
| 主页管理页 | `src/pages/photographer/profile-edit.vue` | 暗色表单：onShow 调 `getMyPhotographerProfile()` 预填（fetch 结果写回 form）；字段 头像（预览 + 从相册上传 + 链接输入，`onerror` 回退）/ 昵称 ≤20（必填，客户端校验「请填写昵称」）/ 简介 textarea ≤200 / 城市（预设 30 城 `picker mode=selector`）/ 接单模式 3 格 segmented（互勉 free / 收费 pay / 两者 both + 动态 hint）/ 互勉说明 textarea ≤300；保存 → `updatePhotographerProfile` 全量提交；错误映射 403→「仅摄影师可编辑主页」（并返回）、400→「参数有误」 |
| 激活页入口 | `src/pages/photographer/activate.vue` | 已激活卡 `.btn-edit` 主页管理 → `navigateTo('/pages/photographer/profile-edit')`（置于 去接单管理 下、作品管理 上） |
| API 层 | `src/api/index.ts` | `getMyPhotographerProfile()` → apiGet `/v1/photographers/profile/mine`；`updatePhotographerProfile({name,description,location,mode,mutualIntro,avatar})` → apiPut `/v1/photographers/profile`；`interface MyPhotographerProfile`（含 description/mode/mutualIntro/certified） |
| 路由 | `src/pages.json` | `pages/photographer/profile-edit`（在 works 之后注册） |

### Notes

- **全量覆盖语义**：PUT 传 5 字段整体覆盖（非 PATCH）；description/mutualIntro 可为空字符串保存。mode 三态由服务端白名单校验（400 而非 403），前端 segmented 限制输入。
- 激活页入口收敛：已激活卡现有 去接单管理 / 主页管理 / 作品管理 / 申请认证（或 ✓ 已认证）四入口，均指向摄影师自助面板。

### 后续（不在本次）

- 头像选择器（当前昵称/简介/互勉仍为文本；城市已是 picker）；作品封面编辑联动；profile/mine 与激活表单（activate.vue 已激活态不可改 mode/intro）去重。
- 头像 `onerror` 兜底已加（profile / 摄影师详情 / 消息会话 / 聊天，回退 `/static/img/avatar-user.svg`）。

## 用户资料编辑（C 端）

C 端用户自助编辑账号资料（昵称/头像链接/**个人简介**），实现于 cend user-profile-edit 计划（backend `f81e67b` + frontend `07234cd`）；**2026-09-15 增补个人简介**（迁移 `000021_users_bio`，`users.bio TEXT`）。

### 后端 API

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `PUT` | `/api/v1/me/profile` | Auth required。`{name 必填, avatar?, bio?}` → 200 `{ok:true}`；缺 name → 400「name is required」；**avatar / bio 空则保留原值**（先 GetByID 再更新）；用户不存在 → 404「user not found」 |

- 后端文件：handler 为 `cmd/api/main.go` 内联路由（与 `GET /api/v1/me` 同模式）；`repository/users.sql.go` `UpdateProfile(ctx, id, name, avatar, bio)`（`UPDATE users SET name=$2, avatar=$3, bio=$4 WHERE id=$1`）；`GetByID`/`UpsertByPhone` 的 SELECT/RETURNING 均含 `bio`（**login 也返回 bio**——`UpsertByPhone` 曾漏 bio 导致登录后简介为空，已修）。`GET /api/v1/me` 返回 `bio`。

### 前端（C 端，暗色霓虹）

| Piece | File | Role |
|-------|------|------|
| 编辑页 | `src/pages/profile/edit.vue` | 暗色表单：onShow 从 `userStore.user` 预填（无 user → toast 请先登录 + redirect 登录页）；昵称 input ≤20（必填）+ 头像链接 input + 圆形头像实时预览；**个人简介 textarea ≤200**；保存 → `updateUserProfile({name,avatar,bio})` → 成功直接更新 `userStore.user` + storage → toast → navigateBack；400 → 参数有误 |
| 入口 | `src/pages/profile/index.vue` | header user-card `.edit-btn`「编辑」胶囊（`v-if="isLoggedIn"`，`@click.stop`）→ `navigateTo('/pages/profile/edit')`；未登录守卫 → 登录页 |
| API 层 | `src/api/index.ts` | `updateUserProfile({name, avatar, bio})` → apiPut `/v1/me/profile`；`mapLoginUser` 映射 `bio` |
| 路由 | `src/pages.json` | `pages/profile/edit`（navigationBarTitleText 编辑资料，在 profile/index 之后注册） |

### Notes

- **头像 URL 约定**：DiceBear 外链（`https://api.dicebear.com/7.x/avataaars/svg?seed=<seed>`），无真上传（YAGNI）；非法 URL 预览裂图（无 onerror 兜底，低风险）。
- **个人中心简介**：`profile/index.vue` 原读 `user.description`（后端从不返回）恒显示占位——现读 `user.bio`（实测显示真实简介）。
- 保存后不调 getMe：直接改 `userStore.user` + storage（与 login 持久化同模式），下次冷启动仍以 storage 为准，一致性 OK。空 avatar 前后端双兜底（后端保留原值、前端 `|| user.avatar`）。

### 后续（不在本次）

- 头像上传（当前仅外链 URL）；手机号/更多字段编辑；头像 onerror 兜底。

## NOTES

- **No `.gitignore` at root** — add before git init.
- **Dark neon homepage** — uses `$dark-*` and `$neon-*` variables from variables.scss; other pages still use light theme
- **Backend mock** — `server/mock_server.go` is a Go standard library mock (port 8081); `server/cmd/api/main.go` is the full Gin+PostgreSQL version
- **Pinia stores** — `useHomeStore` handles homepage data with API→mock fallback; `useUserStore` consumed by login/booking/order flows; `useChatStore` not yet consumed
- **uview-plus CSS active, JS layer disabled on mp-weixin** — path conflict (`node-modules` vs `node_modules`). Native components (swiper, image) used instead. uview-plus theme/variables still working.
- **Vant Weapp installed** (`@vant/weapp`) — ready for mp-weixin once npm build is configured in WeChat DevTools.
- **Real API with mock fallback** — `api/index.ts` calls `:8080`; `src/data/mock.ts` is used when backend is unreachable.
- **TabBar icons are 81×81 PNG** — generated from Material Symbols via sharp. Grey (inactive) / `#6366f1` (active).
- **No subPackages** — all pages in main package. Split if app grows.
- **Auth gate active** — booking/order flows require login; `client.ts` routes 401 to the login page.
- **manifest.json `vueVersion: "3"`** — confirmed correct for Vue 3 project.
- **DiceBear avatars** — cute cartoon style, colorful backgrounds per user. Reliable SVG, no timeout issues.
- **picsum.photos covers** — may timeout occasionally inside WeChat sandbox; CSS gradient fallbacks on image containers.

## 图片上传（本地磁盘存储）

C 端图片上传（头像 / 作品图 / 认证样片），针对「服务器无外网、无对象存储」的场景（P1 最大共性缺口）。

### 后端

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `POST` | `/api/v1/upload` | AuthRequired。multipart 字段 `file`；校验扩展名（jpg/jpeg/png/gif/webp）与大小（≤5MB）；存 `UPLOAD_DIR`（默认 `./static/uploads`，相对 CWD → `server/static/uploads`）；文件名 `<UnixNano>_<8字节hex><ext>`；201 `{url:"/static/uploads/<name>"}`；缺文件 400、类型/大小非法 400 |

- 文件：`server/internal/handler/upload_handler.go`（`UploadHandler`）；路由注册 `cmd/api/main.go`（`os.Getenv("UPLOAD_DIR")`）；静态托管复用既有 `router.Static("/static","./static")`。

### 前端

- `src/api/index.ts`：`pickAndUploadImages(count)` = `uni.chooseImage` + `uni.uploadFile`（`name:'file'`，带 Bearer token）→ 返回本地 url 数组。
- 三处接线（均保留手填外链作为兜底）：`src/pages/profile/edit.vue`（头像「从相册上传」）、`src/pages/photographer/works.vue`（作品图）、`src/pages/photographer/cert-apply.vue`（认证样片）。

### 运行环境

- **DEV**：上传落 `server/static/uploads`；H5 通过 `vite.config.ts` 的 `/static/uploads` 代理到 `:8088` 才能取图（其余 `/static` 仍由 vite 从 `src/static` 托管）。
- **PROD**：nginx `location /static/uploads/` 代理 `api:8080`；api 挂命名卷 `uploads:/app/static/uploads`（`docker-compose.prod.yml`），重建不丢。
- **mp-weixin**：返回相对路径 `/static/uploads/...`，小程序需绝对域名才能显示（同 P0 域名白名单问题）；H5 正常。

### 后续（不在本次）

- 图片压缩/缩略图、删除作品时清理孤儿文件、切换对象存储（有外网后）。

## 数据源（漫展事件抓取）

漫展数据来自外部抓取，不在迁移里（迁移只建表 + 少量 seed；`migrations/000003` 依赖事件已存在）。

| 源 | 入口 | 状态（2026-09-14 实测） |
|----|------|----------------------|
| allcpp.cn | `cmd/sync`（`service.EventSyncer` → `/allcpp/event/getList.do`） | ✅ 可用：`total=235`，一次灌 158 条 upcoming |
| nyato.com | `cmd/ingest`（`internal/ingest/nyato.go` → `/manzhan`） | ❌ 源站对 `/manzhan` 及根路径均返回 404（代理/直连皆不可达，疑似 WAF/IP 封） |
| bilibili | `cmd/ingest -bilibili` | 停用（best-effort，返回空） |

- **运行同步**：`cd server && go run ./cmd/sync`（读 `.env`；出网依赖 `HTTPS_PROXY=http://192.168.170.73:7898`）。
- **封面离线化**：`server/scripts/cache-event-covers.sh` 把 `comic_events.cover_url` 的外链封面抓到 `server/static/remote/covers/` 并改写为 `/static/remote/covers/<file>`；H5 dev 经 `vite.config.ts`、prod 经 `nginx.conf` 代理到后端。跑完 `docker exec comic-redis redis-cli FLUSHALL` 清首页缓存。
- **首页展示量**：`home_repo.GetUpcomingEvents(ctx, 50)`（最多 50 条，按 start_date 升序，`status='upcoming' AND del_flag=false`）。
- **注意**：nyato 源已失效，`config/cron.yaml` 里的 nyato 任务会持续失败；恢复调度前需改指 allcpp 或另寻源。

### 后续（不在本次）

- 把 allcpp 抓取统一进 `cmd/ingest`（当前 allcpp/nyato 两套并存）；
- cron 任务改指可用源，并自动跟进封面缓存；
- 抓取在无外网环境下的替代方案（当前依赖代理）。

## C 端评价（我的评价 / 评价摄影师）

修复于 2026-09-14（**真 bug**）：`comment/index.vue` 原把「我的评价」与「评价某摄影师」混在一页，且 `pid` 兜底为 `'1'`，而四个入口**都没传 `photographerId`** —— 导致「我的评价」显示的是**摄影师 1 收到的评价**，且「评价摄影师」永远只能评摄影师 1。

### 后端

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/api/v1/reviews/mine` | Auth required。当前用户**自己发表**的评价（新→旧），含 `photographerName`；未登录 401 |

- `repository.ListReviewsByUser`（`reviews LEFT JOIN photographers`）；`ReviewItem`/`Review` 新增 `photographerName`（仅该查询填充）。

### 前端

- `comment/index.vue`：`pid = options?.photographerId || ''`（**不再兜底 `'1'`**）；「我的评价」恒显示且来自 `/v1/reviews/mine`；「评价摄影师」分区**仅当 `pid` 存在**时渲染（定向该摄影师）；提交后重载我的评价。
- 入口传参：摄影师详情 → `?photographerId=${id}`；订单列表/详情 → `?photographerId=${order.photographerId}`；个人中心「我的评价」不带参（未登录先跳登录）。

### 注意

- uni-app H5 下**直接打开带 query 的 hash URL**（深链/刷新）时 `onLoad` 的 `options` 可能为空；App 内 `navigateTo` 正常。故深链打开该页会优雅降级为「仅我的评价」。

## C 端布局 QA（2026-09-15）

用 agent-browser 做了**全量 24 页布局体检**（视口 390×844）：逐页检测页面级横向溢出（`documentElement.scrollWidth > innerWidth`）、裂图（`img.naturalWidth===0`）、以及**非横滑容器内**超出视口右边界的元素。结果：修复后 **全部 24 页 ovf=0、无裂图**。同时用视觉模型逐页核查（15 页 OK）。

修复的 3 个真问题：

| 页面 | 问题 | 修复 |
|------|------|------|
| `booking/index.vue` | 备注 `textarea` 缺 `box-sizing: border-box`（`width:100%`+padding+border）→ 页面横向溢出 13px | 加 `box-sizing: border-box` |
| `chat/index.vue` | `.chat-content`（scroll-view）缺 `box-sizing: border-box` → 溢出 25px | 加 `box-sizing: border-box` |
| `settings/index.vue` | **硬编码用户** `{name:'小狐狸'}`，不读登录态 | 改用 `useUserStore()`（未登录显示「未登录」） |

排障要点（避免重复踩坑）：

- 首页「近期漫展」是**有意的横向轮播**（961px 行在 390px 横滑容器内）——不是溢出 bug；检测脚本需排除「有 overflow-x auto/scroll 祖先」的元素。
- 漫展封面/轮播是 **CSS `background-image`**（uni `<image>` 渲染为 div），`<img>` 检查覆盖不到；需单独检测 `backgroundImage` 的 `url()`。
- 截图**时机**会造成假阳性：大图（封面 100KB+）未加载完就截图会显示空白框 → 判定「裂图」前先核对文件是否 200。
- 直接开带 query 的 hash 深链时 `onLoad` 的 `options` 可能为空（uni H5 怪癖），详情页会显示「未知…」——非布局 bug。

## C 端功能修复（2026-09-15）

排查中又发现两个真问题（与「评价页漏传参」「settings 硬编码用户」同类的“漏传参/写死数据”）：

| 页面 | 问题 | 修复 |
|------|------|------|
| `photographer/detail.vue` | 「立即预约」与**服务卡片**跳 `/pages/booking/index` 时**未带 `photographerId`** → 预约页回退到 `bookingPhotographerId`（上次的值）或 `'1'`，**会订错摄影师** | 两处均 `?photographerId=${id}` + 写 `bookingPhotographerId`（服务卡片此前连 storage 都没写） |
| `profile/index.vue` | 统计卡 `粉丝 128 / 关注 64 / 作品 24` 是**写死的假数字** | 改为真实值：**收藏**（`/v1/favorites/:uid` 长度）/ **关注**（`/v1/follows/:uid` 的 `list` 长度）/ **作品**（摄影师取 `/v1/photographers/works/mine` 长度，否则 0）；`onShow` 刷新 |

- 验证：摄影师 3 详情点「立即预约」/服务卡片 → `#/pages/booking/index?photographerId=3`（修复前无参数）；个人中心显示 `0 收藏 / 0 关注 / 2 作品`（与接口一致，修复前 128/64/24）。

## 后端健壮性修复（2026-09-15）

边界探测发现「非法引用 → 500」与「唯一冲突判码不统一」两类问题：

| 端点 | 修复前 | 修复后 |
|------|--------|--------|
| `POST /api/v1/favorites`（不存在的摄影师） | 500（FK 违约） | **404** `photographer not found` |
| `POST /api/v1/bookings`（不存在的摄影师/服务） | 500（FK 违约） | **404** `photographer or service not found` |
| `POST /api/v1/chat/messages`（不存在的 session） | 500（FK 违约） | **404** `session not found` |
| `POST /api/admin/v1/tags` 重复名 / `POST /admins` 重名 | 依赖 `strings.Contains(err,"23505")` | 改为 `pgconn.PgError.Code` 判码（统一） |

- 新增 `server/internal/service/db_errors.go`：`isUniqueViolation(err)`（23505）+ `isForeignKeyViolation(err)`（23503）+ 哨兵 `ErrInvalidReference` / `ErrSessionNotFound`；3 处 `strings.Contains` 全部替换（`cert_application_service` 原有 inline pgconn 也收敛过来）。
- 测试 fake 同步：`admin_content_service_test` / `admin_admin_service_test` 用真 `&pgconn.PgError{Code:"23505"}` 模拟重复（原为含 "23505" 的普通 error）。
- 复验：三类非法引用 → **404**；正常收藏/发消息 → **201**；正常预约/关注/评价不受影响（`go vet`/`go test` 全绿）。
- 说明：`POST /chat/sessions` 对不存在的 `otherUserId` 仍会 201（`chat_sessions` 无 FK）——低危数据完整性问题，暂不修（改动会影响「无 user 关联的摄影师」发起会话的演示流程）。

## C 端功能修复（2026-09-15 第二轮，用户实测反馈）

用户在真实点击中发现 4 个「看着像没做，其实是接错/写死」的 bug：

| 页面 | 问题 | 根因 | 修复 |
|------|------|------|------|
| `search/search.vue` | 点「热门标签」/「热门搜索词」**没有任何反应** | 结果区是 `v-else` 挂在 `v-if="!keyword"` 上：点标签时 `keyword` 仍为空（`handleSearch()` 已拉到数据），模板仍渲染标签块，结果区**永不显示**；`handleSearch(item)` 也没把 `keyword.value` 写上 | 新增 `showResults = !!keyword || selectedTags.length>0`；结果区改 `v-if="showResults"`（标签块常驻，可继续叠加/取消标签）；`handleSearch(kw)` 内 `if (kw) keyword.value = kw` |
| `profile/index.vue` | 登录后简介仍显示**游客占位**「登录后体验完整功能」 | `displayDesc = user.bio \|\| '登录后体验完整功能'`——无简介的登录用户也命中兜底 | `displayDesc`：未登录→「登录后体验完整功能」；已登录无简介→「这个人很懒，什么都没写」 |
| `profile/index.vue` | 认证摄影师的**黄V不在「我的」页显示** | 模板只有 `.role-tag`（摄影师/Coser），从未读 `certified` | 新增 `.cert-badge`「✓ 认证摄影师」；`loadStats()` 中摄影师分支调 `getMyPhotographerProfile()` 取 `certified`（失败降级 false） |
| 订单金额（全链路） | 选 ¥399 服务下单，**订单价格显示 0** | `booking_service.Create` 里 `TotalPrice: 0` 写死，从未读服务价 | 新增 `repository.GetServiceById`；`Create` 查服务价写入 `TotalPrice`（服务不存在回退 0，不阻塞下单）；**历史 0 元订单已回填** `UPDATE bookings SET total_price = s.price FROM services s WHERE ... total_price = 0` |

- 验证（agent-browser + curl，dev :5173 → :8088 → comic-postgres）：
  - 搜索页：冷启动 `hot:true/result:false`；点标签 → `result:true` + 「找到 N 位摄影师」+ 标签块常驻且高亮。
  - 「我的」页三例：有 bio 的 coser→显示真实 bio；未认证摄影师→兜底文案 + 无徽章；认证摄影师→`✓ 认证摄影师`。
  - 订单：`POST /bookings`（服务1 ¥399）→ 响应 `totalPrice:399`；`GET /bookings/:userId` 全部金额正确；订单列表 UI 显示 `¥399`（修复前 `¥0`）。
  - 核心约拍逻辑 8/8：时段占用（`occupied:["10:00"]`）、同档冲突 409、coser 冒充摄影师 403、他摄影师越权 403、`pending→confirmed` 200、非法回退 409、`confirmed→completed` 200、接单面板 join 字段齐全。
- 门禁：`go build`/`go vet`/`go test ./...`（Go 1.22，`GOROOT=/home/user/go-sdk/go1.22`）全绿；`npx vue-tsc --noEmit` EXIT=0；`npm run build:h5` 成功；`docker compose -f deploy/docker-compose.prod.yml build api web && up -d` 已重建 prod（nginx:80 / api:8080）。
- 遗留：booking 3（`service_id=3` 精品套餐 ¥1299）`total_price=999` 为历史演示数据，未改动（非本 bug 的 0 值）。

## C 端鉴权加固 + 会话列表 500 修复（2026-09-15 第三轮，深度 review 发现）

深度自查发现 C 端存在**成片的越权（IDOR）**：按 `:userId` 取数的端点普遍没有归属校验，部分写接口还**信任请求体里的身份**。正确模式其实代码里早有（`notification_handler` 的 403 校验 + 中间件 `middleware.UserID(c)`），只是其它模块没跟上。

### 修复前实测（curl，dev :8088）

| 用例 | 修复前 | 修复后 |
|------|--------|--------|
| 无 token `GET /bookings/1` | **200**（泄露任意用户订单+姓名） | 401 |
| 无 token `POST /bookings {coserId:5}` | **201**（冒名下单） | 401 |
| 无 token `POST /reviews {userId:1,userName:"我是冒充者"}` | **201**（冒名发评价） | 401 |
| 无 token `GET /follows/1` / `POST /follows` / `DELETE /follows/1/1` / `POST /subscribe` | **200** | 401 |
| coser5 token `GET /favorites/1` / `/follows/1` / `/bookings/1` / `/chat/sessions/1` | **200** | 403 |
| coser5 token（伪造 body `userId:1`）`POST /favorites` | 替 user1 收藏 | 记到**自己**名下 |
| coser5 token（伪造 `senderId:1`）`POST /chat/messages` | 可冒名发言 | sender 强制=token 用户 |
| 未认证 `GET /chat/messages/1` | 可读他人会话 | 401；局外人读→403 |
| photo5 token `GET /bookings/photographer/2` | **200**（看别人接单） | 403 |

### 改动

| 文件 | 改动 |
|------|------|
| `internal/handler/authz.go`（新增） | `assertSelf(c, userID)` 统一归属校验（不匹配→403）；`parseUserID(raw)` |
| `favorite_handler.go` | `Add` 忽略 body `userId`、改用 token；`List`/`Remove` 补 `assertSelf` |
| `booking_handler.go` | `Create` 用 token 覆盖 `coserId`；`ListByUser` 补 `assertSelf`；`ListByPhotographer` 改用新增的 `ListByPhotographerForUser`（摄影师身份不匹配→403） |
| `booking_service.go` | 新增 `ListByPhotographerForUser`（经 `GetPhotographerByUserID` 解析调用者摄影师身份）；`CreateBookingRequest.CoserID` 去掉 `binding:"required"`（由 token 决定） |
| `follow_handler.go` | `Follow`/`Subscribe` 用 token 覆盖 `userId`；`Unfollow`/`ListFollows` 补 `assertSelf` |
| `review_handler.go` | 注入 `userRepo`；`Create` 用 token 决定 `userId`，并**从 DB 取** `userName`/`userAvatar`（不再信任请求体） |
| `chat_handler.go` | `CreateSession` 用 token 覆盖 `userId`；`SendMessage` 用 token 覆盖 `senderId`；`ListSessions` 补 `assertSelf`；`ListMessages` 改用 `ListMessagesForUser`（非参与者→403） |
| `chat_service.go` / `chat.sql.go` | 新增 `GetChatSessionParticipants` + `assertParticipant` + `ListMessagesForUser`/`SendMessageForUser` |
| `cmd/api/main.go` | 给 `POST /bookings`、`GET /bookings/:userId`、`POST /reviews`、`POST|DELETE|GET /follows*`、`POST /subscribe` 补 `AuthRequired`；`NewReviewHandler(reviewSvc, userRepo)` |

- **前端零改动**：各页面本就用「自己的 token + 自己的 id」调用，加校验后自然通过（已用 agent-browser 复验订单列表/个人中心/消息页渲染正常）。

### 顺带修复：`GET /chat/sessions/:userId` 500

`listSessions` SQL 里 `m.created_at::text AS last_time` 来自 `LEFT JOIN LATERAL`，**无消息的会话**该列为 NULL → 扫描进 `SessionRow.LastTime string` 报错 → 整个接口 500（即"只要有一个还没聊过的会话，消息页就整体打不开"）。修复：`COALESCE(m.created_at::text, '') AS last_time`。实测 `GET /chat/sessions/5` 500 → **200**。

### 门禁与验证

- `go build`/`go vet`/`go test ./...`（Go 1.22）全绿；dev :8088 与 prod api 镜像均已重建。
- 回归矩阵：**A) 9 个无 token 用例 → 全部 401**；**B) 5 个跨用户用例 → 全部 403**；**C) 3 个伪造身份用例 → 全部记到 token 用户名下**；**D) 正常流程（自己的收藏/关注/订单/会话、下单 ¥399、接单面板、发消息）全部 200/201**。
- 审计数据（SEC-AUDIT 标记的 booking/review/message/notification）已全部清理。

### 仍未做（不在本轮）

- `GET /chat/sessions/:userId` 的 peer 若指向不存在的用户（演示数据 `user2_id=999999`）仍会返回该 bogus peer（不崩，但展示为陌生 id）——数据问题，非代码。
- `created_at` 列在 `chat_messages`/`notifications`/`reviews` 均 `is_nullable=YES`（仅靠 DEFAULT now()），显式插入 NULL 仍可能触发同样的 scan 报错；本轮仅修了必然为 NULL 的 `last_time`。

## 逐页功能清点 + settings 修复 + P1/P2（2026-09-15 第四轮）

### 逐页清点（25 页）

对 `pages.json` 注册的 25 个页面做了「静态 API/click 映射 + 浏览器逐页冒烟（reload + 错误捕获）」，结论：

- **25/25 页均正常渲染，无运行时错误**（`agent-browser errors` 为空），无死按钮（`@click` 未定义函数扫描仅 2 处内联表达式误报）。
- 无需直接 API 的页面均为 wrapper 调用（`cert-apply`/`profile-edit`/`works`/`login`/`index` 走 `@/api` 与 store），非空壳。
- 唯一发现不完善的页面是 `settings/index.vue`（见下）。

### settings/index.vue 三处真缺陷

| 问题 | 根因 | 修复 |
|------|------|------|
| **「退出登录」是假的** | `logout()` 只弹确认框 + toast「已退出登录」，**从不调用 `userStore.logout()`** —— 提示成功但用户仍登录 | 调 `userStore.logout()` + toast + `navigateBack`；未登录时跳登录页（与 `profile/index.vue` 的正确实现对齐） |
| 「编辑个人资料」跳错页 | `goProfileEdit()` → `/pages/profile/index`（资料**查看**页） | → `/pages/profile/edit`（编辑页），未登录先跳登录 |
| 6 个菜单项全是 toast 占位（点一下弹出自己的标题） | 无对应页面 | 新增 `src/pages/settings/doc.vue`（`?type=privacy|terms|about|help` 渲染真实文案），隐私政策/用户协议/关于米拉漫展 → 文档页；`profile/index.vue` 的「帮助与反馈」也接上 `?type=help` |

- `账号与安全` / `黑名单管理` 仍无后端，toast 文案改为诚实的「该功能暂未开放」（原先弹自己的标题，像是已完成）。

### P1：聊天 peer 校验 + 清理 bogus 会话

- 数据里 `chat_sessions` 存在 peer 指向不存在用户的脏会话（`user2_id=999999`）→ 已删除（该校验同时是 `GET /chat/sessions` 曾 500 的诱因之一）。
- `POST /api/v1/chat/sessions` 现在校验 `otherUserId` 存在：不存在 → **404** `user not found`（新增 `repository.UserExists` + `ChatService.AssertPeerExists`，复用既有 `ErrUserNotFound`）。
- 实测：`{otherUserId:999999}` → 404；`{otherUserId:2}` / `{otherUserId:7}` → 201。

### P2：时间列收紧 + 价格逻辑单测

- **迁移 `000022_created_at_not_null`**：`chat_messages`/`notifications`/`reviews` 的 `created_at` `SET NOT NULL`（应用前实测 3 表 0 个 NULL，安全）——彻底关闭「NULL 扫进 string → 500」这一类问题。down 为 `DROP NOT NULL`。
- **`internal/service/booking_service_test.go` 新增 2 个用例**（用记录型 fake DBTX 断言传给 SQL 的实参）：
  - `TestCreateBooking_WritesServicePrice`：`Create` 把服务价 **399** 作为 `TotalPrice` 传进 `CreateBooking`。
  - `TestCreateBooking_ServiceLookupFailsFallsBackToZero`：服务查询失败时回退 0 且**不阻塞下单**。

### 验证

- `go build`/`go vet`/`go test ./...`（Go 1.22）全绿；`npx vue-tsc --noEmit` EXIT=0；`npm run build:h5` 成功；prod `deploy-api`/`deploy-web` 已重建。
- 浏览器复验：settings「编辑」→ `#/pages/profile/edit`；「隐私政策」→ `#/pages/settings/doc?type=privacy` 并渲染真实文案；「退出登录」确认后 `localStorage.token=null`（真登出）。
- prod 实测：无 token `GET /api/v1/bookings/1` → **401**；无 token `POST /api/v1/chat/sessions` → **401**。

### 仍未做（不在本轮）

- `账号与安全` / `黑名单管理` 仍为「暂未开放」占位（无后端支撑）。
- `photographer/orders.vue` 的 `onShow` 不调用 `userStore.init()`：冷启动/HMR 场景下若 store 未初始化会误显示「请先开通摄影师身份」；正常登录路径不受影响（store 在 App 启动时已 init），但更稳妥的做法是依赖统一的应用级 init。

## C 端完善（2026-09-15 第五轮，聊天闭环）

盘点「C 端还有哪些没闭环」，确认分页（`photographer/list`、`event/list` 的 `@scrolltolower`）、401 全局处理（`client.ts` 清 token + 跳登录）、未读角标（`utils/badge.ts` → TabBar）、下单成功跳转、通知已读**均已实现**。真实断点只有 2 个，均在聊天链路：

| 问题 | 根因 | 修复 |
|------|------|------|
| **聊天页不刷新**：退出再进看不到新消息，对方发来的消息永远不出现 | `chat/index.vue` 只有 `onMounted`（uni 页面栈保活，返回时不重触发），且**无任何轮询** | 抽出 `loadMessages()`；新增 `onShow` 刷新 + 可见时 **5s 轮询**，`onHide`/`onUnload` 清除定时器；仅在消息数变化时标记已读 + 刷新角标 + 滚到底，避免每 5s 打一次已读接口 |
| **会话列表顺序不更新**：有新消息的会话不会冒到顶部 | `chat.sql.go` 的 `InsertMessage` 只插消息，**不 bump `chat_sessions.updated_at`**，而 `listSessions` 是 `ORDER BY s.updated_at DESC` | `InsertMessage` 改为原子 CTE：插入消息 + `UPDATE chat_sessions SET updated_at = NOW()`，并 `RETURNING` 消息 id（会话不存在时 FK 违约照旧 → 404） |

- 验证（curl + agent-browser）：
  - **F1**：打开 session 1（5 条消息）→ 由对方经 API 发一条 → 等 6s → **消息数 5→6 且新消息出现**（轮询生效）。
  - **F2**：给较旧的 session 1 发消息 → 会话顺序 `4,3,1` → **`1,4,3`**（置顶生效）。
- `photographer/orders.vue` 的 `userStore.init()` 脆弱点**降级为不修**：`App.vue` 的 `onLaunch` 已调 `init()`，真实冷启动不受影响，此前只在 dev/HMR 注入场景复现。
- 门禁：`go build`/`go vet`/`go test ./...` 全绿 · `vue-tsc` EXIT=0 · `build:h5` 成功 · prod api/web 已重建（无 token `/bookings/1` → 401）。

### 仍在（可选，未做）

- 点击通知只弹标题 toast，**不跳转到对应订单**——`notifications` 表无关联字段（要做得加 `link_type`/`link_id`，与 banners 同款设计）。
- 聊天仍是 5s 轮询而非 WebSocket（mp-weixin 域名白名单限制），对方消息最长延迟 5s。

## 通知跳转（接入订单）+ 迁移幂等性修复（2026-09-15 第六轮）

### 通知 → 订单/接单面板跳转

`notifications` 加 `link_type` / `link_id`（迁移 `000023_notifications_link`），预约类通知点击直达对应页面：

| 迁移 | 内容 |
|------|------|
| `000023_notifications_link.{up,down}.sql` | `notifications` 加 `link_type VARCHAR(20)` + `link_id BIGINT`（可空，历史通知为 NULL） |

| 层 | 改动 |
|----|------|
| `repository.InsertNotification` | 签名加 `linkType *string, linkID *int64`（`querier.go` 接口同步） |
| `repository.ListNotifications` / `ListAllNotifications` | SELECT 加 `link_type, link_id`；`NotificationRow` 加 `LinkType`/`LinkID`（两处 scan 同步，admin 历史列表也带出） |
| `NotificationService` | `Create` 照旧（link 为 NULL）；新增 `CreateLinked(ctx, userID, typ, title, content, linkType, linkID)` |
| `BookingService` | `notify`/`notifyPhotographer` 加 `bookingID`；新增 `notifyLinked` —— **coser 通知** `link_type='order'`、**摄影师通知** `link_type='photographer_orders'`（两类跳转目标不同：coser 去订单详情，摄影师去接单面板） |
| `notification_handler.Create` | body 可选 `linkType`/`linkId` |
| `src/pages/message/index.vue` | 映射 `linkType`/`linkId`；`readNotification`：`order` → `/pages/order/detail?id=<linkId>`；`photographer_orders` → `/pages/photographer/orders`；无 link → 原 toast |

- 验证（dev + prod）：下单 → coser 通知 `{"linkType":"order","linkId":N}`、摄影师通知 `{"linkType":"photographer_orders","linkId":N}`；agent-browser 点「预约成功(#12)」→ `#/pages/order/detail?id=12` 并渲染出对应订单（古风公子/基础套餐/备注）。

### 迁移幂等性修复（根因：prod schema 漂移）

**现象**：prod API 的 `GET /notifications/:userId` 返回 500 —— prod 库缺 `link_type`/`link_id` 列（新代码查旧 schema）。

**根因**：`deploy/migrate.sh` 是「按 `*.up.sql` **全量按序执行** + `set -euo pipefail` + `psql -v ON_ERROR_STOP=1`」，而迁移里有 **21 处非幂等语句**（`CREATE TABLE`/`CREATE INDEX`/`ADD COLUMN` 无 `IF NOT EXISTS`）与 **11 处无 `ON CONFLICT` 的种子 INSERT**（`tags.name`、`photographer_tags` PK、`admins.username` 均有唯一约束）。**一旦重放遇到已应用的语句就中止，后续迁移永不执行** —— prod 因此停在旧 schema。

**修复**：

| 改动 | 说明 |
|------|------|
| 全部 `*.up.sql` 的 `CREATE TABLE`/`CREATE INDEX`/`CREATE UNIQUE INDEX`/`ADD COLUMN` | 补 `IF NOT EXISTS`（21 处）——重放变 no-op |
| `000002_seed.up.sql` | 整段包进 `DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM tags) THEN ... END IF; END $$;` —— 仅空库播种，重放不重复插入 |
| `000012_admins.up.sql` | `INSERT ... ON CONFLICT (username) DO NOTHING` |

- **prod 已补应用 `000022` + `000023`**（先核对 prod `created_at` 无 NULL、`link_type`/`link_id` 缺失，再执行）。
- 验证：`bash deploy/migrate.sh` **EXIT=0 完整通过**（修复前会中止），重复执行仍 EXIT=0（no-op）；prod 种子计数无重复（tags=13 / services=4 / photographers=4 / admins=1 / banners=3）；dev 与 prod 列集合一致（153 列）。
- 门禁：`go build`/`go vet`/`go test ./...` 全绿 · `vue-tsc` EXIT=0 · `build:h5` 成功 · prod api/web 已重建。

### 仍在（可选）

- `账号与安全` / `黑名单管理` 无后端支撑，仍为「暂未开放」。
- 聊天仍是 5s 轮询而非 WebSocket。

## 账号与安全 + 黑名单管理（2026-09-15 第七轮）

把 settings 里最后两个占位项做成真功能。

### 账号与安全

| 层 | 改动 |
|----|------|
| `repository.UserRepo` | 新增 `DeleteTokensByUser(ctx, userID)`（`DELETE FROM user_tokens WHERE user_id=$1`） |
| `cmd/api/main.go` | 新增 `POST /api/v1/me/logout-all`（AuthRequired）→ 清空该用户全部 token → 200 `{ok}` |
| 前端 | 新增 `pages/settings/security.vue`（绑定手机号脱敏 `138****8000` / 昵称 / 账号状态 / **退出所有设备**）；`api/index.ts` 加 `logoutAllDevices()`；settings「账号与安全」接入 |

- 实测：`/me` 200 → `logout-all` 200 → **同一 token 再请求 401**（真失效）→ 重新登录 200；无 token 401。

### 黑名单管理

| 层 | 改动 |
|----|------|
| 迁移 `000024_user_blocks` | `user_blocks(user_id, blocked_user_id, created_at, PK(user_id,blocked_user_id))` + `idx_user_blocks_blocked` |
| `repository/user_blocks.sql.go`（新） | `BlockUser`（ON CONFLICT DO NOTHING）/ `UnblockUser` / `IsBlockedPair`（**双向**）/ `ListBlockedIDs` / `ListHiddenUserIDs`（双向并集）/ `ListBlockedUsers`（join users） |
| `service/block_service.go`+`handler/block_handler.go`（新） | `POST /api/v1/blocks`（拉黑，自身 400 / 不存在 404）、`DELETE /api/v1/blocks/:blockedUserId`、`GET /api/v1/blocks`（`{list}`） |
| `middleware.AuthOptional`（新） | 有合法 token 则注入 `user_id`，无则放行（不 401） |
| `chat_service.GetOrCreateSession` | 拉黑关系（任一方向）→ `ErrForbidden` → **403** |
| `booking_service.Create` | coser 与「摄影师对应 user」存在拉黑关系 → `ErrForbidden` → **403** |
| `GET /api/v1/photographers` | 挂 `AuthOptional`；登录用户传入 `ListHiddenUserIDs`（我拉黑的 + 拉黑我的）→ `SearchPhotographers` 新增 `ExcludeUserIDs` 过滤；**cache key 追加 uid**（否则过滤结果会串给别的用户） |
| 前端 | 新增 `pages/settings/blocks.vue`（列表 + 解除）；`photographer/detail.vue` 底部新增「⊘ 拉黑」入口；`api/index.ts` 加 `getBlocks/blockUser/unblockUser`；settings「黑名单管理」接入 |

- 实测（dev）：拉黑 201 / 不存在 404 / 拉黑自己 400 / 无 token 401；**拉黑后**：聊天 403、下单 403、**反向（被拉黑方主动）也 403**、被拉黑摄影师从搜索消失、URL 带 tag 时过滤仍生效（`tags=日系` 只剩另一位）；匿名搜索仍 5 位全出（不误伤）。
- **踩坑**：`ExcludeUserIDs` 为 nil 时 `cardinality(NULL)` 为 NULL → WHERE 整体不成立 → **匿名搜索返回空列表**。修复：SQL 用 `COALESCE(cardinality($6::bigint[]), 0) = 0` + 服务层 nil → `[]int64{}` 归一化。
- 测试同步：`booking_service_test.go` 的 fake 行序补上新增的 `GetPhotographerById`（黑名单校验）调用；断言改为「按参数个数 8 定位 CreateBooking」而非固定下标。

- 门禁：`go build`/`go vet`/`go test ./...` 全绿 · `vue-tsc` EXIT=0 · `build:h5` 成功 · prod api/web 已重建 · `bash deploy/migrate.sh` 全量重放 EXIT=0（含 000024，幂等）· prod 无 token `/blocks` → 401。

### 仍在（可选）

- 聊天仍是 5s 轮询而非 WebSocket（mp-weixin 域名白名单限制）。
- `黑名单` 未拦截「已有会话」中的历史消息（只挡新会话/新预约/搜索曝光）。

## C 端回归收口：API 冒烟脚本（2026-09-15 第八轮）

### `server/scripts/smoke.sh`（新增）

一份**可重复执行、跑完自动清理**的 C 端 API 回归脚本，覆盖 9 组共 **63 项断言**：

| 组 | 覆盖 |
|----|------|
| A/B | 无 token → 401（8 项）；跨用户 → 403（5 项） |
| C | 伪造身份被忽略（body `userId`/`coserId`/`senderId` 一律以 token 为准） |
| D | 约拍状态机（时段占用、409 冲突、403 越权、`pending→confirmed→completed`、非法回退 409、面板/订单可读） |
| E | 通知链路（`linkType=order` / `photographer_orders`） |
| F | 收藏 / 关注 |
| G | 聊天（参与者校验、局外人 403、伪造 senderId、peer 不存在 404、未读） |
| H | 黑名单（拉黑/自拉黑/不存在、聊天+下单+**反向** 403、搜索隐藏、匿名不误伤、标签筛选组合、解除恢复） |
| I | 账号安全（`logout-all` → 旧 token 401） |

- 用法：`bash server/scripts/smoke.sh`（默认 dev :8088）；打 prod：`BASE_URL=http://127.0.0.1 DB_CONTAINER=deploy-postgres-1 bash server/scripts/smoke.sh`。**`BASE_URL` 只填源站**（脚本自行拼 `/api/v1/...`）。
- **数据集无关**：摄影师 id/用户 id 从测试账号动态解析（`/photographers/by-user/:id`），不硬编码 dev 的 id，因此同一脚本可打 dev/prod。
- 清理：以 `SMOKE` 标记创建数据，`trap EXIT` 自动删除 bookings/reviews/messages/notifications/user_blocks，可反复执行。
- 本轮实测：**dev 63/63、prod 63/63 全绿，EXIT=0**。

### 脚本跑出来的 3 个真问题（已修）

| # | 问题 | 根因 | 修复 |
|---|------|------|------|
| 1 | **解除拉黑后搜索最长 5 分钟仍看不到对方** | 拉黑/解除改变了搜索结果，但 `photographers:*` 列表缓存（5min TTL）不失效 | `photographer_handler.List`：**个性化（有 `excludeUserIDs`）时直接不缓存**（读/写都跳过）；无过滤时走共享缓存。既修了过期，又避免为每个 uid 建缓存 |
| 2 | **prod 摄影师侧功能整体不可用** | prod 库 `photographers.user_id` **全为 NULL**（种子只建摄影师未关联用户）→ `photographerId=null`、`works/mine` 403、`by-user` 404 | 新增迁移 `000025_link_seed_photographers`：按手机号约定把摄影师挂到用户（`1000000000‖p.id`，另含 p=5↔`13700000005`），**幂等**（仅补 NULL）。dev no-op；prod 关联后 `works/mine` 403→**200** |
| 3 | 被拉黑 + 时段已被占用时返回 409 而非 403 | `Create` 里冲突检查先于拉黑检查 | 调整顺序：**鉴权（拉黑）检查前置**，语义上授权先于业务规则；同步更新 `booking_service_test` 的 fake 行序 |

- 门禁：`go build`/`go vet`/`go test ./...` 全绿 · `vue-tsc` EXIT=0 · prod api 已重建 · `migrate.sh` 含 000025 重放 EXIT=0。


## 摄影师自助定价 + 单轮报价

摄影师可**自主维护自己的套餐与价格**（支持 互勉 / 面议），并对「面议」订单进行**单轮报价**（摄影师报价 → coser 接受/拒绝）。实现于 photographer-pricing 计划（9 个任务，commits `bf057e5`…`8b192b6`，PR [#1](https://github.com/Maluslock/comic/pull/1)），**已推 prod 并在 prod 运行**（`master` 未合并前 prod 领先于 master）。

**背景**：原 `services` 是平台级 4 个固定套餐，且 `GetDetail` 无摄影师过滤 → **任意摄影师详情返回同一份列表**，摄影师表无任何价格字段，`mode`(free/pay/both) 与价格彻底脱钩。导致定价权在平台而非供给方、`mode=互勉` 的摄影师仍挂 ¥399、且无协商通道。

### 三态价格语义（唯一口径）

| `services.price` | 语义 | 下单后 |
|------------------|------|--------|
| `NULL` | **面议** | `price_mode='negotiable'`、`price_status='awaiting_quote'`、`total_price=0`（占位） |
| `0` | **互勉** | `price_mode='mutual'`、`price_status='agreed'`、`total_price=0` |
| `> 0` | **固定价** | `price_mode='fixed'`、`price_status='agreed'`、`total_price=价格` |

> `bookings.total_price` **NOT NULL**（Go 字段 `int32`，非指针）：互勉/面议写 `0` 占位，展示靠 `price_status`；coser 接受报价时由 SQL 把 `total_price = quote_price` 落定。

### 数据模型（无新表，一表两用）

| Migration | 内容 |
|-----------|------|
| `000026_photographer_services` | `services` 加 `photographer_id`（**NULL=平台模板**，非 NULL=该摄影师套餐）、`is_active`、`sort_order`；`price DROP NOT NULL`；建索引；**把 4 个模板复制给每位现有摄影师**（幂等，重放插 0 行） |
| `000027_booking_pricing` | `bookings` 加 `price_mode`/`quote_price`/`price_status` + **快照** `service_name`/`service_duration`；存量单回填快照 |
| `000028_drop_bookings_service_fk` | **删掉** `bookings_service_id_fkey` —— 使「删除套餐」不被历史订单阻断（历史靠快照；归属由应用层校验） |

### 后端 API

| Method | Endpoint | Behavior |
|--------|----------|----------|
| `GET` | `/api/v1/photographers/services/mine` | Auth。我的套餐（**含未上架**）→ `{list}`，`price` 为 `number/null`；非摄影师 403 |
| `POST` | `/api/v1/photographers/services` | Auth。`{name,price?,description?,duration,isActive?,sortOrder?}` → 201 `{id}`；`price` 空=面议、`0`=互勉；负数 400「invalid price」；非摄影师 403 |
| `PUT` | `/api/v1/photographers/services/:id` | Auth。全量更新（SQL `WHERE id AND photographer_id`，消除 TOCTOU）→ 200 `{ok}`；**非本人 403**、不存在 404 |
| `DELETE` | `/api/v1/photographers/services/:id` | Auth。硬删 → 200 `{ok}`；**非本人 403**、不存在 404 |
| `GET` | `/api/v1/services/templates` | **公开**。平台模板（`photographer_id IS NULL ORDER BY sort_order,id`）→ bare array，供「一键预填」 |
| `POST` | `/api/v1/bookings/:id/quote` | Auth。摄影师报价 `{price}`（∈(0,99999]）→ 200；非本人 403、状态不符 409、越界 400 |
| `POST` | `/api/v1/bookings/:id/quote/respond` | Auth。coser `{accept}`（**必填**，缺 400）→ 200；非本人 403、状态不符 409 |

**改造既有端点**：`GET /photographers/:id` 的 `services` 改为**该摄影师自有且上架**的（`GetActiveServicesByPhotographer`）；`POST /bookings` **请求体不变**，服务端按套餐 `price` 推导 `price_mode` + 写名称/时长快照 + **校验套餐归属**（不属于该摄影师 → 404 `ErrInvalidReference`）。

### 报价状态机（不改既有 `canTransition`）

```
[固定价] 下单 → agreed                        → pending → confirmed → completed
[互勉]   下单 → agreed, total=0                → pending → confirmed → completed
[面议]   下单 → awaiting_quote, total=0
                ↓ 摄影师报价(0,99999]
              quoted (quote_price=X)
                ↓ coser 接受            ↓ coser 拒绝
        status=confirmed            status=cancelled
        price_status=agreed         price_status=rejected
        total_price=X
```
- **报价即接单意愿**：被接受后直接 `confirmed`（不做二次确认）
- **status 守卫**（fix）：`UpdateStatus`/`AdminUpdateStatus` 拒绝「未定报价的面议单被直接 confirm」（409）；`Quote`/`RespondQuote` 也要求 `status='pending'` —— 否则**已取消订单可被"接受"复活**

### 前端（C 端，暗色霓虹）

| Piece | File | Role |
|-------|------|------|
| 我的套餐管理页 | `src/pages/photographer/services.vue` | list/form 双模式：`共 N 个套餐` / 一键预填 / 新增；卡片含 名称·三态价格·时长·上架·编辑·删除；表单 名称≤50(必填)、**价格留空=面议**、时长、说明≤200；403→toast「仅摄影师可管理套餐」并返回；空态带「＋ 新增套餐」CTA |
| 三态价格渲染 | `src/utils/mappers.ts` `formatPrice()` | 统一 `¥399` / `互勉` / `面议`；用于 `ServiceCard.vue`、`photographer/detail.vue`、`booking/index.vue` |
| 订单报价交互 | `src/utils/quote.ts` + `order/list.vue`、`order/detail.vue`、`photographer/orders.vue` | 按 `priceStatus` 渲染 `待报价`/`已报价 ¥N`/`¥N`/`已拒绝`；摄影师「报价」（`uni.showModal{editable}`，校验整数 1..99999）+ coser「接受/拒绝」（二次确认）；**409 → toast「报价状态已变化，请刷新」**并刷新 |
| 入口 | `activate.vue` | 已激活卡新增 `.btn-services` 套餐管理 |
| 路由 | `src/pages.json` | `pages/photographer/services` |

### 实现中修复的真实缺陷（值得记）

| # | 缺陷 | 根因 / 修法 |
|---|------|-------------|
| 1 | **Booking 字段顺序与 SQL 列顺序错位** | 计划自相矛盾（结构体插在 `UpdatedAt` 前、SQL 追加在 `updated_at` 后）→ **fake 单测全过但真实扫描必 500**。修：统一为 SQL 顺序。**教训：fakes 不能证明 SQL↔struct 对齐，必须真库验证** |
| 2 | **无主（模板）套餐可被任意摄影师预约**（越权串价） | 归属校验写成 `ownerID != nil && ...`，跳过了 `photographer_id IS NULL` → 改为 `ownerID == nil \|\| ownerID != req.PhotographerID` 一律 404 |
| 3 | **删除有历史订单的套餐 → FK 违约 500** | `bookings_service_id_fkey` 无 ON DELETE 且 `service_id` NOT NULL → 迁移 `000028` 删 FK + join 加 `COALESCE(s.name,'')` |
| 4 | **报价状态机忽略订单 status → 已取消订单可被"接受"复活** | 加 `status='pending'` 守卫（服务层） |
| 5 | **快照写了但从不读** | `ListByUser` 用 live join 的 `s.name`，改套餐名会改写历史订单名 → 改为**快照优先**、live 兜底（`COALESCE(b.service_name, s.name, '')`） |
| 6 | `/services/templates` 返回**全部 24 条**而非 4 条模板 | `GetServices` 无 `photographer_id IS NULL` 过滤 → 补过滤 |
| 7 | **部署地雷**：`deploy/Dockerfile.api` 拷贝 **gitignored 预编译二进制** | 单独 `docker compose build api` 会发出 crash-loop 镜像（二进制陈旧 + 动态链接 vs alpine/musl）→ **必须先 `bash deploy/build.sh`**（静态编译 + H5 build）再 compose build |

### 验证

- `go build`/`go vet`/`go test ./...` 全绿 · `npx vue-tsc --noEmit` EXIT=0
- `server/scripts/smoke.sh`：**dev 78/78 ×2 + prod 78/78**，全 EXIT=0（新增套餐/报价用例组；并修掉因「模板不可预约」而失效的硬编码 `serviceId`）
- agent-browser 实测：套餐三态渲染、面议下单→待报价→报价→接受→`¥X`/confirmed 全链路
- 测试与 UI 文档见 `docs/qa/`（40 条用例 Excel + 测试报告 + UI 优化建议 + **多页 UI 审计**）

### 后续（不在本次）

- **多页 UI 审计发现的系统性配色问题未全修**（见 `docs/qa/2026-09-21-C端多页UI审计-第二批.md`）：S-1 白字+青色/渐变底 2.43:1（订单详情状态横幅、首页渐变条）、S-2 标签 pill 紫字压紫底 3.8:1（首页/漫展详情）。**目前只在「我的套餐管理页」单页修过，未在共享层收敛**
- 系统性 load-time 错误（每页 1 条 `agent-browser errors` 的 `{text:"Object"}`，登录页也有、无可见破坏、SPA 切换不触发）——**未定位**，需 CDP init-script 抓栈
- 订单详情「拍摄时长」恒显 0 —— `BookingItem` DTO 未返回 `serviceDuration`（快照列已有值）
- 面议被拒后**不支持重报**（单轮设计）；报价无留言/附件；套餐无排序 UI（`sort_order` 字段已留）
