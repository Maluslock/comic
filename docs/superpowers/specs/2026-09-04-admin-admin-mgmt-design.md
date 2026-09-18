# P2c 后台管理员管理设计

**日期:** 2026-09-04
**状态:** 已批准设计（待实现）
**关联:** P2c admin web（父模块）、admins 表（P2c Task 1 已建）

## 目标

为管理端增加管理员账号管理：管理员列表、新建、禁用/启用、重置密码。这是 P1 安全卫生模块——种子账号 `admin/admin123` 自 P2c 起即标注"生产必须改密"，本模块是唯一实现途径；同时补全管理员账号生命周期（新建/禁用/改密）。

**非目标（YAGNI）**：权限矩阵/RBAC（role 字段预留 admin/super，super 区分 P2 二期）、操作日志（P2 独立模块）、角色 CRUD（固定两档）、邮箱/2FA（内网演示）。

## 数据模型

`admins` 表加 `status` 列（migration 000015）：
```sql
ALTER TABLE admins ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';
```
- `active` / `disabled` 两态。
- `disabled` 时：login 拒绝；已发 token 立即失效（`GetAdminByToken` 过滤 `status='active'`）。

## 管理 API（4 个，全部 AdminAuthRequired，前缀 /api/admin/v1）

| Method | Path | 行为 |
|--------|------|------|
| GET | `/admins` | 200 列表：id/username/role/status/createdAt（**不返回 password_hash**） |
| POST | `/admins` | `{username,password,role}` → 201 `{id}`；username 冲突 → 409；username/password 必填 |
| PUT | `/admins/:id/status` | `{status:active\|disabled}` → 200 `{ok:true}` \| 400 非法 status \| 404 \| **400 保护种子 admin(id=1) 不可禁用** |
| PUT | `/admins/:id/password` | `{password}` → 200 `{ok:true}`；重置后清 token（`SET token=NULL, token_expires_at=NULL`，旧会话失效）\| 404 |

- **种子保护**：`id=1`（username='admin'）禁用 → 400 `{"error":"cannot disable primary admin"}`；重置密码允许（改密是生产必需）。
- Login 校验：`AdminService.Login` 在密码校验后加 `if a.Status=="disabled" → ErrInvalidCredentials`（与无效密码同响应，不泄露状态）。
- `GetAdminByToken` 加 `AND status='active'`（禁用立即踢出所有会话）。

## 前端页面 `admin/views/admin/index.vue`

- 顶部：标题「管理员管理」+「新建管理员」NButton。
- NDataTable：用户名、角色 tag（admin/super）、状态 tag（启用=绿/禁用=红）、创建时间、操作列（重置密码 / 禁用|启用）。
- 操作：
  - 重置密码 → NModal（新密码 NInput password，≥6 位）；确认 → `PUT /admins/:id/password` → message + 提示"旧会话已失效"。
  - 禁用|启用 → NDialog 确认；`id=1`（admin 种子）禁用按钮 disabled + tooltip「主管理员不可禁用」；禁用成功后状态 tag 变红。
- 新建 → NModal（用户名 NInput、密码 NInput password、角色 NSelect admin/super）；409 → message「用户名已存在」。
- 复用 user/index.vue 模式（表格+对话框+message 提示）；单根 `<div>`；浅色主题；中文 UI。
- 菜单：menuIcons 加 `admin: 'mdi:account-cog-outline'`；locale zh-cn `admin: '管理员管理'` / en-us `Admin Management`；elegant-router 自动路由。

## API 层

`admin/src/service/api/admin.ts` 加：
- `fetchAdmins()` → GET `/v1/admins`，`?? []` 兜底
- `createAdmin(payload: {username,password,role})` → POST `/v1/admins`
- `setAdminStatus(id, status)` → PUT `/v1/admins/:id/status`
- `resetAdminPassword(id, password)` → PUT `/v1/admins/:id/password`

## 错误处理

- 401：token 缺失/无效（AdminAuthRequired 已有；disabled 后旧 token 即时 401）。
- 400：非法 status / 用户名密码缺失 / 种子 admin 禁用。
- 404：admin id 不存在。
- 409：username 唯一冲突。
- Login 对 disabled 返回 401（与密码错一致）。

## 测试策略

- 后端 `go test`：Login disabled → ErrInvalidCredentials；GetAdminByToken 禁用后返回错误；SetStatus 种子 400；SetStatus 正常/404；Create 409；ResetPassword 清 token（fake repo）。
- 前端 `pnpm typecheck`（green gate）。
- E2E：管理端登录 → 管理员管理页 → 列表（admin）→ 新建（testadmin/admin123）→ 新管理员可登录 → 禁用 → 旧 token 401 → 登录取新 token 401（disabled 拒）→ 启用 → 登录恢复 → 重置密码 → 旧 token 失效 → 新密码登录 → 清理测试管理员（删除 via SQL 或保留披露）。
- agent-browser 视觉复查：管理员页浅色专业风。

## 时间线（3 Tasks）

1. 后端：000015 admins.status + Login/GetAdminByToken 校验 + 4 API + 种子保护 → 冒烟 + push
2. 前端：admin/index.vue + api 函数 + 菜单图标/locale → typecheck + 浏览器跑通 + push
3. 验证：后端测试 + Playwright E2E + agent-browser 视觉复查 + AGENTS.md 补章节 + push

## 后续（不在本次）

- super 角色区分（仅 super 可管理管理员）
- 操作日志审计（P2）
- 管理员邮箱/2FA（生产）
