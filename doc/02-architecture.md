# 技术架构

这份文档讲清楚「系统由哪些部分组成、它们怎么连起来、关键设计为什么这么定」。改代码之前先读一遍，能少踩很多坑。涉及具体端点、字段、模块细节时，回到 `AGENTS.md` 按模块查。

---

## 1. 架构概述

系统是典型的三层结构，外加一个独立的 Web 管理端和一个数据采集程序：

```
       局域网 / 微信小程序
  ┌──────────────────┐        ┌──────────────────────┐
  │  C 端 uniapp      │        │  admin 后台管理端      │
  │  H5 开发 :5173    │        │  soybean-admin :5174  │
  │  暗色霓虹主题      │        │  浅色专业风            │
  │  (目标微信小程序)  │        │                       │
  └────────┬─────────┘        └───────────┬──────────┘
           │ /api/*                        │ /api/admin/*  (vite proxy)
           └──────────────┬────────────────┘
                          ▼
           ┌──────────────────────────────────────┐
           │        Go API  (Gin)  :8080            │
           │  /api/v1/*        (C 端接口)           │
           │  /api/admin/v1/*  (管理端接口)          │
           │  handler → service → repository         │
           └────────┬───────────────────┬──────────┘
                    ▼                   ▼
           ┌─────────────────┐   ┌─────────────────┐
           │  PostgreSQL      │   │  Redis           │
           │  comic-pg :5433  │   │  comic-redis :6379│
           └─────────────────┘   └─────────────────┘

           ┌───────────────────────┐
           │  ingest 定时采集程序    │──▶ 写入 comic_events 表
           │  nyato.com → cron.yaml │
           └───────────────────────┘
```

要点：

- **C 端和后台是两个完全独立的前端工程**，共享同一个 Go API，只是走不同的路由组（`/api/v1` 与 `/api/admin/v1`）。两边的账号体系、token、视觉风格都是分开的。
- **后端是唯一的数据出口**，前端从不直连数据库。
- **ingest 是独立进程**，和 API 分开部署、分开调度，只负责往 `comic_events` 表写数据。

---

## 2. 技术栈

### C 端（`src/`）

| 项 | 选型 |
|----|------|
| 框架 | uniapp (Vue 3.4 + `<script setup>` Composition API) |
| 语言 | TypeScript（strict 模式，不用 `any`） |
| 状态管理 | Pinia（composition store） |
| 构建 | Vite 5 + `@dcloudio/vite-plugin-uni` |
| UI 库 | uview-plus（**仅用 CSS 层**；mp-weixin 上 JS 组件有路径问题，用原生组件兜底）+ Vant Weapp（已装，待配置） |
| 目标平台 | H5（开发主用）、mp-weixin（微信小程序） |
| 主题 | 暗色霓虹：底色 `#0a0a1a`，主色紫 `#a855f7` + 青 `#06b6d4` |
| 样式 | SCSS，全部 rpx，变量集中在 `src/styles/variables.scss` |

### 后台管理端（`admin/`）

| 项 | 选型 |
|----|------|
| 模板 | soybean-admin v4+ 改造（pnpm monorepo） |
| 框架 | Vue 3.5 + Vite 8 |
| UI 库 | NaiveUI 2.44 + UnoCSS |
| 状态/路由 | Pinia + vue-router（elegant-router 自动生成路由） |
| 图表 | ECharts 6 |
| 主题 | 浅色专业风，仅启用 zh-CN |
| 开发端口 | 5174（5173 归 C 端） |

### 后端（`server/`）

| 项 | 选型 |
|----|------|
| 语言 | Go 1.22 |
| Web 框架 | Gin |
| 数据库驱动 | pgx v5（`pgxpool` 连接池） |
| 数据库 | PostgreSQL 16 |
| 缓存 | Redis 7 |
| SQL 代码生成 | sqlc（配置 `server/sqlc.yaml`，生成的 `*.sql.go` 在 repository 下） |
| 定时任务 | robfig/cron（ingest 的 `-config` 模式） |
| 分层 | handler → service → repository |

### 基础设施

| 服务 | 镜像 | 容器名 | 端口 |
|------|------|--------|------|
| PostgreSQL | postgres:16-alpine | `comic-pg` | 5433 → 5432 |
| Redis | redis:7-alpine | `comic-redis` | 6379 |

