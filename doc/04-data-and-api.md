# 04 数据库设计与 API 参考

> 本文面向接手同事，说明米拉漫展后端的数据结构与对外接口。
> 数据层：PostgreSQL（19 张表，迁移目录 `server/migrations/`）+ Redis 缓存。
> 接口两套：C 端 `/api/v1/*` 与后台 `/api/admin/v1/*`，认证体系完全隔离。
> 模块与页面的对应关系见 `03-modules.md`。

更新时间：2026-09-11 ｜ 后端入口：`server/cmd/api/main.go`（:8080）

---

## 一、数据库概览

- **数据库**：PostgreSQL，本地端口 `5433`（见 `server/internal/config`）。
- **迁移工具**：编号迁移，文件名形如 `0000NN_描述.up.sql` / `.down.sql`，共 17 个迁移（`000001`~`000017`）。
- **表总数**：19 张。
- **缓存**：Redis（`:6379`），best-effort，不可用不影响启动。
- **另一形态**：`server/mock_server.go` 是 std-lib 写的 mock 服务（`:8081`），与真实 Gin 后端分离，仅用于纯前端演示。

### 迁移清单

| 编号 | 文件 | 内容 |
|------|------|------|
| 000001 | `create_tables` | 初始建表：`comic_events`、`banners`、`photographers`、`works`、`tags`、`photographer_tags`、`services`、`reviews`、`bookings` |
| 000002 | `seed` | 初始演示种子数据 |
| 000003 | `real_event_images` | 漫展真实封面图 |
| 000004 | `event_ingest_columns` | `comic_events` 增加 `address` / `image_gallery` / `source_url` / `synced_at` |
| 000005 | `event_del_flag` | `comic_events` 增加 `del_flag` 软删标记 + 部分索引 |
| 000006 | `follows_subscriptions` | 新增 `event_follows`（关注漫展）、`event_subscriptions`（开赛提醒订阅，预留） |
| 000007 | `users_tokens` | 新增 `users`、`user_tokens` |
| 000008 | `favorites_chat` | 新增 `photographer_favorites`、`chat_sessions`、`chat_messages` |
| 000009 | `photographer_user_link` | `photographers` 增加 `user_id`，挂接用户身份 |
| 000010 | `photographer_activation` | `photographers` 增加 `mode` / `mutual_intro` / `certified` / `activated_at` |
| 000011 | `notifications` | 新增 `notifications` 表 |
| 000012 | `admins` | 新增 `admins` 表 + 种子管理员 `admin` |
| 000013 | `users_status` | `users` 增加 `status`（active / disabled） |
| 000014 | `cert_applications` | 新增 `photographer_cert_applications`（含部分唯一索引） |
| 000015 | `admins_status` | `admins` 增加 `status` |
| 000016 | `notification_broadcast` | `notifications.user_id` 改为可空（NULL = 全员广播） |
| 000017 | `works_status` | `works` 增加 `status`（active / down） |

---

## 二、数据表说明

### 业务核心

| 表名 | 用途 | 关键列 |
|------|------|--------|
| `comic_events` | 漫展，抓取同步而来 | `id`、`allcpp_id`(源唯一 ID)、`name`、`location`、`venue`、`start_date`/`end_date`、`cover_url`、`tags TEXT[]`、`type_name`、`status`、`address`、`image_gallery TEXT[]`、`source_url`、`del_flag`(软删)、`synced_at`；索引 `(status, start_date)` |
| `banners` | 首页轮播图，管理端维护 | `image_url`、`title`、`link_type`(当前仅 `event`)、`link_id`(指向 `comic_events.id`)、`sort_order`、`is_active` |
| `photographers` | 摄影师（双角色身份层） | `id`、`user_id`(挂接 users)、`name`、`avatar`、`description`、`location`、`rating`、`review_count`、`order_count`、`mode`(free / pay / both)、`mutual_intro`(互勉说明)、`certified`(黄V)、`activated_at` |
| `works` | 摄影师作品 | `id`、`photographer_id`、`title`、`images TEXT[]`、`description`、`status`(active / down) |
| `services` | 服务档位（全局目录） | `id`、`name`、`price`、`description`、`duration` |
| `reviews` | 评价 | `id`、`photographer_id`、`user_id`、`user_name`、`user_avatar`、`rating`(1-5 CHECK)、`content`、`images TEXT[]` |
| `bookings` | 预约订单 | `id`、`photographer_id`、`coser_id`、`service_id`、`date`、`time`、`status`(pending / confirmed / completed / cancelled)、`total_price`、`remarks` |
| `tags` | 风格标签 | `id`、`name`(唯一)、`usage_count` |
| `photographer_tags` | 摄影师与标签多对多 | `photographer_id`、`tag_id`（复合主键） |

