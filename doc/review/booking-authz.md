# t3 权限与越权评审报告（IDOR / actorTag / 手机号遮罩）

- 评审范围：`comic/server/internal/middleware/auth.go`、`handler/booking_handler.go`、`handler/authz.go`、`service/booking_service.go`、`repository/bookings.sql.go`、`repository/users.sql.go`、`repository/favorites.sql.go`、`cmd/api/main.go`、`comic/src/pages/{booking,order,photographer}/*.vue`
- 环境：API `http://localhost:8088/api/v1`（/health ok）、PG `127.0.0.1:5433 comic/comic123`、服务器时间 2026-10-29 11:2x CST
- 账号：coser `13000000001`(u148) / `13000000002`(u154) / `13800138000`(u5)；摄影师 `10000000001..03`(u1/u2/u3, photographerId=1/2/3)；admin `admin/admin123`
- 性质：**只读评审**，未改动任何源码（`git status --porcelain` 仅 `?? doc/review/`）；所有测试数据已删除并回填派生统计（见文末「清理证明」）
- 探针总量：56 组请求（IDOR / actorTag / 遮罩 / token 隔离 / 存在性 oracle / 提权）

---

## 一、结论摘要

**授权基线是干净的**：全站 47 个路由已逐一枚举，所有写操作都挂了 `AuthRequired`；所有 handler 一律以 token 身份（`middleware.UserID(c)`）判定归属，路径/body 里的 user id 只在「读自己」场景做 `assertSelf` 相等比较；`POST /favorites`、`/follows`、`/notifications` 明确覆盖 body 里的 `userId`。**56 组越权探针中，未发现一条「A 的 token 操作 B 的资源后实际生效」的越权**——所有「看着像越权」的探针，落库归属都正确落在 token 用户身上。

但在这层干净基线之上，发现 **4 个 high / 5 个 medium / 6 个 low** 缺陷。核心结论：

1. **角色化状态机缺失（high）**：`actorTag` 只回答「你是不是订单的某一方」，不回答「这个动作该不该由你这一方做」。coser 可以把自己的 pending 单直接推成 confirmed 再推成 completed，进而满足「已完成才能评价」的守卫、伪造评分、给自己解锁摄影师手机号。**这与 t1 报告的「actorTag 伪造」不是同一件事**（t1 验的是「冒充他人角色」，这里是「用合法角色做不该由他做的动作」）。
2. **禁用用户旧 token 仍有效（high）**：C 端 token 查询未过滤 `users.status`，admin 禁用账号后旧 token 仍可下单。admin 侧有对等守卫，C 端没有。
3. **生产部署假验证码登录仍开启（high）**：prod compose 未传 `APP_ENV`，落到默认 `development` → 任意手机号 + 任意 `1234` 前缀验证码即可登录**任意已注册账号**（实测登录摄影师 `10000000001` 拿到完整 token）。
4. **NULL 字段导致整列表 500（high）**：`activate` 出的摄影师 `avatar` 为 NULL，两个 SQL 缺 `COALESCE` 且扫描目标是非指针 `string` → 该摄影师有单时，对方的「我的订单」「我的收藏」列表整体 500。

---

## 二、授权基线盘点（已验证安全面）

### 2.1 身份来源：写操作一律取 token

| 端点 | file:line | 身份来源 |
|---|---|---|
| `POST /api/v1/bookings` | `handler/booking_handler.go:28` | `req.CoserID = int32(middleware.UserID(c))`（覆盖 body） |
| `PUT /api/v1/bookings/:id/status` | `booking_handler.go:76` | `middleware.UserID(c)` |
| `POST /api/v1/bookings/:id/quote` | `booking_handler.go:106` | `middleware.UserID(c)` |
| `POST /api/v1/bookings/:id/quote/respond` | `booking_handler.go:127` | `middleware.UserID(c)` |
| `GET /api/v1/bookings/:userId` | `booking_handler.go:175` | `assertSelf(c, int64(userID))` |
| `GET /api/v1/bookings/photographer/:pid` | `booking_handler.go:156` | `ListByPhotographerForUser` 内校验档案归属 |
| `POST /api/v1/reviews` | `handler/review_handler.go:31-38` | `req.UserID = int32(middleware.UserID(c))`；`req.UserName/UserAvatar` 被清空后从 DB 取真实值 |
| `POST /api/v1/favorites` | `handler/favorite_handler.go:22-39` | `middleware.UserID(c)` |
| `POST /api/v1/follows` | `handler/follow_handler.go:28` | `req.UserID = middleware.UserIDStr(c)` |
| `POST /api/v1/notifications` | `handler/notification_handler.go:22-46` | `middleware.UserID(c)` |
| `POST /api/v1/chat/sessions` | `handler/chat_handler.go:30` | `middleware.UserID(c)` + `otherUserId` |
| `POST/DELETE /api/v1/chat/messages` | `service/chat_service.go:80-103` | `assertParticipant` |
| `POST /api/v1/blocks` | `handler/block_handler.go:22-41` | `middleware.UserID(c)` |
| 套餐/作品/档案 CRUD | `handler/photographer_service_handler.go:59-114`、`photographer_work_handler.go:22-128` | `middleware.UserID(c)` + service 层 `photographer_id` 限定 |
| `PUT /api/v1/me/profile` | `cmd/api/main.go:220-255` | 只接受 name/avatar/bio，**不接受 phone/role**，无法提权 |

