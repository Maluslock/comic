# P2c 后台内容管理（作品 / 评论 / 标签）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理端内容风控：作品下架/恢复、违规评论删除、标签 CRUD。

**Architecture:** `works.status` 迁移（000017）+ C 端 `GetWorksByPhotographer`/`GetFeaturedWorks` 过滤下架 + 独立 AdminContentRepo（*pgxpool.Pool 模式）+ service（校验）+ handler 三层 + 8 管理 API；前端 content/index.vue（3 tabs：作品/评论/标签），复用现有 admin 模式。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue3.5 + Vite8 + NaiveUI（soybean-admin）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；迁移在 `server/migrations/`（**000017**）。
- `works.status VARCHAR(10) NOT NULL DEFAULT 'active'`；值 `active`/`down`；下架作品 C 端不可见。
- **C 端改动（2 处 query）**：`GetWorksByPhotographer` 加 `AND status='active'`；`GetFeaturedWorks` 加 `AND status='active'`（home 首页 featured works 也不含下架）。
- `reviews` 硬删除（无 status 列）；`tags` CRUD（无 status）+ `photographer_tags` 引用检查（有引用删除 → 400）。
- 管理 API 前缀 `/api/admin/v1`，全部 `AdminAuthRequired`；adminGroup.Use 之后注册。
- 独立 repo 模式：`NewAdminContentRepo(pool *pgxpool.Pool) *AdminContentRepo`；sentinels：`ErrTagExists`、`ErrTagInUse`、`ErrInvalidStatus`（复用 admin_user_service 的或新建——用新建 `ErrInvalidContentStatus` 避免跨 service 依赖）。
- 前端视图 `content/index.vue`（route key `content`，菜单「内容管理」）；menuIcons `content: 'mdi:book-open-page-variant-outline'`；locale zh-cn `content: '内容管理'` / en-us `Content`；elegant-router 产物提交。
- `pnpm typecheck` green gate；浅色主题；中文 UI；单根 `<div>`。
- 服务重启（server/，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- 管理员 admin/admin123；C 端测试：摄影师 10000000001（user 1001）、coser 13800138000（user 1）。
- 无新第三方依赖。所有 go build/vet/test + pnpm typecheck 必须过。
- 测试数据：现有 works=4（id 1-4）、reviews=3、tags=13、photographer_tags=12。test 用后清理（works 恢复 active、reviews 恢复、标签删除）或披露。

---

### Task 1: 后端内容管理 API（works.status + C 端过滤 + 8 API）

**Files:**
- Create: `server/migrations/000017_works_status.up.sql` / `.down.sql`
- Modify: `server/internal/repository/works.sql.go`（GetWorksByPhotographer/GetFeaturedWorks 加 status 过滤；Work struct 若无 Status 字段则加——检查 struct）
- Create: `server/internal/repository/admin_content_repo.go`
- Create: `server/internal/service/admin_content_service.go`
- Create: `server/internal/handler/admin_content_handler.go`
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/admin_content_service_test.go`

**Interfaces:**
- Consumes: `middleware.AdminAuthRequired`、`adminGroup`（Use 之后）
- Produces:
  - repo `AdminContentRepo`：
    - `ListWorks(ctx, photographerID *int64, status string, limit, offset int) ([]AdminWork, int64, error)` — AdminWork{ID,Title,Images []string,PhotographerName,Status,CreatedAt}（LEFT JOIN photographers；status 筛选；images[0] 缩略图由前端取第一条）
    - `SetWorkStatus(ctx, id int64, status string) error`（UPDATE works SET status=$2 WHERE id=$1；RowsAffected=0 → ErrContentNotFound）
    - `ListReviews(ctx, keyword string, photographerID *int64, limit, offset int) ([]AdminReview, int64, error)` — AdminReview{ID,UserName,UserAvatar,Rating,Content,PhotographerName,CreatedAt}
    - `DeleteReview(ctx, id int64) error`（DELETE；RowsAffected=0 → ErrContentNotFound）
    - `ListTags(ctx) ([]AdminTag, error)` — AdminTag{ID,Name,UsageCount}
    - `CreateTag(ctx, name string) (int64, error)`（INSERT RETURNING id；23505 → ErrTagExists）
    - `UpdateTag(ctx, id int64, name string) error`（UPDATE；RowsAffected=0 → ErrContentNotFound；23505 → ErrTagExists）
    - `DeleteTag(ctx, id int64) error`（先检查 photographer_tags 引用 count>0 → ErrTagInUse；无引用 DELETE）
  - `ErrContentNotFound`（repo sentinel）；`ErrTagExists`、`ErrTagInUse`、`ErrInvalidContentStatus`（service sentinels）
  - service `AdminContentService`：ListWorks/SetWorkStatus（校验 status ∈ active/down else ErrInvalidContentStatus）/ListReviews/DeleteReview/ListTags/CreateTag（name 空由 handler binding:"required" 处理 400——service 不设 ErrInvalidName）/UpdateTag/DeleteTag
  - handler `AdminContentHandler`：ListWorks (`?photographerId=&status=&page=&pageSize=`)、SetWorkStatus (`{status}`)、ListReviews (`?keyword=&photographerId=&page=&pageSize=`)、DeleteReview (path id)、ListTags、CreateTag (`{name}`)、UpdateTag (`{name}`)、DeleteTag
- 错误映射：200/201/400 (ErrInvalidContentStatus/ErrInvalidName/ErrTagInUse/非法 status)/404 (ErrContentNotFound)/409 (ErrTagExists)。

- [ ] **Step 1: 迁移**

`000017_works_status.up.sql`: `ALTER TABLE works ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';`
`down.sql`: `ALTER TABLE works DROP COLUMN IF EXISTS status;`
执行 up + 验证 status 列存在。

