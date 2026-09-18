# 移交清单

> 给接手 米拉漫展 的你。这篇回答三个问题：**我用什么账号能跑起来**、**现在数据是什么状态**、**有哪些坑和没做完的事**。
> 配套：`doc/05-dev-guide.md`（规范/流程/环境）、`doc/07-history.md`（开发历程）、`AGENTS.md`（模块细节）。

---

## 0. 项目速览

| 项 | 值 |
|----|-----|
| 项目 | 米拉漫展 (Mira Comic-Con) |
| 形态 | uniapp 小程序（coser ↔ 摄影师 约拍交易平台） |
| C 端代码 | `src/`（Vue3 + TS + Pinia，暗色霓虹） |
| 后端 | `server/`（Go + Gin + PostgreSQL + Redis，:8080） |
| 管理后台 | `admin/`（soybean-admin，浅色专业风，:5174） |
| 主远端 | `gitee` → `http://100.64.0.16:8418/Haxlock/mila-comic.git`，分支 `master` |
| 提交数 | 186 |
| 工作树 | 干净，与 `gitee/master` 同步（0 ahead / 0 behind） |

---

## 1. 交接 checklist（逐项打勾）

- [ ] **跑后端测试**：`cd server && go test ./...`，全绿。
- [ ] **C 端类型检查**：仓库根 `npx vue-tsc --noEmit`，无报错。
- [ ] **admin 类型检查**：`cd admin && pnpm typecheck`。
- [ ] **起后端**：`cd server && go run ./cmd/api`（:8080）。
- [ ] **起 C 端**：`npm run dev:h5`（:5173），看到暗色霓虹首页。
- [ ] **起 admin**：`cd admin && pnpm dev --port 5174`，能看到登录页。
- [ ] **逐端登录验证**：C 端摄影师号、C 端 coser 号、admin，各登一次（见第 2 节）。
- [ ] **改 admin 密码**：把种子 `admin/admin123` 换掉（生产红线，见第 6 节）。
- [ ] **检查 `server/.env`**：DB_PORT=5433、Redis 6379、SERVER_PORT=8080 是否匹配你机器。
- [ ] **确认推送链路**：`git -c http.proxy= -c https.proxy= push gitee master` 能推。
- [ ] **确认 Redis 可用**：首页数据加载正常（缓存命中）。
- [ ] **读一遍 AGENTS.md**，尤其你接下来要动的模块那一节。

---

## 2. 测试账号表

### 2.1 C 端用户（登录用手机号 + 验证码）

**验证码规则**：服务端只校验前缀 `1234`，随便填 `123456` 即可通过。

| 手机号 | 身份 | 角色/绑定 | 备注 |
|--------|------|-----------|------|
| `10000000001` | 摄影师 | 光影行者（photographer 1） | UI 已放行 10 万号段 |
| `10000000002` | 摄影师 | 樱花落（photographer 2） | |
| `10000000003` | 摄影师 | 暗夜骑士（photographer 3） | |
| `10000000004` | 摄影师 | 古风公子（photographer 4） | |
| `13800138000` | coser + 测试摄影师 | 用户 8000，关联 photographer 5 | 双身份演示用 |
| `13900001111` | 未激活号 | 纯 coser | |
| `13000000001` | 未激活号 | 用户 142，coser | E2E 里 禁用→启用 过 |
| `13700001234` | 未激活号 | coser | |

### 2.2 管理后台

| 账号 | 密码 | 地址 | 备注 |
|------|------|------|------|
| `admin` | `admin123` | http://localhost:5174 | **生产必须改密**；id=1 主管理员不可禁用 |

### 2.3 特殊说明

- **10 万号段**：C 端 UI 登录正则 `^1[3-9]\d{9}$` 会拦 100 开头的号。摄影师测试号 `10000000001~04` 已单独放行（commit `108034d`）。若个别场景仍被拦，走存储注入 token 的方式验证（H5 场景）。
- **admin token 每次登录轮换**：用 curl 登录会让浏览器里已有的 admin 会话失效（401 跳登录）。做浏览器 E2E 时保持单一会话，别混用 curl 登录。

---

## 3. 演示数据现状

> 基线数据，供你判断"现在的库长什么样"。具体值以实际库为准。

| 数据 | 状态 |
|------|------|
| 摄影师 | 5 个（id 1-5）；1-4 `certified=true`（有黄V），5 `certified=false` |
| 认证申请 | 7 条：1/3/4 approved，2/5/6/7 rejected；**#5/#6/#7 是测试摄影师 5 的驳回记录，可演示"重新申请"** |
| 作品 works | 4 条，均 active：原神-雷电将军 / 鬼灭之刃 / 魔卡少女樱 / 古风仙侠 |
| 评论 reviews | 3 条（小狐狸 5★ / 月华 5★ / 用户5213 1★，均属光影行者） |
| 标签 tags | 13 个，usageCount 真实统计（0-2） |
| 轮播 banners | 3 条，均上线：ChinaJoy 2026(sort 0) / 第40届萤火虫漫展(sort 1) / CP33 综合同人展(sort 2) |
| 用户 users | 约 17 个 |
| 漫展 events | 约 48 条活跃（`del_flag=false`），由 nyato 抓取管道维护 |
| 管理员 admins | 1 个（种子 admin） |
| 订单 bookings | 若干，含 E2E 测试流转单（pending/confirmed/completed/cancelled 各状态） |

**演示"已驳回 → 重新申请"**：用 `13800138000`（测试摄影师 5）登录 C 端，进认证申请页，会看到历史被驳回 + 重新申请按钮，整条链路可跑。

---

## 4. 已知问题与局限

> 这些不是 bug 清单，是"当前实现的边界"。演示/验收时别当事故。

