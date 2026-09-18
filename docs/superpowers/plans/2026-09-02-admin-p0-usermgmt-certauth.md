# P2c 后台 P0 补全（账号管理 + 认证申请审核）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐后台 P0 双模块：用户账号管理（列表/详情/封禁）+ 摄影师认证申请审核（C 端申请 → 管理端审批 → 黄V 生效 + 通知）。

**Architecture:** 后端沿用既有三层（handler→service→repository）+ AdminManageRepo 独立实例模式：users 加 status 列（login 拒 disabled）+ 新增 cert_applications 表（审批事务联动 certified + notifications）；前端 admin/ 新增 user/index.vue 与 certification/index.vue，复用 table/tabs/remote-pagination 模式。

**Tech Stack:** Go 1.22 + Gin + pgx + bcrypt（已有）；Vue3.5 + Vite8 + NaiveUI + UnoCSS（soybean-admin 改造）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；迁移在 `server/migrations/`（000013+，**不是** server/db/）。
- 迁移执行：`docker exec -i comic-pg psql -U comic -d comic < server/migrations/<file>`。
- 管理 API 前缀 `/api/admin/v1`，除 login 外全部 `AdminAuthRequired`；C 端 API 前缀 `/api/v1`。
- `users.status` 取值：`active` / `disabled`（VARCHAR(10) NOT NULL DEFAULT 'active'）。
- cert_applications.status 取值：`pending` / `approved` / `rejected`。
- C 端 login：`status='disabled'` → 403 `{"error":"account disabled"}`（AuthService.Login 加校验，`User` struct 加 Status 字段）。
- 一摄影师一活跃申请：pending/approved 时 C 端 cert-apply 返回 409；rejected 可重新申请。
- 审批事务：approve → application.status + `photographers.certified=true` + notifications（success）；reject → status + reason + notifications（warning）；非 pending → 409。
- admin 路由注册：main.go 的 `adminGroup`（已有，Use(AdminAuthRequired) 之后继续加）。
- 前端：`pnpm typecheck` 是 green gate（node_modules 35 条基线除外）；admin dev :5174；浅色主题；中文 UI。
- 服务重启（从 server/ 目录，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- 管理员测试账密：`admin` / `admin123`。C 端测试账号：摄影师 `10000000001`（验证码 `123456`，对应摄影师 id=1 光影行者）。
- 所有 `go build ./...` + `go vet ./...` + `go test ./internal/...` 必须过。无新第三方依赖。

---

### Task 1: 后端 users.status + 账号管理 API

**Files:**
- Create: `server/migrations/000013_users_status.up.sql`
- Create: `server/migrations/000013_users_status.down.sql`
- Modify: `server/internal/repository/users.sql.go`（User struct + Status；UpsertByPhone/GetByID 查询带 status）
- Modify: `server/internal/service/auth_service.go`（Login 校验 status）
- Create: `server/internal/repository/admin_user_repo.go`
- Create: `server/internal/service/admin_user_service.go`
- Create: `server/internal/handler/admin_user_handler.go`
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/admin_user_service_test.go`

**Interfaces:**
- Consumes: `repository.NewAdminRepo`（Task 1 P2c 已有）；`middleware.AdminAuthRequired`；`middleware.AdminID`
- Produces:
  - `repository.AdminUserRepo`（`NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo`）：
    - `ListUsers(ctx, keyword string, role string, status string, limit, offset int) ([]AdminUser, int64, error)` → AdminUser{ID, Phone, Name, Avatar, Role, Status, CreatedAt, PhotographerID *int64}（role 由 photographers.user_id LEFT JOIN；keyword 匹配 phone/name ILIKE）
    - `GetUserDetail(ctx, id int64) (*AdminUserDetail, error)` → AdminUserDetail{用户 + Stats{BookingsCount, ReviewsCount, FavoritesCount, FollowsCount} + RecentBookings[]AdminUserBooking}（RecentBookings join service+photographer，取 5）
    - `SetUserStatus(ctx, id int64, status string) error`（RowsAffected=0 → ErrManageNotFound）
  - `service.NewAdminUserService(repo *AdminUserRepo) *AdminUserService`：
    - `List(ctx, keyword, role, status string, page, pageSize int) ([]AdminUser, int64, error)`（page 默认 1，pageSize 默认 20）
    - `Detail(ctx, id int64) (*AdminUserDetail, error)`
    - `SetStatus(ctx, id int64, status string) error`（校验 status ∈ {active, disabled}，否则 ErrInvalidStatus）
  - `handler.NewAdminUserHandler(svc *AdminUserService) *AdminUserHandler`：`List(c)`、`Detail(c)`、`SetStatus(c)`
- `User` struct 新增 `Status string` 字段；`UpsertByPhone`/`GetByID` 的 SELECT/RETURNING 加 `status`；`GetUserByToken` SELECT 也加 status（保持三处一致）
- `AuthService.Login`：UpsertByPhone 后 `if u.Status == "disabled" { return nil, ErrAccountDisabled }`（新增 sentinel `var ErrAccountDisabled = errors.New("account disabled")`），handler 层映射 403

- [ ] **Step 1: 迁移**

`server/migrations/000013_users_status.up.sql`:
```sql
ALTER TABLE users ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';
```
`000013_users_status.down.sql`:
```sql
ALTER TABLE users DROP COLUMN IF EXISTS status;
```
执行：`docker exec -i comic-pg psql -U comic -d comic < server/migrations/000013_users_status.up.sql`
验证：`docker exec comic-pg psql -U comic -d comic -t -c "SELECT column_name FROM information_schema.columns WHERE table_name='users' AND column_name='status';"` → `status`

- [ ] **Step 2: User struct + 查询带 status**

`users.sql.go`: User struct 加 `Status string \`json:"status"\``；UpsertByPhone RETURNING 尾部加 `, status` 与 scan 加 `&u.Status`；GetByID SELECT 加 `status` + scan；GetUserByToken（若其 SELECT 缺 status）同样补。