**读类端点**（`GET /favorites/:userId`、`/chat/sessions/:userId`、`/chat/unread/:userId`、`/notifications/:userId`、`/follows/:userId`）统一走 `assertSelf` 或直接与 `middleware.UserID(c)` 比对 → 他人一律 403。

### 2.2 IDOR 探针矩阵（56 组，节选关键项）

| # | 探针 | 期望 | 实测 | 落库核对 | 判定 |
|---|---|---|---|---|---|
| A1 | coserB `GET /v1/bookings/148` | 403 | 403 `cannot access another user's data` | — | ✅ |
| B1 | coserA `GET /v1/bookings/photographer/1`（非摄影师） | 403 | 403 `forbidden` | — | ✅ |
| B2 | photog2 `GET /v1/bookings/photographer/1` | 403 | 403 `forbidden` | — | ✅ |
| C1 | coserB 无 actorTag 改 coserA 的单 | 403 | 403 | — | ✅ |
| C2 | coserB `actorTag=coser` | 403 | 403 | — | ✅ |
| C3 | photog2 `actorTag=photographer` 改 photog1 的单 | 403 | 403 | — | ✅ |
| C4 | coserA `actorTag=photographer`（冒充） | 403 | 403 | — | ✅ |
| C5 | `actorTag=admin`（未知 tag） | 403 | 403（default-deny） | — | ✅ |
| D1 | photog2 给 photog1 的单报价 | 403 | 403 | — | ✅ |
| D2 | coserA 自己报价（应只有摄影师能报） | 403 | 403 | — | ✅ |
| D3 | coserB 接受 photog1 的单报价 | 403 | 403 | — | ✅ |
| D4 | photog1 接受 coser 的报价 | 403 | 403 | — | ✅ |
| E1 | 面议单（awaiting_quote）直接 confirmed | 409 | 409 `invalid status transition` | — | ✅ |
| F1 | coserB `POST /favorites {userId:148}` | 201 | 201 | `photographer_favorites(user_id=154)` ✅ 落库是 token 用户 | ✅ 假阳性 |
| F2 | coserB `GET /favorites/148` | 403 | 403 | — | ✅ |
| G1 | coserB `POST /follows {userId:148}` | 201 | 201 | `event_follows(user_id=154)` ✅ | ✅ 假阳性 |
| H1 | coserB `POST /notifications {userId:148}` | 201 | 201 | `notifications(user_id=154)` ✅ | ✅ 假阳性 |
| H2 | coserB `GET /notifications/148` | 403 | 403 `cannot read another user's notifications` | — | ✅ |
| H3 | `POST /notifications/9999/read` | 404 | 404 | — | ✅ |
| I1 | photog1 `GET /chat/messages/{非成员会话}` | 403 | 403 | — | ✅ |
| I2 | photog1 `POST /chat/messages` | 403 | 403 | — | ✅ |
| I3 | coserB `POST /chat/sessions/{非成员}/read` | 403 | 403 | — | ✅ |
| J1 | coserB `POST /reviews {userId:148}` | 403 | 403 `only completed shoots can be reviewed` | — | ✅ |
| K1 | photog2 `PUT/DELETE /photographers/services/5` | 403 | 403 `cannot modify another photographer's service` | — | ✅ |
| K2 | photog2 `PUT/DELETE /photographers/works/5` | 403 | 403 `cannot edit/delete another photographer's work` | — | ✅ |
| K3 | coserA `GET /photographers/works/mine` | 403 | 403 `not a photographer` | — | ✅ |
| L1 | coserB 拉黑 photog1 后 `POST /bookings` | 403 | 403 `cannot book this photographer` | — | ✅ |
| L2 | 同上 `POST /chat/sessions` | 403 | 403 `cannot start a chat with this user` | — | ✅ |
| M1 | coserA token → `/api/admin/v1/users` | 401 | 401 `invalid admin token` | — | ✅ 隔离 |
| M2 | admin token → `/api/v1/me` | 401 | 401 `invalid token` | — | ✅ 隔离 |
| M3 | 无 token | 401 | 401 `missing token` | — | ✅ |
| M4 | admin token → `/api/admin/v1/admins` | 200 | 200（role=super，RequireSuper 生效） | — | ✅ |
| N1 | coserB `POST /bookings {serviceId: 别人的套餐}` | 404 | 404 `photographer or service not found` | — | ✅ |
| N2 | coserB `POST /bookings {serviceId: 平台模板}` | 404 | 404 同上 | — | ✅ |
| O1 | coserB `POST /reviews {userName:"冒充摄影师"}` | 落库真名 | 落库 `user_name='用户8000'`、`user_avatar='/static/img/avatar-user.svg'` | ✅ body 字段被丢弃 |

