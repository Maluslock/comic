# 摄影师作品管理 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 摄影师自助上传/删除作品，作品集（详情页）实时展示。

**Architecture:** 后端 PhotographerService 加 CreateWork/MyWorks/DeleteWork（校验摄影师身份与归属）+ repo InsertWork/DeleteWorkByID + 3 端点（AuthRequired）；C 端 works.vue（列表+新增表单+删除确认，暗色霓虹）+ activate 入口 + api 函数 + pages.json。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue3 + uni-app（暗色 $dark-*/$neon-*）。

## Global Constraints

- 0 迁移（works.status 已有，P2c 内容管理加过；INSERT status 默认 active）。
- 3 端点 AuthRequired：POST `/v1/photographers/works`（201/403）、GET `/v1/photographers/works/mine`（200 数组）、DELETE `/v1/photographers/works/:id`（200/403/404）。
- service 归属校验：CreateWork/DeleteWork 用 GetPhotographerByUserID 确认身份，works.photographer_id 与当前摄影师 id 一致（不一致 403）。
- C 端暗色霓虹 + rpx + @/ 别名 + <script setup>；图片 URL 输入（非上传）。
- 服务重启（server/，NO pkill）；`npx vue-tsc --noEmit`（根项目）必须过；无新依赖。

---

### Task 1: 后端作品端点

**Files:**
- Modify: `server/internal/repository/works.sql.go`（InsertWork/DeleteWorkByID）
- Modify: `server/internal/service/photographer_service.go`（CreateWork/MyWorks/DeleteWork）
- Create: `server/internal/handler/photographer_work_handler.go`（3 handlers）
- Modify: `server/cmd/api/main.go`（3 路由）
- Test: `server/internal/service/photographer_service_test.go`（加 3 测试）或新测试文件

**Interfaces:**
- Produces:
  - repo：`InsertWork(ctx, photographerID int64, title string, images []string, description string) (int64, error)`（INSERT RETURNING id；status 默认 active）；`DeleteWorkByID(ctx, id int64) error`（DELETE；RowsAffected=0 → ErrWorksNotFound 或复用，新建 ErrWorkNotFound）
  - service：`CreateWork(ctx, userID int64, title string, images []string, description string) (int64, error)`（非摄影师 → ErrNotPhotographer 复用）；`MyWorks(ctx, userID int64) ([]Work, error)`（GetWorksByPhotographer）；`DeleteWork(ctx, userID, workID int64) error`（先 GetPhotographerByUserID；GetWorkByID 校验归属（新 repo 方法或 SELECT photographer_id FROM works WHERE id=$1）→ 非本人 ErrWorkForbidden；DELETE）
  - handler：`CreateWork(c)`/`MyWorks(c)`/`DeleteWork(c)`（AuthRequired 中间件设置 user_id）
- 错误映射：403 ErrNotPhotographer/ErrWorkForbidden；404 ErrWorkNotFound。

- [ ] **Step 1: repo 加方法**（works.sql.go）：InsertWork/DeleteWorkByID/GetWorkPhotographerID（SELECT photographer_id FROM works WHERE id=$1）
- [ ] **Step 2: service 加方法**（photographer_service.go）：CreateWork/MyWorks/DeleteWork + sentinels ErrWorkNotFound/ErrWorkForbidden
- [ ] **Step 3: handler**（photographer_work_handler.go）：CreateWork bind {title required, images required(≥1), description}；MyWorks；DeleteWork path id
- [ ] **Step 4: 注册路由**（main.go）：3 条 AuthRequired
- [ ] **Step 5: 测试**：TestCreateWork_NotPhotographer/TestCreateWork_OK/TestDeleteWork_NotOwner（403）/TestDeleteWork_NotFound（404）
- [ ] **Step 6: 构建+冒烟+提交**：restart；C 端摄影师 10000000001（user 1001, photographer 1）登录 → POST works {"title":"测试作品","images":["https://picsum.photos/600/450"]} → 201；GET mine → 含该作品；DELETE → 200；GET mine → 无；coser 13000000001 → POST 403。commit "feat(c-end): photographer works API (create/mine/delete) + ownership guard"

---

### Task 2: C 端 works 页 + activate 入口 + 验证

**Files:**
- Create: `src/pages/photographer/works.vue`
- Modify: `src/pages/photographer/activate.vue`（作品管理入口）
- Modify: `src/api/index.ts`（uploadWork/getMyWorks/deleteWork）
- Modify: `src/pages.json`（注册 works）

**Interfaces:**
- Consumes: Task 1 全部 3 API
- Produces: 可用的作品管理页（暗色）

- [ ] **Step 1: api/index.ts 加 3 函数**：`uploadWork(payload {title, images, description?})` → POST；`getMyWorks()` → GET mine；`deleteWork(id)` → DELETE
- [ ] **Step 2: pages.json 注册** `pages/photographer/works`（navigationBarTitleText 作品管理）
- [ ] **Step 3: activate.vue 入口**：isPhotographer 块加「作品管理」按钮（参照去接单管理样式）→ navigateTo works
- [ ] **Step 4: works.vue**（暗色霓虹）：onShow getMyWorks → 列表卡片（缩略图 images[0]/标题/状态 tag active=上架/down=下架灰/删除按钮 NDialog 确认）；新增表单（标题 NInput/图片 URL 动态数组（初始 1 个，+添加 max 5，可删）/说明 textarea → 提交 → toast → reload）
- [ ] **Step 5: vue-tsc + 浏览器验证**：300x5173 摄影师 10000000001 登录 → 激活页作品管理 → works 页 → 上传（1 图）→ 列表出现 → 摄影师详情页作品集含 → 删除 → 消失；commit "feat(c-end): photographer works management page + entry"

---

### Task 3: 验证 + 视觉 + AGENTS.md + 推送

- [ ] **Step 1: 后端全量** build/vet/test
- [ ] **Step 2: E2E** 全链（上传→详情页可见→删除）
- [ ] **Step 3: agent-browser 视觉复查** works 页暗色
- [ ] **Step 4: AGENTS.md** 加「摄影师作品管理」小节
- [ ] **Step 5: commit + push**（git -c http.proxy= -c https.proxy= push gitee master）

---

## 风险与决策记录

- 图片 URL 输入（非上传）：与 Banner/Cert 一致，YAGNI 真上传后续做。
- 归属校验：DeleteWork 必须校验 works.photographer_id == 当前摄影师 id（防删别人作品——安全关键）。
- 作品状态：上传默认 active（先发后审现状），管理端内容管理可下架。
- 测试数据：测试作品上传后删除，DB 恢复。
