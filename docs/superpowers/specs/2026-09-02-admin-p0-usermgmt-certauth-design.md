# P2c 后台功能补全（P0：账号管理 + 认证申请审核）设计

**日期:** 2026-09-02
**状态:** 已批准设计（待实现）
**关联:** P2c admin web（父模块）、P2b 认证申请流（本设计实现 P2b 管理端+C端申请）

## 目标

补齐后台最明显的两个功能缺口（第一阶段的 P0 双模块）：

1. **账号管理**：管理端查看/检索/封禁用户（C 端用户体系目前后台零可见性）。
2. **认证申请审核（P2b 管理端 + C 端申请）**：摄影师通过 C 端提交认证申请 → 管理端审批 → 通过后黄V（certified）生效 + 通知申请人。现有"摄影师审核"页（手动 toggle）保留作为人工调整手段，申请流成为认证的第一入口。

**非目标（YAGNI）**：P1 模块（管理员管理/通知公告/内容管理/轮播图/报表导出）、P2 模块（操作日志/摄影师详情）留待后续；不做聊天监管、复杂 RBAC、支付、实时推送。

## 模块 A：账号管理

### 数据模型
`users` 表新增 `status` 列（migration 000013）：
```sql
ALTER TABLE users ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';
```
- `active` / `disabled` 两态（不做多级冻结，YAGNI）。
- C 端 login：`status='disabled'` → 返回 403 `{error:"account disabled"}`（AuthService.Login 加校验，1 处改动）。
- 封禁时**不清除 token**（简化：已登录会话继续有效，仅拒绝新登录；真离线需清 user_tokens——记 minor 延后）。

### 管理 API（3 个，全部 AdminAuthRequired，前缀 /api/admin/v1）
| Method | Path | 行为 |
|--------|------|------|
| GET | `/users` | 分页列表，`?keyword=&role=&status=&page=&pageSize=`；role=photographer 由 `photographers.user_id` JOIN 推导；返回 id/phone/name/avatar/role/status/createdAt/photographerId |
| GET | `/users/:id` | 聚合详情：资料 + 统计（bookings 数/reviews 数/favorites 数/follows 数）+ 最近 5 条订单（join service/photographer） |
| PUT | `/users/:id/status` | `{status:"active"\|"disabled"}` → 200 `{ok:true}`；404 不存在 |

### 前端页面 `admin/src/views/user/index.vue`
- 顶部：搜索栏（NInput 关键词：手机号/昵称）+ NSelect 角色筛选（全部/摄影师/Coser）+ NSelect 状态筛选 + NButton 查询/重置
- NDataTable：头像（NAvatar）、昵称、手机号、角色 tag、状态 tag（启用=绿/禁用=红）、注册时间、操作列（详情 / 封禁|解封按钮）
- 详情抽屉（NDrawer）：NDescriptions 基本资料 + 统计卡（4 个：订单/评论/收藏/关注）+ 最近订单小表格
- 分页（复用 order/index.vue 的 remote pagination 模式）
- 封禁二次确认（NDialog）

## 模块 B：认证申请审核

### 数据模型（migration 000014）
```sql
CREATE TABLE photographer_cert_applications (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  photographer_id BIGINT NOT NULL REFERENCES photographers(id),
  evidence_images TEXT[] NOT NULL,          -- 样片图（URL 数组）
  evidence_desc  TEXT NOT NULL,             -- 说明（作品简介/拍摄经历等）
  status        VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending/approved/rejected
  review_reason TEXT,                        -- 驳回理由（approved 时可空）
  admin_id      BIGINT REFERENCES admins(id),
  reviewed_at   TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cert_app_user ON photographer_cert_applications(user_id);
CREATE INDEX idx_cert_app_status ON photographer_cert_applications(status, created_at DESC);
```
- 一用户一申请：以 `photographer_id` UNIQUE 约束（或 Go 侧查重后 INSERT）——已 pending/approved 的摄影师不可重复申请，rejected 可重新申请。
- C 端详情页 `photographers.certified=true` 显示认证徽章（已实现）；申请按钮仅对 `certified=false` 的用户显示。

