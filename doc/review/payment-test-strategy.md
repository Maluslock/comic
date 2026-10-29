# 无正式商户号前提下的支付测试策略（mock / 重放 / 小额真测）

> 回答一个问题：「没有正式商户号，支付能不能测？怎么测？」
> 范围：只做测试策略与设计，不含实现代码。
> 关联：`booking-state-machine.md`（订单状态机）、`booking-review-final.md` M-9（支付/退款缺失项）。
> 项目现状核实（2026-10）：`server/migrations/` 至 000032 无任何支付/退款/对账表；`server` 内无 wechatpay/out_trade_no/refund 相关 Go 代码；`src/` 无 `wx.requestPayment` 调用。支付模块尚未开工，本文是开工前的测试基建设计。

---

## 0. 结论先行

1. **v2 沙箱已死，v3 从未有过公开沙箱** → 涉及「真实资金流」的验证必须有真实商户号，绕不开。
2. 但支付代码里 **80% 的逻辑不依赖真实微信**：下单参数组装、调起签名、回调验签、AES-GCM 解密、订单状态机、幂等、对账比对——全部可以用**本地 mock 网关 + 自签密钥对**测到绿。
3. 真正需要商户号的只剩一小部分：真实收银台拉起、真实回调投递、真实退款到账、真实账单下载。这部分用 **0.01 元最小验证集** 一次性打穿，预估 1~2 个工作日、1~2 人。
4. 架构前提：支付层必须抽象成 `PaymentProvider` 接口（`Prepay / QueryOrder / Refund / DownloadBill / VerifyNotify / DecryptNotify`），生产实现包 `wechatpay-go`，测试实现走 mock 网关。没有这个抽象，mock 路径根本插不进去。

---

## 1. 现状核实：沙箱到底还有没有

### 1.1 v2 沙箱（`sandboxnew` + `getsignkey`）已下线

- 2022-09-02 微信团队在开放社区发布置顶公告《【重要】微信支付沙箱功能升级通知》
  （https://developers.weixin.qq.com/community/pay/doc/0002c437a1c6f00ba87ef1a1a5ec01），
  v2 仿真测试系统随后逐步收缩。
- 更早的事实：**v2 沙箱从来只覆盖「付款码支付成功用例 / 付款码支付异常用例」两类接口**，
  JSAPI 的统一下单 `pay/unifiedorder` 一直不支持沙箱（开放社区 2023-11-15 帖子引用官方文档原话，见 1.3 引用列表）。所以就算 v2 沙箱活着，对 JSAPI 也没用。
- 现在调用 `https://api.mch.weixin.qq.com/sandboxnew/...` / `pay/getsignkey` 已不可用。

### 1.2 v3 没有公开沙箱

- 微信支付商户文档中心（pay.weixin.qq.com/doc/v3）从产品页到通用规则页，**没有任何「沙箱/仿真环境」入口**；v3 文档提供的替代物只有两个：
  1. **签名验签工具**（Windows 客户端，`/doc/v3/merchant/4012365352`）：离线计算/校验请求签名、调起收银台签名、回调验签——只验证「签名格式与算法」是否正确，不联网、不产生交易。
  2. **商户签名验签／加解密测试接口** `POST /v3/security/echo`（`/doc/v3/merchant/4014551946`）：微信把你上传的密文原样回显，用于自测签名、验签、公钥加密、AES-GCM 解密——**仍然需要真实商户号才能调用**。
- 开放社区多方印证（官方答复口径一致：v3 无沙箱，联调用真实商户号 + 小额真实交易）：
  - 2023-11-15「微信支付 JSAPI 支付有沙箱环境吗？」
  - 2024-02-28「微信支付 jsapi v3 有沙盒模式吗?」
  - 2024-12-15「微信支付不提供沙箱环境……测试环境的订单怎么对账？」
  - 2025-09-14「V3 找不到沙箱环境测试 API 地址」
  - 2025-11-21「微信支付 V3 没有沙箱环境吗？这样的话直接使用真实环境开发测试吗？」
  - GitHub `wechatpay-apiv3/wechatpay-go` issue #172「SDK 缺少沙箱测试环境」（closed，官方未提供）。

