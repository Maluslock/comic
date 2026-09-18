# 摄影师主页管理 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 摄影师编辑主页资料（昵称/简介/城市/接单模式/互勉说明）+ 补全 GetByUser 缺失字段。

**Architecture:** 后端 2 端点（PUT/GET profile，AuthRequired）+ repo UpdatePhotographerProfile + GetByUser 补全 mode/mutualIntro/certified；C 端 profile-edit.vue 表单 + activate 入口 + api 函数 + pages.json。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue3 + uni-app（暗色）。

## Global Constraints

- 0 迁移（photographers 表字段齐全）。
- PUT `/v1/photographers/profile`（AuthRequired）`{name,description,location,mode,mutualIntro}` 全量提交 → 200 {ok}|403；GET `/v1/photographers/profile/mine`（AuthRequired）→ 200 完整资料 | 403。
- mode ∈ {free,pay,both}（参考 activate 校验）；mutualIntro 可空。
- GetByUser 补全：返回加 mode/mutualIntro/certified（P2a 遗留修复）。
- C 端暗色霓虹；photo load-onShow 填充表单；保存全量提交。
- 服务重启（server/，NO pkill）；vue-tsc（根）必须过；无新依赖。

---

### Task 1: 后端 profile 端点

**Files:**
- Modify: `server/internal/repository/photographers.sql.go`（UpdatePhotographerProfile；确认 PhotographerWithTags 含 mode/mutual_intro/certified——若缺则补 SELECT）
- Modify: `server/internal/service/photographer_service.go`（UpdateProfile/MyProfile；GetByUser 补字段）
- Create: `server/internal/handler/photographer_profile_handler.go`
- Modify: `server/cmd/api/main.go`（2 路由）
- Test: `server/internal/service/photographer_service_test.go`（加 3 测试）

**Interfaces:**
- Produces:
  - repo：`UpdatePhotographerProfile(ctx, id int64, name, description, location string, mode string, mutualIntro *string) error`（UPDATE photographers SET name=$2, description=$3, location=$4, mode=$5, mutual_intro=$6, updated_at=NOW() WHERE id=$1；RowsAffected=0 → ErrPhotographerNotFound 或复用——新建 ErrProfileNotFound）
  - service：`UpdateProfile(ctx, userID int64, req ProfileUpdate) error`（GetPhotographerByUserID 校验身份 → 非摄影师 ErrNotPhotographer；mode 校验 ∈ {free,pay,both} else ErrInvalidMode）；`MyProfile(ctx, userID int64) (*PhotographerItem, error)`（补 mode/mutualIntro/certified——若 PhotographerItem 无这些字段加；GetByUser 同修）
  - `ProfileUpdate{Name, Description, Location, Mode string, MutualIntro *string}`
  - handler：`UpdateProfile(c)`/`MyProfile(c)`
- 错误映射：403 ErrNotPhotographer；400 ErrInvalidMode/绑定。

- [ ] **Step 1: repo**：确认 PhotographerWithTags（models.go）字段——若缺 mode/mutual_intro/certified，加字段 + GetPhotographerByUserID/GetPhotographerById SELECT 补列；加 UpdatePhotographerProfile。
- [ ] **Step 2: service**：ProfileUpdate 结构 + UpdateProfile/MyProfile + ErrInvalidMode/ErrProfileNotFound sentinels；GetByUser 返回补 mode/mutualIntro/certified。
- [ ] **Step 3: handler**：photographer_profile_handler.go（UpdateProfile bind ProfileUpdate、MyProfile）
- [ ] **Step 4: 路由**：PUT/GET /v1/photographers/profile（照 AuthRequired）
- [ ] **Step 5: 测试**：TestUpdateProfile_NotPhotographer/OK/InvalidMode；TestMyProfile_IncludesModeCertified
- [ ] **Step 6: 构建+冒烟+提交**：10000000001 登录 → GET profile/mine → 含 mode/mutualIntro/certified → PUT {"name":"光影行者","description":"新简介","location":"北京","mode":"both","mutualIntro":"提供互勉机会"} → 200 → GET → 更新 → 恢复（PUT 回原值）→ coser 403。commit "feat(c-end): photographer profile API (update/mine) + GetByUser fields"

---

### Task 2: C 端 profile-edit 页 + 入口 + 验证

**Files:**
- Create: `src/pages/photographer/profile-edit.vue`
- Modify: `src/pages/photographer/activate.vue`（主页管理入口）
- Modify: `src/api/index.ts`（getMyPhotographerProfile/updatePhotographerProfile）
- Modify: `src/pages.json`（注册 profile-edit）

**Interfaces:**
- Consumes: Task 1 全部 2 API
- Produces: 可用的主页编辑页（暗色）

- [ ] **Step 1: api**：getMyPhotographerProfile() → GET profile/mine；updatePhotographerProfile(payload) → PUT profile
- [ ] **Step 2: pages.json**：pages/photographer/profile-edit（navigationBarTitleText 主页管理）
- [ ] **Step 3: activate 入口**：isPhotographer 块加「主页管理」（与接单管理/作品管理并列）
- [ ] **Step 4: profile-edit.vue**（暗色）：onShow getMyPhotographerProfile 填充（昵称/简介 textarea/城市/接单模式 NSelect 互勉=free/收费=pay/两者=both/互勉说明 textarea）+ 保存按钮 → updatePhotographerProfile → toast → 返回
- [ ] **Step 5: vue-tsc + 浏览器**：13800138000 登录 → 主页管理 → 改简介 → 保存 → 详情页简介更新；commit "feat(c-end): photographer profile edit page + entry"

---

### Task 3: 验证 + 视觉 + AGENTS.md + 推送

- [ ] **Step 1**: 后端全量 build/vet/test
- [ ] **Step 2**: E2E（编辑→详情页联动）
- [ ] **Step 3**: agent-browser 暗色视觉
- [ ] **Step 4**: AGENTS.md 加「摄影师主页管理」小节（含 GetByUser 补全说明）
- [ ] **Step 5**: commit + push（git -c http.proxy= -c https.proxy= push gitee master）

---

## 风险与决策记录

- 全量提交：前端表单加载全量再提交，后端直接赋值（字段缺省=空串——前端全量保证）。
- mode 枚举校验复用 activate 语义（free/pay/both）。
- GetByUser 补全是 P2a 遗留修复（1 处 SELECT/struct 扩展——低风险）。
- 测试数据恢复：修改的简介原值恢复。
