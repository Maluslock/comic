# P2b C 端认证申请补全 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 摄影师端认证申请闭环：C 端激活页入口 + 申请/状态查看页（4 状态渲染），后端补状态查询端点。

**Architecture:** 后端 `GET /api/v1/photographers/cert-application`（复用 CertApplicationService 的 photographers + store 依赖，加 GetMyApplication 方法 + handler + 路由）；C 端新建 cert-apply.vue（未申请/待审核/已驳回/已通过 4 状态）+ activate 页入口 + api 函数 + pages.json 注册。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue3 + uni-app（uniapp，**暗色霓虹主题** $dark-*/$neon-*，非 admin 浅色）。

## Global Constraints

- 数据库：`comic-pg` docker（:5433）；**0 迁移**（cert-apply 后端已有，仅加 GET 端点）。
- 后端：`GET /api/v1/photographers/cert-application`（AuthRequired）→ 200 `{application: null}`（未申请）| 200 `{application:{id,status,reviewReason?,createdAt}}` | 403（非摄影师）。
- 复用 `CertApplicationService` 现有依赖：`photographers.GetPhotographerByUserID` + `store.GetByPhotographerID`（不可改其构造签名——用现有字段；若 GetByPhotographerID 在 interface 中已有则直接用；无 record → 返回 application null）。
- handler `MyApplication` 在已有 `CertApplicationHandler` 加方法（同一 handler struct，service 已注入）；main.go 注册（AuthRequired）。
- C 端 uniapp：暗色霓虹主题（$dark-bg-primary/$neon-purple/$neon-gradient 等 SCSS 变量，参照 src/pages/photographer/activate.vue 与 detail.vue 风格）；rpx；<script setup lang="ts">；约定 @/ 别名；单组件文件。
- C 端 api 层加 2 函数（src/api/index.ts）：`applyCertification(evidenceImages: string[], evidenceDesc: string)` → POST；`getMyCertApplication()` → GET。
- pages.json 注册 `pages/photographer/cert-apply`。
- C 端类型检查：`npx vue-tsc --noEmit`（根项目，非 admin）。
- 服务重启（server/，NO pkill）：`export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH; cd server && go build -o /tmp/mila-api ./cmd/api; kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2; (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3`。
- C 端测试账号：摄影师 10000000001（user 1001, certified=true 光影行者）、10000000002（user 1002 樱花落 certified=true）、10000000003（user 1003 暗夜骑士 certified=false, 申请#2 已驳回 reapply 可）、10000000004（user 1004 古风公子 certified=true）；验证码 `123456`（前缀 1234）。
- 无新依赖。后端 go build/vet/test + 前端 vue-tsc 必须过。
- 测试数据状态：E2E 若提交新申请（如暗夜骑士重提）→ 审批后留下 certified=true 或驳回——披露并注意演示数据一致性（暗夜骑士当前 certified=false 演示"待审核/可申请"状态，E2E 尽量用其他摄影师或完成后恢复）。

---

### Task 1: 后端 GET cert-application 状态端点

**Files:**
- Modify: `server/internal/service/cert_application_service.go`（加 GetMyApplication）
- Modify: `server/internal/handler/cert_application_handler.go`（加 MyApplication）
- Modify: `server/cmd/api/main.go`（注册路由）
- Test: `server/internal/service/cert_application_service_test.go`（加 2 测试）

**Interfaces:**
- Consumes: `CertApplicationService` 现有字段（store certAppStore、photographers photographerLookup）；`middleware.AuthRequired`、`middleware.UserID`；`ErrNotPhotographer`（已有 sentinel）
- Produces:
  - `CertApplicationService.GetMyApplication(ctx, userID int64) (*repository.CertApplication, error)` — GetPhotographerByUserID → pgx.ErrNoRows → ErrNotPhotographer；GetByPhotographerID → pgx.ErrNoRows → return nil（未申请）；其他 err 透传
  - `CertApplicationHandler.MyApplication(c)` — UserID 从 middleware → svc.GetMyApplication → 200 `{application: null}` 或 `{application: {...}}`；403 映射 ErrNotPhotographer
  - JSON：application 含 id/status/reviewReason(omitempty)/createdAt（repository.CertApplication 已有 json tags：id/userId/photographerId/evidenceImages/evidenceDesc/status/reviewReason,omitempty/adminId,omitempty/reviewedAt,omitempty/createdAt——**注意**：C 端只需 id/status/reviewReason/createdAt，返回完整对象或构筑精简 DTO；推荐 200 直接返回完整 CertApplication 对象（json tags 已含所需）——但 userId/photographerId/evidenceImages/evidenceDesc 是否暴露待定：**精简 DTO** `CertApplicationStatus{ID, Status, ReviewReason *string, CreatedAt}` 更干净。

- [ ] **Step 1: service 加 GetMyApplication**

`cert_application_service.go`：
```go
// GetMyApplication returns the caller's latest cert application (nil if never applied).
func (s *CertApplicationService) GetMyApplication(ctx context.Context, userID int64) (*repository.CertApplication, error) {
	profile, err := s.photographers.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotPhotographer
		}
		return nil, err
	}
	existing, err := s.store.GetByPhotographerID(ctx, int64(profile.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return existing, nil
}
```

