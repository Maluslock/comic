# LOCAL_ENV — 本机运行环境（与 AGENTS.md 默认值的差异）

本文件记录**当前这台机器**（`/home/user/comic`，主机 IP `192.168.170.121`）上把本项目跑起来的实际参数与步骤。
项目文档（AGENTS.md）里的默认端口是 `:8080 / :6379`，本机因端口被别的服务占用而改用下列端口。

## 服务一览

| 服务 | 地址 | 说明 |
|------|------|------|
| C 端 H5 | http://192.168.170.121:5173/ | uniapp H5 dev（`host: '::'`，IPv4+IPv6） |
| 后台管理 | http://192.168.170.121:5174/ | soybean-admin dev（已从默认 5173 改为 5174） |
| Go API | http://192.168.170.121:8088/ | `.env` 里 `SERVER_PORT=8088`（**8080 被 password-xl-service 占用**） |
| PostgreSQL | localhost:5433 | 容器 `comic-postgres`（postgres:16-alpine，卷 `comic_pgdata16`） |
| Redis | localhost:6380 | 容器 `comic-redis`（redis:6-alpine；**6379 被 Dify 的 docker-redis-1 占用**） |

> 注意：本机是一个多租户机器，跑着 Dify / Grafana / Prometheus / MySQL 等**无关生产容器**，不要动它们。
> 已停的 `comic-pg` 容器是别的项目的库（`mydb/admin`），与本项目无关，未触碰。

## 依赖与工具链

- **Go 1.22.12**：装在 `/home/user/go-sdk/go1.22`（系统自带的是 1.18，`go.mod` 要求 1.22，不能用）。
  ```bash
  export PATH=/home/user/go-sdk/go1.22/bin:$PATH
  export GOPROXY=https://goproxy.cn,direct
  ```
- Node v22.16.0 / npm 10.9.2 / pnpm 11.25.0
- 依赖已安装：根 `node_modules`（npm）、`admin/node_modules`（pnpm）
- **`admin/.env`**：tarball 是 git archive，`.env` 被 gitignore 排除，已按上游 soybean-admin 模板重建（`VITE_ROUTE_HOME=dashboard`）。

## 启动 / 重启

```bash
# 1) 数据库 + 缓存
docker start comic-postgres comic-redis   # 首次用 docker run 创建（见下）

# 2) 后端（必须在 server/ 目录启动，才会加载 server/.env）
export PATH=/home/user/go-sdk/go1.22/bin:$PATH GOPROXY=https://goproxy.cn,direct
cd /home/user/comic/server
go build -o /tmp/comic-api ./cmd/api
setsid /tmp/comic-api </dev/null >/tmp/comic-api.log 2>&1 &

# 3) C 端 H5
cd /home/user/comic && nohup npm run dev:h5 >/tmp/vite-h5.log 2>&1 &

# 4) 后台管理
cd /home/user/comic/admin && nohup pnpm dev >/tmp/vite-admin.log 2>&1 &

# 健康检查
curl -s localhost:8088/health            # {"status":"ok"}
curl -s localhost:5173/api/v1/home | head -c 200   # 经 vite 代理到 :8088
```

首次创建容器（若容器不存在）：
```bash
docker volume create comic_pgdata16
docker run -d --name comic-postgres -e POSTGRES_USER=comic -e POSTGRES_PASSWORD=comic123 \
  -e POSTGRES_DB=comic -p 5433:5432 -v comic_pgdata16:/var/lib/postgresql/data postgres:16-alpine
docker run -d --name comic-redis -p 6380:6379 -v comic_redisdata:/data redis:6-alpine redis-server --appendonly yes
```

## 数据库初始化（仅首次）

```bash
cd /home/user/comic/server
for f in $(ls migrations/*.up.sql | sort); do
  docker exec -i comic-postgres psql -U comic -d comic -v ON_ERROR_STOP=1 -q < "$f" || break
done
# 演示数据（摄影师账号/认证/漫展/cert 申请/订单）—— migrations 里没有，属运行时数据
docker exec -i comic-postgres psql -U comic -d comic -v ON_ERROR_STOP=1 -q < seed/local_demo.sql
# 漫展封面（000003 依赖事件已存在，须在 seed 后重跑一次）
docker exec -i comic-postgres psql -U comic -d comic -v ON_ERROR_STOP=1 -q < migrations/000003_real_event_images.up.sql
docker exec comic-redis redis-cli FLUSHALL   # 清首页缓存
```

