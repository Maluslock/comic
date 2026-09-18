# 米拉漫展 — Next Phase Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete remaining 20% for production-ready mini-program: polish, auth, order flow, .gitignore, Gitee push, QA signoff.

**Architecture:** Frontend Vue 3 + Pinia dark-neon theme (all 13 pages), Backend Gin+PG with 10 REST endpoints + Redis cache layer, Docker compose for full stack.

**Tech Stack:** Vue 3, TypeScript, Pinia, Vite, UniApp, Go 1.22, Gin, pgx, PostgreSQL 16, Redis 7, SCSS dark-neon variables.

## Global Constraints

- Vue 3 `<script setup lang="ts">` only, Composition API, no Options API
- SCSS dark-neon variables only (`$dark-bg-*`, `$dark-text-*`, `$neon-*`), no hardcoded colors (except `#fff` on gradient)
- rpx units for all spacing/sizing, no px/rem
- `@/` import alias always, no relative imports across `src/`
- No emoji — use SVG icons in `/static/icons/` or geometric text symbols (★ ✕ → ›)
- TypeScript strict mode, no `any`, no `@ts-ignore`
- Go handlers: return proper HTTP codes (404 for not-found, 400 for invalid-input, 500 sanitized)
- `uni.request` only in `src/api/client.ts` — all pages use `apiGet`/`apiPost` from there
- DTO mapping uses `src/utils/mappers.ts` shared functions
- Pages registered in `src/pages.json` before use

---

### Task 1: Add `.gitignore` and project hygiene

**Files:**
- Create: `.gitignore`

- [ ] **Step 1: Create `.gitignore`**

```
node_modules/
dist/
.env
*.log
.env.local
.DS_Store
Thumbs.db
*.swp
*.swo
*~
.tmp/
tmp/
```

- [ ] **Step 2: Verify git status clean**

```bash
cd /vol1/1000/code/comic && git status
# Expected: no untracked node_modules or dist
```

---

### Task 2: Fix remaining UI polish items from QA

**Files:**
- Modify: `src/pages/photographer/detail.vue` — add :active to .service-item, .footer-btn, .work-item
- Modify: `src/pages/message/index.vue` — add :active to .notification-item, .chat-item
- Modify: `src/pages/order/list.vue` — add :active to .order-card, .empty-btn, .btn-outline, .btn-primary
- Modify: `src/pages/order/detail.vue` — add :active to .btn-outline, .btn-primary
- Modify: `src/pages/booking/index.vue` — add :active to .date-picker, .btn-primary
- Modify: `src/pages/chat/index.vue` — add :active to .send-btn, .back-btn
- Modify: `src/pages/portfolio/index.vue` — add :active to .gallery-item

MUST DO:
- Each :active gets `.scale(0.97)` + enhanced glow border matching existing patterns
- Dark neon styling: `box-shadow: 0 0 12rpx $neon-purple-glow` on active
- $dark-bg-card-hover background on press

---

### Task 3: Complete order fulfillment flow

**Files:**
- Modify: `src/pages/order/list.vue` — add "确认完成" button, persist status changes
- Modify: `src/pages/order/detail.vue` — add cancel/confirm actions, wire to storage
- Modify: `src/pages/booking/index.vue` — ensure photographerId flows correctly from detail page

MUST DO:
- Cancel booking: set status='cancelled', persist to storage, update UI
- Confirm complete: set status='completed', persist to storage, update UI
- All status changes survive page reload (storage-backed)
- Verify flow: photographer detail → booking → submit → order list → order detail → cancel/confirm

---

### Task 4: Start Redis and add caching verification

**Files:**
- Modify: `docker-compose.yml` (ensure Redis service runs)
- Run: Docker Redis container

- [ ] **Step 1: Start Redis**

```bash
docker run -d --name comic-redis -p 6379:6379 redis:7-alpine redis-server --appendonly yes
```

- [ ] **Step 2: Verify Redis connected**

```bash
curl -s http://localhost:8080/api/v1/home | head -1
# Check API logs: should show "caching enabled"
tail -3 /tmp/comic-api.log
```

---

### Task 5: Gitee push

**Files:**
- Modify: `.git/config` — add Gitee remote

- [ ] **Step 1: Add Gitee remote**

```bash
cd /vol1/1000/code/comic
git remote add gitee http://Haxlock:Ly2024jy!@100.64.0.16:8418/Haxlock/mila-comic.git
```

- [ ] **Step 2: Stage all files (excluding node_modules/dist/.env)**

```bash
echo "node_modules/" >> .gitignore
echo "dist/" >> .gitignore
echo ".env" >> .gitignore
git add -A
git status  # Verify no unwanted files
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: sitewide dark neon theme + Gin backend + event list + mapper layer + API client + Go tests"
```

- [ ] **Step 4: Push to Gitee**

```bash
git push -u gitee master --force
```

---

### Task 6: Final verification sweep

- [ ] **Go tests pass**: `cd server && go test ./internal/handler/ -v`
- [ ] **Frontend builds**: `npm run build:h5` (or at least vue-tsc --noEmit)
- [ ] **Backend runs**: `curl http://localhost:8080/health` → `{"status":"ok"}`
- [ ] **Event list loads**: `curl http://localhost:8080/api/v1/events` → 8 events
- [ ] **404 works**: `curl http://localhost:8080/api/v1/events/99999` → HTTP 404
- [ ] **Image URLs load**: grep `neeko-copilot` or `dicebear` → 0 matches
- [ ] **No uni.request outside client.ts**: grep `uni.request` minus client.ts → 0 matches
- [ ] **No err.Error() in handlers**: grep `err.Error()` in handlers → 0 matches