- [ ] **Step 2: handler 加 MyApplication**

`cert_application_handler.go`（找到 `Submit` 所在 handler，加方法）：
```go
func (h *CertApplicationHandler) MyApplication(c *gin.Context) {
	userID := middleware.UserID(c)
	app, err := h.svc.GetMyApplication(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "application unavailable"})
		return
	}
	if app == nil {
		c.JSON(http.StatusOK, gin.H{"application": nil})
		return
	}
	// 精简 DTO（camelCase）
	c.JSON(http.StatusOK, gin.H{"application": gin.H{
		"id":           app.ID,
		"status":       app.Status,
		"reviewReason": app.ReviewReason,
		"createdAt":    app.CreatedAt,
	}})
}
```

- [ ] **Step 3: 注册路由**（main.go cert 相关路由旁）
```go
router.GET("/api/v1/photographers/cert-application", middleware.AuthRequired(userRepo), certH.MyApplication)
```

- [ ] **Step 4: 测试**（cert_application_service_test.go 加）
- TestGetMyApplication_NotPhotographer：GetPhotographerByUserID → pgx.ErrNoRows → ErrNotPhotographer
- TestGetMyApplication_NoApplication：store.GetByPhotographerID → pgx.ErrNoRows → (nil, nil)
- TestGetMyApplication_HasApplication：返回 existing（status "pending"）

- [ ] **Step 5: 构建 + 冒烟 + 提交**

```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./internal/... -count=1
# 重启 + 冒烟（Global Constraints 配方）
# C 端登录摄影师 10000000001 → tokenC1（已认证，应返回 application 或 null）
curl -s localhost:8080/api/v1/photographers/cert-application -H "Authorization: Bearer $tokenC1"   # 200 application (若申请过) 或 null
# coser 13800138000 → tokenCOSER（非摄影师）→ 403
curl -s localhost:8080/api/v1/photographers/cert-application -H "Authorization: Bearer $tokenCOSER"  # 403
# 暗夜骑士 10000000003 → tokenC3（有 rejected 申请）→ 200 application status=rejected + reviewReason
curl -s localhost:8080/api/v1/photographers/cert-application -H "Authorization: Bearer $tokenC3"   # 200 application rejected
```
提交：`git commit -m "feat(c-end): GET photographer cert-application status endpoint"`；NO push。

---

### Task 2: C 端 cert-apply 页 + activate 入口 + api 函数

**Files:**
- Create: `src/pages/photographer/cert-apply.vue`
- Modify: `src/pages/photographer/activate.vue`（入口按钮/已认证徽章）
- Modify: `src/api/index.ts`（applyCertification/getMyCertApplication）
- Modify: `src/pages.json`（注册 cert-apply）

**Interfaces:**
- Consumes: Task 1 API（GET /v1/photographers/cert-application → {application}；POST /cert-apply 已有）
- Produces: 可用的认证申请页

- [ ] **Step 1: api/index.ts 加 2 函数**（参照 getFavorites/getMe 模式）
```ts
export function applyCertification(evidenceImages: string[], evidenceDesc: string) {
  return apiPost<{ id: number }>('/v1/photographers/cert-apply', { evidenceImages, evidenceDesc })
}
export function getMyCertApplication() {
  return apiGet<{ application: null | { id: number; status: string; reviewReason?: string | null; createdAt: string } }>('/v1/photographers/cert-application')
}
```

- [ ] **Step 2: pages.json 注册**
`src/pages.json` pages 数组加 `{ "path": "pages/photographer/cert-apply", "style": { "navigationBarTitleText": "认证申请" } }`（参照 activate 注册样式）。

- [ ] **Step 3: activate.vue 入口**
isPhotographer 区域（`v-if="isPhotographer"` 块）：
```html
<view v-if="isPhotographer" class="activated-tip">
  <text class="tip-text">你已是摄影师</text>
  <view class="btn-go" @click="goOrders">去接单管理</view>
  <view class="btn-cert" @click="goCertApply">
    <text>{{ certified ? '✓ 已认证摄影师' : '申请认证' }}</text>
  </view>
</view>
```
- script 加 `certified` ref（onShow 时从 userStore.user 或后端 me 读取摄影师 certified——检查现有 activate 页是否已拿 certified；若无，onShow 调 apiGet `/v1/photographers/by-user/:userId` 或 `getPhotographerDetail` 拿 certified）。
- `goCertApply()` → `uni.navigateTo({ url: '/pages/photographer/cert-apply' })`。
- 样式 `.btn-cert` 参照 `.btn-go`（暗色霓虹：$neon-purple 边框/渐变）。

