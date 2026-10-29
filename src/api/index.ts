import { mockPhotographers, mockWorks, mockEvents, tags } from '@/data/mock'
import { apiGet, apiPost, apiPut, apiDelete } from '@/api/client'
import type {
  HomeResponse,
  OrderEventItem,
  DisputeItem,
  BookingDisputeResponse,
  CreditSummary,
  CreditEventsResponse
} from '@/types'

import { resolveImage } from '@/utils/mappers'
import { getApiBase } from '@/utils/apiBase'
import { mockCreditSummary, mockCreditEvents } from '@/data/creditMock'

async function uploadFile(filePath: string): Promise<string> {
  const token = uni.getStorageSync('token') as string

  return new Promise<string>((resolve, reject) => {
    uni.uploadFile({
      url: `${getApiBase()}/v1/upload`,
      filePath,
      name: 'file',
      header: token ? { Authorization: `Bearer ${token}` } : {},
      success: (res) => {
        if (res.statusCode !== 200 && res.statusCode !== 201) {
          reject(new Error('upload failed'))
          return
        }
        try {
          // 后端返回 /static/uploads/xxx 相对路径，小程序端必须补 origin，否则入库即裂图
          resolve(resolveImage((JSON.parse(res.data) as { url: string }).url))
        } catch (e) {
          reject(e)
        }
      },
      fail: reject
    })
  })
}

export function pickAndUploadImages(count = 1): Promise<string[]> {
  return new Promise<string[]>((resolve, reject) => {
    uni.chooseImage({
      count,
      success: async (res) => {
        try {
          const urls: string[] = []
          for (const path of res.tempFilePaths) urls.push(await uploadFile(path as string))
          resolve(urls)
        } catch (e) {
          reject(e)
        }
      },
      fail: reject
    })
  })
}

export interface CertApplication {
  id: number
  status: string
  reviewReason?: string | null
  createdAt: string
}

export function applyCertification(evidenceImages: string[], evidenceDesc: string) {
  return apiPost<{ id: number }>('/v1/photographers/cert-apply', { evidenceImages, evidenceDesc })
}

export function getMyCertApplication() {
  return apiGet<{ application: null | CertApplication }>('/v1/photographers/cert-application')
}

export function getMyCertApplications() {
  return apiGet<{ list: CertApplication[] }>('/v1/photographers/cert-applications')
}

export function updateUserProfile(payload: { name: string; avatar: string; bio?: string }) {
  return apiPut<{ ok: boolean }>('/v1/me/profile', payload)
}

export function logoutAllDevices() {
  return apiPost<{ ok: boolean }>('/v1/me/logout-all', {})
}

export interface BlockedUser {
  userId: number
  name: string
  avatar: string
  createdAt: string
}

export function getBlocks() {
  return apiGet<{ list: BlockedUser[] }>('/v1/blocks')
}

export function blockUser(blockedUserId: number) {
  return apiPost<{ ok: boolean }>('/v1/blocks', { blockedUserId })
}

export function unblockUser(blockedUserId: number) {
  return apiDelete<{ ok: boolean }>(`/v1/blocks/${blockedUserId}`)
}

export function markNotificationRead(id: string | number) {
  return apiPost<{ ok: boolean }>(`/v1/notifications/${id}/read`, {})
}

export function markChatSessionRead(sessionId: number | string) {
  return apiPost<{ ok: boolean }>(`/v1/chat/sessions/${sessionId}/read`, {})
}

export interface MyReview {
  id: number
  photographerId: number
  photographerName?: string
  rating: number
  content: string
  images?: string[]
  createdAt: string
}

export function getMyReviews() {
  return apiGet<MyReview[]>('/v1/reviews/mine')
}

export interface MyPhotographerProfile {
  id: number
  name: string
  avatar: string
  location: string
  description: string
  mode: string
  mutualIntro: string
  certified: boolean
  rating: number
  reviewCount: number
  orderCount: number
}

export function getMyPhotographerProfile() {
  return apiGet<MyPhotographerProfile>('/v1/photographers/profile/mine')
}

