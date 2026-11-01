#!/usr/bin/env python3
"""把 server/static/remote/covers/ 下的存量封面压到实际显示尺寸并转成 webp，
同步改写 comic_events.cover_url（以及 banners.image_url —— 见下方“为什么也改 banners”），
原图完整归档到 server/static/remote/_orig_covers/，可用 --restore 一键回滚。

为什么需要这一步：
    covers/ 里有 365 张原图，共 258MB，平均 721KB，最大的 3600x1800 / 1.7MB。
    小程序首屏（hero + 横滑 + 海报墙）一次就要拉 13 张，合计 9MB+，弱网下图片区一片空白。
    实际渲染宽度只有 750px 以内（卡片 ~330px、hero 整屏 ~750px），原来的 4400x2200 纯属浪费带宽。

转换规则（与验收口径一致）：
    * 最长边上限 750px，保持宽高比，**不放大**（原图已经小于 750 的，只做格式转换）；
    * webp quality 75，compression_level 6，lanczos 重采样；
    * 有真实透明像素的图用 yuva420p 保留 alpha，没有的走 yuv420p 省字节；
    * ffmpeg 固定 /usr/bin/ffmpeg，启动时校验 -encoders 里有 libwebp，否则直接退出。

为什么原图要归档而不是删除：
    压缩是一次性的破坏性操作，线上出问题必须能立刻退回。原图统一挪到
    _orig_covers/<原文件名>，并在 _orig_covers/.optimize-covers.manifest.json 里
    记录「原文件名 / 原字节 / 新字节 / DB 旧值」，--restore 靠它逐条反着做。

为什么也改 banners.image_url：
    banners 表里 3 行的 image_url 指向的正是本批 covers 文件（1006166/1006474/1006404），
    转换后同名 .jpg 会被移走，不改 banner 的话首页 hero 直接 404。这不是新增功能，
    是同一批文件的 URL 同步，属于本次数据迁移的一部分。

用法（在 comic/server 下执行，脚本用绝对路径定位，任何目录都可以）：
    python3 scripts/optimize-covers.py --dry-run              # 只算方案，不落盘不改库
    python3 scripts/optimize-covers.py --limit 10             # 先试 10 张
    python3 scripts/optimize-covers.py                        # 全量
    python3 scripts/optimize-covers.py --restore             # 完整回滚（文件 + DB）
    python3 scripts/optimize-covers.py --restore --dry-run   # 只看会回滚什么

可重复执行：已转换过的（manifest 里有记录，或 .jpg/.png 已经不在了而 .webp 在）直接跳过。

依赖：python3 标准库 + psql（PGPASSWORD）+ /usr/bin/ffmpeg + /usr/bin/ffprobe。
"""
import argparse
import json
import os
import socket
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
SERVER = os.path.dirname(HERE)
REMOTE = os.path.join(SERVER, "static", "remote")
COVERS = os.path.join(REMOTE, "covers")
ARCHIVE = os.path.join(REMOTE, "_orig_covers")
# 36 个老文件（2026-10-28 之前的一批小尺寸原图）与本次要归档的原图同名但字节不同，
# 直接覆盖会丢数据，所以先把它们挪进 .pre_archive/，--restore 时再挪回去。
PRE_ARCHIVE = os.path.join(ARCHIVE, ".pre_archive")
MANIFEST = os.path.join(ARCHIVE, ".optimize-covers.manifest.json")

FFMPEG = "/usr/bin/ffmpeg"
FFPROBE = "/usr/bin/ffprobe"

MAX_EDGE = 750
QUALITY = 75
COMPRESSION_LEVEL = 6
SRC_EXTS = (".jpg", ".jpeg", ".png", ".webp")
# 认为“真的有透明像素”的阈值：alpha 通道最小值低于它就保留 alpha。
# 多数 rgba PNG 的 alpha 全 255（只是有通道而已），当成不透明处理能再省 20%~30% 字节。
ALPHA_YMIN_THRESHOLD = 250

REPORT_PATH = "/tmp/cover-report.txt"

