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
#   0  无回归，且每页的 .meta（元素数 n / 文本长度 len）没有塌陷
#   1  有页面 total 上升（REGRESSION）
#   2  测量不完整/不可信：基线与复测页集不一致（缺页 / 多出页）、
#      .meta 缺失或解析失败、或 .meta 塌陷（len 或 n 归零、或跌幅 ≥50%）
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
    """{name: {'total': int, 'n': int|None, 'len': int|None}}
    .meta 缺失或解析失败 → n/len 记 None（由下面的检查判 FATAL）。"""
    out={}
    for f in sorted(glob.glob(d+'/*.json')):
        k=os.path.basename(f)[:-5]
        try:
            total=json.loads(json.loads(open(f).read().strip()))['total']
        except Exception:
            continue
        n=ln=None
        mf=os.path.join(d, k+'.meta')
        if os.path.exists(mf):
            try:
                m=json.loads(json.loads(open(mf).read().strip()))
                n, ln = m.get('n'), m.get('len')
            except Exception:
                n=ln=None
        out[k]={'total':total, 'n':n, 'len':ln}
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

# --- .meta 强制（Ruling T7-2 Blocking 1）---
# 只比 total 的话，一个「悄悄不再渲染」的页面会报 0→0 而通过 —— 正是本项目栽过两次
# 的空态假通过。这里把 .meta 的 n（元素数）/ len（文本长度）变成硬约束：
#   * 任一侧 .meta 缺失/不可解析 → FATAL
#   * 复测 len 归零，或相对基线跌幅 ≥50% → FATAL（文本塌陷）
#   * 复测 n 归零，或相对基线跌幅 ≥50% → FATAL（元素塌陷）
# 覆盖「基线 total=0 且复测 total=0，但 .meta 崩了」这一关键情形：其判定与 total 无关。
meta_fail=[]
for k in sorted(b):
    bn, bl = b[k]['n'], b[k]['len']
    an, al = a[k]['n'], a[k]['len']
    if bl is None or bn is None:
        meta_fail.append('%s: 基线 .meta 缺失或不可解析（n=%r len=%r）' % (k, bn, bl))
        continue
    if al is None or an is None:
        meta_fail.append('%s: 复测 .meta 缺失或不可解析（n=%r len=%r）' % (k, an, al))
        continue
    if al == 0:
        meta_fail.append('%s: 文本长度 len=%d（基线 %d）—— 页面疑似空白' % (k, al, bl))
    elif al < bl * 0.5:
        meta_fail.append('%s: 文本长度 len %d→%d（-%.0f%%，≥50%% 塌陷）' % (k, bl, al, 100.0*(bl-al)/bl))
    if an == 0:
        meta_fail.append('%s: 元素数 n=%d（基线 %d）—— 页面疑似未渲染' % (k, an, bn))
    elif an < bn * 0.5:
        meta_fail.append('%s: 元素数 n %d→%d（-%.0f%%，≥50%% 塌陷）' % (k, bn, an, 100.0*(bn-an)/bn))
if meta_fail:
    sys.exit('FATAL: .meta 塌陷 %d 项 —— 空态/未渲染不得判为通过：\n  - %s'
             % (len(meta_fail), '\n  - '.join(meta_fail)))

# --- total 回归 ---
rose=[k for k in b if a[k]['total']>b[k]['total']]
for k in rose: print('REGRESSION', k, b[k]['total'], '->', a[k]['total'])

# --- 正常输出也报 .meta 实测值（delta 明细），让一轮运行自证测到了什么 ---
print('页数: 基线 %d / 复测 %d' % (len(b), len(a)))
print('仍不达标页:', {k:a[k]['total'] for k in sorted(a) if a[k]['total']})
print('.meta 明细 (n=元素数, len=文本长度; Δ=复测-基线):')
for k in sorted(b):
    bn, bl = b[k]['n'], b[k]['len']
    an, al = a[k]['n'], a[k]['len']
    print('  %-52s n %4d→%-4d (Δ%+d)   len %4d→%-4d (Δ%+d)' % (k, bn, an, an-bn, bl, al, al-bl))
sys.exit(1 if rose else 0)
PY
