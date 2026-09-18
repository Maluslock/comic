#!/usr/bin/env bash
# C 端 API 回归冒烟：覆盖鉴权/伪造身份/约拍状态机/收藏关注/聊天/黑名单/通知/账号安全。
# 用法：
#   bash server/scripts/smoke.sh                                 # 默认打 dev :8088
#   BASE_URL=http://127.0.0.1 DB_CONTAINER=deploy-postgres-1 \
#     bash server/scripts/smoke.sh                               # 打 prod（经 nginx，源站即可）
#
# BASE_URL 只填源站（脚本自行拼 /api/v1/...）。
# 可重复执行：脚本用 SMOKE 标记创建数据，并在结束/中断时自动清理（需可访问 DB 容器）。
set -uo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8088}"
DB_CONTAINER="${DB_CONTAINER:-comic-postgres}"
DB_USER="${DB_USER:-comic}"
DB_NAME="${DB_NAME:-comic}"
CODE="${LOGIN_CODE:-123456}"

COSER_PHONE="${COSER_PHONE:-13800138000}"
PHOTO2_PHONE="${PHOTO2_PHONE:-10000000002}"
PHOTO5_PHONE="${PHOTO5_PHONE:-13700000005}"
USER1_PHONE="${USER1_PHONE:-10000000001}"

SLOT_DATE="${SLOT_DATE:-2026-12-30}"
SLOT_TIME="${SLOT_TIME:-10:00}"
SEARCH_DATE_TAG="日系"

PASS=0
FAIL=0
declare -a FAILED_LIST=()

if [ -t 1 ]; then G=$'\033[32m'; R=$'\033[31m'; D=$'\033[2m'; Z=$'\033[0m'; else G=; R=; D=; Z=; fi

pass() { PASS=$((PASS + 1)); printf "  ${G}✓${Z} %s\n" "$1"; }
fail() { FAIL=$((FAIL + 1)); FAILED_LIST+=("$1"); printf "  ${R}✗${Z} %s\n" "$1"; }
section() { printf "\n${D}── %s${Z}\n" "$1"; }

login() {
  curl -s -X POST "$BASE_URL/api/v1/login" -H 'Content-Type: application/json' \
    -d "{\"phone\":\"$1\",\"code\":\"$CODE\"}" | jq -r '.token // empty'
}

code_of() { curl -s -o /dev/null -w '%{http_code}' "$@"; }

expect() {
  local want="$1" desc="$2"
  shift 2
  local got
  got=$(code_of "$@")
  if [ "$got" = "$want" ]; then pass "$desc [$got]"; else fail "$desc (want $want, got $got)"; fi
}

expect_true() {
  local desc="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then pass "$desc"; else fail "$desc (want $want, got $got)"; fi
}

db_ok() { docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc 'SELECT 1' >/dev/null 2>&1; }

