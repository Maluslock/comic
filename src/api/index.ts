import { mockPhotographers, mockWorks, mockEvents, tags } from '@/data/mock'
import type { PriceUnit } from '@/types'
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

/**
 * 上传图片，返回可直接入库的绝对 URL。
 *
 * 这里刻意把**每一种失败原因**都翻译成人话再抛出，因为小程序端 uploadFile
 * 的失败不会进 Network 面板，也不会打到服务端 —— 服务端日志一片空白，
 * 用户只看到「上传失败」。三种真实成因完全不同，必须区分：
 *
 * 1) 未登录（无 token）→ 服务端 401，请求确实发出去了；
 * 2) 合法域名（uploadFile 白名单）没配 → **请求根本没发出去**，
 *    errMsg 形如 `uploadFile:fail url not in domain list`。
 *    注意 request 与 uploadFile 是**两套独立白名单**：登录能通不代表上传能通；
 * 3) 服务端 4xx（如魔数不是图片）→ 会带回 error 字段，不能吞掉。
 */
/**
 * 把本地临时文件读成 base64（去掉 DataURL 前缀）。
 * 用于 `/v1/upload/base64` 通道：它走普通 request，不经过 uploadFile 白名单。
 */
function readFileAsBase64(filePath: string): Promise<string> {
  return new Promise((resolve, reject) => {
    // H5 的 uni 对象**没有** getFileSystemManager（实测 uni-h5.es.js 里零命中），
    // 硬调会抛 "is not a function"。这里提前判掉，让 H5 直接走 multipart 通道，
    // 不要靠 try/catch 兜底掩盖真实原因。
    const fsm = typeof uni.getFileSystemManager === 'function' ? uni.getFileSystemManager() : null
    if (!fsm) {
      reject(new Error('当前平台不支持读取本地文件'))
      return
    }
    fsm.readFile({
      filePath,
      encoding: 'base64',
      success: (res) => {
        const data = typeof res.data === 'string' ? res.data : ''
        if (!data) {
          reject(new Error('读取图片失败'))
          return
        }
        resolve(data)
      },
      fail: (err) => {
        const raw = String((err && (err as AnyObject).errMsg) || '')
        reject(new Error(raw || '读取图片失败'))
      }
    })
  })
}

/**
 * 上传图片，返回可直接入库的绝对 URL。
 *
 * 两条通道，按顺序尝试：
 *   ① `POST /v1/upload/base64`（走 uni.request）—— 主路径。
 *      小程序里 uni.uploadFile 用的是**独立于 uni.request 的「uploadFile 合法域名」白名单**。
 *      现场症状：登录（request）一切正常，头像（uploadFile）永远失败、**服务端零日志**——
 *      请求在客户端就被拦了，根本发不出去。base64 通道复用 request 白名单，绕开这个坑。
 *   ② `POST /v1/upload`（multipart）—— 兜底。base64 失败时再试，两种情况互补：
 *      若白名单正常而 base64 因体积大失败，走这条能成；反之亦然。
 *
 * 两条都失败时抛出**最有信息量**的那条错误，让用户/开发者看到真实成因
 * （未登录 / 域名未配置 / 格式不支持 / 超时），而不是笼统的「上传失败」。
 */
export async function uploadFile(filePath: string): Promise<string> {
  const token = uni.getStorageSync('token') as string
  if (!token) {
    // 提前失败，避免发一个注定 401 的请求再被上层 catch 静默吞掉
    throw new Error('未登录，请先登录再上传')
  }

  const errors: string[] = []

  // ① base64 通道（request 白名单）
  try {
    const b64 = await readFileAsBase64(filePath)
    const res = await apiPost<{ url: string }>('/v1/upload/base64', { data: b64 })
    if (res?.url) return resolveImage(res.url)
    errors.push('base64 通道返回空 url')
  } catch (e) {
    errors.push(e instanceof Error ? e.message : 'base64 通道失败')
  }

  // ② multipart 通道（uploadFile 白名单）
  try {
    return await uploadViaMultipart(filePath, token)
  } catch (e) {
    errors.push(e instanceof Error ? e.message : 'multipart 通道失败')
  }

  throw new Error(errors.filter(Boolean).join('；') || '上传失败，请重试')
}

