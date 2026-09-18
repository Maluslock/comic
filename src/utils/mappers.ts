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

export interface LoginUserDTO {
  id: number
  name: string
  phone: string
  avatar: string
  bio?: string
  photographerId?: number
}

export function mapLoginUser(u: LoginUserDTO): User {
  return {
    id: String(u.id),
    name: u.name,
    avatar: u.avatar,
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
export function mapEventItem(e: any): ComicEvent {
  return {
    id: String(e.id),
    name: e.name,
    location: e.location,
    venue: e.venue,
    address: e.address || '',
    startDate: new Date(e.startDate).getTime(),
    endDate: new Date(e.endDate).getTime(),
    cover: e.coverUrl,
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
export function mapPhotographerItem(p: any): Photographer {
  return {
    id: String(p.id),
    name: p.name,
    avatar: p.avatar || '',
    role: 'photographer' as const,
    description: p.description || '',
    rating: p.rating,
    reviewCount: p.reviewCount,
    orderCount: p.orderCount,
    location: p.location || '',
    tags: p.tags || [],
    userId: p.userId,
    works: [],
    services: [],
    reviews: [],
    createdAt: Date.now(),
  }
}

/**
 * Map backend ReviewItem DTO to frontend Review.
 */
export function mapReviewItem(r: any): Review {
  return {
    id: String(r.id),
    userId: String(r.userId || ''),
    userName: r.userName || '',
    userAvatar: r.userAvatar || '',
    photographerId: String(r.photographerId || ''),
    rating: r.rating || 0,
    content: r.content || '',
    createdAt: typeof r.createdAt === 'string' ? new Date(r.createdAt).getTime() : (r.createdAt || Date.now()),
  }
}

/**
 * Map backend WorkItem DTO to frontend Work.
 */
export function mapWorkItem(w: any): Work {
  return {
    id: String(w.id),
    photographerId: '',
    title: w.title,
    images: w.images || [],
    description: '',
    tags: [],
    createdAt: Date.now(),
  }
}

/**
 * Map backend BannerItem DTO to banner display object.
 */
export function mapBannerItem(b: any): {
  id: number
  image: string
  title: string
  linkId?: number | null
} {
  return {
    id: b.id,
    image: b.imageUrl,
    title: b.title,
    linkId: b.linkId,
  }
}