export function updatePhotographerProfile(payload: {
  name: string
  description: string
  location: string
  mode: string
  mutualIntro: string
  avatar: string
}) {
  return apiPut<{ ok: boolean }>('/v1/photographers/profile', payload)
}

export interface MyWork {
  id: number
  title: string
  images: string[]
  description?: string | null
  /** backend mine endpoint omits status (DB default 'active'); admin can down it */
  status?: string
  /** backend mine endpoint returns created_at (snake_case); createdAt kept for future camelCase */
  created_at?: string
  createdAt?: string
}

export function uploadWork(payload: { title: string; images: string[]; description?: string }) {
  return apiPost<{ id: number }>('/v1/photographers/works', payload)
}

export function updateWork(id: number, payload: { title: string; images: string[]; description?: string }) {
  return apiPut<{ ok: boolean }>(`/v1/photographers/works/${id}`, payload)
}

export function getMyWorks() {
  return apiGet<MyWork[]>('/v1/photographers/works/mine')
}

export function deleteWork(id: number) {
  return apiDelete<{ ok: boolean }>(`/v1/photographers/works/${id}`)
}

export interface MyService {
  id: number
  name: string
  /** null = 面议, 0 = 互勉, >0 = 固定价 */
  price: number | null
  description: string
  duration: number
  /** 是否上架到公开页面（下架后仍在自己列表可见，但不可被下单） */
  isActive: boolean
  sortOrder: number
}

/** 套餐可写字段。isActive / sortOrder 省略时，后端保留数据库现值。 */
export interface ServiceUpsertPayload {
  name: string
  price: number | null
  description: string
  duration: number
  isActive?: boolean
  sortOrder?: number
}

/** 平台模板只暴露预填所需字段（无 isActive / sortOrder —— 模板不可直接上架）。 */
export interface ServiceTemplate {
  id: number
  name: string
  price: number | null
  description: string
  duration: number
}

export function getMyServices() {
  return apiGet<{ list: MyService[] | null }>('/v1/photographers/services/mine')
}

export function getServiceTemplates() {
  return apiGet<ServiceTemplate[] | null>('/v1/services/templates')
}

export function createService(payload: ServiceUpsertPayload) {
  return apiPost<{ id: number }>('/v1/photographers/services', payload)
}

export function updateService(id: number, payload: ServiceUpsertPayload) {
  return apiPut<{ ok: boolean }>(`/v1/photographers/services/${id}`, payload)
}

export function deleteService(id: number) {
  return apiDelete<{ ok: boolean }>(`/v1/photographers/services/${id}`)
}

/* ───────────────────── 担保交易：履约节点 + 争议 ───────────────────── */

/**
 * 摄影师核销：输入 coser 的 6 位核销码，确认对方已到场。
 * 后端会校验核销码是否与订单一致（不一致 → 403 forbidden）。
 */
export function checkinBooking(bookingId: number | string, code: string) {
  return apiPost<Record<string, unknown>>(`/v1/bookings/${bookingId}/checkin`, { code })
}

/** 摄影师提交交付：上传成片凭证（图片 URL 数组）。 */
export function deliverBooking(bookingId: number | string, evidence: string[]) {
  return apiPost<Record<string, unknown>>(`/v1/bookings/${bookingId}/deliver`, { evidence })
}

/** coser 确认收到交付，订单转 completed。 */
export function acceptDelivery(bookingId: number | string) {
  return apiPost<Record<string, unknown>>(`/v1/bookings/${bookingId}/accept`, {})
}

/** 订单履约事件流（时间线数据源）。仅订单双方可读。 */
export function getOrderEvents(bookingId: number | string) {
  return apiGet<{ list: OrderEventItem[] }>(`/v1/bookings/${bookingId}/events`)
}

/** 订单当前争议 + 我在该订单中的角色。无未关闭争议时 dispute 为 null。 */
export function getBookingDispute(bookingId: number | string) {
  return apiGet<BookingDisputeResponse>(`/v1/bookings/${bookingId}/dispute`)
}

