# 03 功能模块清单

> 米拉漫展（Mira Comic-Con）：连接漫展 coser 与摄影师的约拍交易平台。
> 三端形态：C 端 uniapp 小程序（微信 + H5）、后台 `admin/`（soybean-admin）、Go 后端（Gin + PostgreSQL + Redis）。
> 本文面向接手同事，列出已交付模块、「它做什么」和「代码在哪」。数据表与接口细节见 `04-data-and-api.md`。

更新时间：2026-09-11 ｜ 代码库：`/vol1/1000/code/comic` ｜ 分支：`master`

---

## 一、C 端模块

C 端页面位于 `src/pages/<模块>/`，路由注册在 `src/pages.json`。底部 TabBar 三个入口：首页、消息、个人中心。视觉已统一为暗色霓虹主题（紫 `#a855f7` + 青 `#06b6d4` 于 `#0a0a1a` 底色）。

### 通用浏览

| 模块 | 页面路径 | 功能说明 | 关联接口 |
|------|----------|----------|----------|
| 首页 | `src/pages/index/index.vue` | 漫展驱动的 7 段式首页：轮播、近期漫展、热门标签、推荐摄影师、精选作品等。数据经 Redis 缓存 | `GET /api/v1/home` |
| 漫展列表 | `src/pages/event/list.vue` | 漫展列表，可按城市 / 状态筛选 | `GET /api/v1/events` |
| 漫展详情 | `src/pages/event/detail.vue` | 封面 + 倒计时 + 场馆信息；封面上的关注按钮（＋关注 / 已关注），未登录跳登录 | `GET /api/v1/events/:id`、`GET/POST/DELETE /api/v1/follows` |
| 漫展日历 | `src/pages/calendar/index.vue` | 自绘月视图 + 年面板，按日期聚合漫展；关注态绑定当前登录用户（`utils/follow.ts` 统一状态源） | `GET /api/v1/events`、`GET /api/v1/follows/:userId` |
| 搜索 | `src/pages/search/search.vue` | 关键词 + 标签筛选摄影师 | `GET /api/v1/photographers`、`GET /api/v1/tags` |
| 摄影师列表 | `src/pages/photographer/list.vue` | 摄影师列表，支持标签 / 城市过滤 | `GET /api/v1/photographers` |
| 摄影师详情 | `src/pages/photographer/detail.vue` | 资料 + 作品 + 服务 + 评价；收藏、发起聊天、进入预约；展示认证黄V 徽章与接单模式标签 | `GET /api/v1/photographers/:id`、`GET /api/v1/photographers/:id/timeslots`、`POST/DELETE /api/v1/favorites`、`POST /api/v1/chat/sessions` |
| 作品集 | `src/pages/portfolio/index.vue` | 浏览摄影师作品 | `GET /api/v1/photographers/:id`、`GET /api/v1/home` |

### 交易主链

| 模块 | 页面路径 | 功能说明 | 关联接口 |
|------|----------|----------|----------|
| 预约下单 | `src/pages/booking/index.vue` | 选服务 / 日期 / 时段；加载该摄影师当日已占用时段并置灰，需登录 | `GET /api/v1/photographers/:id/timeslots`、`POST /api/v1/bookings` |
| 订单列表 | `src/pages/order/list.vue` | 我的订单；coser 可取消预约 / 确认完成 / 联系摄影师 | `GET /api/v1/bookings/:userId`、`PUT /api/v1/bookings/:id/status`、`POST /api/v1/chat/sessions` |
| 订单详情 | `src/pages/order/detail.vue` | 订单明细 + 状态操作 + 跳转聊天 | 同上 |
| 评价 | `src/pages/comment/index.vue` | 对已完成拍摄打分（1-5 星）+ 文字 + 图片 | `POST /api/v1/reviews`、`GET /api/v1/photographers/:id` |
| 收藏 | `src/pages/favorite/list.vue` | 我收藏的摄影师，支持取消收藏 | `GET /api/v1/favorites/:userId`、`DELETE /api/v1/favorites/:userId/:photographerId` |
| 关注漫展 | `src/pages/follow/list.vue` | 已关注漫展列表（`onShow` 刷新），取消关注 | `GET /api/v1/follows/:userId`、`DELETE /api/v1/follows/:userId/:eventId` |

### 社交与账号

