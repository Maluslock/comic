/**
 * 信用分本地 mock 数据。
 *
 * 用途：后端只读接口（GET /v1/me/credit、/v1/me/credit/events）落地前，
 * 让个人中心信用卡片与信用明细页能先把 UI 跑通。
 * t11 上线后把 src/api/index.ts 里的 CREDIT_USE_MOCK 置为 false 即可切真实接口，
 * 页面结构与文案映射都不用改。
 *
 * reason 用库里的枚举原文（credit_events.reason CHECK 约束允许的值）。
 */
import type { CreditEventItem, CreditSummary } from '@/types'

/** 概览 mock：有加分也有扣分，覆盖三种等级展示。 */
export const mockCreditSummary: CreditSummary = {
  score: 87,
  level: 'fair',
  successCount: 12,
  cancelCount: 1,
  noshowCount: 0,
  disputeLostCount: 1,
  frozenUntil: null,
  updatedAt: '2026-10-29T19:30:00+08:00'
}

/** 明细 mock：按 created_at 倒序。 */
export const mockCreditEvents: CreditEventItem[] = [
  {
    id: 4,
    delta: 2,
    reason: 'order_completed',
    refType: 'booking',
    refId: 102,
    bookingId: 102,
    description: '订单完成',
    createdAt: '2026-10-29T19:20:00+08:00'
  },
  {
    id: 3,
    delta: -10,
    reason: 'dispute_lost',
    refType: 'dispute',
    refId: 13,
    bookingId: 103,
    description: '争议裁决败诉',
    createdAt: '2026-10-28T15:05:00+08:00'
  },
  {
    id: 2,
    delta: 2,
    reason: 'order_completed',
    refType: 'booking',
    refId: 98,
    bookingId: 98,
    description: '订单完成',
    createdAt: '2026-10-27T11:40:00+08:00'
  },
  {
    id: 1,
    delta: -3,
    reason: 'late_cancel',
    refType: 'booking',
    refId: 91,
    bookingId: 91,
    description: '24 小时内取消预约',
    createdAt: '2026-10-26T09:12:00+08:00'
  }
]