- [ ] **Step 2: C 端作品过滤**（works.sql.go）
- `getFeaturedWorks` SQL 加 `AND status = 'active'`；`getWorksByPhotographer` SQL 加 `AND status = 'active'`。
- 检查 `Work` struct 是否含 Status（featured 不含也行——只需过滤，不返回）。确认构建通过。

- [ ] **Step 3: AdminContentRepo**（admin_content_repo.go，*pgxpool.Pool 模式）
- 按 Interfaces 方法实现；images TEXT[] 用 pgx `[]string`；tags 引用检查 `SELECT count(*) FROM photographer_tags WHERE tag_id=$1`。

- [ ] **Step 4: AdminContentService**（admin_content_service.go）+ sentinels

- [ ] **Step 5: AdminContentHandler**（admin_content_handler.go，参照 admin_user_handler 的 query 解析）

- [ ] **Step 6: 注册路由**（main.go adminGroup Use 之后）
```go
contentRepo := repository.NewAdminContentRepo(pool)
contentSvc := service.NewAdminContentService(contentRepo)
contentH := handler.NewAdminContentHandler(contentSvc)
adminGroup.GET("/works", contentH.ListWorks)
adminGroup.PUT("/works/:id/status", contentH.SetWorkStatus)
adminGroup.GET("/reviews", contentH.ListReviews)
adminGroup.DELETE("/reviews/:id", contentH.DeleteReview)
adminGroup.GET("/tags", contentH.ListTags)
adminGroup.POST("/tags", contentH.CreateTag)
adminGroup.PUT("/tags/:id", contentH.UpdateTag)
adminGroup.DELETE("/tags/:id", contentH.DeleteTag)
```

- [ ] **Step 7: 测试**（fake store 风格）
- TestSetWorkStatus_Invalid：status="banana" → ErrInvalidContentStatus
- TestSetWorkStatus_NotFound：repo 0 rows → ErrContentNotFound
- TestDeleteReview_NotFound：同上
- TestCreateTag_Duplicate：23505 → ErrTagExists
- TestDeleteTag_InUse：引用 count>0 → ErrTagInUse
- TestDeleteTag_NoRefs：无引用 → repo.DeleteTag 成功
- TestListWorks_Filters：status/摄影师筛选参数传递正确

