#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO_BIN="${GO:-go}"

echo "[1/3] H5 build"
(cd "$ROOT" && npm run build:h5)

echo "[2/3] admin build"
(cd "$ROOT/admin" && pnpm build)

echo "[3/3] api build (static, CGO_ENABLED=0)"
(cd "$ROOT/server" && CGO_ENABLED=0 "$GO_BIN" build -tags timetzdata -ldflags="-s -w" -o build/comic-api ./cmd/api)

echo "done: dist/build/h5 | admin/dist | server/build/comic-api"
