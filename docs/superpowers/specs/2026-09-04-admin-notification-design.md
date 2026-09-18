# P2c 后台通知公告管理设计

**日期:** 2026-09-04
**状态:** 已批准设计（待实现）
**关联:** P2c admin web（父模块）、P3 消息中心通知真实化（通知链路已建）

## 目标

管理端主动推送通知公告：管理员发布全员/指定用户公告 → C 端消息页收到。这是 P3 真实现通知链路的运营延伸——P3 只有订单事件触发的系统自动通知（预约成功/取消等），缺管理员主动广播。

**非目标（YAGNI）**：撤回/删除公告、定时推送、富文本、已读率统计（read 为展示性字段）、全员目标分页（单条广播即可）。

## 数据模型

`notifications.user_id` 改为可空（migration 000016）：
```sql
ALTER TABLE notifications ALTER COLUMN user_id DROP NOT NULL;
```
- `user_id IS NULL` 表示**全员广播**。
- **C 端查询改动（1 处）**：`ListNotifications` SQL `WHERE user_id = $1` → `WHERE (user_id = $1 OR user_id IS NULL)`（用户收到个人通知 + 全员广播）。P3 消息页前端零改动（仍调 `GET /v1/notifications/:userId`）。

## 管理 API（2 个，AdminAuthRequired，前缀 /api/admin/v1）

| Method | Path | 行为 |
|--------|------|------|
| POST | `/admin/v1/notifications` | `{type,title,content,targetType:'all'\|'single',userId?}` → 201 `{id}`；all → 写 user_id NULL；single → userId 必填（缺省 400）|
| GET | `/admin/v1/notifications` | 历史公告列表（id/type/title/content/targetType/userId/createdAt，`?page=&pageSize=` 分页，最新在前）|

- `type` 仅允许 `success`/`info`/`warning`（否则 400）。
- 复用 `notification_service.Create`（已有，签名 `Create(ctx, userID int64, typ, title, content string) (int64, error)`）——all 时传 userID=0？**不能**（0 会误写 user_id=0）——需要 service 加 `CreateBroadcast(ctx, typ, title, content)` 或 repo 加 `InsertBroadcast`（user_id NULL）。
- **决定**：repo 加 `InsertBroadcast(ctx, typ, title, content string) (int64, error)`（INSERT 不带 user_id，默认 NULL）；service 加 `Publish(ctx, req PublishRequest) (int64, error)`（校验 type/targetType/single 必填 userId）。

## 前端页面 `admin/views/notification/index.vue`

- 顶部：NCard「发布公告」表单（类型 NSelect：success 成功/info 提示/warning 警告；标题 NInput 必填；内容 NInput textarea 必填；目标 NRadio：全员/指定用户；指定用户时 NSelect 选用户手机号（从用户列表加载前缀 20 条或搜索））。
- NDataTable：历史公告（标题/类型 tag/目标（全员 or 用户名）/时间/分页——remote pagination）。
- 发布成功 → `$message` 成功 + 表单清空 + reload 列表。
- 复用 user/index.vue 模式；单根 `<div>`；浅色主题；中文 UI。
- 菜单：menuIcons 加 `notification: 'mdi:bell-outline'`；locale zh-cn `notification: '通知公告'` / en-us `Notification`；elegant-router 自动路由。

## API 层

`admin/src/service/api/admin.ts` 加：
- `publishNotification(payload: {type,title,content,targetType:'all'|'single',userId?})` → POST `/v1/notifications`
- `fetchNotifications(params)` → GET `/v1/notifications`，`?? {list:[],total:0}` 兜底

## 错误处理

- 401：token 缺失/无效（AdminAuthRequired 已有）。
- 400：type 非法 / targetType 非法 / single 缺 userId / title 或 content 缺失。
- 404：single 目标用户不存在（userId 无效）——service 校验 user 存在（查 users 表）；不存在 → 404。

## 测试策略

- 后端 `go test`：Publish single（userId 存在）/single 用户不存在 404/all 广播（InsertBroadcast 被调）/type 非法 400/targetType 非法 400；C 端 ListNotifications 广播可见（fake repo 或集成）。
- 前端 `pnpm typecheck`（green gate）。
- E2E：管理端登录 → 发布公告页 → 发全员公告 → C 端登录（任意用户）→ 消息页看到该公告 + 系统通知；发指定用户公告（另一用户看不到）；类型 tag/目标显示正确。
- agent-browser 视觉复查：发布表单 + 历史列表页浅色专业风。

## 时间线（3 Tasks）

1. 后端：000016 user_id 可空 + repo InsertBroadcast + C 端 ListNotifications 条件 + service Publish + 2 管理 API → 冒烟（含 C 端收到广播）+ push
2. 前端：notification/index.vue + api 函数 + 菜单图标/locale → typecheck + 浏览器跑通 + push
3. 验证 + E2E（双端）+ agent-browser 视觉复查 + AGENTS.md 补章节 + push

## 后续（不在本次）

- 公告撤回/删除
- 定时排期投放
- 富文本公告
- 已读率统计（read 标记升级为真实已读 API）