---

## 3. 目录结构说明

```
comic/
├── src/                    # C 端 uniapp 工程
│   ├── api/                # API 客户端：index.ts（业务接口）+ client.ts（uni.request 封装 + token 注入）
│   ├── components/         # 展示型组件（卡片、骨架屏等，全部暗色）
│   ├── data/mock.ts        # 后端不可用时的兜底 mock 数据
│   ├── pages/              # 页面（每个页面一个目录，index.vue）
│   ├── stores/             # Pinia：home / user / chat
│   ├── styles/             # variables.scss（设计 token）+ global.scss（工具类）
│   ├── types/index.ts      # 全部领域类型定义
│   ├── utils/              # 工具函数（如 follow.ts 关注状态）
│   ├── static/             # tabBar 图标、SVG 图标
│   ├── pages.json          # 页面注册 + tabBar + 导航栏配置
│   └── manifest.json       # 平台配置
│
├── admin/                  # 后台管理端（soybean-admin 改造）
│   ├── src/views/          # 各模块页面：dashboard / photographer / order / event /
│   │                       #   user / certification / admin-accounts / banner /
│   │                       #   notification / content
│   ├── src/service/api/    # 管理端 API 封装（admin.ts）
│   ├── src/store/          # 认证 store（token 存 localStorage）
│   ├── src/router/routes/  # 路由定义（elegant-router 自动）
│   └── .env                # VITE_ADMIN_API_BASE_URL=/api/admin/v1
│
├── server/                 # Go 后端
│   ├── cmd/api/            # API 服务入口
│   ├── cmd/ingest/         # 数据采集入口（cron.yaml 驱动）
│   ├── cmd/sync/           # 一次性同步入口
│   ├── config/cron.yaml    # 采集任务调度配置
│   ├── internal/
│   │   ├── handler/        # HTTP 层：解析请求、调 service、写响应
│   │   ├── service/        # 业务逻辑层
│   │   ├── repository/     # 数据访问层（sqlc 生成 + 手写 repo）
│   │   ├── middleware/     # auth.go（C 端）/ admin_auth.go（管理端）
│   │   ├── ingest/         # 抓取实现（nyato.go / bilibili.go / schedule.go）
│   │   ├── cache/          # Redis 封装
│   │   ├── config/         # 配置加载
│   │   └── db/query/       # sqlc 查询定义
│   ├── migrations/         # 17 个数据库迁移（000001 ~ 000017，up/down 成对）
│   ├── .env                # 本地配置（DB_PORT=5433 等）
│   ├── mock_server.go      # 遗留的 std-lib mock（:8081，仅供独立演示）
│   └── Makefile            # dev / build / test / migrate 快捷命令
│
├── docs/superpowers/
│   ├── plans/              # 20 份实现计划
│   └── specs/              # 19 份设计文档
│
├── doc/                    # 本交接文档目录
├── docker-compose.yml      # 容器编排参考
├── start.sh                # 一键启动脚本（部分功能，见 quickstart）
└── AGENTS.md               # 项目知识库（模块级细节都在这）
```

---

## 4. 认证体系

系统里有**两套完全隔离的 token 体系**，分别对应 C 端用户和管理员。它们用不同的表、不同的中间件、不同的登录接口，互不通用。

### C 端用户

| 层 | 位置 | 职责 |
|----|------|------|
| 数据表 | `users` + `user_tokens`（迁移 000007） | 用户身份 + 已签发的 token |
| 登录 | `POST /api/v1/login`（手机号 + 验证码） | 校验后签发 token |
| 中间件 | `server/internal/middleware/auth.go`（`AuthRequired`） | 校验 `Authorization: Bearer <token>`，把 `userID` 注入 context，无效返回 401 |
| 当前用户 | `GET /api/v1/me` | 返回 id / name / phone / avatar / `photographerId` |
| 前端 | `src/api/client.ts` | 从 `uni.getStorageSync('token')` 取 token 注入请求头；收到 401 跳登录页 |

登录细节：验证码**只校验前 4 位是否为 `1234`**，测试期填 `123456` 即可。`users.status` 为 `disabled` 时登录返回 403 `account disabled`。

