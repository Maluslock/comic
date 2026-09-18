# 开发规范与流程

> 面向接手 米拉漫展 的开发者。读完这篇，你应该能独立开一个新模块、跑通测试、按规矩提交代码。
> 配套阅读：`AGENTS.md`（899 行项目知识库，所有模块的实现细节都在那）+ `doc/06-handover.md`（账号/环境/坑）+ `doc/07-history.md`（怎么走到今天的）。

## 0. 一句话项目画像

米拉漫展是一个 uniapp 小程序：漫展场景下 coser 找摄影师约拍。Vue 3 + TS + Pinia + Vite 前端，Go（Gin + PostgreSQL + Redis）后端，外加一个独立的管理后台（admin/）。

| 部分 | 技术 | 端口 | 目录 |
|------|------|------|------|
| C 端小程序 | uniapp + Vue3 + TS | 5173 (H5) | `src/` |
| 后端 API | Go + Gin + pgx + Redis | 8080 | `server/` |
| 管理后台 | soybean-admin (Vue3 + NaiveUI) | 5174 | `admin/` |
| 数据库 | PostgreSQL | 5433 | `server/migrations/` |
| 缓存 | Redis | 6379 | 首页缓存 |
| 遗留 mock | Go 标准库 | 8081 | `server/mock_server.go` |

---

## 1. 代码规范

### 1.1 前端通用（C 端 src/ 与 admin/ 各自适用）

- **只用 `<script setup lang="ts">`**，禁止 Options API。
- **尺寸一律 rpx**（C 端），不写 px/rem。admin 用模板约定（px / rem / UnoCSS class）。
- **设计 token 优先**：能用 SCSS 变量就用，不硬编码颜色尺寸。
- **路径别名 `@/`**：跨目录 import 一律 `@/components/...`，不用相对路径。
- **TypeScript strict**：`"strict": true`，不接受 `any`。后端同类要求见 1.4。
- **页面即目录**：`src/pages/<name>/index.vue`，`<style lang="scss" scoped>`。
- **Pinia 组合式 store**：`defineStore('name', () => { ... })`，不用 options 写法。
- **不直接 `uni.request`**：所有请求走 `src/api/index.ts`（内部包一层 `client.ts` 注入 Bearer token）。
- **生命周期用 uniapp 钩子**：`@dcloudio/uni-app` 的 `onLaunch`/`onShow`，不混用原生 Vue 生命周期。
- **图标不用 emoji**：用 SVG（`src/static/icons/`）或几何字符（★ ☆ ◇ ◆ → ✓ ⚠ ℹ）。
- **图片策略**：用户头像用 DiceBear（`https://api.dicebear.com/7.x/avataaars/svg?seed=<seed>`），封面/作品图用 picsum.photos；图片容器要有渐变兜底。
- **改文件用编辑器/Write 工具**，不要用 PowerShell（会破坏 UTF-8）。

### 1.2 C 端暗色霓虹主题（强制）

C 端是统一的暗色霓虹风，底色 `#0a0a1a`，双主色紫 `#a855f7` + 青 `#06b6d4`。所有新页面必须遵守：

| 用途 | 变量 | 值 |
|------|------|-----|
| 页面底色 | `$dark-bg-primary` | `#0a0a1a` |
| 卡片底 | `$dark-bg-card` | rgba(255,255,255,0.05) |
| 次级面板 | `$dark-bg-secondary` | `#12122a` |
| 主文字 | `$dark-text-primary` | `#e2e8f0` |
| 次文字 | `$dark-text-secondary` | `#94a3b8` |
| 弱文字/时间戳 | `$dark-text-tertiary` | `#64748b` |
| 主强调 | `$neon-purple` | `#a855f7` |
| 双色渐变 | `$neon-gradient` | linear-gradient(135deg, 紫, 青) |
| 分割线/边框 | `$dark-border` | rgba(255,255,255,0.08) |

几条肉眼可见的规则：

- 卡片容器统一 `1rpx solid $dark-border` + 深色阴影（`0 X Y rgba(0,0,0,0.2~0.4)`）。
- 区块标题左侧一条霓虹渐变竖条：`::before` 6rpx × 28rpx，用 `$neon-gradient`。
- 选中/激活态：`$neon-purple` 描边 + `box-shadow: 0 0 12rpx $neon-purple-glow`。
- 卡片按下：`scale(0.97~0.98)` + 阴影收敛。
- **消灭所有硬编码灰**（`#ccc` / `#f0f0f0` / `#ddd` 一律换变量）。
- profile 类页面转暗色时，保留原有顶部渐变头。

### 1.3 admin 浅色专业风（边界：不要互串）

管理后台保持 soybean-admin 模板的浅色专业风（NaiveUI + UnoCSS）。**两边主题绝不能互相污染**：

