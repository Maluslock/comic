import { mockPhotographers, mockWorks, mockEvents, tags } from '@/data/mock'
import { apiGet, apiPost, apiPut, apiDelete } from '@/api/client'
import type { HomeResponse } from '@/types'

const BASE_URL = '/api'

async function uploadFile(filePath: string): Promise<string> {
  const token = uni.getStorageSync('token') as string

  return new Promise<string>((resolve, reject) => {
    uni.uploadFile({
      url: `${BASE_URL}/v1/upload`,
      filePath,
      name: 'file',
      header: token ? { Authorization: `Bearer ${token}` } : {},
      success: (res) => {
        if (res.statusCode !== 200 && res.statusCode !== 201) {
          reject(new Error('upload failed'))
          return
        }
        try {
          resolve((JSON.parse(res.data) as { url: string }).url)
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
  /** backend `mine` endpoint does not return this yet; optional for forward-compat */
  isActive?: boolean
}

export function getMyServices() {
  return apiGet<{ list: MyService[] | null }>('/v1/photographers/services/mine')
}

export function getServiceTemplates() {
  return apiGet<MyService[] | null>('/v1/services/templates')
}

export function createService(payload: {
  name: string
  price: number | null
  description: string
  duration: number
}) {
  return apiPost<{ id: number }>('/v1/photographers/services', payload)
}

export function updateService(
  id: number,
  payload: { name: string; price: number | null; description: string; duration: number }
) {
  return apiPut<{ ok: boolean }>(`/v1/photographers/services/${id}`, payload)
}

export function deleteService(id: number) {
  return apiDelete<{ ok: boolean }>(`/v1/photographers/services/${id}`)
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
