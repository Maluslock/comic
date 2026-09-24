#!/usr/bin/env bash
# C 端对比度审计（可重复）。用法：bash scripts/contrast-audit.sh [OUT_DIR]  —— 注意 OUT_DIR 内的 *.json/*.meta 会先被清空
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:5173}"
API_URL="${API_URL:-http://127.0.0.1:8088}"
# 用 `-` 而非 `:-`：省略参数时才取默认值，显式传入空串则落到下面的守卫被拒绝
# （否则 `${1:-.audit/out}` 会把空串悄悄变成默认目录，守卫的 "" 分支永远不可达）。
OUT_DIR="${1-.audit/out}"
AB="${AGENT_BROWSER:-agent-browser}"

# 清理前拒绝会命中仓库根 / 上级 / 系统根的取值（见 Ruling T7-2a）。
case "$OUT_DIR" in
  ""|.|./|..|../|/) echo "FATAL: 拒绝 OUT_DIR='$OUT_DIR'（会删掉无关 JSON）" >&2; exit 2;;
esac
# 只允许写在 <repo>/.audit/ 之内（见 Ruling T7-2 Minor 3）。黑名单拦不住
# `admin`（会删 admin/package.json、admin/tsconfig.json）、`.omo/run-continuation`、
# `.oxfmtrc.json` 之类 —— 改成白名单：必须落在 <repo>/.audit/ 里。
#
# 注意顺序：审计目录通常**尚不存在**（脚本稍后才 `mkdir -p`），此时 `cd` 解析会失败，
# 不能因此误杀。所以先按字面量前缀判定「在 `<repo>/.audit/` 之下」（`..` 之类危险穿越
# 会在这一步被拒），目录已存在时再解析绝对路径，防止经由符号链接逃出 `.audit/`。
_repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
_audit_abs=""
[ -n "$_repo_root" ] && _audit_abs="$(cd "$_repo_root/.audit" 2>/dev/null && pwd -P || true)"
# 仓库/.audit 解析不出来时必须拒绝（fail-closed）：此时白名单没有基准，放行等于放行一切
[ -n "$_audit_abs" ] || {
  echo "FATAL: 无法解析 <repo>/.audit（不在 git 工作树内？）—— 拒绝清理" >&2; exit 2; }
