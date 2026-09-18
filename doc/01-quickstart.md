# 快速启动与故障排查

这份文档面向第一次在本机搭建项目的开发者。按顺序做完「依赖 → 数据库 → 后端 → C 端 → 后台」，再对照「验证」一节确认全部正常，最后收藏「常用命令」和「故障排查」。

---

## 0. 前置依赖

| 依赖 | 版本要求 | 用途 | 检查命令 |
|------|---------|------|---------|
| Node.js | 20+ | C 端 + 后台前端 | `node -v` |
| npm | 随 Node | C 端包管理 | `npm -v` |
| pnpm | 最新 | 后台是 pnpm monorepo | `pnpm -v` |
| Go | 1.22+ | 后端 API 与采集程序 | `go version` |
| Docker | 最新 | 跑 PostgreSQL / Redis | `docker -v` |

Go 在这个环境里需要显式设置环境变量（不要依赖 shell 默认值）：

```bash
export GOROOT=/home/Haxlock/go
export GOPATH=/home/Haxlock/gopath
export PATH=/home/Haxlock/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
```

---

## 端口与目录速览

| 服务 | 端口 | 启动目录 |
|------|------|---------|
| C 端 H5 | `5173` | 仓库根目录 |
| 后台管理端 | `5174` | `admin/` |
| 后端 API | `8080` | `server/` |
| PostgreSQL | `5433`（容器内 5432） | Docker |
| Redis | `6379` | Docker |

**记住一条：后端必须从 `server/` 目录启动。** 它启动时读取 `server/.env`，里面的 `DB_PORT=5433` 覆盖了默认的 5432。在别的目录启动会连不上数据库。

---

## 1. 启动 PostgreSQL 和 Redis

首次：

```bash
docker run -d --name comic-pg \
  -e POSTGRES_USER=comic -e POSTGRES_PASSWORD=comic123 -e POSTGRES_DB=comic \
  -p 5433:5432 postgres:16-alpine

docker run -d --name comic-redis -p 6379:6379 redis:7-alpine
```

之后每次开机只要 `docker start comic-pg comic-redis` 即可。

确认两个都起来了：

```bash
docker ps --format '{{.Names}} | {{.Ports}}' | grep -E 'comic-pg|comic-redis'
```

> 仓库里有个 `./start.sh` 一键脚本，但它只建 PG、只跑前两个迁移、并且**不启动 Redis**，只适合临时演示。正常开发按本文档的步骤一步步来。

---

## 2. 数据库迁移

后端**不会**在启动时自动跑迁移，需要手动按顺序执行。全部 17 个迁移文件在 `server/migrations/`，命名是 `0000NN_xxx.up.sql` / `0000NN_xxx.down.sql`。

首次建库按顺序执行所有 `.up.sql`：

```bash
for f in $(ls server/migrations/*.up.sql | sort); do
  echo ">> $f"
  docker exec -i comic-pg psql -U comic -d comic < "$f"
done
```

执行完可以确认表是否齐全（应有 19 张表，含 users / photographers / bookings / comic_events / admins / banners 等）：

```bash
docker exec comic-pg psql -U comic -d comic -c "\dt"
```

> 迁移脚本没有 `schema_migrations` 记录表，重复执行可能报「已存在」。如果只是想重置，直接删掉 `comic-pg` 容器重建再跑一遍最省事。
> 也可以用 `server/Makefile` 的 `make migrate-up`（依赖 `golang-migrate` 命令和 `DATABASE_URL`），效果一样。

---

## 3. 启动后端 API

```bash
cd server
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH GOPROXY=https://goproxy.cn,direct
go build -o /tmp/mila-api ./cmd/api
(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &)
```

验证：

```bash
curl -s http://localhost:8080/api/v1/home | head -c 200
```

能返回 JSON 就说明后端、PG、Redis 三者都通了。

### 后端重启配方（重要）

改了后端代码后要重启，用下面这条，靠 `ss` 找到 PID 精确 kill：

```bash
kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')
cd server && go build -o /tmp/mila-api ./cmd/api && (setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &)
```

**不要用 `pkill -f` 来杀后端。** 它会把执行命令的 shell 自己也匹配上，导致命令中途被杀（俗称「自杀」）。日志统一看 `/tmp/mila-api.log`。

---

## 4. 启动 C 端 (uniapp H5)

在仓库根目录：

```bash
npm install        # 首次
npm run dev:h5     # 开发服务，:5173
```

`vite.config.ts` 里 `host: '::'` 已配双栈监听，同局域网的其他设备（手机）可以用本机 IP 直接访问。C 端的 `/api` 请求由 vite proxy 转发到 `http://127.0.0.1:8080`，所以后端起来后前端不需要改地址。

---

## 5. 启动后台管理端

```bash
cd admin
pnpm i                       # 首次
pnpm dev --port 5174         # :5174
```

admin 的 `vite.config.ts` 默认端口也是 5173，会跟 C 端冲突，vite 会自动顺延到 5174。这里显式写 `--port 5174` 更清晰。**5173 是 C 端专用，不要占用或修改。**

管理端接口走 vite proxy：`/api/admin` → `http://localhost:8080`，`.env` 里 `VITE_ADMIN_API_BASE_URL=/api/admin/v1`。用相对路径的好处是局域网 IP 访问也能通（proxy 在服务端转发）。

---

## 6. 验证清单

全部启动后，逐条确认：