dbq() { docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1" 2>/dev/null; }

cleanup() {
  dbq "DELETE FROM notifications WHERE content LIKE '%#SMOKE%' OR content LIKE '%SMOKE%';
       DELETE FROM bookings WHERE remarks LIKE 'SMOKE%';
       DELETE FROM reviews WHERE content LIKE 'SMOKE%';
       DELETE FROM chat_messages WHERE content LIKE 'SMOKE%';
       DELETE FROM user_blocks;" >/dev/null 2>&1
}
trap 'cleanup; echo; [ "$FAIL" -eq 0 ] && echo "PASS=$PASS FAIL=0" || echo "PASS=$PASS FAIL=$FAIL"' EXIT

printf "${D}C-end smoke → %s${Z}\n" "$BASE_URL"

if ! db_ok; then
  printf "${R}警告${Z}: 无法访问 DB 容器 %s，脚本创建的数据不会被清理（重复执行可能因时段冲突失败）\n" "$DB_CONTAINER"
fi

section "登录测试账号"
T_COSER=$(login "$COSER_PHONE")
T_P2=$(login "$PHOTO2_PHONE")
T_P5=$(login "$PHOTO5_PHONE")
T_U1=$(login "$USER1_PHONE")
for v in T_COSER T_P2 T_P5 T_U1; do
  if [ -n "${!v}" ]; then pass "$v 登录成功"; else fail "$v 登录失败（检查手机号/验证码）"; fi
done
if [ -z "$T_COSER" ] || [ -z "$T_P2" ]; then
  printf "${R}关键账号登录失败，终止。${Z}\n"
  exit 1
fi

COSER_ID=$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/me" | jq -r '.id // empty')
P2_ID=$(curl -s -H "Authorization: Bearer $T_P2" "$BASE_URL/api/v1/me" | jq -r '.id // empty')
P5_ID=$(curl -s -H "Authorization: Bearer $T_P5" "$BASE_URL/api/v1/me" | jq -r '.id // empty')
expect_true "coser id 解析" "$([ -n "$COSER_ID" ] && echo ok)" "ok"

# 不硬编码摄影师的 id/用户：从测试账号动态解析，使脚本对任意数据集成立
PID=$(curl -s -H "Authorization: Bearer $T_P2" "$BASE_URL/api/v1/photographers/by-user/$P2_ID" | jq -r '.id // empty')
P2_USER=$(curl -s "$BASE_URL/api/v1/photographers/$PID" | jq -r '.userId // empty')
expect_true "测试账号已开通摄影师（photographerId=$PID）" "$([ -n "$PID" ] && echo ok)" "ok"
expect_true "摄影师已关联用户（userId=$P2_USER）" "$([ -n "$P2_USER" ] && echo ok)" "ok"
if [ -z "$PID" ] || [ -z "$P2_USER" ]; then
  printf "${R}测试账号 %s 不是（或未关联用户）摄影师，无法继续摄影师侧用例。${Z}\n" "$PHOTO2_PHONE"
  exit 1
fi

cleanup

section "A. 鉴权：无 token 应 401"
expect 401 "GET  /bookings/$P2_USER" "$BASE_URL/api/v1/bookings/$P2_USER"
expect 401 "POST /bookings" -X POST "$BASE_URL/api/v1/bookings" -H 'Content-Type: application/json' -d '{"photographerId":1,"serviceId":1,"date":"2026-12-31","time":"09:00"}'
expect 401 "POST /reviews" -X POST "$BASE_URL/api/v1/reviews" -H 'Content-Type: application/json' -d '{"photographerId":1,"rating":5,"content":"x"}'
expect 401 "GET  /follows/$P2_USER" "$BASE_URL/api/v1/follows/$P2_USER"
expect 401 "POST /follows" -X POST "$BASE_URL/api/v1/follows" -H 'Content-Type: application/json' -d '{"eventId":1}'
expect 401 "POST /blocks" -X POST "$BASE_URL/api/v1/blocks" -H 'Content-Type: application/json' -d '{"blockedUserId":1}'
expect 401 "POST /me/logout-all" -X POST "$BASE_URL/api/v1/me/logout-all"
expect 401 "GET  /notifications/1" "$BASE_URL/api/v1/notifications/1"

section "B. 鉴权：跨用户应 403"
expect 403 "coser 读 user1 的收藏" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/favorites/$P2_USER"
expect 403 "coser 读 user1 的订单" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/bookings/$P2_USER"
expect 403 "coser 读 user1 的关注" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/follows/$P2_USER"
expect 403 "coser 读 user1 的会话" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/chat/sessions/$P2_USER"
expect 403 "photo5 看 photo2 接单面板" -H "Authorization: Bearer $T_P5" "$BASE_URL/api/v1/bookings/photographer/$PID"

section "C. 伪造身份应被忽略（以 token 为准）"
expect 201 "伪造 userId=1 收藏（应记到 coser 名下）" -X POST "$BASE_URL/api/v1/favorites" \
  -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' -d '{"userId":1,"photographerId":3}'
expect_true "收藏落在 coser 名下" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/favorites/$COSER_ID" | jq -r '[.[]?|select(.photographerId==3)]|length')" "1"
expect 204 "清理该收藏" -X DELETE "$BASE_URL/api/v1/favorites/$COSER_ID/3" -H "Authorization: Bearer $T_COSER"

expect 201 "伪造 userId=1/userName 发评价" -X POST "$BASE_URL/api/v1/reviews" \
  -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' \
  -d '{"photographerId":1,"userId":1,"userName":"SMOKE-冒充者","rating":5,"content":"SMOKE-review"}'
expect_true "评价 userId 记为 coser" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/reviews/mine" | jq -r --argjson cid "$COSER_ID" '[.[]?|select(.content=="SMOKE-review")][0].userId == $cid')" "true"

section "D. 约拍状态机"
expect 200 "时段查询" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/photographers/$PID/timeslots?date=$SLOT_DATE"
BOOK=$(curl -s -X POST "$BASE_URL/api/v1/bookings" -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' \
  -d "{\"photographerId\":$PID,\"coserId\":1,\"serviceId\":1,\"date\":\"$SLOT_DATE\",\"time\":\"$SLOT_TIME\",\"remarks\":\"SMOKE-booking\"}")
BID=$(echo "$BOOK" | jq -r '.id // empty')
expect_true "下单 201 且金额=服务价 399" "$(echo "$BOOK" | jq -r '.totalPrice // "x"')" "399"
expect_true "伪造 coserId=1 被忽略" "$(echo "$BOOK" | jq -r '.coserId // "x"')" "$COSER_ID"
expect 409 "同档重复下单" -X POST "$BASE_URL/api/v1/bookings" -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' \
  -d "{\"photographerId\":$PID,\"serviceId\":1,\"date\":\"$SLOT_DATE\",\"time\":\"$SLOT_TIME\"}"
expect 403 "coser 冒充摄影师确认" -X PUT "$BASE_URL/api/v1/bookings/$BID/status" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d '{"status":"confirmed","actorTag":"photographer"}'
expect 403 "photo5 越权确认 photo2 的单" -X PUT "$BASE_URL/api/v1/bookings/$BID/status" -H "Authorization: Bearer $T_P5" \
  -H 'Content-Type: application/json' -d '{"status":"confirmed","actorTag":"photographer"}'
expect 200 "摄影师确认接单" -X PUT "$BASE_URL/api/v1/bookings/$BID/status" -H "Authorization: Bearer $T_P2" \
  -H 'Content-Type: application/json' -d '{"status":"confirmed","actorTag":"photographer"}'
expect 409 "非法回退 confirmed→pending" -X PUT "$BASE_URL/api/v1/bookings/$BID/status" -H "Authorization: Bearer $T_P2" \
  -H 'Content-Type: application/json' -d '{"status":"pending","actorTag":"photographer"}'
expect 200 "摄影师标记完成" -X PUT "$BASE_URL/api/v1/bookings/$BID/status" -H "Authorization: Bearer $T_P2" \
  -H 'Content-Type: application/json' -d '{"status":"completed","actorTag":"photographer"}'
expect 200 "接单面板可读" -H "Authorization: Bearer $T_P2" "$BASE_URL/api/v1/bookings/photographer/$PID"
expect 200 "我的订单可读" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/bookings/$COSER_ID"

section "E. 通知（带订单跳转字段）"
NOTIF=$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/notifications/$COSER_ID")
expect_true "coser 通知含 linkType=order" \
  "$(echo "$NOTIF" | jq -r --argjson b "$BID" '[.[]?|select(.linkId==$b)][0].linkType // "none"')" "order"
NOTIF_P2=$(curl -s -H "Authorization: Bearer $T_P2" "$BASE_URL/api/v1/notifications/$P2_ID")
expect_true "摄影师通知含 linkType=photographer_orders" \
  "$(echo "$NOTIF_P2" | jq -r --argjson b "$BID" '[.[]?|select(.linkId==$b)][0].linkType // "none"')" "photographer_orders"

section "F. 收藏 / 关注"
expect 200 "coser 收藏列表" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/favorites/$COSER_ID"
expect 200 "coser 关注列表" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/follows/$COSER_ID"
expect 404 "收藏不存在的摄影师" -X POST "$BASE_URL/api/v1/favorites" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d '{"photographerId":999999}'

section "G. 聊天"
SESS=$(curl -s -X POST "$BASE_URL/api/v1/chat/sessions" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"userId\":1,\"otherUserId\":$P2_ID}")
SID=$(echo "$SESS" | jq -r '.id // empty')
expect_true "建会话 201" "$([ -n "$SID" ] && echo ok)" "ok"
expect 200 "读会话消息（参与者）" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/chat/messages/$SID"
expect 403 "局外人读会话消息" -H "Authorization: Bearer $T_P5" "$BASE_URL/api/v1/chat/messages/$SID"
expect 201 "发消息" -X POST "$BASE_URL/api/v1/chat/messages" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"sessionId\":$SID,\"senderId\":1,\"content\":\"SMOKE-msg\"}"
expect 403 "局外人发消息" -X POST "$BASE_URL/api/v1/chat/messages" -H "Authorization: Bearer $T_P5" \
  -H 'Content-Type: application/json' -d "{\"sessionId\":$SID,\"content\":\"SMOKE-outsider\"}"
expect_true "伪造 senderId 被忽略" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/chat/messages/$SID" | jq -r '[.[]?|select(.content=="SMOKE-msg")][0].senderId == '"$COSER_ID"'')" "true"
expect 404 "与不存在的用户建会话" -X POST "$BASE_URL/api/v1/chat/sessions" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d '{"otherUserId":999999}'
expect 200 "未读数" -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/chat/unread/$COSER_ID"

section "H. 黑名单"
expect 201 "拉黑 photo2 的账号" -X POST "$BASE_URL/api/v1/blocks" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"blockedUserId\":$P2_USER}"
expect 400 "拉黑自己" -X POST "$BASE_URL/api/v1/blocks" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"blockedUserId\":$COSER_ID}"
expect 404 "拉黑不存在的用户" -X POST "$BASE_URL/api/v1/blocks" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d '{"blockedUserId":999999}'
expect 403 "拉黑后向其发起会话" -X POST "$BASE_URL/api/v1/chat/sessions" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"otherUserId\":$P2_ID}"
expect 403 "拉黑后预约该摄影师" -X POST "$BASE_URL/api/v1/bookings" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d "{\"photographerId\":$PID,\"serviceId\":1,\"date\":\"2027-01-05\",\"time\":\"10:00\",\"remarks\":\"SMOKE-block\"}"
expect 403 "反向（被拉黑方主动）" -X POST "$BASE_URL/api/v1/chat/sessions" -H "Authorization: Bearer $T_P2" \
  -H 'Content-Type: application/json' -d "{\"otherUserId\":$COSER_ID}"
expect_true "拉黑后搜索隐藏该摄影师" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/photographers?size=20" | jq -r "[.list[]?|select(.id==$PID)]|length")" "0"
expect_true "匿名搜索仍可见该摄影师" \
  "$(curl -s "$BASE_URL/api/v1/photographers?size=20" | jq -r "[.list[]?|select(.id==$PID)]|length")" "1"
expect_true "标签筛选下仍过滤" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/photographers?tags=$SEARCH_DATE_TAG&size=20" | jq -r "[.list[]?|select(.id==$PID)]|length")" "0"
expect 200 "解除拉黑" -X DELETE "$BASE_URL/api/v1/blocks/$P2_USER" -H "Authorization: Bearer $T_COSER"
expect_true "解除后搜索恢复可见" \
  "$(curl -s -H "Authorization: Bearer $T_COSER" "$BASE_URL/api/v1/photographers?size=20" | jq -r "[.list[]?|select(.id==$PID)]|length")" "1"

section "I. 账号安全：退出所有设备"
T_TMP=$(login "$COSER_PHONE")
expect 200 "临时会话 /me" -H "Authorization: Bearer $T_TMP" "$BASE_URL/api/v1/me"
expect 200 "退出所有设备" -X POST "$BASE_URL/api/v1/me/logout-all" -H "Authorization: Bearer $T_TMP"
expect 401 "退出后旧 token 失效" -H "Authorization: Bearer $T_TMP" "$BASE_URL/api/v1/me"

echo
if [ "$FAIL" -eq 0 ]; then
  printf "${G}全部通过${Z}：%d/%d\n" "$PASS" "$((PASS + FAIL))"
else
  printf "${R}失败 %d 项${Z}（通过 %d）:\n" "$FAIL" "$PASS"
  for f in "${FAILED_LIST[@]}"; do printf "  ${R}✗${Z} %s\n" "$f"; done
  exit 1
fi
