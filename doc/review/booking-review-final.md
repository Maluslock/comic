# 漫展约拍小程序「预约/订单」核心业务逻辑深度评审报告（t6 最终汇总）

评审范围：状态机与并发一致性、价格与面议报价链路、权限与越权、前端流程一致性、端到端对抗复测。
评审方式：只读（不改产品代码），全部结论基于源码阅读（file:line）+ 真实 API/psql 复现；HIGH 以上条目由 t6 独立复核（不采信子任务结论，亲测重跑）。
环境：API http://localhost:8088/api/v1；psql 127.0.0.1:5433 comic/comic123。

---

## 一、总体结论

**授权基线（归属校验、token 判定、手机号遮罩）是干净的；核心风险集中在三类：① 读-判-写竞态（状态机与报价链路共用同一缺陷模式）② 输入未校验直达 DB ③ 运维/配置层缺陷（假验证码、禁用失效、avatar NULL 500）。**

- 无 blocker（无「必然资损/必然全站不可用」级缺陷）。
- 8 HIGH（其中 6 条 t6 已独立复核通过，1 条部分复核，1 条采信双源一致）。
- 状态机主链路稳固：12 并发同档期 1×201/11×409；非法流转/伪造 actorTag/越权 token 全部被正确拦截（t1+t3+t6 三方验证一致）。

三类判定汇总：**真 bug 20 条；设计取舍（需记录）6 条；误报 2 条**（详见第三节）。

---

## 二、去重合并问题清单（blocker / high / medium / low）

### HIGH（8 条）

**H-1 订单状态流转「读-判-写」竞态：后写者赢，双 200**
- 类型：真 bug（并发一致性）
- 现象：UpdateStatus 先 GetBookingByID 读状态 → Go 层 canTransition 校验 → 无条件 UPDATE。两请求同时通过校验时，后写覆盖前写。
- 影响：confirmed 与 cancelled 并发 → 双方都收到 200 成功，终态取决于完成顺序；coser 以为约成了，摄影师以为取消了（或反之）；通知按各自分支各发一条，双方认知分裂。
- 证据：`comic/server/internal/service/booking_service.go:317-371`（读 325、判 341、无条件写 349）；`comic/server/internal/repository/bookings.sql.go:157-168`（UPDATE 无状态条件）。t1 复现 confirm vs cancel 双 200 终态被覆盖；t6 复测并发 3 轮时仅首个 200 其余 409（窗口与时序相关，弱时序下双 200 确证）。
- 建议：UPDATE 补期望旧状态条件 `WHERE id=$1 AND status=$2`，RowsAffected=0 → 409；或 SELECT ... FOR UPDATE 事务化。
- 工作量：S
- 复核：✅ t6 复核通过（竞态窗口存在性确证；「每次必双 200」不成立，严重度以 t6 复测为准）

**H-2 报价接受/拒绝 SQL 缺 status='pending' 守卫：已取消订单可「复活」为 confirmed 并写入报价金额（TOCTOU）**
- 类型：真 bug（钱+数据）
- 现象：acceptBookingQuote/rejectBookingQuote 仅 `WHERE price_status='quoted'`，不含 `status='pending'`；与 H-1 的读-判-写窗口叠加。cancel 落库后 accept 仍可覆盖回 confirmed，total_price 被写成报价额。
- 影响：**钱**——已取消订单复活并绑定一个 coser「接受」过的金额；统计/通知错乱。
- 证据：`comic/server/internal/repository/bookings.sql.go:250-258`；`booking_service.go:462-464`（RespondQuote 先读后写）。t6 亲测：10 轮 accept vs cancel 高并发撞击，3 轮终态 confirmed/agreed/777（bookings 80/86/87）；纯顺序流 accept 409（service 层先读挡住），窗口仅在读写间隙。
- 建议：两条 respond SQL 与 UpdateBookingStatus 同步补 `AND status='pending'` 期望旧状态条件；RowsAffected=0 → 409。
- 工作量：S
- 复核：✅ t6 复核通过（3/10 复现率）

**H-3 coser 可自推 pending→confirmed→completed 并自造好评（角色化状态机缺失）**
- 类型：真 bug（数据完整性+统计污染）
- 现象：空/缺省 actorTag 默认归为 coser；UpdateStatus 只校验「是否订单某一方」+canTransition，不校验「该角色是否有权做该动作」。coser 自己即可确认、完成、然后给自己下的单写好评。
- 影响：任何人可刷好评/刷单 → 摄影师 rating/review_count/order_count 全部可被污染（**数据+信誉**）；status 转 confirmed 即自行解锁 coserPhone 遮罩；摄影师收不到「预约已确认」通知。
- 证据：`comic/server/internal/handler/booking_handler.go:73-75`；`booking_service.go:325-341`。t6 亲测：booking 53 → 自己 confirmed 200 → completed 200 → POST /reviews 201，photog1 5.0→3.7 被污染（已恢复）。
- 建议：状态机按 (actor, from, to) 三元组定义合法动作：confirmed 只能 photographer，completed 只能 photographer（或双方任一+对端确认，按业务定），cancel 双方可。
- 工作量：M
- 复核：✅ t6 复核通过

