# Handoff Prompt · 米拉漫展（comic）

> 用法：新开一个 DSH 会话 / 换模型 / 换人接手时，把下面整段（从「---BEGIN---」到「---END---」）粘进对话框。
> 配套必读：`comic/doc/08-final-handoff.md`（本文档的详版）。

---BEGIN---

你在接手一个**已经在生产跑着**的成熟项目，不是从零开始。先把下面这些记住，动手前不要重新发明轮子。

## 一句话现状

「米拉漫展」——漫展 × 摄影师约拍平台。C 端 uni-app（H5 + 微信小程序）、Go/Gin 后端、Vue3 管理后台、PG + Redis。
`master @ d962446`，`mp-build @ d8ee687`（小程序产物分支）。**代码在跑，别乱重构。**

## 硬环境事实（错了会白折腾半小时）

工作目录：`/vol1/@appshare/fn-deepseek-harness/微信小程序开发`
主仓库：`comic/`

| 项 | 值 |
| --- | --- |
| 容器 | `comic-dev-one`，id `7ce50593505e`，挂载 `comic/ → /app` |
| 端口 | API `8088` · H5 `5173` · admin `5174` · PG `5433` · Redis `6380` |
| 数据库 | `psql -h 127.0.0.1 -p 5433 -U comic -d comic`，密码 `comic123`，**库名是 `comic` 不是 `comic_dev`** |
| 生产 | `https://glxy.haxlock.top`（API `/api`，后台 `/admin/`） |
| 小程序 | appid `wxd46e40692ecd0173` |
| Go | `.tools/go`（1.22.12） |
| 仓库 | `github.com/Maluslock/comic` |

**重启后端让改动生效**：

```bash
export HOME=/vol1/@appshare/fn-deepseek-harness/微信小程序开发/.tools/home
trim-cli --profile fnos docker container restart 7ce50593505e --yes
```

（token 过期报 `failed to call HTTP proxy .../ogh/ac/w` → 先 `trim-cli --profile fnos login --refresh`）

**本地起后端调试**（三条铁律）：

```bash
export GOROOT=/vol1/@appshare/fn-deepseek-harness/微信小程序开发/.tools/go
export PATH="$GOROOT/bin:$PATH"
export GOCACHE=/tmp/gocache GOPROXY=off    # 默认 GOCACHE 只读，必须覆盖
cd comic/server                            # ① 必须 cd server，否则读不到 .env
go build -o ../.cache/dev-api ./cmd/api
SERVER_PORT=8094 ../.cache/dev-api         # ② 是 SERVER_PORT 不是 PORT
                                           # ③ 二进制放 .cache/，/tmp 会被清
```

mock 登录：`code` 必须以 `1234` 开头（`123456` 可用），`POST /api/v1/login {"phone":"13000000001","code":"123456"}`。

## 发布链路（三条，都不一样）

1. **代码** → `git push origin master`。GitHub 网络极不稳，`RPC failed` / `HTTP2 framing` / 443 超时是常态，**循环重试 5 次 + `ls-remote` 校验**，实测第 3 次成功。不是代码问题，别改代码。
2. **小程序产物** →
   ```bash
   cd comic && npm run build:mp-weixin
   rsync -a --delete --exclude '.git' --exclude 'README.md' dist/build/mp-weixin/ ../artifacts/mp-weixin/
   cd .. && bash scripts/publish-mp-build.sh
   ```
   脚本读 `artifacts/mp-weixin`（**不是** `dist/build/mp-weixin`），跳过 rsync 会报「产物无变化」。产物走 `.mp-build-repo` 固定工作区保证**线性历史**——历史上用临时目录 `git init` 强推导致用户 `git pull` 必报 `fatal: refusing to merge unrelated histories`，**别再改回去**。
   Windows 侧拉包：`git fetch --depth=1 origin mp-build && git reset --hard FETCH_HEAD`。
3. **后端上线** → 重启容器（见上），会自动 `go build`。

## 架构骨架

