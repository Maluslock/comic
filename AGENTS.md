# PROJECT KNOWLEDGE BASE

**Generated:** 2026-07-06
**Updated:** 2026-07-24 (homepage redesign: dark neon theme, Pinia store, Go mock server, QuickActionBar, HomeSkeleton)
**Commit:** N/A
**Branch:** N/A

## OVERVIEW
漫展摄影平台 — Uniapp mini-program connecting cosplayers with photographers at comic conventions. Vue 3 + TypeScript + Pinia + Vite, targeting WeChat MP (mp-weixin) and H5. Mock-data-driven with Go backend. Dark neon theme (purple #a855f7 + cyan #06b6d4) for homepage.

## STRUCTURE
```
comic/
├── src/
│   ├── api/index.ts        # Mock API (photographers, works, events, bookings)
│   ├── components/          # 6 presentational components (incl. new)
│   ├── data/mock.ts         # Mock data (photographers, works, events, tags)
│   ├── pages/               # 10 page dirs, 12 .vue files
│   ├── static/tab/          # TabBar icons (PNG: DiceBear + Material Symbols)
│   ├── stores/              # Pinia stores (user, chat, home)
│   ├── styles/              # variables.scss + global.scss
│   ├── types/index.ts       # All domain interfaces (incl. ComicEvent)
│   ├── uni.scss             # uview-plus theme entry
│   ├── App.vue              # Root (lifecycle hooks + uview-plus/global SCSS)
│   ├── main.ts              # SSR entry (Vue + Pinia + uviewPlus)
│   ├── manifest.json        # Platform config (vueVersion: "3", mergeVirtualHostAttributes)
│   └── pages.json           # Routes + tabBar + easycom config
├── server/
│   ├── cmd/api/main.go      # Go+Gin backend (PostgreSQL, gin, viper)
│   ├── mock_server.go        # Standalone Go mock server (std lib, port 8081)
│   └── internal/            # handler, service, repository, model, config
├── vite.config.ts           # @dcloudio/vite-plugin-uni + SCSS variable injection
├── tsconfig.json            # strict: true, @/* alias
├── index.html               # H5 entry
└── package.json             # uview-plus, @vant/weapp, pinia, sharp, dayjs, clipboard
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add a new page | `src/pages.json` → `src/pages/<name>/` | Create dir + index.vue, register in pages.json |
| Add API endpoint | `src/api/index.ts` | Currently all mock; swap to real HTTP for production |
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
| `useUserStore` | store | `src/stores/user.ts` | Auth state; NOT yet consumed by any page |
| `useChatStore` | store | `src/stores/chat.ts` | Chat state; NOT yet consumed by any page |
| `useHomeStore` | store | `src/stores/home.ts` | Home data (loads /api/v1/home→mock fallback); consumed by index page |
| `getPhotographers` | api fn | `src/api/index.ts` | Filterable photographer list (mock) |
| `getComicEvents` | api fn | `src/api/index.ts` | Upcoming comic conventions (mock, 5 events) |
| `getTags` | api fn | `src/api/index.ts` | Style tags (日系, 古风, etc.) |
| `getWorks` | api fn | `src/api/index.ts` | Portfolio works (mock) |
| `createBooking` | api fn | `src/api/index.ts` | Booking creation (mock) |
| `User` | interface | `src/types/index.ts` | Base user (role: photographer\|coser) |
| `Photographer` | interface | `src/types/index.ts` | Extends User with works/services/reviews/rating |
| `ComicEvent` | interface | `src/types/index.ts` | Comic convention (name, venue, dates, photographerCount) |
| `Work` | interface | `src/types/index.ts` | Portfolio work (images, tags) |
| `Booking` | interface | `src/types/index.ts` | Booking with status enum |
| `PhotographerCard` | component | `src/components/PhotographerCard.vue` | Card: circle avatar, left purple accent, shadow-md |
| `WorkCard` | component | `src/components/WorkCard.vue` | Image card with gradient overlay, shadow-sm |
| `ServiceCard` | component | `src/components/ServiceCard.vue` | Service tier card with selection |
| `ReviewCard` | component | `src/components/ReviewCard.vue` | Review with rating + avatar |
| `QuickActionBar` | component | `src/components/QuickActionBar.vue` | 4-grid shortcut nav (neon border, dark bg) |
| `HomeSkeleton` | component | `src/components/HomeSkeleton.vue` | Shimmer skeleton for homepage loading |
| `mockEvents` | data | `src/data/mock.ts` | 5 mock comic events (CP30, CD28, 萤火虫, IDO42, CJ) |
| `mockPhotographers` | data | `src/data/mock.ts` | 4 mock photographers |
| `mockServices` | data | `src/data/mock.ts` | 4 service tiers (¥399-¥1299) |

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

## COMMANDS
```bash
npm run dev:h5            # H5 dev server (port 5173)
npm run build:h5          # H5 production → dist/build/h5
npm run dev:mp-weixin     # WeChat MP dev → dist/dev/mp-weixin (watch mode)
npm run build:mp-weixin   # WeChat MP production → dist/build/mp-weixin
```

WeChat DevTools → import `dist/build/mp-weixin`. No lint/test scripts yet. To type-check: `npx vue-tsc --noEmit`.

## NOTES

- **No `.gitignore` at root** — add before git init.
- **Dark neon homepage** — uses `$dark-*` and `$neon-*` variables from variables.scss; other pages still use light theme
- **Backend mock** — `server/mock_server.go` is a Go standard library mock (port 8081); `server/cmd/api/main.go` is the full Gin+PostgreSQL version
- **Pinia stores** — `useHomeStore` handles homepage data with API→mock fallback; `useUserStore`/`useChatStore` exist but not yet consumed
- **uview-plus CSS active, JS layer disabled on mp-weixin** — path conflict (`node-modules` vs `node_modules`). Native components (swiper, image) used instead. uview-plus theme/variables still working.
- **Vant Weapp installed** (`@vant/weapp`) — ready for mp-weixin once npm build is configured in WeChat DevTools.
- **All API is mock** — `api/index.ts` returns hardcoded data. `getHomeData()` provides mock fallback when backend unavailable.
- **TabBar icons are 81×81 PNG** — generated from Material Symbols via sharp. Grey (inactive) / `#6366f1` (active).
- **No subPackages** — all pages in main package. Split if app grows.
- **No login/auth gate** — booking/chat/order pages have no auth check.
- **manifest.json `vueVersion: "3"`** — confirmed correct for Vue 3 project.
- **DiceBear avatars** — cute cartoon style, colorful backgrounds per user. Reliable SVG, no timeout issues.
- **picsum.photos covers** — may timeout occasionally inside WeChat sandbox; CSS gradient fallbacks on image containers.
