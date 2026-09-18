# 摄影师自助定价 + 单轮报价 — 设计文档

- 日期：2026-09-17
- 状态：**待用户复核**（复核通过后才写实施计划）
- 范围：C 端约拍定价链路（摄影师侧 + coser 侧）
- 注：本仓库非 git 仓库，无法 commit，文档以文件形式留存

---

## 1. 背景与问题（需求重析）

### 现状（代码事实）

| 事实 | 证据 |
|------|------|
| `services` 是**平台级 4 个固定套餐**（¥399/¥699/¥1299/¥599） | `services` 表 4 行，无摄影师字段 |
| 任意摄影师详情返回**同一份**服务列表 | `photographer_service.go:260` → `s.queries.GetServices(ctx)`，**无摄影师过滤** |
| 摄影师表**无任何价格字段** | `photographers` 仅 `mode`(free/pay/both) + `mutual_intro` |
| 预约金额 = 全局套餐价，下单时快照进 `bookings.total_price` | `booking_service.go:141` `GetServiceById` |
| 无「摄影师↔服务」关联 | 全库表扫描确认 |

### 为什么这是缺陷（不是实现 bug）

1. **定价权在平台、不在供给方**：新人无法低价入行，头部无法溢价，市场是死的。
2. **`mode=互勉` 自相矛盾**：声明互勉的摄影师，页面照样挂 ¥399 套餐。
3. **完全没有协商通道**：coser 不能出价，摄影师不能报价。
4. `mode` 字段**有语义但与价格彻底脱钩**，是半成品。

### 目标

- 摄影师可**自主维护自己的套餐与价格**（含互勉 / 面议）。
- 面议套餐支持**单轮报价**：摄影师报价 → coser 接受/拒绝。
- 历史订单**不受套餐变更影响**（快照隔离）。

### 非目标（YAGNI）

- 多轮出价/还价
- 平台抽成、支付、退款、押金
- 动态定价/算法推荐
- admin 端套餐管理（现状 admin 完全不碰 services，保持不动）

---

## 2. 设计决策（已与用户确认）

| # | 决策 | 理由 |
|---|------|------|
| D1 | **自助定价 + 单轮报价 都要** | 定价权归供给方是地基；协商是协作层 |
| D2 | **套餐完全自定义**，平台 4 套餐降级为「一键预填模板」 | 自由度最高；模板解决冷启动 |
| D3 | **价格字段承载语义**：`0=互勉`、`NULL=面议`、`>0=固定价`；`mode` 降为展示标签 | 一套模型表达全部，避免 mode 与价格打架 |
| D4 | **单轮报价** | 状态机清晰、可审计；多轮复杂度高收益低 |
| D5 | **`services` 加 `photographer_id`，一表两用**（NULL=模板） | `bookings.service_id` 外键全部保持有效，迁移最小 |

---

## 3. 数据模型

### 3.1 `services` 改造

```sql
ALTER TABLE services ADD COLUMN IF NOT EXISTS photographer_id BIGINT;
ALTER TABLE services ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE services ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE services ALTER COLUMN price DROP NOT NULL;   -- 面议需要 NULL
CREATE INDEX IF NOT EXISTS idx_services_photographer ON services(photographer_id);
```

字段语义：

| 字段 | 模板行（`photographer_id IS NULL`） | 摄影师行 |
|------|--------------------------------------|----------|
| `price` | 建议价 | `NULL`=面议 / `0`=互勉 / `>0`=固定价 |
| `is_active` | 恒 true | 摄影师可下架 |
| `sort_order` | 预填顺序 | 展示顺序 |

**模板行不再直接上架**，仅作「一键预填」来源。

### 3.2 `bookings` 扩展

```sql
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_mode VARCHAR(10) NOT NULL DEFAULT 'fixed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS quote_price INTEGER;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_status VARCHAR(20) NOT NULL DEFAULT 'agreed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_name VARCHAR(100);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_duration INTEGER;
```