- C 端页面严禁引入 admin 的浅色组件风格。
- admin 页面严禁照搬 C 端的 `$dark-*` / `$neon-*`。
- admin 路由页必须**单根 `<div>`**（`vite-plugin-vue-transition-root-validator` 会拦截多根并报错）。

一句话记住：C 端是夜店霓虹，admin 是办公室日光灯，各玩各的。

### 1.4 后端 Go 规范

- Gin 路由集中在 `server/cmd/api/main.go` 注册；业务分层为 handler → service → repository。
- 错误语义用 HTTP 状态表达：400 参数、401 未登录、403 越权、404 不存在、409 冲突/状态非法。
- 事务用 repo 层原子方法（如认证审批 `ReviewApproveTx`），不要在 service 里拼多写。
- 通知类副作用 best-effort：写入失败只记日志，绝不阻塞主流程（见 `booking_service.notify`）。
- repo 层测试优先 fake/interfaces（如 `worksStore` seam），handler 测试注意 mock 扫描列顺序与 migration 同步。

---

## 2. 目录约定：加一个功能要动哪些文件

| 场景 | 位置 |
|------|------|
| 新增页面 | `src/pages.json` 注册 → `src/pages/<name>/index.vue` |
| 新增接口调用 | `src/api/index.ts`（HTTP 走 `src/api/client.ts`），保留 `src/data/mock.ts` 兜底 |
| 新增数据模型 | `src/types/index.ts`（所有 interface 集中在这） |
| 新增 store | `src/stores/`，组合式写法 |
| 新增全局样式 | `src/styles/global.scss`；变量进 `src/styles/variables.scss`（自动注入所有 SFC） |
| 新增后端路由 | `server/cmd/api/main.go`（C 端 v1 / 管理端 `/api/admin/v1`）+ 对应 handler/service/repo |
| 数据库变更 | `server/migrations/0000NN_xxx.{up,down}.sql`（连号，up/down 都要写） |
| 管理端新页 | `admin/src/views/<name>/index.vue` + `admin/src/service/api/admin.ts`（elegant-router 自动路由） |

---

## 3. 开发流程（AI 辅助，已是既定事实）

本项目全程 AI 辅助开发，流程固定且已被验证有效。**每个功能模块都走同一套流水线**：

```
brainstorming（想清楚要什么）
   ↓
spec  设计文档  → docs/superpowers/specs/<date>-<module>-design.md
   ↓
plan  实施计划  → docs/superpowers/plans/<date>-<module>.md
   ↓
SDD  子代理驱动开发（Subagent-Driven Development）
   ├─ 每个 task 派一个 implementer 子代理实现
   ├─ 每个 task 派一个 reviewer 子代理审查
   └─ 逐 task 闭环，最后整体验证 + 推送
```

### 3.1 流程要点

- **先设计后动手**：不喜欢"直接开写"，先出 spec 定接口/数据模型/边界，再出 plan 拆 task。
- **task 粒度**：一个 plan 通常 3-7 个 task，每个 task 有明确的文件清单与验收标准。
- **实现与审查分离**：implementer 和 reviewer 是不同子代理，避免自说自话。
- **SDD 台账**：过程产物在 `.superpowers/sdd/<module>/`，属于开发脚手架，接手后可以忽略甚至删除，不影响运行。
- **文档同步**：模块落地后回写 `AGENTS.md` 对应小节（这是项目最重要的知识库，别偷懒）。

### 3.2 现有 spec / plan 清单

`docs/superpowers/specs/` 19 篇，`docs/superpowers/plans/` 20 篇。找某个模块的设计就翻这两处。

| 日期 | 模块 |
|------|------|
| 2026-07-26 | layer1 backend/frontend hardening |
| 2026-07-28 | backend data overhaul / next phase |
| 2026-08-05 | city popup + scheduled ingest |
| 2026-08-24 | full-pipeline auth & data integration / order-domain state machine |
| 2026-08-27 | favorites + chat |
| 2026-08-28 | event follow / notifications / p2a dual-role |
| 2026-09-01 | admin web |
| 2026-09-02 | admin banner management / admin P0 (user mgmt + cert audit) |
| 2026-09-04 | admin admin-mgmt / admin content / admin notification / cend cert-apply / work-management |
| 2026-09-07 | profile-edit / user-profile-edit |

> 命名规律：`<date>-<module>.md`（plan），`<date>-<module>-design.md`（spec）。新模块照着这套命名走即可。

---

## 4. 测试与验证

每次改完，按改动范围挑对应命令跑。**验证不是可选项。**

