# P2c 后台轮播图管理（Banner）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理端轮播图管理：列表/新增/编辑/上下线，管理端改动即时反映到 C 端首页轮播。

**Architecture:** 后端沿用既有 admin 三层（独立 repo + service + handler + main.go 路由，复用 AdminAuthRequired 与 adminGroup）；`banners` 表已存在且 C 端 GetActiveBanners 已消费（0 迁移、0 C 端改动）；前端 admin/ 新增 banner/index.vue（缩略图表格 + 新增/编辑抽屉 + 上下线开关），复用现有 admin 视图模式。

**Tech Stack:** Go 1.22 + Gin + pgx（管理端独立 repo 模式）；Vue3.5 + Vite8 + NaiveUI + UnoCSS（soybean-admin）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；**0 迁移**（banners 表已存在）。
- 管理 API 前缀 `/api/admin/v1`，全部 `AdminAuthRequired`（adminGroup.Use 之后注册）。
- banners 表列：id/image_url/title/link_type/link_id/sort_order/is_active/created_at/updated_at。
- `link_type` 仅支持 `event`（link_id 指向 comic_events.id）；其他值 → 400。
- 请求体校验：POST/PUT 的 title/imageUrl/linkType 必填（binding:"required"）；linkId/sortOrder 非必填数字（缺省 0）。
- 响应 JSON：camelCase（id/imageUrl/title/linkType/linkId/sortOrder/isActive/createdAt）。
- GET /banners 按 sort_order ASC 排序（现有 3 条：ChinaJoy 2026/sort 0、萤火虫/sort 1、CP33/sort 2）。
- 独立 repo 模式（参照 server/internal/repository/admin_user_repo.go：NewAdminBannerRepo(pool *pgxpool.Pool) *AdminBannerRepo；ErrManageNotFound 复用或新建 ErrBannerNotFound）。
- 前端：admin/ 内新增视图（elegant-router 自动生成路由）、menuIcons 加 banner 键、locale zh-cn/en-us 加 banner 路由标题；`pnpm typecheck` 是 green gate；浅色主题；中文 UI；单根 `<div>` 模板约束。
- 服务重启（server/ 目录，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- 管理员账号：admin/admin123。Banner 测试链接 linkId=1（ChinaJoy 2026 event id=1）。
- 所有 go build/vet/test + pnpm typecheck 必须过。无新第三方依赖。

---

### Task 1: 后端 banners 管理 API

**Files:**
- Create: `server/internal/repository/admin_banner_repo.go`
- Create: `server/internal/service/admin_banner_service.go`
- Create: `server/internal/handler/admin_banner_handler.go`
- Modify: `server/cmd/api/main.go`（adminGroup 注册 4 路由）
- Test: `server/internal/service/admin_banner_service_test.go`

