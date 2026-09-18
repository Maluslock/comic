# P2c 独立 Web 后台管理端 设计

**日期:** 2026-09-01
**状态:** 已批准设计（待实现）
**相关:** P2a 双角色身份体系（发起方）、P3 通知真实化（Dashboard 复用 notifications 计数可后续加）

## 目标

搭建米拉漫展的独立 Web 后台管理端（方案 A：独立 Web 管理端，沿用 P2a 决策），覆盖：

1. **Dashboard 数据监控面板**（电商后台风格）：用户量、日活（DAU）、注册/登录 7 日趋势、订单统计与状态分布、摄影师/认证数、漫展数。
2. **摄影师认证审核**：后台直接管理 `certified` 开关（黄V 认证徽章；P2b 摄影师申请流本期不做，后台 API 不预留应用表）。
3. **订单管理**：全站订单总览（筛选/分页）+ 管理员状态操作（走同一状态机，跨角色例外）。
4. **漫展管理**：漫展列表 + 上下架（`del_flag` 软删复用）。

**非目标（YAGNI）**：多语言 i18n、复杂权限组（后台单角色 `admin`，预留 `super` 角色列）、P2b 认证申请流、系统通知公告推送、摄影师端消息/评价干预。均二期。

## 总体架构

```
comic/
├── server/                          # 现有 Go 后端 :8080（并入 admin API）
│   ├── db/migrations/               # 新: 000012_admins.up/down.sql
│   ├── internal/
│   │   ├── repository/
│   │   │   ├── admin.go             # AdminRepo（admins 表 CRUD）
│   │   │   └── admin_stats.go       # AdminStatsRepo（dashboard 统计）
│   │   ├── service/
│   │   │   ├── admin_service.go     # 认证/摄影/订单/漫展管理
│   │   │   └── admin_stats.go       # dashboard 聚合
│   │   ├── middleware/admin_auth.go # AdminAuthRequired（admin JWT）
│   │   └── handler/
│   │       ├── admin_handler.go     # login + 3 组管理端点
│   │       └── admin_dashboard.go   # dashboard 端点
│   └── cmd/api/main.go              # /api/admin/v1/* 路由组
├── admin/                           # 新: 独立 Web 管理端（soybean-admin 改造，Vue3+Vite+TS）
│   ├── src/views/
│   │   ├── dashboard/               # 数据面板页
│   │   ├── photographer/            # 摄影师审核列表
│   │   ├── order/                   # 订单管理列表
│   │   └── event/                   # 漫展管理列表
│   └── src/store/                    # 复用模板 store（token 存调）
└── docs/superpowers/specs/2026-09-01-admin-web-design.md
```

- **后端**: `/api/admin/v1/*` 路由组，独立 `admins` 表 + 用户名/密码登录 + Admin JWT；复用现有 repository/service（订单状态机、摄影师查询、漫展查询）；1 个 :8080 门户（现有 CORS 全放行）。
- **前端**: 拉取 `soybeanjs/soybean-admin`（MIT，14.9k★，Vue3.5+Vite8+TS6+Pinia3+NaiveUI+UnoCSS，16.9MB）→ 置于 `admin/` 子目录；移除 vue-i18n（后台无多语言）；**浅色专业电商风**（模板默认，不做暗色霓虹——仅 C 端小程序保持暗色主题）。

## 数据模型

### admins 表（migration 000012_admins）