/** multipart 上传（uni.uploadFile）。保留给「uploadFile 白名单已配好」的环境。 */
function uploadViaMultipart(filePath: string, token: string): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    uni.uploadFile({
      url: `${getApiBase()}/v1/upload`,
      filePath,
      name: 'file',
      header: { Authorization: `Bearer ${token}` },
      success: (res) => {
        if (res.statusCode !== 200 && res.statusCode !== 201) {
          // 后端把原因放在 error 字段，透出去否则用户永远只看到「上传失败」
          let detail = ''
          try {
            detail = (JSON.parse(res.data) as { error?: string }).error || ''
          } catch { /* 非 JSON 响应，保持空 */ }
          reject(new Error(detail || `HTTP ${res.statusCode}`))
          return
        }
        try {
          resolve(resolveImage((JSON.parse(res.data) as { url: string }).url))
        } catch {
          reject(new Error('响应解析失败'))
        }
      },
      fail: (err) => {
        const raw = String((err && (err as AnyObject).errMsg) || '')
        if (raw.includes('domain list')) {
          reject(new Error('uploadFile 合法域名未配置'))
          return
        }
        reject(new Error(raw || 'uploadFile 失败'))
      }
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

/**
 * 只更新头像，不需要 nickName。
 *
 * 单独开这个端点是因为「登录后补传头像」这条链路本来就没有昵称要改，
 * 硬走 /v1/me/profile 会被后端 name 的 binding:"required" 直接 400——
 * 这也是「头像上传成功但库里还是空」的成因之一。
 * 后端会同时把头像同步到该用户的摄影师档案（photographers.avatar）。
 */
export function updateMyAvatar(avatar: string) {
  return apiPut<{ ok: boolean; avatar: string }>('/v1/me/avatar', { avatar })
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
  /** null = 面议, 0 = 互勉, >0 = 计价数字 */
  price: number | null
  description: string
  duration: number
  /** 计价方式：fixed / per_photo / per_hour / per_day */
  priceUnit: PriceUnit
  /** 按张时的包含张数；null = 按需议定 */
  photoCount: number | null
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
  priceUnit: PriceUnit
  /** 按张的包含张数；传数字或省略，绝不传空串（后端 *int32 空串会 400） */
  photoCount: number | null
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
export const CREDIT_USE_MOCK = false

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

export interface PhotographerReview {
  id: number
  rating: number
  content: string
  userName: string
  userAvatar?: string
  images?: string[]
  createdAt: string
}

export interface ReviewPage {
  list: PhotographerReview[]
  total: number
  hasMore: boolean
}

/** 摄影师评价分页（后端 GET /v1/reviews/:photographerId?limit&offset）。 */
export async function getPhotographerReviews(
  photographerId: number | string,
  limit = 10,
  offset = 0
): Promise<ReviewPage> {
  return apiGet<ReviewPage>(`/v1/reviews/${photographerId}?limit=${limit}&offset=${offset}`)
}

/* ───────────────────── 摄影师入驻漫展（申请 → 审核） ───────────────────── */

/** 一条入驻申请的 C 端视图。status: pending | approved | rejected | withdrawn。 */
export interface EventApplication {
  id: number
  eventId: number
  status: string
  message?: string | null
  reviewReason?: string | null
  appliedAt: string
  reviewedAt?: string | null
}

/** 我的入驻列表项：申请 + 漫展卡片字段（后端一次 join 带回来，前端不必二次拉取）。 */
export interface MyEventApplication extends EventApplication {
  eventName: string
  startDate?: string | null
  endDate?: string | null
  location?: string | null
  venue?: string | null
  coverUrl?: string | null
}

/** 提交「摄影师入驻某漫展」申请。message 是可选的自我介绍（设备/出图时效/样片链接）。 */
export function applyEventPhotographer(eventId: number | string, message: string) {
  return apiPost<{ id: number; eventId: number; status: string; appliedAt: string }>(
    `/v1/events/${eventId}/photographer-apply`,
    { message }
  )
}

/**
 * 我在某漫展的最近一条申请。
 * 从未申请时后端返回 200 { application: null }（不是 404），据此显示「申请入驻」按钮。
 */
export function getMyEventApplication(eventId: number | string) {
  return apiGet<{ application: EventApplication | null }>(`/v1/events/${eventId}/photographer-apply/mine`)
}

/** 撤回我在某漫展的申请（pending/approved → withdrawn）。 */
export function withdrawEventApplication(eventId: number | string) {
  return apiDelete<{ ok: boolean }>(`/v1/events/${eventId}/photographer-apply`)
}

/** 我的全部入驻申请（按提交时间倒序，含已驳回/已撤回历史）。 */
export function getMyEventApplications() {
  return apiGet<{ list: MyEventApplication[] | null }>('/v1/photographer/event-applications')
}

/** 城市选择器的一行：城市名 + 该城市在办漫展场次数。 */
export interface CityItem {
  name: string
  eventCount: number
}

/**
 * 全量城市清单（按在办场次降序）。
 *
 * 不能用首页 `/v1/home` 里的 50 场 events 在前端推导城市 —— 那样只能得到
 * 首页那一页出现过的二十来个城市，全库有 73 个（见 server/internal/repository/cities.go）。
 * 后端只统计 del_flag=false 且 end_date>=NOW() 的场次，与 C 端列表口径一致。
 */
export function getCities() {
  return apiGet<{ list: CityItem[]; total: number }>('/v1/cities')
}