| 检查项 | 地址 / 命令 | 期望结果 |
|--------|-----------|---------|
| C 端首页 | http://localhost:5173 | 暗色霓虹首页，有轮播图和「近期漫展」 |
| 后台登录 | http://localhost:5174 | 登录页，用 admin / admin123 能进工作台 |
| 后端健康 | `curl http://localhost:8080/api/v1/home` | 返回 JSON，非 5xx |
| PG | `docker exec comic-pg pg_isready -U comic` | `accepting connections` |
| Redis | `docker exec comic-redis redis-cli ping` | `PONG` |

---

## 测试账号

| 端 | 账号 | 说明 |
|----|------|------|
| C 端摄影师 | 手机号 `10000000001` ~ `10000000004` | 4 位测试摄影师 |
| C 端验证码 | `123456` | **只校验前 4 位是不是 `1234`**，所以填 123456 通过 |
| C 端 coser | `13800138000` | 用户 id 8000 |
| 后台管理 | `admin` / `admin123` | 种子账号，**生产部署前必须改密** |

---

## 常用命令

### C 端（根目录）

| 命令 | 作用 |
|------|------|
| `npm run dev:h5` | H5 开发服务 (:5173) |
| `npm run build:h5` | H5 生产构建 → `dist/build/h5` |
| `npm run dev:mp-weixin` | 微信小程序开发（watch）→ `dist/dev/mp-weixin` |
| `npm run build:mp-weixin` | 微信小程序生产构建 → `dist/build/mp-weixin` |
| `npx vue-tsc --noEmit` | C 端类型检查 |

微信小程序：微信开发者工具导入 `dist/build/mp-weixin`。

### 后台（admin/）

| 命令 | 作用 |
|------|------|
| `pnpm dev --port 5174` | 后台开发服务 (:5174) |
| `pnpm typecheck` | 后台类型检查 |
| `pnpm build` | 生产构建 |

### 后端（server/）

| 命令 | 作用 |
|------|------|
| `go build -o /tmp/mila-api ./cmd/api` | 编译 API |
| `go test ./...` | 跑后端全部测试 |
| `go vet ./...` | 静态检查 |
| `go run ./cmd/ingest` | 单次抓取 nyato（1 页） |
| `go run ./cmd/ingest -pages 5` | 单次抓取 5 页 |
| `go run ./cmd/ingest -config config/cron.yaml` | 以 cron 守护进程方式运行（推荐） |

### 维护命令

| 命令 | 作用 |
|------|------|
| `docker exec comic-redis redis-cli FLUSHALL` | 清 Redis 缓存（改了 home 数据后必做） |
| `git -c http.proxy= -c https.proxy= push gitee master` | 推送到 Gitee（绕过全局代理） |
| `./push-gitee.sh` | 同上，封装好的脚本 |

Gitee 远程地址：`http://100.64.0.16:8418/Haxlock/mila-comic.git`（本地 remote 名 `gitee`）。

---

## 故障排查

### 后端连不上数据库 / 连到了错的端口

99% 是启动目录不对。必须 `cd server` 后再启动，让它读到 `.env`（`DB_PORT=5433`）。确认方式：看 `/tmp/mila-api.log` 里连的是不是 `:5432`。

### 后端进程杀不掉或杀了自己

用精确 PID：

```bash
kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')
```

不要用 `pkill -f "mila-api"`。如果之前已经用 `pkill -f` 导致 shell 卡死，重开一个终端即可。

### 推 Gitee 失败

本机全局代理（`100.64.0.11:7897`）会干扰推送。每次推送显式绕过：

```bash
git -c http.proxy= -c https.proxy= push gitee master
```

### 后台浏览器会话突然掉登录

后台的 admin token **每次登录都会轮换**。如果你在浏览器里已经登录，又在终端 `curl` 调了一次 `/api/admin/v1/login`，旧 token 立刻失效，浏览器就被踢到登录页。做 E2E 测试时保持「单一 admin 会话」：要么浏览器要么 curl，别同时来。

### C 端登录验证码填不对

后端登录逻辑**只校验验证码前 4 位是否为 `1234`**。开发期随便填 `123456`、`1234xx` 都能过。

### 摄影师测试号在 UI 上登录不进去

C 端登录页的手机号正则是 `^1[3-9]\d{9}$`，而测试摄影师号是 `10000000001`，以 `10` 开头，被正则挡住。这是既有约束，不是 bug。调试摄影师流程时可以：走后台或 API 拿 token 后注入 storage，或临时用真实号段。跑摄影师 H5 流程时注意这点。

### 改了首页/漫展数据但前端不刷新

首页数据走 Redis 缓存（key `home`，TTL 5 分钟）。直接改库不会立刻反映。强制刷新：

```bash
docker exec comic-redis redis-cli FLUSHALL
```

管理端下架作品时会主动失效 home 缓存，所以那条路径是即时的。

### 端口被占

- `5173` 被占：检查是不是 C 端已经在跑，或别的项目占了。C 端和 admin 都默认 5173，admin 要显式 `--port 5174`。
- `8080` 被占：`ss -tlnp | grep 8080` 看是谁，可能上一个后端进程没退干净，按上面配方 kill 掉。
- `5433` / `6379` 被占：检查是否已有同名或同端口的容器在跑。

### H5 页面接口 404 或跨域

C 端 `/api` 由 vite proxy 指向 `:8080`。先确认后端在跑；再确认请求确实走的是 `/api/...` 相对路径（不要写死 `http://localhost:8080`）。后端也带了宽松 CORS 中间件，直连理论上也没问题。

### 前端类型检查报错但页面能跑

C 端用 `npx vue-tsc --noEmit`（根目录）；后台用 `cd admin && pnpm typecheck`。两者配置独立，别混用。项目目前没有 lint/test 脚本（后台有 `pnpm lint`，走 oxlint + eslint）。