PSQL_BASE = [
    "psql",
    # -q：抑制 "UPDATE 3" 这类 command tag。UPDATE ... RETURNING 在 -tA 下仍会打 tag，
    # 不抑制的话 psql_query 会把 tag 当成一行数据，行数统计就多 1。
    "-q",
    "-h", os.environ.get("DB_HOST", "127.0.0.1"),
    "-p", os.environ.get("DB_PORT", "5433"),
    "-U", os.environ.get("DB_USER", "comic"),
    "-d", os.environ.get("DB_NAME", "comic"),
    "-tA", "-F", "\t",
]
PGENV = dict(os.environ, PGPASSWORD=os.environ.get("DB_PASSWORD", "comic123"))

REDIS_HOST = os.environ.get("REDIS_HOST", "127.0.0.1")
REDIS_PORT = int(os.environ.get("REDIS_PORT", "6380"))

URL_PREFIX = "/static/remote/covers/"


def human(n):
    for unit in ("B", "KB", "MB", "GB"):
        if abs(n) < 1024 or unit == "GB":
            return f"{n:.0f}{unit}" if unit == "B" else f"{n:.1f}{unit}"
        n /= 1024.0


def psql(sql):
    r = subprocess.run(PSQL_BASE + ["-c", sql], env=PGENV, capture_output=True, text=True)
    if r.returncode != 0:
        raise RuntimeError(f"psql failed: {r.stderr.strip()}")
    return r.stdout


def psql_query(sql):
    out = psql(sql)
    rows = []
    for line in out.splitlines():
        if not line.strip():
            continue
        # 兜底：即使 psql 版本忽略 -q 打了 command tag，也不要把 (UPDATE n) 当数据行。
        if line.startswith(("UPDATE ", "INSERT ", "SELECT ", "DELETE ")) and line.split(" ")[-1].isdigit():
            continue
        rows.append(line.split("\t"))
    return rows


def check_ffmpeg():
    if not os.path.exists(FFMPEG):
        sys.exit(f"FATAL: {FFMPEG} 不存在")
    r = subprocess.run([FFMPEG, "-hide_banner", "-encoders"], capture_output=True, text=True)
    if "libwebp" not in r.stdout:
        sys.exit(f"FATAL: {FFMPEG} 没有 libwebp 编码器，无法输出 webp")


def probe(path):
    """返回 (width, height, pix_fmt)；读不出来返回 None。"""
    r = subprocess.run(
        [FFPROBE, "-v", "error", "-select_streams", "v:0",
         "-show_entries", "stream=width,height,pix_fmt", "-of", "json", path],
        capture_output=True, text=True,
    )
    if r.returncode != 0:
        return None
    try:
        streams = json.loads(r.stdout).get("streams") or []
    except json.JSONDecodeError:
        return None
    if not streams:
        return None
    st = streams[0]
    w, h = st.get("width"), st.get("height")
    if not w or not h:
        return None
    return int(w), int(h), st.get("pix_fmt") or ""


def has_real_alpha(path, pix_fmt):
    if "a" not in pix_fmt and pix_fmt != "pal8":
        return False
    r = subprocess.run(
        [FFMPEG, "-nostdin", "-v", "error", "-i", path,
         "-vf", "alphaextract,signalstats,metadata=print:file=-", "-f", "null", "-"],
        capture_output=True, text=True,
    )
    for line in r.stdout.splitlines():
        if "lavfi.signalstats.YMIN" in line:
            try:
                return int(line.rsplit("=", 1)[1]) < ALPHA_YMIN_THRESHOLD
            except ValueError:
                return True
    return True


def scale_filter(w, h):
    """最长边压到 MAX_EDGE 以内，且不放大。

    min(750, iw) 保证原图比 750 小时原样输出；-2 让 ffmpeg 按比例算另一边并对齐到偶数
    （yuv420p / yuva420p 都要求偶数边）。
    """
    return (
        f"scale=w='if(gte(iw,ih),min({MAX_EDGE},iw),-2)'"
        f":h='if(gte(iw,ih),-2,min({MAX_EDGE},ih))':flags=lanczos"
    )