| 字段 | 取值 | 说明 |
|------|------|------|
| `price_mode` | `fixed` / `mutual` / `negotiable` | 下单时冻结 |
| `quote_price` | NULL / 正数 | 摄影师报价 |
| `price_status` | `agreed` / `awaiting_quote` / `quoted` / `rejected` | 价格是否定案 |
| `service_name` / `service_duration` | 快照 | **下单时冻结**，套餐改名/删除不影响历史单 |

**快照是硬要求**（见 4.3 盲点修复）。

---

## 4. 关键盲点（补强扫描发现，必须在实现中修复）

### 4.1 🔴 订单列表把「面议」渲染成 ¥0

`getBookingsByUserWithDetails` 现有 `COALESCE(s.price, 0) AS service_price`。
→ 面议套餐 `price IS NULL` → 显示 ¥0。

**修**：订单列表/详情的服务名与金额**优先读快照**（`b.service_name` / `b.total_price`），并按 `b.price_mode` + `b.price_status` 渲染「面议 / 待报价 / 已报价 ¥X / 互勉」。

### 4.2 🔴 订单服务名实时读表 → 套餐改名/删除会串历史单

现 join 取 `s.name AS service_name`。
→ 摄影师改名或删套餐后，历史订单显示会变/变空。

**修**：显示优先级 = `b.service_name`（快照） > `s.name`；写单时必定填快照。

### 4.3 `bookings` 列清单出现在 **6 处 SQL + 6 处 Scan**

实测 `bookings.sql.go`：

| 行 | 语句 | Scan |
|----|------|------|
| 13 | `createBooking` RETURNING | 39 |
| 56 | `getBookingsByUser` SELECT | 71 |
| 95 | `getBookingByID` SELECT | 102 |
| 142 | `updateBookingStatus` RETURNING | 148 |
| 153 | `getBookingsByUserWithDetails` SELECT（`b.` 前缀 + join） | 172 |
| 188 | `getBookingsByPhotographer` SELECT（`b.` 前缀 + join） | 205 |

→ 加 5 个字段要**同步 6 处 SELECT/RETURNING 与 6 处 Scan**，漏一处即运行时报错。（初稿写"4 处"是错的，已按实测更正。）

### 4.4 测试助手会破

`admin_manage_service_test.go:133 bookingScanValues()` 返回 **11 个字段**（被 booking / admin_manage 两批测试复用）。
→ 加列后必须同步；实施计划需包含这一步。

### 4.5 `services.price` 当前是 NOT NULL

→ 迁移必须先 `DROP NOT NULL`，否则面议存不进去。

### 4.6 🔴 下单未校验「套餐属于该摄影师」

现状 `CreateBooking` 直接用客户端的 `photographerId` + `serviceId`，**不校验二者归属**。
→ 有了私有套餐后，可拿 A 摄影师的 ¥1 套餐去预约 B 摄影师（越权/串价）。

**修**：`Create` 必须校验 `service.photographer_id == req.PhotographerID`，不匹配 → 400/403。

### 4.7 admin 侧无影响

扫描确认 admin handler/service/repo **完全不碰 `services`** → 范围可控。
另实测 `admin/src/views/photographer/index.vue` **不展示任何服务/价格** → admin 前端同样无需改动。

### 4.8 其他 touchpoint 核查（确认**不需**改动）

| 位置 | 核查结论 |
|------|----------|
| `server/mock_server.go`（:8081 遗留 mock） | 实测**不提供 services** → 无需改动 |
| `src/data/mock.ts` 的 `mockServices` | 实测全仓**仅定义处引用（死代码）** → 无功能影响。可选清理；不改也不会破（`price` 变 nullable 后数字字面量仍合法） |
| `src/types/index.ts:36 Service.price: number` | **需改** `number \| null`（已列在第 7 节） |

---

## 5. 状态机与报价流程

```
[固定价 price_mode=fixed]      下单 → price_status=agreed, total_price=套餐价
[互勉   price_mode=mutual]     下单 → price_status=agreed, total_price=0
[面议   price_mode=negotiable] 下单 → price_status=awaiting_quote, total_price=NULL
                                       │
                                       │ 摄影师报价 quote_price=X (>0)
                                       ▼
                                 price_status=quoted
                                       │
                    coser 接受 ────────┴──────── coser 拒绝
                        │                              │
                 status=confirmed               status=cancelled
                 price_status=agreed            price_status=rejected
                 total_price=X
```

