# 全链路贯通：用户体系 + 真实数据接入 — 设计文档

**日期**: 2026-08-24
**状态**: 设计定稿（无人值守模式，假设用户认可推荐方案 A，可随时叫停调整）
**范围**: 米拉漫展小程序前后端 — 把 8 个"看起来有、实际假"的模块接到真实后端

## 1. 背景与问题

项目当前只有首页/漫展详情/日历接了真实后端（Gin+PG+Redis）。其余模块 UI 齐全但数据是 mock/硬编码：

| 模块 | 现状 | 问题 |
|------|------|------|
| 摄影师 list/detail | `api/index.ts` mock | 真实 API `/api/v1/photographers` 已存在但未使用 |
| 搜索 search | mock `getPhotographers/getTags` | 与真实 API 零连接 |
| 预约 booking | `photographerId` 硬编码 `'1'` + mock | 无真实摄影师/服务/时间槽 |
| 订单 list/detail | 纯 `mockOrders` 数组 | **无任何 API 调用** |
| 聊天 chat | 静态消息数组 | 无后端会话 |
| 消息 message | `notifications`/`sessions` 硬编码 | 无后端 |
| 评论 comment | `reviews: []` 空数组 | 后端 `/api/v1/reviews` 存在但未用 |
| 作品 portfolio | mock `getWorks()` | 后端 works API 存在但未用 |
| 登录 login | `userStore.login(mockUser)` 假登录 | 后端 `/api/v1/login` 存在但前端未调，无 token、无鉴权 |

**根因**: 前端 API 层 `api/index.ts` 提供 mock 函数，页面直接 import mock；`api/client.ts` 的真实 HTTP 客户端只被首页/日历使用。两套 API 并存导致页面"能跑但假"。

## 2. 目标

Demo 演示优先：让 App 跑通 **浏览漫展 → 浏览摄影师 → 登录 → 预约 → 查看订单** 完整链路，全部真实数据。视觉完整可演示。

**明确不做**（YAGNI）: 真实 WebSocket 聊天、短信验证码、支付、社交 feed、管理后台。

## 3. 架构决策

### 3.1 API 层统一

- **保留** `src/api/client.ts`（apiGet/apiPost/apiDelete）为唯一真实 HTTP 通道
- `src/api/index.ts` 的 mock 函数逐步废弃；页面全部改走 client.ts + Pinia store
- `client.ts` 增加：
  - 请求头注入 `Authorization: Bearer <token>`（从 storage 读取）
  - 401 统一处理（清 token + 跳登录）
  - 可选：请求/响应日志开关

### 3.2 用户体系（批 1 地基）

**后端**:
- 新增 `users` 表迁移（`000007_users.up.sql`）: id, phone UNIQUE, name, avatar, created_at
- `AuthService.Login` 改为: 验证码 `1234` 通过后 **upsert user by phone**，返回真实 user.ID（不再硬编码 1）
- 新增 token 中间件: `Authorization: Bearer <token>` → 查 token 表/内存 → 注入 `user_id` 到 context
- 新增 `GET /api/v1/me`（token 校验后返回当前用户）
- token 存储: 内存 map（demo 够用）或 `user_tokens` 表 —— 选内存 map + 过期时间（demo 简化，注明生产需换 JWT/Redis）

**前端**:
- `stores/user.ts`: 持久化 token + user 到 `uni.storage`；`login(phone, code)` 调真实 `/api/v1/login`；`logout()`；`isLoggedIn` computed
- `pages/login/index.vue`: 接真实登录 API，成功后跳转来源页
- 需要登录的操作（booking/order/follow/subscribe）做登录守卫

### 3.3 数据贯通（批 2）

| 模块 | 前端改动 | 后端配合 |
|------|---------|---------|
| photographer/list | 改调 `apiGet('/v1/photographers', {keyword, location, tag})` | 已有 List API（支持 keyword/location/tag/page/size） |
| photographer/detail | 改调 `apiGet('/v1/photographers/:id')` | **详情响应已含 services + works + reviews**，一次调用全得 |
| search | 改调真实 photographers + tags API | tags API 已有 |
| portfolio | 复用摄影师详情的 works 字段，或调详情 API | works 由 `/api/v1/photographers/:id` 返回，无独立端点 |
| booking | 路由参数传 photographerId；services 来自详情 API | services 在详情响应中；**timeSlots 后端无支撑**，demo 用前端生成 |
| order/list | 改调 `apiGet('/v1/bookings/:userId')` 删除 mockOrders | 已有 ListByUser |
| order/detail | 改调 `apiGet('/v1/bookings/:userId')` 过滤 | 同上 |
| comment | 复用详情 API 的 reviews 字段展示；提交走 POST | 已有 Create/ListByPhotographer |

### 3.4 简化版消息/聊天 + 上架预留（批 3）

- message: 接轻量通知（后端简单通知表 或 前端 mock 标注 demo）—— 选前端 mock + demo 标注，降低范围
- chat: 保持静态，demo 标注"即将上线"
- 上架预留: `manifest.json` AppID 占位、API 域名白名单注释、订阅消息模板 ID 占位

## 4. 数据流

```
页面 → Pinia store → api/client.ts (Bearer token) → Gin handler → service → repository → PG/Redis
```

登录守卫流程: 页面 onLoad 检查 `userStore.isLoggedIn` → 未登录 → `uni.navigateTo('/pages/login/index?redirect=...')` → 登录成功 → 跳回。

## 5. 错误处理

- API 层: 统一 401 → 清 token + 提示 + 跳登录；5xx → 提示"服务异常"；网络错误 → 提示重试
- 页面层: loading 状态（沿用现有 skeleton/loading 模式）；空数据 → empty-hint
- 登录失败: 验证码错误提示（后端 401）

## 6. 测试策略

- 后端: Go handler/service 测试（login upsert、token 中间件、photographers 过滤）
- 前端: `vue-tsc --noEmit` 类型检查 + Playwright H5 实测全链路（登录→预约→订单）
- 回归: 首页/漫展详情/日历不受影响

## 7. 里程碑（分批交付）

- **批 1（地基）**: users 表 + token 中间件 + /api/v1/me + 前端 token 持久化 + 真实登录页
- **批 2（贯通）**: photographer/list/detail + search + portfolio + booking + order + comment 全部接真实 API
- **批 3（收尾）**: 消息 demo 标注 + 聊天标注 + 上架预留 + 全链路验证

## 8. 假设记录（无人值守）

1. 目标 = Demo 演示优先（方案 A），非上架优先/功能深度优先
2. token 用内存 map（demo），生产需换 JWT
3. 验证码仍为 mock `1234`（无短信服务）
4. 聊天/消息保持简化，不做实时
5. 每个批次完成后提交推送 Gitee
