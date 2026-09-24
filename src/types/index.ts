export interface User {
  id: string
  name: string
  avatar: string
  role: 'photographer' | 'coser'
  photographerId?: number
  phone?: string
  bio?: string
  description?: string
  tags: string[]
  location?: string
  createdAt: number
}

export interface Photographer extends User {
  works: Work[]
  services: Service[]
  reviews: Review[]
  rating: number
  reviewCount: number
  orderCount: number
  userId?: number
  /** 价格汇总：仅列表接口（/v1/photographers）返回；其余接口为 undefined → 卡片不显示价格 */
  minPrice?: number | null
  hasFree?: boolean
  hasNegotiable?: boolean
  serviceCount?: number
}

export interface Work {
  id: string
  photographerId: string
  title: string
  images: string[]
  description?: string
  tags: string[]
  createdAt: number
  photographerName?: string
}

export interface Service {
  id: string
  name: string
  /** null = 面议/negotiable, 0 = 互勉/mutual, >0 = 固定价/fixed */
  price: number | null
  description: string
  duration: number
}

/** Pricing mode derived from the booked package: fixed price / mutual (free) / negotiable. */
export type PriceMode = 'fixed' | 'mutual' | 'negotiable'

/** Single-round quote lifecycle for negotiable bookings. */
export type PriceStatus = 'agreed' | 'awaiting_quote' | 'quoted' | 'rejected'

/** Fields every order view carries so the amount can be rendered by state. */
export interface OrderPriceFields {
  priceMode: PriceMode
  priceStatus: PriceStatus
  quotePrice: number | null
  totalPrice: number
}

export interface Booking extends OrderPriceFields {
  id: string
  photographerId: string
  coserId: string
  serviceId: string
  date: string
  time: string
  status: 'pending' | 'confirmed' | 'completed' | 'cancelled'
  remarks?: string
  createdAt: number
}

export interface Review {
  id: string
  userId: string
  userName: string
  userAvatar: string
  photographerId: string
  rating: number
  content: string
  images?: string[]
  createdAt: number
}

export interface Message {
  id: string
  senderId: string
  receiverId: string
  content: string
  type: 'text' | 'image'
  status: 'sent' | 'delivered' | 'read'
  createdAt: number
}

export interface ChatSession {
  id: string
  userId: string
  userName: string
  userAvatar: string
  lastMessage: string
  unreadCount: number
  updatedAt: number
}

export interface ComicEvent {
  id: string
  name: string
  location: string
  venue: string
  address?: string
  startDate: number
  endDate: number
  cover: string
  tags: string[]
  imageGallery?: string[]
  photographerCount: number
  status: 'upcoming' | 'ongoing' | 'ended'
  description?: string
}

// Backend HomeResponse DTO types
export interface BannerItem {
  id: number
  imageUrl: string
  title: string
  linkType: string
  linkId: number | null
}

export interface EventItem {
  id: number
  name: string
  location: string
  venue: string
  startDate: string  // ISO8601 from backend
  endDate: string    // ISO8601 from backend
  coverUrl: string
  tags: string[]
  status: string
  typeName: string
}

export interface TagItem {
  name: string
  usageCount: number
}

export interface PhotographerItem {
  id: number
  name: string
  avatar: string
  location: string
  rating: number
  reviewCount: number
  orderCount: number
  tags: string[]
  /** 见 Photographer.minPrice：仅列表接口返回 */
  minPrice?: number | null
  hasFree?: boolean
  hasNegotiable?: boolean
  serviceCount?: number
}

export interface WorkItem {
  id: number
  title: string
  images: string[]
  photographerName: string
}

export interface HomeResponse {
  banners: BannerItem[]
  upcomingEvents: EventItem[]
  hotTags: TagItem[]
  recommendedPhotographers: PhotographerItem[]
  featuredWorks: WorkItem[]
}
