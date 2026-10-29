# 微信支付 v3 接入调研与实施清单（漫展约拍小程序）

- 调研对象：`comic` Uniapp 小程序 + Go/Gin/PostgreSQL/Redis 后端
- 范围：普通商户直连、小程序支付（与 JSAPI 共用下单接口）、订单状态整合、退款、对账、无正式商户号测试
- 文档依据：微信支付官方文档，本次逐项抓取核对；不包含生产代码改动
- 结论先行：**采用普通商户直连的小程序支付，预约履约状态与资金状态分离；固定价全额预付，支付成功由服务端自动确认预约；退款、对账、通知去重均落 PostgreSQL，Redis 只做加速不做资金真相。**

## 1. 项目现状与关键决策

### 1.1 已具备、待补齐

已有微信登录链路：`wx.login → code → jscode2session → openid`，见 `server/internal/service/wechat.go:14-18,48-90`、`server/internal/service/auth_service.go:96-115`。配置已有 `WX_APPID/WX_APP_SECRET`，见 `server/internal/config/config.go:22-24`。现有手机号登录用户与微信登录用户的合并策略仍需单独确认，支付只取**当前登录用户在同一小程序 AppID 下的 openid**。

已有预约状态机：`pending → confirmed → completed`，`pending/confirmed → cancelled`，见 `server/internal/service/booking_service.go:75-84,317-370`。固定价/互勉/面议及价格快照见 `server/internal/service/booking_service.go:204-245`。当前尚无支付表、支付 API、微信通知、退款和账单任务。

### 1.2 接入边界

