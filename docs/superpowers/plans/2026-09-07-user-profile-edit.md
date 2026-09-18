# 用户资料编辑 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 用户编辑昵称/头像（个人中心入口 + 编辑页）。

**Architecture:** 后端 PUT /v1/me/profile（AuthRequired）+ UserRepo.UpdateProfile + main.go 内联 handler；C 端 profile/edit.vue（暗色表单）+ profile 页编辑入口 + api 函数 + userStore 更新。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue3 + uni-app（暗色）。

## Global Constraints

- 0 迁移（users 表有 name/avatar）。
- PUT `/v1/me/profile`（AuthRequired）`{name required, avatar optional}` → 200 {ok}|400(name 空)。
- C 端暗色霓虹 + rpx；DiceBear URL 提示。
- 服务重启（server/，NO pkill）；vue-tsc（根）必须过；无新依赖。

---

### Task 1: 后端 PUT /v1/me/profile

**Files:**
- Modify: `server/internal/repository/users.sql.go`（UpdateProfile）
- Modify: `server/cmd/api/main.go`（PUT 路由 + 内联 handler）
- Test: `server/internal/repository/users_test.go` 或 service tests（如存在用户 service 测试——无则 repo 级 skip，用冒烟覆盖）

**Interfaces:**
- Produces: `UserRepo.UpdateProfile(ctx, id int64, name, avatar string) error`（UPDATE users SET name=$2, avatar=$3 WHERE id=$1；RowsAffected=0 → 返回 pgx.ErrNoRows 或自定义——AuthRequired 保证用户存在）
- main.go 路由：`router.PUT("/api/v1/me/profile", middleware.AuthRequired(userRepo), func(c){ bind {name required, avatar optional}; userRepo.UpdateProfile(UserID, name, avatar) → 200 {ok:true} | 400 })`

- [ ] **Step 1**: repo UpdateProfile
- [ ] **Step 2**: main.go PUT 路由 + 内联 handler（name binding:"required"；avatar 可选）
- [ ] **Step 3**: build/vet/test + 冒烟：登录 13800138000 → PUT {"name":"米拉小测试","avatar":"https://api.dicebear.com/7.x/avataaars/svg?seed=test"} → 200；GET /v1/me → name/avatar 更新；PUT {"name":""} → 400；**恢复** 原 name/avatar（先 GET 记录原值）。
- [ ] **Step 4**: commit "feat(c-end): user profile update API"

---

### Task 2: C 端 profile 编辑入口 + edit 页 + 验证

**Files:**
- Create: `src/pages/profile/edit.vue`
- Modify: `src/pages/profile/index.vue`（编辑入口）
- Modify: `src/api/index.ts`（updateUserProfile）
- Modify: `src/pages.json`（注册 profile/edit）

**Interfaces:**
- Consumes: PUT /v1/me/profile + GET /v1/me
- Produces: 可用的资料编辑页（暗色）

- [ ] **Step 1**: api：updateUserProfile({name, avatar}) → PUT /v1/me/profile
- [ ] **Step 2**: pages.json 注册 pages/profile/edit（navigationBarTitleText 编辑资料）
- [ ] **Step 3**: profile/index.vue 头像/昵称区加「编辑」→ navigateTo('/pages/profile/edit')
- [ ] **Step 4**: edit.vue（暗色）：onShow 拉 userStore.user 或 /v1/me 填充（昵称 input/头像 URL input placeholder DiceBear 提示）+ 保存 → updateUserProfile → userStore.user.name/avatar 更新（或重新 getMe）→ toast → 返回
- [ ] **Step 5**: vue-tsc + 浏览器：登录 13800138000 → 个人中心 → 编辑 → 改昵称 → 保存 → 个人中心显示新昵称；恢复原昵称
- [ ] **Step 6**: commit "feat(c-end): user profile edit page + entry"

---

### Task 3: 验证 + 视觉 + AGENTS.md + 推送

- [ ] **Step 1**: 后端全量 build/vet/test
- [ ] **Step 2**: E2E（编辑→个人中心联动）
- [ ] **Step 3**: agent-browser 暗色视觉
- [ ] **Step 4**: AGENTS.md 加「用户资料编辑」小节
- [ ] **Step 5**: commit + push（git -c http.proxy= -c https.proxy= push gitee master）

---

## 风险与决策记录

- 头像 URL 输入（DiceBear 提示；YAGNI 真上传）。
- name 必填（空 → 400）；avatar 可选（空则保留？——**决定：avatar 可选，空则设为默认 DiceBear seed=phone**——不，保留原值更简单：handler 里 avatar 为空字符串时用原 avatar——从 GetByID 读原值；或 UPDATE 用 COALESCE——**简化：avatar 可选，空时 UpdateProfile 传原 avatar（handler 先 GetByID 读原值）**）。
- 测试恢复：昵称/头像恢复原值。
