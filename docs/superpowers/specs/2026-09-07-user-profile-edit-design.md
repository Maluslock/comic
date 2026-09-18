# 用户资料编辑（昵称/头像）设计 + 计划

**日期:** 2026-09-07
**状态:** 设计（模块 4/4，自主执行）

## 目标

用户编辑自己的昵称/头像。目前昵称是登录自动生成（用户xxx），头像 DiceBear 种子——无编辑手段。

**非目标**：手机号更换（登录主键）、真头像上传（URL 输入，YAGNI 一致）、实名认证。

## 后端（1 端点，AuthRequired）

| Method | Path | 行为 |
|--------|------|------|
| PUT | `/v1/me/profile` | `{name required, avatar optional}` → 200 `{ok:true}`；name 空 → 400 |

- UserRepo 加 `UpdateProfile(ctx, id int64, name, avatar string) error`（UPDATE users SET name=$2, avatar=$3 WHERE id=$1）。
- handler：profile/me 路由或新建 user_profile_handler——**决定**：main.go 内联 handler（me 已有内联模式）或小 handler；用 me 同风格 gin.H。
- C 端：`src/pages/profile/edit.vue`（暗色）——昵称 NInput + 头像 URL NInput（placeholder 提示 DiceBear 格式）+ 保存 → `PUT /v1/me/profile` → 更新 userStore.user（name/avatar）→ toast + 返回。
- profile/index.vue：头像/昵称区加「编辑」入口 → navigateTo('/pages/profile/edit')。
- api/index.ts：`updateUserProfile({name, avatar})` → PUT /v1/me/profile + 更新 userStore（或页面直接改 store）。

## 测试

- go test：UpdateProfile 正常/404（不存在 id——AuthRequired 下一般不会）。
- vue-tsc clean；E2E：登录 → 个人中心 → 编辑 → 改昵称/头像 URL → 保存 → profile 页与 me 接口更新。
- 视觉：暗色复查。