- [ ] **Step 3: Login 403 校验**

`auth_service.go`：
```go
var ErrAccountDisabled = errors.New("account disabled")
```
Login 中 UpsertByPhone 成功后：
```go
if u.Status == "disabled" {
    return nil, ErrAccountDisabled
}
```
`handler/auth.go` Login：`if errors.Is(err, service.ErrAccountDisabled) { c.JSON(403, gin.H{"error": "account disabled"}); return }`（在现有 500/400 处理之前）。

- [ ] **Step 4: AdminUserRepo**

`server/internal/repository/admin_user_repo.go`（参照 admin_manage.go 模式，独立 *pgxpool.Pool 实例）:
```go
type AdminUser struct {
	ID            int64  `json:"id"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Avatar        string `json:"avatar"`
	Role          string `json:"role"` // 'photographer' | 'coser'（LEFT JOIN photographers 推导）| 'coser' 默认
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	PhotographerID *int64 `json:"photographerId,omitempty"`
}
type AdminUserBooking struct {
	ID               int64  `json:"id"`
	Status           string `json:"status"`
	Date             string `json:"date"`
	Time             string `json:"time"`
	Price            int32  `json:"price"`
	PhotographerName string `json:"photographerName"`
	ServiceName      string `json:"serviceName"`
}
type AdminUserDetail struct {
	AdminUser
	Stats struct {
		BookingsCount  int64 `json:"bookingsCount"`
		ReviewsCount   int64 `json:"reviewsCount"`
		FavoritesCount int64 `json:"favoritesCount"`
		FollowsCount   int64 `json:"followsCount"`
	} `json:"stats"`
	RecentBookings []AdminUserBooking `json:"recentBookings"`
}
```
SQL 要点（动态 WHERE，全部参数化）：
- List: `FROM users u LEFT JOIN photographers p ON p.user_id = u.id`；`WHERE ($1::text = '' OR u.phone ILIKE '%'||$1||'%' OR u.name ILIKE '%'||$1||'%') AND ($2::text = '' OR $2::text = 'all' OR ($2::text = 'photographer' AND p.user_id IS NOT NULL) OR ($2::text = 'coser' AND p.user_id IS NULL)) AND ($3::text = '' OR $3::text = 'all' OR u.status = $3)` ORDER BY u.id DESC，`LIMIT $4 OFFSET $5`；count 同 WHERE 单独查询。
- GetUserDetail: 各项 count + RecentBookings `LEFT JOIN photographers p ON p.id=b.photographer_id LEFT JOIN services s ON s.id=b.service_id ORDER BY b.created_at DESC LIMIT 5`。
- SetUserStatus: `UPDATE users SET status=$2 WHERE id=$1`；RowsAffected==0 → ErrManageNotFound（复用 admin_manage.go 的 sentinel）。

- [ ] **Step 5: AdminUserService**

`server/internal/service/admin_user_service.go`：
```go
var ErrInvalidStatus = errors.New("invalid status") // 不导出的 sentinel 可放 service 包
type AdminUserService struct{ repo *repository.AdminUserRepo }
func NewAdminUserService(repo *repository.AdminUserRepo) *AdminUserService
func (s *AdminUserService) List(ctx, keyword, role, statusQuery string, page, pageSize int) ([]repository.AdminUser, int64, error)
func (s *AdminUserService) Detail(ctx, id int64) (*repository.AdminUserDetail, error)
func (s *AdminUserService) SetStatus(ctx, id int64, status string) error // 校验 active/disabled
```

- [ ] **Step 6: AdminUserHandler**

`server/internal/handler/admin_user_handler.go`：
```go
type AdminUserHandler struct{ svc *service.AdminUserService }
func NewAdminUserHandler(svc) *AdminUserHandler
func (h *AdminUserHandler) List(c)   // query: keyword, role, status, page, pageSize → 200 {list, total}
func (h *AdminUserHandler) Detail(c) // path id → 200 AdminUserDetail | 404
func (h *AdminUserHandler) SetStatus(c) // body {status} → 200 {ok} | 400 非法 | 404
```

- [ ] **Step 7: 注册路由**

main.go adminGroup（Use 之后）：
```go
adminUserRepo := repository.NewAdminUserRepo(pool)
adminUserSvc := service.NewAdminUserService(adminUserRepo)
adminUserH := handler.NewAdminUserHandler(adminUserSvc)
adminGroup.GET("/users", adminUserH.List)
adminGroup.GET("/users/:id", adminUserH.Detail)
adminGroup.PUT("/users/:id/status", adminUserH.SetStatus)
```

- [ ] **Step 8: 测试**

`server/internal/service/admin_user_service_test.go`（沿用 admin_manage_service_test.go 的 fake row 脚本风格）：
- TestSetStatus_Valid：SetStatus(1,"disabled") → nil
- TestSetStatus_Invalid：SetStatus(1,"banana") → ErrInvalidStatus
- TestSetStatus_NotFound：repo RowsAffected=0 路径 → ErrManageNotFound
- TestAuthLogin_Disabled：mock repo 返回 Status="disabled" → ErrAccountDisabled（或集成：真库造一个 disabled 用户 login → 403）

- [ ] **Step 9: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/...
# 重启 + 冒烟（见 Global Constraints）
TOKEN=$(curl -s -X POST localhost:8080/api/admin/v1/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s localhost:8080/api/admin/v1/users -H "Authorization: Bearer $TOKEN"             # 200, list 16 用户
curl -s localhost:8080/api/admin/v1/users/1 -H "Authorization: Bearer $TOKEN"          # 200, 详情含 stats/recentBookings
curl -s -X PUT localhost:8080/api/admin/v1/users/16/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"disabled"}'   # 200
curl -s -X POST localhost:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"<161xxx phone>","code":"123456"}'   # 403 account disabled（用一个真实存在的手机号）
curl -s -X PUT localhost:8080/api/admin/v1/users/16/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"active"}'    # 200 恢复
```
注意：Login 验证码逻辑是 `req.Code[:4] != "1234"`（非 123456）——测试用 code="123456" 通过，或按现有逻辑用任意前缀 1234 的码实测。

