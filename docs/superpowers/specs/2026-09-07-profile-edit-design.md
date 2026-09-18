# 摄影师主页管理（资料编辑）设计 + 计划

**日期:** 2026-09-07
**状态:** 设计（模块 2/4，自主执行）

## 目标

摄影师编辑自己的主页资料：昵称/简介/城市/接单模式/互勉说明。补全 P2a 遗留（GetByUser 缺 mode/mutualIntro/certified 字段——本模块一并修）。

**非目标**：头像上传（登录种子 avatar，YAGNI）、标签自选（P2a 已有摄影师详情 tags，编辑延后）、作品管理（模块 1 已做）。

## 后端（2 端点，AuthRequired）

| Method | Path | 行为 |
|--------|------|------|
| PUT | `/v1/photographers/profile` | `{name?, description?, location?, mode?, mutualIntro?}` → 200 `{ok:true}`；非摄影师 403 |
| GET | `/v1/photographers/profile/mine` | 200 完整资料（id/name/avatar/location/description/mode/mutualIntro/certified/rating/reviewCount/orderCount）|

- repo：`UpdatePhotographerProfile(ctx, id int64, name, description, location string, mode string, mutualIntro *string) error`（UPDATE 全字段或 COALESCE；RowsAffected=0 → ErrProfileNotFound）——**注意**：部分字段缺省时用 COALESCE 保留原值（前端只传改变的字段）或全量传（前端表单加载全量再提交——**决定：前端全量提交**，后端 UPDATE 直接赋值，字段缺省=空串覆盖需前端保证）。
- 修复 GetByUser（P2a 遗留）：返回加 `mode/mutualIntro/certified`（repo PhotographerWithTags 可能已有——检查 models.go 的 PhotographerWithTags 是否含这些列；含则直接映射）。
- service：`UpdateProfile(userID, req)`（非摄影师 → ErrNotPhotographer）+ `MyProfile(userID)`。
- C 端：`src/pages/photographer/profile-edit.vue`（暗色霓虹）——onShow 拉 mine 填充表单（昵称/简介/城市/接单模式 NSelect 互勉·收费·两者/互勉说明）+ 保存 → toast+返回。入口：activate 页「主页管理」。
- api/index.ts：`updatePhotographerProfile(payload)` + `getMyPhotographerProfile()`。
- pages.json 注册 `pages/photographer/profile-edit`。

## 测试

- 后端 go test：UpdateProfile 非摄影师 403/正常；MyProfile 返回含 mode/mutualIntro/certified。
- vue-tsc clean；E2E：摄影师登录 → 主页管理 → 改简介/城市 → 保存 → 详情页更新。
- 视觉：暗色复查。