**Interfaces:**
- Consumes: `middleware.AdminAuthRequired`（已有）；`repository.NewAdminRepo` 模式；`adminGroup`（main.go Use 之后）
- Produces:
  - `repository.NewAdminBannerRepo(pool *pgxpool.Pool) *AdminBannerRepo`
  - `AdminBanner` struct：`{ID int64, ImageURL string, Title string, LinkType string, LinkID int32, SortOrder int32, IsActive bool, CreatedAt string}`
  - `AdminBannerRepo` 方法：
    - `List(ctx) ([]AdminBanner, error)` — `SELECT id, image_url, title, link_type, COALESCE(link_id,0), COALESCE(sort_order,0), is_active, created_at::text FROM banners ORDER BY sort_order ASC`
    - `Create(ctx, imageURL, title, linkType string, linkID, sortOrder int32) (int64, error)` — `INSERT INTO banners (image_url, title, link_type, link_id, sort_order) VALUES ($1,$2,$3,$4,$5) RETURNING id`
    - `GetByID(ctx, id int64) (*AdminBanner, error)` — 单行，pgx.ErrNoRows → ErrBannerNotFound
    - `Update(ctx, id int64, imageURL, title, linkType string, linkID, sortOrder int32, isActive bool) error` — `UPDATE banners SET image_url=$2,title=$3,link_type=$4,link_id=$5,sort_order=$6,is_active=$7,updated_at=NOW() WHERE id=$1`；RowsAffected=0 → ErrBannerNotFound
    - `SetStatus(ctx, id int64, isActive bool) error` — `UPDATE banners SET is_active=$2,updated_at=NOW() WHERE id=$1`；RowsAffected=0 → ErrBannerNotFound
  - `service.NewAdminBannerService(repo *AdminBannerRepo) *AdminBannerService`：
    - `List(ctx) ([]AdminBanner, error)`
    - `Create(ctx, req BannerUpsert) (int64, error)` — 校验 LinkType=="event" 否则 ErrInvalidLinkType
    - `Update(ctx, id int64, req BannerUpsert) error`
    - `SetStatus(ctx, id int64, isActive bool) error`
  - `BannerUpsert`：`{ImageURL string, Title string, LinkType string, LinkID int32, SortOrder int32}`
  - sentinels：`ErrInvalidLinkType`、`ErrBannerNotFound`（或复用 repository 的）
  - `handler.NewAdminBannerHandler(svc *AdminBannerService) *AdminBannerHandler`：`List(c)`、`Create(c)`、`Update(c)`、`SetStatus(c)`
- 错误映射：404 ErrBannerNotFound；400 ErrInvalidLinkType/binding required。

- [ ] **Step 1: AdminBannerRepo**

`server/internal/repository/admin_banner_repo.go`（参照 admin_user_repo.go 模式）：
```go
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBannerNotFound = errors.New("banner not found")

type AdminBanner struct {
	ID        int64  `json:"id"`
	ImageURL  string `json:"imageUrl"`
	Title     string `json:"title"`
	LinkType  string `json:"linkType"`
	LinkID    int32  `json:"linkId"`
	SortOrder int32  `json:"sortOrder"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type AdminBannerRepo struct{ pool *pgxpool.Pool }

func NewAdminBannerRepo(pool *pgxpool.Pool) *AdminBannerRepo { return &AdminBannerRepo{pool: pool} }
// List/Create/GetByID/Update/SetStatus 按 Interfaces 签名实现（SQL 见上）
```

- [ ] **Step 2: AdminBannerService**

`server/internal/service/admin_banner_service.go`：
```go
package service

var ErrInvalidLinkType = errors.New("invalid link type")

type BannerUpsert struct {
	ImageURL  string
	Title     string
	LinkType  string
	LinkID    int32
	SortOrder int32
}

