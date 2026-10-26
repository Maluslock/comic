import type { ComicEvent, Photographer, Work, User, Review } from '@/types'

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
  if (/^(https?:)?\/\//.test(url) || url.startsWith('data:') || !/^\/static\/(remote|img|uploads)\//.test(url)) return url
  const base = (import.meta.env.VITE_API_BASE_URL as string | undefined) || ''
  const media = (import.meta.env.VITE_MEDIA_ORIGIN as string | undefined) || ''
  if (/^https?:\/\//.test(media)) return media.replace(/\/$/, '') + url
  if (/^https?:\/\//.test(base)) {
    try {
      return new URL(base).origin + url
    } catch {
      return url
    }
  }
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
