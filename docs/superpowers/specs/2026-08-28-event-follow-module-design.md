# 关注漫展模块完善 — 设计文档

**日期**: 2026-08-28
**状态**: 设计定稿（用户确认"尝试完善此功能模块"）
**范围**: 米拉漫展小程序 — 让"关注漫展"成为真实、完整、可感知的 C 端功能（绑定登录用户 + 详情页入口 + 我的关注列表）

## 1. 背景与问题

视觉审阅 + 代码盘点发现关注功能目前的三大缺陷：

1. **USER_ID 硬编码 'demo'**（`src/pages/calendar/index.vue:145`）——所有关注数据挂在假用户 'demo' 上，登录用户关注了漫展但**不是自己的关注**；换账号互不可见；订阅提醒无法正确绑定。
2. **无自然入口**——关注按钮只藏在日历页"点选日期后的事件面板"里，用户看完漫展详情无法关注（最自然的场景缺失，`event/detail.vue` 无关注按钮）。
3. **无回顾入口**——个人中心无"关注的漫展"入口，关注了之后没有任何地方能看到自己关注的漫展列表（对照：摄影师收藏已有"我的收藏"入口）。

## 2. 目标

1. **绑定登录用户**：日历页关注逻辑改用 `userStore.user.id`，关注数据持久化到各自账号
2. **详情页入口**：漫展详情页添加"关注"按钮（封面区，最常用场景）
3. **我的关注列表**：个人中心菜单加"关注的漫展"入口 + 列表页（展示关注过的漫展，带开赛倒计时/日期）

**明确不做**（YAGNI）：微信订阅消息真推送（AppID 未就绪，后端表/API 已预留）；关注通知；关注数统计。

## 3. 架构决策

### 3.1 绑定登录用户（后端 API 无需改，前端改）

`event_follows` 表已有（`user_id VARCHAR(100)`，`event_id`，UNIQUE(user_id, event_id)），
`/v1/follows` 三接口已有（POST/DELETE/GET）。**只需前端把 `USER_ID = 'demo'` 换成登录用户 id**。

统一策略：**登录守卫**——未登录点击"关注"→ toast + 跳登录（redirect 回原页）；已登录用 `userStore.user.id`。

### 3.2 漫展详情页关注按钮

`src/pages/event/detail.vue` 封面区（`cover-info` 上方或旁）添加关注按钮：
- 状态：`已关注`（实心样式）/ `关注`（描边样式），复用现有暗色霓虹风格（`$neon-purple` + glow）
- 交互：点击 → 登录守卫 → POST/DELETE `/v1/follows` → 本地状态翻转 + toast
- 初始化：onLoad 后拉取 `/v1/follows/:userId` 判断是否已关注（复用日历页 loadFollows 逻辑，抽成共享函数）

### 3.3 我的关注列表页

- 新增 `src/pages/follow/list.vue`：关注漫展列表（复用事件卡样式：封面/名称/地点/日期/开赛倒计时 T 天后），空状态（"还没有关注的漫展" + 去逛逛），取消关注按钮
- 数据源：`GET /v1/follows/:userId` 返回 `[{eventId, name, location, venue, startDate, endDate, coverUrl, status}]`（后端已 JOIN comic_events，字段齐全）
- 注册 `pages/follow/list` 到 pages.json（navigationStyle: custom，同 favorite/list 模式）
- 个人中心 `profile/index.vue`：菜单加"关注的漫展"项（菱形图标 ◆，与收藏/订单风格一致）→ navigateTo follow/list

### 3.4 共享关注逻辑抽取

为避免日历/详情/列表三处重复，抽 `src/utils/follow.ts`：
- `loadFollowedIds(userId): Promise<Set<string>>` — GET /v1/follows/:userId → Set(eventId)
- `toggleFollow(userId, eventId, followed): Promise<boolean>` — POST/DELETE，返回新状态
- `requireLogin(): boolean` — 复用 requireAuth 模式（已存在于 utils/auth.ts）

## 4. 数据流

```
日历页/详情页:
  初始化 → loadFollowedIds(userStore.user.id) → Set → 渲染 关注/已关注 状态
  点击关注 → toggleFollow(userId, eventId, !followed)
    → POST/DELETE /v1/follows → 本地 Set 更新 + toast

我的关注列表:
  onShow → GET /v1/follows/:userId → 渲染列表（倒计时 = startDate - now）
  取消关注 → DELETE /v1/follows/:userId/:eventId → 移除该项
```

## 5. 错误处理

- 401 未登录 → client.ts 统一跳登录（已有）
- 未登录点击关注 → toast + navigateTo login?redirect=原页
- 网络失败 → toast "操作失败，请重试"
- 空状态 → "还没有关注的漫展" + 去逛逛按钮

## 6. 测试策略

- 前端: vue-tsc 类型检查（无新错误）+ Playwright H5 实测：
  1. 未登录点关注 → 跳登录
  2. 登录 → 详情页关注 → 状态变"已关注"
  3. 个人中心 → 关注的漫展 → 列表显示该漫展（含倒计时）
  4. 日历页关注状态与详情页同步（同一 Set 来源）
  5. 取消关注 → 列表移除
- 回归: 日历页关注功能仍工作（不再 demo 用户）；收藏/订单不受影响
- 后端: 无 API 改动（已有接口回归验证即可）

## 7. 里程碑

- M1: `src/utils/follow.ts` 共享逻辑 + 日历页绑登录用户
- M2: 漫展详情页关注按钮
- M3: follow/list.vue + 个人中心入口 + pages.json
- M4: 全链路验证 + 视觉复查 + AGENTS.md + 推送

## 8. 假设记录

1. 方向 = 完善关注漫展模块（用户确认）
2. 后端 follows API 已够用，零后端改动（除非发现 bug）
3. 微信订阅提醒后置（AppID 未就绪）
4. 视觉复查发现问题列 task 纠正（用户授权）