- [ ] **Step 8: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/... -count=1
# 重启（Global Constraints 配方）
TOKEN=$(curl -s -X POST localhost:8080/api/admin/v1/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s "localhost:8080/api/admin/v1/works?page=1&pageSize=20" -H "Authorization: Bearer $TOKEN"          # 200, 4 works
curl -s -X PUT localhost:8080/api/admin/v1/works/1/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"down"}'   # 200
curl -s localhost:8080/api/v1/photographers/1 -H "Authorization: Bearer $tokenC" | python3 -c "import sys,json; [print(w['title']) for w in json.load(sys.stdin)['works']]"   # 不含 id=1 的作品（下架不可见）
curl -s -X PUT localhost:8080/api/admin/v1/works/1/status -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"active"}'   # 200 恢复
curl -s "localhost:8080/api/admin/v1/reviews?page=1&pageSize=20" -H "Authorization: Bearer $TOKEN"          # 200, 3 reviews
curl -s "localhost:8080/api/admin/v1/tags" -H "Authorization: Bearer $TOKEN"                                 # 200, 13 tags
curl -s -X POST localhost:8080/api/admin/v1/tags -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"测试标签"}'   # 201
curl -s -X PUT localhost:8080/api/admin/v1/tags/:id -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"测试标签改"}'   # 200
curl -s -X DELETE localhost:8080/api/admin/v1/tags/:id -H "Authorization: Bearer $TOKEN"                      # 200（无引用）
# 清理测试标签（已删则无需）；恢复 works 原状态
```
提交：`git commit -m "feat(admin): content management (works status/down + reviews delete + tags CRUD) + C-end down filter"`；NO push。

---

### Task 2: 前端 content/index.vue（3 tabs）

**Files:**
- Create: `admin/views/content/index.vue`
- Modify: `admin/src/service/api/admin.ts`（8 函数）
- Modify: `admin/src/typings/api/admin.d.ts`（AdminWork/AdminReview/AdminTag 类型）
- Modify: `admin/build/plugins/router.ts`（menuIcons content）
- Modify: `admin/src/locales/langs/zh-cn.ts` + `en-us.ts`（route.content）
- （elegant-router 产物提交）

**Interfaces:**
- Consumes: Task 1 全部 8 API
- Produces: 可用的内容管理页

- [ ] **Step 1: api/admin.ts 加 8 函数**（fetchAdminWorks/setWorkStatus/fetchReviews/deleteReview/fetchTags/createTag/updateTag/deleteTag；列表 `?? {list:[],total:0}` / `?? []` tag）
- [ ] **Step 2: admin.d.ts** — AdminWork{id,title,images,photographerName,status,createdAt} / AdminReview{id,userName,userAvatar,rating,content,photographerName,createdAt} / AdminTag{id,name,usageCount}
- [ ] **Step 3: content/index.vue** — NTabs 作品/评论/标签：
  - 作品 tab：NDataTable（NImage 100×70 缩略图取 images[0]、标题、摄影师、状态 tag 上架=success 绿/下架=error 红、时间、操作下架|恢复按钮 NDialog 确认）+ 状态筛选 NSelect + remote 分页
  - 评论 tab：NDataTable（NAvatar、用户名、摄影师、评分 NTag、内容 ellipsis、时间、操作删除 NDialog 确认）+ keyword 搜索 + remote 分页
  - 标签 tab：NDataTable（名称、使用量、操作编辑/删除）+ 新增标签 NButton → NModal（name 必填）；编辑 NModal；删除 NDialog 确认
- [ ] **Step 4: menuIcons + locales** — `content: 'mdi:book-open-page-variant-outline'`；zh-cn `content: '内容管理'`；en-us `Content`；elegant-router regenerate
- [ ] **Step 5: typecheck + 浏览器验证** — `pnpm typecheck` exit 0；:5174（admin/admin123 单会话）：菜单「内容管理」；作品 tab 4 行 + 下架/恢复；评论 tab 3 行 + 删除（用后恢复或披露）；标签 tab 13 行 + 新增/编辑/删除（测试标签）；清理（恢复 works 状态、测试标签删除）或披露
- [ ] **Step 6: Commit** — `git commit -m "feat(admin-web): content management page (works/reviews/tags)"`；NO push

---

### Task 3: 验证 + E2E（双端）+ 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c 补《内容管理》章节）
- Test-only

- [ ] **Step 1: 后端全量** — build/vet/test 全绿
- [ ] **Step 2: Playwright E2E（双端）**：
  ① admin :5174 → 内容管理页 → 作品 tab 下架一个（work 1）→ C 端摄影师详情页该作品不可见（:5173 摄影师 10000000001→详情）→ 恢复
  ② 评论 tab 删除一条（reviews 3 条选 1）→ C 端摄影师详情评论数减少 → 恢复（如无恢复手段则披露删除）
  ③ 标签 tab 新增/重命名/删除（测试标签）→ 标签列表变化
- [ ] **Step 3: agent-browser 视觉复查** — content 页截图（3 tabs 浅色专业风、表格/弹窗/中文）
- [ ] **Step 4: AGENTS.md** — 加「内容管理」小节：000017、8 API、C 端 works 过滤、reviews 硬删、tags CRUD+引用检查、后续（审核流/软删/合并/编辑）
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: admin content management in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`；验证 unpushed=0

---

## 风险与决策记录

- **works.status DROP NOT NULL 无**：status 用默认 active + NOT NULL（新作品自动上架，符合"先发后审"现状）。
- **C 端过滤强度**：GetWorksByPhotographer（摄影师详情页 works）+ GetFeaturedWorks（首页 featured）两处——确保下架作品全链路不可见。
- **评论硬删不可逆**：删除是永久行为（无软删/恢复），YAGNI 决定；UI 需 NDialog 二次确认。
- **标签删除引用检查**：photographer_tags count>0 → 400（保护关联数据）；测试标签用全新名（无引用）可行删。
- **images[0] 缩略图**：前端取数组第一条；works 无 cover 列（orig = images TEXT[]）。
- **测试数据清理**：work 下架恢复 active、评论删除披露（无恢复手段）、测试标签删除——Task 3 明确恢复。