### 管理员

| 层 | 位置 | 职责 |
|----|------|------|
| 数据表 | `admins`（迁移 000012 + 000015 加 status） | 用户名 / bcrypt 密码哈希 / token / 状态 |
| 登录 | `POST /api/admin/v1/login`（用户名 + 密码） | bcrypt 校验后签发并**轮换** token |
| 中间件 | `server/internal/middleware/admin_auth.go`（`AdminAuthRequired`） | 校验 admin token，与 C 端中间件独立 |
| 前端 | `admin/src/store/modules/auth/` | token 存 localStorage（key `SOY_token`） |

关键特性：

- **token 每次登录轮换**，旧 token 立即失效。
- **禁用即时生效**：`GetAdminByToken` 查询带 `AND status='active'`，被禁用的管理员已签发 token 立刻 401；登录接口也校验 status（与密码错误返回同一响应，防账号枚举）。
- **种子保护**：主管理员 id=1 不允许被禁用。
- **重置密码清 token**：重置后该账号所有会话强制登出。

### 隔离验证

用 C 端 token 访问 `/api/admin/v1/*` 会 401，用 admin token 访问 `/api/v1/*` 也会 401。两条链路各自独立，不会串。

### 双角色身份（闲鱼式）

一个用户既是 coser 也可以是摄影师，靠身份层和交易层分离实现：

- **身份层**：`photographers.user_id` 把摄影师身份挂到用户上（迁移 000009）。开通后 `/api/v1/me` 返回 `photographerId`，前端据此显示角色 badge、切换「我是摄影师 / 接单管理」入口。
- **交易层**：`bookings` 同时服务两边。更新订单状态时带 `actorTag`（`coser` 默认 / `photographer`），服务端校验调用者的 `photographerId` 是否等于订单的 `photographerId`，不符则 403。身份校验在状态机校验之前执行。

---

## 5. 数据流示例

### 首页

`GET /api/v1/home` 是读得最频繁的接口，链路是「缓存优先」：

```
C 端首页
  └─ GET /api/v1/home
       └─ Redis 查 key "home"（TTL 5 分钟）
            ├─ 命中 → 直接返回
            └─ 未命中 → PostgreSQL 聚合查询
                 ├─ banners（is_active=true, sort_order ASC, LIMIT 5）
                 ├─ upcomingEvents（未过期漫展）
                 ├─ hotTags
                 ├─ recommendedPhotographers
                 └─ featuredWorks（status='active'）
                     → 写回 Redis（key "home"）→ 返回
```

轮播图由管理端维护（`banners` 表），管理端改完实时驱动首页。作品下架时管理端会主动 `cache.Delete("home")`，所以不等 5 分钟 TTL 也会立即从首页精选消失。手动改库后要 `docker exec comic-redis redis-cli FLUSHALL` 才能看到变化。

### 订单状态机

`bookings` 的状态流转是受约束的，非法流转一律 409：

```
pending ──▶ confirmed ──▶ completed
   │            │
   └────────────┴──▶ cancelled
```

| 目标状态 | 允许的来源 |
|---------|-----------|
| `confirmed` | 仅 `pending` |
| `completed` | 仅 `confirmed` |
| `cancelled` | `pending` 或 `confirmed` |
| 其它（如 `completed → pending`） | 409 Invalid transition |

- 创建订单时，同一摄影师 + 同一天 + 同一时段若已有非取消订单，返回 409 冲突。
- `GET /api/v1/photographers/:id/timeslots?date=YYYY-MM-DD` 返回该摄影师当日被占用的时段，前端把已占时段置灰。

### 通知

订单生命周期事件会写通知，走「booking 事件 → notifications 表 → C 端消息页」：

```
预约创建 / 状态变更
  └─ booking_service.notify()
       └─ 写 notifications 表（写失败只记日志，不阻塞主流程）
            └─ C 端消息页 GET /api/v1/notifications/:userId
                 （按 created_at DESC LIMIT 20）
```

- 通知类型 `success` / `info` / `warning`，对应不同图标。
- 管理端可发**全员广播**（`user_id = NULL` 的行），查询条件 `WHERE user_id = $1 OR user_id IS NULL`，广播行对所有用户可见。
- 当前 `read` 标记只是前端展示，没有已读接口，下次进页面重置。

