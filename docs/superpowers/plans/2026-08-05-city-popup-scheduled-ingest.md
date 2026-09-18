# City Popup Fix + Event Title Ellipsis + Scheduled Ingest + City Tags

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 2 UI bugs (city popup scroll, event title ellipsis), add scheduled event ingestion from nyato + Bilibili, and make city list dynamic from DB events.

**Architecture:** Frontend CSS fixes in index.vue + dynamic city list from store; backend ingest extended with cron scheduling + Bilibili source; city tags derived from event location.

**Tech Stack:** Vue 3, SCSS dark-neon, Go 1.22, Gin, pgx, PostgreSQL, cron (robfig/cron/v3 or time.Ticker).

## Global Constraints

- Vue 3 `<script setup lang="ts">`, SCSS dark-neon variables only, rpx units
- `@/` import alias, no `any`
- Go: no panic, errors wrapped `%w`, handlers 404/400/500
- Ingest upsert idempotent via allcpp_id = FNV(name+city)
- No picsum for events — real images only
- Go env: GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=$GOROOT/bin:$PATH GOPROXY=https://goproxy.cn,direct

---

### Task 1: Fix city popup scrolling

**Files:**
- Modify: `src/pages/index/index.vue:746-751` (.city-popup) and `:773-775` (.city-list)

- [ ] **Step 1: Add max-height + flex to .city-popup**

```scss
.city-popup {
  width: 100%;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  background: $dark-bg-secondary;
  border-radius: $border-radius-lg $border-radius-lg 0 0;
  padding-bottom: env(safe-area-inset-bottom);
}
```

- [ ] **Step 2: Make .city-list scrollable**

```scss
.city-list {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-sm 0;
}
```

- [ ] **Step 3: Verify**

```bash
# Restart frontend (pkill node uni; relaunch) then in browser:
# Open homepage → click city selector → popup shows → scroll city list with ≥8 cities
```

---

### Task 2: Fix event title ellipsis

**Files:**
- Modify: `src/pages/index/index.vue:641-645` (.event-name)

- [ ] **Step 1: Make .event-name a block with ellipsis**

```scss
.event-name {
  display: block;
  font-size: 26rpx;
  font-weight: 600;
  color: $dark-text-primary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

- [ ] **Step 2: Check similar inline-text issues elsewhere**

```bash
grep -rn "class=\"event-name\|class=\"work-title\|class=\"chat-name" src/pages/ --include="*.vue" | head -10
# For each found <text> that should truncate: ensure the class has display:block + ellipsis rules
```

- [ ] **Step 3: Verify**

```bash
# Long event name (e.g. 成都第二十四届世界线动漫展) shows ... in 280rpx card
```

---

### Task 3: Add cron scheduling to ingest

**Files:**
- Modify: `server/cmd/ingest/main.go` — add `-schedule` flag + daily loop

- [ ] **Step 1: Add -schedule flag**

```go
var (
    baseURL  = flag.String("url", "https://www.nyato.com/manzhan", "nyato event list URL")
    dsn      = flag.String("dsn", "", "postgres DSN (default: env DB_* vars)")
    pages    = flag.Int("pages", 1, "number of pages to crawl")
    schedule = flag.String("schedule", "", "cron expression (e.g. \"0 3 * * *\") for daily ingest; empty = run once")
)
```

- [ ] **Step 2: Implement scheduled loop**

```go
// After flag.Parse():
if *schedule != "" {
    // Simple daily loop: parse "0 3 * * *" → run at 03:00 daily
    for {
        runOnce() // fetch + upsert, error-tolerant
        next := nextRunTime(*schedule) // compute next 03:00
        time.Sleep(time.Until(next))
    }
} else {
    runOnce()
}
```

Implement `nextRunTime(spec string) time.Time` — support only `"0 H * * *"` daily format (parse hour H, compute next occurrence; if other spec, error out). Keep it simple — no full cron parser dependency.

- [ ] **Step 3: Build**

```bash
cd server && GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=$GOROOT/bin:$PATH GOPROXY=https://goproxy.cn,direct go build ./cmd/ingest
```

- [ ] **Step 4: Verify scheduling math with a test**

```go
// server/internal/ingest/schedule_test.go
func TestNextRunTime(t *testing.T) {
    // now = 2026-08-05 10:00 → "0 3 * * *" → 2026-08-06 03:00
    // now = 2026-08-05 02:00 → "0 3 * * *" → 2026-08-05 03:00
}
```

---

### Task 4: Add Bilibili event source (best-effort)

**Files:**
- Create: `server/internal/ingest/bilibili.go`
- Modify: `server/cmd/ingest/main.go` — fetch from Bilibili when flag set

- [ ] **Step 1: Implement best-effort Bilibili fetch**

```go
package ingest

