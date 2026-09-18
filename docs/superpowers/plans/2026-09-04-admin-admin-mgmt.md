# P2c 后台管理员管理实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理端管理员账号管理：列表/新建/禁用启用/重置密码；disabled 立即失效 token；种子 admin 不可禁用。

**Architecture:** 后端 admins 表加 status 列（000015 迁移）+ AdminRow.Status + Login/GetAdminByToken 校验 + 4 管理 API（独立 AdminAdminRepo 或扩展 AdminRepo）；前端 admin/views/admin/index.vue（表格+新建/重置弹窗+禁用启用确认），复用 user/index.vue 模式。

**Tech Stack:** Go 1.22 + Gin + pgx + bcrypt（已有）；Vue3.5 + Vite8 + NaiveUI（soybean-admin）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；迁移在 `server/migrations/`（**000015**，不是 server/db/）。
- `admins.status VARCHAR(10) NOT NULL DEFAULT 'active'`；值 active/disabled。
- `AdminRow` 加 `Status string` 字段；`adminCols` 常量加 `status`；scanAdmin 加 `&a.Status`；**三处查询**（GetAdminByUsername/GetAdminByToken/新增的 GetByID）带 status。
- `GetAdminByToken` 加 `AND status='active'`（disabled 旧 token 立即 401）。
- `AdminService.Login` 密码校验后加 `if a.Status=="disabled" { return "", nil, ErrInvalidCredentials }`（与密码错同响应）。
- 管理 API 前缀 `/api/admin/v1`，全部 `AdminAuthRequired`；adminGroup.Use 之后注册。
- **种子保护**：`id==1`（username='admin'）禁用 → 400 `{"error":"cannot disable primary admin"}`；重置密码允许。
- 字段映射：AdminRow 无 passwordHash 泄漏——列表/详情返回 **id/username/role/status/createdAt**（**绝不返回 password_hash**）。
- 前端：`admin/views/admin/index.vue`（elegant-router 自动路由 `%admin`？注意：视图目录名 `admin` 可能与路由常量冲突——**改用 `src/views/admin-accounts/index.vue`** 以生成 route key `admin-accounts`，避免与已有的 admin 相关命名冲突；菜单标题「管理员管理」）；menuIcons 加 `admin-accounts: 'mdi:account-cog-outline'`；locale zh-cn `admin-accounts: '管理员管理'` / en-us `Admin Management`；elegant-router 产物提交。
- `pnpm typecheck` 是 green gate（node_modules 35 条基线除外）；浅色主题；中文 UI；单根 `<div>` 模板约束。
- 服务重启（server/ 目录，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- 管理员：admin/admin123。无新第三方依赖。所有 go build/vet/test + pnpm typecheck 必须过。

---

### Task 1: 后端 admins.status + 管理 API

**Files:**
- Create: `server/migrations/000015_admins_status.up.sql` / `.down.sql`
- Modify: `server/internal/repository/admin.go`（AdminRow.Status + adminCols + scan + GetAdminByToken 过滤 + GetByID + SetStatus + ClearToken）
- Modify: `server/internal/service/admin_service.go`（Login 校验 + List/Create/SetStatus/ResetPassword）
- Create: `server/internal/handler/admin_admin_handler.go`（4 API）
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/admin_admin_service_test.go`

**Interfaces:**
- Consumes: `repository.NewAdminRepo`、`middleware.AdminAuthRequired`、`adminGroup`（main.go Use 之后）、`bcrypt.GenerateFromPassword`（service 已有依赖）
- Produces:
  - `repository.AdminRepo` 新增：
    - `GetByID(ctx, id int64) (*AdminRow, error)`（SELECT adminCols WHERE id=$1；ErrAdminNotFound）
    - `List(ctx) ([]AdminRow, error)`（SELECT adminCols ORDER BY id ASC）
    - `SetStatus(ctx, id int64, status string) error`（UPDATE admins SET status=$2 WHERE id=$1；RowsAffected=0 → ErrAdminNotFound）
    - `ClearToken(ctx, id int64) error`（UPDATE admins SET token=NULL, token_expires_at=NULL WHERE id=$1）
    - `Create(ctx, username, passwordHash, role string) (int64, error)`（INSERT RETURNING id；23505 → ErrUsernameExists）
  - `service.AdminService` 新增：
    - `List(ctx) ([]AdminAccount, error)` — AdminAccount{ID, Username, Role, Status, CreatedAt}（无 passwordHash）
    - `Create(ctx, username, password, role string) (int64, error)` — bcrypt 生成 hash；role 仅允许 admin/super else ErrInvalidRole；username 冲突返 ErrUsernameExists
    - `SetStatus(ctx, id int64, status string) error` — status ∈ active/disabled else ErrInvalidStatus；id==1 && disabled → ErrPrimaryAdmin
    - `ResetPassword(ctx, id int64, password string) error` — 新 hash + ClearToken
  - sentinels：`ErrUsernameExists`、`ErrInvalidRole`、`ErrInvalidStatus`、`ErrPrimaryAdmin`（service 包）
  - `handler.NewAdminAdminHandler(svc *AdminService) *AdminAdminHandler`：`List(c)`、`Create(c)`、`SetStatus(c)`、`ResetPassword(c)`
- 错误映射：404 ErrAdminNotFound；400 ErrInvalidStatus/ErrPrimaryAdmin/ErrInvalidRole/binding；409 ErrUsernameExists。

- [ ] **Step 1: 迁移**

`000015_admins_status.up.sql`: `ALTER TABLE admins ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';`
`down.sql`: `ALTER TABLE admins DROP COLUMN IF EXISTS status;`
执行：`docker exec -i comic-pg psql -U comic -d comic < server/migrations/000015_admins_status.up.sql`
验证：`docker exec comic-pg psql -U comic -d comic -t -c "SELECT column_name FROM information_schema.columns WHERE table_name='admins' AND column_name='status';"` → status