提交：`git add ... && git commit -m "feat(admin): user management API (list/detail/status) + login disabled guard"`
NO push。

---

### Task 2: 后端认证申请（C 端 cert-apply + 管理端 3 API + 审批事务）

**Files:**
- Create: `server/migrations/000014_cert_applications.up.sql`
- Create: `server/migrations/000014_cert_applications.down.sql`
- Create: `server/internal/repository/cert_application_repo.go`
- Create: `server/internal/service/cert_application_service.go`
- Create: `server/internal/handler/cert_application_handler.go`（C 端 cert-apply）
- Create: `server/internal/handler/admin_cert_handler.go`（管理端 3 API）
- Modify: `server/cmd/api/main.go`
- Modify: `server/internal/service/notification_service.go`（如需要 bulk/broadcast——按 spec 本设计只通知单用户，复用 Create）
- Test: `server/internal/service/cert_application_service_test.go`

**Interfaces:**
- Consumes: `middleware.AuthRequired`（C 端）、`middleware.AdminAuthRequired`、`middleware.UserID`；`photographers`（GetPhotographerByUserID，Queries 已有）、`NotificationService.Create(ctx, userID int64, typ, title, content string) (int64, error)`（已有）
- Produces:
  - `repository.NewCertApplicationRepo(pool *pgxpool.Pool) *CertApplicationRepo`：
    - `Create(ctx, userID, photographerID int64, evidenceImages []string, evidenceDesc string) (int64, error)`（INSERT RETURNING id；调用前 service 查重）
    - `GetByPhotographerID(ctx, photographerID int64) (*CertApplication, error)` → 存在则返回（用于重复检测 + 详情复用）
    - `List(ctx, status string, limit, offset int) ([]AdminCertApplication, int64, error)` → AdminCertApplication{ID, UserID, UserName, PhotographerID, PhotographerName, EvidenceImages []string, EvidenceDesc, Status, CreatedAt, ReviewReason *string, AdminID *int64, ReviewedAt *time.Time}
    - `GetByID(ctx, id int64) (*AdminCertApplication, error)`（RowsAffected/no rows → ErrManageNotFound）
    - `Review(ctx, id int64, action string, reason string, adminID int64) error`（UPDATE ... WHERE status='pending'；RowsAffected=0 → 若存在但非 pending 由 service 判 409；不存在 → ErrManageNotFound）
    - `SetCertifiedByPhotographerID(ctx, photographerID int64, certified bool) error`（UPDATE photographers SET certified=$2 WHERE id=$1）
  - `service.NewCertApplicationService(repo *CertApplicationRepo, photographerRepo *Queries, notifySvc *NotificationService) *CertApplicationService`：
    - `Submit(ctx, userID int64, evidenceImages []string, evidenceDesc string) (int64, error)`：查 GetByPhotographerID（用 photographerID）→ 若 status ∈ {pending, approved} → ErrAlreadyApplied；否则 repo.Create；非摄影师用户（无 photographer 记录）→ ErrNotPhotographer
    - `List(ctx, status string, page, pageSize int) ([]AdminCertApplication, int64, error)`
    - `Detail(ctx, id int64) (*AdminCertApplication, error)`
    - `Review(ctx, id int64, action string, reason string, adminID int64) error`：action ∈ {approve, reject} else ErrInvalidAction；approve → 事务；reject → reason 必填（空 → ErrReasonRequired）
  - `handler.NewCertApplicationHandler(svc) *CertApplicationHandler`：`Submit(c)`（C 端 POST /api/v1/photographers/cert-apply）
  - `handler.NewAdminCertHandler(svc) *AdminCertHandler`：`List(c)`、`Detail(c)`、`Review(c)`
  - sentinels：`ErrAlreadyApplied`、`ErrNotPhotographer`、`ErrInvalidAction`、`ErrReasonRequired`（service 包）
