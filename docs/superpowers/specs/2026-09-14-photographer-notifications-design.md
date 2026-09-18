# 摄影师侧通知 + 通知端点加固 — 设计

**日期:** 2026-09-14
**状态:** 已定稿（用户授权自主推进；评审：Metis 有效，Oracle 因模型故障输出损坏，改为自行逐条核实）

## 背景与问题

1. **摄影师侧通知缺失。** 预约的两个参与方是 coser 与摄影师，但 `server/internal/service/booking_service.go` 的通知**只发 coser**：`Create`、`UpdateStatus`、`AdminUpdateStatus` 三处共 4 个 `notify(b.CoserID, ...)` 调用点。摄影师被预约后**收不到任何通知**，只能主动打开「接单管理」页才发现——核心交易闭环的供给侧是静默的。
2. **通知端点信任客户端 userId（P2 遗留）。** `POST /api/v1/notifications` 与 `GET /api/v1/notifications/:userId` 直接用请求里的 userId，登录用户可给任意用户写通知、读任意用户通知。

## 目标

- 摄影师在关键节点收到通知：新预约、coser 取消、管理员驱动的流转。
- 通知端点改用 token 的身份（`middleware.UserID`），不再信任客户端 userId。

## 非目标（YAGNI）

- 不做 WebSocket / 移动推送 / 已读回执（依赖微信域名，且范围过大）。
- 不新增数据表、不新增通知类型、不新增端点。

## 关键约束（实现者最容易踩的隐藏需求）

- `bookings.photographer_id` 是**摄影师档案 id**（`photographers.id`），**不是** `users.id`；而 `notify()` 的形参是 **user id**。必须用 `GetPhotographerById(ctx, photographerID).UserID` 解析。
- `PhotographerWithTags.UserID` 类型是 `*int64`，**可能为 NULL**（migration 000009 之前的历史 seed 摄影师）→ 必须跳过通知，**不得 panic、不得写到 user id 0**。
- `notify()` 形参是 `int32`，`UserID` 是 `*int64` → 需要 deref + 类型转换。
- **actor 感知**：摄影师自己触发的流转不得再通知摄影师（避免自通知噪音）。
- 通知写入失败必须 best-effort（沿用现有 log-and-ignore，绝不阻塞预约主流程）。

## 方案（已选 A）

### A. 后端最小增量（选用）

- `booking_service` 新增 helper：
  ```go
  func (s *BookingService) notifyPhotographer(ctx context.Context, photographerID int32, typ, title, content string) {
      p, err := s.queries.GetPhotographerById(ctx, photographerID)
      if err != nil || p.UserID == nil { return }
      s.notify(ctx, int32(*p.UserID), typ, title, content)
  }
  ```
- `Create()`：在现有 coser 通知之后，补
  `notifyPhotographer(req.PhotographerID, "info", "收到新预约", fmt.Sprintf("您收到一条新预约(#%d)，请及时确认接单", booking.ID))`。
- `UpdateStatus()`：
  - `cancelled` 且 `actorTag == "coser"` → `notifyPhotographer(b.PhotographerID, "warning", "预约已取消", "对方取消了预约。")`。
  - `cancelled` 且 `actorTag == "photographer"`（拒单）→ **不**通知摄影师；coser 既有通知保持不变。
  - `confirmed` / `completed` → 不变（摄影师是 actor）。
- `AdminUpdateStatus()`：`confirmed` / `cancelled` / `completed` 三处，在现有 coser 通知之外**补发摄影师通知**（admin 既非 coser 也非摄影师，双方都不应被蒙在鼓里）。

### B. 抽象 `BookingNotifier` 接口做 actor 路由（否决）

对本项目属过度设计（YAGNI）；现有 `notify()` 模式已足够。

### C. 通知中心 / 推送 / WebSocket（否决）

依赖微信小程序域名白名单与 AppID 审批，且范围过大；不在本批。

## 通知端点加固（同域小项）

- `POST /api/v1/notifications`：**忽略** body 里的 `userId`，写入 token 对应用户。
- `GET /api/v1/notifications/:userId`：若 path 的 userId ≠ token 的 userID → **403**；否则返回本人通知。

## 数据流

预约事件 → `booking_service` → `notify()` / `notifyPhotographer()` → `InsertNotification(user_id)` → C 端消息页 `GET /v1/notifications/:userId`（消息页已按登录用户的 id 拉取，**摄影师无需前端改动即可看到**）。

## 错误处理

- `GetPhotographerById` 失败或 `UserID == nil` → 跳过（可记日志），不影响预约成功/状态流转。
- `InsertNotification` 失败 → log-and-ignore（与现有 `notify()` 一致）。

## 测试

单元测试（`booking_service_test.go`，通过 notifier seam/fake 断言「谁收到了什么」）：

1. `Create` → coser 收到原「预约成功」，摄影师收到「收到新预约」。
2. coser（actorTag=coser）取消 → 摄影师收到「预约已取消」。
3. 摄影师（actorTag=photographer）拒单 → 摄影师**不**收到新通知（actor 感知）。
4. `AdminUpdateStatus` confirm → coser 与摄影师**都**收到。
5. 摄影师 `user_id = NULL` → 不 panic、不写入。

端到端（curl，dev 栈）：

- coser 登录 → `POST /v1/bookings` → `GET /v1/notifications/<摄影师 user_id>` 含「收到新预约」。
- 摄影师登录 → 确认接单 → coser 侧通知仍正常。
- 加固：用 A 的 token 调 `GET /v1/notifications/<B 的 user_id>` → 403。

## 兼容性

- C 端消息页传的就是本人 id → 加固后不受影响。
- 无客户端调用 `POST /v1/notifications`（预约通知由服务端直连 repo 写入）→ 加固无破坏。
- 管理端通知端点 `/api/admin/v1/notifications` 不受影响。

## 风险与缓解

| 风险 | 级别 | 缓解 |
|------|------|------|
| 过度通知（通知了 actor 自己） | 低 | actor 守卫 |
| 传错 id（把 `photographer_id` 当 user id） | 高（静默 bug） | 显式解析 + 单测覆盖 |
| NULL `user_id` 崩溃 | 中 | nil 守卫 + 跳过 |
| coser 侧通知被回归 | 低 | 不改动现有 coser 调用点，只新增 |
| 端点加固误伤消费方 | 低 | 已核对无客户端依赖（见「兼容性」） |

## 落地拆解（3 个 task）

1. **核心**：`notifyPhotographer` helper + 接入 `Create` 与 `UpdateStatus`(coser 取消)。
2. **对称 + 测试**：`AdminUpdateStatus` 补通知；新增上述 5 个单测用例。
3. **加固 + 验证**：通知端点改用 token 身份；`go build`/`vet`/`test ./...`；curl E2E 全链路；更新 AGENTS.md。
