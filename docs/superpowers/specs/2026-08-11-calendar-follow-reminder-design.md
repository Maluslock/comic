# 漫展日历页 + 关注 + 开赛提醒 设计

**日期:** 2026-08-11
**状态:** 已批准 (用户确认: 自绘日历 / 后端关注表 / 订阅消息预留)

## 背景

首页"漫展日历"需要一个独立页面（非搜索页）。用户类比"奶茶小程序点单"：关注漫展 = 下单，开赛提醒 = 取餐通知。

## 设计

### 页面：`pages/calendar/index`（新建）

```
┌──────────────────────────────┐
│  ‹  2026年8月  ›             │ ← 月份切换（左右箭头）
│  一 二 三 四 五 六 日         │ ← 星期头
│  1  2  3  4  5  6  7        │ ← 日期格（有漫展 → 霓虹圆点）
│  ...                        │ ← 今天高亮/霓虹描边
├──────────────────────────────┤
│ 8月14日 · 2场漫展             │ ← 选中日期事件面板
│ ┌──────────────────────────┐ │
│ │ [封面] 第40届萤火虫漫展    │ │ ← 事件卡片
│ │ 广州·保利世贸 8/14-17      │ │
│ │        [关注提醒]         │ │ ← 关注按钮
│ └──────────────────────────┘ │
└──────────────────────────────┘
```

### 功能

1. **自绘月视图日历**
   - 6×7 网格，星期头（一~日），月份切换（‹ 2026年8月 ›）
   - 有漫展的日期显示霓虹圆点（紫/青），今天霓虹描边高亮
   - 点击日期 → 下方显示当天漫展列表

2. **关注漫展（后端表）**
   - DB 新表 `event_follows`：id, user_id, event_id, created_at, UNIQUE(user_id, event_id)
   - API：
     - `POST /api/v1/follows` body `{userId, eventId}` — 关注
     - `DELETE /api/v1/follows/:userId/:eventId` — 取消关注
     - `GET /api/v1/follows/:userId` — 我的关注列表
   - 前端关注按钮 → 调 API → storage 缓存状态；已关注卡片显示"已关注 · X天后开赛"

3. **开赛提醒（预留）**
   - 前端封装 `src/utils/subscribe.ts`：`subscribeEventReminder(event)` 调 `uni.requestSubscribeMessage`（模板 ID 常量占位，demo 阶段空实现 + console.log）
   - 后端预留 `POST /api/v1/subscribe` handler（存订阅记录表 `event_subscriptions`：id, user_id, event_id, template_id, status, created_at）
   - 正式版只需填 AppID + 模板 ID 即可启用

### 导航接入

- 首页 CTA "近期热门漫展" → `/pages/calendar/index`
- 首页 "近期漫展 更多›" → `/pages/calendar/index`
- `/pages/event/list` 保留

### 数据流

```
GET /api/v1/events?status=upcoming → 前端按月分组 → 日历打点
点击日期 → 过滤当天事件 → 事件卡片
关注 → POST /api/v1/follows → storage 同步
提醒 → subscribe.ts 预留 → 后端 subscribe 表
```

## 文件清单

| 文件 | 改动 |
|------|------|
| `server/migrations/000006_follows_subscriptions.up.sql` | 新建：event_follows + event_subscriptions 表 |
| `server/internal/repository/` + `db/query/` | follows/subscriptions 查询 |
| `server/internal/service/follow_service.go` | 关注业务 |
| `server/internal/handler/follow_handler.go` | POST/DELETE/GET follows + POST subscribe |
| `server/cmd/api/main.go` | 注册路由 |
| `src/pages/calendar/index.vue` | 新建：自绘日历 + 事件面板 |
| `src/utils/subscribe.ts` | 订阅消息封装（预留） |
| `src/pages/index/index.vue` | 导航改到 calendar |
| `src/pages.json` | 注册 calendar 页 |

## 验收

1. 日历月视图正确渲染（当月 + 前后月切换），有漫展日期打点
2. 点击日期显示当天漫展，点击卡片跳详情
3. 关注/取消关注 → DB 持久化，刷新后状态保留
4. 已关注事件显示倒计时
5. 首页 CTA 跳日历页（非搜索页）
