# Layer 1: Backend Error Taxonomy + Event List Page + Mock Sync

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backend: distinguish 404/400/500 in all handlers. Frontend: new event list page with city/status filters, wire navigation from homepage. Sync mock data with real DB data.

**Architecture:** Go handlers check sentinel errors (`pgx.ErrNoRows` → 404, `strconv.ParseInt` fail → 400). New Vue page `event/list.vue` follows `photographer/list.vue` dark theme pattern. Mock data (`api/index.ts`, `main.go mockHomeHandler`) updated to match PG seed.

**Tech Stack:** Go 1.22+ Gin pgx, Vue 3 TypeScript Pinia SCSS dark-neon variables.

## Global Constraints

- Vue 3 `<script setup lang="ts">` only, rpx units, `@/` imports
- SCSS dark-neon variables only (`$dark-bg-primary`, `$dark-bg-card`, `$dark-border`, `$neon-purple`, etc.)
- No emoji, no hardcoded colors (except `#fff` on gradients)
- Go handler patterns: `c.JSON(http.StatusXXX, gin.H{"error": msg})`
- Go service pattern: return `error`, handler maps to HTTP code
- `pages.json` must register new page

---

### Task 1: Backend — Add sentinel errors and HTTP status mapping

**Files:**
- Modify: `server/internal/handler/event_handler.go:55-58`
- Modify: `server/internal/handler/home_handler.go` (add `pgx.ErrNoRows` import)
- Modify: `server/internal/handler/photographer_handler.go` (add error check)
- Create: `server/internal/handler/errors.go`

**Interfaces:**
- Produces: `func isNotFound(err error) bool` — exported helper for all handlers

- [ ] **Step 1: Create `errors.go` with `isNotFound` helper**

```go
// server/internal/handler/errors.go
package handler

import (
	"errors"
	"github.com/jackc/pgx/v5"
)

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
```

- [ ] **Step 2: Fix `event_handler.go` Detail — return 404 on not found**

Current code at line 55-58:
```go
data, err := h.svc.GetDetail(c.Request.Context(), id)
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
}
```

Replace with:
```go
data, err := h.svc.GetDetail(c.Request.Context(), id)
if err != nil {
    if isNotFound(err) {
        c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
    }
    return
}
```

- [ ] **Step 3: Fix `event_handler.go` List — same pattern**

Add `if isNotFound... http.StatusNotFound` before `http.StatusInternalServerError`.

- [ ] **Step 4: Fix `home_handler.go` GetHome**

```go
data, err := h.svc.GetHomeData(c.Request.Context())
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
    return
}
```

(Home always 500 since DB failure is the only error case — keep as-is but sanitize error message.)

- [ ] **Step 5: Fix `photographer_handler.go` Detail**

Same pattern as event_handler: check `isNotFound`, return 404 if true, else 500.

- [ ] **Step 6: Fix `photographer_handler.go` List**

Same: 500 with sanitized message.

- [ ] **Step 7: Fix `booking_handler.go` Create and ListByUser**

Same pattern: 500 with sanitized message.

- [ ] **Step 8: Fix `review_handler.go` Create and ListByPhotographer**

Same pattern.

- [ ] **Step 9: Fix `tag_handler.go` GetAll and `auth_handler.go` Login**

Same pattern: 500 with sanitized message.

- [ ] **Step 10: Build verification**

```bash
cd server && go build -o /dev/null ./cmd/api 2>&1
# Expected: no errors
```

---

### Task 2: Frontend — New Event List Page

**Files:**
- Create: `src/pages/event/list.vue`
- Modify: `src/pages.json` (add `pages/event/list` entry)
- Modify: `src/pages/index/index.vue:95` (wire "更多 ›" to event list)
- Modify: `src/pages/index/index.vue:47` (CTA "近期热门漫展" to event list)

**Interfaces:**
- Consumes: `GET /api/v1/events?status=upcoming&page=1&size=10`
- Produces: Navigation target for `goEventList()` and `section-more` click

- [ ] **Step 1: Create `src/pages/event/list.vue`**