### 认证申请

```
C 端摄影师提交 POST /api/v1/photographers/cert-apply
  └─ 写 photographer_cert_applications（部分唯一索引保证一摄影师一活跃申请）
       └─ 管理端审批 PUT /api/admin/v1/cert-applications/:id/review
            ├─ approve → 事务（申请 approved + photographers.certified=true）+ 通知
            └─ reject  → 写 rejected + review_reason（必填）+ 通知
                 └─ C 端可重新申请（rejected 后可再提交）
```

`photographers.certified=true` 后，C 端摄影师主页显示「黄V」认证徽章。

### 其它已打通流程

收藏（`photographer_favorites`）、关注漫展（`event_follows`）、聊天（`chat_sessions` + `chat_messages`，进入页面轮询，无 WebSocket）、评价（`reviews`）都已在 C 端接入真实接口。细节见 `AGENTS.md` 对应小节。

---

## 6. 数据采集（ingest）

`server/cmd/ingest` 是一个独立进程，从第三方站点抓取漫展信息写入 `comic_events`。

### 数据源与任务

配置在 `server/config/cron.yaml`，用标准 5 段 cron 表达式：

| 任务 | 调度 | 数据源 | 行为 |
|------|------|--------|------|
| `nyato-daily-full` | `0 3 * * *`（每天 03:00） | nyato，5 页 | 全量抓取 + upsert |
| `nyato-afternoon-incremental` | `30 14 * * *`（每天 14:30） | nyato，2 页 | 日间增量补充 |
| `expire-cleanup` | `0 1 * * *`（每天 01:00） | expire | 把 `start_date < NOW()` 的漫展标记 `del_flag=true` |
| `bilibili-sync` | `0 4 * * *` | bilibili | `enabled: false`，best-effort，目前返回空 |

`enabled: false` 的任务启动时跳过。每次抓取顺带做过期清理。

### 运行方式

```bash
cd server
go run ./cmd/ingest                          # 单次抓 1 页
go run ./cmd/ingest -pages 5                 # 单次抓 5 页
go run ./cmd/ingest -config config/cron.yaml # cron 守护进程（推荐）
```

### 抓取要点

- **WAF 对抗**：nyato.com 会拒绝 Go 的 HTTP/2（报 `stream error: INTERNAL_ERROR`）。`internal/ingest/nyato.go` 强制走 HTTP/1.1 并伪装浏览器 UA/Referer。**不要回退这部分**。
- **文本清洗**：抓来的文本在写库前做 UTF-8 清洗（防 SQLSTATE 22021 编码错误）。
- 数据源用 hash 去重/判重。

---

## 7. 关键设计决策

### 视觉：两端两套风格

- **C 端暗色霓虹**：底色 `#0a0a1a`，紫青双色霓虹（`$neon-purple` / `$neon-cyan`），渐变 `$neon-gradient` 用于标题/CTA。规范集中在 `src/styles/variables.scss` 的 `$dark-*` / `$neon-*` 变量。换页时用对应的 `$dark-*` 替换浅色变量，禁止硬编码灰色。
- **admin 浅色专业风**：沿用 soybean-admin 的浅色设计体系，**不要**把暗色霓虹规范带进 admin。认证/审核类页面必须单个根 `<div>`（模板有 `vite-plugin-vue-transition-root-validator` 校验）。

### 图片：URL 输入，不做上传

头像用 DiceBear（`https://api.dicebear.com/7.x/avataaars/svg?seed=<seed>`），封面/作品图用 picsum.photos。所有涉及图片的地方（用户头像、作品、认证证据、轮播图）都只是**文本框填外链**，没有文件上传。这是刻意的 YAGNI 决定，降低演示复杂度。真要上线需要补上传能力。

### API 命名与分层

- C 端接口前缀 `/api/v1/`，管理端前缀 `/api/admin/v1/`。
- 后端严格三层：handler 只做 HTTP 解析和组装，业务逻辑在 service，SQL 在 repository（部分由 sqlc 生成）。
- 前端不直连数据库，`uni.request` 也不在页面里直接调，统一走 `src/api/index.ts`（内部用 `client.ts`）。