def convert(src, dst, keep_alpha):
    vf = scale_filter(*probe(src)[:2])
    cmd = [
        FFMPEG, "-nostdin", "-v", "error", "-y", "-i", src,
        "-vf", vf, "-map_metadata", "-1",
        "-c:v", "libwebp", "-quality", str(QUALITY),
        "-compression_level", str(COMPRESSION_LEVEL),
        "-pix_fmt", "yuva420p" if keep_alpha else "yuv420p",
        dst,
    ]
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        return None, (r.stderr or "").strip()[:300]
    info = probe(dst)
    if not info:
        return None, "输出文件无法被 ffprobe 解析"
    return info, None


def redis_cmd(*args):
    """不依赖 redis 模块：手写 RESP，发一条命令读一条回复（够用就行）。"""
    payload = f"*{len(args)}\r\n".encode()
    for a in args:
        b = str(a).encode()
        payload += b"$%d\r\n%s\r\n" % (len(b), b)
    with socket.create_connection((REDIS_HOST, REDIS_PORT), timeout=5) as s:
        s.sendall(payload)
        return s.recv(4096)


def flush_cache():
    """首页/列表接口是 5 分钟 TTL 的 Redis 缓存，改完 DB 不清就会继续吐老 URL。"""
    try:
        out = []
        for key in ("home",):
            out.append(f"DEL {key}: {redis_cmd('DEL', key).decode(errors='replace').strip()}")
        for pattern in ("events:*", "photographers:*"):
            cursor = b"0"
            deleted = 0
            for _ in range(50):
                reply = redis_cmd("SCAN", cursor, "MATCH", pattern, "COUNT", 500)
                parts = reply.decode(errors="replace").split("\r\n")
                if len(parts) < 3:
                    break
                cursor = parts[1].encode()
                keys = [k for k in parts[3:] if k and k != "*1"]
                if keys:
                    deleted += int(redis_cmd("DEL", *keys).decode(errors="replace").strip().lstrip(":"))
                if cursor == b"0":
                    break
            out.append(f"DEL {pattern}: {deleted} 个 key")
        return True, "; ".join(out)
    except Exception as e:  # noqa: BLE001
        return False, f"清缓存失败（服务端会等 5 分钟 TTL 自然过期）：{e}"


def load_manifest():
    if not os.path.exists(MANIFEST):
        return {"version": 1, "entries": []}
    with open(MANIFEST, encoding="utf-8") as f:
        return json.load(f)


def save_manifest(m):
    os.makedirs(ARCHIVE, exist_ok=True)
    tmp = MANIFEST + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(m, f, ensure_ascii=False, indent=1)
        f.write("\n")
    os.replace(tmp, MANIFEST)


def plan(db_rows, manifest):
    # 幂等：原图文件名与产出的 webp 名都算「已处理」，
    # 否则第二次运行会把上一轮生成的 .webp 当新品再压一遍（越压越糊、越压越小但没意义）。
    done = {e["name"] for e in manifest["entries"]}
    done |= {e["webp"] for e in manifest["entries"] if e.get("webp")}
    items = []
    for name in sorted(os.listdir(COVERS)):
        lower = name.lower()
        if not lower.endswith(SRC_EXTS) or name.startswith("."):
            continue
        base, ext = os.path.splitext(name)
        src = os.path.join(COVERS, name)
        if name in done:
            items.append({"name": name, "status": "skip-done"})
            continue
        if not os.path.isfile(src) or os.path.getsize(src) == 0:
            items.append({"name": name, "status": "skip-empty"})
            continue
        info = probe(src)
        if not info:
            items.append({"name": name, "status": "skip-broken"})
            continue
        w, h, pix_fmt = info
        target = base + ".webp"
        items.append({
            "name": name,
            "src": src,
            "ext": ext,
            "base": base,
            "target": target,
            "dims": (w, h),
            "pix_fmt": pix_fmt,
            "bytes": os.path.getsize(src),
            "db_url": db_rows.get(name),
            "status": "todo",
        })
    return items


