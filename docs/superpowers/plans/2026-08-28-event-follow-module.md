# 关注漫展模块完善 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让"关注漫展"成为真实、完整、可感知的 C 端功能——绑定登录用户、详情页加关注入口、个人中心加"我的关注"列表。

**Architecture:** 前端纯改动（后端 follows API 已完整）——抽共享 follow utils（loadFollowedIds/toggleFollow），日历页改绑登录用户，详情页加关注按钮，新增关注列表页 + 个人中心入口。

**Tech Stack:** Vue 3 script setup + uni-app（src/）；后端 Gin 零改动。

## Global Constraints

- Vue 3 script setup lang="ts" only；TypeScript strict，禁新增 `as any`
- SCSS 变量/rpx/`@/` 别名；无 emoji 图标（几何符号 ◆★◇☆ ✓ 允许）
- 暗色霓虹主题：`$dark-bg-primary`/`$dark-bg-card`/`$neon-purple`/`$neon-cyan`/`$dark-text-*`/`$spacing-*`/`$font-size-*`
- 后端零改动（follows API 已齐：POST /v1/follows {userId, eventId}、DELETE /v1/follows/:userId/:eventId、GET /v1/follows/:userId → [{eventId,name,location,venue,startDate,endDate,coverUrl,status}]）
- 登录守卫：未登录点关注 → toast 请先登录 + navigateTo login?redirect=原页
- 用户 id 来源：`useUserStore().user.id`（string）
- 每个任务 commit；不 push（编排者统一推 Gitee `git -c http.proxy= -c https.proxy= push gitee master`）

---

## 文件结构总览

| 文件 | 职责 |
|------|------|
| `src/utils/follow.ts` | 共享关注逻辑：loadFollowedIds / toggleFollow |
| `src/pages/calendar/index.vue` | USER_ID demo → 登录用户；用共享函数 |
| `src/pages/event/detail.vue` | 封面区加关注按钮 + 初始化判断 |
| `src/pages/follow/list.vue` | 新页面：我的关注漫展列表 |
| `src/pages.json` | 注册 pages/follow/list |
| `src/pages/profile/index.vue` | 菜单加"关注的漫展"入口 |

---

### Task 1: 共享关注逻辑 utils + 日历页绑登录用户

**Files:**
- Create: `src/utils/follow.ts`
- Modify: `src/pages/calendar/index.vue`

**Interfaces:**
- Produces:
  - `loadFollowedIds(userId: string): Promise<Set<string>>` — GET /v1/follows/:userId → Set(String(eventId))
  - `toggleFollow(userId: string, eventId: string, followed: boolean): Promise<boolean>` — followed ? DELETE : POST；返回新状态；失败 throw（调用方 toast）

- [ ] **Step 1: 创建 follow.ts**

```ts
import { apiGet, apiPost, apiDelete } from '@/api/client'

export async function loadFollowedIds(userId: string): Promise<Set<string>> {
  try {
    const res = await apiGet<{ list: Array<{ eventId: number }> }>(`/v1/follows/${userId}`)
    return new Set((res.list || []).map(f => String(f.eventId)))
  } catch {
    return new Set()
  }
}

export async function toggleFollow(userId: string, eventId: string, followed: boolean): Promise<boolean> {
  if (followed) {
    await apiDelete(`/v1/follows/${userId}/${eventId}`)
    return false
  }
  await apiPost('/v1/follows', { userId, eventId: Number(eventId) })
  return true
}
```

- [ ] **Step 2: 日历页改绑登录用户**

calendar/index.vue 修改：
- 删 `const USER_ID = 'demo'`
- import：`import { loadFollowedIds, toggleFollow } from '@/utils/follow'` + `import { useUserStore } from '@/stores/user'`，加 `const userStore = useUserStore()`
- `loadFollows()` 改：
```ts
async function loadFollows() {
  if (!userStore.user) { follows.value = new Set(); return }
  follows.value = await loadFollowedIds(userStore.user.id)
}
```
- `toggleFollow(id: string)` 改为使用共享函数（保留 toast + catch）：
```ts
async function toggleFollow(id: string) {
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/calendar/index' }), 800)
    return
  }
  try {
    const nowFollowed = follows.value.has(id)
    const newState = await toggleFollow(userStore.user.id, id, nowFollowed)
    if (newState) follows.value.add(id)
    else follows.value.delete(id)
    uni.showToast({ title: newState ? '关注成功' : '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}
```
- 重命名内部函数避免同名冲突（共享函数也叫 toggleFollow——日历页的改为 `onToggleFollow`，模板 `@click.stop="onToggleFollow(evt.id)"`）

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "calendar|follow" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/utils/follow.ts src/pages/calendar/index.vue
git commit -m "feat(follow): shared follow utils + calendar binds logged-in user (was demo)"
```

---

### Task 2: 漫展详情页关注按钮

**Files:**
- Modify: `src/pages/event/detail.vue`

**Interfaces:**
- Consumes: `loadFollowedIds`/`toggleFollow`（Task 1）；`useUserStore`
- Produces: `isFollowed` ref + `onToggleFollow()`；模板封面区关注按钮

- [ ] **Step 1: script 加关注状态**

```ts
import { loadFollowedIds, toggleFollow } from '@/utils/follow'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()
const isFollowed = ref(false)