### 用户与社交

| 表名 | 用途 | 关键列 |
|------|------|--------|
| `users` | 用户账号 | `id`、`phone`(唯一)、`name`、`avatar`、`status`(active / disabled) |
| `user_tokens` | C 端会话 token | `token`(主键)、`user_id`(FK, 级联删除)、`expires_at` |
| `photographer_favorites` | 收藏摄影师 | `user_id`、`photographer_id`、`UNIQUE(user_id, photographer_id)` |
| `event_follows` | 关注漫展 | `user_id VARCHAR`、`event_id`、`UNIQUE(user_id, event_id)` |
| `event_subscriptions` | 开赛提醒订阅（演示预留） | `user_id`、`event_id`、`template_id`、`status`、`UNIQUE(user_id, event_id)` |
| `chat_sessions` | 单聊会话（规范 `user1_id < user2_id`） | `id`、`user1_id`、`user2_id`、`updated_at`、`UNIQUE(user1_id, user2_id)` |
| `chat_messages` | 会话消息 | `id`、`session_id`(FK, 级联删除)、`sender_id`、`content` |
| `notifications` | 系统通知 / 公告 | `id`、`user_id`(可空，NULL = 全员广播)、`type`(success / info / warning)、`title`、`content`、`read` |

### 管理与认证

| 表名 | 用途 | 关键列 |
|------|------|--------|
| `admins` | 后台管理员账号 | `id`、`username`(唯一)、`password_hash`(bcrypt)、`role`(admin / super)、`token`、`token_expires_at`、`status`(active / disabled) |
| `photographer_cert_applications` | 认证申请 | `id`、`user_id`、`photographer_id`、`evidence_images TEXT[]`、`evidence_desc`、`status`(pending / approved / rejected)、`review_reason`、`admin_id`(FK)、`reviewed_at`；部分唯一索引 `(photographer_id) WHERE status <> 'rejected'` |

> **一摄影师一活跃申请**由 `photographer_cert_applications` 的部分唯一索引硬保证：并发重提会命中 23505 并映射成 409「already applied」，`rejected` 后可重新申请（新行）。

---

## 三、API 参考：C 端 `/api/v1`

认证列「是」表示需请求头 `Authorization: Bearer <user token>`。

### 首页 / 漫展

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | `/home` | 首页聚合：轮播 + 近期漫展 + 热门标签 + 推荐摄影师 + 精选作品（Redis 缓存） | 否 |
| GET | `/events` | 漫展列表（可按城市 / 状态筛选） | 否 |
| GET | `/events/:id` | 漫展详情 | 否 |

### 摄影师

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | `/photographers` | 摄影师列表（关键词 / 标签 / 城市筛选） | 否 |
| GET | `/photographers/:id` | 摄影师详情（含作品 / 服务 / 评价） | 否 |
| GET | `/photographers/:id/timeslots?date=YYYY-MM-DD` | 该日已占用时段，返回 `{ occupied: [...] }` | 否 |
| POST | `/photographers/activate` | 开通摄影师（幂等，重复开通返回原 id） | 是 |
| GET | `/photographers/by-user/:userId` | 按用户查摄影师身份（未开通 404） | 是 |
| PUT | `/photographers/profile` | 全量更新主页（昵称 / 简介 / 城市 / 接单模式 / 互勉说明） | 是 |
| GET | `/photographers/profile/mine` | 我的摄影师主页 | 是 |
| POST | `/photographers/cert-apply` | 提交认证申请（非摄影师 404，已有活跃申请 409） | 是 |
| GET | `/photographers/cert-application` | 我的最新一条申请（精简 DTO；无记录返回 `application:null`） | 是 |
| POST | `/photographers/works` | 发布作品（title + 至少 1 张图） | 是 |
| GET | `/photographers/works/mine` | 我的作品列表（含下架，仅自己可见） | 是 |
| DELETE | `/photographers/works/:id` | 删除作品（归属校验，非本人 403） | 是 |