### 2.3 手机号遮罩（正确，SQL 层）

遮罩写在 SQL，**不在应用层**，因此号码根本不出现在响应体里：

```sql
-- comic/server/internal/repository/bookings.sql.go:205-218
CASE WHEN b.status IN ('confirmed','completed') THEN COALESCE(u.phone,'') ELSE '' END AS coser_phone
```

实测矩阵（`GET /v1/bookings/photographer/1`，清理后复检）：

| booking 状态 | coserPhone | coserName |
|---|---|---|
| `pending` | `''` | 正常返回（未被清空） |
| `cancelled` | `''` | 正常返回 |
| `confirmed` | 真实号 | 正常返回 |
| `completed` | 真实号 | 正常返回 |
| 面议 `awaiting_quote`（pending） | `''` | 正常返回 |
| 面议 `quoted`→`agreed`+accept（转 confirmed） | 真实号 | 正常返回 |

- **quote 阶段不泄露**：报价时 `status` 仍是 `pending` → 号码为空；只有 coser 接受报价、订单转 `confirmed` 才出现。符合设计。
- **admin 路径不绕过**：管理端订单列表 SQL（`repository/admin_manage.go:104-133`）**不含 phone 列**；`/api/admin/v1/export/orders` 的 CSV 表头为「ID,状态,日期,时间,金额,创建时间,Coser,摄影师」，无手机号。admin 用户/摄影师列表与 users CSV 含明文手机号，属 admin 域内职责范围（`admin_export_handler.go:75/112`、`admin_manage.go:69/179`、`admin_user_repo.go:67/98`）。
- 仓内唯一一处真库集成测试覆盖了遮罩：`repository/bookings_privacy_integration_test.go:TestIntegration_GetBookingsByPhotographer_MasksCoserPhoneUntilConfirmed`。
- **但遮罩的解锁条件绑定在 `status` 上**——见 F-02：coser 能自己把 status 推到 confirmed，从而**自行解锁手机号**。

---

## 三、分级问题清单

### HIGH-1｜coser 可自行「确认接单」→ 自行「完成」→ 自造好评（角色化状态机缺失）

- **file:line**
  - `comic/server/internal/handler/booking_handler.go:73-75`：`if req.ActorTag == "" { req.ActorTag = "coser" }`
  - `comic/server/internal/service/booking_service.go:325-340`：actorTag 分支只校验「你是不是这一方」
  - `comic/server/internal/service/booking_service.go:75-84` `canTransition`：只看状态机，不看角色
  - `comic/server/internal/service/review_service.go:40-46` + `repository/reviews.sql.go:126-131` `hasCompletedBookingWith`：以 `status='completed'` 为「可评价」唯一门槛
- **复现（已验证）**
  ```bash
  TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"phone":"13000000001","code":"1234"}' http://localhost:8088/api/v1/login | jq -r .token)
  # 1) coser 给摄影师 2 下单
  curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"photographerId":2,"serviceId":9,"date":"2027-06-01","time":"11:00"}' \
    http://localhost:8088/api/v1/bookings            # → 201 {"id":47,...,"status":"pending"}
  # 2) coser 自己确认接单
  curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"status":"confirmed"}' \
    http://localhost:8088/api/v1/bookings/47/status   # → 200 status=confirmed  ← 应为 403
  # 3) coser 自己标记完成
  curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"status":"completed"}' \
    http://localhost:8088/api/v1/bookings/47/status   # → 200 status=completed
  # 4) coser 自造 5 星好评
  curl -s -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"photographerId":2,"rating":5,"content":"自造好评"}' \
    http://localhost:8088/api/v1/reviews              # → 201（首次），重复 → 409
  ```
  DB 核对：`bookings id=47 status=completed`；`reviews id=9 (photographer_id=2,user_id=148,rating=5)`；`GET /api/v1/photographers/2` → `rating=5, reviewCount=1, orderCount=1`（已被污染）。