- [ ] **Step 4: cert-apply.vue**（4 状态渲染）
- `<script setup lang="ts">`：reactive `form = { evidenceImages: ['', '', ''], evidenceDesc: '' }`；status ref（'not-applied'|'pending'|'approved'|'rejected'）；application ref。
- onLoad/onShow 调 `getMyCertApplication()`：null → not-applied；status 映射。
- 未申请/重新申请（rejected→点击重新申请切回表单）：
  - 表单：evidenceImages 3 个 input（图片 URL，placeholder https://picsum.photos/...）+「+ 添加样片」按钮（添加输入框，max 5）+ each 带删除；evidenceDesc textarea 必填。
  - 提交：校验（至少 1 个非空 URL + desc 非空）→ `applyCertification` → 成功 toast「已提交，等待审核」→ 重新拉状态。
- pending：状态卡片「审核中」+ 提交时间（createdAt 格式化）。
- rejected：状态卡片「已驳回」+ reviewReason + 「重新申请」按钮 → 切回表单。
- approved：状态卡片「已通过（黄V 已生效）」+ 提示查看摄影师主页。
- 样式：暗色霓虹（$dark-bg-primary 页面、卡片 $dark-bg-card + 1rpx $dark-border、主按钮 $neon-gradient、状态色：pending=黄/approved=绿/rejected=红——用现有暗色方案参考 detail.vue 的 .cert-badge）。

- [ ] **Step 5: typecheck + 浏览器验证**
```bash
cd /vol1/1000/code/comic && npx vue-tsc --noEmit 2>&1 | grep -E "cert-apply|activate" | head -5   # 无错误
```
浏览器（:5173 C 端，暗色）：
- 摄影师 10000000003（暗夜骑士，rejected 可重提）登录 → 激活页 → 显示「申请认证」→ 点击 → cert-apply 页显示「已驳回 + 理由」→ 点击重新申请 → 表单 → 提交（1 URL + desc）→ 「审核中」→ 返回激活页。
- 摄影师 10000000001（已认证）→ 激活页显示「✓ 已认证摄影师」（若有申请记录）或申请按钮（若有）——验证显示逻辑按 certified 字段。
- coser 登录 → 激活页无「申请认证」路径（激活页只有已激活摄影师见——若未激活不显示入口）。
- 注意：cert-apply 页若暗夜骑士提交了新申请（pending），会破坏其"rejected 可重新提交"演示状态——E2E 完成后**用 SQL 恢复**（`UPDATE photographer_cert_applications SET status='rejected', review_reason='样片质量不足...' WHERE photographer_id=3` + 对应 photographers.certified=false）或披露（最好恢复，保持演示数据一致性）。

- [ ] **Step 6: Commit**
`git commit -m "feat(c-end): cert-apply page (4 states) + activate entry + api"`；NO push。

---

### Task 3: 验证 + E2E（流转）+ 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`（P2c/P2b 补《C 端认证申请》小节）
- Test-only

- [ ] **Step 1: 后端全量** — build/vet/test 全绿
- [ ] **Step 2: Playwright E2E（C 端 :5173，暗色）**：
 ① 摄影师 10000000003 登录 → 激活页「申请认证」→ cert-apply 页（rejected 状态+理由可见）→ 重新申请 → 提交（样片 URL+desc）→ 「审核中」
 ② 管理端 :5174 审核该新申请（approve）→ C 端回 cert-apply 页刷新 → 「已通过」
 ③ 激活页返回 → 「已认证摄影师」徽章出现（若 activate 页可刷新认证状态）
 ④ 恢复演示数据：新申请已 approved 则摄影师 1003 certified=true（可留作演示「待审核→已通过」案例并披露）；若需保持 1003 为 rejected 演示，则 SQL 恢复（自行裁决并披露）
- [ ] **Step 3: agent-browser 视觉复查（暗色霓虹）** — cert-apply 页 + activate 页截图：暗色主题（#0a0a1a 底、霓虹紫/青）、4 状态渲染、中文无乱码
- [ ] **Step 4: AGENTS.md** — 加《C 端认证申请》小节：GET 端点、C 端 2 页（activate 入口 + cert-apply 4 状态）、api 函数、演示数据状态
- [ ] **Step 5: Commit + 推送** — `git add AGENTS.md && git commit -m "docs: C-end cert-apply in AGENTS.md"`；`git -c http.proxy= -c https.proxy= push gitee master`；验证 unpushed=0

---

## 风险与决策记录

- **精简 DTO**：GET 返回 `{application: {id,status,reviewReason,createdAt}}`（不暴露 userId/photographerId/evidenceImages/Desc 到 C 端——最小暴露原则；repository.CertApplication json tags 完整但 handler 构造精简 gin.H）。
- **activate 页 certified 读取（已固化）**：activate 页 onShow 调 `getPhotographerDetail(photographerId)`（已有 API，detail 页消费同字段）拿 certified——**不碰后端**（P2a 遗留 GetByUser 缺 certified 是已知 gap，不改，避免范围蔓延）。若该实现遇阻上报，不自行改后端。
- **演示数据恢复**：暗夜骑士（1003）E2E 后恢复 rejected 或留 approved 披露——Task 3 Step 2 裁决。
- **C 端暗色**：cert-apply 页严格用 $dark-*/$neon-* 变量（不能把 admin 浅色/抄过来）。
