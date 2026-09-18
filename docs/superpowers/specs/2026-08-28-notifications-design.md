# P3 消息中心真实化 — 设计文档

**日期**: 2026-08-28
**状态**: 设计定稿（用户确认 C 包后做 B 包）
**范围**: 米拉漫展小程序 — 消息页通知区从硬编码 demo 改为真实通知（订单事件 → 通知表 → 消息页）

## 1. 背景

消息页通知区目前是硬编码 3 条 demo（"预约成功/新消息/订单提醒"），且挂着"demo 演示数据 · 正式版将接入真实通知"横幅。会话列表已接真实（Task 8），**通知是最后一块 demo**。做完后全站零 demo 残留。

## 2. 目标

1. 后端 `notifications` 表 + 读写 API
2. 事件写通知：预约成功（coser）、状态变更（取消/完成，通知 coser + 摄影师）
3. 消息页通知区接真实 API，移除硬编码数组 + demo 横幅

**明确不做**：推送/站外通知（需 AppID）；已读/未读状态管理（read 字段预留，前端暂不操作）；通知删除。

## 3. 架构决策

### 3.1 DB（迁移 000011_notifications）

```sql
CREATE TABLE IF NOT EXISTS notifications (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  type VARCHAR(20) NOT NULL DEFAULT 'info',      -- success/info/warning
  title VARCHAR(100) NOT NULL,
  content TEXT NOT NULL,
  read BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
```

### 3.2 后端

- repository: `InsertNotification(ctx, userID int64, type, title, content string) (int64, error)` + `ListNotifications(ctx, userID int64) ([]NotificationRow, error)`（created_at DESC 限 20 条）
- service: `NotificationService.Create/List`
- handler: `POST /api/v1/notifications`（内部/测试用）+ `GET /api/v1/notifications/:userId`（AuthRequired）
- **事件触发点**（booking_service）：
  - `Create` 成功后 → 给 coser 通知"预约成功，等待摄影师确认"
  - `UpdateStatus` 各迁移成功 → 通知对应各方：
    - confirmed → 通知 coser "摄影师已确认接单"
    - cancelled → 通知 coser（摄影师拒绝 or 自己取消）
    - completed → 通知摄影师 "订单已完成"
  - demo 简化：通知只写用户侧（coser 和 photographer 的用户 id）

### 3.3 前端

- message/index.vue：`notifications` 改从 `apiGet('/v1/notifications/:userId')` 加载；移除硬编码数组 + demo 横幅；onShow 刷新（已有模式）
- 时间格式化复用 formatSessionTime 思路（显示 MM/DD HH:mm）

## 4. 数据流

```
预约成功 → POST /events? 否——服务内直接 InsertNotification(coser)
  → 消息页 onShow → GET /notifications/:userId → 列表渲染
状态变更 → InsertNotification(相关方) → 同上
```

## 5. 错误处理

- 通知写入失败不阻断主流程（日志 + 忽略）
- 401 → client.ts 跳登录
- 空列表 → 空状态"暂无通知"

## 6. 测试策略

- 后端: Go 测试 Insert/List + 事件触发（mock repo）
- 前端: vue-tsc + Playwright 实测（预约后消息页出现"预约成功"通知；状态变更后出现对应通知）
- 视觉: agent-browser 截图消息页（真实通知，无 demo 横幅）

## 7. 里程碑

- B1: 迁移 + repo + service + handler + 事件触发
- B2: 前端消息页接真实 API + 移除 demo
- B3: 验证 + 视觉复查 + AGENTS.md + 推送

## 8. 假设记录

1. 通知写入 demo 简化（绑定事件关键方）
2. read 字段预留，前端暂不操作已读
3. 推送/站外通知后置（AppID）
