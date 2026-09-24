/**
 * 套餐/订单时长的唯一格式化入口。
 *
 * 收敛原因：此前「我的套餐」页对 0/空时长显示「未填写」，而「预约」页显示「0分钟」——
 * 同一份数据显示出两种口径，且「0分钟」看起来像一个真实时长。这里统一为「未填写」。
 *
 * 单位一律「分钟」入参（后端字段即分钟）。
 */
export function formatDuration(minutes: number | null | undefined): string {
  if (minutes === null || minutes === undefined) return '未填写'
  if (typeof minutes !== 'number' || !isFinite(minutes) || minutes <= 0) return '未填写'
  if (minutes < 60) return `${minutes} 分钟`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m === 0 ? `${h} 小时` : `${h} 小时 ${m} 分钟`
}
