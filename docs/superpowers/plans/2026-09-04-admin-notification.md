# P2c 后台通知公告管理实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理端发布通知公告（全员/指定用户）→ C 端消息页收到；复用 P3 通知链路，0 新表。

**Architecture:** `notifications.user_id` 改可空（000016）+ repo 加 `InsertBroadcast`（user_id NULL）+ C 端 `ListNotifications` 改 `WHERE (user_id = $1 OR user_id IS NULL)` + service `Publish`（校验 type/targetType/single 用户存在）+ 2 管理 API（发布/历史）；前端 notification/index.vue（发布表单 + 历史列表），复用现有 admin 模式。

**Tech Stack:** Go 1.22 + Gin + pgx（sqlc-style Queries）；Vue3.5 + Vite8 + NaiveUI（soybean-admin）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；迁移在 `server/migrations/`（**000016**）。
- 000016：`ALTER TABLE notifications ALTER COLUMN user_id DROP NOT NULL;`（down：重新 SET NOT NULL——注意 NULL 行需清理，down.sql 用 `DELETE FROM notifications WHERE user_id IS NULL` 前置）。
- **C 端查询改动（唯一 C 端文件）**：`notifications.sql.go` `ListNotifications` SQL → `WHERE (user_id = $1 OR user_id IS NULL)`——保持 `LIMIT 20 ORDER BY created_at DESC`。
- `NotificationRow.UserID` 为 `int64`，广播行 user_id NULL → **scan 需改 `*int64`**（或 `pgtype.Int8`）——必须处理（否则 scan NULL 到 int64 报错）。
- adminGroup：`POST /admin/v1/notifications`（发布）、`GET /admin/v1/notifications`（历史列表，分页）。
- type 仅 `success`/`info`/`warning`；targetType 仅 `all`/`single`；single 必填 userId（存在性校验，不存在 404）。
- 前端视图目录 `notification/index.vue`（route key `notification`，菜单「通知公告」）；menuIcons `notification: 'mdi:bell-outline'`；locale zh-cn `notification: '通知公告'` / en-us `Notification`；elegant-router 产物提交。
- `pnpm typecheck` green gate；浅色主题；中文 UI；单根 `<div>`。
- 服务重启（server/ 目录，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- 管理员 admin/admin123；C 端测试用户 13800138000（user 1，coser，phone 13800138000）。
- 无新第三方依赖。所有 go build/vet/test + pnpm typecheck 必须过。
- 测试数据清理：E2E/冒烟发的广播公告会留在 C 端消息页——用完后 SQL 删除（`DELETE FROM notifications WHERE title LIKE '测试公告%'`）或保留为演示（披露）。

---

### Task 1: 后端通知广播（迁移 + repo + C 端查询 + Publish + 2 管理 API）

