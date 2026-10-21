/**
 * 本地日期工具。
 *
 * 为什么要单独一个文件：`new Date().toISOString().split('T')[0]` 取的是 **UTC** 日期，
 * 在东八区本地时间 00:00–08:00 之间会得到「昨天」。用它当预约默认日期，用户一进页面
 * 默认就是过去的一天（服务端也会按过去日期拒绝）。小程序/微信同样受影响。
 * 日期必须用本地时区拼，不能用 toISOString。
 */

/** 把 Date 格式化为本地时区的 YYYY-MM-DD。 */
export function localDateKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/** 今天的本地日期 YYYY-MM-DD（可作为日期选择器的下限）。 */
export function todayKey(): string {
  return localDateKey(new Date())
}