type AdminBannerService struct{ repo *repository.AdminBannerRepo }
// List/Create/Update/SetStatus 按 Interfaces 签名；Create/Update 校验 LinkType=="event"
```

- [ ] **Step 3: AdminBannerHandler**

`server/internal/handler/admin_banner_handler.go`（参照 admin_user_handler.go）：
- List `c.JSON(200, banners)`
- Create bind `BannerUpsert` 带 json tags（imageUrl/title/linkType/linkId/sortOrder，title/imageUrl/linkType `binding:"required"`）→ 201 `{"id":...}` | 400（invalid link type 或 binding）
- Update path id + body → 200 `{"ok":true}` | 404 | 400
- SetStatus `{isActive:bool}` → 200 `{"ok":true}` | 404

- [ ] **Step 4: 注册路由**

main.go adminGroup（现有 220 行后加）：
```go
adminBannerRepo := repository.NewAdminBannerRepo(pool)
adminBannerSvc := service.NewAdminBannerService(adminBannerRepo)
adminBannerH := handler.NewAdminBannerHandler(adminBannerSvc)
adminGroup.GET("/banners", adminBannerH.List)
adminGroup.POST("/banners", adminBannerH.Create)
adminGroup.PUT("/banners/:id", adminBannerH.Update)
adminGroup.PUT("/banners/:id/status", adminBannerH.SetStatus)
```

- [ ] **Step 5: 测试**

`admin_banner_service_test.go`（fake repo 风格，参照 cert_application_service_test.go）：
- TestCreate_ValidLinkType：Create("", imageURL,"title","event",1,0) → nil（repo 收到正确参数）
- TestCreate_InvalidLinkType：Create("",...,"page",...) → ErrInvalidLinkType（repo.Create 不被调用）
- TestUpdate_NotFound：repo.Update RowsAffected=0 → ErrBannerNotFound
- TestSetStatus_NotFound：同上

- [ ] **Step 6: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/service/... -run Banner -count=1
# 重启（Global Constraints 配方）
TOKEN=$(curl -s -X POST localhost:8080/api/admin/v1/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s localhost:8080/api/admin/v1/banners -H "Authorization: Bearer $TOKEN"                     # 200, 3 条
curl -s -X POST localhost:8080/api/admin/v1/banners -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"imageUrl":"https://picsum.photos/seed/bannertest/750/360","title":"测试轮播","linkType":"event","linkId":1,"sortOrder":9}'   # 201
curl -s -X PUT localhost:8080/api/admin/v1/banners/4 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"imageUrl":"https://picsum.photos/seed/bannertest/750/360","title":"测试轮播改","linkType":"event","linkId":1,"sortOrder":9,"isActive":true}'   # 200
curl -s -X PUT localhost:8080/api/admin/v1/banners/4/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"isActive":false}'   # 200
curl -s localhost:8080/api/v1/home | python3 -c "import sys,json; [print(b['title'],b['isActive'] if 'isActive' in b else '') for b in json.load(sys.stdin).get('banners',[])]"   # 测试轮播（isActive=false）应不在列表
curl -s -X PUT localhost:8080/api/admin/v1/banners/4/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"isActive":true}'   # 200 恢复
# 清理测试数据（保留状态下）——可选择保留为演示或删除：无 DELETE 端点，用 SQL 删或保留披露
```
提交：`git commit -m "feat(admin): banner management API (list/create/update/status)"`；NO push。

---

### Task 2: 前端 banner/index.vue

**Files:**
- Create: `admin/src/views/banner/index.vue`
- Modify: `admin/src/service/api/admin.ts`（fetchBanners/createBanner/updateBanner/setBannerStatus）
- Modify: `admin/src/typings/api/admin.d.ts`（AdminBanner 类型）
- Modify: `admin/build/plugins/router.ts`（menuIcons 加 banner: 'mdi:image-multiple-outline'）
- Modify: `admin/src/locales/langs/zh-cn.ts` + `en-us.ts`（route.banner）
- （elegant-router 自动生成路由 + 类型，若有自动产物一并提交）

**Interfaces:**
- Consumes: 全部 Task 1 API（GET /banners、POST /banners、PUT /banners/:id、PUT /banners/:id/status）
- Produces: 可用的轮播图管理页

- [ ] **Step 1: api/admin.ts 加函数**

```ts
export async function fetchBanners() { return adminRequest.get<Api.Admin.AdminBanner[]>('/v1/banners').then(r => r.data ?? []); }
export async function createBanner(payload: Partial<Api.Admin.AdminBanner>) { return adminRequest.post('/v1/banners', payload); }
export async function updateBanner(id: number, payload: Partial<Api.Admin.AdminBanner>) { return adminRequest.put(`/v1/banners/${id}`, payload); }
export async function setBannerStatus(id: number, isActive: boolean) { return adminRequest.put(`/v1/banners/${id}/status`, { isActive }); }
```
（按项目实际 request helper 命名/模式适配——参照 fetchAdminUsers）

- [ ] **Step 2: admin.d.ts 加类型**

