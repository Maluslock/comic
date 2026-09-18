# P3 消息中心真实化 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 消息页通知区从硬编码 demo 改为真实通知（订单事件 → notifications 表 → 消息页），全站零 demo 残留。

**Architecture:** 新增 notifications 表 + InsertNotification/ListNotifications repo + NotificationService + GET/POST API + booking_service 事件触发（Create/UpdateStatus 写通知）；前端消息页通知区接真实 API + 移除硬编码数组/demo 横幅。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue 3 script setup + uni-app。

## Global Constraints

- Go: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH`（构建/测试重启用）
- PG: `postgres://comic:comic123@127.0.0.1:5433/comic`；重启 API: `kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')` + `(setsid /tmp/mila-api ... &)`（从 server/ 目录启动含 .env）；**禁用 pkill**
- TS strict，禁新增 `as any`；rpx/SCSS/`@/`；无 emoji（✓ ∟ ⚠ 几何符号允许，但通知图标用几何：✓ 对应 success，●/i 对应 info/warning 用几何符号）
- 通知类型: success/info/warning；表名 `notifications`（迁移 000011，next after 000010）
- 事件触发 demo 简化：Create → coser 通知；UpdateStatus confirmed → coser；cancelled → coser；completed → photographer
- 通知写入失败不阻塞主流程（log + ignore）
- 每任务 commit；不 push

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `server/migrations/000011_notifications.{up,down}.sql` | notifications 表 |
| `server/internal/repository/notifications.sql.go` | InsertNotification + ListNotifications + NotificationRow |
| `server/internal/repository/querier.go` | 注册 |
| `server/internal/service/notification_service.go` | NotificationService.Create/List |
| `server/internal/service/booking_service.go` | Create/UpdateStatus 事件触发写通知 |
| `server/internal/handler/notification_handler.go` | POST + GET endpoints |
| `server/cmd/api/main.go` | 注册路由 |
| `src/pages/message/index.vue` | 通知区接真实 API + 移除 demo |

---

### Task 1: 迁移 + repo

- Create: `server/migrations/000011_notifications.up.sql` / `.down.sql`
- Create: `server/internal/repository/notifications.sql.go`（InsertNotification/ListNotifications/NotificationRow{ID,UserID,Type,Title,Content,Read,CreatedAt string fields per json tags}）
- Modify: `server/internal/repository/querier.go`

```sql
-- up
CREATE TABLE IF NOT EXISTS notifications (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  type VARCHAR(20) NOT NULL DEFAULT 'info',
  title VARCHAR(100) NOT NULL,
  content TEXT NOT NULL,
  read BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
-- down
DROP TABLE IF EXISTS notifications;
```

repo（notification.go 模式）：
- InsertNotification(ctx, userID int64, typ, title, content string) (int64, error) — INSERT RETURNING id
- ListNotifications(ctx, userID int64) ([]NotificationRow, error) — WHERE user_id=$1 ORDER BY created_at DESC LIMIT 20

验证: `docker exec -i comic-pg psql -U comic -d comic < ...up.sql` + `\dt` 显示 notifications；`go build ./...`
Commit: `feat(db,repo): notifications table + insert/list`

### Task 2: service + 事件触发

- Create: `server/internal/service/notification_service.go`（Create/List）
- Modify: `server/internal/service/booking_service.go`

```go
// notification_service.go
func (s *NotificationService) Create(ctx, userID int64, typ, title, content string) (int64, error) { return s.queries.InsertNotification(...) }
func (s *NotificationService) List(ctx, userID int64) ([]NotificationItem, error) { ... }
type NotificationItem struct { ID int64 json:"id"; Type string; Title string; Content string; Read bool; CreatedAt string }
```

booking_service 触发（注入 notifQueries 或直接 queries）：
- Create 成功后: `s.queries.InsertNotification(ctx, req.CoserID, "success", "预约成功", fmt.Sprintf("您的「%s」摄影预约已提交，等待摄影师确认", photographerName))`（photographerName 从 req 无——用 photographerId 简占位或查；指定: content 用 `"您的摄影预约(#%d)已提交，等待摄影师确认", booking.ID`）
- UpdateStatus: after success, switch newStatus:
  - confirmed → InsertNotification(coserID, "info", "预约已确认", "摄影师已确认接单，请按时赴约")
  - cancelled → InsertNotification(coserID, "warning", "预约已取消", "您的预约已被取消")
  - completed → InsertNotification(photographerUserID, "success", "订单已完成", "订单已完成，好评率+1")

注意: UpdateStatus 里取 coserID/photographerID from booking (b.CoserID/b.PhotographerID)；photographer 用户 id 需查 GetPhotographerByUserID or photographer_id→user_id join——demo: 用 booking.PhotographerID 作为 user_id 占位（seed 1:1 简化，标注）或查 photographers.user_id。**指定**: 查 `s.queries.GetPhotographerByUserID` 不适用（需反查）——用新增 repo `GetUserIDByPhotographer(ctx, pid) (int64, error)` 或简化: photographer 通知发给 booking's photographer 的 user_id，通过已有 photographers.user_id（photographer.id 即 booking.photographer_id → 查 photographers 行）。**方案**: notification 只发给 coser（demo 简化，photographer 通知 P2c 后置）——completed 不写 photographer 通知，或写 coser "订单已完成"确认。**最终指定（简洁）**：
- Create → coser "预约成功"
- confirmed → coser "摄影师已确认接单"
- cancelled → coser "预约已取消"
- completed → coser "拍摄已完成，记得评价哦"

Commit: `feat(service): notification service + booking event triggers`

### Task 3: handler + 路由

- Create: `server/internal/handler/notification_handler.go`（POST /api/v1/notifications {userId,type,title,content} → 201；GET /api/v1/notifications/:userId → 200）
- Modify: `cmd/api/main.go`（AuthRequired 两路由）

冒烟: login → POST notification → GET list 含记录
Commit: `feat(api): notifications endpoints`

### Task 4: 前端消息页

- Modify: `src/pages/message/index.vue` — 移除硬编码 notifications 数组 + demo 横幅；apiGet(`/v1/notifications/${userStore.user.id}`) → map {id,type,title,content,createdAt} → display（含 remove demo banner template block）；onShow 已有（sessions），通知同 onShow 加载；时间格式化 formatNotifyTime（MM/DD HH:mm）

Commit: `feat(message): real notifications via API, remove demo`

### Task 5: 验证 + 视觉 + AGENTS.md + 推送

- Playwright: 下单 → 消息页出现"预约成功"通知；确认接单（摄影师）→ coser 消息页出现"已确认"；取消 → "已取消"通知；无 demo 横幅
- 视觉: agent-browser 截图消息页（真实通知，无 demo）
- go test 全绿、vue-tsc、build:h5
- AGENTS.md: notifications 模块
- Commit + push

---

## 自审

- Spec 覆盖: 3.1 DB → T1；3.2 后端+触发 → T2/3；3.3 前端 → T4；M3 → T5
- 类型一致: NotificationItem/DTO 字段跨任务一致（id/type/title/content/read/createdAt）
- 取舍: photographer 通知后置（demo 只给 coser 写通知，completed 通知 coser 评价）；read 字段预留
