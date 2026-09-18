# photographer-notifications - Work Plan

## TL;DR (For humans)

**What you'll get:** 摄影师在「被预约 / 对方取消 / 管理员改单」时会收到站内通知（现在只有 coser 收得到）；同时堵住「通知接口信任客户端 userId」的越权口子。

**Why this approach:** 通知链路（`notify()` → `notifications` 表 → 消息页）已经跑通，本批只在其上加「摄影师」这一侧，纯后端增量，无新表/新端点/前端改动。

**What it will NOT do:** 不做推送/WebSocket/已读回执；不做通知撤回/定时；不改管理端通知端点。

**Effort:** Small（3 个 task）
**Risk:** Low — 模式已存在；主要风险是「档案 id ≠ 用户 id」的解析与 NULL 守卫（已在设计中显式处理）。

Your next move: approve and run `$start-work`, or ask questions.

---

> TL;DR (machine): Small effort, Low risk. Backend-only. 3 tasks in 2 waves.

## Scope

### Must have
- `booking_service.go` 新增 `notifyPhotographer(photographerID int32, ...)`：`GetPhotographerById(...).UserID` 解析、nil 安全、best-effort。
- `Create()` 补发摄影师「收到新预约」。
- `UpdateStatus()`：`cancelled` 且 actor=coser → 补发摄影师「预约已取消」；actor=photographer 不通知摄影师。
- `AdminUpdateStatus()`：confirmed/cancelled/completed 补发摄影师通知。
- 通知端点改用 token 身份：`POST /v1/notifications` 忽略 body userId；`GET /v1/notifications/:userId` 与 token 不符 → 403。
- 单测 5 例（见 spec）+ curl E2E。

### Won't have
- 推送 / WebSocket / 未读回执；通知撤回/定时/富文本；`/api/admin/v1/notifications` 改动；任何 schema 变更。

### Design Notes (來自 Metis 審查 / 自行核實)
- **`photographer_id ≠ users.id`**：`bookings.photographer_id` 指向 `photographers.id`；`notify()` 要 users.id。必须经 `GetPhotographerById(...).UserID` 解析（`*int64`，可 NULL）。
- **NULL 守卫**：历史 seed 摄影师 `user_id` 可能为 NULL → 跳过，不 panic、不写 user 0。
- **actor 感知**：避免自通知噪音（摄影师自己确认/完成不再通知自己）。
- **admin 路径易漏**：`AdminUpdateStatus` 是 `UpdateStatus` 的近似副本，两处都要改。
- **端点加固无破坏**：已核对 C 端消息页传的是本人 id；无客户端调用 POST。

## Tasks

### Wave 1

#### Task 1 — 摄影师侧通知核心
- 文件：`server/internal/service/booking_service.go`、`server/internal/service/booking_service_test.go`
- 内容：
  1. 新增 `notifyPhotographer(ctx, photographerID int32, typ, title, content string)`（nil 安全解析）。
  2. `Create()` 末尾补 `notifyPhotographer(req.PhotographerID, "info", "收到新预约", "您收到一条新预约(#<id>)，请及时确认接单")`。
  3. `UpdateStatus()` 的 `cancelled` 分支：actor=="coser" 时补 `notifyPhotographer(b.PhotographerID, "warning", "预约已取消", "对方取消了预约。")`。
- 验收：`go build ./... && go vet ./...`；新增单测（Create 双收、coser 取消摄影师收、摄影师拒单不收、NULL 不崩）通过。
- 依赖：无。

### Wave 2

#### Task 2 — admin 对称通知
- 文件：`server/internal/service/booking_service.go`、`server/internal/service/booking_service_test.go`
- 内容：`AdminUpdateStatus()` 的 confirmed/cancelled/completed 三处，在现有 coser 通知之外补 `notifyPhotographer`；补一条单测（admin confirm → 双方都收）。
- 验收：`go test ./internal/service/...` 全绿。
- 依赖：Task 1（helper）。

#### Task 3 — 通知端点加固 + 全链验证
- 文件：`server/internal/handler/notification_handler.go`、`server/cmd/api/main.go`（如需）、`AGENTS.md`
- 内容：
  1. `POST /v1/notifications`：忽略 body `userId`，用 `middleware.UserID(c)`。
  2. `GET /v1/notifications/:userId`：path userId ≠ token userID → 403。
  3. `go build`/`vet`/`test ./...`；curl E2E（coser 下单→摄影师收通知；他人越权读→403）；更新 AGENTS.md。
- 验收：E2E 通过；AGENTS.md 记录。
- 依赖：Task 1/2。

## Verification (final)
- `cd server && go build ./... && go vet ./... && go test ./...`
- 重启 dev 后端（`:8088`）→ curl E2E。
- 必要时 `vue-tsc`（本批无前端改动，可跳过）。