// FetchBilibiliEvents scrapes Bilibili's anime convention calendar page.
// NOTE: Bilibili has anti-scraping; this is best-effort. If the page
// structure changes or fetch fails, return empty slice + error (caller
// continues with nyato data — never fatal).
func FetchBilibiliEvents() ([]EventCard, error) {
    // GET https://www.bilibili.com/ (漫展/活动 section)
    // Parse for convention-like entries: name + city + date + image
    // Return []EventCard (reuse struct). On any parse failure: return nil, nil (skip source)
}
```

- [ ] **Step 2: Wire into main.go runOnce**

```go
// After nyato upsert, if *bilibili flag true:
cards, err := ingest.FetchBilibiliEvents()
if err != nil || len(cards) == 0 { log.Printf("bilibili: skipped (%v)", err); return }
for _, c := range cards { upsert(c) }
```

- [ ] **Step 3: Verify**

```bash
cd server && go build ./cmd/ingest && DB_HOST=localhost DB_PORT=5433 DB_USER=comic DB_PASSWORD=comic123 DB_NAME=comic \
  go run ./cmd/ingest -bilibili
# Expect: bilibili skipped (parse fail or empty) OR new events upserted — either acceptable, must not crash
```

---

### Task 5: Dynamic city list from DB events

**Files:**
- Modify: `src/pages/index/index.vue` — cityList from store
- Modify: `src/stores/home.ts` — expose `cityOptions` computed

- [ ] **Step 1: Add cityOptions computed to home store**

```typescript
// src/stores/home.ts
const cityOptions = computed(() => {
  const cities = new Set<string>(['全部'])
  events.value.forEach(e => { if (e.location) cities.add(e.location) })
  return Array.from(cities)
})
```

- [ ] **Step 2: Use it in index.vue**

```typescript
// Remove hardcoded cityList const; use:
import { storeToRefs } from 'pinia'
const { cityOptions } = storeToRefs(store)
// template: v-for="city in cityOptions"
```

- [ ] **Step 3: Verify**

```bash
# DB has events in 上海/广州/成都/杭州/西安/北京 + 13 ingested cities → city popup lists them all
```

---

### Task 6: City tag association

**Files:**
- Modify: `server/internal/ingest/main.go` — append city to tags on upsert
- Modify: `server/internal/db/query/events.sql` — include city tag in ingest upsert

- [ ] **Step 1: Append city as tag**

```go
// In cmd/ingest upsert loop, before calling UpsertEventFromIngest:
tags := []string{}
if card.City != "" {
    city := strings.TrimSuffix(card.City, "市")
    tags = append(tags, city) // e.g. "西安市" → "西安"
}
// pass tags in params
```

- [ ] **Step 2: Verify**

```bash
docker exec comic-pg psql -U comic -d comic -c "SELECT name, tags FROM comic_events WHERE source_url IS NOT NULL LIMIT 3;"
# Expect: tags contain city name (e.g. 乌兰察布·橙子猫… tags = {乌兰察布})
```

---

### Task 7: Final verification + commit + push

- [ ] **Go tests**: `cd server && go test ./internal/... -count=1` → PASS
- [ ] **Frontend type-check**: `npx vue-tsc --noEmit` → no new errors
- [ ] **City popup scrolls** in browser with 8+ cities
- [ ] **Event title ellipsis** shows `...` for long names
- [ ] **Scheduled ingest**: `go run ./cmd/ingest -schedule="0 3 * * *"` starts and waits (verify via log "next run at")
- [ ] **City tags** present on ingested rows
- [ ] Commit + `unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY && git -c http.proxy= push gitee master`