| 项目 | 建议 |
|---|---|
| 收款主体 | 一个已完成认证的小程序主体 + 同主体普通商户号；平台作为商户收款方 |
| 支付产品 | 小程序支付，调用 `/v3/pay/transactions/jsapi`；不是 H5、Native、APP 或合单支付 |
| 收费范围 | 固定价、已报价且 coser 接受的面议单；互勉单不创建微信支付单 |
| 首期结算 | 平台商户统一收款；摄影师提现/分账是另一项产品，不在普通支付里隐式实现 |
| 支付比例 | 首期固定为全额预付；尾款、定金比例、违约扣款、分账需另立资金规则后再扩展 |
| 金额单位 | 新支付表与微信接口统一用“分”的整数；现有 `bookings.total_price INTEGER` 单位须用真实样本确认，只在支付边界显式转换一次 |
| 密钥模式 | 新接入使用微信支付公钥 + 公钥 ID 验签；商户 API 私钥签名；APIv3 密钥解密通知 |
| 服务实现 | 优先按官方 Go `WXPayUtility` 示例/标准库实现并锁定版本；官方服务端 SDK 页面已注明停止更新维护，见 [SDK 概述](https://pay.weixin.qq.com/doc/v3/merchant/4012076498) |

**必须先拍板的业务点：**平台是否是实际收款及售后主体、摄影师如何获得收入、是否全额预付、各取消时点的退款比例、现价格单位、小程序与商户主体是否一致。主体和结算未定时，可先完成后端与仿真测试，不上真实交易入口。

## 2. 资质、开户与配置清单

### 2.1 开通顺序

- [ ] 准备已认证微信小程序及 `AppID/AppSecret`；服务类目覆盖实际摄影服务，交易类小程序按公众平台规则接入订单发货管理。
- [ ] 准备可开户主体：个体工商户、企业、事业单位、政府机关、社会组织；普通商户开户资料及结算账户。
- [ ] 已有商户号：商户平台申请小程序/JSAPI 产品权限，发起 AppID 绑定授权，公众平台管理员确认。
- [ ] 无商户号：按接入准备页同步申请商户号和支付权限，再完成 AppID 绑定。
- [ ] 获取 `mchid`，下载/保管商户 API 证书、私钥 `apiclient_key.pem`、证书序列号。
- [ ] 设置 32 字符 APIv3 密钥；新接入下载微信支付公钥并记录 `PUB_KEY_ID_...` 公钥 ID。
- [ ] 准备公网 HTTPS API 域名，固定支付通知与退款通知 URL；回调路由不能挂现有用户 `AuthRequired`，安全依据是微信签名。
- [ ] 配置服务器出网、DNS、TLS、时间同步；保存每次 API 响应的 `Request-ID` 供排查。
- [ ] 完成隐私信息展示、服务/退款规则、订单发货管理配置及真实小额验收。

官方准入与绑定见 [产品介绍](https://pay.weixin.qq.com/doc/v3/merchant/4012791894)、[开发接入准备](https://pay.weixin.qq.com/doc/v3/merchant/4015459512)、[商户接入准备](https://pay.weixin.qq.com/doc/v3/merchant/4012071573)。**个人主体不在普通商户支持范围；小程序 web-view 内不能改用 JSAPI/H5 支付绕过。**

### 2.2 服务端配置（只列名称，密钥由用户保管）

```dotenv
WX_PAY_ENABLED=false
WX_APPID=<小程序AppID>
WX_APP_SECRET=<仅服务端>
WX_PAY_MCHID=<普通商户号>
WX_PAY_MCH_CERT_SERIAL_NO=<商户API证书序列号>
WX_PAY_MCH_PRIVATE_KEY_PATH=<私钥文件挂载路径>
WX_PAY_API_V3_KEY=<32字符APIv3密钥>
WX_PAY_PUBLIC_KEY_PATH=<微信支付公钥文件路径>
WX_PAY_PUBLIC_KEY_ID=<PUB_KEY_ID_...>
WX_PAY_NOTIFY_URL=https://api.example.com/api/v1/payments/wechat/notify
WX_PAY_REFUND_NOTIFY_URL=https://api.example.com/api/v1/payments/wechat/refund-notify
WX_PAY_API_BASE=https://api.mch.weixin.qq.com
```

私钥、AppSecret、APIv3 密钥仅服务端挂载，不进小程序包、Git、日志、管理后台响应或普通业务表。启用真实支付时启动校验所有配置及公钥/私钥可加载性；仿真模式必须在响应和日志中明显标识，生产环境禁用仿真开关。

## 3. 数据模型与订单状态机

### 3.1 不改动原履约枚举，新增独立资金模型

`bookings.status` 继续表示履约：`pending/confirmed/completed/cancelled`。新增支付主单、退款单、通知表和可靠事件表；前端展示由两者组合，绝不以 `cancelled` 推断“已退款”。

| 表 | 核心字段及约束 |
|---|---|
| `payment_orders` | `id, booking_id, payer_user_id, out_trade_no UNIQUE, appid, mchid, amount_fen, description_snapshot, notify_url, status(NOTPAY/SUCCESS/CLOSED), transaction_id UNIQUE可空, payer_openid, success_time, raw_trade_state, created_at, updated_at`；金额创建后不可改 |
| `payment_refunds` | `id, payment_order_id, out_refund_no UNIQUE, refund_id UNIQUE可空, amount_fen, reason, status(PROCESSING/SUCCESS/ABNORMAL/CLOSED), operator_admin_id, created_at, success_time, updated_at`；按支付单累计成功退款额校验 |
| `wechat_notifications` | `notification_id UNIQUE, event_type, resource_type, headers_metadata, ciphertext_payload, processed_at, processing_result`；先验签解密，再入库/消费 |
| `payment_outbox` | `event_id UNIQUE, aggregate_type/id, event_type, payload, published_at, retry_count`；与资金状态同事务写入，异步发通知/发货管理任务 |
| `reconciliation_runs/items` | 账单日期、类型、文件哈希、导入状态、本地/微信单号、差异类型、金额、处理人、处理结论；同一文件哈希可幂等重导 |

约束建议：一个 booking 同时最多一个有效支付主单；同一 `out_trade_no` 重试下单，不因重复点击生成新号。价格或付款人改变时先确认旧单未支付并关单，再新建支付意图。支付单、退款单只通过条件更新前进，更新需带当前状态，禁止无条件覆盖。

### 3.2 推荐业务流程

| 类型 | 创建预约 | 支付入口 | 支付成功 | 完成/取消 |
|---|---|---|---|---|
| 固定价 | 原流程创建 `pending`，冻结价格和档期 | 立即创建/复用 NOTPAY 支付单 | 服务端事务内 `pending→confirmed`、记支付成功、写 outbox | 仅 SUCCESS 可完成；已付取消进入退款流 |
| 面议 | 原报价流程，coser 接受后现有逻辑为 `confirmed` 且价格 agreed | 仅 agreed 后允许支付 | 保持 confirmed，记录已付；不得再改价 | 同上 |
| 互勉 | 原流程 | 无支付按钮 | 不适用 | 不创建退款单 |

支付成功后向双方发送“支付成功/预约已确认”通知；通知异步发送，不回滚已落库支付。`completed` 的服务端守卫增加“固定价/面议必须支付成功”，管理端同样执行，不能只靠前端隐藏按钮。

未支付超时：业务可设置 30 分钟支付窗口（具体时长由用户定），定时查单仍为 NOTPAY 后调用关单，再将 `pending→cancelled` 释放档期。`prepay_id` 官方有效期 2 小时；`time_expire` 不是自动关单依据，默认 7 天未付由微信关单，因此业务仍需自己的查单+关单任务。

**竞态处理：**关单/取消与 SUCCESS 回调并发时锁定支付主单和 booking。若款已成功而预约已取消，保留支付成功事实，生成“已付已取消待退款”异常/自动全额退款任务；不得丢弃回调、把已付款改成未付款或静默恢复预约。

### 3.3 前端组合展示

| booking / 资金事实 | 用户文案 | 可执行动作 |
|---|---|---|
| pending + NOTPAY | 待支付 | 继续支付、取消 |
| confirmed + SUCCESS | 已支付，待拍摄 | 查看、联系、按规则取消 |
| completed + SUCCESS | 已完成 | 评价、售后 |
| cancelled + SUCCESS + 无退款单 | 已取消，退款处理中 | 查看退款进度 |
| 任意 + PROCESSING | 退款处理中 | 查看进度 |
| 任意 + 部分退款成功 | 已部分退款 | 查看明细 |
| cancelled + 全额退款成功 | 已退款 | 查看凭证 |
| 任意 + ABNORMAL/对账差异 | 退款/账务处理中 | 联系客服 |

## 4. 下单、二次签名与支付确认

### 4.1 后端接口草案

| 方法/路径 | 鉴权与作用 |
|---|---|
| `POST /api/v1/bookings/:id/payments` | 当前 coser；校验金额、openid、状态，创建/复用支付单，微信下单，返回二次签名参数 |
| `GET /api/v1/payments/:outTradeNo` | 仅付款用户；返回本地资金状态及退款摘要，不返 openid/密钥 |
| `POST /api/v1/payments/wechat/notify` | 公网原始报文；验签、解密、核对、幂等入账 |
| `POST /api/v1/payments/wechat/refund-notify` | 公网原始报文；退款通知处理 |
| `POST /api/admin/v1/payments/:outTradeNo/refunds` | 管理端鉴权+审计；按剩余可退金额创建退款 |
| `GET /api/admin/v1/reconciliation/exceptions` | 管理端查看差异、处理记录，不直接改资金事实 |

### 4.2 服务端下单

接口：`POST https://api.mch.weixin.qq.com/v3/pay/transactions/jsapi`。必填：

```json
{
  "appid": "<小程序AppID>",
  "mchid": "<商户号>",
  "description": "<漫展/摄影师/套餐快照，1~127字节>",
  "out_trade_no": "PAY_<6~32字符内>",
  "notify_url": "https://api.example.com/api/v1/payments/wechat/notify",
  "amount": {"total": 39900, "currency": "CNY"},
  "payer": {"openid": "<当前登录用户在该AppID下的openid>"},
  "time_expire": "<可选RFC3339>"
}
```

- 后端以 token 用户、booking 归属和价格快照为准，忽略客户端传入的 `coserId/openid/amount`。
- `out_trade_no` 同商户号唯一，6–32 位，字符按官方限制；`amount.total` 是正整数分。
- openid 必须属于下单 AppID，且实际付款微信用户必须一致；AppID 必须与 mchid 绑定。
- HTTP `Authorization: WECHATPAY2-SHA256-RSA2048 ...`；签名串为 `METHOD\n完整URL含query\ntimestamp\nnonce\n原始body\n`，使用商户私钥 SHA256-RSA 并 Base64。
- 校验 HTTP 状态、业务错误码和**微信支付应答签名**，记录 `Request-ID`；仅成功时保存 prepay 信息供本次调起，不暴露商户私钥。
- 网络超时不等于下单失败：按 `out_trade_no` 查单确认，不盲目换号重下。prepay_id 过期时原参数重新下单。

### 4.3 返回小程序的二次签名

```json
{
  "timeStamp": "1710000000",
  "nonceStr": "<每次安全随机>",
  "package": "prepay_id=<微信返回的prepay_id>",
  "signType": "RSA",
  "paySign": "<服务端签名>"
}
```

`paySign` 的签名串为 `appId\ntimeStamp\nnonceStr\npackage\n`；`timeStamp` 为秒级字符串。`wx.requestPayment` 的 `success/fail/cancel` 都只驱动 UI 查询：success 不能入账，cancel 保持未支付，fail 后仍查后端；未支付继续使用同一支付单。小程序与 JSAPI 共用下单接口，但 prepay_id 只能在同场景使用。

官方：[下单](https://pay.weixin.qq.com/doc/v3/merchant/4012791897)、[二次签名与调起](https://pay.weixin.qq.com/doc/v3/merchant/4012791898)、[开发指引](https://pay.weixin.qq.com/doc/v3/merchant/4012791911)、[常见问题](https://pay.weixin.qq.com/doc/v3/merchant/4012791910)。

## 5. 回调验签、解密、幂等与补偿

### 5.1 固定处理顺序

1. Gin 使用原始字节 body；禁止先 JSON 解码再序列化用于验签，禁止框架改写 body。
2. 读取 `Wechatpay-Serial/Wechatpay-Signature/Wechatpay-Timestamp/Wechatpay-Nonce`。按 serial 选择微信支付公钥或存量平台证书；公钥 ID 形如 `PUB_KEY_ID_...`。
3. 对 `Wechatpay-Timestamp\nWechatpay-Nonce\n原始body\n` 做 RSA-SHA256 验签；检查时间戳偏差，失败返回非 2xx。
4. 校验外层 `id/event_type/resource_type/resource`；确认 `resource.algorithm=AEAD_AES_256_GCM`，用 32 字节 APIv3 密钥对 `resource.ciphertext` 做 AES-256-GCM 解密，认证标签随密文，使用 `nonce/associated_data`。
5. 支付通知只接受 `TRANSACTION.SUCCESS` 入账；解密后逐项核对 `appid/mchid/out_trade_no`、本地金额、`trade_state=SUCCESS`、`payer.openid`、`amount.total/payer_total`、`transaction_id/success_time`。
6. 在**单个数据库事务**中锁支付主单和 booking，按通知 `id` 唯一去重，保存资金事实、状态条件迁移及 outbox；不在这里做慢通知、查微信或发货 HTTP 调用。
7. 提交后 5 秒内返回 HTTP 200/204 空包体。重复且已处理通知也返回成功。验签/解密/归属金额不一致不更新资金状态并返回非 2xx；可重试 DB 故障返回非 2xx。

失败/超时按官方节奏重试：`15s/15s/30s/3m/10m/20m/30m/30m/30m/60m/3h/3h/3h/6h/6h`，最多 15 次。官方可能用 `WECHATPAY/SIGNTEST/` 错误签名探测，验签失败必须非 2xx；不能只“记录后 200”。

### 5.2 通知丢失与乱序

- 主动查单：`GET /v3/pay/transactions/out-trade-no/{out_trade_no}?mchid=...`；有 transaction_id 时可走 `/v3/pay/transactions/id/{transaction_id}`。
- 用户返回页查询 + 短周期任务补偿；官方参考间隔为 `5s/30s/1m/3m/5m/10m/30m`，任务节奏可按量调整。
- 同一事务执行“验签后的查单响应 → 状态核对 → 幂等入账”，复用通知的领域处理函数。
- 终态不回退；SUCCESS 与 CLOSED 冲突以真实查单为准并记异常，退款通知只改变退款子状态，不把支付成功改回未支付。
- 超过补偿窗口仍不一致进入人工差异队列，T+1 对账兜底。

官方：[支付通知](https://pay.weixin.qq.com/doc/v3/merchant/4012791902)、[验签](https://pay.weixin.qq.com/doc/v3/merchant/4013053249)、[AES-GCM 解密](https://pay.weixin.qq.com/doc/v3/merchant/4012071382)、[按商户单号查单](https://pay.weixin.qq.com/doc/v3/merchant/4012791900)、[关单](https://pay.weixin.qq.com/doc/v3/merchant/4012791901)。

## 6. 退款设计

### 6.1 触发与金额

- 用户/摄影师/管理员取消已支付预约时，统一进入交易服务；现有 `PUT /bookings/:id/status` 与 `AdminUpdateStatus` 的取消分支不能绕开该服务。
- 后端按已支付金额减去已受理/成功退款金额计算可退余额，校验退款政策、操作者权限、订单归属；金额由服务端计算或严格复核。
- 每笔退款生成稳定 `out_refund_no`；同一业务操作重试必须复用原号。不同部分退款使用新号。
- 接口 `POST /v3/refund/domestic/refunds`：`transaction_id/out_trade_no` 二选一，必传 `out_refund_no`、`amount.refund/total/currency`，选填 `reason/notify_url`。
- 成功受理只置 `PROCESSING`，以退款通知或查退款为最终依据；不直接展示“退款成功”。
- 支持部分退款；累计不得超过实付。官方一笔订单最多 50 次部分退款，支付成功一年内可退；不同部分退款建议间隔 1 分钟。

### 6.2 查询、通知和异常

- 查退款：`GET /v3/refund/domestic/refunds/{out_refund_no}`。官方推荐每 1 分钟查询；超过 5 分钟仍处理中可按 `5/10/20/30` 分钟退避。
- 退款通知事件：`REFUND.SUCCESS/ABNORMAL/CLOSED`，沿用支付通知的验签、AES-GCM 解密、幂等、5 秒应答规则。
- 解密核对商户号、原支付单号、`out_refund_no/refund_id/refund_status`、退款金额。SUCCESS/CLOSED 为业务处理终点，ABNORMAL 进入人工/异常退款流程；不盲目再次扣款式重试。
- `REFUND.ABNORMAL` 不等于可换号重退；先查微信与本地累计退款，再由管理员按官方异常退款渠道处理。
- 保留退款申请、微信受理、最终结果、操作员和通知 ID 的审计链；booking 的履约取消与资金退款结果独立展示。

官方：[申请退款](https://pay.weixin.qq.com/doc/v3/merchant/4012791903)、[查退款](https://pay.weixin.qq.com/doc/v3/merchant/4012791904)、[退款通知](https://pay.weixin.qq.com/doc/v3/merchant/4012791906)。

## 7. T+1 对账与差错处理

### 7.1 每日任务（Asia/Shanghai）

- [ ] 每日 10:00 后申请昨日交易账单：`GET /v3/bill/tradebill?bill_date=yyyy-MM-DD&bill_type=ALL&tar_type=GZIP`。
- [ ] 同期申请资金账单：`GET /v3/bill/fundflowbill`，分别按 `BASIC/OPERATION/FEES` 需要的账户类型导入；首日至少交易账单必须上线。
- [ ] 记录 `download_url/hash_type/hash_value`；下载链接 5 分钟有效，过期重新申请。下载请求带 v3 Authorization，但文件响应按官方规则不验签，改核文件 hash。
- [ ] 原始账单加密归档、解析结果版本化，以日期+账单类型+hash 幂等重导；接口只支持近三个月，不能依赖事后无限补拉。
- [ ] 账单金额单位是元，以十进制定点解析为分，禁止浮点运算；账单字段前导反引号按文本处理。

### 7.2 对账维度与动作

| 差异 | 判定 | 处理 |
|---|---|---|
| 微信 SUCCESS、本地 NOTPAY | 通知/补偿遗漏 | 立即查单验签补入账，再触发后续 outbox |
| 本地 SUCCESS、账单无记录 | 本地归属或伪造风险 | 查单、核对 mchid/AppID/通知证据，冻结自动后续业务 |
| 金额/币种/openid 不符 | 高风险差异 | 不改写本地原始金额，进入人工队列并告警 |
| 退款状态或金额不符 | 通知遗漏/重复 | 按 out_refund_no 查退款后修正 |
| 本地成功、微信 CLOSED 等终态冲突 | 状态异常 | 以查证事实修正并保留审计，不静默删单 |
| 手续费/资金流水差异 | 结算差异 | 用资金账单核手续费、净流入和退款，生成财务差异单 |

任何自动修复都限定为“拉取已验签事实并幂等重放”，不能由对账任务凭账单文本直接任意改库或自动退款。每日产出交易笔数、支付/退款金额、单边账、待处理时长和结果报表。

官方：[交易账单](https://pay.weixin.qq.com/doc/v3/merchant/4012791907)、[资金账单](https://pay.weixin.qq.com/doc/v3/merchant/4012791908)、[下载账单](https://pay.weixin.qq.com/doc/v3/merchant/4012791909)。

## 8. 无正式商户号时的测试方案

### 8.1 立即可做：真实密码学 + 仿真网关

- [ ] 抽象 `WechatPayGateway`（下单/查单/关单/退款/查退款/账单），本地 fake 与真实实现共用领域服务。
- [ ] 生成仅供测试的商户 RSA 密钥对、微信支付 RSA 公钥对和 32 字节 APIv3 测试密钥；用同一生产代码路径签名、验签、AES-GCM 加解密。
- [ ] fake 返回 prepay_id，二次签名做单元测试；小程序按钮在 `APP_ENV=development` 下走明确的模拟支付结果，不改生产支付状态。
- [ ] 本地构造真实格式支付/退款通知，通过公网隧道仅测试 HTTP 原始 body、响应码和重试；隧道流量不冒充微信生产回调。
- [ ] 用 PostgreSQL 测试库验证事务、唯一索引和并发，不仅 mock repository。
- [ ] 微信开发者工具用于 UI、登录与参数拼装联调；其模拟能力不替代真实微信支付验收集。

### 8.2 必测用例

- [ ] 双击支付只产生一个 out_trade_no；prepay 过期原单重下；价格改变不能复用旧金额。
- [ ] A 用户不能给 B 的 booking 付款；mock openid、openid 属另一 AppID、金额篡改均被服务端拒绝。
- [ ] 正确通知成功入账；错签名、错 serial、错密钥、改一字节原始 body、改金额/mchid/AppID/openid 均不入账。
- [ ] 同一通知并发 20 次只入账一次；成功后重复通知持续 200；DB 暂时失败能重试。
- [ ] SUCCESS 重复、延迟、与查单并发；CLOSED/SUCCESS 冲突；关单与支付同时发生；已付已取消生成退款异常。
- [ ] 前端 success 但后端无通知时不直接成功；cancel 后仍为未支付；通知丢失由查单补偿。
- [ ] 全额/多次部分退款、超额退款、同 out_refund_no 重试、ABNORMAL/CLOSED、通知先于管理页面查询。
- [ ] 固定价未付不能 completed；互勉无支付；面议未 agreed 不能付款；管理端取消也不能绕过退款。
- [ ] T+1 微信单边、本地单边、金额差、退款差、重复导入、hash 不符、download_url 过期。

### 8.3 有商户号后的真实验收

用测试商户号/隔离配置和已绑定测试小程序完成 `0.01 元` 级真实支付、取消、全额/部分退款、通知重试、T+1 账单下载；核对商户平台流水、数据库、退款到账及账单一致。真实配置通过独立部署环境注入，测试通过才开启生产入口。

## 9. 合规、安全与运行风控

- [ ] 商户平台展示的收款方、实际服务提供者、合同/退款规则保持一致；平台收款与摄影师结算需明确业务方案。
- [ ] 交易类小程序接入订单发货管理；预约服务以拍摄档期、完成状态映射履约节点，发货状态更新用 outbox 重试并保留微信受理结果。
- [ ] 页面明示总价、服务内容、时长、档期、取消和退款规则；提供支付/退款明细及售后入口。
- [ ] openid、交易号、付款人信息按最小化收集和权限隔离；导出/日志脱敏，不在通知 URL 携带用户隐私。
- [ ] 私钥/APIv3 密钥/AppSecret 使用受限文件或密钥管理，定期轮换；证书轮换期验证新旧配置，私钥丢失按官方流程重办。
- [ ] 微信 API 客户端设置连接/总超时、有限重试和熔断；查询类可退避重试，写入类先查原单，不无限并发重试。
- [ ] 告警：验签失败突增、通知积压、NOTPAY 超时、已付已取消、退款 ABNORMAL、对账单边账、账单 hash 失败、商户 API Request-ID 错误簇。
- [ ] 资金操作审计不可由普通管理员删除；退款需管理端权限和二次确认，可配高额复核；每日检查成功退款累计不超过实付。
- [ ] 上线后持续检查公众平台站内信、服务类目及发货管理要求；`no permission/banned` 按官方 FAQ 排查，不能换支付产品绕过。

## 10. 实施顺序与验收门槛

1. **规则冻结**：确认收款/结算主体、全额预付、退款政策、金额单位、支付超时；产品画出组合状态。
2. **资金底座**：新增支付/退款/通知/outbox/对账表及唯一约束；完成 fake 网关和并发测试。
3. **支付链路**：真实下单、二次签名、查单关单、通知验签解密；接入现有固定价与面议流程。
4. **售后链路**：用户与管理端取消统一走交易服务，退款通知/查询、异常处理和审计。
5. **履约与对账**：支付后发货管理 outbox、每日账单、差异队列和财务报表。
6. **真实验收**：第 8.2 节全部自动化通过 + 第 8.3 节真实小额及 T+1 对账通过 + 密钥/权限/告警检查后，才将 `WX_PAY_ENABLED` 切到生产 true。

## 11. 官方文档索引

| 主题 | 官方链接 |
|---|---|
| 小程序支付产品/准入/结算 | https://pay.weixin.qq.com/doc/v3/merchant/4012791894 |
| 开发准备、AppID 绑定 | https://pay.weixin.qq.com/doc/v3/merchant/4015459512 |
| 商户主体与接入准备 | https://pay.weixin.qq.com/doc/v3/merchant/4012071573 |
| JSAPI/小程序下单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791897 |
| 小程序二次签名 | https://pay.weixin.qq.com/doc/v3/merchant/4012791898 |
| 开发指引/状态/有效期 | https://pay.weixin.qq.com/doc/v3/merchant/4012791911 |
| 支付结果与后续检查 | https://pay.weixin.qq.com/doc/v3/merchant/4012075249 |
| 支付通知 | https://pay.weixin.qq.com/doc/v3/merchant/4012791902 |
| 按微信订单号查单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791899 |
| 按商户订单号查单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791900 |
| 关单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791901 |
| 申请退款 | https://pay.weixin.qq.com/doc/v3/merchant/4012791903 |
| 查询退款 | https://pay.weixin.qq.com/doc/v3/merchant/4012791904 |
| 退款通知 | https://pay.weixin.qq.com/doc/v3/merchant/4012791906 |
| 交易账单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791907 |
| 资金账单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791908 |
| 下载账单 | https://pay.weixin.qq.com/doc/v3/merchant/4012791909 |
| 商户请求签名 | https://pay.weixin.qq.com/doc/v3/merchant/4012365337 |
| 证书密钥与签名验签 | https://pay.weixin.qq.com/doc/v3/merchant/4012365342 |
| 商户 API 证书 | https://pay.weixin.qq.com/doc/v3/merchant/4012072428 |
| APIv3 密钥 | https://pay.weixin.qq.com/doc/v3/merchant/4012072195 |
| 证书密钥配置概览 | https://pay.weixin.qq.com/doc/v3/merchant/4013070756 |
| AES-256-GCM 解密 | https://pay.weixin.qq.com/doc/v3/merchant/4012071382 |
| 微信支付应答/回调验签 | https://pay.weixin.qq.com/doc/v3/merchant/4013053249 |
| SDK 维护状态 | https://pay.weixin.qq.com/doc/v3/merchant/4012076498 |
| 官方 Go WXPayUtility 示例 | https://pay.weixin.qq.com/doc/v3/merchant/4015119334 |
| 小程序支付常见问题 | https://pay.weixin.qq.com/doc/v3/merchant/4012791910 |

## 12. 需要用户提供/确认的信息

| 信息 | 用途 | 当前处理 |
|---|---|---|
| 已认证小程序 AppID、AppSecret、主体类型 | 登录、支付权限、绑定 | 现有环境变量，值待用户配置 |
| 普通商户 mchid、开户主体与小程序主体关系 | 直连收款及权限 | 待提供 |
| 商户 API 证书序列号、私钥文件、APIv3 密钥 | 请求签名、通知解密 | 用户通过安全渠道配置，不写入本文 |
| 微信支付公钥和公钥 ID（或存量平台证书） | 应答/通知验签 | 新接入优先公钥模式 |
| 正式 API 域名、支付/退款通知 URL | 接收回调 | 待提供 |
| 收款及摄影师结算方案 | 决定普通商户还是后续分账/服务商 | **需拍板** |
| 全额/定金、支付超时、取消退款比例 | 状态机及退款金额 | 本方案默认全额、30 分钟待确认 |
| 现 `total_price/quote_price` 单位与样例 | 元到分转换及历史数据 | **需确认** |
| 手机号账号与微信 openid 账号合并规则 | 防止同人重复账号/付款归属 | **需确认** |
| 测试商户号、测试微信号及真实验收窗口 | 真实小额/退款/账单验收 | 无商户号时先仿真 |

---

本清单只形成设计和调研产物，未修改 `comic` 的生产代码、数据库迁移或部署配置。
