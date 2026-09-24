#!/usr/bin/env bash
# C 端对比度审计的回归守卫（可重复）。用法：
#   bash scripts/contrast-regression-check.sh [BASE_DIR] [NEW_DIR]
#   默认 BASE_DIR=.audit/task6、NEW_DIR=.audit/final
#
# 为什么它必须是脚本而不是 plan 里的一段 markdown 片段（Ruling T7-2b）：
# Ruling T7-1 加的「复测多出基线外的页 → FATAL」防护只有在**真的被执行**时才有意义。
# 写成片段容易在复制时被漏掉（本项目已两次栽在「看着通过、其实没测」上）。
#
# 退出码：
#   0  无回归（且页集双向一致）
#   1  有页面 total 上升（REGRESSION）
#   2  测量不完整/不可信：基线缺失或为空、复测缺页、复测多出基线外的页
set -uo pipefail

BASE_DIR="${1:-.audit/task6}"
NEW_DIR="${2:-.audit/final}"

# 基线必须存在且非空：`.audit/` 是 gitignore 的，缺基线时 diff 会拿空字典对比并静默
# 输出「无回归」—— 那正是本项目栽过的假通过。
test -n "$(ls -A "$BASE_DIR"/*.json 2>/dev/null)" || {
  echo "FATAL: 回归基线 $BASE_DIR 为空或缺失 —— 拒绝给出「无回归」结论" >&2; exit 2; }
test -n "$(ls -A "$NEW_DIR"/*.json 2>/dev/null)" || {
  echo "FATAL: 复测目录 $NEW_DIR 为空或缺失 —— 本次运行不是有效结论" >&2; exit 2; }

python3 - "$BASE_DIR" "$NEW_DIR" <<'PY'
import json, glob, os, sys
def load(d):
    out={}
    for f in glob.glob(d+'/*.json'):
        try: out[os.path.basename(f)]=json.loads(json.loads(open(f).read().strip()))['total']
        except Exception: pass
    return out
b,a=load(sys.argv[1]),load(sys.argv[2])
if not b: sys.exit('FATAL: 基线为空，拒绝给出「无回归」结论')
missing=[k for k in b if k not in a]
if missing: sys.exit('FATAL: 复测缺页 %s（测量不完整，不得判定通过）' % missing)
extra=[k for k in a if k not in b]
if extra:
    # 复测里出现基线没有的页 —— 多为 OUT_DIR 残留的旧文件（见 Ruling T7-1）。
    # 这些页无法与基线比较，既不能计入回归，也**不得**当作通过：必须显式列出并阻断。
    sys.exit('FATAL: 复测含基线之外的 %d 页 %s —— 疑为 OUT_DIR 残留旧文件，请清理后重跑' % (len(extra), extra))
rose=[k for k in b if a.get(k,0)>b[k]]
for k in rose: print('REGRESSION', k, b[k], '->', a[k])
print('页数: 基线 %d / 复测 %d' % (len(b), len(a)))
print('仍不达标页:', {k:a.get(k) for k in sorted(a) if a.get(k)})
sys.exit(1 if rose else 0)
PY