### 订单 / 评价

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/bookings` | 创建预约（同摄影师 / 日期 / 时段冲突 409） | 否 |
| PUT | `/bookings/:id/status` | 状态流转，body 带 `actorTag`（`coser` 默认 / `photographer`）；跨角色越权 403 | 是 |
| GET | `/bookings/:userId` | 用户订单列表 | 否 |
| GET | `/bookings/photographer/:photographerId` | 摄影师订单面板列表（含 coser join 字段） | 是 |
| POST | `/reviews` | 发表评价 | 否 |
| GET | `/reviews/:photographerId` | 摄影师评价列表 | 否 |

### 认证

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/login` | 手机号 + 验证码登录，返回 token | 否 |
| GET | `/me` | 当前用户（含 `photographerId`，未开通则不含） | 是 |
| PUT | `/me/profile` | 更新昵称 / 头像（avatar 为空保留原值） | 是 |

### 收藏 / 聊天 / 标签 / 关注 / 通知

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/favorites` | 收藏摄影师 | 是 |
| DELETE | `/favorites/:userId/:photographerId` | 取消收藏 | 是 |
| GET | `/favorites/:userId` | 收藏列表（含摄影师 join 字段） | 是 |
| POST | `/chat/sessions` | 获取或创建会话，返回 `{ id }` | 是 |
| GET | `/chat/sessions/:userId` | 会话列表（含对端信息 + 最后一条消息） | 是 |
| GET | `/chat/messages/:sessionId` | 消息列表（升序） | 是 |
| POST | `/chat/messages` | 发送消息 | 是 |
| GET | `/tags` | 标签列表 | 否 |
| POST | `/follows` | 关注漫展 | 否 |
| DELETE | `/follows/:userId/:eventId` | 取消关注 | 否 |
| GET | `/follows/:userId` | 关注列表，返回对象 `{ list: [...] }`（非裸数组） | 否 |
| POST | `/subscribe` | 订阅开赛提醒（演示预留） | 否 |
| POST | `/notifications` | 创建通知 | 是 |
| GET | `/notifications/:userId` | 通知列表（含广播行，裸数组，最新优先，LIMIT 20） | 是 |

---

## 四、API 参考：后台 `/api/admin/v1`

除 `POST /login` 外，所有接口需请求头 `Authorization: Bearer <admin token>`。

### 登录 / 工作台

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/login` | 管理员登录，签发并轮换 token（旧 token 立即失效） |
| GET | `/dashboard` | 指标：`totals` + `trends`（7 天）+ `ordersByStatus` |

### 摄影师 / 订单 / 漫展

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/photographers` | 摄影师列表，`?certified=true|false` 可选过滤 |
| PUT | `/photographers/:id/certified` | 人工切换认证，body `{certified: bool}` |
| GET | `/orders` | 分页订单列表，`?status=&page=&pageSize=`，含 join 字段 |
| PUT | `/orders/:id/status` | 状态机流转（跳过身份校验，保留状态机；非法流转 409） |
| GET | `/events` | 全部漫展（含 `delFlag`） |
| PUT | `/events/:id/status` | 上架 / 下架，body `{delFlag: bool}` |

### 用户 / 认证审核

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/users` | 用户列表，`?keyword=&role=&status=&page=&pageSize=` |
| GET | `/users/:id` | 用户详情 + `stats` + `recentBookings` |
| PUT | `/users/:id/status` | 封禁 / 启用，body `{status:"active"|"disabled"}` |
| GET | `/cert-applications` | 申请列表，`?status=pending|approved|rejected&page=&pageSize=` |
| GET | `/cert-applications/:id` | 申请详情（证据图 / 驳回理由 / 审核时间） |
| PUT | `/cert-applications/:id/review` | 审批，body `{action:"approve"|"reject", reason?}`；reject 必填 reason；非 pending 复审 409 |

