# P2c 后台内容管理（作品 / 评论 / 标签）设计

**日期:** 2026-09-04
**状态:** 已批准设计（待实现）
**关联:** P2c admin web（父模块）；works/reviews/tags（C 端已有消费）

## 目标

管理端内容风控：作品下架/恢复、违规评论删除、标签 CRUD。这是 P1 最后一个模块——UGC 平台（作品/评论）的基础运营能力。

**非目标（YAGNI）**：作品编辑（内容源自 ingester，仅下架）、评论软删/恢复（硬删除，YAGNI）、标签合并、批量操作、作品/评论审核流（先发后审现状）。

## 数据模型

`works.status`（migration 000017）：
```sql
ALTER TABLE works ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';
```
- `active` / `down` 两态；下架作品 C 端不可见。
- `reviews` **硬删除**（无 status 列，YAGNI）；`tags` 直接 CRUD（无 status）。
- `photographer_tags` 标签引用表：删除标签前检查引用（无引用才删，有引用 400）。

## C 端改动（2 处）

- 作品查询过滤下架：`queries.GetWorks` / `GetPhotographerWorks`（或类似）加 `AND status='active'`；摄影师详情 works 部分同样过滤。
- 评论：硬删天然生效（无 C 端改动）。

## 管理 API（8 个，全部 AdminAuthRequired，前缀 /api/admin/v1）

| Method | Path | 行为 |
|--------|------|------|
| GET | `/works` | 作品列表：id/title/images[0] 缩略图/photographerName/status/createdAt；`?photographerId=&status=&page=&pageSize=` → {list,total} |
| PUT | `/works/:id/status` | `{status:'active'\|'down'}` → 200 `{ok:true}` \| 400 非法 \| 404 |
| GET | `/reviews` | 评论列表：id/userName/userAvatar/rating/content/createdAt/photographerName；`?keyword=&photographerId=&page=&pageSize=` → {list,total} |
| DELETE | `/reviews/:id` | 删除 → 200 `{ok:true}` \| 404 |
| GET | `/tags` | 标签列表：id/name/usageCount |
| POST | `/tags` | `{name}` → 201 `{id}` \| 409 重名 \| 400 空名 |
| PUT | `/tags/:id` | `{name}` 重命名 → 200 `{ok:true}` \| 404 \| 409 重名 |
| DELETE | `/tags/:id` | 删除 → 200 `{ok:true}` \| 404 \| 400 有摄影师引用 |

- 独立 repo（*pgxpool.Pool 模式）+ service（校验）+ handler 三层。
- sentinels：`ErrTagExists`（23505/查重）、`ErrTagInUse`（引用检查）、`ErrInvalidStatus`（复用现有或新建）。

## 前端页面 `admin/views/content/index.vue`（单页 3 tabs）

- NTabs：作品 / 评论 / 标签。
- **作品 tab**：NDataTable（缩略图 NImage 100×70、标题、摄影师、状态 tag 上架=绿/下架=红、时间、操作：下架|恢复按钮）+ 筛选（摄影师 NSelect + 状态 NSelect）+ remote 分页。
- **评论 tab**：NDataTable（头像、用户名、摄影师、评分（N 星或数字 tag）、内容、时间、操作：删除 NDialog 确认）+ keyword 搜索 + remote 分页。
- **标签 tab**：NDataTable（名称、使用量、操作：编辑/删除）+ 新增标签按钮 → NModal；删除 NDialog 确认。
- 复用 banner/user/notification 模式；单根 `<div>`；浅色主题；中文 UI。
- 菜单：menuIcons 加 `content: 'mdi:book-open-page-variant-outline'`；locale zh-cn `content: '内容管理'` / en-us `Content`；elegant-router 自动路由。

## API 层

`admin/src/service/api/admin.ts` 加 8 函数：fetchAdminWorks/fetchAdminWorkDetail(不需要——列表已够)/setWorkStatus、fetchReviews/deleteReview、fetchTags/createTag/updateTag/deleteTag。

## 错误处理

- 401：token 缺失/无效。400：非法 status/空标签名/标签有引用。404：作品/评论/标签不存在。409：标签重名。

## 测试策略

- 后端 `go test`：work 下架后 C 端不可见（repo level 或集成）；review 删除；tag 重名 409/有引用 400/正常 CRUD。
- 前端 `pnpm typecheck`（green gate）。
- E2E：管理端登录 → 内容管理页 → 作品 tab 下架一个 → C 端作品列表不可见 → 恢复；评论 tab 删除一条 → C 端评论数减少；标签 tab 新增/重命名/删除（无引用）。
- agent-browser 视觉复查：内容页 3 tabs 浅色专业风。

## 时间线（3 Tasks）

1. 后端：000017 works.status + C 端作品过滤 + 8 管理 API → 冒烟 + push
2. 前端：content/index.vue（3 tabs）+ api 函数 + 菜单图标/locale → typecheck + 浏览器跑通 + push
3. 验证 + E2E（双端）+ agent-browser 视觉复查 + AGENTS.md 补章节 + push

## 后续（不在本次）

- 作品内容审核流（先审后发）
- 评论软删/恢复
- 标签合并/批量操作
- 作品编辑（内容源管理）
