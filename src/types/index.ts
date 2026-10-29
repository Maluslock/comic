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

/* ───────────────────────── 担保交易（P0 履约节点） ───────────────────────── */

/**
 * 担保交易的履约子状态。刻意与 `Booking.status` 分开：
 * status 是粗状态（pending/confirmed/completed/cancelled），被状态机、列表 tab、
 * 「已完成订单才能评价」判定和档期唯一索引四处硬依赖，不能塞新值。
 * 子状态只在 status === 'confirmed' 期间有值。
 */
export type BookingSubStatus = 'in_service' | 'delivered' | 'disputed'

/** 订单接口返回的担保交易附加字段（后端 BookingItem DTO 的 camelCase 字段）。 */
export interface EscrowBookingFields {
  /** in_service（待核销/拍摄中） | delivered（待确认交付） | disputed（争议中） | null */
  subStatus?: BookingSubStatus | null
  /** 核销码：只有 coser 视角（/v1/bookings/:id）返回，摄影师视角为 null */
  checkinCode?: string | null
  /** 核销时间 ISO8601 */
  checkinAt?: string | null
  /** 交付时间 ISO8601 */
  deliveredAt?: string | null
  /** coser 需在此时间前确认交付，逾期系统自动完成 */
  acceptDeadline?: string | null
  /** 摄影师提交的交付凭证（图片 URL） */
  deliveryEvidence?: string[]
  /** 爽约方：coser | photographer | both */
  noShowParty?: string | null
  /** 取消方：coser | photographer | system */
  cancelledBy?: string | null
}

/** 订单事件（GET /v1/bookings/:id/events 的 list 元素）。 */
export type OrderEventType =
  | 'confirmed'
  | 'checkin'
  | 'deliver'
  | 'accept'
  | 'auto_accept'
  | 'complete_legacy'
  | 'cancel'
  | 'auto_cancel'
  | 'noshow'
  | 'dispute_open'
  | 'dispute_respond'
  | 'dispute_escalate'
  | 'dispute_resolve'
  | 'credit_delta'

export interface OrderEventItem {
  id: number
  eventType: OrderEventType | string
  /** 操作方角色：coser | photographer | system | admin */
  actorRole?: string | null
  actorId?: number | null
  /** ISO8601 */
  createdAt: string
  /** 真正的 JSON 对象（后端已从 JSONB 反序列化） */
  payload?: Record<string, unknown> | null
}

/** 争议类型：爽约 / 未交付 / 质量问题 / 其他。 */
export type DisputeType = 'no_show' | 'not_delivered' | 'quality' | 'other'

/** 争议状态：open 待响应 → responding 已响应 → escalated 已升级 → resolved 系统默认裁决 → closed 管理员裁决。 */
export type DisputeStatus = 'open' | 'responding' | 'escalated' | 'resolved' | 'closed'

export interface DisputeItem {
  id: number
  bookingId: number
  openedBy: number
  openedByRole: 'coser' | 'photographer'
  type: DisputeType | string
  reason: string
  status: DisputeStatus | string
  /** 响应截止（ISO8601），open → responding 的截止时间 */
  respondDeadline: string
  /** 举证截止（ISO8601） */
  evidenceDeadline: string
  /** 裁决说明（管理员填写） */
  resolution?: string | null
  createdAt: string
  updatedAt: string
}

/** GET /v1/bookings/:id/dispute 的返回体。 */
export interface BookingDisputeResponse {
  dispute: DisputeItem | null
  /** 当前用户在该订单中的角色（服务端判定） */
  myRole: 'coser' | 'photographer'
}

/**
 * 信用等级。
 * 后端阈值：>=90 good、>=70 fair、其余 poor（与 docs/担保交易-最终方案.md 第六节一致）。
 */
export type CreditLevel = 'good' | 'fair' | 'poor'

/**
 * 信用分变动原因。
 * 后端返回库里的枚举原文；接口层可能再抽象成 success/cancel 语义，
 * 前端文案映射两套都要覆盖。
 */
export type CreditReason =
  | 'success'
  | 'order_completed'
  | 'cancel'
  | 'late_cancel'
  | 'dispute_lost'
  | 'coser_no_show'
  | 'photographer_no_show'
  | 'delivery_overdue'
  | (string & {})

/** GET /v1/me/credit 的返回体（后端无记录时返回默认 100 分，不 404）。 */
export interface CreditSummary {
  /** 当前信用分，基准 100 */
  score: number
  /** 等级，由 score 推导 */
  level: CreditLevel
  /** 成功订单数 */
  successCount: number
  /** 取消订单数 */
  cancelCount: number
  /** 爽约次数 */
  noshowCount: number
  /** 争议败诉次数 */
  disputeLostCount: number
  /** 冻结截止时间（ISO8601），未冻结为空 */
  frozenUntil?: string | null
  /** 最后更新时间（ISO8601） */
  updatedAt?: string | null
}

/** credit_events 的一行。 */
export interface CreditEventItem {
  id: number
  /** 增减分，正数加分负数减分 */
  delta: number
  /** 原因枚举原文 */
  reason: CreditReason
  /** 关联对象类型：booking | dispute */
  refType?: string | null
  /** 关联对象 id */
  refId?: number | null
  /** 关联订单 id */
  bookingId?: number | null
  /** 后端补充说明 */
  description?: string | null
  /** 发生时间（ISO8601） */
  createdAt: string
}

/** GET /v1/me/credit/events 的返回体。 */
export interface CreditEventsResponse {
  list: CreditEventItem[]
  /** 总条数 */
  total: number
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
  /** 详细地址（详情页/日历展示用，列表接口可能缺省） */
  address?: string
  startDate: string  // ISO8601 from backend
  endDate: string    // ISO8601 from backend
  coverUrl: string
  tags: string[]
  status: string
  typeName: string
  /** 详情接口附带的图片集与简介 */
  imageGallery?: string[]
  description?: string
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