def run_convert(items, limit, manifest):
    created = []
    failed = []
    total_before = total_after = 0
    processed = 0

    os.makedirs(ARCHIVE, exist_ok=True)
    todos = [it for it in items if it["status"] == "todo"]
    if limit:
        todos = todos[:limit]

    for it in todos:
        name = it["name"]
        src = it["src"]
        target = it["target"]
        dst = os.path.join(COVERS, target)
        # 临时文件必须落在同一文件系统：/tmp 与 /vol1 跨设备 os.replace 会 Errno 18。
        tmp = os.path.join(COVERS, f".opt_tmp_{it['base']}.webp")

        keep_alpha = has_real_alpha(src, it["pix_fmt"])
        before = it["bytes"]
        info, err = convert(src, tmp, keep_alpha)
        if err or not os.path.exists(tmp) or os.path.getsize(tmp) == 0:
            if os.path.exists(tmp):
                os.unlink(tmp)
            it["status"] = "failed"
            it["error"] = err or "输出为空"
            failed.append(it)
            print(f"FAIL {name}: {it['error']}", file=sys.stderr)
            continue

        after = os.path.getsize(tmp)
        # webp 不一定更小：小于 1KB 的图或者已经是 webp 再压可能反而变大。
        # 变大就保留原图（不转换、不动 DB），计入报告，但不删任何东西。
        if after >= before and it["ext"] != ".webp":
            os.unlink(tmp)
            it["status"] = "skip-bigger"
            print(f"SKIP {name}: webp {after}B >= 原图 {before}B，保留原图")
            continue

        w0, h0 = it["dims"]
        w1, h1 = info[0], info[1]
        ratio0 = w0 / h0
        ratio1 = w1 / h1
        if max(w0, h0) > MAX_EDGE and max(w1, h1) > MAX_EDGE:
            os.unlink(tmp)
            it["status"] = "failed"
            it["error"] = f"输出仍然超过 {MAX_EDGE}px: {w1}x{h1}"
            failed.append(it)
            continue
        if abs(ratio0 - ratio1) / ratio0 > 0.03:
            os.unlink(tmp)
            it["status"] = "failed"
            it["error"] = f"宽高比变化过大 {w0}x{h0} -> {w1}x{h1}"
            failed.append(it)
            continue

        # 归档原图：同名老文件先进 .pre_archive/，绝不覆盖丢数据。
        archived = os.path.join(ARCHIVE, name)
        pre = None
        if os.path.exists(archived):
            os.makedirs(PRE_ARCHIVE, exist_ok=True)
            pre = os.path.join(PRE_ARCHIVE, name)
            os.replace(archived, pre)

        os.replace(src, archived)          # 原图搬家（同设备 rename，快且原子）
        os.replace(tmp, dst)               # 新 webp 落位

        entry = {
            "name": name,
            "base": it["base"],
            "ext": it["ext"],
            "webp": target,
            "archived": os.path.relpath(archived, REMOTE),
            "pre_archive": os.path.relpath(pre, REMOTE) if pre else None,
            "orig_bytes": before,
            "new_bytes": after,
            "orig_dims": [w0, h0],
            "new_dims": [w1, h1],
            "alpha": keep_alpha,
            "db_url": it.get("db_url"),
            "cache_busted": True,
        }
        manifest["entries"].append(entry)
        created.append(entry)
        it["status"] = "done"
        it["new_bytes"] = after
        it["new_dims"] = (w1, h1)
        total_before += before
        total_after += after
        processed += 1
        pct = (1 - after / before) * 100 if before else 0
        print(f"OK   {name:>24} {before:>9}B -> {after:>7}B ({pct:5.1f}% down)  {w0}x{h0} -> {w1}x{h1}"
              + ("  [alpha]" if keep_alpha else ""))

    return created, failed, total_before, total_after, processed


