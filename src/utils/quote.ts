import { apiPost, ApiError } from '@/api/client'
import type { OrderPriceFields, PriceMode, PriceStatus } from '@/types'
import { formatPrice } from '@/utils/mappers'

export function toPriceMode(value: unknown): PriceMode {
  return value === 'mutual' || value === 'negotiable' ? value : 'fixed'
}

export function toPriceStatus(value: unknown): PriceStatus {
  if (value === 'awaiting_quote' || value === 'quoted' || value === 'rejected') return value
  return 'agreed'
}

/** Render an order amount from its pricing mode and single-round quote state. */
export function renderOrderPrice(order: OrderPriceFields): string {
  if (order.priceMode === 'negotiable') {
    if (order.priceStatus === 'awaiting_quote') return '待报价'
    if (order.priceStatus === 'quoted') {
      return order.quotePrice === null ? '已报价' : `已报价 ¥${order.quotePrice}`
    }
    if (order.priceStatus === 'agreed') return `¥${order.totalPrice}`
    return '已拒绝'
  }
  return formatPrice(order.priceMode === 'mutual' ? 0 : order.totalPrice)
}

/**
 * Turn a quote/respond failure into an honest toast. State-guard conflicts (409)
 * must never read as success, so they get their own message instead of a generic one.
 */
export function toastQuoteError(e: unknown, badRequestMessage = '参数有误'): void {
  const code = e instanceof ApiError ? e.status : 0
  if (code === 409) {
    uni.showToast({ title: '报价状态已变化，请刷新', icon: 'none' })
    return
  }
  if (code === 403) {
    uni.showToast({ title: '无权操作该订单', icon: 'none' })
    return
  }
  if (code === 400) {
    uni.showToast({ title: badRequestMessage, icon: 'none' })
    return
  }
  uni.showToast({ title: '操作失败，请重试', icon: 'none' })
}

function parseQuotePrice(raw: string): number | null {
  if (!/^\d+$/.test(raw)) return null
  const price = Number(raw)
  if (price < 1 || price > 99999) return null
  return price
}

/**
 * Collect a price with the native editable modal and POST the photographer quote.
 * `refresh` always runs after the attempt so the list reflects server truth.
 */
export function promptQuote(bookingId: number | string, refresh: () => void): void {
  uni.showModal({
    title: '报价',
    editable: true,
    placeholderText: '请输入报价金额（1-99999）',
    success: (res) => {
      if (!res.confirm) return
      const price = parseQuotePrice((res.content || '').trim())
      if (price === null) {
        uni.showToast({ title: '请输入 1-99999 的整数金额', icon: 'none' })
        return
      }
      void postQuote(bookingId, price, refresh)
    },
  })
}

async function postQuote(bookingId: number | string, price: number, refresh: () => void): Promise<void> {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPost(`/v1/bookings/${bookingId}/quote`, { price })
    uni.hideLoading()
    uni.showToast({ title: '报价成功', icon: 'success' })
  } catch (e) {
    uni.hideLoading()
    toastQuoteError(e, '金额需为 1-99999 的整数')
  } finally {
    refresh()
  }
}

/** Confirm then accept/reject a quote from the coser side. */
export function confirmRespondQuote(bookingId: number | string, accept: boolean, refresh: () => void): void {
  uni.showModal({
    title: accept ? '接受报价' : '拒绝报价',
    content: accept ? '确定接受该报价吗？接受后订单将确认。' : '拒绝报价后订单将取消，确定拒绝吗？',
    success: (res) => {
      if (res.confirm) void postRespond(bookingId, accept, refresh)
    },
  })
}

async function postRespond(bookingId: number | string, accept: boolean, refresh: () => void): Promise<void> {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPost(`/v1/bookings/${bookingId}/quote/respond`, { accept })
    uni.hideLoading()
    uni.showToast({ title: accept ? '已接受报价' : '已拒绝报价', icon: 'success' })
  } catch (e) {
    uni.hideLoading()
    toastQuoteError(e, '参数有误')
  } finally {
    refresh()
  }
}