- 审批事务（service.Review 内）：GetByID → status != pending → ErrAlreadyReviewed(409)；approve：`repo.Review(id,"approved","",adminID)` + `repo.SetCertifiedByPhotographerID(pid,true)` + `notifySvc.Create(ctx, userID, "success", "认证通过", "恭喜！您的摄影师认证已通过，黄V 徽章已生效")`；reject：`repo.Review(id,"rejected",reason,adminID)` + `notifySvc.Create(ctx, userID, "warning", "认证未通过", "很遗憾，您的认证申请未通过。" + reason)`。写通知失败仅 log 不阻断（同 booking notify 模式）。

- [ ] **Step 1: 迁移**

`server/migrations/000014_cert_applications.up.sql`:
```sql
CREATE TABLE photographer_cert_applications (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  photographer_id BIGINT NOT NULL REFERENCES photographers(id),
  evidence_images TEXT[] NOT NULL,
  evidence_desc TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  review_reason TEXT,
  admin_id BIGINT REFERENCES admins(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cert_app_user ON photographer_cert_applications(user_id);
CREATE INDEX idx_cert_app_status ON photographer_cert_applications(status, created_at DESC);
CREATE UNIQUE INDEX idx_cert_app_photographer_active ON photographer_cert_applications(photographer_id) WHERE status <> 'rejected';
```
`down.sql`: `DROP TABLE IF EXISTS photographer_cert_applications;`
执行 + 验证表存在。