async function initFollowState() {
  if (!userStore.user) { isFollowed.value = false; return }
  const set = await loadFollowedIds(userStore.user.id)
  isFollowed.value = set.has(id)   // id 是路由 eventId（string）
}

async function onToggleFollow() {
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/event/detail?id=' + id }), 800)
    return
  }
  try {
    const newState = await toggleFollow(userStore.user.id, id, isFollowed.value)
    isFollowed.value = newState
    uni.showToast({ title: newState ? '关注成功' : '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}
```

在 loadEvent 成功设置 event.value 后调用 `initFollowState()`。

- [ ] **Step 2: 模板加按钮（封面区 cover-status 下方）**

在 `cover-info` 块内 `cover-status` 之后加：

```html
          <view
            class="follow-btn"
            :class="{ followed: isFollowed }"
            @click.stop="onToggleFollow"
          >
            <text>{{ isFollowed ? '已关注' : '＋ 关注' }}</text>
          </view>
```

样式（复用暗色霓虹）：

```scss
.follow-btn {
  display: inline-flex;
  align-items: center;
  margin-top: $spacing-sm;
  padding: 8rpx 28rpx;
  font-size: 24rpx;
  color: #fff;
  background: rgba(168, 85, 247, 0.25);
  border: 1rpx solid $neon-purple;
  border-radius: $border-radius-xl;
  transition: all 0.2s;

  &.followed {
    background: $neon-gradient;
    border-color: transparent;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}
```

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "event/detail" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/pages/event/detail.vue
git commit -m "feat(event-detail): follow button on cover (bind logged-in user)"
```

---

### Task 3: 我的关注列表页 + 个人中心入口

**Files:**
- Create: `src/pages/follow/list.vue`
- Modify: `src/pages.json`
- Modify: `src/pages/profile/index.vue`

**Interfaces:**
- Consumes: `apiGet('/v1/follows/:userId')` → FollowItem[{eventId,name,location,venue,startDate,endDate,coverUrl,status}]；`apiDelete`；`useUserStore`
- Produces: follow/list.vue 渲染关注漫展（封面/名称/地点/日期/倒计时 T 天后），空状态，取消关注；onShow 刷新（同 message 页教训——TabBar 外的普通页也要 onShow 防返回不刷新）

- [ ] **Step 1: follow/list.vue**

```vue
<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">关注的漫展</text>
    </view>
    <view v-if="loading" class="empty-hint">加载中...</view>
    <view v-else-if="follows.length === 0" class="empty-state">
      <text class="empty-icon">◇</text>
      <text class="empty-text">还没有关注的漫展</text>
      <view class="btn-go" @click="goHome">去逛逛</view>
    </view>
    <view v-else class="list">
      <view v-for="f in follows" :key="f.eventId" class="follow-card" @click="goDetail(f.eventId)">
        <image class="cover" :src="f.coverUrl" mode="aspectFill" />
        <view class="info">
          <text class="name ellipsis">{{ f.name }}</text>
          <text class="meta">{{ f.location }} · {{ f.venue }}</text>
          <text class="date">{{ formatRange(f.startDate, f.endDate) }}</text>
          <text v-if="daysUntil(f.startDate) > 0" class="countdown">{{ daysUntil(f.startDate) }}天后开赛</text>
        </view>
        <text class="unfollow" @click.stop="unfollow(f.eventId)">取消</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const follows = ref<any[]>([])
const loading = ref(true)
const userStore = useUserStore()

onShow(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    loading.value = false
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/follows/${userStore.user.id}`)
    follows.value = res || []
  } catch {
    follows.value = []
  } finally {
    loading.value = false
  }
})