### 演示期的取舍（YAGNI）

项目处在功能验证阶段，很多地方刻意从简，接手后要清楚这些边界：

- 聊天**无 WebSocket**，进页面轮询一次，未读数恒为 0。
- 无已读通知接口；`read` 只是前端本地状态。
- 管理端多数模块**无删除端点**（禁用/下架代替硬删），测试数据清理走 SQL。
- 评论删除是硬删，无恢复。
- 通知写入信任前端传来的 `userId`，没有服务端交叉校验（与 follows/chat 同模式）。

这些都是有意为之，不是遗漏。要投生产，这些点需要重新评估。

### 开发流程：SDD（规格驱动）

这个项目用「spec → plan → 执行 → 验证」的流程推进，痕迹都在 `docs/superpowers/`：

1. `specs/YYYY-MM-DD-<feature>-design.md`：设计文档，写清目标、数据模型、接口、取舍。
2. `plans/YYYY-MM-DD-<feature>.md`：任务拆解。
3. 按计划实现，每个任务有独立验证（后端测试 + curl E2E + 浏览器 E2E + 视觉复查）。
4. 提交，更新 `AGENTS.md` 对应小节。

新功能建议沿用这个套路：先写设计文档，再动手。想知道某个模块「为什么长这样」，去 `specs/` 找对应设计。

---

## 8. 数据库概览

17 个迁移（`server/migrations/000001` ~ `000017`），19 张表。按主题分：

| 主题 | 表 | 引入迁移 |
|------|----|---------|
| 基础业务 | `comic_events`、`photographers`、`services`、`works`、`reviews`、`tags`、`photographer_tags`、`banners` | 000001 及更早 |
| 漫展抓取与关注 | `event_follows`、`event_subscriptions` | 000004 ~ 000006 |
| 用户与鉴权 | `users`、`user_tokens` | 000007 |
| 收藏与聊天 | `photographer_favorites`、`chat_sessions`、`chat_messages` | 000008 |
| 摄影师身份 | `photographers` 加 `user_id` / `mode` / `certified` 等列 | 000009、000010 |
| 通知 | `notifications` | 000011、000016（加广播） |
| 管理端 | `admins` | 000012、000015 |
| 账号状态 | `users.status` | 000013 |
| 认证申请 | `photographer_cert_applications` | 000014 |
| 作品上下架 | `works.status` | 000017 |

几条容易踩的约束：

- `photographer_cert_applications` 有**部分唯一索引** `(photographer_id) WHERE status <> 'rejected'`，保证一个摄影师只能有一个活跃申请；`rejected` 后可重新申请。
- `chat_sessions` 用有序二元组（`user1_id <= user2_id`）保证两人之间只有一个会话。
- `works.status` 默认 `'active'`，新作品自动上架（先发后审）；公开查询过滤 `status='active'`，本人管理列表不过滤（下架的自己也看得到、能删）。
- 后端**不在启动时跑迁移**，迁移需手动执行（步骤见 `01-quickstart.md`）。

---

## 9. 快速对照

| 想做什么 | 去哪 |
|---------|------|
| 加一个 C 端页面 | `src/pages/<name>/` + 注册 `src/pages.json` |
| 加一个接口 | 前端 `src/api/index.ts`（真 HTTP + mock 兜底）；后端 `internal/handler` → `service` → `repository` |
| 加一个数据模型 | 类型放 `src/types/index.ts`；后端加迁移 + sqlc query |
| 加一个 Pinia store | `src/stores/`，composition 风格 |
| 改设计 token | `src/styles/variables.scss`（C 端） |
| 加管理端页面 | `admin/src/views/<name>/index.vue`（elegant-router 自动生成路由） |
| 加管理端接口 | `internal/handler/admin_*.go` + 注册到 `cmd/api/main.go` 的 adminGroup |
| 查某模块实现细节 | `AGENTS.md` 对应章节 |
| 查某功能设计原因 | `docs/superpowers/specs/` |

更细的「每个端点干什么、字段是什么、有哪些已知缺口」，全部在 `AGENTS.md`。这份架构文档只负责让你理解全局；具体到某行代码，回到知识库。