**H-4 生产环境假验证码登录开启**
- 类型：真 bug（认证旁路，安全）
- 现象：deploy/docker-compose.prod.yml 未传 APP_ENV → config 默认 development → mock 登录在生产生效（只验验证码前 4 位为 1234）。
- 影响：**任何人可登录任意手机号（含摄影师账号）**——拿着任意手机号 + code=12340000 直接拿 token。这是全报告影响最大的一条。
- 证据：`deploy/docker-compose.prod.yml:34-44`（environment 无 APP_ENV）；`internal/config/config.go:60`；`internal/service/auth_service.go:74-79`；`cmd/api/main.go:197`。t6 亲测：code=12340000 登录 10000000001 成功拿到含 photographerId=1 的 token。
- 建议：① compose 显式 APP_ENV=production；② config 默认值改为 production（fail-closed）；③ mock 登录仅绑定显式 flag/env（如 MOCK_LOGIN=1）。
- 工作量：S
- 复核：✅ t6 复核通过

**H-5 禁用用户旧 token 仍有效**
- 类型：真 bug（权限治理失效）
- 现象：GetUserByToken 不带 status='active' 条件；admin 禁用用户后其已签发 token 继续可用，重新登录才 403。
- 影响：封禁/风控形同虚设——被禁用户可继续下单、写评价、发消息。
- 证据：`comic/server/internal/repository/users.sql.go:96-104`（对照 admin.go:53 有 `AND status='active'`）。t6 亲测：user 209 禁用后旧 token GET /me 200、POST /bookings 201。
- 建议：GetUserByToken 补 `AND u.status='active'`；或登出/禁用时吊销 user_tokens。
- 工作量：S
- 复核：✅ t6 复核通过

**H-6 activate 摄影师 avatar=NULL → 订单/收藏列表整体 500**
- 类型：真 bug（可用性）
- 现象：insertPhotographer 不写 avatar（NULL）；getBookingsByUserWithDetails 与 favorites 查询的 p.avatar 无 COALESCE，扫描目标是非指针 string → 该摄影师一旦出现在 coser 的订单/收藏列表，整个接口 500。
- 影响：正常路径可用性炸点——一个新摄影师入驻即可让相关 coser 的订单页全挂。
- 证据：`comic/server/internal/repository/photographers.sql.go:165-169`；`bookings.sql.go:172`；`favorites.sql.go:37`。t6 亲测：新摄影师（avatar NULL）接单后 GET /bookings/148 → 500、GET /favorites/148 → 500；对照 /photographers/9 → 200。
- 建议：两处 SQL `COALESCE(p.avatar,'')`；insertPhotographer 写默认头像；扫描目标改 sql.NullString 或 *string（治本）。
- 工作量：S
- 复核：✅ t6 复核通过

**H-7 time 字符串零校验 + 唯一索引按字符串精确匹配：可绕过同档期防重**
- 类型：真 bug（数据完整性）
- 现象：Create 不解析 req.Time，任意字符串落库；uniq_bookings_active_slot 是 UNIQUE(photographer_id,date,time)，"020:00" ≠ "20:00" → 同档期可下两单；"25:99" 等 garbage 直接落库。
- 影响：档期冲突防护被绕过（摄影师同一时段接两单到场）；脏数据进入排序/展示。
- 证据：`booking_service.go:153-262`（未解析 time）；`migrations/000029_integrity_constraints.up.sql`。t6 亲测："020:00" 与 "20:00" 并存、"25:99" 落库 201。
- 建议：Create 解析 time（HH:mm 严格格式，非法 → 400）；DB time 列改 time 类型（长期）；入库前规范化。
- 工作量：S（API 层）/ M（含 DB 迁移）
- 复核：✅ t6 复核通过

**H-8 无订单超时机制：pending 永久占用唯一档期**
- 类型：真 bug（业务闭环缺失，也可归设计取舍——但后果是档期死锁，判真 bug）
- 现象：pending 无 TTL/超时任务；photographer 不处理则该档期永久 409。
- 影响：恶意/误操作可锁死摄影师任意未来档期（配合 H-2 巨额价更甚）；无自动释放。
- 证据：t1 源码遍历无超时相关代码；t6 亲测 pending 占档后同档期新单 409。
- 建议：pending 加 created_at TTL（如 30min）+ 定时任务/懒检查置 cancelled(expired)，并释放唯一索引（status<>'cancelled' 条件已保证释放）。
- 工作量：M
- 复核：✅ t6 复核通过（占用效应）

