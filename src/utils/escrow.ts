/**
 * 担保交易（C 端）公共工具：子状态文案、倒计时、履约时间线、争议文案。
 *
 * 设计约束：
 * - 主状态（pending/confirmed/completed/cancelled）文案由各页面自己维护，本文件不碰，
 *   保证「不改现有粗状态文案」。
 * - 子状态（in_service/delivered/disputed）只描述 confirmed 阶段的履约进度。
 * - 不用 window / URL / toISOString（小程序端不可用），时间一律走本地时区手动拼接。
 */
import type { BookingSubStatus, DisputeStatus, DisputeType, OrderEventItem } from '@/types'

/** 子状态的视觉色调，对应页面里的 class 后缀。 */
export type EscrowTone = 'cyan' | 'purple' | 'warning' | 'error' | 'success'

/** 订单在担保交易下的展示状态（主状态由页面拼，子状态在这里）。 */
export interface EscrowState {
  /** 子状态文案；无子状态时为 '' */
  subText: string
  /** 色调 */
  tone: EscrowTone
  /** 是否需要用户处理（待确认交付 / 争议中 / 待核销） */
  todo: boolean
}

/**
 * 由子状态 + 核销时间推导展示状态。
 *
 * 坑：后端 `setBookingConfirmed` 在摄影师确认接单时就把 sub_status 写成 'in_service'，
 * 所以「待核销」和「拍摄中」的区分只能看 checkin_at 是否为空。
 */
export function escrowState(
  status: string,
  subStatus: BookingSubStatus | null | undefined,
  checkinAt?: string | null
): EscrowState {
  if (status !== 'confirmed') {
    return { subText: '', tone: 'cyan', todo: false }
  }
  if (subStatus === 'disputed') {
    return { subText: '争议中', tone: 'error', todo: true }
  }
  if (subStatus === 'delivered') {
    return { subText: '待确认交付', tone: 'warning', todo: true }
  }
  if (subStatus === 'in_service') {
    if (checkinAt) {
      return { subText: '拍摄中', tone: 'purple', todo: false }
    }
    return { subText: '待核销', tone: 'cyan', todo: true }
  }
  // confirmed 但没有子状态：老数据，保守提示待核销
  return { subText: '待核销', tone: 'cyan', todo: true }
}

/** 待办排序权重：数字越小越靠前。 */
export function escrowTodoWeight(status: string, subStatus?: BookingSubStatus | null): number {
  if (status !== 'confirmed') return 100
  if (subStatus === 'disputed') return 0
  if (subStatus === 'delivered') return 1
  if (subStatus === 'in_service') return 2
  return 3
}

/** 争议状态文案。 */
export function disputeStatusText(status: DisputeStatus | string): string {
  switch (status) {
    case 'open':
      return '待响应'
    case 'responding':
      return '已响应'
    case 'escalated':
      return '已升级待裁决'
    case 'resolved':
      return '系统已裁决'
    case 'closed':
      return '已裁决'
    default:
      return '处理中'
  }
}

/** 争议类型文案。 */
export function disputeTypeText(type: DisputeType | string): string {
  switch (type) {
    case 'no_show':
      return '对方爽约'
    case 'not_delivered':
      return '未按约交付'
    case 'quality':
      return '成片质量争议'
    case 'other':
      return '其他争议'
    default:
      return '交易争议'
  }
}

/** 把 ISO 时间格式化成 'MM-DD HH:mm'（本地时区，不用 toISOString）。 */
export function formatDateTime(iso?: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}

/**
 * 把时间戳（毫秒）格式化成 'MM-DD HH:mm'。
 * 单独提供是因为订单创建时间在接口里是 ISO 字符串、在本地缓存里是时间戳，
 * 两条来源都要能落到时间线上；这里直接用本地 getter，避开 toISOString 的 UTC 偏移。
 */
export function formatTimestamp(ts?: number | null): string {
  if (!ts || Number.isNaN(ts)) return ''
  const d = new Date(ts)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}

