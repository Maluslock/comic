/**
 * 占位名判定 —— 必须与后端 `server/internal/service/photographer_service.go` 的
 * `isPlaceholderName()` 逐条对齐，否则「后端认为是占位、前端当成真名」或反之，
 * 用户会看到「微信用户 / 11 / 摄影师15」这类系统值当成自己的艺名。
 *
 * 后端规则（三条）：
 *  1. 空串（trim 后）或「微信用户」→ 占位。
 *  2. 以「摄影师」开头且剩余部分全为数字 → 占位（迁移 000042 的兜底名「摄影师15」）。
 *  3. 整串全为数字 → 占位（历史脏数据「11」）。
 *
 * 反向边界（必须返回 false，否则会把真名误判成占位）：
 *  「摄影师」三字本身、`摄影师小王`、`摄影师15号`、`摄影师 15`、`阿伟摄影`。
 */

/** 后端 `isDigits()` 的等价实现：空串返回 false，非纯 ASCII 数字返回 false。 */
function isDigits(value: string): boolean {
  if (value === '') return false
  for (let i = 0; i < value.length; i += 1) {
    const code = value.charCodeAt(i)
    if (code < 48 || code > 57) return false
  }
  return true
}

export function isPlaceholderName(name: string | null | undefined): boolean {
  const trimmed = (name ?? '').trim()
  if (trimmed === '' || trimmed === '微信用户') return true
  if (trimmed.startsWith('摄影师')) return isDigits(trimmed.slice('摄影师'.length))
  return isDigits(trimmed)
}