### C 端 API（1 个，AuthRequired）
| Method | Path | 行为 |
|--------|------|------|
| POST | `/api/v1/photographers/cert-apply` | `{evidenceImages:[], evidenceDesc}` → 201 `{id}`；摄影师必须已激活（有 photographer_id）；已有 pending/approved 申请 → 409 `{error:"already applied"}` |

### 管理 API（3 个，AdminAuthRequired）
| Method | Path | 行为 |
|--------|------|------|
| GET | `/admin/v1/cert-applications` | 分页，`?status=&page=&pageSize=`；返回 id/userId/name(user join)/photographerId/photographerName/evidenceImages/evidenceDesc/status/createdAt |
| GET | `/admin/v1/cert-applications/:id` | 详情（含已审核 admin 信息） |
| PUT | `/admin/v1/cert-applications/:id/review` | `{action:"approve"\|"reject", reason?}` → 200；**approve 事务**：application.status=approved + `photographers.certified=true` + 写 notifications（成功通知申请人）；reject：status=rejected + reason + notifications（warning 通知）；非法状态（非 pending）→ 409 |

### 前端页面 `admin/src/views/certification/index.vue`
- NTabs 状态筛选（待审核/已通过/已驳回）
- NDataTable：申请人、摄影师、提交时间、状态 tag、操作列（通过 / 驳回）
- 详情抽屉：材料图（NImage 预览组）+ 说明 + 申请人信息
- 通过 → 确认后调 API；驳回 → NModal 填理由（必填）
- 复用 order/index.vue tabs+table+pagination 模式

## 复用与架构

- 后端：复用 `repository`（pqxpool 遍历）、`service`（handler→service→repo）、`middleware.AdminAuthRequired`（已有）、`notify`/notifications 表（已有）。
- 前端：复用 `admin/src/service/api/admin.ts`（加函数）、`request/admin.ts`（已封装 token，不改）；photographer/index.vue 的 table+switch 模式、order/index.vue 的 tabs+remote pagination 模式。
- DTO JSON 字段 camelCase，与现有 admin API 一致。

## 错误处理

- 401：token 缺失/无效（AdminAuthRequired 已有）。
- 403：C 端 login disabled 用户；C 端 cert-apply 未激活摄影师。
- 404：users/:id、cert-applications/:id 不存在。
- 409：重复申请、审批非 pending 状态。

## 测试策略

- 后端 `go test`：UserRepo.Status 切换、CertApplicationRepo 查重/审批事务（approve 联动 certified + notification）、login disabled 403、非法状态 409。
- 前端 `pnpm typecheck`（green gate，node_modules 35 条基线除外）。
- E2E（Playwright）：管理端登录 → 用户列表搜索/封禁/解封 → 用户详情汇总；C 端摄影师（10000000001）提交申请 → 管理端审批通过 → C 端详情页黄V 徽章出现 + 通知收到；驳回流同理。
- agent-browser 视觉复查：user/certification 两页浅色专业风。

## 时间线（4 Tasks）

1. 后端：000013 users.status + UserRepo+Service+Handler（users 列表/详情/status）；login disabled 403 → 冒烟 + push
2. 后端：000014 cert_applications + C 端 cert-apply + 管理端 3 API（含审批事务）→ 冒烟 + push
3. 前端：user/index.vue + certification/index.vue + api 函数 → pnpm typecheck + 本地跑通 + push
4. 验证：后端测试 + Playwright E2E（双端全链）+ agent-browser 视觉复查 + AGENTS.md + push

## 后续（不在本次）

- P1：管理员管理/通知公告/内容管理/轮播图/报表导出
- P2：操作日志/摄影师详情
- minor 延后：封禁不清除已发 token（真正离线需清 user_tokens）；申请材料进一步扩展（作品链接字段）
