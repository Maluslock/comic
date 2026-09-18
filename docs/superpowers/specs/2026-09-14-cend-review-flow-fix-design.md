# C 端「评价」流程修复（我的评价 / 评价某摄影师）— 设计

**日期:** 2026-09-14
**状态:** 已定稿（用户授权自主推进）

## 背景与问题（真 bug）

`src/pages/comment/index.vue` 把两件事混在一个页面：**「我的评价」**与**「评价某摄影师」**，并用 `pid = options?.photographerId || '1'` 兜底。

而四个入口**都没传 `photographerId`**：
- 个人中心 →「我的评价」`profile/index.vue:171`
- 摄影师详情 →「评价」`photographer/detail.vue:264`
- 订单列表 →「去评价」`order/list.vue:213`
- 订单详情 →「去评价」`order/detail.vue:285`

后果：
1. **「我的评价」显示的不是我的评价** —— 页面用 `pid='1'` 拉取 `/v1/photographers/1`，把**摄影师 1 收到的评价**当作「我的评价」展示。
2. **「评价摄影师」永远指向摄影师 1** —— 从任意摄影师详情/订单进来，都只能评摄影师 1。
3. 后端**没有**「我发表的评价」接口。

## 目标

1. 「我的评价」= 当前登录用户**自己发表**的评价（新接口）。
2. 「评价某摄影师」= 针对入口带来的**正确**摄影师（`photographerId` 由入口传入）。
3. 未登录时给出登录引导。

## 非目标（YAGNI）

- 评价的编辑/删除、分页、点赞、图片上传（现有 `images` 字段保持）。
- 不做「只能评价已完单摄影师」的业务校验（后续可加）。

## 方案

### 后端

- 新增 `GET /api/v1/reviews/mine`（`AuthRequired`）→ 当前用户发表的评价（新→旧），含 `photographerName`。
  - Repo：`ListReviewsByUser(ctx, userID) ([]Review, error)`（`reviews LEFT JOIN photographers`）。
  - `Review` 结构新增 `PhotographerName string`（仅该查询填充，`omitempty`）。
  - Service：`ReviewService.ListByUser(ctx, userID) ([]ReviewItem, error)`。
  - `ReviewItem` 新增 `photographerName`（可选）。
  - Handler：`MyReviews`。
- 路由：`GET /api/v1/reviews/mine`（注意与既有 `GET /api/v1/reviews/:photographerId` 同级；gin v1.10 支持 static+param 兄弟节点，注册时静态路由在前）。

### 前端

`src/pages/comment/index.vue`：

- `onLoad(options)`：`pid = options?.photographerId || ''`（**不再兜底 '1'**）。
- **我的评价**（始终展示）：`getMyReviews()` → `/v1/reviews/mine`；未登录 → 空列表 + 登录引导。
- **评价摄影师**（仅当 `pid` 非空时展示）：按 `pid` 拉取该摄影师用于展示头（名/头像），「评价」按钮打开评分弹窗。
- 提交：`POST /v1/reviews`（`photographerId = Number(pid)`），成功后关闭弹窗并**重新加载我的评价**。
- 页面标题/分区文案保持「我的评价」「评价摄影师」。

### 入口修正

| 入口 | 改动 |
|------|------|
| `profile/index.vue` `goReviews` | 保持 `?` 不带参（纯「我的评价」） |
| `photographer/detail.vue` `goReviews` | → `/pages/comment/index?photographerId=${id}` |
| `order/list.vue` `goReview(order)` | → `/pages/comment/index?photographerId=${order.photographerId}` |
| `order/detail.vue` `goReview` | → `/pages/comment/index?photographerId=${order.photographerId}` |

## 数据流

入口（带/不带 photographerId）→ comment 页 → `GET /v1/reviews/mine`（我的评价） + 可选 `GET /v1/photographers/:pid`（被评摄影师展示头）→ 提交 `POST /v1/reviews` → 重新拉「我的评价」。

## 错误处理

- 未登录：我的评价显示空态并提示登录；提交时校验登录。
- `pid` 非法/缺失：隐藏「评价摄影师」分区，不报错。
- 拉取失败：列表空态 + toast（best-effort）。

## 测试

- 后端单测（可选 seam）或 curl：
  - `GET /v1/reviews/mine` 只返回**本人**评价（造两条不同 user 的评价验证隔离）；未登录 → 401。
- 前端（agent-browser）：
  - 个人中心「我的评价」→ 页面只展示本人评价，无摄影师 1 的他人评价。
  - 摄影师详情「评价」→ 页面「评价摄影师」显示的是**该摄影师**。
  - 订单「去评价」→ 显示订单对应摄影师。

## 风险

| 风险 | 缓解 |
|------|------|
| 路由 `/reviews/mine` 与 `/reviews/:photographerId` 冲突 | gin v1.10 支持；启动时若 panic 则改用 `/reviews/mine/list` |
| 「我的评价」接口越权读他人评价 | 接口只用 token 的 userID，不接受客户端 userId |
| 旧入口遗漏 | 已 grep 全部 4 处入口 |

## 落地拆解（3 task）

1. 后端：`listReviewsByUser` + `Review.PhotographerName` + service `ListByUser` + handler `MyReviews` + 路由。
2. 前端：`comment/index.vue` 重构（我的评价 + 可选评价摄影师）+ 4 个入口传参。
3. 验证：`go build/vet/test` + curl（隔离性/401）+ agent-browser（三个入口）+ prod 重建 + AGENTS.md。