| 目标 | 命令 | 工作目录 |
|------|------|----------|
| 后端全量测试 | `go test ./...` | `server/` |
| C 端类型检查 | `npx vue-tsc --noEmit` | 仓库根 |
| admin 类型检查 | `pnpm typecheck` | `admin/` |
| H5 起服务 | `npm run dev:h5` | 仓库根 |
| 小程序产物 | `npm run build:mp-weixin` → 导入 `dist/build/mp-weixin` | 仓库根 |
| admin 起服务 | `pnpm dev --port 5174` | `admin/` |

- **Go 测试文件分布**：`server/internal/handler/`、`server/internal/service/`、`server/internal/ingest/`。
- **E2E / 浏览器验证**：用 Playwright 或 agent-browser。Chromium 在 `/home/Haxlock/.cache/ms-playwright/chromium-1234/chrome-linux64/chrome`。
- **没人跑 lint 脚本**（项目未配置），所以类型检查 + 测试就是主要防线。
- 后端改了数据结构或首页逻辑，记得配合 Redis 操作（见第 6 节）。

---

## 5. Git 提交与推送规范

### 5.1 提交信息

前缀 + 按模块原子提交：

| 前缀 | 用途 |
|------|------|
| `feat:` | 新功能（可带 scope，如 `feat(c-end):` / `feat(admin):` / `feat(api):`） |
| `fix:` | 修 bug（如 `fix(security):` / `fix(works):`） |
| `docs:` | 文档（含 AGENTS.md 更新） |
| `refactor:` | 重构，不改行为 |
| `chore:` | 杂项（脚本、配置） |
| `test:` | 测试 |

原则：一个提交讲一件事。设计和实现分开提交（`docs:` 设计/计划一篇，`feat:`/`fix:` 实现按层一篇）。

### 5.2 推送到 Gitee（重要）

远端：`gitee` → `http://100.64.0.16:8418/Haxlock/mila-comic.git`，分支 `master`。

本机有代理会拦住内网 Gitee，**推送必须显式绕过代理**：

```bash
git -c http.proxy= -c https.proxy= push gitee master
```

`origin`（GitHub `Maluslock/comic.git`）是另一条远端，日常以内网 Gitee 为准。

---

## 6. 本地环境 recipe

### 6.1 Go 环境变量

```bash
export GOROOT=/home/Haxlock/go
export GOPATH=/home/Haxlock/gopath
export PATH=/home/Haxlock/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
```

### 6.2 启动后端（有两个硬性前提）

```bash
cd /vol1/1000/code/comic/server   # 必须从 server/ 启动
go run ./cmd/api                  # :8080
```

- **必须 cd 到 `server/`**：`.env` 在那儿，`DB_PORT=5433`（注意不是默认 5432）。从仓库根跑会读不到配置。
- **重启服务用 `kill <PID>`**：不要用 `pkill -f`，模式会匹配到自己的进程，等于自杀。
- **Redis**：改了首页数据/轮播/精选作品后，缓存可能让改动不生效，执行 `FLUSHALL` 清一下（管理端下架作品已自带缓存失效，无需手动）。
- 抓取任务：`go run ./cmd/ingest -config config/cron.yaml`（YAML 驱动 cron，推荐）；单跑 `go run ./cmd/ingest -pages 5`。

---

## 7. 常见坑速查

| 坑 | 表现 | 对策 |
|----|------|------|
| git 推送被代理拦 | push 超时/拒绝 | `git -c http.proxy= -c https.proxy= push gitee master` |
| `pkill -f` 自杀 | 命令把自己 kill 了 | 用 `kill <PID>` |
| 后端从根目录起 | 读不到 .env / 连不上库 | 一定 `cd server/` 再 `go run` |
| Redis 缓存 | 改数据前端不变 | `FLUSHALL` |
| admin 多根模板 | 页面白屏 / 校验报错 | 路由页单根 `<div>` |
| 10 万号段登录 | UI 正则拦截测试号 | 走 token 注入（见交接文档） |
| admin token 轮换 | curl 登录后浏览器 401 | 别用 curl 登录占同一账号 |
| 端口撞车 | 5173 被占 | C 端固定 5173，admin 用 5174 |

---

## 8. 上手检查清单

新接手的第一天，按这个顺序走一遍：

1. `cd server && go test ./...`（全绿说明后端环境 OK）。
2. `npx vue-tsc --noEmit`（根目录，C 端类型干净）。
3. `cd admin && pnpm typecheck`。
4. 起后端 + 起 C 端 H5 + 起 admin，用测试账号各登录一遍（账号见 `doc/06-handover.md`）。
5. 挑一个已有模块的 spec + plan 读一遍，感受这套 AI 工作流。
6. 改一行无关紧要的文案，走一遍 `feat/fix` 提交 + Gitee 推送，确认链路通。

跑通这 6 步，剩下的都是业务代码的事。