async function unfollow(eventId: number) {
  if (!userStore.user) return
  try {
    await apiDelete(`/v1/follows/${userStore.user.id}/${eventId}`)
    follows.value = follows.value.filter(f => f.eventId !== eventId)
    uni.showToast({ title: '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}

function formatRange(start: string, end: string): string {
  const s = new Date(start)
  const e = new Date(end)
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  return `${s.getFullYear()}-${pad(s.getMonth() + 1)}-${pad(s.getDate())} 至 ${pad(e.getMonth() + 1)}-${pad(e.getDate())}`
}

function daysUntil(start: string): number {
  const ms = new Date(start).setHours(0, 0, 0, 0) - new Date().setHours(0, 0, 0, 0)
  return Math.max(0, Math.ceil(ms / 86400000))
}

function goDetail(eventId: number) {
  uni.navigateTo({ url: `/pages/event/detail?id=${eventId}` })
}

function goHome() {
  uni.switchTab({ url: '/pages/index/index' })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.list { padding: 100rpx 32rpx 32rpx; }
.follow-card { display: flex; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx; margin-bottom: 20rpx; }
.cover { width: 160rpx; height: 120rpx; border-radius: 12rpx; background: $dark-bg-secondary; flex-shrink: 0; }
.info { flex: 1; margin-left: 20rpx; overflow: hidden; }
.name { display: block; font-size: 28rpx; color: $dark-text-primary; font-weight: 600; }
.meta { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 6rpx; }
.date { display: block; font-size: 22rpx; color: $dark-text-secondary; margin-top: 6rpx; }
.countdown { display: inline-block; font-size: 20rpx; color: $neon-cyan; margin-top: 8rpx; }
.unfollow { font-size: 24rpx; color: $neon-purple; padding: 8rpx 16rpx; align-self: center; }
.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $dark-text-tertiary; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0; }
.btn-go { padding: 16rpx 48rpx; background: $neon-gradient; color: #fff; border-radius: 32rpx; font-size: 28rpx; }
.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }
</style>
```

- [ ] **Step 2: pages.json 注册**

```json
{
  "path": "pages/follow/list",
  "style": {
    "navigationStyle": "custom"
  }
}
```

- [ ] **Step 3: profile 菜单加入口**

profile/index.vue 在"我的收藏"菜单项后加：

```html
        <view class="menu-item" @click="goFollows">
          <text class="menu-marker">◇</text>
          <text class="menu-text">关注的漫展</text>
          <text class="menu-arrow">›</text>
        </view>
```

加函数：

```ts
function goFollows() {
  uni.navigateTo({ url: '/pages/follow/list' })
}
```

- [ ] **Step 4: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "follow|profile" | head -5`
Expected: 无新错误

- [ ] **Step 5: Commit**

```bash
git add src/pages/follow/list.vue src/pages.json src/pages/profile/index.vue
git commit -m "feat(follow): my follows page + profile entry"
```

---

### Task 4: 全链路验证 + 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Playwright H5 全链路**

登录（13800138000/123456 + 勾协议）→ 漫展详情页（如 /pages/event/detail?id=1）点关注（验证按钮变"已关注"）→ 个人中心 → 关注的漫展（列表显示该漫展含倒计时）→ 取消关注（列表移除）→ 日历页关注状态同步（跟随同一 Set 来源）。

- [ ] **Step 2: 后端 API 回归**

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
# 关注 → 列表 → 取消 全链路
curl -s -o /dev/null -w "follow: %{http_code}\n" -X POST http://127.0.0.1:8080/api/v1/follows -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"userId":"1","eventId":1}'
curl -s http://127.0.0.1:8080/api/v1/follows/1 -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('follows:', [f['eventId'] for f in d.get('list',[])])"
curl -s -o /dev/null -w "unfollow: %{http_code}\n" -X DELETE http://127.0.0.1:8080/api/v1/follows/1/1 -H "Authorization: Bearer $TOKEN"
```
Expected: follow 200 / list 含 1 / unfollow 204

- [ ] **Step 3: 回归**

Run: `cd server && go test ./... 2>&1 | tail -3`（全绿）；`npx vue-tsc --noEmit 2>&1 | grep -cE "error"`（≤1 pre-existing）；`npm run build:h5 2>&1 | tail -2`（构建成功）

- [ ] **Step 4: 视觉复查（用户授权）**

agent-browser 截图：详情页关注按钮（未关注/已关注两态）、我的关注列表页（有数据/空态）、个人中心菜单（"关注的漫展"入口）。发现问题列 task 纠正。

- [ ] **Step 5: AGENTS.md 更新**

新增"关注漫展模块"小节：前端 utils/follow.ts 共享逻辑、详情页关注按钮、follow/list 页、日历页绑登录用户（不再是 demo）；后端 follows API 零改动说明。

- [ ] **Step 6: 提交推送**

```bash
git add AGENTS.md
git commit -m "docs: AGENTS.md — event follow module"
git -c http.proxy= -c https.proxy= push gitee master
```

---

## 自审记录

- **Spec 覆盖**: 3.1 绑登录用户 → Task 1；3.2 详情页按钮 → Task 2；3.3 列表页 + 入口 → Task 3；M4 验证 → Task 4。spec 的错误处理/测试策略均覆盖。
- **占位符**: 无；每任务含完整代码/命令。
- **类型一致**: `loadFollowedIds(userId: string): Promise<Set<string>>` / `toggleFollow(userId: string, eventId: string, followed: boolean): Promise<boolean>` Task 1 定义，Task 2 使用；FollowItem 字段名（eventId/name/location/venue/startDate/endDate/coverUrl/status）与后端 JSON 一致。
- **已知取舍**: 日历页原 toggleFollow 重命名为 onToggleFollow（避免与共享函数同名）；follow/list 用 onShow（防返回不刷新，吸取 message 页教训）；后端零改动。