/**
 * 倒计时文案：'剩余 2 天 3 小时' / '剩余 45 分钟' / '已超时'。
 * 传过去的时间返回 '已超时'，用于争议响应/举证截止。
 */
export function countdownText(deadline?: string | null, now: number = Date.now()): string {
  if (!deadline) return ''
  const t = new Date(deadline).getTime()
  if (Number.isNaN(t)) return ''
  const diff = t - now
  if (diff <= 0) return '已超时'
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '剩余不到 1 分钟'
  if (minutes < 60) return `剩余 ${minutes} 分钟`
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  if (hours < 24) {
    return restMinutes > 0 ? `剩余 ${hours} 小时 ${restMinutes} 分` : `剩余 ${hours} 小时`
  }
  const days = Math.floor(hours / 24)
  const restHours = hours % 24
  return restHours > 0 ? `剩余 ${days} 天 ${restHours} 小时` : `剩余 ${days} 天`
}

/** 履约时间线的一节点。 */
export interface TimelineNode {
  /** 事件类型 */
  key: string
  /** 节点标题 */
  title: string
  /** 节点说明 */
  desc: string
  /** 时间文案（空 = 未发生） */
  time: string
  /** 原始 ISO 时间，用于排序 */
  rawTime: string
  /** 操作方文案 */
  actor: string
  /** 是否已发生 */
  done: boolean
}

const EVENT_TITLES: Record<string, string> = {
  confirmed: '摄影师确认接单',
  checkin: '拍摄核销',
  deliver: '摄影师提交交付',
  accept: '确认交付',
  auto_accept: '系统自动确认交付',
  complete_legacy: '订单完成',
  cancel: '订单取消',
  auto_cancel: '系统自动取消',
  noshow: '标记爽约',
  dispute_open: '发起争议',
  dispute_respond: '争议响应',
  dispute_escalate: '争议升级',
  dispute_resolve: '争议裁决',
  credit_delta: '信用分变更',
}

const EVENT_DESCS: Record<string, string> = {
  confirmed: '摄影师已确认本次预约',
  checkin: '摄影师输入核销码，确认 coser 已到场',
  deliver: '摄影师提交了拍摄交付凭证',
  accept: 'coser 已确认收到成片，订单完成',
  auto_accept: '超过确认时限，系统自动确认交付',
  complete_legacy: '订单已结束',
  cancel: '订单已取消',
  auto_cancel: '超时未确认，系统自动取消',
  noshow: '一方未按时到场，已标记爽约',
  dispute_open: '订单进入争议流程，等待对方响应',
  dispute_respond: '对方已提交争议说明',
  dispute_escalate: '争议已升级，等待管理员裁决',
  dispute_resolve: '争议已裁决，订单状态已恢复',
  credit_delta: '信用分发生变更',
}

function actorLabel(actorRole?: string | null, myRole?: string | null): string {
  if (actorRole === 'system') return '系统'
  if (actorRole === 'admin') return '平台'
  if (actorRole === 'coser') return myRole === 'coser' ? '我' : 'coser'
  if (actorRole === 'photographer') return myRole === 'photographer' ? '我' : '摄影师'
  return '系统'
}

/**
 * 由订单事件构建履约时间线。
 *
 * 期望的主链路：预约 → 确认 → 核销 → 交付 → 完成。
 * 事件由后端 GET /v1/bookings/:id/events 返回（按 created_at 升序）。
 * 没发生的节点也会占位，保证时间线结构稳定可读。
 *
 * 兜底说明：后端目前只在 checkin/deliver/accept 等节点写 order_events，
 * 「预约」与「确认接单」没有事件记录，因此这两个节点用订单自身字段兜底：
 * - 预约：订单创建时间（createdAt）存在即视为已提交。
 * - 确认接单：订单状态已离开 pending 即视为已确认。
 */
