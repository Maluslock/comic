# P2a 双角色身份体系：人人可开通摄影师 + 接单面板 — 设计文档

**日期**: 2026-08-28
**状态**: 设计定稿（用户确认参考 click-shot 优点、按修正设计开工）
**范围**: 米拉漫展小程序 — 闲鱼式分层身份体系的 P2a：人人可开通摄影师身份 + 双身份共存 + 摄影师接单面板

## 1. 背景与决策依据

**产品本质**：漫展约拍是兴趣社交（coser 与摄影师是同一群人，角色随时互换），非纯商业撮合。因此采用**闲鱼式分层**：

- **L1 零门槛**：任何用户一键开通"摄影师"身份，注册即发布（互勉约拍 or 自定低价），无审核
- **L2 认证**：人工审核的"认证摄影师"（作品+履历），头像旁徽章——**用黄V/金V语义**（个人创作者认证；蓝V 留给未来机构——行业共识：蓝V=营业执照机构认证）
- **双身份共存**：同一账号既是 coser 又是 photographer（Airbnb 房东/房客、B站观众/UP主模式），按行为切换

**借鉴 click-shot（GitHub My-Polaris/click-shot）的 4 个优点**：
1. **分层标签体系**（实名用户/自由摄影师/约拍模特/VIP——每级权限与认定）
2. **三区分区**（收费区=提供服务 / 付费区=求服务 / 互勉区=无金钱交易）
3. **约拍门槛**（点击约拍需 1 元定金，防恶意；未达成 24h 退回）
4. **平台托管定金**（交易单成立先付定金，完成付尾款才释放给摄影师）

## 2. 目标（P2a 范围）

1. **一键开通摄影师身份**（零门槛）：用户点"我是摄影师"→ 填档案 → 立即生效，无审核
2. **双身份共存**：同一账号 coser + photographer 并存；个人中心显示真实身份标签（不再是摆设切换）
3. **摄影师接单面板**：摄影师账号看到"收到的预约"（pending→确认接单/拒绝；confirmed→完成）
4. **接单模式区分**（吸收 click-shot 三区思想，P2a 简化版）：photographer 档案加 `mode`（free=互勉 / pay=收费 / both）+ 简介；详情页展示模式标签
5. **徽章展示位预留**：认证摄影师（黄V/金V）徽章位先展示（certified 字段），认证申请/审核流程 P2b 做

**明确不做**（P2b/P2c）：认证申请+审核后台；1元定金/平台托管交易（支付相关，P4 或后置）；机构/蓝V；VIP 订阅制；模特身份（约拍模特是独立角色，YAGNI 后置）。

## 3. 架构决策

### 3.1 身份模型（登录即角色，双身份共存）

- `users` 表是唯一身份；**photographers.user_id 绑定 = 该用户是摄影师**（可多个用户绑一个？否——1:1，一个 user 一个 photographer 档案）
- 认证时：`login`/`me` 返回中加 `photographerId`（若有）——**据此前端识别双身份**
- 角色不再用 `users.role` 单值（现有字段保留兼容，但身份推导改为：有摄影师档案 = 也是摄影师）
- **双身份共存**：user 同时有 coser 行为（下单/收藏）和 photographer 档案（接单）——无需切换，两面板都在

### 3.2 数据库改动

- `photographers` 表加字段：
  - `mode VARCHAR`（'free' | 'pay' | 'both'）默认 'free'——接单模式
  - `mutual_intro TEXT`——互勉说明（用户想约啥）
  - `certified BOOLEAN DEFAULT false`——认证摄影师（展示徽章位；P2b 做申请置位）
  - `activated_at TIMESTAMPTZ`——开通时间
- 无新表（认证申请表 P2b 再加）

### 3.3 后端 API