case "$OUT_DIR" in
  .audit/*|"$_repo_root"/.audit/*) : ;;
  *) echo "FATAL: OUT_DIR='$OUT_DIR' 不在 <repo>/.audit/ 之内 —— 拒绝清理" >&2; exit 2;;
esac
# 解析真实路径，确认仍在 .audit/ 内（含符号链接逃逸防护）
_out_abs="$(cd "$OUT_DIR" 2>/dev/null && pwd -P || true)"
if [ -z "$_out_abs" ]; then
  _out_parent="$(dirname "$OUT_DIR")"
  _out_abs="$(cd "$_out_parent" 2>/dev/null && pwd -P || true)/$(basename "$OUT_DIR")"
fi
case "$_out_abs/" in
  "$_audit_abs"/*) : ;;
  *) echo "FATAL: OUT_DIR='$OUT_DIR' 解析为 '$_out_abs'，逃出 <repo>/.audit/ —— 拒绝清理" >&2; exit 2;;
esac

command -v "$AB" >/dev/null 2>&1 || export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PAYLOAD="$(cat "$SCRIPT_DIR/contrast-audit.js")"
mkdir -p "$OUT_DIR"
# 每次运行只保留本次页集：残留旧文件会污染后续对比（见 Ruling T7-1）
rm -f "$OUT_DIR"/*.json "$OUT_DIR"/*.meta

set_role() {  # $1=phone  $2=期望角色：photographer（pid≠0）| coser（pid=0）
  local resp token uid pid
  resp="$(curl -s -X POST "$API_URL/api/v1/login" -H 'Content-Type: application/json' \
    -d "{\"phone\":\"$1\",\"code\":\"123456\"}" --max-time 8)"
  token="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])' 2>/dev/null || true)"
  # 登录失败必须硬停：否则 token 为空、页面全部按**未登录态**测量，
  # 会产出一张看起来合理但完全错误的表（正是本项目两次栽过的假通过）。
  [ -n "$token" ] || {
    echo "FATAL: 账号 $1 登录失败（API 挂了或账号不对）—— 拒绝在未登录态下测量" >&2
    exit 1
  }
  uid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"]["id"])')"
  pid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"].get("photographerId") or 0)')"
  # 角色必须与请求的一致（见 Ruling T7-2 Blocking 2）：dev API 会把任意合法手机号
  # 自动注册成**普通用户**，所以打错的摄影师号会拿到 photographerId=0 —— 那样整轮
  # 摄影师侧页面都在错误/空角色下测量，却照样输出 total=0 + OK。宁可硬停。
  if [ "${2:-}" = "photographer" ] && [ "$pid" = "0" ]; then
    echo "FATAL: 账号 $1 不是摄影师（photographerId=0）—— 拒绝在错误角色下测量摄影师侧页面" >&2
    exit 1
  fi
  if [ "${2:-}" = "coser" ] && [ "$pid" != "0" ]; then
    echo "FATAL: 账号 $1 已开通摄影师身份（photographerId=$pid）—— 拒绝把它当 coser 用" >&2
    exit 1
  fi
  # open 建立 origin，再用 reload 触发整页重载，否则 Pinia store 读不到注入的 token
  "$AB" open "$BASE_URL/#/pages/index/index" >/dev/null 2>&1
  "$AB" eval "(function(){uni.setStorageSync('token','$token');uni.setStorageSync('user',JSON.stringify({id:$uid,name:'audit',phone:'$1',avatar:'',bio:'',photographerId:$pid}));return 1;})()" >/dev/null 2>&1
}

FAILED=0

audit_page() {  # $1=route  $2=role
  # 文件名带角色前缀：同一路由会在 coser 与 photographer 两轮各测一次
  # （如 pages/profile/index），无前缀会互相覆盖、diff 看不出差异。
  local n; n="$2__$(printf '%s' "$1" | tr '/?=' '___')"
  "$AB" open "$BASE_URL/#/$1" >/dev/null 2>&1
  "$AB" reload >/dev/null 2>&1
  sleep 3.2
  "$AB" eval "JSON.stringify({n:document.querySelectorAll('*').length,len:document.body.innerText.replace(/\s+/g,' ').trim().length})" > "$OUT_DIR/$n.meta" 2>&1
  "$AB" eval "$PAYLOAD" > "$OUT_DIR/$n.json" 2>&1
  printf '%-34s ' "$1"
  if ! python3 - "$OUT_DIR/$n.json" "$OUT_DIR/$n.meta" <<'PY'
import json, sys
ok = True
try:
    d = json.loads(json.loads(open(sys.argv[1]).read().strip()))
    print('total=%-3d exempt=%-2d worst=%-6s' % (
        d['total'], d.get('exempt', 0),
        d['findings'][0]['ratio'] if d['findings'] else '-'), end=' ')
except Exception:
    print('PARSE-FAIL', end=' '); ok = False
try:
    print('textLen=%d' % json.loads(json.loads(open(sys.argv[2]).read().strip()))['len'])
except Exception:
    print('(no meta)'); ok = False
sys.exit(0 if ok else 1)
PY
  then
    FAILED=$((FAILED + 1))
  fi
}

# order/detail 的 4 个状态变体都跑：青(pending)/渐变(confirmed)/绿(completed)/深底(cancelled)。
# 只跑 id=1/2 会看不见 .cancelled 被误改（深字压深底 1.07:1）。
COSER_PAGES="pages/index/index pages/event/detail?id=1 pages/event/list pages/photographer/list \
pages/photographer/detail?id=1 pages/search/search pages/order/list pages/order/detail?id=1 \
pages/order/detail?id=2 pages/order/detail?id=3 pages/order/detail?id=4 pages/message/index \
pages/profile/index pages/favorite/list pages/follow/list pages/comment/index pages/booking/index \
pages/portfolio/index pages/calendar/index pages/settings/index pages/login/index \
pages/chat/index pages/profile/edit pages/settings/doc pages/settings/security pages/settings/blocks"

PHOTOGRAPHER_PAGES="pages/photographer/services pages/photographer/works pages/photographer/orders \
pages/photographer/activate pages/photographer/profile-edit pages/photographer/cert-apply \
pages/profile/index"

echo "== role=coser =="
set_role 13800138000 coser
for p in $COSER_PAGES; do audit_page "$p" coser; done

echo "== role=photographer =="
set_role 10000000001 photographer
for p in $PHOTOGRAPHER_PAGES; do audit_page "$p" photographer; done

# 退出码必须是有意义的通过/失败信号：任何页面测量失败 → 本次运行无效。
if [ "$FAILED" -ne 0 ]; then
  echo "FATAL: $FAILED 页测量失败 —— 本次运行不是有效结论，勿据此判定通过" >&2
  exit 1
fi
echo "OK: 全部页面测量成功"