- [ ] **Step 2: CertApplicationRepo**（参照 admin_manage.go 的独立 *pgxpool.Pool 风格；[]string 扫描用 pgx 的 `[]string` 支持 + `pgtype` TextArray fallback 处理）

- [ ] **Step 3: CertApplicationService**（含 Submit/List/Detail/Review 全逻辑 + sentinels）

- [ ] **Step 4: C 端 Handler + 管理端 Handler**

- [ ] **Step 5: 注册路由**

main.go：
```go
certRepo := repository.NewCertApplicationRepo(pool)
certSvc := service.NewCertApplicationService(certRepo, queries, notificationSvc)  // notificationSvc 已存在
certH := handler.NewCertApplicationHandler(certSvc)
adminCertH := handler.NewAdminCertHandler(certSvc)
router.POST("/api/v1/photographers/cert-apply", middleware.AuthRequired(userRepo), certH.Submit)
adminGroup.GET("/cert-applications", adminCertH.List)
adminGroup.GET("/cert-applications/:id", adminCertH.Detail)
adminGroup.PUT("/cert-applications/:id/review", adminCertH.Review)
```

- [ ] **Step 6: 测试**

`cert_application_service_test.go`（fake row 风格）：
- TestSubmit_AlreadyApplied（pending 存在 → ErrAlreadyApplied）
- TestSubmit_NewUser（无申请 → id）
- TestReview_Approve（事务联动：Review 调 SetCertifiedByPhotographerID + notify）
- TestReview_Reject_NoReason → ErrReasonRequired
- TestReview_NonPending → ErrAlreadyReviewed

- [ ] **Step 7: 构建 + 冒烟 + 提交**

冒烟（Global Constraints 重启后）：
```bash
# C 端登录摄影师 10000000001 → tokenC
curl -s -X POST localhost:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"10000000001","code":"123456"}' → tokenC
curl -s -X POST localhost:8080/api/v1/photographers/cert-apply -H "Authorization: Bearer $tokenC" -H 'Content-Type: application/json' -d '{"evidenceImages":["https://picsum.photos/600/450"],"evidenceDesc":"3年拍摄经验，擅长日系古风"}' → 201 {id}
# 管理端
curl -s "localhost:8080/api/admin/v1/cert-applications?status=pending" -H "Authorization: Bearer $TOKEN" → 200 含该申请
curl -s -X PUT localhost:8080/api/admin/v1/cert-applications/:id/review -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"action":"approve"}' → 200
docker exec comic-pg psql -U comic -d comic -t -c "SELECT certified FROM photographers WHERE id=1" → t
curl -s localhost:8080/api/v1/photographers/1 -H "Authorization: Bearer $tokenC" → certified=true
```
提交：`git commit -m "feat(admin): cert application (C-end submit + admin review + certified+notify transaction)"`

---

### Task 3: 前端 user/index.vue + certification/index.vue

**Files:**
- Create: `admin/src/views/user/index.vue`
- Create: `admin/src/views/certification/index.vue`
- Modify: `admin/src/service/api/admin.ts`（加 fetchAdminUsers/fetchAdminUserDetail/setUserStatus/fetchCertApplications/fetchCertApplicationDetail/reviewCertApplication）
- Modify（路由生成依赖视图目录）: 无手工路由——elegant-router auto-gen；确认 build/plugins/router.ts onRouteMetaGen 对 user/certification 有 icon（若 menuIcons 缺 key 则加，参照 Task 4 P2c 修复的 guard 模式）

**Interfaces:**
- Consumes: 全部 Task 1-2 API（/api/admin/v1/users、/users/:id、/users/:id/status、/cert-applications、/cert-applications/:id、/cert-applications/:id/review）
- Produces: 可用的两个管理页

- [ ] **Step 1: api/admin.ts 加函数**

（参照现有 fetchAdminPhotographers 模式）：
```ts
export async function fetchAdminUsers(params: Record<string, any>) { ... GET /v1/users, `?? {list:[], total:0}` 兜底 }
export async function fetchAdminUserDetail(id: string|number) { ... GET /v1/users/:id }
export async function setUserStatus(id: string|number, status: string) { ... PUT /v1/users/:id/status, `{status}` }
export async function fetchCertApplications(params) { ... GET /v1/cert-applications, `?? {list:[], total:0}` }
export async function fetchCertApplicationDetail(id) { ... GET /v1/cert-applications/:id }
export async function reviewCertApplication(id, body: {action:'approve'|'reject', reason?:string}) { ... PUT /v1/cert-applications/:id/review }
```
（路由无手工注册——elegant-router 由视图目录自动生成，对 user/certification 生成菜单。若 onRouteMetaGen 的 menuIcons 缺这两个 key，仿 Task 4 的 guard `if (menuIcons[key])` 不加会生成 icon:{} 污染——已由 guard 排除，但为美观在 menuIcons 中加 `user: 'mdi:account-group', certification: 'mdi:star-check-outline'`。）