```sql
CREATE TABLE admins (
  id            BIGSERIAL PRIMARY KEY,
  username      VARCHAR(64) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,        -- bcrypt
  role          VARCHAR(20) NOT NULL DEFAULT 'admin',  -- 预留 admin/super
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**种子**：初始管理员 `admin` / `admin123`（bcrypt 哈希随迁移写入，后续开发可改）。生产部署必须改密（文档提示）。

### 复用现有表（无迁移改动）

| 表 | 用于 |
|----|------|
| `photographers` | 审核列表（`certified` 布尔切换，P2a 已有列） |
| `bookings` | 订单管理（`photographer_id`/`coser_id` join `photographers`/`users`；`status` 状态机） |
| `comic_events` | 漫展管理（`del_flag` 上下架，P2a-ingest 已用） |
| `users` / `user_tokens` | Dashboard 统计（注册量 = users.created_at；日活 ≈ user_tokens.created_at 当日登录去重） |

## 管理 API（全部 `AdminAuthRequired`，除 login）

| Method | Path | 行为 |
|--------|------|------|
| POST | `/api/admin/v1/login` | `{username,password}` → 200 `{token, admin:{id,username,role}}`；错误 401 |
| GET | `/api/admin/v1/dashboard` | 200 dashboard 聚合（下详） |
| GET | `/api/admin/v1/photographers` | 200 列表：id/name/avatar/location/rating/certified/mode/phone(user join)/orderCount；可选 `?certified=` 筛选 |
| PUT | `/api/admin/v1/photographers/:id/certified` | `{certified:bool}` → 200 `{ok:true}`；404 不存在 |
| GET | `/api/admin/v1/orders` | 200 分页列表：id/status/date/time/price/createdAt/coserName/photographerName；`?status=&page=&pageSize=` |
| PUT | `/api/admin/v1/orders/:id/status` | `{status}` 管理员改状态。**跨角色例外**：跳过 actorTag 校验（管理员不是 coser/摄影师），但**保留状态机合法转移校验**（非法转移 409） |
| GET | `/api/admin/v1/events` | 200 列表：id/name/location/venue/startDate/endDate/status/delFlag/typeName |
| PUT | `/api/admin/v1/events/:id/status` | `{delFlag:bool}` 上下架 → 200 `{ok:true}` |

**约束**：
- 管理 token 与 C 端 user token 分离（不同 secret 前缀/claim `role=admin`），中间件严格校验 admin claim；user 无法访问 admin 路由。
- 管理员改订单状态仅允许：`pending→confirmed`、`confirmed→completed`、`pending/confirmed→cancelled`（与现有状态机一致）；不允许 completed→pending 等回退（409）。

## Dashboard 指标

### API 响应（GET /api/admin/v1/dashboard）
```json
{
  "totals": { "users":16, "photographers":12, "certified":0, "orders":10,
              "pendingOrders":4, "events":8 },
  "trends": {
    "date":["08-26","08-27","08-28","08-29","08-30","08-31","09-01"],
    "newUsers":[0,0,1,2,0,1,0],
    "activeUsers":[1,2,3,2,4,2,2],
    "orders":[0,1,0,2,1,0,2]
  },
  "ordersByStatus": { "pending":4, "confirmed":2, "completed":3, "cancelled":1 }
}
```

### 指标定义（数据源实测可行）
| 指标 | 定义 | 来源 |
|------|------|------|
| 用户总量 | `count(users)` | users |
| 今日新增 | `created_at::date = current_date` | users |
| 日活 DAU | `count(distinct user_id) where created_at::date = current_date`（登录行为近似） | user_tokens |
| 摄影师/认证 | count / `certified=true` | photographers |
| 订单/待确认 | count / `status='pending'` | bookings |
| 漫展数 | `count(*) where del_flag=false` | comic_events |

趋势：近 7 日历日 `date_trunc('day', created_at)` group by，Go 侧按日期序列补零（缺失日=0）。时区按 Asia/Shanghai（服务端 `created_at` 为 timestamptz，按 SQL 本地日期即可，与现库一致）。

## 前端页面（admin/）

### 页面构成（soybean 模板默认浅色主题 + 移除 i18n）
1. **登录页**：复用模板登录布局，改接 `/api/admin/v1/login`（用户名/密码），token 存 `localStorage`（模板默认）。
2. **Dashboard**（默认首页）：
   - 顶部指标卡 4-6 个（用户总量+今日新增、日活 DAU、摄影师/已认证、订单/待确认、漫展）
   - 三图：注册+日活双线折线（7 日）、订单状态环形（ECharts pie）、预约量柱状（7 日）
3. **摄影师审核**：表格（头像/昵称/城市/评分/接单模式/认证状态）+ 认证状态切换按钮（通过/取消认证，Tag 黄V/灰）。
4. **订单管理**：表格（订单号/状态Tag/摄影师/用户/日期时间/价格）+ 状态筛选 Tabs + 操作按钮（确认/完成/取消，按状态机可用性禁用）+ 分页。
5. **漫展管理**：表格（名称/城市/场馆/日期/状态）+ 上下架开关。

### 模板改造点（soybean-admin）
- **移除 i18n**：删 `vue-i18n` 依赖 + `locales` 相关配置/目录（改完后无 15 国语言）。
- 移除演示页/示例菜单（白屏/404/系统管理未用页），保留登录+3 组管理页。
- 请求封装指向 `http://localhost:8080/api/admin/v1/`（H5 dev 跨域由现有 CORS 放行）；`Authorization: Bearer <admin token>`。
- 路由菜单：Dashboard/摄影师审核/订单管理/漫展管理。
- 主题：模板默认浅色；色板沿用 soybean 默认（不做暗色）。

### 视觉规范
- 设计走 **frontend-design + ui-ux-pro-max** skill（模板既有设计体系上微调，风格对齐现代电商后台）。
- 视觉验收用 **agent-browser**（:5173 后台页截图对照）。

## 技术约束

- 后端 Go 1.22+；`go build ./...` + `go test ./...` 过；新代码遵循现有 layering（handler→service→repository）。
- 前端 Node ≥ 20（本机 v22.18.0 ✓）；`pnpm i` + `pnpm dev`；`vue-tsc` 过。
- 管理端与 C 端主题刻意分离：C 端暗色霓虹（uniapp），后台浅色专业（soybean 默认），互不影响。
- admin 目录内 `.git` 删除（模板并入本仓库）；AGENTS.md 更新。

## 错误处理

- 401：token 缺失/无效/role 非 admin（AdminAuthRequired）。
- 404：photographer/event 不存在。
- 409：订单非法状态转移（管理员操作同样受状态机约束）。
- 前端：axios 拦截 401 → 跳登录；请求失败 toast。

## 测试策略

- 后端：`go test ./internal/handler/admin*` 单测重点——login（正确/错误密码）、dashboard 聚合形状、certified 切换、订单状态机例外（合法/非法转移 409）。
- 前端：`vue-tsc --noEmit` + 手测（登录/各页渲染/审核切换/订单操作/漫展上下架）。
- E2E：Playwright 登录 admin → Dashboard 数据渲染 → 审核切换 → 订单操作 → 漫展上下架全链；agent-browser 截图视觉复查。

## 时间线（5 Tasks）

1. 后端：admins 迁移 + AdminRepo + bcrypt + AdminAuth + login API（种子 admin/admin123）→ 冒烟 + push
2. 后端：dashboard stats repo/service/handler + 冒烟 + push
3. 后端：审核/订单/漫展管理 API（certified 切换、订单状态机例外、del_flag）+ 冒烟 + push
4. 前端：拉 soybean-admin → admin/ 改造（去 i18n/演示页、登录+3 组页面+Dashboard、接 API、浅色主题）→ 本机跑通 + push
5. 验证：后端单测 + Playwright 全链 E2E + agent-browser 视觉复查 + AGENTS.md + push

## 后续（不在本次）

- P2b：摄影师端认证申请流（后台 pending 列表 + 通过/拒绝）
- 后台权限组（super/admin 分权）
- 系统通知公告推送
- Dashboard 增加收入/评价/活跃时段等运营指标