def rewrite_db(entries, dry_run=False):
    """只改真正转换成功的行；old -> new 精确匹配，匹配不上就不动。

    统计口径是「真的被改写了几行」（UPDATE ... RETURNING 回读），不是「提交了多少个 mapping」——
    否则报告里会写出 banners 改写 363 行这种假数字（banners 总共才 3 行）。
    """
    ev = []
    banners = []
    for e in entries:
        if not e.get("db_url"):
            continue
        old = e["db_url"]
        new = URL_PREFIX + e["webp"]
        ev.append((old, new))
        banners.append((old, new))

    updated_events = updated_banners = 0
    if ev:
        values = ",".join(
            "('" + o.replace("'", "''") + "','" + n.replace("'", "''") + "')" for o, n in ev
        )
        if dry_run:
            rows = psql_query(
                "SELECT count(*) FROM comic_events e JOIN (VALUES " + values +
                ") AS v(old, new) ON e.cover_url = v.old"
            )
            updated_events = int(rows[0][0]) if rows else 0
        else:
            updated_events = len(psql_query(
                "UPDATE comic_events AS e SET cover_url = v.new, updated_at = NOW() "
                "FROM (VALUES " + values + ") AS v(old, new) "
                "WHERE e.cover_url = v.old RETURNING e.id"
            ))
    if banners:
        values = ",".join(
            "('" + o.replace("'", "''") + "','" + n.replace("'", "''") + "')" for o, n in banners
        )
        if dry_run:
            rows = psql_query(
                "SELECT count(*) FROM banners b JOIN (VALUES " + values +
                ") AS v(old, new) ON b.image_url = v.old"
            )
            updated_banners = int(rows[0][0]) if rows else 0
        else:
            updated_banners = len(psql_query(
                "UPDATE banners AS b SET image_url = v.new, updated_at = NOW() "
                "FROM (VALUES " + values + ") AS v(old, new) "
                "WHERE b.image_url = v.old RETURNING b.id"
            ))
    return updated_events, updated_banners


def restore(manifest, dry_run):
    entries = manifest.get("entries") or []
    if not entries:
        print("manifest 里没有记录，没什么可回滚的。")
        return 0
    db_back = []
    restored = 0
    for e in entries:
        archived = os.path.join(REMOTE, e["archived"])
        dst = os.path.join(COVERS, e["name"])
        webp = os.path.join(COVERS, e["webp"])
        if not os.path.exists(archived) and os.path.exists(dst):
            continue  # 已经还原过了
        if not os.path.exists(archived):
            print(f"WARN 归档缺失，跳过 {e['name']}", file=sys.stderr)
            continue
        if e.get("db_url"):
            # 顺序必须是 (原 url, 当前 webp url)：下面 SQL 是 SET x = v.old WHERE x = v.new。
            db_back.append((e["db_url"], URL_PREFIX + e["webp"]))
        if dry_run:
            print(f"RESTORE {e['name']} <- {e['archived']}  (删 {e['webp']})")
            restored += 1
            continue
        if os.path.exists(webp) and e["webp"] != e["name"]:
            os.unlink(webp)
        os.replace(archived, dst)
        if e.get("pre_archive"):
            pre = os.path.join(REMOTE, e["pre_archive"])
            if os.path.exists(pre):
                os.replace(pre, archived)
        restored += 1

    if dry_run:
        print(f"（dry-run）将还原 {restored} 个文件，回滚 {len(db_back)} 行 DB")
        return 0

    if db_back:
        values = ",".join(
            "('" + o.replace("'", "''") + "','" + n.replace("'", "''") + "')" for o, n in db_back
        )
        psql("UPDATE comic_events AS e SET cover_url = v.old, updated_at = NOW() "
             "FROM (VALUES " + values + ") AS v(old, new) WHERE e.cover_url = v.new")
        psql("UPDATE banners AS b SET image_url = v.old, updated_at = NOW() "
             "FROM (VALUES " + values + ") AS v(old, new) WHERE b.image_url = v.new")
    save_manifest({"version": 1, "entries": []})
    try:
        if os.path.isdir(PRE_ARCHIVE) and not os.listdir(PRE_ARCHIVE):
            os.rmdir(PRE_ARCHIVE)
    except OSError:
        pass
    ok, detail = flush_cache()
    print(f"已还原 {restored} 个文件、回滚 {len(db_back)} 行 DB；{'缓存已清: ' if ok else ''}{detail}")
    return 0