- [ ] **Step 2: AdminRepo 扩展**

`admin.go`：
```go
type AdminRow struct {
	ID             int64
	Username       string
	PasswordHash   string
	Role           string
	Status         string   // NEW
	Token          *string
	TokenExpiresAt *time.Time
}
const adminCols = "id, username, password_hash, role, status, token, token_expires_at"
// scanAdmin 加 &a.Status
// ErrUsernameExists 定义在 service 包（Step 3）；repo.Create 直接返回 pgx 错误，service 检测 23505 映射
// GetAdminByToken 改：
"SELECT "+adminCols+" FROM admins WHERE token = $1 AND token_expires_at > NOW() AND status = 'active'"
// 新增 GetByID/List/SetStatus/ClearToken/Create（SQL 见 Interfaces）
```

- [ ] **Step 3: AdminService 扩展**

`admin_service.go` 新增（sentinels：ErrUsernameExists/ErrInvalidRole/ErrInvalidStatus/ErrPrimaryAdmin 定义在 service 包）：
- `Login` 密码校验后：`if a.Status == "disabled" { return "", nil, ErrInvalidCredentials }`
- `List`/`Create`（bcrypt）/`SetStatus`（种子保护 id==1）/`ResetPassword`（hash + ClearToken）
- Create 的 23505 检测：repo.Create 返回 pgx 错误时 `strings.Contains(err.Error(), "23505")` → ErrUsernameExists（或 repo 层映射后 service 透传）。

- [ ] **Step 4: AdminAdminHandler**

`server/internal/handler/admin_admin_handler.go`（参照 admin_user_handler.go）：
- List → 200 `[]AdminAccount`
- Create bind `{username binding:"required", password binding:"required", role}` → 201 `{id}` | 400 | 409
- SetStatus bind `{status binding:"required"}` → 200 `{ok:true}` | 400（含种子保护）/404
- ResetPassword bind `{password binding:"required"}` → 200 `{ok:true}` | 404

- [ ] **Step 5: 注册路由**

main.go adminGroup（Use 之后，banner 路由后）：
```go
adminAdminH := handler.NewAdminAdminHandler(adminSvc)   // adminSvc 已存在（Task 1 P2c）
adminGroup.GET("/admins", adminAdminH.List)
adminGroup.POST("/admins", adminAdminH.Create)
adminGroup.PUT("/admins/:id/status", adminAdminH.SetStatus)
adminGroup.PUT("/admins/:id/password", adminAdminH.ResetPassword)
```

- [ ] **Step 6: 测试**

`admin_admin_service_test.go`（fake store 风格，参照 cert_application_service_test.go）：
- TestAdminLogin_Disabled：mock GetAdminByUsername 返回 Status=disabled → Login → ErrInvalidCredentials
- TestAdminSetStatus_PrimaryAdmin：SetStatus(1,"disabled") → ErrPrimaryAdmin（repo.SetStatus 不被调用）
- TestAdminSetStatus_Invalid：SetStatus(2,"banana") → ErrInvalidStatus
- TestAdminSetStatus_NotFound：repo RowsAffected=0 → ErrAdminNotFound
- TestAdminCreate_UsernameExists：repo 23505 → ErrUsernameExists
- TestAdminResetPassword_ClearsToken：成功时 repo.ClearToken 被调用

