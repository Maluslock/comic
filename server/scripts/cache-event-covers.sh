#!/usr/bin/env bash
# 把漫展封面的外链图片抓到本地，供无外网环境显示。
# 外链封面（imagecdn3.allcpp.cn 等）经本机 HTTP(S)_PROXY 下载到 server/static/remote/covers/，
# 并把 comic_events.cover_url 改写为 /static/remote/covers/<文件>。
# 可重复执行（已存在的文件跳过；已是本地路径的不动）。
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIR="$ROOT/static/remote/covers"
PG="${PG_CONTAINER:-comic-postgres}"
DB_USER="${DB_USER:-comic}"
DB_NAME="${DB_NAME:-comic}"
mkdir -p "$DIR"

psql_q() { docker exec "$PG" psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1"; }

count=0
while IFS= read -r url; do
  [ -z "$url" ] && continue
  hash="$(printf '%s' "$url" | sha1sum | cut -c1-16)"
  ext="${url%%\?*}"; ext="${ext##*.}"; ext="$(printf '%s' "$ext" | tr 'A-Z' 'a-z')"
  case "$ext" in jpg|jpeg|png|webp|gif|svg) : ;; *) ext=jpg ;; esac
  file="$hash.$ext"
  if [ ! -s "$DIR/$file" ]; then
    curl -fsSL --connect-timeout 10 --max-time 60 -o "$DIR/$file" "$url" 2>/dev/null || { rm -f "$DIR/$file"; continue; }
  fi
  psql_q "UPDATE comic_events SET cover_url='/static/remote/covers/$file' WHERE cover_url='$url'" >/dev/null
  count=$((count + 1))
  [ $((count % 25)) -eq 0 ] && echo "  cached $count ..."
done < <(psql_q "SELECT DISTINCT cover_url FROM comic_events WHERE cover_url LIKE 'http%' ORDER BY cover_url")

echo "cover cache done: $count updated, $(ls -1 "$DIR" 2>/dev/null | wc -l) files in $DIR"