### 1.3 依据小结

| 主张 | 依据 |
|---|---|
| v2 沙箱下线 | 2022-09-02 官方沙箱升级公告 + 现调用 `getsignkey` 失效 |
| v2 沙箱从未支持 JSAPI 下单 | 官方文档原文（仅付款码支付两类用例），社区帖引用 |
| v3 无公开沙箱 | 文档中心无沙箱入口；社区 2023→2025 多帖官方口径一致；wechatpay-go #172 |
| 官方唯一测试抓手 | 签名验签工具（离线）+ `/v3/security/echo`（需真商户号） |
| 结论 | **无商户号阶段 = 本地 mock 测业务逻辑；真测阶段 = 真实商户号 + 0.01 元验证集** |

### 1.4 替代路径（无商户号能做什么）

| 能力 | 无商户号可行性 | 手段 |
|---|---|---|
| 下单/回调/退款的全套业务逻辑 | ✅ 完全可测 | 本地 mock 网关（第 2 节） |
| 回调验签 + AES-GCM 解密代码 | ✅ 完全可测 | mock 自建 RSA 密钥对 + 测试 APIv3 密钥（§2.3） |
| 请求签名算法正确性 | ✅ 基本可测 | 用自测密钥按官方签名串格式生成，用官方「签名验签工具」交叉验证格式 |
| 收银台真实拉起 | ❌ 不可能 | 需真实 appid+商户号，且开发者工具不弹真实支付窗（§2.4） |
| 真实回调投递、真实对账单 | ❌ 不可能 | 真测阶段（第 5 节） |
| 真实退款到账 | ❌ 不可能 | 真测阶段（第 5 节） |

---

## 2. 本地 mock 网关设计（mockpay）

### 2.1 架构：PaymentProvider 接口 + 双实现

```
server/internal/payment/
├── provider.go          # 接口定义
├── wechat.go            # 生产实现（wechatpay-go），商户号就绪后接
├── mockpay.go           # 测试实现：进程内假网关
└── mockpay_server.go    # （可选）独立进程版：模拟微信侧角色
```

接口（示意签名）：

```go
type PaymentProvider interface {
    Prepay(ctx, PrepayReq) (PrepayResp, error)      // → prepay_id
    QueryOrder(ctx, outTradeNo string) (OrderState, error)
    Refund(ctx, RefundReq) (RefundResp, error)
    DownloadBill(ctx, billDate string) ([]BillRow, error)
}
// 回调侧独立于 provider：
type Notifier interface {
    VerifyNotify(req *http.Request, rawBody []byte) (*NotifyEvent, error) // 验签
    DecryptNotify(n *NotifyEvent) (*PaymentResult, error)                 // AES-GCM 解密
}
```

关键设计点：**回调验签/解密不做成假货**。mock 网关发出的回调必须是「格式 100% 同构的真回调」，由后端真实验签代码处理，只是验签公钥换成 mock 的密钥对。这样验签、解密、幂等、状态机代码在 mock 阶段就是被测对象。

### 2.2 伪造 jsapi 下单响应

- mockpay 收到 `Prepay` 请求后：
  1. 校验必填字段（appid / mchid / out_trade_no / amount.total / payer.openid / notify_url）齐全合法——**负向用例在这里测**（缺字段、amount.total≤0、out_trade_no 格式非法）；
  2. 生成 `prepay_id = "mock" + 随机串`，按 `out_trade_no` 记录在 mockpay 自己的内存/测试表里（模拟微信侧订单）；
  3. 后端拿到 prepay_id 后，按官方「调起收银台」规则签名：签名串 = `appId\ntimeStamp\nnonceStr\nprepay_id=xxx\n`，用商户 API 证书私钥（测试用自生成密钥）做 SHA256-RSA——**这段调起签名代码同样被真实执行**；
  4. 返回给小程序的五元组 `{timeStamp, nonceStr, package, signType:"RSA", paySign}` 与真实结构一致。
- 微信侧幂等语义顺带模拟：同一 `out_trade_no` 重复 Prepay，mockpay 返回**同一个 prepay_id**（真实微信对未支付订单重复下单也是幂等的），供 §3.1 用例使用。

