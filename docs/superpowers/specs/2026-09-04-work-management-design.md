# 摄影师作品管理（C 端上传/删除）设计

**日期:** 2026-09-04
**状态:** 设计（模块 1/4，自主执行）

## 目标

摄影师自助管理作品：上传（URL 输入，与 Banner/Cert 一致）+ 删除；作品出现在其详情页作品集（GetWorksByPhotographer 已消费，status 过滤已有）。

**非目标**：真图片上传（uni.uploadFile→对象存储，YAGNI）、作品编辑（上传/删除即可）、管理员侧作品管理已有（P2c 内容管理）。

## 后端（3 端点，AuthRequired）

| Method | Path | 行为 |
|--------|------|------|
| POST | `/v1/photographers/works` | `{title required, images[] required(≥1), description optional}` → 201 `{id}`；非摄影师 403 |
| GET | `/v1/photographers/works/mine` | 当前摄影师作品列表：id/title/images/description/status/createdAt |
| DELETE | `/v1/photographers/works/:id` | 删除自己的作品；非本人 403；不存在 404 |

- service：PhotographerService 加 `CreateWork(userID, req)`（校验摄影师身份 via GetPhotographerByUserID）+ `MyWorks(userID)` + `DeleteWork(userID, id)`（归属校验：works.photographer_id == 当前摄影师 id）。
- repo：调现有 `GetWorksByPhotographer`（mine 用）+ 新增 `InsertWork`（status 默认 active）+ `DeleteWorkByID`。
- C 端：`src/pages/photographer/works.vue`（暗色霓虹）——作品列表卡片（缩略图/标题/状态 tag）+ 新增表单（标题/图片 URL 动态 1-N/说明）+ 删除确认；入口：activate 页（摄影师已激活显示「作品管理」入口，与「去接单管理」并列）。
- api/index.ts 加 3 函数：`uploadWork({title, images, description})`、`getMyWorks()`、`deleteWork(id)`。
- pages.json 注册 `pages/photographer/works`。

## 测试

- 后端 go test：CreateWork 非摄影师 403 / 正常创建；DeleteWork 归属校验 403 / 404。
- vue-tsc clean；E2E：摄影师登录 → 作品管理 → 上传（2 图）→ 列表出现 → 详情页作品集包含 → 删除 → 消失。
- 视觉：暗色复查。
