#!/usr/bin/env python3
"""把 comic_events 里仍未本地化的外链封面抓到 server/static/remote/covers/，
并把 cover_url 改写为 /static/remote/covers/<allcpp_id>.<ext>。

为什么需要这一步：
    cmd/sync 走 internal/service/event_sync.go，cover_url 会被拼成
    https://imagecdn3.allcpp.cn/upload/... 这种外链。小程序端 utils/mappers.ts 的
    resolveImage() 只给 /static/remote/ 前缀补 origin，外链在无外网环境会直接空白，
    封面也会随 CDN 失效。所以同步之后必须本地化。

为什么不复用 scripts/cache-event-covers.sh：
    那个脚本依赖 `docker exec comic-postgres`，并且按 sha1sum 前 16 位命名；
    本仓库历史封面是按 <allcpp_id>.<ext> 命名的，两边命名不一致会重复存一份。

用法（在 comic/server 下执行，或任意目录，脚本用绝对路径定位）：
    python3 scripts/cache-allcpp-covers.py
    python3 scripts/cache-allcpp-covers.py --dry-run

依赖：psql（PGPASSWORD 环境变量）、python3 标准库。可重复执行（已存在的非空文件跳过）。
"""
import argparse
import os
import subprocess
import sys
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
SERVER = os.path.dirname(HERE)
DIR = os.path.join(SERVER, "static", "remote", "covers")

PSQL_BASE = [
    "psql",
    "-h", os.environ.get("DB_HOST", "127.0.0.1"),
    "-p", os.environ.get("DB_PORT", "5433"),
    "-U", os.environ.get("DB_USER", "comic"),
    "-d", os.environ.get("DB_NAME", "comic"),
    "-tA", "-F", "\t",
]
ENV = dict(os.environ, PGPASSWORD=os.environ.get("DB_PASSWORD", "comic123"))

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

MAGIC = {
    b"\xff\xd8\xff": "jpg",
    b"\x89PNG\r\n\x1a\n": "png",
    b"GIF87a": "gif",
    b"GIF89a": "gif",
}


def psql(sql):
    return subprocess.run(PSQL_BASE + ["-c", sql], env=ENV,
                          capture_output=True, text=True, check=True).stdout


def sniff(path):
    """按 magic bytes 判定真实图片类型，避免把 HTML 错误页存成 .jpg。"""
    with open(path, "rb") as f:
        head = f.read(16)
    for sig, ext in MAGIC.items():
        if head.startswith(sig):
            return ext
    if head[:4] == b"RIFF" and head[8:12] == b"WEBP":
        return "webp"
    return None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true", help="只列出待本地化的行，不下载不改库")
    args = ap.parse_args()

    os.makedirs(DIR, exist_ok=True)
    sql = ("SELECT allcpp_id, cover_url FROM comic_events "
           "WHERE del_flag = false AND cover_url LIKE 'http%' "
           "ORDER BY allcpp_id")
    out = psql(sql)
    rows = [line.split("\t") for line in out.splitlines() if line.strip()]
    print(f"rows needing local cover: {len(rows)}")
    if args.dry_run or not rows:
        for allcpp_id, url in rows[:20]:
            print(f"  {allcpp_id}\t{url}")
        return 0

    ok = skip = fail = 0
    updates = []
    for allcpp_id, url in rows:
        url = url.strip()
        # 临时文件放在目标目录内：/tmp 与 /vol1 是不同设备，跨设备 rename 会
        # 报 OSError: [Errno 18] Invalid cross-device link。
        tmp = os.path.join(DIR, f".tmp_{allcpp_id}.bin")
        try:
            req = urllib.request.Request(url, headers={
                "User-Agent": UA,
                "Referer": "https://www.allcpp.cn/",
            })
            with urllib.request.urlopen(req, timeout=30) as r, open(tmp, "wb") as f:
                f.write(r.read())
        except Exception as e:  # noqa: BLE001
            print(f"FAIL download {allcpp_id} {url}: {e}", file=sys.stderr)
            fail += 1
            continue

        ext = sniff(tmp)
        if not ext:
            print(f"FAIL not an image {allcpp_id} {url}", file=sys.stderr)
            os.unlink(tmp)
            fail += 1
            continue

        name = f"{allcpp_id}.{ext}"
        dst = os.path.join(DIR, name)
        if os.path.exists(dst) and os.path.getsize(dst) > 0:
            os.unlink(tmp)
            skip += 1
        else:
            os.replace(tmp, dst)
            ok += 1
        updates.append((f"/static/remote/covers/{name}", url))

    print(f"downloaded {ok}, already-cached {skip}, failed {fail}")
    if updates:
        values = ",".join(
            "('" + p.replace("'", "''") + "','" + u.replace("'", "''") + "')"
            for p, u in updates
        )
        sql = ("UPDATE comic_events AS e SET cover_url = v.path, updated_at = NOW() "
               "FROM (VALUES " + values + ") AS v(path, url) WHERE e.cover_url = v.url")
        psql(sql)
        print(f"rewrote cover_url for {len(updates)} rows")
    return 1 if fail else 0


if __name__ == "__main__":
    sys.exit(main())