### 2.3 伪造并签名回调（核心）

mockpay 用三样测试密钥构造一条与真实回调完全同构的 HTTP 请求：

| 要素 | 真实值来源 | mock 对应 |
|---|---|---|
| Wechatpay-Serial | 微信平台证书序列号 / 公钥 ID | mock 平台证书序列号（配置进后端验签模块的信任列表） |
| Wechatpay-Signature | 微信平台私钥对 `timestamp\nnonce\nbody\n` 签名 | mock RSA 私钥按同一格式签名 |
| resource.ciphertext | 微信用 APIv3 密钥做 AEAD_AES_256_GCM 加密 | 测试 APIv3 密钥（如 32 字节 `mockapiv3key...`）加密同样结构的 resource |
| event_type | `TRANSACTION.SUCCESS` / `REFUND.SUCCESS` | 同名，还附加 `TRANSACTION.CLOSED`、`REFUND.ABNORMAL` 等用于负向用例 |

流程：`mockpay 构造回调 → POST 到后端 /api/v1/payments/notify（本地回环或内网穿透）→ 后端跑真实验签+解密+幂等+状态机 → 返回 {"code":"SUCCESS"}`。
**篡改用例**：把 body 改一个字节再重放 → 验签必须失败并返回 401；把 ciphertext 解出的 amount.total 改掉 → 状态机必须拒单（§4.2）。
**重放用例**：同一条回调原样重发 N 次 → 幂等（§3.2）。

- 本地联调回调可达性：mockpay 与后端同机，直接 `http://127.0.0.1:8080/...` 回环投递即可；若要联调公网通知路径，用内网穿透（ngrok / frp / sakurafrp）挂临时 HTTPS 域名。

### 2.4 如何驱动 wx.requestPayment 在开发者工具里走通

先说清楚一个限制，避免排错浪费时间：

- **微信开发者工具不会弹真实支付窗**。`wx.requestPayment` 在工具里要么直接 fail（`requestPayment:fail`），要么提示不支持；真机预览如果 appid 没绑商户号，也会 fail。这不是 bug，是平台限制。
- 所以 mock 阶段的驱动方案是**双轨**：

| 轨道 | 触发方式 | 覆盖内容 |
|---|---|---|
| A. 前端交互轨 | 小程序端检测 mock 模式（后端 Prepay 响应带 `mock:true` 标志或环境配置），**不调** `wx.requestPayment`，改为弹「模拟收银台」UI（成功 / 取消 / 密码错误三个按钮），点「成功」调 `POST /api/v1/payments/mock/confirm` | 前端支付流程、loading/结果页/订单页状态流转全部走通 |
| B. 回调真实轨 | `mock/confirm` 由 mockpay 网关（而非业务 handler）直接生成**签名回调**投递到 notify 端点 | 验签→解密→幂等→状态机→通知，全链路真实代码 |

- 注意红线：mock 确认接口**只能存在于 `PAY_MOCK=1` 构建**，生产构建编译期剔除（Go build tag / 环境变量双重保险），上线前用 grep + 启动自检确认 `mock/confirm` 返回 404。这是防「mock 接口漏到生产 = 免费下单」的硬性验收项。
- 取消路径同样重要：前端点「取消」→ 前端记录用户取消态，但**不**改订单状态（真实微信里用户取消不会回调，订单保持 pending 直到超时关单）；「超时关单」用 mockpay 模拟 `TRANSACTION.CLOSED` 回调或后端定时任务关单来测。

---

## 3. 幂等与并发用例

前置：bookings 状态机现状为 `pending/confirmed/completed/cancelled`（见 booking-state-machine.md）。支付接入建议在 bookings 上加 `pay_status`（`unpaid/paid/refunding/refunded`）+ 独立 `payments` 表（含 `transaction_id`、`out_trade_no`、`amount_total`、`paid_at`），`out_trade_no = "BK" + bookingId + 重付序号`。下述断言以该设计为基准。

### 3.1 同一 out_trade_no 重复下单

