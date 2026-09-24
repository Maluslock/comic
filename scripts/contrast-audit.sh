#!/usr/bin/env bash
# C 端对比度审计（可重复）。用法：bash scripts/contrast-audit.sh [OUT_DIR]
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:5173}"
API_URL="${API_URL:-http://127.0.0.1:8088}"
OUT_DIR="${1:-.audit/out}"
AB="${AGENT_BROWSER:-agent-browser}"

command -v "$AB" >/dev/null 2>&1 || export PATH="/home/user/.nvm/versions/node/v22.16.0/bin:$PATH"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PAYLOAD="$(cat "$SCRIPT_DIR/contrast-audit.js")"
mkdir -p "$OUT_DIR"

set_role() {  # $1=phone
  local resp token uid pid
  resp="$(curl -s -X POST "$API_URL/api/v1/login" -H 'Content-Type: application/json' \
    -d "{\"phone\":\"$1\",\"code\":\"123456\"}" --max-time 8)"
  token="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')"
  uid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"]["id"])')"
  pid="$(printf '%s' "$resp" | python3 -c 'import sys,json;print(json.load(sys.stdin)["user"].get("photographerId") or 0)')"
  # open 建立 origin，再用 reload 触发整页重载，否则 Pinia store 读不到注入的 token
  "$AB" open "$BASE_URL/#/pages/index/index" >/dev/null 2>&1
  "$AB" eval "(function(){uni.setStorageSync('token','$token');uni.setStorageSync('user',JSON.stringify({id:$uid,name:'audit',phone:'$1',avatar:'',bio:'',photographerId:$pid}));return 1;})()" >/dev/null 2>&1
}

audit_page() {  # $1=route
  local n; n="$(printf '%s' "$1" | tr '/?=' '___')"
  "$AB" open "$BASE_URL/#/$1" >/dev/null 2>&1
  "$AB" reload >/dev/null 2>&1
  sleep 3.2
  "$AB" eval "JSON.stringify({n:document.querySelectorAll('*').length,len:document.body.innerText.replace(/\s+/g,' ').trim().length})" > "$OUT_DIR/$n.meta" 2>&1
  "$AB" eval "$PAYLOAD" > "$OUT_DIR/$n.json" 2>&1
  printf '%-34s ' "$1"
  python3 - "$OUT_DIR/$n.json" "$OUT_DIR/$n.meta" <<'PY'
import json, sys
try:
    d = json.loads(json.loads(open(sys.argv[1]).read().strip()))
    print('total=%-3d worst=%-6s' % (d['total'], d['findings'][0]['ratio'] if d['findings'] else '-'), end=' ')
except Exception:
    print('PARSE-FAIL', end=' ')
try:
    print('textLen=%d' % json.loads(json.loads(open(sys.argv[2]).read().strip()))['len'])
except Exception:
    print('')
PY
}

COSER_PAGES="pages/index/index pages/event/detail?id=1 pages/event/list pages/photographer/list \
pages/photographer/detail?id=1 pages/search/search pages/order/list pages/order/detail?id=1 \
pages/order/detail?id=2 pages/message/index pages/profile/index pages/favorite/list pages/follow/list \
pages/comment/index pages/booking/index pages/portfolio/index pages/calendar/index pages/settings/index \
pages/login/index"

PHOTOGRAPHER_PAGES="pages/photographer/services pages/photographer/works pages/photographer/orders \
pages/photographer/activate pages/photographer/profile-edit pages/photographer/cert-apply"

echo "== role=coser =="
set_role 13800138000
for p in $COSER_PAGES; do audit_page "$p"; done

echo "== role=photographer =="
set_role 10000000001
for p in $PHOTOGRAPHER_PAGES; do audit_page "$p"; done