/** 发起争议。type: no_show | not_delivered | quality | other。 */
export function openDispute(
  bookingId: number | string,
  payload: { type: string; reason: string; evidence?: string[] }
) {
  return apiPost<DisputeItem>(`/v1/bookings/${bookingId}/disputes`, payload)
}

/** 响应争议（仅非申请方）。 */
export function respondDispute(
  disputeId: number | string,
  payload: { response: string; evidence?: string[] }
) {
  return apiPost<DisputeItem>(`/v1/disputes/${disputeId}/respond`, payload)
}

/** 升级争议，等待管理员裁决。 */
export function escalateDispute(disputeId: number | string, reason?: string) {
  return apiPost<DisputeItem>(`/v1/disputes/${disputeId}/escalate`, reason ? { reason } : {})
}

/** 摄影师标记对方爽约（party 默认 coser）。 */
export function markNoShow(bookingId: number | string, party: 'coser' | 'photographer' = 'coser') {
  return apiPost<Record<string, unknown>>(`/v1/bookings/${bookingId}/noshow`, { party })
}

/**
 * 信用分开关：后端只读接口（t11）落地前用本地 mock 跑通 UI。
 * t11 上线后把它置为 false，页面结构不用动。
 */
export const CREDIT_USE_MOCK = true

/** 我的信用分概览。 */
export async function getMyCredit(): Promise<CreditSummary> {
  if (!CREDIT_USE_MOCK) {
    return apiGet<CreditSummary>('/v1/me/credit')
  }
  try {
    return await apiGet<CreditSummary>('/v1/me/credit')
  } catch {
    // 后端接口还没上线（404/500）→ 回落到 mock，页面照常渲染
    return { ...mockCreditSummary }
  }
}

/** 我的信用分明细（时间倒序）。 */
export async function getMyCreditEvents(params?: { limit?: number; offset?: number }): Promise<CreditEventsResponse> {
  const limit = params?.limit ?? 20
  const offset = params?.offset ?? 0
  const query = { limit: String(limit), offset: String(offset) }
  if (!CREDIT_USE_MOCK) {
    return apiGet<CreditEventsResponse>('/v1/me/credit/events', query)
  }
  try {
    return await apiGet<CreditEventsResponse>('/v1/me/credit/events', query)
  } catch {
    const list = mockCreditEvents.slice(offset, offset + limit)
    return { list, total: mockCreditEvents.length }
  }
}

export async function getHomeData(): Promise<HomeResponse> {
  return {
    banners: [
      { id: 1, imageUrl: '/static/img/banner-1.jpg', title: 'ChinaJoy 2026', linkType: 'event', linkId: 1 },
      { id: 2, imageUrl: '/static/img/banner-2.jpg', title: '第40届萤火虫漫展', linkType: 'event', linkId: 2 },
      { id: 3, imageUrl: '/static/img/banner-3.jpg', title: 'CP33 综合同人展', linkType: 'event', linkId: 3 },
    ],
    upcomingEvents: mockEvents.map((e, i) => ({
      id: i + 1,
      name: e.name,
      location: e.location,
      venue: e.venue,
      startDate: new Date(e.startDate).toISOString(),
      endDate: new Date(e.endDate).toISOString(),
      coverUrl: e.cover,
      tags: e.tags,
      status: e.status,
      typeName: '漫展',
    })),
    hotTags: tags.map((t, i) => ({
      name: t,
      usageCount: (i + 1) * 10 + Math.floor(Math.random() * 50),
    })),
    recommendedPhotographers: mockPhotographers.map((p, i) => ({
      id: i + 1,
      name: p.name,
      avatar: p.avatar,
      location: p.location || '',
      rating: p.rating,
      reviewCount: p.reviewCount,
      orderCount: p.orderCount,
      tags: p.tags,
    })),
    featuredWorks: mockWorks.map((w, i) => ({
      id: i + 1,
      title: w.title,
      images: w.images,
      photographerName: '',
    })),
  }
}