| # | 操作 | 断言 |
|---|---|---|
| D1 | 同一 out_trade_no 连续 Prepay 2 次（间隔 < 5 分钟） | 两次返回**同一 prepay_id**；payments 表仍 1 条待支付记录；bookings 不新增 |
| D2 | 同一 out_trade_no，第二次请求金额不同（篡改场景） | mockpay/后端拒绝或忽略金额变更，金额以后端 DB 中 bookings.total_price 为准 |
| D3 | 用户对已 paid 的订单再次发起支付 | 后端返回 409/幂等返回已有结果，不创建新预支付单 |
| D4 | 已支付订单用**新** out_trade_no 再次下单 | 后端拒绝（pay_status=paid 不允许再开单），防止一单多付 |

### 3.2 回调重复投递（微信语义：至少一次，必会重复）

| # | 操作 | 断言 |
|---|---|---|
| N1 | 同一条支付成功回调**原样**投递 3 次（间隔 100ms） | bookings 只变 paid 一次；payments 的 `transaction_id` 唯一约束生效，第 2、3 次命中幂等表直接返回 200（响应体 `{"code":"SUCCESS"}`，否则微信会一直重试）；后续动作（通知、统计、发券）只执行一次 |
| N2 | 两条**不同通知 id** 但同一 transaction_id 的回调 | 结果同 N1（幂等键必须是 transaction_id，不是微信通知 id） |
| N3 | 回调先于「用户点支付按钮的响应」到达（先回调后返回） | 订单最终一致为 paid；前端轮询查单兜底生效 |
| N4 | 回调处理中途后端崩溃（kill -9 后重启，回调重投） | 事务内完成状态变更，重投后无副作用；无「半支付」状态 |

### 3.3 支付成功 vs 用户取消并发

| # | 操作 | 断言 |
|---|---|---|
| C1 | 回调置 paid 与用户 cancel 请求**几乎同时**到达（用 Go goroutine + `sync.WaitGroup` 精确对撞，或 hey 双端点并发压） | 两者只有一个胜出：若 paid 先提交，cancel 返回 409；若 cancelled 先提交，回调处理发现状态已 cancelled → 不改状态、记录「需人工/自动退款」事件；DB 最终只有一种终态，`-race` 下无 data race |
| C2 | 同 C1，但 DB 层观察 | 状态变更是单事务原子操作（`UPDATE ... WHERE status='pending'` 行锁/条件更新），不允许「读-改-写」三步非原子 |
| C3 | 超时关单任务与支付回调并发（回调恰好在关单事务提交前一毫秒到达） | 关单用条件更新 `WHERE status='pending' AND pay_status='unpaid'`；回调到时发现已 closed → 走退款/冲正分支并有明确记录 |
| C4 | 重复点击「立即支付」按钮（前端抖动） | 前端防抖 + 后端 D1 幂等兜底，只产生一笔待支付 |

并发测试执行方式：全部写成 Go 集成测试（真 Postgres，testcontainers 或本地 docker），`go test -race`；回调重放用 `hey -n 50 -c 10` 打 notify 端点断言仍只置 paid 一次。

---

## 4. 金额与对账

### 4.1 分/元换算

- **全链路只存分（int）**：DB `amount_total INT`（或 bigint）、微信 `amount.total` 单位分、`out_trade_no` 无金额参与。元→分只发生在**前端展示/输入**层，且用整数运算（`(price / 100)` + `(price % 100)` 格式化），**禁止 float**。
- 测试断言：
  - 0.01 元 → total=1；888.00 元 → total=88800；99.99 元 → total=9999（浮点陷阱：`99.99*100` 在 JS=9998.999…，必须 `Math.round(99.99*100)` 或字符串分转）。
  - 输入侧 fuzz：`0.1+0.2`、`19.9`、`1234.567` 等边界值展示无精度丢失。
  - total_price 为 0（互勉单）**不允许进入支付流程**（断言：mockpay 收到 total=0 的 Prepay 直接报参数错误；后端在 Prepay 前拦截）。

### 4.2 金额篡改

