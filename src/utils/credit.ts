/**
 * 信用分文案与配色映射。
 *
 * 设计约束：
 * - 不碰主状态文案规则，纯展示层工具。
 * - 不用 window / URL / toISOString（小程序端不可用）。
 * - reason 后端可能返回库里的枚举原文（order_completed / dispute_lost …），
 *   也可能返回接口层抽象后的 success / cancel 语义，两套都要覆盖。
 */
import type { CreditEventItem, CreditLevel } from '@/types'

/** 等级色调，对应页面里的 class 后缀。 */
export type CreditTone = 'good' | 'fair' | 'poor'

/** 等级中文文案。 */
export const CREDIT_LEVEL_TEXT: Record<CreditLevel, string> = {
  good: '信用良好',
  fair: '信用一般',
  poor: '信用较差'
}

/** 等级短文案（标签用）。 */
export const CREDIT_LEVEL_SHORT: Record<CreditLevel, string> = {
  good: '良好',
  fair: '一般',
  poor: '较差'
}

/**
 * 由分数推导等级。
 * 阈值与后端一致：>=90 good、>=70 fair、其余 poor。
 * 后端未返回 level 时前端兜底，避免标签空白。
 */
export function creditLevelFromScore(score: number): CreditLevel {
  if (score >= 90) return 'good'
  if (score >= 70) return 'fair'
  return 'poor'
}

/** 等级文案（带兜底推导）。 */
export function creditLevelText(level?: CreditLevel | string | null, score?: number | null): string {
  if (level === 'good' || level === 'fair' || level === 'poor') {
    return CREDIT_LEVEL_TEXT[level]
  }
  if (typeof score === 'number' && Number.isFinite(score)) {
    return CREDIT_LEVEL_TEXT[creditLevelFromScore(score)]
  }
  return ''
}

/** 等级短文案（带兜底推导）。 */
export function creditLevelShort(level?: CreditLevel | string | null, score?: number | null): string {
  if (level === 'good' || level === 'fair' || level === 'poor') {
    return CREDIT_LEVEL_SHORT[level]
  }
  if (typeof score === 'number' && Number.isFinite(score)) {
    return CREDIT_LEVEL_SHORT[creditLevelFromScore(score)]
  }
  return ''
}

/** 等级色调。 */
export function creditTone(level?: CreditLevel | string | null, score?: number | null): CreditTone {
  if (level === 'good' || level === 'fair' || level === 'poor') return level
  if (typeof score === 'number' && Number.isFinite(score)) return creditLevelFromScore(score)
  return 'fair'
}

/**
 * 信用变动原因文案。
 * 覆盖两套枚举：
 * - 库里的原文（credit_events.reason CHECK 约束）：order_completed / photographer_no_show /
 *   coser_no_show / dispute_lost / late_cancel / delivery_overdue
 * - 接口层抽象语义：success / cancel
 */
const REASON_TEXT: Record<string, string> = {
  success: '订单完成',
  order_completed: '订单完成',
  cancel: '取消预约',
  late_cancel: '24 小时内取消预约',
  dispute_lost: '争议裁决败诉',
  coser_no_show: 'Coser 爽约',
  photographer_no_show: '摄影师爽约',
  delivery_overdue: '超时未交付',
  no_show: '标记爽约',
  dispute: '争议处理',
  credit_delta: '信用分变更'
}

/** 原因文案（未知枚举回落到原文，不编造）。 */
export function creditReasonText(reason?: string | null): string {
  if (!reason) return '信用分变更'
  return REASON_TEXT[reason] || reason
}

/**
 * 明细行展示模型。
 */
export interface CreditEventView {
  id: number
  /** 原因文案 */
  title: string
  /** 原因枚举原文（副标题） */
  reason: string
  /** 增减分文字（带正负号） */
  deltaText: string
  /** 是否加分 */
  plus: boolean
  /** 时间文案 */
  time: string
  /** 关联订单 id（有则显示） */
  bookingId?: number | null
  /** 后端补充说明 */
  description?: string | null
}

/** 把 ISO 时间格式化成 'MM-DD HH:mm'（本地时区，不用 toISOString）。 */
function formatDateTime(iso?: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}

/** 把 credit_events 的一行转成展示模型。 */
export function toCreditEventView(ev: CreditEventItem): CreditEventView {
  const delta = typeof ev.delta === 'number' ? ev.delta : 0
  return {
    id: ev.id,
    title: creditReasonText(ev.reason),
    reason: ev.reason || '',
    deltaText: delta > 0 ? `+${delta}` : String(delta),
    plus: delta > 0,
    time: formatDateTime(ev.createdAt),
    bookingId: ev.bookingId ?? ev.refId ?? null,
    description: ev.description ?? null
  }
}

/**
 * 冻结提示文案。
 * 传入冻结截止时间，返回可展示的提示；未冻结返回 ''。
 */
export function frozenHint(frozenUntil?: string | null, now: number = Date.now()): string {
  if (!frozenUntil) return ''
  const t = new Date(frozenUntil).getTime()
  if (Number.isNaN(t) || t <= now) return ''
  const mm = String(new Date(t).getMonth() + 1).padStart(2, '0')
  const dd = String(new Date(t).getDate()).padStart(2, '0')
  return `已冻结接单，解冻时间 ${mm}-${dd}`
}