export function buildTimeline(
  events: OrderEventItem[],
  myRole?: string | null,
  opts?: { status?: string | null; createdAt?: string | number | null }
): TimelineNode[] {
  const byType = new Map<string, OrderEventItem>()
  for (const ev of events) {
    if (!ev || !ev.eventType) continue
    // 同一类型多次发生取最早一条（例如 auto_accept 与 accept）
    if (!byType.has(ev.eventType)) byType.set(ev.eventType, ev)
  }

  const pick = (type: string): OrderEventItem | undefined => byType.get(type)

  const nodes: TimelineNode[] = []

  const status = opts?.status || ''
  const createdAtRaw = opts?.createdAt
  const createdAtIso = typeof createdAtRaw === 'string' && createdAtRaw ? createdAtRaw : ''
  const createdAtText = createdAtIso
    ? formatDateTime(createdAtIso)
    : typeof createdAtRaw === 'number'
      ? formatTimestamp(createdAtRaw)
      : ''

  // 预约（后端事件流里没有 booked 事件，用订单创建时间兜底）
  nodes.push({
    key: 'booked',
    title: '提交预约',
    desc: '订单已创建，等待摄影师确认',
    time: createdAtText,
    rawTime: createdAtIso,
    actor: '我',
    done: !!createdAtText,
  })

  const push = (key: string, type: string, overrideTitle?: string): void => {
    const ev = pick(type)
    nodes.push({
      key,
      title: overrideTitle || EVENT_TITLES[type] || type,
      desc: EVENT_DESCS[type] || '',
      time: ev ? formatDateTime(ev.createdAt) : '',
      rawTime: ev ? ev.createdAt : '',
      actor: ev ? actorLabel(ev.actorRole, myRole) : '',
      done: !!ev,
    })
  }

  // 确认接单：没有 order_event，退回用订单状态推导（离开 pending 即已确认）
  const confirmedEvent = pick('confirmed')
  const confirmedDone = !!confirmedEvent || (!!status && status !== 'pending')
  nodes.push({
    key: 'confirmed',
    title: EVENT_TITLES.confirmed,
    desc: confirmedEvent
      ? EVENT_DESCS.confirmed
      : confirmedDone
        ? '摄影师已确认本次预约'
        : '等待摄影师确认接单',
    time: confirmedEvent ? formatDateTime(confirmedEvent.createdAt) : '',
    rawTime: confirmedEvent ? confirmedEvent.createdAt : '',
    actor: confirmedEvent ? actorLabel(confirmedEvent.actorRole, myRole) : confirmedDone ? '摄影师' : '',
    done: confirmedDone,
  })

  push('checkin', 'checkin')
  push('deliver', 'deliver')

  const accept = pick('accept')
  const autoAccept = pick('auto_accept')
  nodes.push({
    key: 'accept',
    title: '确认交付',
    desc: accept ? EVENT_DESCS.accept : autoAccept ? EVENT_DESCS.auto_accept : 'coser 确认收到成片后订单完成',
    time: accept ? formatDateTime(accept.createdAt) : autoAccept ? formatDateTime(autoAccept.createdAt) : '',
    rawTime: (accept || autoAccept)?.createdAt || '',
    actor: accept ? actorLabel(accept.actorRole, myRole) : autoAccept ? '系统' : '',
    done: !!accept || !!autoAccept,
  })

  const legacy = pick('complete_legacy')
  nodes.push({
    key: 'completed',
    title: '订单完成',
    desc: legacy ? EVENT_DESCS.complete_legacy : '订单完成并结算',
    time: legacy ? formatDateTime(legacy.createdAt) : '',
    rawTime: legacy?.createdAt || '',
    actor: legacy ? actorLabel(legacy.actorRole, myRole) : '',
    done: !!legacy,
  })

  // 异常分支（取消 / 爽约 / 争议）追加在后面
  const extraTypes = ['noshow', 'dispute_open', 'dispute_respond', 'dispute_escalate', 'dispute_resolve', 'cancel', 'auto_cancel']
  for (const type of extraTypes) {
    const ev = pick(type)
    if (!ev) continue
    nodes.push({
      key: type,
      title: EVENT_TITLES[type] || type,
      desc: EVENT_DESCS[type] || '',
      time: formatDateTime(ev.createdAt),
      rawTime: ev.createdAt,
      actor: actorLabel(ev.actorRole, myRole),
      done: true,
    })
  }

  return nodes
}