**Files:**
- Create: `server/migrations/000016_notification_broadcast.up.sql` / `.down.sql`
- Modify: `server/internal/repository/notifications.sql.go`（ListNotifications 条件 + Row.UserID *int64 + InsertBroadcast）
- Create: `server/internal/service/notification_admin_service.go`（Publish/ListHistory）
- Create: `server/internal/handler/admin_notification_handler.go`（2 API）
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/notification_admin_service_test.go`

**Interfaces:**
- Consumes: `service.NotificationService`（已有 Create/List——admin 服务可用它 List 或直接调 repo）；`repository.UserRepo.GetByID`（single 用户存在校验）；`adminGroup`（Use 之后）
- Produces:
  - repo（notifications.sql.go 扩展，sqlc-style 手写）：
    - `InsertBroadcast(ctx, typ, title, content string) (int64, error)` — `INSERT INTO notifications (type, title, content) VALUES ($1,$2,$3) RETURNING id`（user_id 默认 NULL）
    - `ListNotifications` SQL 改 + `NotificationRow.UserID` 改 `*int64`（scan `&i.UserID` 兼容 NULL）
    - `ListAllNotifications(ctx, limit, offset int) ([]NotificationRow, int64, error)` — 历史（含 broadcast），`ORDER BY created_at DESC LIMIT $1 OFFSET $2` + count
  - service `NotificationAdminService`：
    - `Publish(ctx, req PublishRequest) (int64, error)` — 校验 type/targetType；single 校验 userId 存在（UserRepo.GetByID，ErrUserNotFound → ErrUserNotFound 透传）；all → repo.InsertBroadcast；single → NotificationService.Create
    - `ListHistory(ctx, page, pageSize int) ([]HistoryItem, int64, error)` — HistoryItem{ID,Type,Title,Content,UserID *int64,TargetType (derived: UserID==nil→"all" else "single"),CreatedAt}
  - `PublishRequest{Type, Title, Content, TargetType string, UserID *int64}`
  - sentinels：`ErrInvalidType`、`ErrInvalidTargetType`（service）
  - handler `AdminNotificationHandler`：`Publish(c)`、`List(c)`
- 错误映射：404 ErrUserNotFound（single 无效用户）；400 ErrInvalidType/ErrInvalidTargetType/binding；409 无（广播无冲突）。

- [ ] **Step 1: 迁移**

`000016_notification_broadcast.up.sql`:
```sql
ALTER TABLE notifications ALTER COLUMN user_id DROP NOT NULL;
```
`down.sql`:
```sql
DELETE FROM notifications WHERE user_id IS NULL;
ALTER TABLE notifications ALTER COLUMN user_id SET NOT NULL;
```
执行 up + 验证：`docker exec comic-pg psql -U comic -d comic -t -c "SELECT is_nullable FROM information_schema.columns WHERE table_name='notifications' AND column_name='user_id';"` → YES

- [ ] **Step 2: repo 扩展**（notifications.sql.go）
```go
// NotificationRow.UserID 改 *int64（broadcast NULL 可扫）
// ListNotifications: WHERE (user_id = $1 OR user_id IS NULL)
// 新增 InsertBroadcast + ListAllNotifications（见 Interfaces）
```

- [ ] **Step 3: service Publish/ListHistory**（notification_admin_service.go；UserRepo 构造注入）

- [ ] **Step 4: handler**（admin_notification_handler.go，参照 admin_user_handler.go 的 query/path 解析 + json binding）

- [ ] **Step 5: 注册路由**（main.go adminGroup Use 之后，admin 路由后）：
```go
notifAdminSvc := service.NewNotificationAdminService(queries, notificationSvc, userRepo)
notifAdminH := handler.NewAdminNotificationHandler(notifAdminSvc)
adminGroup.POST("/notifications", notifAdminH.Publish)
adminGroup.GET("/notifications", notifAdminH.List)
```
（注：`notificationSvc` 已在 main.go 存在；`userRepo` 也存在。）

- [ ] **Step 6: 测试**（fake store 风格）：
- TestPublish_All：TargetType=all → InsertBroadcast 被调
- TestPublish_Single：TargetType=single+userId 有效 → NotificationService.Create 被调
- TestPublish_Single_UserNotFound：userId 无效 → ErrUserNotFound
- TestPublish_InvalidType：type=banana → ErrInvalidType
- TestPublish_InvalidTargetType：targetType=group → ErrInvalidTargetType
- TestListHistory_Broadcast：UserID=nil 行 → TargetType=all

- [ ] **Step 7: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/... -count=1
# 重启（Global Constraints 配方）
TOKEN=$(curl -s -X POST localhost:8080/api/admin/v1/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s -X POST localhost:8080/api/admin/v1/notifications -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"type":"info","title":"测试公告","content":"全员必读","targetType":"all"}'   # 201
curl -s "localhost:8080/api/admin/v1/notifications?page=1&pageSize=20" -H "Authorization: Bearer $TOKEN"    # 200 含该公告 targetType=all
# C 端验证：登录 13800138000 → GET /api/v1/notifications/1 → 含"测试公告"（broadcast 可见）
curl -s -X POST localhost:8080/api/admin/v1/notifications -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"type":"warning","title":"指定公告","content":"仅指定用户","targetType":"single","userId":1}'   # 201
curl -s -X POST localhost:8080/api/admin/v1/notifications -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"type":"info","title":"坏目标","content":"x","targetType":"single","userId":999999}'   # 404
curl -s -X POST localhost:8080/api/admin/v1/notifications -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"type":"banana","title":"坏类型","content":"x","targetType":"all"}'   # 400
# 清理：docker exec comic-pg psql -U comic -d comic -c "DELETE FROM notifications WHERE title IN ('测试公告','指定公告')"
```
提交：`git commit -m "feat(admin): notification broadcast (publish all/single + history + C-end reads broadcast)"`；NO push。

