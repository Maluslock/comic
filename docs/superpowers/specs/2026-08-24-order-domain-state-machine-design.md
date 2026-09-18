# 订单域核心：状态机 + 时间冲突 + 订单联查 — 设计文档

**日期**: 2026-08-24
**状态**: 设计定稿（无人值守模式，假设用户认可方案 A 交易闭环的子项目 P1，可随时叫停调整）
**范围**: 米拉漫展小程序后端 + 前端 — 让"预约 → 状态流转 → 完成"成为真实可用的业务闭环

## 1. 背景与问题

全链路贯通（2026-08-24 批次）后，预约创建与订单查看已接真实 API，但**订单状态流转仍是假的**：

- `src/pages/order/detail.vue` 的 `cancelOrder`/`confirmOrder` 只调用 `updateStoredOrderStatus()` 改本地 `uni.storage`，无任何后端调用——刷新/换设备即丢失，摄影师侧永远看不到状态变化
- `src/pages/order/list.vue` 同样只有本地兜底合并，无真实状态操作
- 后端 `bookings` 表有 `status` 字段（默认 `pending`）但**没有更新 API、没有状态机校验**
- 预约创建无冲突检测：`src/pages/booking/index.vue` 的 `isTimeDisabled` 恒返回 `false`，同一摄影师同一时段可被无限重复预约
- 订单列表/详情缺摄影师姓名/头像联查（`BookingItem` 只有 `photographer_id`，前端显示"摄影师#id"）
- 预约提交成功后仍写本地 storage 兜底，双数据源易不一致

**最初需求对照**：预约撮合平台的核心是"订单能流转、时段不冲突、双方看得懂"——这三项都是业务基本盘，必须真实。

## 2. 目标

让 coser 侧交易闭环真实可用：
1. 订单状态可通过 API 流转（取消/确认完成），前后端一致
2. 预约创建时检测时间冲突，同一摄影师同一时段不可重复预约
3. 订单列表/详情显示真实摄影师姓名/头像/服务名

**明确不做**（YAGNI）：支付、真实 WebSocket、摄影师侧入驻/接单、通知域（P2）、评价关联订单（P3）。

## 3. 架构决策

### 3.1 订单状态机（后端）

状态集合：`pending`（待确认）→ `confirmed`（已确认）→ `completed`（已完成）；`pending`/`confirmed` → `cancelled`（已取消，终态）。

| 当前状态 | 允许迁移 |
|---------|---------|
| pending | confirmed, cancelled |
| confirmed | completed, cancelled |
| completed | （终态，无迁移） |
| cancelled | （终态，无迁移） |

非法迁移返回 HTTP 409 + 明确错误信息。

**API**: `PUT /api/v1/bookings/:id/status`，请求体 `{ "status": "confirmed" | "completed" | "cancelled" }`。

**鉴权**: 走 `middleware.AuthRequired`（批 1 已实现）。校验操作者是该订单的 coser 或摄影师（demo 阶段：coser 可 cancelled/completed；摄影师确认逻辑 P3 再做——本轮允许 coser 直接 confirmed，作为 demo 简化，注明）。

**前端 pending→confirmed 触发**（demo 闭环必需，否则订单永远停在 pending 无法到达"确认完成"）: order/detail 与 order/list 在订单状态为 `pending` 时显示"确认接单"按钮（标注为演示用，模拟摄影师确认），点击调 `PUT /status {confirmed}`。按钮规则:
- pending → 显示 [确认接单(演示)] [取消预约]
- confirmed → 显示 [确认完成] [取消预约]
- completed → 显示 [去评价]
- cancelled → 无操作按钮

### 3.2 时间冲突检测（后端）

- 新增 `GET /api/v1/photographers/:id/timeslots?date=YYYY-MM-DD`：返回该摄影师当天已被占用的时段数组（排除 cancelled 状态的订单）
- `POST /api/v1/bookings` 的 `Create` 中：查同 photographer_id + date + time 且 status != 'cancelled' 的订单，若存在 → 返回 409 `{"error": "该时段已被预约"}`
- 前端 booking 页 `loadTimeSlots` 改调真实 API，`disabledTimes` 从响应填充（替代硬编码 `['10:00', '15:00']`）

### 3.3 订单联查摄影师（后端 + 前端）

- `ListByUser` 返回的 `BookingItem` 增加 `photographerName`、`photographerAvatar`（SQL LEFT JOIN photographers）
- 前端 order/list、order/detail 移除"摄影师#id"兜底逻辑，直接用联查字段
- 本地 storage 兜底仅作**只读** fallback（后端不可达时展示上次缓存），不再用于状态写入

### 3.4 前端状态操作真实化

- `order/detail.vue`：`cancelOrder`/`confirmOrder` 改调 `apiPut('/v1/bookings/:id/status')`（需在 client.ts 增加 `apiPut`），成功后刷新订单数据，失败 toast；新增 pending 状态的"确认接单"按钮（演示用）
- `order/list.vue`：同样的按钮改调真实 API；tab 切换后重新拉取
- 移除 `updateStoredOrderStatus` 本地写入逻辑

## 4. 数据流

```
order/detail 取消 → apiPut /bookings/:id/status {cancelled}
  → 后端状态机校验 → UPDATE bookings SET status
  → 前端 reload 订单 → 状态显示"已取消"

booking 选时间 → GET /photographers/:id/timeslots?date
  → 返回占用时段 → 前端灰色禁用
  → 提交 → POST /bookings → 后端冲突检测 → 409 或 201
```

## 5. 错误处理

- 409 冲突（非法状态迁移 / 时段被占）→ 前端 toast 具体错误
- 404 订单不存在 → toast "订单不存在"
- 401 未登录 → 现有 client.ts 统一处理跳登录
- 5xx → "服务异常"

## 6. 测试策略

- 后端: Go 测试——状态机迁移表（合法/非法全组合）、冲突检测（占用→409，取消后→可约）、联查字段存在
- 前端: vue-tsc 类型检查 + Playwright H5 实测：预约 → 订单列表 → 取消 → 状态刷新为已取消；同一时段二次预约被拒
- 回归: 首页/漫展/摄影师链路不受影响；`go test ./...` 全绿

## 7. 里程碑

- M1: 后端——apiPut 客户端 + 状态机 API + 冲突检测 + timeslots API + 联查字段
- M2: 前端——order 两页状态操作真实化 + booking 时间槽真实化 + 移除本地状态写入
- M3: 全链路验证 + AGENTS.md 更新 + 推送

## 8. 假设记录（无人值守）

1. 方向 = 方案 A 交易闭环，本轮只做子项目 P1（订单域核心）
2. demo 阶段 coser 可直接确认订单（摄影师接单流程 P3 再做）
3. 通知/评价/摄影师侧不在本轮范围
4. 每个里程碑完成即提交推送 Gitee