## 演示账号

| 角色 | 手机号 | 验证码 | 说明 |
|------|--------|--------|------|
| 摄影师 1 光影行者 | 10000000001 | 123456 | 已认证（黄V），有 2 件作品 |
| 摄影师 2 樱花落 | 10000000002 | 123456 | 已认证 |
| 摄影师 3 暗夜骑士 | 10000000003 | 123456 | 已认证 |
| 摄影师 4 古风公子 | 10000000004 | 123456 | 已认证 |
| coser | 13800138000 | 123456 | 普通用户 |
| 测试摄影师 | 13700000005 | 123456 | 未认证，有一条已驳回认证申请 |
| 未激活测试号 | 13900001111 | 123456 | 普通用户，未开通摄影师 |
| 后台管理员 | admin | admin123 | 管理端 `:5174` 登录 |

## 本机已知限制（非代码问题）

- 部分外网 CDN 被墙：`picsum.photos`（作品图）、`api.iconify.design`（后台图标）会 `ERR_CONNECTION_CLOSED` → 图片/图标可能不显示，功能不受影响。
- 原机器是 `/vol1/1000/code/comic`；本机工作目录是 `/home/user/comic`，二者不是同一环境。
  Git：原有 `.git`（GitHub 早期 3 提交）已备份为 `.git.early-github.bak`；完整项目源码来自 `mila-comic-master.tar.gz`（快照在 `.snapshots/`），**不含 git 历史**，gitee remote 与凭据未恢复。

## 离线图片（服务器无外网）

服务器本身只经代理 `192.168.170.73:7898` 上网；浏览器/小程序用不了这个代理，所以所有外链图片已抓到本地：

- **抓取脚本**：`scripts/cache-images.sh`（curl 自动走代理，可重复执行）→ 输出 `src/static/img/`（C 端 H5 + 小程序随包），并同步一份到 `server/static/img/`（后端托管）。
- **数据库**：迁移 `server/migrations/000018_local_images.up.sql` 把 摄影师/用户头像、作品图、评论头像、banner、漫展封面、认证样片 全部改为 `/static/img/...`（全新库顺序执行到 000018 即自动生效）。
- **后端**：`cmd/api/main.go` 增加 `router.Static("/static", "./static")`（相对 CWD，**必须从 `server/` 目录启动**）。
- **admin**：`vite.config.ts` 增加 `/static` → :8088 代理；`src/plugins/iconify.ts` 在未设置 `VITE_ICONIFY_URL` 时改用 `@iconify/json` 本地预载 `mdi` / `material-symbols` / `ant-design`，图标不再访问 `api.iconify.design`（首次加载会编译 ~12MB JSON，稍慢，之后浏览器缓存）。
- **登录头像**：`internal/service/auth_service.go` 改为固定 `/static/img/avatar-user.svg`（原来按手机号生成 DiceBear 外链）。
- **新增图片**：跑一次 `scripts/cache-images.sh`，重启后端即可。
- 另：`.gitignore` 的截图规则由 `*.png/*.jpg` 收窄为根级 `/*.png、/*.jpg`，避免把 `src/static`、`server/static` 里的图片误忽略。

## 生产部署（docker，P0）

`deploy/` 下是一套离线可构建的单入口部署（仅用本机已有基础镜像，不拉 Docker Hub）：

- 入口：`http://<host>/`（H5）＋ `http://<host>/admin/`（后台）；`/api/` 反代到后端 api:8080；图片由 nginx 直接托管（H5 产物已含 `static/img`）。
- compose 项目名 `deploy`，容器 `deploy-{postgres,redis,api,web}`；PG/Redis 仅内网，只有 web 暴露 `${WEB_PORT:-80}`。
- 本机已起来：`web 0.0.0.0:80->80`（IPv4+IPv6）、api/postgres/redis 内部。

```bash
cd deploy
cp .env.example .env
./build.sh                                    # H5 + admin + 后端静态二进制（需 node/pnpm/go 在 PATH）
docker compose -f docker-compose.prod.yml up -d --build
./migrate.sh                                  # 首次灌 000001~000018
# 停止：docker compose -f docker-compose.prod.yml down   （加 -v 会清数据卷）
```

注意：开发栈（:5173/:5174/:8088，容器 `comic-postgres`/`comic-redis`）与生产栈（:80）并存互不干扰。
prod 库是全新迁移（`comic_events` 为 0，事件靠 ingest 或演示数据策略决定是否灌入，见 P0 #7）。

