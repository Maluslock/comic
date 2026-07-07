export interface User {
  id: string
  name: string
  avatar: string
  role: 'photographer' | 'coser'
  phone?: string
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
}

export interface Work {
  id: string
  photographerId: string
  title: string
  images: string[]
  description?: string
  tags: string[]
  createdAt: number
}

export interface Service {
  id: string
  name: string
  price: number
  description: string
  duration: number
}

export interface Booking {
  id: string
  photographerId: string
  coserId: string
  serviceId: string
  date: string
  time: string
  status: 'pending' | 'confirmed' | 'completed' | 'cancelled'
  totalPrice: number
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
  startDate: number
  endDate: number
  cover: string
  tags: string[]
  photographerCount: number
  status: 'upcoming' | 'ongoing' | 'ended'
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