### 4.1 功能上没做的

| 项 | 现状 | 影响 |
|----|------|------|
| 图片/头像上传 | 全部是 **URL 文本框**（DiceBear / picsum 外链） | 没有真文件上传 |
| 微信订阅消息 | 未接入（AppID 预留） | 没有微信推送，站内通知有 |
| 操作日志审计 | 未做 | 管理员操作无处追溯 |
| super 角色 | 只有字段预留，**无权限区分** | admin 和 super 行为一致 |
| 摄影师详情管理端 | 管理端无独立摄影师详情页 | 只能在列表看/认证 |
| 认证申请图片 | 样片也是 URL 输入 | 同上 |
| 作品编辑 | 只能发布/删除，不能改 | 改内容要删了重建 |
| 聊天实时性 | 进页面拉取，无 WebSocket | 无实时推送，未读数恒为 0 |

### 4.2 技术债 / 注意点

- **C 端部分页面无 mock fallback**：依赖后端运行。后端没起时，某些页面会空/报错，不是前端坏了。
- **10 万号段登录**：见 2.3；UI 正则是历史遗留。
- **admin token 轮换**：见 2.3。
- **chat peer 映射**：demo 里 `photographer.id` 直接当 peer `userId`（seed 数据满足），真实数据需保证摄影师有对应 user 行。
- **`GetByUser` 字段**：P2a 曾漏 mode/certified，已在 `2d6f79d` 修复，注意别再回退。
- **`TestPhotographerHandler_Detail_Success`**：曾因 mock 扫描顺序 panic，已随 C 包债务修复处理；若再动 photographers 表列，记得同步 mock。

---

## 5. 后续路线图

按优先级排列，不是承诺，是候选池。

### P2 完善（收尾现有体系）

- **操作日志审计**：管理员禁用/重置密码/认证审批的操作留痕。
- **摄影师详情管理端**：管理端能看单个摄影师的完整画像（作品/评分/订单）。
- **通知 read 标记**：补 `POST /notifications/:id/read`，解决消息页未读点每次 `onShow` 重置（详见 AGENTS.md P3 Known gaps）。

### 部署生产化

- 后端打 Go binary（`go build`），不要 `go run`。
- admin `pnpm build` 出静态产物。
- 反向代理（nginx）统一入口 + 静态托管；考虑 docker 化。
- PostgreSQL / Redis 接生产实例，迁移民 migration 上库。

### 微信小程序上架准备

- `manifest.json` 填真实 AppID（现为预留）。
- 配置 request/upload 域名白名单。
- 申请并接入订阅消息模板。
- **admin 改密**（生产红线）。
- 清理演示数据 / 确认 seed 账号策略。

---

## 6. 注意事项合集（避免踩坑）

> 这些坑大多在开发期亲身踩过，写下来免得你重蹈。

| 类别 | 坑 | 对策 |
|------|-----|------|
| 代理 | 内网 Gitee 被 http 代理拦 | `git -c http.proxy= -c https.proxy= push gitee master` |
| 进程 | `pkill -f go` 会把自己一起杀 | 用 `kill <PID>` 精确结束 |
| 工作目录 | 从仓库根跑后端，读不到 .env | 必须 `cd server/` 再启动 |
| 数据库端口 | 用默认 5432 连不上 | `.env` 里是 **5433** |
| 缓存 | 改首页/轮播/作品数据不生效 | Redis `FLUSHALL`（管理端下架已自动失效） |
| 端口 | 5173 / 5174 / 8080 / 5433 / 6379 | 见下表，别占错 |
| 双栈监听 | vite 监听 `::`（IPv6 双栈） | 部分环境 localhost 走 IPv4，必要时用 127.0.0.1 |
| 前端主题 | 两边主题互串 | C 端暗色霓虹 / admin 浅色，泾渭分明 |
| admin 模板 | 多根节点白屏 | 路由页单根 `<div>` |
| admin 会话 | curl 登录顶掉浏览器 | 单一会话原则 |
| PowerShell | 改文件乱码 | 用 Write/Edit 工具 |

### 端口表

| 端口 | 服务 | 说明 |
|------|------|------|
| 5173 | C 端 H5 | **固定，禁止占用/修改** |
| 5174 | admin 开发 | soybean-admin 改此端口避免冲突 |
| 8080 | 后端 API | Gin（C 端 `/api/v1` + 管理端 `/api/admin/v1`） |
| 8081 | 遗留 mock | Go 标准库单机演示，一般不用 |
| 5433 | PostgreSQL | 注意不是 5432 |
| 6379 | Redis | 首页缓存 |

---

## 7. 关键文件索引

| 找什么 | 去哪 |
|--------|------|
| 全部模块实现细节 | `AGENTS.md` |
| 前端页面 | `src/pages/`，路由 `src/pages.json` |
| API 调用层 | `src/api/index.ts` + `client.ts` |
| 后端路由 | `server/cmd/api/main.go` |
| 数据库 schema | `server/migrations/`（000001~000017） |
| 抓取管道 | `server/internal/ingest/`，配置 `server/config/cron.yaml` |
| 管理后台页面 | `admin/src/views/` |
| 设计与计划 | `docs/superpowers/specs/` + `docs/superpowers/plans/` |
| 开发过程台账 | `.superpowers/sdd/`（脚手架，可忽略） |

---

## 8. 交出去的那句话

代码是干净的（186 提交、工作树整洁、与远端同步），文档是全的（AGENTS.md + 本目录三篇），流程是通的（spec→plan→SDD 已验证多轮）。你需要做的是：**跑通一节 checklist，改掉 admin 密码，然后挑路线图里的第一项开工。** 剩下的交给 `AGENTS.md`。