- **影响**
  1. 「摄影师确认接单」这一业务动作被 coser 单方完成，摄影师完全不知情——`booking_service.go:354-364` 在 coser 触发的流转里只通知 coser，摄影师只收到 Create 时的「收到新预约」。
  2. **手机号遮罩被自行解锁**：status 转 confirmed 后，接单面板立刻返回 coser 真号（实测 `coserPhone='13000000001'`）。
  3. **评分/单量可自证**：`rating`/`review_count`/`order_count` 是派生字段（`repository/reviews.sql.go:147-154`），completed 由 coser 自造 → `hasCompletedBookingWith` 守卫被绕过 → 好评可刷；`order_count` 还能污染「接单最多」排序键。
  4. 唯一约束 `uniq_reviews_user_photographer` 只限制「同一摄影师同一 coser 一次」，对「对任意摄影师各刷一次」无效。
- **建议修法**
  - 在 `canTransition` 之上加角色矩阵：`confirmed` 只能由 `actorTag=photographer` 发起；`cancelled` 两方均可；`completed` 由摄影师发起（若产品要求 coser 也可点「确认完成」，则该动作必须另记 `completed_by` 并在 `hasCompletedBookingWith` 里要求 `completed` 是由摄影师确认后的状态）。
  - 评估「coser 先 completed 再评价」的产品语义：若确实允许 coser 完成，则 `hasCompletedBookingWith` 需额外校验「该订单存在摄影师确认记录」。
  - `booking_handler.go:73-75` 建议显式拒绝空串 actorTag（与缺省区分），并把默认值改为服务端按「该订单归属」推断角色而非写死 `coser`。

### HIGH-2｜被管理员禁用的用户，旧 token 仍然有效

- **file:line**：`comic/server/internal/repository/users.sql.go:96-104`
  ```sql
  SELECT u.id, COALESCE(u.phone,''), u.name, u.avatar, u.status, u.created_at
  FROM user_tokens t JOIN users u ON u.id = t.user_id
  WHERE t.token = $1 AND t.expires_at > NOW()
  -- 缺 AND u.status = 'active'
  ```
  `comic/server/internal/middleware/auth.go:13-30` 只校验 token 存在与未过期。
  对照 admin 侧 `comic/server/internal/repository/admin.go:51-58`：`AND token_expires_at > NOW() AND status = 'active'` —— admin 禁用即时失效，C 端没有对等守卫。
- **复现（已验证）**
  ```bash
  # 1) 禁用前登录拿 token（u6 / 13900001111）
  T=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"phone":"13900001111","code":"1234"}' http://localhost:8088/api/v1/login | jq -r .token)
  # 2) admin 禁用该用户
  A=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"admin123"}' http://localhost:8088/api/admin/v1/login | jq -r .token)
  curl -s -X PUT -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
    -d '{"status":"disabled"}' http://localhost:8088/api/admin/v1/users/6/status   # → 200 {"ok":true}
  # 3) 用禁用前的 token 访问
  curl -s -H "Authorization: Bearer $T" http://localhost:8088/api/v1/me          # → 200（仍返回 phone）
  curl -s -X POST -H "Authorization: Bearer $T" -H 'Content-Type: application/json' \
    -d '{"photographerId":1,"serviceId":5,"date":"2026-12-01","time":"10:00"}' \
    http://localhost:8088/api/v1/bookings                                          # → 201（成功下单！）
  # 4) 重新登录被拒
  curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"phone":"13900001111","code":"1234"}' http://localhost:8088/api/v1/login  # → 403 {"error":"account disabled"}
  ```
- **影响**：管理员的「禁用」只挡住新登录，挡不住已签发的 7 天 token（`auth_service.go:119-142`）。被禁用账号仍能读资料、聊天、下单、评价、举报。禁用形同虚设。
- **建议修法**：`GetUserByToken` 增加 `AND u.status = 'active'`；或 `AuthRequired` 中比对 `u.Status`，disabled 直接 401（并可顺带 `DELETE FROM user_tokens WHERE user_id=$1` 使旧 token 立即作废）。同时补一条集成测试。

### HIGH-3｜生产部署下「手机号 + 任意 1234 前缀验证码」可登录任意已注册账号

- **file:line**
  - `comic/server/internal/service/auth_service.go:76-83`：`if len(req.Code) < 4 || req.Code[:4] != "1234" { return error }` —— 只校验前 4 位，无随机性、无时效、无次数限制；且**服务端根本没有下发验证码的通道**。
  - `comic/server/internal/config/config.go:59`：`viper.SetDefault("APP_ENV", "development")`
  - `comic/server/cmd/api/main.go:197`：`authSvc.SetMockLoginAllowed(cfg.AppEnv != "production")`
  - `comic/deploy/docker-compose.prod.yml:34-44`（api service environment 段）：只有 `DB_* / REDIS_* / SERVER_PORT / USE_MOCK / ALLCPP_BASE_URL` —— **没有 `APP_ENV`**
  - `comic/deploy/.env.example`：无 `APP_ENV`
  - `comic/deploy/Dockerfile.api`：仅 `ENV TZ=Asia/Shanghai`
  - `comic/server/.env:11` 与 `comic/server/.env.example:20`：`APP_ENV=development`