- **报价即接单意愿**：被接受后直接 `confirmed`，不做二次确认（避免双确认的别扭）。
- 单轮：拒绝即终止该单（可重新下单）。
- 三种模式的 `confirmed → completed` / `cancelled` 沿用现有状态机（`canTransition` 不变）。
- 面议单在 `awaiting_quote`/`quoted` 阶段，任一方可取消（走现有 cancelled 分支）。

---

## 6. API 设计

### 6.1 摄影师套餐管理（全部 AuthRequired + 归属校验）

| Method | Endpoint | 说明 |
|--------|----------|------|
| `GET` | `/api/v1/photographers/services/mine` | 我的套餐（含未上架），按 sort_order |
| `POST` | `/api/v1/photographers/services` | `{name, price?, duration, description?, isActive?}` → 201 `{id}` |
| `PUT` | `/api/v1/photographers/services/:id` | 全量更新；**非本人套餐 → 403** |
| `DELETE` | `/api/v1/photographers/services/:id` | 硬删；**非本人 → 403**；已被历史订单引用不阻断（靠快照） |
| `GET` | `/api/v1/services/templates` | 平台模板（供"一键预填"） |

- `price` 可空（面议）；`0` 合法（互勉）；负数 → 400。
- 非摄影师调用 → 403 `not a photographer`（复用现有模式）。

### 6.2 报价

| Method | Endpoint | 角色 | 说明 |
|--------|----------|------|------|
| `POST` | `/api/v1/bookings/:id/quote` | 摄影师 | `{price}` → 200；仅 `price_mode=negotiable` 且 `price_status=awaiting_quote` 可报；`price` ∈ (0, 99999] |
| `POST` | `/api/v1/bookings/:id/quote/respond` | coser | `{accept: true/false}` → 200；仅 `price_status=quoted` 可响应 |

错误码：
- 非本人订单 → 403
- 状态不允许（重复报价 / 未报价就响应）→ 409
- 金额越界 → 400

### 6.3 改造现有接口

- `GET /api/v1/photographers/:id`：`services` 改为**该摄影师的**（`photographer_id = :id AND is_active`）。
- `POST /api/v1/bookings`：**请求体不变**（不新增 `priceMode` 字段）。服务端按所选套餐的 `price` 推导：`NULL → negotiable`、`0 → mutual`、`>0 → fixed`；下单时写 `price_mode` + 名称/时长快照，并执行 4.6 的归属校验。

---

## 7. 前端改动

| 页面 | 改动 |
|------|------|
| `photographer/profile-edit.vue` 或新增 `photographer/services.vue` | **我的套餐**：列表（名称/价格/时长/上下架/排序）+ 新增/编辑/删除 + 「一键预填平台模板」 |
| `photographer/detail.vue` | 展示该摄影师套餐；价格渲染 `¥399` / `互勉` / `面议` |
| `booking/index.vue` | 同上渲染；面议套餐下单 → 提示"提交后等待摄影师报价" |
| `order/list.vue`、`order/detail.vue` | 面议单显示价格状态；**摄影师**：报价按钮（弹窗输金额）；**coser**：接受/拒绝按钮 |
| `components/ServiceCard.vue` | ⚠️ 实测它在渲 `¥{{ service.price }}`（服务卡片组件）→ 必须支持 `面议`/`互勉` 渲染，否则显示 `¥null` |
| `photographer/list.vue` | 实测今天**不显示价格** → 本项是**新增**（非改造）：`¥399 起` / `互勉` / `面议` |
| `types/index.ts` | `Service.price` 改 `number \| null`；新增 `priceMode` / `priceStatus` / `quotePrice` |

**价格展示规则（统一，避免各处口径不一）**：

| 摄影师的套餐构成 | 列表页显示 | 详情页 |
|------------------|-----------|--------|
| 有任意付费套餐（含混合） | `¥<最低价> 起` | 逐个套餐真实价 |
| 仅有 0 元套餐 | `互勉` | `互勉` |
| 仅有面议套餐 | `面议` | `面议` |
| 无上架套餐 | `暂未设置` | 空态 |

