import type { ComicEvent, Photographer, Work, User, Review } from '@/types'
import { getApiBase } from './apiBase'

/**
 * Render a service price in its tri-state form:
 *   null / undefined -> 面议 (negotiable)
 *   0                -> 互勉 (mutual / free collaboration)
 *   > 0              -> ¥<price>
 */
export function formatPrice(price: number | null | undefined): string {
  if (price === null || price === undefined) return '面议'
  if (price === 0) return '互勉'
  return `¥${price}`
}

/**
 * 图片地址解析：后端把封面托管在 /static/remote/**，H5 由 vite 代理同源；
 * 小程序端没有同源概念，构建时给了 VITE_API_BASE_URL（绝对地址），这里补上 origin。
 * 本地静态图（/static/img、/static/tab 等）原样返回，避免误伤。
 */
export function resolveImage(url: string): string {
  if (!url) return ''
  // 只处理后端托管的三类前缀；tab 图标等包内资源原样返回
  // 只有后端托管的 remote/uploads 需要补 origin；/static/img 是包内资源，补了反而 404。
  if (/^(https?:)?\/\//.test(url) || url.startsWith('data:') || !/^\/static\/(remote|uploads)\//.test(url)) return url
  // 构建期变量优先级更高；未注入时回落到运行时基址（storage 可切内网/域名）。
  // 小程序包内没有 /static/remote，必须拿到绝对 origin，否则会被当成本地文件读取（500）。
  const base = (import.meta.env.VITE_API_BASE_URL as string | undefined) || getApiBase()
  const media = (import.meta.env.VITE_MEDIA_ORIGIN as string | undefined) || ''
  if (/^https?:\/\//.test(media)) return media.replace(/\/$/, '') + url
  // 小程序运行时没有 URL 构造器（部分基础库），用正则取 origin，别用 new URL。
  const m = /^(https?:\/\/[^/]+)/i.exec(base)
  if (m) return m[1] + url
  // 相对 /api（H5 dev 代理）或纯静态部署：媒体走同源，由 vite/nginx 把
  // /static/remote 反代到 API；生产未配反代时需设 VITE_MEDIA_ORIGIN。
  return url
}

export interface LoginUserDTO {
  id: number
  name: string
  phone: string
  avatar: string
  bio?: string
  photographerId?: number
}

/**
 * 页面路径 + query 的安全拼接：小程序没有 URL/URLSearchParams 构造器，
 * query 值必须手写 encodeURIComponent。redirect 里嵌套的 /xx?id=1 若不整体编码，
 * 登录页解析 options 时会把 id 当成登录页自己的参数，跳回后丢 id 落到坏页面。
 */
export function withQuery(path: string, params: Record<string, string | number | undefined>): string {
  const parts: string[] = []
  Object.keys(params).forEach((k) => {
    const v = params[k]
    if (v === undefined || v === null || v === '') return
    parts.push(encodeURIComponent(k) + '=' + encodeURIComponent(String(v)))
  })
  return parts.length ? `${path}?${parts.join('&')}` : path
}

/** 编码完整页面路径（含自身 query），用于塞进 redirect= 参数。 */
export function encodeRedirect(path: string): string {
  return encodeURIComponent(path)
}

/** 解码 redirect= 的值（与 encodeRedirect 配对使用；小程序 options 无内置解码）。
 * 带非法 % 序列的值（如未编码的老跳转方传入了含 % 的串）会抛 URIError，兜底原样返回。 */
export function decodeRedirect(raw: string): string {
  try {
    return decodeURIComponent(raw)
  } catch {
    return raw
  }
}

export interface EventDTO {
  id: number | string
  name: string
  location: string
  venue: string
  address?: string
  /** 后端给 ISO8601 字符串；历史代码里也有时间戳 */
  startDate: string | number
  endDate: string | number
  coverUrl?: string
  tags?: string[]
  imageGallery?: string[]
  photographers?: unknown[]
  status?: string
  description?: string
}

export interface PhotographerDTO {
  id: number | string
  name: string
  avatar?: string
  description?: string
  rating?: number
  reviewCount?: number
  orderCount?: number
  location?: string
  tags?: string[]
  userId?: number | string
  minPrice?: number | null
  hasFree?: boolean
  hasNegotiable?: boolean
  serviceCount?: number
}

export interface ReviewDTO {
  id: number | string
  userId?: number | string
  userName?: string
  userAvatar?: string
  photographerId?: number | string
  rating?: number
  content?: string
  createdAt?: string | number
}

export interface WorkDTO {
  id: number | string
  title: string
  images?: string[]
  photographerName?: string
}

export interface BannerDTO {
  id: number
  imageUrl: string
  title: string
  linkId?: number | null
}

export function mapLoginUser(u: LoginUserDTO): User {
  return {
    id: String(u.id),
    name: u.name,
    avatar: resolveImage(u.avatar || ''),
    phone: u.phone,
    bio: u.bio || '',
    role: 'coser',
    photographerId: u.photographerId,
    tags: [],
    createdAt: Date.now(),
  }
}

/**
 * Map backend EventItem DTO to frontend ComicEvent.
 * Handles both HomeResponse.upcomingEvents and EventDetailResponse.
 */
export function mapEventItem(e: EventDTO): ComicEvent {
  return {
    id: String(e.id),
    name: e.name,
    location: e.location,
    venue: e.venue,
    address: e.address || '',
    startDate: new Date(e.startDate).getTime(),
    endDate: new Date(e.endDate).getTime(),
    cover: resolveImage(e.coverUrl || ''),
    tags: e.tags || [],
    imageGallery: e.imageGallery || [],
    photographerCount: e.photographers?.length || 0,
    status: (e.status as ComicEvent['status']) || 'upcoming',
    description: e.description || '',
  }
}

/**
 * Map backend PhotographerItem DTO to frontend Photographer.
 */
export function mapPhotographerItem(p: PhotographerDTO): Photographer {
  return {
    id: String(p.id),
    name: p.name,
    avatar: resolveImage(p.avatar || ''),
    role: 'photographer' as const,
    description: p.description || '',
    rating: p.rating ?? 0,
    reviewCount: p.reviewCount ?? 0,
    orderCount: p.orderCount ?? 0,
    location: p.location || '',
    tags: p.tags || [],
    userId: typeof p.userId === 'number' ? p.userId : undefined,
    works: [],
    services: [],
    reviews: [],
    createdAt: Date.now(),
    // 价格汇总：只有列表接口给；缺失时保持 undefined（而不是 0），卡片据此决定不显示价格
    minPrice: typeof p.minPrice === 'number' ? p.minPrice : null,
    hasFree: p.hasFree === true,
    hasNegotiable: p.hasNegotiable === true,
    serviceCount: typeof p.serviceCount === 'number' ? p.serviceCount : undefined,
  }
}

/**
 * Map backend ReviewItem DTO to frontend Review.
 */
export function mapReviewItem(r: ReviewDTO): Review {
  return {
    id: String(r.id),
    userId: String(r.userId || ''),
    userName: r.userName || '',
    userAvatar: resolveImage(r.userAvatar || ''),
    photographerId: String(r.photographerId || ''),
    rating: r.rating || 0,
    content: r.content || '',
    createdAt: typeof r.createdAt === 'string' ? new Date(r.createdAt).getTime() : (r.createdAt || Date.now()),
  }
}

/**
 * Map backend WorkItem DTO to frontend Work.
 */
export function mapWorkItem(w: WorkDTO): Work {
  return {
    id: String(w.id),
    photographerId: '',
    title: w.title,
    images: (w.images || []).map(resolveImage),
    description: '',
    tags: [],
    createdAt: Date.now(),
    photographerName: w.photographerName || '',
  }
}

/**
 * Map backend BannerItem DTO to banner display object.
 */
export function mapBannerItem(b: BannerDTO): {
  id: number
  image: string
  title: string
  linkId?: number | null
} {
  return {
    id: b.id,
    image: resolveImage(b.imageUrl),
    title: b.title,
    linkId: b.linkId,
  }
}