- **复现（已验证）**
  ```bash
  # code 前缀即可，长度不限
  curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"phone":"10000000001","code":"12340000"}' http://localhost:8088/api/v1/login
  # → 200，拿到 user 1（摄影师 10000000001）的 token，且响应含 photographerId=1
  # 随后该 token 可读接单面板、works/mine、profile/mine
  ```
  错误验证码 `9999` → 401 `unauthorized`；`1234` 与 `12340000` 均成功。
- **影响**：按仓库现有 prod compose 部署，APP_ENV 落到默认 `development` → 假验证码登录在生产开启 → **任何已知手机号都能被接管**（摄影师/coser 均可），且接管摄影师账号后可看到 coser 手机号（confirmed 订单）。这是本次评审中风险最高的部署侧缺陷。
- **建议修法**
  1. `deploy/docker-compose.prod.yml` api service 显式加 `APP_ENV: "production"`；`deploy/.env.example` 补 `APP_ENV=production`。
  2. `config.go:59` 默认值改为 `production`（fail-safe：默认关掉 mock 登录，而不是默认打开）。
  3. 真正的手机号登录需要接短信验证码（一次性、带 TTL、可限流）；在真短信接入前，生产应直接下线该端点（`allowMockLogin=false` 已支持，但必须确保配置到位）。
  4. 加「登录失败限流 + 同手机号尝试计数」。

### HIGH-4｜`activate` 出的摄影师 `avatar` 为 NULL → 对方「我的订单」「我的收藏」整列表 500

- **file:line**
  - 上游根因：`comic/server/internal/repository/photographers.sql.go:165-169`
    ```sql
    INSERT INTO photographers (name, description, mode, mutual_intro, user_id, activated_at)
    VALUES ($1,$2,$3,$4,$5,NOW())   -- 不写 avatar / location
    ```
    `photographers.avatar` 列 nullable 且无默认（`\d photographers` 确认）。
  - 崩溃点 1：`comic/server/internal/repository/bookings.sql.go:170-179` `getBookingsByUserWithDetails` 选 `p.avatar AS photographer_avatar`，**无 COALESCE**，扫描目标 `models.go:135-142 BookingWithDetails{PhotographerAvatar string}`（非指针）→ NULL 扫描失败 → 整个列表 500。
  - 崩溃点 2：`comic/server/internal/repository/favorites.sql.go:37` `p.avatar`（无 COALESCE），扫描目标 `favorites.sql.go:7-15 FavoriteRow{Avatar string}`（非指针）→ 收藏列表 500。
  - 对照：其余扫描 `*string`/有 COALESCE 的位置都是安全的 —— `users.sql.go:15`（列 NOT NULL DEFAULT ''）、`chat.sql.go:61`、`user_blocks.sql.go:100`、`bookings.sql.go:201`（均已 COALESCE）。**全库只有上面两处漏了。**
- **复现（已验证）**
  ```bash
  # 1) 用户 154 开通摄影师身份
  curl -s -X POST -H "Authorization: Bearer $T_COSER_B" -H 'Content-Type: application/json' \
    -d '{"name":"NULL头像摄影师","mode":"both","intro":"x"}' \
    http://localhost:8088/api/v1/photographers/activate      # → 201 {"photographerId":9}
  # DB: photographers.id=9, avatar IS NULL, user_id=154
  # 2) 用户 148 给该摄影师下单
  curl -s -X POST -H "Authorization: Bearer $T_COSER_A" -H 'Content-Type: application/json' \
    -d '{"photographerId":9,"serviceId":39,"date":"2027-03-01","time":"10:00"}' \
    http://localhost:8088/api/v1/bookings                        # → 201
  # 3) 用户 148 打开自己的订单列表
  curl -s -H "Authorization: Bearer $T_COSER_A" http://localhost:8088/api/v1/bookings/148
  # → 500 {"error":"internal server error"}   ← 整页挂掉
  # 4) 收藏同样
  curl -s -X POST -H "Authorization: Bearer $T_COSER_A" -H 'Content-Type: application/json' \
    -d '{"photographerId":9}' http://localhost:8088/api/v1/favorites
  curl -s -H "Authorization: Bearer $T_COSER_A" http://localhost:8088/api/v1/favorites/148
  # → 500 {"error":"internal server error"}
  ```
  对照组：`GET /api/v1/photographers/9` → 200（`avatar:""`，service 层 `derefString` 兜底）；`GET /api/v1/bookings/photographer/9` → 200（SQL 用了 `COALESCE(u.name,'')`）。同一批 NULL 数据，走不同 SQL 就是 200 或 500，取决于扫描目标是不是指针。
  另外，遗留摄影师（`user_id` 为 NULL）同样触发 500：`photographerUserId` 被 `derefInt64` 转成 0。