Template pattern — follow `photographer/list.vue` dark theme layout:
```vue
<template>
  <view class="page">
    <view class="filter-bar">
      <scroll-view scroll-x class="filter-scroll" :show-scrollbar="false">
        <view class="filter-list">
          <view v-for="s in statusTabs" :key="s.key" class="filter-item"
            :class="{ active: currentStatus === s.key }" @click="setStatus(s.key)">
            {{ s.label }}
          </view>
        </view>
      </scroll-view>
    </view>
    <scroll-view scroll-y class="content" @scrolltolower="loadMore">
      <view class="event-list">
        <view v-for="e in events" :key="e.id" class="event-card" @click="goDetail(e.id)">
          <image :src="e.cover" class="event-cover" mode="aspectFill" />
          <view class="event-badge" :class="e.status">{{ statusLabel(e.status) }}</view>
          <view class="event-info">
            <text class="event-name ellipsis">{{ e.name }}</text>
            <text class="event-location">{{ e.location }} · {{ e.venue }}</text>
            <text class="event-date">{{ formatDate(e.startDate) }} - {{ formatDate(e.endDate) }}</text>
          </view>
        </view>
      </view>
      <view v-if="loading" class="loading">加载中...</view>
      <view v-if="!loading && events.length >= total" class="no-more">没有更多了</view>
      <view class="bottom-space" />
    </scroll-view>
  </view>
</template>
```

Script — minimal, follow patterns:
```typescript
<script setup lang="ts">
import { ref } from 'vue'
import type { ComicEvent } from '@/types'

const events = ref<ComicEvent[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const currentStatus = ref('upcoming')

const statusTabs = [
  { key: 'upcoming', label: '即将开始' },
  { key: 'ongoing', label: '进行中' },
  { key: 'ended', label: '已结束' },
]

function statusLabel(s: string): string {
  const m: Record<string, string> = { upcoming: '即将', ongoing: '进行中', ended: '已结束' }
  return m[s] || s
}

function formatDate(ts: number): string {
  const d = new Date(ts)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

async function loadEvents() {
  loading.value = true
  try {
    const res = await uni.request({
      url: `/api/v1/events?status=${currentStatus.value}&page=${page.value}&size=10`,
      method: 'GET',
      timeout: 5000,
    })
    if (res.statusCode === 200) {
      const data = res.data as { list: ComicEvent[]; total: number }
      if (page.value === 1) events.value = data.list
      else events.value = [...events.value, ...data.list]
      total.value = data.total
    }
  } catch { /* ignore, keep empty */ }
  loading.value = false
}

function setStatus(s: string) {
  currentStatus.value = s
  page.value = 1
  events.value = []
  loadEvents()
}

function loadMore() {
  if (loading.value || events.value.length >= total.value) return
  page.value++
  loadEvents()
}

function goDetail(id: string) {
  uni.navigateTo({ url: `/pages/event/detail?id=${id}` })
}

loadEvents()
</script>
```

Style — dark neon only:
```scss
<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.filter-bar { position: fixed; top: 0; left: 0; right: 0; background: $dark-bg-secondary;
  z-index: 100; padding-top: env(safe-area-inset-top); border-bottom: 1rpx solid $dark-border; }
.filter-scroll { white-space: nowrap; }
.filter-list { display: inline-flex; padding: $spacing-sm $spacing-md; gap: $spacing-sm; }
.filter-item { font-size: $font-size-sm; color: $dark-text-secondary; padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-xl; border: 1rpx solid transparent; transition: all 0.15s;
  &.active { background: rgba($neon-purple,0.15); color: $neon-purple; border-color: rgba($neon-purple,0.3); }
  &:active { background: $dark-bg-card-hover; }
}
.content { height: 100vh; padding-top: calc(env(safe-area-inset-top) + 80rpx); }
.event-list { padding: $spacing-md; display: flex; flex-direction: column; gap: $spacing-md; }
.event-card { position: relative; background: $dark-bg-card; border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg; overflow: hidden; transition: transform 0.15s;
  &:active { transform: scale(0.97); border-color: $neon-purple-glow; }
}
.event-cover { width: 100%; height: 320rpx; background: linear-gradient(135deg,$neon-purple-dim,rgba(6,182,212,0.1)); }
.event-badge { position: absolute; top: 12rpx; right: 12rpx; padding: 4rpx 14rpx; border-radius: $border-radius-sm;
  font-size: 20rpx; font-weight: 600;
  &.upcoming { background: rgba(0,0,0,0.7); color: $neon-cyan; }
  &.ongoing { background: rgba(0,0,0,0.7); color: $success-color; }
  &.ended { background: rgba(0,0,0,0.7); color: $dark-text-tertiary; }
}
.event-info { padding: $spacing-sm; }
.event-name { display: block; font-size: 28rpx; font-weight: 600; color: $dark-text-primary; }
.event-location { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 4rpx; }
.event-date { display: block; font-size: 22rpx; color: $neon-cyan; font-weight: 500; margin-top: 4rpx; }
.loading, .no-more { text-align: center; padding: $spacing-md; font-size: $font-size-sm; color: $dark-text-tertiary; }
.bottom-space { height: 120rpx; }
</style>
```