```
comic/
├── src/            uni-app C 端（33 页，tabBar：首页/摄影师/消息/我的）
│   ├── utils/mappers.ts     ← 核心：所有后端 DTO 在这里归一化
│   ├── utils/nav.ts         ← 导航统一出口 navTo（带 animationType:'none'）
│   ├── stores/{user,home,style}.ts
│   ├── styles/tokens/       ← 唯一色值来源
│   └── components/          MButton/MCard/MTag/MPriceTag/PhotographerCard/StylePicker…
├── server/         Go 后端（80 路由，44 迁移）
│   ├── cmd/api  cmd/ingest(cron)  cmd/events-ingest
│   ├── internal/{handler,service,repository,middleware,eventsource,cache}
│   └── migrations/ 000001~000044（下一个 000045）
├── admin/          Vue3 后台（15 模块，hash 路由 + /admin/ base）
└── doc/            01~08 + home-hero-redesign + review/
```

设计 token：主色 `#FF6B00`、次色 `#00D2D3`、页面 `#0c0d14`、卡片 `#151722`、次级面 `#1D202F`、三级文字 `#8A94A8`（5.84:1）。

## 你必须知道的 6 条业务约定

1. **图片路径必须过 `resolveImage()`**（`src/utils/mappers.ts`）。它只补 `/static/remote/**` 与 `/static/uploads/**` 两种前缀的 origin。
2. **`photographers.name` 是展示名/艺名**，只有 `PUT /v1/photographers/profile` 与 `Activate` 首次建档能改；`users.name` 是账号昵称，`PUT /v1/me/profile` 只写 users。`name_is_custom` 控制是否跟随同步。
3. **互勉不走商品**：`bookings.service_id = 0` 是哨兵，后端按 `mutualHasFreeFallback(mode, serviceCount)` 判定 —— **必须 `serviceCount == 0` 才放行**，否则有付费套餐的 `both` 摄影师会被白嫖。这是付款红线。
4. **`/v1/tags` 返回裸数组**（`c.JSON(data)`，不是 `{list}`），13 个。解析要兼容两种形状。
5. **分页参数是 `size`**，`limit` 全仓无人读。
6. **`user_tokens` 主键是 `token`，没有 id 列**。

## 前端不能踩的坑（全踩过）

- 页面根节点用 `.page{position:fixed;inset:0;flex column;overflow:hidden}` + `.scroll-content{flex:1;height:auto;min-height:0;box-sizing:border-box}`。**不要** `min-height:100vh` + `height:100vh`（真机顶栏会被 window 滚出去）。
- 所有 flex 滚动子项必须 `min-height:0`，否则长内容把输入栏顶到视口外。
- `<input>` 加 `:adjust-position="false"` + 自己算键盘高度（H5 无 `uni.onKeyboardHeightChange`，用 `window.visualViewport` 兜底，基准取 `viewport.height` 不是 `innerHeight`）。
- **封面用 `<image>` 三态**（skeleton / `is-loaded` / `is-error`），不要 `background-image`（小程序端没有 error 事件）。骨架色用 `$surface-2 #1D202F`。
- `@keyframes` 写 `global.scss`，不要写在页面 scoped 里。
- rpx 在 H5 不是合法 CSS 单位，`calc(12rpx + …)` 会算出 0 → 用 `uni.upx2px()`。
- 判断图片可见性看 uni-image 元素的 computed opacity，内层 `<img>` opacity 恒 0。
- 微信昵称只能 `<input type="nickname">` 拿；`chooseAvatar` 路径无扩展名，后端按**文件头魔数**识别格式。
- **切页白屏已定性完毕，别再重开**：`navigateTo` 新建 webview 首帧默认白底是平台机制，开发者无接口控制。已做的缓解：`animationType:'none'`、8 页深色骨架、32 页 `page-meta root-background-color`、`app.wxss` 从 86KB 瘦到 2.3KB。

## 后端不能踩的坑

- 改静态文件中间件时：`WriteHeader` 必须「最后一次生效」。gin `createStaticHandler` 会先试探 `404` 再 `200`，锁死首个 code 会让**所有图片 404 但内容完整**。
- SQL 里 `$2` 同时出现在 `varchar` 与 `text` 上下文会 `SQLSTATE 42P08` → 显式 `$2::varchar`。
- `time` 是 varchar，不能直接和 `CURRENT_TIME` 比。
- 改迁移必须同步改 `server/internal/db/query/*.sql`，否则 `sqlc generate` 把新列吃掉。
- 列表缓存 key 不能含 viewer 维度（会切碎缓存）；需要「是不是我自己」就在出口加 `IsSelf` 字段。
- 空数组不能被当成「加载成功」置 `loaded=true`，否则永久锁死重试。