- **影响**：任何一个用户自助开通摄影师身份后，其 avatar 即为 NULL；此后**所有与该摄影师有过订单/收藏的用户**，其「我的订单」「我的收藏」整页 500（不是单条降级，是整列表挂掉）。属于可被普通用户触发的可用性事故 + 轻度 DoS。
- **建议修法**
  1. 两处 SQL 补 `COALESCE(p.avatar,'')`（`bookings.sql.go:172`、`favorites.sql.go:37`）。
  2. `insertPhotographer` 写入默认头像（`/static/img/avatar-user.svg`）与默认 location，或给 `photographers.avatar` 加 `DEFAULT ''` + `NOT NULL`（配套迁移）。
  3. 扫描目标统一改成 `*string`（或全部用 `derefString`），消除「同类字段一个指针一个非指针」的隐患。
  4. 补一条「activate 后下单 → 订单列表不 500」的集成测试。

### MEDIUM-1｜手机号登录每次都覆盖用户自己改过的昵称与头像（数据破坏）

- **file:line**：`comic/server/internal/service/auth_service.go:87-89` 生成 `name = fmt.Sprintf("用户%s", phone 后4位)` → `comic/server/internal/repository/users.sql.go:27-37`
  ```sql
  INSERT INTO users (...) VALUES (...)
  ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, avatar = EXCLUDED.avatar
  ```
- **复现（已验证）**：把 u148 的 name 改成「我的昵称ABC」、avatar 改成 `custom.png` → 重新 `POST /api/v1/login` → DB 立刻变回 `用户0001` + `/static/img/avatar-user.svg`。
- **影响**：用户改一次昵称，下次登录就被系统改回去；头像同理。功能上「改了保存不了」，是真实的数据破坏。（WeChat 登录走 `UpsertByWeChat`，空串保留旧资料，不受影响。）
- **建议修法**：`ON CONFLICT (phone) DO UPDATE` 里只更新「首次插入时缺失的字段」（如 `COALESCE(NULLIF(users.name,''), EXCLUDED.name)`），不要覆盖已有值；昵称生成逻辑放到 INSERT 分支。

### MEDIUM-2｜并发开通摄影师身份会生成重复档案（无 UNIQUE 约束 + 读-判-写竞态）

- **file:line**：`comic/server/internal/service/photographer_service.go:135-158` `Activate` = 先 `GetPhotographerByUserID`（有则返回）再 `INSERT`；DB 侧 `pg_indexes` 只有 `idx_photographers_user`（**普通索引，非 UNIQUE**）。
- **复现（已验证）**：8 个并发 `POST /api/v1/photographers/activate` 打同一全新 user 195 → 返回 4 个不同 `photographerId`（10/11/12/13），DB 里 `user_id=195` 有 **4 条摄影师档案**。对**已激活**用户并发则幂等返回同一 id。
- **影响**：`/me` 只回一个 photographerId（实测 u195 → 10），档案 11/12/13 成为孤儿；`GET /v1/bookings/photographer/11` → 403 forbidden（`getPhotographerByUserID` 是 `:one`，多行报错 → ErrForbidden），这 3 个档案的套餐与订单不可达；且将来要加 `UNIQUE(user_id)` 需先清理这批重复行。
- **建议修法**：迁移加 `CREATE UNIQUE INDEX uniq_photographers_user ON photographers(user_id) WHERE user_id IS NOT NULL`（partial，兼容遗留 NULL 行）；`Activate` 的 INSERT 捕获 23505 → 重查并返回既有 id。

### MEDIUM-3｜前端把「摄影师档案 id」当「users.id」用，会聊错人

- **file:line**
  - `comic/src/pages/photographer/detail.vue:283-297`：`otherUserId: Number(photographer.value?.userId || id)` —— `id` 是 **photographer 档案 id**，`userId` 才是 users.id
  - `comic/src/pages/order/list.vue:258`、`comic/src/pages/order/detail.vue:312`：同一 fallback
- **复现（已验证）**：`GET /api/v1/photographers/7` → `{"id":7,"name":"遗留摄影师","userId":0}`（`photographers.user_id` 为 NULL → service 层 `derefInt64` → 0）→ 前端 `0 || 7` → **7**；而 `users.id=7` 是「用户0005」(13700000005)，与「遗留摄影师」毫无关系。模拟前端：`POST /v1/chat/sessions {otherUserId:7}` → 201，DB `chat_sessions(user1_id=7,user2_id=148)` —— **会话建到了无关用户身上**。
- **影响**：与「摄影师」沟通时可能把消息发给陌生用户（内容泄露风险）；「联系摄影师」按钮点开就是错的会话。
- **建议修法**：后端对 `user_id IS NULL` 的档案明确返回 `userId: null`（不要返回 0，0 会被 `||` 当假值吞掉）；前端删除 `|| id` fallback，改用 `photographer.userId` 且 `> 0` 才可发起会话，否则提示「该摄影师暂不支持私信」。