| 模块 | 页面路径 | 功能说明 | 关联接口 |
|------|----------|----------|----------|
| 聊天 | `src/pages/chat/index.vue` | 会话内消息收发；进入页面拉取，无 WebSocket | `GET /api/v1/chat/messages/:sessionId`、`POST /api/v1/chat/messages` |
| 消息 | `src/pages/message/index.vue` | 系统通知（来自 `notifications` 表，非演示假数据）+ 会话列表 | `GET /api/v1/notifications/:userId`、`GET /api/v1/chat/sessions/:userId` |
| 个人中心 | `src/pages/profile/index.vue` | 用户信息、角色徽章、菜单入口（订单 / 收藏 / 关注 / 摄影师面板）；编辑资料入口 | `GET /api/v1/me` |
| 编辑资料 | `src/pages/profile/edit.vue` | 修改昵称 / 头像链接，保存后直接更新本地 user 并持久化 | `PUT /api/v1/me/profile` |
| 设置 | `src/pages/settings/index.vue` | 通用设置页 | 本地状态 |
| 登录 | `src/pages/login/index.vue` | 手机号 + 验证码登录，登录后存 token | `POST /api/v1/login` |

### 摄影师侧（自助面板）

一个用户可开通摄影师身份（闲鱼式双角色）。开通后个人中心显示「摄影师」徽章，并解锁下列入口。

| 模块 | 页面路径 | 功能说明 | 关联接口 |
|------|----------|----------|----------|
| 开通摄影师 | `src/pages/photographer/activate.vue` | 填昵称 / 接单模式（互勉 / 收费 / 两者）/ 简介一键开通；已开通显示入口。同时展示认证状态按钮 | `POST /api/v1/photographers/activate`、`GET /api/v1/photographers/by-user/:userId`、`GET /api/v1/photographers/:id` |
| 认证申请 | `src/pages/photographer/cert-apply.vue` | 4 状态：审核中 / 已通过·黄V已生效 / 已驳回+理由+重新申请 / 未申请表单（样片链接 1-5 个 + 说明 ≤300 字） | `POST /api/v1/photographers/cert-apply`、`GET /api/v1/photographers/cert-application` |
| 接单管理 | `src/pages/photographer/orders.vue` | 摄影师订单面板：待确认 → 确认接单 / 拒绝；已确认 → 完成拍摄。请求带 `actorTag:'photographer'` | `GET /api/v1/bookings/photographer/:photographerId`、`PUT /api/v1/bookings/:id/status` |
| 主页管理 | `src/pages/photographer/profile-edit.vue` | 编辑主页：昵称 / 简介 / 城市 / 接单模式 / 互勉说明（全量覆盖语义） | `GET /api/v1/photographers/profile/mine`、`PUT /api/v1/photographers/profile` |
| 作品管理 | `src/pages/photographer/works.vue` | 发布 / 删除作品；列表含下架作品（自己可见），图片仅填外链 URL | `GET /api/v1/photographers/works/mine`、`POST /api/v1/photographers/works`、`DELETE /api/v1/photographers/works/:id` |

---

## 二、后台管理模块（`admin/`）

Soybean-admin v4 模板改造，浅色专业风，开发端口 `5174`（`5173` 是 C 端 H5，禁止占用）。接口前缀统一为 `/api/admin/v1`，除登录外均需管理员 token。

| 模块 | 前端视图 | 功能说明 | 关联接口 |
|------|----------|----------|----------|
| 工作台 | `admin/src/views/dashboard/` | 6 张统计卡（用户量 / 今日新增 / 日活 / 摄影师 / 订单 / 漫展）+ 3 个 ECharts（用户增长、订单状态分布、每日订单） | `GET /dashboard` |
| 摄影师审核 | `admin/src/views/photographer/` | 摄影师列表，按是否认证过滤，人工开关 `certified` | `GET /photographers?certified=`、`PUT /photographers/:id/certified` |
| 订单管理 | `admin/src/views/order/` | 分页订单列表（含摄影师 / 用户 join 字段），状态机流转（非法流转 409） | `GET /orders`、`PUT /orders/:id/status` |
| 漫展管理 | `admin/src/views/event/` | 全部漫展（含已下架），上架 / 下架 | `GET /events`、`PUT /events/:id/status` |
| 用户管理 | `admin/src/views/user/` | 用户列表（关键词 / 角色 / 状态筛选）+ 详情统计（订单 / 评价 / 收藏 / 关注数）+ 封禁 / 启用 | `GET /users`、`GET /users/:id`、`PUT /users/:id/status` |
| 认证审核 | `admin/src/views/certification/` | 三 tab（待审核 / 已通过 / 已驳回）+ 证据图片查看；通过或驳回（驳回理由必填） | `GET /cert-applications`、`GET /cert-applications/:id`、`PUT /cert-applications/:id/review` |
| 管理员管理 | `admin/src/views/admin-accounts/` | 管理员列表 / 新建 / 禁用启用 / 重置密码；主管理员（id=1）不可禁用 | `GET/POST /admins`、`PUT /admins/:id/status`、`PUT /admins/:id/password` |
| 通知公告 | `admin/src/views/notification/` | 发布通知（全员 / 指定用户）+ 通知历史列表 | `POST /notifications`、`GET /notifications` |
| 轮播图管理 | `admin/src/views/banner/` | 轮播图列表 / 新增 / 编辑 / 上下线；管理端维护即实时驱动 C 端首页轮播 | `GET/POST /banners`、`PUT /banners/:id`、`PUT /banners/:id/status` |
| 内容管理 | `admin/src/views/content/` | 作品上 / 下架、评论搜索与硬删、标签 CRUD（删除前做引用检查） | `GET /works`、`PUT /works/:id/status`、`GET /reviews`、`DELETE /reviews/:id`、`GET/POST/PUT/DELETE /tags` |