```ts
namespace Api.Admin {
  interface AdminBanner {
    id: number;
    imageUrl: string;
    title: string;
    linkType: string;
    linkId: number;
    sortOrder: number;
    isActive: boolean;
    createdAt: string;
  }
}
```

- [ ] **Step 3: banner/index.vue**

顶部「轮播图管理」+「新增 Banner」NButton；NDataTable（无分页，全量）：缩略图 NImage 120×70（width=120 height=70, fallbackSrc）；标题；linkType tag（event →「活动」）；linkId；sortOrder；NSwitch isActive（`onUpdateValue` → setBannerStatus + reload）；操作列「编辑」。新增/编辑 NDrawer 表单：imageUrl NInput（placeholder 图片 URL）、title NInput（必填）、linkType NSelect（仅「活动」=event）、linkId NInputNumber（min 0）、sortOrder NInputNumber（min 0）、isActive NSwitch（仅编辑显示）。保存 → create/update → 成功 message + 关闭 + reload。单根 `<div>` 包裹；浅色。

- [ ] **Step 4: 菜单/路由/类型/差异**

menuIcons 加 `banner: 'mdi:image-multiple-outline'`（若 key 不存在时 guard 已保护）；zh-cn.ts route 加 `banner: '轮播图管理'`；en-us.ts 加 `banner: 'Banner Management'`。elegant-router 自动生成（dev server 起时 regenerate 或跑 `pnpm dev` 短时）——生成物提交。vue-tsc 需过（locale RouteKey 需同步两文件，否则 Schema 报错）。

- [ ] **Step 5: typecheck + 浏览器验证**

```bash
cd /vol1/1000/code/comic/admin && pnpm typecheck  # exit 0
```
浏览器 :5174（admin/admin123，单会话）：菜单出现「轮播图管理」；列表显示 3 条 + 缩略图；新增一条（测试标题，linkId=1）→ 列表 4 条；编辑标题 → 生效；开关下线 → 状态变；（用 API 或 SQL 清理测试行或保留披露）；Vite proxy 已通（相对路径 /api/admin/v1）。

- [ ] **Step 6: Commit**

`git add admin/src && git commit -m "feat(admin-web): banner management page"`；NO push。

---

### Task 3: 验证 + E2E + 视觉 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c 补《轮播图管理》章节）
- Test-only

- [ ] **Step 1: 后端全量** — `go build ./... && go vet ./... && go test ./...` 全绿
- [ ] **Step 2: Playwright E2E**（:5174 单 admin 会话 + C 端 :5173 验证首页）：
  ① 管理端登录 → 轮播图页列表 3 条 → 新增「测试轮播 linkId=1」→ 4 条 → 编辑标题 → 生效 → 下线 → C 端 home API 不含它 → 上线 → 恢复
  ② 清理测试行（SQL DELETE 或保留披露）
  ③ 截图留档
- [ ] **Step 3: agent-browser 视觉复查** — banner 页截图：浅色专业风、缩略图渲染、表格/NSwitch 正常、中文无乱码
- [ ] **Step 4: AGENTS.md** — 加《P2c 轮播图管理》小节：banners 表（0 迁移）、4 API、前端页、0 C 端改动说明、后续（上传/排期/link_type 扩展）
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: admin banner management in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`；验证 unpushed=0

---

## 风险与决策记录

- **无幂等冲突**：banners 表 0 迁移、C 端零改动——最大风险点（C 端回归）不存在。
- **link_type 单一值 event**：保持简单；扩展（页面/外链）明确延后。
- **GET /banners 无分页**：banners 少量（3-10 条），全量列表；如未来增长再加分页（YAGNI）。
- **无 DELETE 端点**：下线（isActive=false）即运营停用；若需硬删走 SQL（测试清理用）——计划中明示。
- **测试数据清理**：Task 1 冒烟/Task 2 浏览器测试会新增测试行——若保留披露；若清理用 `DELETE FROM banners WHERE title='测试轮播...'`（SQL）。
