# cend-review-flow-fix - Work Plan

## TL;DR (For humans)

**What you'll get:** 「我的评价」真正显示你自己写的评价；从摄影师详情/订单点「去评价」会评**对应那位**摄影师（现在永远评摄影师1，且「我的评价」显示的是摄影师1收到的评价）。

**Why this approach:** 这是入口漏传参 + 页面职责混淆的真 bug，修数据源与入口两处即可，不动评价表结构。

**What it will NOT do:** 不做评价编辑/删除/分页/图片；不校验「只能评已完单」。

**Effort:** Small（3 task） **Risk:** Low

---

> TL;DR (machine): Small effort, Low risk. Backend endpoint + frontend page/entry fix. 3 tasks in 2 waves.

## Scope

### Must have
- `GET /api/v1/reviews/mine`（AuthRequired）→ 本人评价，含 `photographerName`。
- `comment/index.vue`：我的评价用新接口；「评价摄影师」按入口 `photographerId` 定向。
- 4 个入口传参修正（profile 不带；detail/order×2 带）。

### Won't have
- 评价编辑/删除、分页、图片上传、完单校验。

## Tasks

### Wave 1 — 后端
**Task 1**
- `repository/reviews.sql.go`：`listReviewsByUser`（`reviews LEFT JOIN photographers`）+ `ListReviewsByUser`。
- `repository/models.go`：`Review` 加 `PhotographerName string`。
- `service/review_service.go`：`ReviewItem` 加 `photographerName`；`ListByUser`。
- `handler/review_handler.go`：`MyReviews`。
- `cmd/api/main.go`：`GET /api/v1/reviews/mine`（静态路由，AuthRequired）。
- 验收：`go build/vet/test` 绿；curl 本人隔离 + 未登录 401。

### Wave 2 — 前端
**Task 2**
- `src/api/index.ts`：`getMyReviews()`。
- `src/pages/comment/index.vue`：`pid` 可空；我的评价 ← `/v1/reviews/mine`；评价摄影师分区仅 `pid` 存在时渲染；提交后重载我的评价；未登录引导。
- 入口：`photographer/detail.vue`、`order/list.vue`（`goReview(order)`）、`order/detail.vue`。

**Task 3 — 验证**
- `vue-tsc`；agent-browser 三入口核验；prod 重建（api+web）；AGENTS.md 更新。

## Verification (final)
- `cd server && go build ./... && go vet ./... && go test ./...`
- curl：`/v1/reviews/mine` 只含本人；未登录 401。
- agent-browser：个人中心「我的评价」/ 摄影师详情「评价」/ 订单「去评价」三处。
