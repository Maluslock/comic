# 生产部署（Docker）

面向 P0「部署生产化」。单入口 nginx 同时托管 C 端 H5（`/`）、管理后台（`/admin/`），并把 `/api/` 反代到 Go 后端；PG/Redis 由 compose 内置。

## 组成

| 文件 | 作用 |
|------|------|
| `build.sh` | 在宿主机产出三份产物：H5 `dist/build/h5`、admin `admin/dist`、后端 `server/build/comic-api`（静态链接） |
| `Dockerfile.api` | `alpine` + 预编译二进制 + `server/static`（本地图片） |
| `Dockerfile.web` | `nginx:alpine` + H5 产物 + admin 产物 + `nginx.conf` |
| `nginx.conf` | `/` → H5；`/admin/` → admin；`/api/` → `api:8080`（IPv4+IPv6 监听） |
| `docker-compose.prod.yml` | postgres/redis/api/web，仅暴露 web |
| `migrate.sh` | 按序把 `server/migrations/*.up.sql` 灌入 compose 的 postgres |

镜像只用本机已有的基础镜像（`nginx:alpine`/`alpine`/`postgres:16-alpine`/`redis:6-alpine`），**不依赖 Docker Hub**；前端/后端在宿主机构建，容器只做打包运行。

## 一键部署

```bash
cd deploy
cp .env.example .env                 # 改 DB_PASSWORD / WEB_PORT
./build.sh                           # 需要 node/npm/pnpm + go(1.22+) 在 PATH
docker compose -f docker-compose.prod.yml up -d --build
./migrate.sh                         # 首次初始化数据库（000001~000018）
```

访问：`http://<host>:${WEB_PORT}/`（C 端）、`http://<host>:${WEB_PORT}/admin/`（后台）。

## 说明 / 待办

- **改密（P0）**：后台种子 `admin/admin123` 仅演示；上线前用「管理员管理 → 重置密码」改掉。
- **AppID / 域名白名单（P0）**：`src/manifest.json` 的 AppID 与微信 `request/uploadFile` 合法域名需在微信侧配置，本仓库无法代填。
- **订阅消息（P0）**：模板申请与前端接入未做（后端 `POST /api/v1/subscribe` 已预留）。
- **HTTPS**：本配置为 HTTP。上域名后在外层终止 TLS（或改造 `nginx.conf` 监听 443）。
- **数据卷**：`pgdata`/`redisdata`/`uploads` 由 compose 管理；`docker compose down -v` 会清空数据。
- **上传图持久化**：`uploads` 卷挂到 api 的 `/app/static/uploads`；nginx `location /static/uploads/` 代理到 api，重建容器不丢图。
- admin 生产构建使用 `VITE_BASE_URL=/admin/`（见 `admin/.env.prod`）。