def first_screen(api_base="http://127.0.0.1:8088"):
    """首屏口径：hero(banners[0]) + /api/v1/home 前 12 条 upcomingEvents。"""
    import urllib.request
    try:
        with urllib.request.urlopen(f"{api_base}/api/v1/home", timeout=8) as r:
            data = json.loads(r.read().decode())
    except Exception as e:  # noqa: BLE001
        return None, f"取 /api/v1/home 失败：{e}"
    urls = []
    banners = data.get("banners") or []
    if banners:
        urls.append(banners[0].get("imageUrl"))
    for ev in (data.get("upcomingEvents") or [])[:12]:
        urls.append(ev.get("coverUrl"))
    rows = []
    total = 0
    for u in urls:
        if not u:
            continue
        p = os.path.join(SERVER, u.lstrip("/"))
        size = os.path.getsize(p) if os.path.exists(p) else -1
        rows.append((u, size))
        if size > 0:
            total += size
    return (rows, total), None


def write_report(items, entries, failed, created_count, first, elapsed):
    """entries 传 manifest 全量记录 —— 分多次（--limit）跑时报告也要能看到全部 365 张的对照，
    只在报告里写「本轮 N 张」会让后来接手的人以为只压了一部分。"""
    lines = []
    lines.append("# 封面压缩前后对比（optimize-covers.py）")
    lines.append(f"# 生成时间: {time.strftime('%Y-%m-%d %H:%M:%S')}  耗时 {elapsed:.1f}s")
    lines.append(f"# 转换规则: 最长边 <= {MAX_EDGE}px / libwebp quality {QUALITY} / compression_level {COMPRESSION_LEVEL}")
    lines.append("")
    lines.append(f"{'序号':>4}  {'文件名':<26} {'原大小':>10} {'新大小':>10} {'压缩比':>8}  {'原尺寸':>12} -> {'新尺寸':<12}")
    lines.append("-" * 96)
    idx = 0
    conv_before = conv_after = 0
    for e in entries:
        idx += 1
        pct = (1 - e["new_bytes"] / e["orig_bytes"]) * 100 if e["orig_bytes"] else 0
        conv_before += e["orig_bytes"]
        conv_after += e["new_bytes"]
        lines.append(
            f"{idx:>4}  {e['name']:<26} {human(e['orig_bytes']):>10} {human(e['new_bytes']):>10} "
            f"{pct:>7.1f}%  {str(e['orig_dims'][0]) + 'x' + str(e['orig_dims'][1]):>12} -> "
            f"{str(e['new_dims'][0]) + 'x' + str(e['new_dims'][1]):<12}"
            + ("  [alpha]" if e.get("alpha") else "")
        )
    lines.append("-" * 96)
    pct = (1 - conv_after / conv_before) * 100 if conv_before else 0
    note = f"（本次新增 {created_count} 张）" if created_count is not None else ""
    lines.append(f"合计 {len(entries)} 张：{human(conv_before)} -> {human(conv_after)}（省 {pct:.1f}%）{note}")
    if failed:
        lines.append("")
        lines.append(f"转换失败 {len(failed)} 张（这些行保持原值、原图未动）:")
        for it in failed:
            lines.append(f"    {it['name']}: {it.get('error', 'unknown')}")
    skips = [it for it in items if it["status"].startswith("skip")]
    if skips:
        by = {}
        for it in skips:
            by.setdefault(it["status"], []).append(it["name"])
        lines.append("")
        lines.append("跳过:")
        for k, v in by.items():
            lines.append(f"    {k}: {len(v)} 张 {v[:5]}{' ...' if len(v) > 5 else ''}")
    lines.append("")
    if first:
        rows, total = first
        lines.append(f"首屏 13 张（hero banners[0] + upcomingEvents 前 12）:")
        for u, size in rows:
            lines.append(f"    {'MISS' if size < 0 else human(size):>9}  {u}")
        lines.append(f"    合计: {human(total)}（{total} B），目标 <= 800KB -> {'PASS' if total <= 800 * 1024 else 'FAIL'}")
    lines.append("")
    total_dir = sum(
        os.path.getsize(os.path.join(COVERS, f))
        for f in os.listdir(COVERS) if os.path.isfile(os.path.join(COVERS, f))
    )
    n_files = len([f for f in os.listdir(COVERS) if os.path.isfile(os.path.join(COVERS, f))])
    leftover = [f for f in os.listdir(COVERS) if f.lower().endswith((".jpg", ".jpeg", ".png"))]
    lines.append(f"covers/ 目录当前: {n_files} 个文件, {human(total_dir)}（目标 <= 40MB -> "
                 f"{'PASS' if total_dir <= 40 * 1024 * 1024 else 'FAIL'}）")
    lines.append(f"残留 .jpg/.jpeg/.png: {len(leftover)} 个 {leftover[:10]}")
    with open(REPORT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")
    return {"conv_before": conv_before, "conv_after": conv_after, "dir_total": total_dir,
            "n_files": n_files, "leftover": leftover}


def main():
    global REPORT_PATH
    ap = argparse.ArgumentParser(description="存量封面压缩：限宽 750 + webp + 回写 DB + 原图归档")
    ap.add_argument("--dry-run", action="store_true", help="只列出方案，不写文件、不改库")
    ap.add_argument("--limit", type=int, default=0, help="本次最多处理 N 张（0 = 全部）")
    ap.add_argument("--restore", action="store_true", help="回滚：还原原图 + 回滚 DB cover_url")
    ap.add_argument("--no-cache-flush", action="store_true", help="不主动清 Redis 首页/列表缓存")
    ap.add_argument("--report", default=REPORT_PATH, help=f"报告路径（默认 {REPORT_PATH}）")
    args = ap.parse_args()

    REPORT_PATH = args.report

    check_ffmpeg()
    if not os.path.isdir(COVERS):
        sys.exit(f"FATAL: 目录不存在 {COVERS}")
    manifest = load_manifest()

    if args.restore:
        return restore(manifest, args.dry_run)

    # DB 现状：covers 下的 URL -> 它属于哪一行
    rows = psql_query(
        "SELECT cover_url FROM comic_events WHERE cover_url LIKE '" + URL_PREFIX + "%' "
        "UNION SELECT image_url FROM banners WHERE image_url LIKE '" + URL_PREFIX + "%'"
    )
    db_urls = {}
    for (u,) in rows:
        if u:
            db_urls[os.path.basename(u)] = u
    if not rows:
        print("WARN: DB 里没有 /static/remote/covers/ 的引用，只做文件转换", file=sys.stderr)

    items = plan(db_urls, manifest)
    todos = [it for it in items if it["status"] == "todo"]
    already = len(items) - len(todos)
    print(f"covers/ 待处理 {len(todos)} 张，已转换/跳过 {already} 张"
          + (f"，本次 --limit {args.limit}" if args.limit else ""))

    if args.dry_run:
        shows = todos[: (args.limit or 20)]
        for it in shows:
            w, h = it["dims"]
            print(f"  PLAN {it['name']:>26} {human(it['bytes']):>9} {w}x{h}"
                  f"  -> {it['target']}   db={it.get('db_url') or '-'}")
        if len(todos) > len(shows):
            print(f"  ... 还有 {len(todos) - len(shows)} 张")
        est = sum(min(it["bytes"], it["bytes"] * 0.25) for it in todos)
        print(f"预测: {len(todos)} 张 {human(sum(i['bytes'] for i in todos))} -> 约 {human(est)}")
        return 0

    t0 = time.time()
    created, failed, before, after, processed = run_convert(items, args.limit, manifest)
    elapsed = time.time() - t0

    if created:
        save_manifest(manifest)
        ev, bn = rewrite_db(created)
        print(f"DB: comic_events 改写 {ev} 行, banners 改写 {bn} 行")
        if not args.no_cache_flush:
            ok, detail = flush_cache()
            print(("缓存已清: " if ok else "WARN ") + detail)

    first, ferr = first_screen()
    if ferr:
        print(f"WARN 首屏统计失败：{ferr}", file=sys.stderr)
    stats = write_report(items, manifest["entries"], failed, len(created), first, elapsed)
    print(f"报告: {REPORT_PATH}")
    print(f"目录: {stats['n_files']} 个文件 {human(stats['dir_total'])}, "
          f"残留 jpg/png {len(stats['leftover'])} 个")
    if first:
        rows_, total = first
        print(f"首屏 13 张合计 {human(total)} ({total} B) "
              f"-> {'PASS' if total <= 800 * 1024 else 'FAIL'}")
    if failed:
        print(f"失败 {len(failed)} 张：{[f['name'] for f in failed]}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