- [ ] **Step 7: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/... -count=1
# 重启 + 冒烟（Global Constraints 配方）
TOKEN=$(curl -s -X POST localhost:8080/api/admin/v1/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s localhost:8080/api/admin/v1/admins -H "Authorization: Bearer $TOKEN"                        # 200, [admin]
curl -s -X POST localhost:8080/api/admin/v1/admins -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"username":"testadmin","password":"admin123","role":"admin"}'   # 201
curl -s -X PUT localhost:8080/api/admin/v1/admins/1/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"disabled"}'   # 400 cannot disable primary admin
curl -s -X PUT localhost:8080/api/admin/v1/admins/2/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"disabled"}'   # 200
# testadmin 旧 token（若有）应 401；其 login 也应 401 invalid credentials
curl -s -X PUT localhost:8080/api/admin/v1/admins/2/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"active"}'    # 200 恢复
# 清理 testadmin（无 DELETE 端点）→ SQL: docker exec comic-pg psql -U comic -d comic -c "DELETE FROM admins WHERE username='testadmin'"（或保留披露）
```
提交：`git commit -m "feat(admin): admin account management (list/create/status/password) + status guard"`；NO push。

---

### Task 2: 前端 admin-accounts/index.vue

**Files:**
- Create: `admin/src/views/admin-accounts/index.vue`
- Modify: `admin/src/service/api/admin.ts`（fetchAdmins/createAdmin/setAdminStatus/resetAdminPassword）
- Modify: `admin/src/typings/api/admin.d.ts`（AdminAccount 类型）
- Modify: `admin/build/plugins/router.ts`（menuIcons admin-accounts）
- Modify: `admin/src/locales/langs/zh-cn.ts` + `en-us.ts`（route.admin-accounts）
- （elegant-router 产物一并提交）

**Interfaces:**
- Consumes: Task 1 全部 4 API
- Produces: 可用的管理员管理页

- [ ] **Step 1: api/admin.ts 加函数**（参照 fetchAdminUsers 模式）
```ts
fetchAdmins() → GET /v1/admins, `?? []`
createAdmin(payload: {username,password,role}) → POST /v1/admins
setAdminStatus(id, status: 'active'|'disabled') → PUT /v1/admins/:id/status
resetAdminPassword(id, password) → PUT /v1/admins/:id/password
```
- [ ] **Step 2: admin.d.ts** — AdminAccount {id,username,role,status,createdAt}
- [ ] **Step 3: admin-accounts/index.vue** — NDataTable（用户名/角色tag/状态tag/创建时间/操作）+ 新建 NModal（username/password/role NSelect 仅 admin/super）+ 重置密码 NModal（password，≥6）+ 禁用/启用 NDialog 确认（id===1 禁用按钮 disabled，tooltip 主管理员不可禁用）
- [ ] **Step 4: menuIcons + locales** — `admin-accounts: 'mdi:account-cog-outline'`；zh-cn `admin-accounts: '管理员管理'`；en-us `Admin Management`；elegant-router regenerate（dev server 跑短时）
- [ ] **Step 5: typecheck + 浏览器验证** — `pnpm typecheck` exit 0；:5174（admin/admin123 单会话）：菜单出现「管理员管理」；列表 [admin]；新建 testadmin → 列表 2 条；重置 testadmin 密码 → 提示旧会话失效；禁用 → 状态红；启用 → 恢复；清理（SQL 或披露）
- [ ] **Step 6: Commit** — `git add admin/src && git commit -m "feat(admin-web): admin account management page"`；NO push

---

### Task 3: 验证 + E2E + 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c 补《管理员管理》章节）
- Test-only

- [ ] **Step 1: 后端全量** — build/vet/test 全绿
- [ ] **Step 2: Playwright E2E**（:5174 单 admin 会话）：
  ① 管理端登录 → 管理员管理页 → 列表 [admin] → 新建 testadmin（密码 admin123）→ 2 条 → testadmin 登录（新会话）成功 → testadmin 被禁用 → 旧 token 401 → testadmin 登录 401 → 启用 → 登录恢复 → 重置密码（newpass123）→ 旧 token 失效 → 新密码登录成功 → SQL 清理 testadmin 或披露
  ② 截图留档
- [ ] **Step 3: agent-browser 视觉复查** — 管理员页截图：浅色专业风、表格/弹窗正常、中文
- [ ] **Step 4: AGENTS.md** — 加「管理员管理」小节：000015 迁移、4 API、种子保护、disabled token 即时失效、admin/super 角色预留、后续（super 区分/日志/2FA）
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: admin admin-management in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`；验证 unpushed=0

---

## 风险与决策记录

- **视图目录名 `admin` 与路由冲突**：统一用 `admin-accounts/index.vue`（route key admin-accounts，菜单「管理员管理」），避免 elegant-router 与既有 admin 语义冲突。全计划一致采用该命名。
- **password_hash 泄漏**：列表/详情 DTO（AdminAccount）**不含** passwordHash 字段——repo 内部 AdminRow 保留（登录用），handler 映射时剥离。
- **种子保护实现**：service.SetStatus 检查 `id==1`（硬编码主管理员 id；或查 username=='admin'——用 id==1 更精确，与种子迁移一致）。重置密码允许（生产改密必需）。若未来加 super 角色，种子保护由 role=super 取代。
- **disabled 与密码错同响应 401**：不泄露账号状态（防枚举）。
- **无 DELETE 端点**：管理员删除不做（YAGNI——禁用即可）；测试清理用 SQL。
- **token 轮换**：登录轮换 token（既有行为）；重置密码清 token 强制登出。
