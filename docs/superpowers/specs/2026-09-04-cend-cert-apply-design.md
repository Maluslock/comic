# P2b C 端认证申请补全 设计

**日期:** 2026-09-04
**状态:** 已批准设计（待实现）
**关联:** P2b 认证申请流（后端 API 已通）、P2c 认证审核页面（管理端已上线）

## 目标

补全摄影师端认证申请闭环：C 端激活页入口 + 申请/状态查看页。后端 `POST /api/v1/photographers/cert-apply` 已存在（P2c 建），管理端审核页已上线——当前缺 **C 端入口 UI 与状态查询端点**，导致摄影师无法自助提交/查看，两端不闭环。

**非目标（YAGNI）**：图片上传（URL 输入，同 Banner 决策）、多申请历史（一摄影师一活跃申请语义，GET 仅返回最新）、催审/申诉流程、C 端申请记录列表。

## 后端（1 个新 C 端端点）

```
GET /api/v1/photographers/cert-application    (AuthRequired)
→ 200 { application: null }                        （未申请过）
→ 200 { application: { id, status, reviewReason?, createdAt } }  （最近一条）
→ 403                                             （非摄影师，无 photographer_id）
```
- 复用 `CertApplicationService.GetByPhotographerID`（已有，返回 repository.CertApplication{ID, Status, ReviewReason *string, CreatedAt}）。
- service 加 `GetMyApplication(ctx, userID int64) (*CertApplication, error)`：查 photographer_id（`queries.GetPhotographerByUserID`）→ 无则 ErrNotPhotographer；有则 GetByPhotographerID → 无记录返回 nil。
- handler 加 `MyApplication(c)`；main.go 注册（非 USE_MOCK 分支，AuthRequired）。
- JSON 字段：`application: {id, status: 'pending'|'approved'|'rejected', reviewReason: string|null, createdAt}`。

## C 端（2 处 UI）

### 1. 激活页入口（`src/pages/photographer/activate.vue`）
- 已激活（isPhotographer）区域：
  - `certified=false` → 显示「申请认证」按钮 → 跳转 `/pages/photographer/cert-apply`
  - `certified=true` → 显示「✓ 已认证摄影师」状态（黄V 徽章样式，参照详情页 .cert-badge）

### 2. 认证申请页（新建 `src/pages/photographer/cert-apply.vue`）
三种状态渲染：
- **未申请**：申请表单——样片图 URL 输入（3-5 个，动态添加/删除）+ 说明 textarea（必填）→ 提交 `POST /v1/photographers/cert-apply` → 成功提示「已提交，等待审核」→ 切换到状态查看
- **待审核**：状态卡片「审核中」+ 提交时间
- **已驳回**：状态卡片「已驳回」+ 驳回理由 + 「重新申请」按钮（回到表单，可再提交）
- **已通过**：状态卡片「已通过（黄V 已生效）」
- 页面载入调 `GET /v1/photographers/cert-application` 初始化状态。

### 前置交互
- 激活页入口在 onShow（或返回时）刷新 certified 状态（申请通过后返回激活页应显示已认证徽章）。

## API 层（`src/api/index.ts` 加 2 函数）

- `applyCertification(evidenceImages: string[], evidenceDesc: string)` → POST `/v1/photographers/cert-apply`
- `getMyCertApplication()` → GET `/v1/photographers/cert-application`

## 错误处理

- 401：token 缺失/失效（AuthRequired 已有；client.ts 统一路由登录页）。
- 403：已申请 409（重复提交，服务层 ErrAlreadyApplied → 409 已有）。
- 400：表单不满足（evidenceImages 空/evidenceDesc 空）。

## 测试策略

- 后端 `go test`：GetMyApplication 非摄影师 403 / 无记录 application=null / 有记录返回 status；复用现有 CertApplication 测试模式。
- 前端 `vue-tsc`（根项目 `npx vue-tsc --noEmit`——注意这是 uniapp 项目非 admin）。
- E2E（C 端 :5173）：摄影师 10000000001（已认证）→ 激活页显示已认证徽章；1002 樱花落（certified=true 当前）→ 同；1003 暗夜骑士（已驳回可重提）→ 激活页显示申请认证 → 申请页显示已驳回+理由 → 重新申请提交 → 待审核 → （管理端审批）→ 返回激活页已认证；1004 古风公子（已认证）同理。
- agent-browser 视觉复查：激活页 + 申请页暗色霓虹——**注意：C 端是暗色主题**（$dark-*/$neon-*），非浅色。

## 时间线（3 Tasks）

1. 后端：`GET /v1/photographers/cert-application` + service.GetMyApplication + handler + 路由 → 冒烟 + push
2. 前端：cert-apply 页（4 状态渲染）+ activate 页入口 + api 函数 + pages.json → typecheck + 浏览器跑通 + push
3. 验证：后端测试 + C 端 E2E（提交→审批→状态流转→徽章）+ agent-browser 视觉 + AGENTS.md + push

## 后续（不在本次）

- 图片上传（uni.uploadFile 到服务端存储）
- 申请多历史/催审
- 摄影师作品上传（独立模块）