---

## 8. 迁移与兼容

1. **000026**：`services` 加 3 列 + `price DROP NOT NULL` + 索引；
   **并把 4 个模板复制成每位现有摄影师的套餐**（避免演示环境出现"空套餐"）。
2. **000027**：`bookings` 加 5 列；存量单回填 `price_mode='fixed'` + `price_status='agreed'` + 名称/时长快照。
3. 全部语句**幂等**（`IF NOT EXISTS` / 条件 UPDATE），符合 `deploy/migrate.sh` 全量重放要求。
4. dev + prod 双库应用；prod 已知数据缺口（见 09-15 第七/八轮）继续保持幂等修复。

---

## 9. 边界与错误处理

| 场景 | 处理 |
|------|------|
| 套餐 `price` 为 `NULL`（面议）下单 | `price_mode=negotiable`、`total_price=NULL`、`price_status=awaiting_quote` |
| 报价金额 ≤0 或 >99999 | 400 |
| 重复报价 | 409 |
| coser 未报价就接受/拒绝 | 409 |
| 摄影师改价后，已报价订单 | **不影响**（`quote_price` + 快照已冻结） |
| 套餐被删 | 历史订单靠快照照常显示 |
| 拉黑关系 | 下单校验**仍然生效**（第九轮已实现，不可回退） |
| 时段冲突 | 仍然生效；且**鉴权先于冲突**（第九轮已实现顺序） |
| 摄影师无任何上架套餐 | 详情页显示空态"暂未设置套餐"，不显示平台模板 |

---

## 10. 测试策略

1. **单元测试**（`internal/service`）：
   - `Create` 按 `price` 推导 `price_mode` / `price_status` / `total_price` 的三种分支
   - **4.6 归属校验**：套餐不属于该摄影师 → 拒绝
   - 报价状态机：`awaiting_quote → quoted → agreed/rejected`；非法转移 409
2. **同步受影响的既有测试**：`bookingScanValues`（11→16 字段）、`photographer_service_test`、`photographer_handler_test`
3. **回归冒烟**：`server/scripts/smoke.sh` 新增「摄影师套餐 CRUD + 报价流程 + 面议渲染」用例组，dev/prod 双跑
4. **前端**：`vue-tsc` + agent-browser 走通「摄影师建套餐 → coser 面议下单 → 摄影师报价 → coser 接受 → 订单价格正确」

---

## 11. 风险与未决问题

| 项 | 说明 |
|----|------|
| 风险 | 迁移把模板复制给每位摄影师会**放大 services 行数**（演示环境 4→4×N），可接受 |
| 风险 | **6 处 SQL/Scan + `ServiceCard.vue`**（见 4.3 / 第 7 节）同步遗漏 → scope 比初稿大，靠冒烟脚本兜底 |
| 未决 | 面议单被拒后是否允许"重新下单同一时段"——本设计**允许**（原单已 cancelled） |
| 未决 | 价格区间展示（`¥399 起`）是否需要——本设计**列表页做，详情页不做聚合** |
| 未决 | 报价是否支持留言（如"含妆造+100"）——**不做**，YAGNI |

---

## 12. 建议实施顺序（供计划阶段参考）

1. **迁移 + 后端数据层**：000026 / 000027 + `bookings.sql.go` **6 处** SQL/Scan + 测试助手同步
2. **后端套餐 CRUD + 归属校验（4.6）**：改 `GetDetail` 取该摄影师套餐、新增套餐管理接口
3. **后端报价状态机**：quote / respond + 状态校验 + 单测
4. **前端摄影师侧**：我的套餐管理页 + 预填模板
5. **前端 coser 侧**：详情/预约的价格渲染 + 订单页报价交互
6. **回归**：`smoke.sh` 新增套餐/报价用例组 + dev/prod 双跑 + agent-browser 全链路

> 每阶段独立可验证，1-3 为后端（可先用 curl 验证），4-5 为前端，6 收口。