### MEDIUM-4｜CORS 全开（`Access-Control-Allow-Origin: *` 且允许 Authorization 头）

- **file:line**：`comic/server/cmd/api/main.go:33-44` `corsMiddleware`
- **影响**：任意第三方站点可用用户 token 跨域调 API。当前 token 存于 `uni.getStorageSync('token')`（`src/api/client.ts`），H5 端等价 localStorage —— 一旦 C 端存在 XSS 或被投毒，token 可被外带并跨域使用，叠加 HIGH-2（旧 token 不可吊销）后果放大。本身不构成越权（需先拿到 token）。
- **建议修法**：白名单 `Access-Control-Allow-Origin`（按环境配置），`Allow-Credentials` 与 `*` 不得同用；H5 端考虑改用 `HttpOnly` cookie 或至少缩短 token TTL 并提供「登出全部设备」。

### MEDIUM-5｜订单 id 存在性 oracle（403 vs 404）

- **file:line**：`comic/server/internal/service/booking_service.go:318-324`（先 `GetBookingByID` → 404）+ `:325-340`（再判归属 → 403）
- **复现（已验证）**
  ```bash
  curl -s -o /dev/null -w '%{http_code}\n' -X PUT -H "Authorization: Bearer $T_COSER_A" \
    -H 'Content-Type: application/json' -d '{"status":"cancelled"}' \
    http://localhost:8088/api/v1/bookings/2/status        # → 403 forbidden（存在但非本人）
  curl -s -o /dev/null -w '%{http_code}\n' -X PUT -H "Authorization: Bearer $T_COSER_A" \
    -H 'Content-Type: application/json' -d '{"status":"cancelled"}' \
    http://localhost:8088/api/v1/bookings/99999/status     # → 404 booking not found（不存在）
  ```
- **影响**：可枚举订单 id 的存在性与总量（含他人订单），但**读不到任何内容**（remarks / 手机号 / 金额都取不到，因为归属校验在返回数据之前）。属低强度信息泄露。
- **建议修法**：对「存在但非本人」与「不存在」统一返回 404（不区分）；或对归属失败一律 404。

### MEDIUM-6｜`POST /api/v1/notifications` 允许用户伪造「系统通知」

- **file:line**：`comic/server/internal/handler/notification_handler.go:22-46` —— type/title/content 完全由客户端指定，无服务端签发标记、无 type 白名单。
- **复现（已验证）**：`POST /api/v1/notifications {userId:148, type:"success", title:"系统通知", content:"点击链接领取奖励"}` → 201，落库 `user_id=154`（归属正确 ✔），但**内容与 title 由用户伪造**，前端消息页按 type 渲染图标（`notifyIcon`），看起来与系统通知无异。
- **影响**：只能写给自己（不能群发、不能写给别人），属「自我钓鱼/社工」级别，风险有限；但在共用设备场景可伪装成平台官方通知。
- **建议修法**：新增 `source` 列（`system` / `user`），消息页对 `source=user` 的条目用不同图标与文案前缀，或把 `POST /notifications` 收归内部服务（管理端 `POST /api/admin/v1/notifications` 已具备广播能力，C 端这个写入口实际可下线）。

### LOW-1｜`actorTag=""`（空串）等同于缺省 `coser`，不是拒绝

- `booking_handler.go:73-75`。实测 `actorTag=""` → 走 coser 分支（对本人订单 200，对他人订单 403）。未构成越权，但与「未知 tag 一律 403」的口径不一致，建议显式区分并记录。

### LOW-2｜`parseUserID` 解析失败静默返回 0

- `comic/server/internal/handler/authz.go:22-24`。用于 `follow_handler.go` 的 `assertSelf(c, parseUserID(userID))`：非数字 userId → 0 → 403（安全），但错误码应为 400。（`favorite_handler.go` / `chat_handler.go` / `notification_handler.go` 用 `strconv.ParseInt` 显式返 400，口径不一。）

### LOW-3｜`GET /api/v1/photographers/by-user/:userId` 无 `assertSelf`

- `handler/photographer_handler.go:132-148`。返回的是公开档案（name/mode/certified/tags/services/works/reviews），**不含 phone**，实测确认。属公开数据，不构成越权；但与「路径里的 userId 一律要 assertSelf」的团队约定不一致，建议在注释里显式说明「此处为公开读」。