- `POST /api/v1/photographers/activate`（AuthRequired）body `{userId, name, styleTags[], intro, mode, mutualIntro}` —— 一键开通：创建 photographers 行（user_id 绑定）+ 返回 photographerId
- `PUT /api/v1/photographers/:id/profile`（AuthRequired）—— 更新档案（mode/简介/标签）
- `GET /api/v1/photographers/by-user/:userId` —— 查当前用户的摄影师档案（或空）
- `GET /api/v1/bookings/photographer/:photographerId`（AuthRequired）—— 按摄影师查收到的订单（联查 coser 姓名/头像/电话）
- `login`/`me` 返回加 `photographerId?: number`（有档案则带）

### 3.4 前端

| 位置 | 改动 |
|------|------|
| 个人中心 | 移除"切换身份"摆设；显示真实身份双标签（Coser + 摄影师徽章位）；若未开通显示"我是摄影师"入口（一键开通）；已开通显示"接单管理"入口 |
| 摄影师开通页 | 新页面 `pages/photographer/activate`（表单：昵称/头像/风格标签/简介/接单模式选择 free/pay/both + 互勉说明）——参照摄影师的暗色霓虹表单风格 |
| 接单管理页 | 新页面 `pages/photographer/orders`（收到的预约列表：coser 信息/时间/服务/状态 + 确认接单/拒绝/完成按钮）——复用订单列表联动（同 UpdateStatus API） |
| 订单列表（coser 视角） | 移除"确认接单（演示）"（coser 不该确认）；保留取消/确认完成/评价 |
| 摄影师详情页 | 展示 mode 标签（互勉/收费/both）+ 认证徽章位（certified 显示） |

### 3.5 权限校验（防越权）

- `UpdateStatus`：**严格校验操作者身份**——coser 只能改自己下的单（cancelled/completed）；摄影师只能改自己收到的单（confirmed/cancelled/completed 对应状态机）。handler 查 booking 的 coser_id/photographer_id vs token user——防"coser 把摄影师订单标确认"（现有 409 只挡状态机迁移，不挡越权）
- demo 简化：摄影师账号 = user 有 photographer 档案；校验 = booking.photographer_id ↔ token user 的 photographer.id 匹配

## 4. 数据流

```
开通: 个人中心"我是摄影师" → POST /photographers/activate → photographerId → 双身份
接单: 摄影师账号 → 接单管理 → GET /bookings/photographer/:pid
  → pending 订单 → 确认接单(PUT status=confirmed) / 拒绝(cancelled)
  → confirmed → 完成(completed)
coser 下单后: 订单列表(自己) → 状态流转；摄影师在接单管理看到同一单
```

## 5. 错误处理

- 未登录 → AuthRequired 401 → client.ts 跳登录
- 重复开通 → 409（已有档案）
- 越权状态修改 → 403
- 空接单列表 → empty 状态"暂无预约"

## 6. 测试策略

- 后端: Go 测试——activate 幂等、by-user 查询、photographer bookings 联查、UpdateStatus 越权 403（coser 改摄影师单）
- 前端: vue-tsc + Playwright H5 实测：
  1. coser 登录 → 个人中心"我是摄影师"→ 开通（填表单）→ 双身份标签出现
  2. 摄影师登录（10000000001 或刚开通账号）→ 接单管理 → 看到收到的预约（用另一账号下单造数据）
  3. coser 视角订单列表无"确认接单（演示）"
  4. 越权：coser 尝试 PUT 摄影师订单 status → 403
- 回归: 现有 coser 下单/订单/评价链路不受影响；go test 全绿

## 7. 里程碑

- M1: DB 迁移（photographers 加 mode/mutual_intro/certified/activated_at）+ activate/by-user/photographer-bookings API + login/me 带 photographerId + UpdateStatus 越权校验
- M2: 前端个人中心双身份 + 开通页 + 接单管理页 + 订单列表移除演示 + 详情页徽章/模式标签
- M3: 全链路验证 + 视觉复查 + AGENTS.md + 推送

## 8. 假设记录

1. 方向 = 闲鱼式分层 P2a（用户确认，参考 click-shot）
2. 双身份共存，不做角色切换摆设
3. 认证摄影师徽章位先展示（certified 字段），认证申请 P2b
4. 1元定金/平台托管 = 支付范畴后置（记入 P4 候选）
5. 视觉复查发现问题列 task 纠正（用户授权）