---

## 三、后端能力

- **Web 框架**：Gin，端口 `:8080`；PostgreSQL（本地 `5433`）+ Redis 缓存。另保留 std-lib mock `server/mock_server.go`（`:8081`）供无后端演示。
- **双认证栈隔离**：C 端走 `user_tokens` 表 + `AuthRequired` 中间件；管理端走 `admins` 表 token + `AdminAuthRequired`。两套 token 互不通用，C 端 token 访问 admin 路由直接 401，反之亦然。
- **订单状态机**：`pending → confirmed → completed`，`pending/confirmed → cancelled`，其余流转 409。创建订单时同摄影师 / 同日期 / 同时段的非取消订单冲突返回 409；提供当日占用时段查询供前端置灰。
- **通知链路**：预约创建与状态变更写入 `notifications`（仅 coser 侧）；管理端可发全员广播（`user_id` 为 NULL 的行对所有用户可见）。
- **认证审批事务**：approve 用单原子事务（申请标记 approved + `photographers.certified=true` + 审核人 / 时间），随后 best-effort 发通知；reject 单写 + 理由，同样 best-effort 通知。一摄影师一活跃申请由部分唯一索引硬保证。
- **缓存一致性**：首页数据缓存于 Redis（TTL 5min）；管理端下架 / 恢复作品会立即失效 `home` 缓存，不等 TTL。
- **数据抓取（ingest）**：`cmd/ingest` 抓取 nyato.com 漫展，支持 YAML cron 调度（`server/config/cron.yaml`）。任务：每日 03:00 全量 5 页、14:30 增量 2 页、01:00 过期清理。每次抓取顺带把过期漫展标记 `del_flag=true`。nyato 拒绝 Go HTTP/2，抓取强制 HTTP/1.1 + 浏览器 UA。
- **容错**：Redis 为 best-effort，不可用时服务照常启动（`USE_MOCK=true` 时甚至可脱离数据库返回种子数据）。

---

## 四、演示数据状态

当前数据库（截至交付）的演示快照，方便接手后直接演示各条链路。

| 数据 | 状态 |
|------|------|
| 摄影师 1-4 | 光影行者 / 樱花落 / 暗夜骑士 / 古风公子，`certified=true`（黄V）。其中暗夜骑士(3) 是 E2E 审批通过 |
| 摄影师 5 | 测试摄影师，`certified=false` |
| 认证申请 | 共 7 条：#1 樱花落 approved、#2 暗夜骑士 rejected、#3 古风公子 approved、#4 暗夜骑士 approved、#5/#6/#7 测试摄影师 rejected（可演示「已驳回 → 重新申请」完整流程） |
| 作品 works | 4 条：1/2 属光影行者、3 属樱花落、4 属古风公子，均 `status='active'` |
| 评价 reviews | 3 条（小狐狸 5★ / 月华 5★ / 用户5213 1★，均属光影行者） |
| 标签 tags | 13 个，`usageCount` 为真实关联统计（0-2） |
| 轮播图 banners | 3 条：ChinaJoy 2026(sort 0) / 第40届萤火虫(sort 1) / CP33(sort 2)，均 `isActive=true` |
| 用户 users | 约 17 个 |
| 漫展 | 48 个活跃；总量约 145（含 `del_flag=true` 软删的过期漫展） |
| 订单 bookings | 基线约 10 单 |
| 管理员 admins | 仅种子账号 1 个（id=1，role=admin，status=active） |

### 测试账号

| 角色 | 账号 | 说明 |
|------|------|------|
| C 端摄影师 | `10000000001` ~ `10000000004` | 分别对应摄影师 1-4，均已认证 |
| C 端 coser | `13800138000` | 用户 8000，可下常规订单 |
| 未激活用户 | `13900001111` | 用于演示「开通摄影师」 |
| 验证码 | `123456` | 任意以 `1234` 为前缀的 6 位数字即可（如 `123456`） |
| 后台管理员 | `admin` / `admin123` | 本地种子账号，**生产部署前必须改密** |

### 已知边界（演示简化，非缺陷）

- 聊天无 WebSocket，进入页面才拉取消息，不做实时推送。
- 未读数固定为 0；通知的已读标记仅前端本地置位，刷新后回到未读。
- 图片均为外链 URL（DiceBear 头像 / picsum 封面 / 站点真实图），无文件上传能力。
- 消息页、关注列表刷新依赖页面 `onShow`，跨页面返回靠重新拉取保证一致。