### LOW-4｜上传只校验扩展名白名单，无内容嗅探

- `handler/upload_handler.go`：仅 jpg/jpeg/png/gif/webp + 5MB 上限 + 随机文件名，无 magic bytes 校验。可上传伪装成图片的 HTML。实测 `html` → 400（扩展名被挡），但改名为 `.jpg` 的 HTML 会被接受。

### LOW-5｜允许自己跟自己建会话

- `POST /api/v1/chat/sessions {otherUserId: 自己}` → 201，DB `chat_sessions(user1_id=148,user2_id=148)`。无业务意义，属数据卫生问题。

### LOW-6｜admin 用户/摄影师列表与 CSV 导出含明文手机号

- `admin_export_handler.go:75/112`、`admin_manage.go:69/179`、`admin_user_repo.go:67/98`。在 admin 职责范围内，但导出 CSV 明文手机号建议加权限分级/水印/审计留痕。

---

## 四、修复优先级建议

| 优先级 | 条目 | 理由 |
|---|---|---|
| P0 | HIGH-3（prod APP_ENV + 验证码） | 生产环境任意手机号可被接管，风险最高、修复最便宜（加一行 compose + 改默认值） |
| P0 | HIGH-2（token 过滤 status） | 一行 SQL 修复，否则「禁用账号」功能无效 |
| P0 | HIGH-1（角色化状态机） | 直接影响评分/单量可信度与手机号遮罩，需改状态机 + 补守卫 |
| P1 | HIGH-4（NULL avatar 500） | 两处 `COALESCE` + 迁移补默认值，普通用户可触发 |
| P1 | MEDIUM-2（activate 唯一约束） | 加 partial UNIQUE，避免数据继续恶化 |
| P2 | MEDIUM-1 / MEDIUM-3 / MEDIUM-5 / MEDIUM-6 | 数据破坏、聊错人、信息泄露、伪造通知 |
| P3 | MEDIUM-4（CORS）、LOW-1..6 | 加固项 |

---

## 五、清理证明（测试数据已全部回滚）

评审期间新建的数据已全部删除，种子数据未受影响：

```
bookings        : 仅剩 seed 1,2,3（测试新增 34,37,39,40,41,42,43,44,45,46,47 已删）
services        : 仅剩 seed 1-20（测试新增 35,36,37,38,39,40,41 已删，含面议 price IS NULL 套餐）
photographers   : 仅剩 seed 1-5（测试新增 6,7,9,10,11,12,13 已删）
reviews         : 仅剩 seed 1,2（测试新增 5,7,9 已删）
users           : seed 1-7,147,148,154 保留（测试新增 187,189,195 已删）
notifications   : 剩 12 条（测试期 52 条已删）
chat_sessions   : 0 条（测试期 1-6 已删，chat_messages 级联删除）
photographer_favorites / event_follows / event_subscriptions / chat_read_state / user_blocks : 已清空
```

- u6（`13900001111`）的 `status` 已由 `disabled` 恢复为 `active`。
- u148 的昵称已恢复为 `用户0001`。
- 摄影师派生统计已按 `reviews`/`bookings` 事实重算回填（`rating`/`review_count`/`order_count`，等价于迁移 000030 的口径）：1→(5.0,2,0)、2→(0,0,0)、3→(0,0,1)、4→(0,0,0)；photographer 5 为 `local_demo.sql` 的演示数字，未改动。
- 仓库源码零改动：`git status --porcelain` 仅 `?? doc/review/`（本报告）。

---

## 六、与兄弟任务的交叉点

- **t1（状态机与并发）** 报告的「actorTag 伪造/空 tag/非法流转全部 403/409」与本报告 HIGH-1 **不冲突**：t1 验的是「冒充他人角色」（他人 token + 他人 tag），本报告是「用合法角色做该角色不该做的动作」（本人 token + 合法 tag，缺角色化转换规则）。建议在 t6 汇总时并列呈现，避免被读成重复。
- **t2（价格与面议）** 报告的 `respond` SQL 缺 `status='pending'` 条件，与本报告 HIGH-1 叠加会放大：coser 自确认后（status=confirmed）`price_status='quoted'` 的单仍可被 accept/reject SQL 复活。
- **t4（前端流程一致性）** 需同步确认：C 端页面（`order/list.vue:189`、`order/detail.vue:326`）目前**不传 actorTag**，依赖服务端默认 `coser`；若按 HIGH-1 收紧为「confirmed 仅摄影师」，前端 coser 侧无「确认接单」按钮（`order/list.vue:41-49` 按钮矩阵中确实没有），行为一致、无需改前端；但 `order/list.vue` 的「确认完成」按钮（confirmed → completed）会与新规则冲突，需产品确认 completed 的归属。
