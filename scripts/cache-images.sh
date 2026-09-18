#!/usr/bin/env bash
# 缓存外链图片到本地，使项目在“服务器无外网”时可离线显示图片。
# 依赖本机代理：curl 自动读取 HTTP(S)_PROXY（本机为 http://192.168.170.73:7898）。
# 产物：
#   src/static/img/     —— C 端 H5 + 小程序随包静态资源（离线可用）
#   server/static/img/  —— Go 后端托管（admin 端经 /static 代理访问）
# 可重复执行（已存在且非空的文件会跳过）。
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/src/static/img"
SRV="$ROOT/server/static/img"
mkdir -p "$OUT" "$SRV"

DICEBEAR="https://api.dicebear.com/7.x/avataaars/svg"
PICSUM="https://picsum.photos/seed"

fetch() { # <name> <url>
  local name="$1" url="$2"
  if [ -s "$OUT/$name" ]; then printf "skip  %-30s\n" "$name"; return 0; fi
  if curl -fsSL --connect-timeout 10 --max-time 90 -o "$OUT/$name" "$url"; then
    printf "ok    %-30s (%s bytes)\n" "$name" "$(wc -c <"$OUT/$name")"
  else
    printf "FAIL  %-30s <- %s\n" "$name" "$url"; rm -f "$OUT/$name"
  fi
}

echo "== 头像（DiceBear SVG）=="
fetch avatar-photographer1.svg "$DICEBEAR?seed=photographer1&backgroundColor=b6e3f4"
fetch avatar-photographer2.svg "$DICEBEAR?seed=photographer2&backgroundColor=ffd5dc"
fetch avatar-photographer3.svg "$DICEBEAR?seed=photographer3&backgroundColor=c0aede"
fetch avatar-photographer4.svg "$DICEBEAR?seed=photographer4&backgroundColor=d1d4f9"
fetch avatar-photographer5.svg "$DICEBEAR?seed=photographer5&backgroundColor=fce4ec"
fetch avatar-coser1.svg        "$DICEBEAR?seed=coser1&backgroundColor=ffdfbf"
fetch avatar-coser1b.svg       "$DICEBEAR?seed=coser1&backgroundColor=ffe0b2"
fetch avatar-coser2.svg        "$DICEBEAR?seed=coser2&backgroundColor=c9e9f6"
fetch avatar-test1.svg         "$DICEBEAR?seed=test1&backgroundColor=e0f2f1"
fetch avatar-user.svg          "$DICEBEAR?seed=user&backgroundColor=eceff4"

echo "== 作品图（picsum JPG 600x450）=="
for i in 1 2 3 4 5; do fetch "work-$i.jpg" "$PICSUM/coswork$i/600/450"; done

echo "== 漫展封面（官方外链）=="
fetch cover-chinajoy.jpg "https://cms3.chinajoy.net/157/upload/resources/image/103312.jpg"
fetch cover-ccg.jpg      "https://english.shanghai.gov.cn/cmsres/04/04acd83239cf4fa8a45c056e2b0f01c0/9c3f75159398444056f2b7e40b37e8ab.jpg"

echo "== 兜底 banner（mock，750x360）=="
for i in 1 2 3; do fetch "banner-$i.jpg" "$PICSUM/comic$i/750/360"; done

echo "== 其余演示头像（150x150）=="
fetch avatar-guest.jpg               "$PICSUM/guest/150/150"
fetch avatar-fox.jpg                 "$PICSUM/fox/150/150"
fetch avatar-coser-chat.jpg          "$PICSUM/FemaleCoser/150/150"
fetch avatar-photographer-chat.jpg   "$PICSUM/PhotographerLight/150/150"

echo "== 认证样片（picsum 600x450）=="
fetch cert-1.jpg  "$PICSUM/cert1/600/450"
fetch cert-2.jpg  "$PICSUM/cert2/600/450"
fetch cert-2b.jpg "$PICSUM/cert2b/600/450"
fetch cert-3.jpg  "$PICSUM/cert3/600/450"
fetch cert-5.jpg  "$PICSUM/cert5/600/450"

echo "== 同步到 server/static/img =="
cp -f "$OUT"/* "$SRV"/
echo "done: $(ls -1 "$OUT" | wc -l) files in src/static/img"