- [ ] **Step 2: Register in `pages.json`**

Add before `pages/search/search` entry:
```json
{
  "path": "pages/event/list",
  "style": { "navigationBarTitleText": "漫展列表" }
},
```

- [ ] **Step 3: Wire homepage navigation**

In `src/pages/index/index.vue`:
- Line 95 `goEventList` → change from `goEventDetail(upcoming[0].id)` to `uni.navigateTo({ url: '/pages/event/list' })`
- Line 47 CTA "近期热门漫展" stays as `goEventList()` (both go to list now)

- [ ] **Step 4: Verify build**

```bash
cd /vol1/1000/code/comic && npx vue-tsc --noEmit 2>&1 | grep -v "event/detail\|pre-existing"
```

---

### Task 3: Sync mock data with real DB

**Files:**
- Modify: `src/api/index.ts:119-161` (getHomeData — update banner titles)
- Modify: `server/cmd/api/main.go:36-72` (mockHomeHandler — update banner titles + linkId)

**Interfaces:**
- Consumes: PG seed data (8 events — ChinaJoy, 萤火虫, CP33, CCG, 世界线, 梦乡, IJOY, COMICUP)
- Produces: Mock responses match DB responses

- [ ] **Step 1: Update `main.go` mockHomeHandler banners**

```go
"banners": []gin.H{
    {"id": 1, "imageUrl": "https://picsum.photos/seed/cj2026/750/360", "title": "ChinaJoy 2026", "linkType": "event", "linkId": 1},
    {"id": 2, "imageUrl": "https://picsum.photos/seed/firefly40/750/360", "title": "第40届萤火虫漫展", "linkType": "event", "linkId": 2},
    {"id": 3, "imageUrl": "https://picsum.photos/seed/cp33/750/360", "title": "CP33 综合同人展", "linkType": "event", "linkId": 3},
},
```

- [ ] **Step 2: Update `main.go` mockHomeHandler events**

Replace old event names (上海 CP30, 成都 CD28, 广州萤火虫, 北京 IDO42, 杭州 CJ漫展) with real ones:
```go
"upcomingEvents": []gin.H{
    {"id": 1, "name": "ChinaJoy 2026", "location": "上海", "venue": "上海新国际博览中心", "startDate": "2026-08-01T00:00:00+08:00", "endDate": "2026-08-04T00:00:00+08:00", ...},
    // ... all 8 events matching PG seed
},
```

- [ ] **Step 3: Update `api/index.ts` getHomeData banners**

```typescript
banners: [
  { id: 1, imageUrl: 'https://picsum.photos/seed/cj2026/750/360', title: 'ChinaJoy 2026', linkType: 'event', linkId: 1 },
  { id: 2, imageUrl: 'https://picsum.photos/seed/firefly40/750/360', title: '第40届萤火虫漫展', linkType: 'event', linkId: 2 },
  { id: 3, imageUrl: 'https://picsum.photos/seed/cp33/750/360', title: 'CP33 综合同人展', linkType: 'event', linkId: 3 },
],
```

---

### Task 4: Verification

- [ ] **Backend**: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/events/99999` → 404
- [ ] **Backend**: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/events/1` → 200
- [ ] **Frontend**: `curl -s http://localhost:5173/#/pages/event/list` → page renders
- [ ] **Navigation**: Home "更多 ›" → event list page loads with 8 events
- [ ] **Banner click**: Home banner ChinaJoy → event/detail shows ChinaJoy (not COMICUP)