- [ ] **Step 2: user/index.vue**

搜索栏（关键字 NInput + 角色 NSelect 全部/摄影师/Coser + 状态 NSelect 全部/启用/禁用 + 查询/重置）、NDataTable（NAvatar/昵称/手机号/角色 tag/状态 tag/注册时间/操作）、操作列：详情 → NDrawer（NDescriptions 资料 + 4 统计卡 + 最近订单表）、封禁/解封 NDialog 确认。remote pagination（复用 event/index.vue 修复后的模式：`remote` + tablePagination computed + @update:page/@update:page-size）。

- [ ] **Step 3: certification/index.vue**

NTabs（待审核/已通过/已驳回）+ NDataTable（申请人/摄影师/时间/状态/操作）+ 操作：通过（NDialog 确认）、驳回（NModal 填理由必填）+ 详情 NDrawer（NImage 预览 evidenceImages + 说明）。remote pagination。

- [ ] **Step 4: pnpm typecheck + 浏览器验证**

```bash
cd /vol1/1000/code/comic/admin && pnpm typecheck   # exit 0
```
浏览器（:5174 admin/admin123）：菜单出现「用户管理」「认证审核」；用户页搜索/封禁/解封/详情可用；认证页在 C 端提交申请后有数据 → 审批通过 → C 端详情页黄V 可见。

- [ ] **Step 5: Commit**

`git add admin/src && git commit -m "feat(admin-web): user management + certification audit pages"`

---

### Task 4: 验证 + E2E + 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c 补全章节）
- Test-only 全链验证

- [ ] **Step 1: 后端全量测试** — `go build ./... && go vet ./... && go test ./...` 全绿
- [ ] **Step 2: Playwright E2E 全链**（后端 :8080 + admin :5174 + C 端 :5173）：
  ① 管理端登录 → 用户管理搜索「8000」→ 封禁用户 → C 端该用户 login 403 → 解封 → login 恢复
  ② C 端摄影师 10000000001 提交认证申请（含图片）→ 管理端认证审核看到 pending → 通过 → C 端摄影师详情 certified=true + 通知收到
  ③ 驳回流：另一摄影师提交 → 驳回填理由 → 通知收到 + 可重新申请
  ④ 恢复现场：测试中封禁/认证状态复位（未认证的恢复未认证，除非留作演示数据）
- [ ] **Step 3: agent-browser 视觉复查** — user 页 + certification 页截图（浅色专业风、中文渲染、表格/抽屉正常）
- [ ] **Step 4: AGENTS.md** — 加「账号管理 + 认证申请审核」小节（迁移 000013/000014、API 表、审批事务说明、测试账号、后续 P1/P2）
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: admin P0 modules in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`

---

## 风险与决策记录

- **验证码误读**：现有 `Login` 校验 `req.Code[:4] != "1234"`（前缀 1234 即可），冒烟用 `code:"123456"`（前缀 1234 ✓）——**不要按 AGENTS.md 的"123456"按字面写死判断**。
- **数据状态**：Task 5 P2c E2E 曾把订单 #3/#4 留在 confirmed——本计划冒烟/E2E 中如果触碰订单，尽量不动；认证流程测试用的摄影师（id=1）完成后**恢复 certified 状态**或保留为演示（Task 4 Step 3 明确）。
- **`users.status` 迁移**：000013 是对现有表加列，应用后历史用户全部 `active`（默认值）——无需回填。
- **cert_applications 唯一索引**：用部分唯一索引（`WHERE status <> 'rejected'`，PostgreSQL 支持）保证"一摄影师一活跃申请"且 rejected 可重提——SQL 必须用该写法，不用 Go 层先查后插的竞态方案。
- **notifications 广播**：本计划审批通知是**单用户**（复用现有 Create），不做全员广播（那是 P1 模块）。