## 团队协作纪律（用 AgentTeams 时）

- 建队用 `approval=required`，先把 DAG 一次性写全再让用户批。
- 已 started 的卡契约**改不动**（`edit_plan` 会报错）；只能改从未开始的 pending 卡。
- 依赖里不要放「可能 FAIL」的卡，否则下游永远 claim 不到。要解锁就 `edit_plan` 清空 dependencies，**不要**把 failed 粉饰成 completed。
- DSH 重启会让团队 halt 并**取消所有任务**，`resume` 只恢复调度器，必须重建等价卡。
- 成员模型以 `agent_teams_status` 实际显示为准，plan 里写的常被运行时覆盖。`workbuddy/*` 路由唤醒不稳，`deepseek-official/*` 正常。
- 成员上下文超 262K 会死循环，提前缩范围。

## 验证纪律（最重要）

> **凡是「兜底 / 保险 / 防御性」代码，先做 A/B 消除实验再决定留不留。**

反面教材：曾给 23 页加 `::before` 深色兜底层，313 行里 240 行（76%）完全无效——被同色不透明背景遮死，注入 `display:none` 后页面毫无变化，最后全删。

- 声称「修好了」之前必须给证据：`vue-tsc --noEmit` + 双端 build + **产物断言** + 真浏览器实测。
- 验 TS 逻辑从 `dist` 产物 require，别硬抽源码（TS 类型注解会让 `new Function` 失败）。
- CDP 截图验亮度：`captureBeyondViewport:true` 会让视口外测成 0 → 逐卡 `scrollIntoView` 后拍视口图。
- Web 登录注入四键缺一不可：`token` / `user` / `uni-storage-token` / `uni-storage-user`；`goto()` 后必须 `reload()` 才跑 `onLaunch`。
- **桩服务器（SimpleHTTP / 硬编码 JSON）冒充真接口一律判 FAIL。**

## 验收基线

```sql
SELECT (SELECT count(*) FROM bookings)||'|'||(SELECT count(*) FROM services)||'|'||
       (SELECT count(*) FROM photographers)||'|'||(SELECT count(*) FROM users);
-- 期望 23|22|6|18
```

- `GET /v1/tags` → 裸数组 len=13
- `GET /v1/cities` → total=73
- `GET /v1/photographers/5` → `serviceCount=0 hasFree=true services=[(0,'互勉约拍',0)]`
- `GET /v1/photographers/1` → `both serviceCount=4 hasFree=false minPrice=399`，无 id=0
- `POST /bookings {photographerId:1, serviceId:0}` → **404**（白嫖红线）
- `dist/build/mp-weixin` → 1.6M / 242 文件 / 单文件 >200k 为 0

## 现在最该做的事（按性价比）

1. **轮换凭证**：`WX_APP_SECRET` 和 DB 密码曾进过公开仓库，历史未重写，**至今没换**。半小时，风险最高。
2. **配 uploadFile 合法域名 + 真机微信登录一次**：`users.wx_openid` 全表为空 → 订阅消息一条都发不出去。
3. **`resolveImage` 前缀白名单**：现在只认两种前缀，后端加静态目录会再现「图全黑」。
4. `usageCount` 恒 0（`tags.usage_count` 13 行从未维护），热门标签排序没有依据。
5. 支付（已调研完、用户决定后置）、escrow P1（定金登记 + 争议举证附件）。

## 开工前请先做这三步

1. 读 `comic/doc/08-final-handoff.md`（详版，含完整踩坑表和里程碑）。
2. 跑一遍验收基线 SQL + 上面 3 个 curl，确认环境和你以为的一致。
3. 告诉我你打算改什么、改哪几个文件、怎么验证 —— 再动手。

不要一上手就重构。这个项目每一处「看起来奇怪」的写法，背后大概率有一个踩过的坑。

---END---
