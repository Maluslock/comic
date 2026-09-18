#!/bin/bash
# 米拉漫展 一键启动脚本
echo "=== 米拉漫展 启动 ==="

# 1. PostgreSQL (Docker)
echo "[1/4] 启动 PostgreSQL..."
docker rm -f comic-pg 2>/dev/null
docker run -d --name comic-pg \
  -e POSTGRES_USER=comic -e POSTGRES_PASSWORD=comic123 -e POSTGRES_DB=comic \
  -p 5433:5432 postgres:latest
echo "  等待 PG 就绪..."
until docker exec comic-pg pg_isready -U comic 2>/dev/null; do sleep 2; done

# 2. 数据库迁移
echo "[2/4] 运行数据库迁移..."
cat server/migrations/000001_create_tables.up.sql | docker exec -i comic-pg psql -U comic -d comic >/dev/null 2>&1
cat server/migrations/000002_seed.up.sql | docker exec -i comic-pg psql -U comic -d comic >/dev/null 2>&1
echo "  迁移完成"

# 3. Backend API
echo "[3/4] 编译并启动后端..."
cd server
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath GOPROXY=https://goproxy.cn,direct
$GOROOT/bin/go build -o /tmp/comic-api ./cmd/api 2>/dev/null
cd ..
export DB_HOST=localhost DB_PORT=5433 DB_USER=comic DB_PASSWORD=comic123 DB_NAME=comic \
       REDIS_HOST=localhost REDIS_PORT=6379 SERVER_PORT=8080
setsid /tmp/comic-api </dev/null >/tmp/comic-api.log 2>&1 &
echo "  后端: http://localhost:8080"

# 4. Frontend
echo "[4/4] 启动前端..."
setsid node ./node_modules/.bin/uni --host 0.0.0.0 </dev/null >/tmp/vite-dev.log 2>&1 &
echo "  前端: http://localhost:5173"
echo "=== 启动完成 ==="