---

### Task 2: 前端 notification/index.vue

**Files:**
- Create: `admin/views/notification/index.vue`
- Modify: `admin/src/service/api/admin.ts`（publishNotification/fetchNotifications）
- Modify: `admin/src/typings/api/admin.d.ts`（NotificationItem/HistoryItem 类型）
- Modify: `admin/build/plugins/router.ts`（menuIcons notification）
- Modify: `admin/src/locales/langs/zh-cn.ts` + `en-us.ts`（route.notification）
- （elegant-router 产物提交）

**Interfaces:**
- Consumes: Task 1 全部 API（POST /notifications、GET /notifications?page=&pageSize=）
- Produces: 可用的通知公告页

- [ ] **Step 1: api/admin.ts 加函数**：`publishNotification(payload: {type,title,content,targetType:'all'|'single',userId?})` → POST；`fetchNotifications(params)` → GET，`?? {list:[],total:0}`
- [ ] **Step 2: admin.d.ts** — NotificationItem {id,type,title,content,userId:number|null,targetType:'all'|'single',createdAt}
- [ ] **Step 3: notification/index.vue** — 发布表单 NCard（类型 NSelect 成功/提示/警告、标题 NInput 必填、内容 textarea 必填、目标 NRadio 全员/指定用户；指定用户时 NSelect 用户（fetchAdminUsers 前 20 或搜索））+ NDataTable 历史（标题/类型 tag/目标 tag 全员|指定/时间/remote pagination）+ 发布成功 message + reload
- [ ] **Step 4: 菜单/locale** — `notification: 'mdi:bell-outline'`；zh-cn `notification: '通知公告'`；en-us `Notification`；elegant-router regenerate
- [ ] **Step 5: typecheck + 浏览器验证** — `pnpm typecheck` exit 0；:5174（admin/admin123 单会话）：菜单「通知公告」；发布表单可用（测试公告 all）→ 历史列表出现；指定用户（13800138000 用户 id=1）→ 列表 targetType=single；清理测试公告（SQL）或披露
- [ ] **Step 6: Commit** — `git commit -m "feat(admin-web): notification broadcast page"`；NO push

---

### Task 3: 验证 + E2E（双端）+ 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c 补《通知公告管理》章节）
- Test-only

- [ ] **Step 1: 后端全量** — build/vet/test 全绿
- [ ] **Step 2: Playwright E2E（双端）**：
  ① admin :5174 登录 → 通知公告页 → 发布全员公告（测试公告）→ 历史出现
  ② C 端 :5173 登录 13800138000 → 消息页 → 系统通知出现「测试公告」
  ③ 发布指定用户公告（userId=1）→ 消息页出现；另一用户（10000000001 摄影师）看不到
  ④ 清理测试公告（SQL）或披露
- [ ] **Step 3: agent-browser 视觉复查** — 通知公告页截图（浅色专业风、发布表单/历史表格渲染、中文）
- [ ] **Step 4: AGENTS.md** — 加「通知公告管理」小节：000016、2 API、C 端 ListNotifications 广播可见、后续（撤回/定时/富文本/已读率）
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: admin notification broadcast in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`；验证 unpushed=0

---

## 风险与决策记录

- **NotificationRow.UserID *int64**：broadcast 行 user_id NULL，scan int64 会报错——必须改 *int64（或 pgtype.Int8）。所有使用 NotificationRow 的地方（P3 消息页 List mapping）需适配（服务层 map 时 *int64 → 前端 userId 可能 null）。
- **C 端 ListNotifications 改动**：唯一 C 端后端文件改——P3 消息页前端零改动（仍调 GET /v1/notifications/:userId），但返回的 notification.userId 可能为 null（broadcast）——前端 mapping `id: String(n.id)` 等不变，userId 字段若前端未用可忽略（P3 消息页 mapping 只用 id/type/title/content/read/time——安全）。
- **down.sql 需清理 NULL 行**：加 `DELETE FROM notifications WHERE user_id IS NULL` 前置，否则 SET NOT NULL 失败。
- **不建新表**：broadcast 行直接存 notifications（user_id NULL），复用 P3 消息页零改动。
- **single 目标用户校验**：UserRepo.GetByID 检查存在 → 404；User 结构已含 Status（Task P0 加的）——GetByID 返回完整 User。