| # | 操作 | 断言 |
|---|---|---|
| A1 | 前端提交带 `amount` 字段（后端根本不该读） | 后端金额**只取自 bookings.total_price**，请求中的金额字段被忽略；用例证明改包不影响实际下单金额 |
| A2 | 回调中解出的 amount.total ≠ 下单金额 | 拒单 + 告警事件 + 不置 paid；原始报文入库留证 |
| A3 | 回调中 mchid/appid 与配置不符 | 拒单（防串号攻击） |
| A4 | 查单（QueryOrder）返回金额与 DB 不符 | 以微信侧为准触发差异工单（对账逻辑复用） |

### 4.3 对账文件模拟与差异检测

- mockpay 的 `DownloadBill` 生成与微信真实账单同构的 CSV（表头与「下载账单」接口格式一致：`记账时间,微信支付单号,商户订单号,业务类型,交易状态,支付金额,申请退款金额,费率备注,…`，含 `交易状态`：SUCCESS/REFUND/REVOKED，以及汇总行）。
- 对账 job（每日 T+1）逻辑：DB payments ↔ 账单逐单比对。**必须设计的用例**：

| 用例 | 构造方式 | 期望 |
|---|---|---|
| 全对平 | 正常流 | 差异报告为空 |
| 账单有、DB 无 | mockpay 账单塞一笔 DB 不存在的单（模拟丢单/乱序回调） | 差异报告出现「账单有 DB 无」条目 |
| DB 有、账单无 | DB 手工插入已支付记录 | 差异报告出现「DB 有账单无」条目（本地单未掉账，多为掉单 bug） |
| 金额不符 | mockpay 返回金额被改成 990 分 | 差异报告金额差异条目 |
| 重复记账行 | 账单含 `ACQUIRE_WITH...` 重复行 | 对账 job 按 (商户订单号,交易状态) 去重后再比 |
| 退款行 | 账单含 REFUND 行 | 与 refunds 表比对，部分退款累计 ≤ 支付金额 |
| 时间边界 | 支付 23:59:59 的单落在次日账单 | 对账窗口取「支付日 + 次日」两日账单合并比对 |

- 差异检测的验收标准：任意一行差异都能在报告中定位到 out_trade_no、两侧金额、差异类型，且告警可送达（现阶段写日志/通知表即可）。

---

## 5. 真测路径（拿到测试商户号之后）

### 5.1 前置条件

1. 注册微信支付商户号（个体工商户主体即可，免费，绑定小程序 appid，开通 JSAPI 支付）；
2. 商户平台配置：APIv3 密钥、API 证书（或微信支付公钥）、回调域名（HTTPS）；
3. 后端 `wechatpay-go` 生产实现接通，`PAY_MOCK=0`。

### 5.2 0.01 元最小验证集（金额全部 0.01 元，除非用例本身要求多金额）

| # | 用例 | 验证点 |
|---|---|---|
| R1 | 下单 → 真机拉起收银台 → 实付 0.01 → 回调 → 订单 paid | 全链路首次打穿 |
| R2 | 支付成功后**屏蔽回调**（回调端点临时返回 500），观察微信重试 + 主动查单补偿是否把订单捞回 paid | 掉单自愈 |
| R3 | 下单后不支付，直接关页面 → 订单仍 pending → 超时关单生效 | 关单路径 |
| R4 | 快速连点「立即支付」 | 真实幂等（同一 prepay_id，只扣一次款） |
| R5 | R1 的订单退款 0.01 → `REFUND.SUCCESS` 回调 → 订单 refunded → **微信零钱确认到账** | 退款闭环 + 真实资金回退 |
| R6 | 调 `/v3/security/echo` 回显测试 | 签名/验签/加解密与微信侧互通确认 |
| R7 | 下载 T+1 真实账单 → 对账 job 跑真实数据 | 差异=0，字段映射正确 |
| R8 | iOS 与 Android 各跑一遍 R1（不同机型收银台表现） | 端侧兼容 |

### 5.3 人力与耗时预估

| 事项 | 人数 | 耗时 |
|---|---|---|
| 商户号注册+审核+配置（若尚无） | 1（行政/老板侧，非开发阻塞，可并行） | 1~5 天等待（审批周期，不占开发工时） |
| 证书/APIv3 密钥配置 + wechatpay-go 接通 + R6 | 1 开发 | 0.5 天 |
| R1~R8 用例执行 | 1 开发 + 1 微信号（付款人=开发者本人即可）；R8 建议 iOS/Android 各 1 人 | 0.5~1 天 |
| 退款到账观察 | 0（异步等待） | 零钱原路退回一般数分钟~24h |
| **合计** | **1~2 人** | **1~2 个工作日**（不含商户号审批等待） |