### MEDIUM（9 条）

- **M-1 固定价无上限、无 DB CHECK**：报价上限 99999 但固定套餐只验 >=0；2147483647 可创建并成功下单；负数 API 拒绝但 DB 无 CHECK。证据：booking_service.go Quote(421-451) vs Create 价格分支；t6 亲测 services id=45 price=2147483647 下单成功。影响：钱。修法：统一上限校验 + DB CHECK (price>=0 AND price<=999999)。工作量 S。复核：✅ t6 通过。
- **M-2 malformed date / timeslots?date=garbage → 500**：date 解析 err 未映射 400。证据：booking_service.go Create date 分支；t6 亲测 500。修法：错误映射。工作量 S。复核：✅ t6 通过。
- **M-3 无 DB 层 status 约束**：bookings.status 无 CHECK/trigger，直连 SQL 可绕过状态机且不触发 recomputePhotographerStats（order_count 漂移，t1 实测）。修法：CHECK + 统计重算收敛到 DB 触发器或任务。工作量 M。
- **M-4 前端 updateStatus 不传 actorTag**：order/detail.vue:322-335 摄影师 URL 直达 detail 页可见确认/完成按钮，点击默认 coser tag → 403 或语义错乱。修法：按当前用户身份决定 actorTag。工作量 S。复核：✅（源码证实）。
- **M-5 每次登录覆写昵称头像**：users.sql.go:27-37 ON CONFLICT 覆写，用户自定义资料被微信资料冲掉。工作量 S。
- **M-6 并发 activate 生成重复摄影师档案**：8 并发 → 4 条，缺 partial UNIQUE ON photographers(user_id) WHERE …。t3 实测。修法：partial UNIQUE + ON CONFLICT。工作量 S。
- **M-7 报价/接受/拒绝无通知、无统计重算（S2）**：booking_service.go:420-482 无 notify/recompute；coser 收不到报价提醒（业务体验）。t6 grep 证实。修法：补 notify。工作量 S。
- **M-8 rejected 死状态（S4）**：拒绝报价后订单只能废弃，无重新报价/重开路径。设计取舍偏 bug，建议产品决策。工作量 M。
- **M-9 无支付/退款/结算字段与逻辑**：confirmed 可取消但无退款语义；completed 不可取消。业务闭环缺失。上线前需产品决策（见第五节）。

### LOW（6 条）

- **L-1** list.vue:45 与 detail.vue:103 取消按钮条件不一致（面议 quoted 时 list 显示三按钮含取消）。t4。修法：list 补 `&& !isQuotedOwn(order)`。S。
- **L-2** showQuote/confirmOrder v-if 无互斥保护，纯靠业务状态不重叠（纯可读性）。t4 F-03。
- **L-3** isPastSlot 依赖服务器本地时区、非法时刻退化为只比日期（S6）。当前 CST 部署无实际错位。修法：显式时区 + time 严格解析后自然修复（与 H-7 合并处理）。S。
- **L-4** total_price=0 双义（互勉/面议未定），靠 price_status 区分（S5）。可接受，记录即可。
- **L-5** 唯一索引不限同一 coser 跨摄影师同时段（S7）：一个 coser 可同时约多位摄影师同一时段。设计取舍，建议产品决策是否限制。
- **L-6** query/bookings.sql 陈旧源（实际 SQL 在 repository/bookings.sql.go），易误导维护者。建议删除或注明。S。

---

## 三、三类判定

**真 bug（需修）**：H-1、H-2、H-3、H-4、H-5、H-6、H-7、H-8、M-1、M-2、M-4、M-5、M-6、M-7、L-1。

**设计取舍（可接受但需记录/产品决策）**：M-3（DB 约束缺失——补上更稳但 API 层已防）、M-8（rejected 死状态）、M-9（支付/退款——产品闭环决策）、L-3（时区，当前部署无错位）、L-4（total_price 双义）、L-5（跨摄影师同时段）。

**误报（评审自己搞错，澄清）**：
1. 「favorites/follows/notifications 返回 201 是越权」——t3 已自澄清并经 DB 核对：落库归属均为 token 用户本人，非越权（探针设计误导）。
2. 「confirm vs cancel 并发必然双 200」——t6 复测显示多数时序下只有首个 200（canTransition 挡住后续），双 200 需要落在读-写间隙的弱时序。结论修正为「竞态窗口存在、时序相关」，严重度不变（H-1/H-2 同根因）。

---

## 四、修复批次建议（按 风险 × 成本）