### 管理员 / 通知公告 / 轮播图

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admins` | 管理员列表（DTO 已剥离 passwordHash） |
| POST | `/admins` | 新建管理员，body `{username, password, role}`；重名 409 |
| PUT | `/admins/:id/status` | 禁用 / 启用；id=1 禁用 400 |
| PUT | `/admins/:id/password` | 重置密码；重置后该账号所有已登录会话 401 |
| POST | `/notifications` | 发布通知，body `{type, title, content, targetType, userId?}`；全员写 NULL 广播行 |
| GET | `/notifications` | 通知历史，`?page=&pageSize=`，`targetType` 由 userId 推导 |
| GET | `/banners` | 轮播图列表（按 `sort_order` 升序） |
| POST | `/banners` | 新增轮播，`link_type` 仅支持 `event` |
| PUT | `/banners/:id` | 全字段更新 |
| PUT | `/banners/:id/status` | 上下线，body `{isActive: bool}` |

### 内容管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/works` | 作品分页，`?photographerId=&status=&page=&pageSize=` |
| PUT | `/works/:id/status` | 上 / 下架，body `{status:"active"|"down"}`；同时失效 home 缓存 |
| GET | `/reviews` | 评论分页，`?keyword=&photographerId=`（keyword 模糊匹配内容 / 用户名 / 摄影师名） |
| DELETE | `/reviews/:id` | 硬删评论 |
| GET | `/tags` | 标签列表，`usageCount` 为真实关联统计 |
| POST | `/tags` | 新增标签，重名 409 |
| PUT | `/tags/:id` | 重命名标签，重名 409 |
| DELETE | `/tags/:id` | 删除标签；被引用（`photographer_tags` count>0）400 |

---

## 五、认证说明

- **C 端**：`POST /api/v1/login` 校验手机号 + 验证码，写入/返回 `user_tokens` 行。前端 `src/api/client.ts` 从本地存储读 token，自动注入 `Authorization: Bearer <token>`；收到 401 时跳转登录页。
- **管理端**：独立 `admins` 表，`POST /api/admin/v1/login` 用 bcrypt 校验后签发并轮换 token；`AdminAuthRequired` 中间件校验。
- **两套隔离**：C 端 token 与 admin token 存在不同表、走不同中间件。C 端 token 访问 admin 路由 401，admin token 访问 C 端受保护路由同样不认。
- **禁用即时生效**：`users.status='disabled'` 时 `auth_service.Login` 在签发前拦截并返回 403；`admins.status='disabled'` 时 `GetAdminByToken` 带条件查询，已签发 token 立即 401；重置管理员密码会清空 token，强制该账号所有会话登出。
- **已知简化**：follows / chat / notifications 等接口信任客户端传入的 `userId`，服务端未做交叉校验；`bookings` 摄影师侧操作通过 `actorTag` + 身份比对（403）做越权防护。

---

## 六、缓存与数据抓取

- **缓存**：Redis 为 best-effort。首页数据缓存，TTL 5 分钟；管理端下架 / 恢复作品时调用 `cache.Delete("home")` 立即失效，保证 C 端可见性与管理端一致。
- **抓取调度**（`server/config/cron.yaml`，由 `cmd/ingest -config` 注册）：

| 任务 | 调度 | 来源 | 动作 |
|------|------|------|------|
| `nyato-daily-full` | `0 3 * * *`（03:00） | nyato | 全量抓取 5 页并 upsert |
| `nyato-afternoon-incremental` | `30 14 * * *`（14:30） | nyato | 增量抓取 2 页 |
| `expire-cleanup` | `0 1 * * *`（01:00） | expire | 过期漫展标记 `del_flag=true` |
| `bilibili-sync` | 禁用 | bilibili | 预留来源（空实现） |

- **抓取注意事项**：nyato.com 拒绝 Go HTTP/2（报 `stream error: INTERNAL_ERROR`），`internal/ingest/nyato.go` 强制 HTTP/1.1 + 浏览器 UA / Referer，不要改回 HTTP/2。抓取的文本入库前做 UTF-8 清洗（防 SQLSTATE 22021）。
- **演示路径**：`go run ./cmd/ingest -pages 5` 跑一次全量；`go run ./cmd/ingest -config config/cron.yaml` 启动 cron 常驻。

---

## 七、快速上手命令

```bash
# 后端 API（需 Go 1.22+，PG :5433，Redis :6379）
go run ./cmd/api

# 无数据库 mock 模式
USE_MOCK=true go run ./cmd/api

# C 端 H5 / 微信小程序
npm run dev:h5
npm run dev:mp-weixin

# 后台管理（开发端口 5174，勿占用 5173）
cd admin && pnpm dev --port 5174
```