---

## 6. 上线前必须人工确认清单（不可自动化，逐项打勾）

- [ ] 真实微信 App + 真机（iOS、Android 各至少 1 台）完成真实支付各 1 单，商户平台后台可见交易记录
- [ ] 真实退款到账：付款人微信「零钱/账单明细」确认收到退款（不是只看 DB 状态）
- [ ] 商户平台交易流水与 DB payments 总额逐日一致（上线前至少 3 天无差异）
- [ ] 回调地址为生产 HTTPS 域名、证书链有效、已与商户平台配置一致；`PAY_MOCK` 在生产为 0 且 `mock/confirm` 返回 404
- [ ] 生产密钥（APIv3 密钥、商户私钥）不在代码库/前端包中，来源为密管或环境注入
- [ ] 对账 job 已按日跑通且告警通道可达（人为制造一条差异验证告警能收到）
- [ ] 回调失败重试与查单补偿任务的日志监控就位
- [ ] 灰度方案确认：先内部 0.01 元订单，再放开小额真实订单
- [ ] 消费者投诉 2.0 / 退款客服路径有人负责（平台要求商户在时限内响应）

---

## 7. 测试手段 × 能证明什么 × 不能证明什么

| 测试手段 | 能证明 | 不能证明 |
|---|---|---|
| mockpay 下单（§2.2） | 参数组装正确、金额取数正确（来自 bookings）、下单幂等、负向参数校验 | 微信真实接收并认可该下单（appid/mchid 绑定关系、字段在真实风控下的合法性） |
| 前端模拟收银台（§2.4-A） | 支付流程交互、订单状态展示、loading/异常分支 | 真实收银台拉起、真机表现、真实用户取消/密码错误等端侧行为 |
| mock 签名回调（§2.3） | 验签代码、AES-GCM 解密、幂等表、状态机、通知链路 100% 正确 | 微信签名细节的兼容性（如 WECHATPAY/SIGNTEST/ 探测流量、公钥 ID vs 平台证书序列号的真实取值差异——但有 `/v3/security/echo` R6 兜底） |
| 并发/重放用例（§3） | 状态机并发安全、回调幂等、cancel 与 paid 对撞收敛、`-race` 干净 | 微信真实的重试节奏与投递顺序（真实环境只能靠 R2 观察一次） |
| 金额/对账用例（§4） | 分元精度、防篡改、对账比对逻辑、差异报告 | 真实账单的字段与边界（日切、跨天单、手续费列）——R7 兜底 |
| mock 退款 | 退款状态机、累计退款不超额 | 真实资金退回到账、到账时效、银行渠道差异——R5 兜底 |
| 0.01 元真测（§5） | 端到端资金流真实成立、证书/密钥/回调域名配置正确、真实账单可对、真实退款到账 | 高金额、高并发下的真实风控表现；费率/结算周期细节（上线后 T+1 对账持续监控） |

**一句话**：mock 阶段证明「我们的代码是对的」，真测阶段证明「我们的代码和微信是对得上的」；两边缺一不可，前者占工作量 80%，后者占风险 80%。

---

## 附：文档引用

- 沙箱升级公告：developers.weixin.qq.com/community/pay/doc/0002c437a1c6f00ba87ef1a1a5ec01（2022-09-02）
- 签名验签工具：pay.weixin.qq.com/doc/v3/merchant/4012365352
- 商户签名验签／加解密测试 `/v3/security/echo`：pay.weixin.qq.com/doc/v3/merchant/4014551946
- JSAPI 支付产品介绍：pay.weixin.qq.com/doc/v3/merchant/4012062524
- 社区佐证帖：2023-11-15（JSAPI 无沙箱）、2024-02-28、2024-12-15、2025-09-14、2025-11-21；GitHub wechatpay-apiv3/wechatpay-go#172
- 项目内参照：`doc/review/booking-state-machine.md`、`doc/review/booking-review-final.md`（M-9）