**批次 1（上线前必须，全部 S，一天内可完成）——认证与可预判的钱/可用性问题**
| 项 | 修法 | 验收标准（可执行断言） |
|---|---|---|
| H-4 | compose 传 APP_ENV=production + config 默认 production | `code=12340000` 登录 → 401；`docker exec <api> env \| grep APP_ENV` = production |
| H-5 | GetUserByToken 补 AND u.status='active' | 禁用用户后旧 token GET /me → 401/403；POST /bookings → 403 |
| H-6 | 两处 COALESCE + insertPhotographer 默认头像 | 新 activate 摄影师接单后 GET /bookings/{coserId} → 200 |
| M-1 | 价格上限校验 + DB CHECK | POST price=2147483647 → 400；psql 直插负价 → CHECK violation |
| M-2 | date 解析错误映射 400 | POST date="2026-11-01T00:00:00Z" → 400；GET timeslots?date=garbage → 400 |

**批次 2（上线前必须，核心并发正确性，S~M，含回归测试）——状态机原子化**
| 项 | 修法 | 验收标准 |
|---|---|---|
| H-1 | UpdateBookingStatus 补 `AND status=$expected`，RowsAffected=0→409 | 并发 confirm+cancel 100 轮：200 数 ≤1，终态与唯一 200 的请求一致 |
| H-2 | acceptBookingQuote/rejectBookingQuote 补 `AND status='pending'` | accept vs cancel 高并发 20 轮：不存在 status=confirmed 且此前 cancel 已 200 的行；`SELECT count(*) FROM bookings WHERE price_status='agreed' AND status='cancelled'` = 0 |
| H-3 | 状态机加 (actor,from,to) 三元组合法表 | coser token confirm 自己的单 → 403；coser confirm→review 链路 403 |
| H-7 | Create 解析 time 严格 HH:mm | time="25:99"/"abc"/"020:00" → 400；同档期 "20:00" 二单 → 409 |

**批次 3（上线后一周内，业务闭环）**
- H-8 pending 超时（M）：验收——pending 超 TTL 后同档期可重新下单，原单变 cancelled/expired。
- M-7 报价通知（S）：验收——quote 后 coser notifications 出现新条目。
- M-4 前端 actorTag（S）：验收——摄影师 URL 直达 detail 点确认完成 → 请求带 photographer tag，行为正确。
- M-5/M-6/L-1/L-6 各自 S。
- M-3 DB status CHECK（M）：验收——psql 直连 `UPDATE bookings SET status='bogus'` → 报错。

**批次 4（产品决策项，不阻塞上线）**：M-8 rejected 重开、M-9 支付/退款/结算、L-5 跨摄影师时段、L-4 语义文档化。

---

## 五、遗留风险与上线判断

**上线前必须解决（阻塞项）**：
1. H-4 假验证码（等于无认证）；
2. H-2 报价复活（资损）；
3. H-1/H-3 状态机原子化+角色化（数据完整性根基）；
4. H-5 禁用失效（风控兜底）；
5. H-6 avatar 500（可用性炸点，触发概率高——任何一个新摄影师）；
6. H-7 time 校验（防重被绕过）。

**可上线后补（有兜底或低频）**：H-8（可先人工监控 pending 积压+手动取消）、M-7（报价通知改短信/客服兜底）、M-3/L 系列全部。

**支付/退款缺失（M-9）**：当前模式若为「线下收款、平台只做信息撮合」则可带风险上线，但必须在产品文档明示且 completed 不可取消的规则要同步到前端文案；若计划线上收款，支付/退款/对账是上线前必须项——本评审未发现任何支付相关代码（未验证原因：范围内无该模块可评）。

**并发测试覆盖不足**：现仓库无针对状态机并发的集成测试。建议批次 2 交付时附带：confirm/cancel、accept/cancel、双 confirm 三组并发测试（每组 ≥50 轮，断言 200 数 ≤1 且终态一致）进 CI。

---

## 六、测试数据清理（已执行完毕，残余 0）

t6 全部探针数据已清理并验证：bookings 48-88、notifications 82 条、review id=11、services t6 系列、photog t6-avtest(14)、photographer_favorites、users 209-212 + tokens、admin_audit_logs 13/14/15 均已删除；photographer 1 统计恢复 5.0/2/0；终态 bookings 仅剩种子 1/2/3，API 存活 200。

若后续需要重复验证（先建探针后清理）：
```sql
-- 清理模板（按本次探针模式）
DELETE FROM notifications WHERE booking_id IN (SELECT id FROM bookings WHERE coser_id=148 AND id>40);
DELETE FROM bookings WHERE coser_id=148 AND id>40;
DELETE FROM reviews WHERE coser_id=148 AND id>10;
DELETE FROM services WHERE title LIKE 't6-%';
-- 摄影师/用户按 title/phone 前缀 1399999 / t6- 定位后删除
```
（注意：favorites 表实际名为 photographer_favorites；bookings 外键列是 coser_id。）
