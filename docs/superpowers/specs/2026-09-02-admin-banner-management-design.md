# P2c 后台轮播图管理（Banner）设计

**日期:** 2026-09-02
**状态:** 已批准设计（写 spec + 计划中）
**关联:** P2c admin web（父模块）、C 端首页（消费 banners）

## 目标

为管理端增加轮播图管理，实现「管理端改一行 → C 端首页轮播立即生效」的运营闭环。Banner 是 P1 中工作量最小、风险最低的模块：`banners` 表已完整存在且 C 端已在消费，0 迁移、0 C 端改动。

**非目标（YAGNI）**：图片上传（粘贴 URL）、拖拽排序（数字 sort_order）、定时排期、链接非 event 类型、删除（下线即可）。

## 数据支撑

`banners` 表（已存在，无迁移）：
```sql
id BIGSERIAL PK, image_url TEXT, title VARCHAR, link_type VARCHAR,
link_id INTEGER, sort_order INTEGER, is_active BOOLEAN,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```
- C 端流程已通：`banners.sql.go` GetActiveBanners → home_repo.GetHome → home_service → C 端首页轮播。
- `link_type` 现有值：`event`（link_id 指向 comic_events.id）；本轮仅支持 `event`（页面链接延后）。
- 现有数据：3 条（ChinaJoy 2026 / 第40届萤火虫漫展 / CP33 综合同人展），is_active=t。

## 管理 API（4 个，全部 AdminAuthRequired，前缀 /api/admin/v1）

| Method | Path | 行为 |
|--------|------|------|
| GET | `/banners` | 200 列表（按 sort_order ASC）：id/imageUrl/title/linkType/linkId/sortOrder/isActive/createdAt |
| POST | `/banners` | `{imageUrl,title,linkType,linkId,sortOrder}` → 201 `{id}`；title/imageUrl/linkType 必填，linkType 仅允许 `event` |
| PUT | `/banners/:id` | `{imageUrl,title,linkType,linkId,sortOrder,isActive}` → 200 `{ok:true}`；404 不存在 |
| PUT | `/banners/:id/status` | `{isActive:bool}` → 200 `{ok:true}`；404 不存在 |

- 校验失败（缺 title 等）→ 400；非 event linkType → 400。
- 与 P2c 现有实现一致：独立 repo（*pgxpool.Pool 模式）、service 层、handler 层。

## 前端页面 `admin/src/views/banner/index.vue`

- 顶部：标题「轮播图管理」+「新增 Banner」NButton。
- NDataTable（全量列表，无需分页——banners 少量）：
  - 缩略图列：NImage 120×70（width/height 固定，响应式）
  - 标题、链接类型 tag（event → 「活动」）、链接 ID、排序、状态（NSwitch 上线/下线，调 `PUT /banners/:id/status`）
  - 操作列：编辑 → NDrawer 表单（图片 URL NInput、标题 NInput、linkType NSelect（仅「活动」）、linkId NInput number、sortOrder NInput number、isActive NSwitch）
- 新增/编辑共用 NDrawer 表单（编辑预填，新增清空）。
- 浅色主题、中文 UI；单根 `<div>` 模板约束；复用现有 admin 视图模式（photographer/index.vue 的 drawer+校验、event/index.vue 的 NSwitch 交互）。
- 菜单：menuIcons 加 `banner: 'mdi:image-multiple-outline'`；locale zh-cn `banner: '轮播图管理'` / en-us `Banner Management`。

## API 层

`admin/src/service/api/admin.ts` 加：
- `fetchBanners()` → GET `/v1/banners`，`?? []` 兜底
- `createBanner(payload)` → POST `/v1/banners`
- `updateBanner(id, payload)` → PUT `/v1/banners/:id`
- `setBannerStatus(id, isActive)` → PUT `/v1/banners/:id/status`

## 错误处理

- 401：token 缺失/无效（AdminAuthRequired 已有）。
- 400：请求体缺字段 / linkType 非 event。
- 404：banner id 不存在（PUT/status 两处）。

## 测试策略

- 后端 `go test`：banner 新增/编辑/状态切换/404/400 校验路径（fake repo 风格，沿用 cert/user 测试模式）。
- 前端 `pnpm typecheck`（green gate）。
- E2E：管理端登录 → 轮播图页列表显示 3 条 → 新增一条（标题「测试轮播」，linkId=1 ChinaJoy）→ 列表出现 4 条 → 编辑标题 → 状态下线 → C 端 home API 不返回它 → 恢复上线 → 删除测试数据（或保留并披露）。
- agent-browser 视觉复查：banner 页浅色专业风、缩略图渲染、表格正常。

## 时间线（3 Tasks）

1. 后端：banners 管理 API（repo+service+handler+路由 4 端点）+ 测试 → 冒烟 + 推送
2. 前端：banner/index.vue + api/admin.ts 函数 + 菜单图标/locale → typecheck + 浏览器跑通 + 推送
3. 验证：后端测试 + Playwright E2E + agent-browser 视觉复查 + AGENTS.md 补章节 + 推送

## 后续（不在本次）

- 图片上传（管理端直传对象存储）
- 拖拽排序 / 定时排期投放
- link_type 扩展非 event（页面跳转、外链）
- Banner 删除（若运营需要硬删，加 DELETE 端点）
