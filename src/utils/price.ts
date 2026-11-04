/**
 * 列表卡片的价格文案。
 *
 * 摄影师自己给套餐定价（price 为 NULL = 面议 / 0 = 互勉 / > 0 = 固定价），一位摄影师
 * 可以同时挂着多种套餐，卡片上只能显示一个词。取闲鱼/淘宝多 SKU 商品的惯例：**显示最低
 * 可成交价**，即 0 元（互勉）优先于固定价，固定价优先于无法比较的面议 —— 也就是
 * 「¥最低价 起」里的「起」必须真的是最低那一档。点进详情页才展开全部套餐。
 */

import type { PriceUnit } from '@/types'

export interface PriceSummary {
  /** 上架套餐里的最低固定价（> 0）；无固定价时为 null */
  minPrice?: number | null
  /** 上架套餐里的最低按张单价（> 0）；无按张套餐时为 null */
  minPricePerPhoto?: number | null
  /** 存在 0 元（互勉）套餐 */
  hasFree?: boolean
  /** 存在面议套餐 */
  hasNegotiable?: boolean
  /** 上架套餐数。**undefined 表示后端没给这份汇总**，与 0（真的没有套餐）不是一回事 */
  serviceCount?: number
}

export type PriceTone = 'price' | 'free' | 'negotiable' | 'muted'

export interface PriceLabel {
  /** 空串 = 不渲染 */
  text: string
  tone: PriceTone
}

export function describePrice(p: PriceSummary): PriceLabel {
  // 首页推荐、漫展详情等接口不返回价格汇总：不显示，也不猜（别把「没查到」说成「暂未设置」）
  if (typeof p.serviceCount !== 'number') {
    return { text: '', tone: 'muted' }
  }
  if (p.serviceCount <= 0) {
    return { text: '暂未设置', tone: 'muted' }
  }
  if (p.hasFree) {
    return { text: '互勉', tone: 'free' }
  }
  if (typeof p.minPrice === 'number' && p.minPrice > 0) {
    return { text: `¥${p.minPrice} 起`, tone: 'price' }
  }
  // 按张单价是补充维度：整套价在前（一次拍摄花多少钱才可比），按张紧随其后，面议垫底
  if (p.minPricePerPhoto != null && p.minPricePerPhoto > 0) {
    return { text: `¥${p.minPricePerPhoto}/张 起`, tone: 'price' }
  }
  if (p.hasNegotiable) {
    return { text: '面议', tone: 'negotiable' }
  }
  return { text: '暂未设置', tone: 'muted' }
}

/* ─────────────── 套餐级单位感知价格渲染（services.vue / 预约页共用） ─────────────── */

export const PRICE_UNITS: readonly PriceUnit[] = ['fixed', 'per_photo', 'per_hour', 'per_day']

/** 计价方式文案。 */
export const PRICE_UNIT_LABELS: Record<PriceUnit, string> = {
  fixed: '整套价',
  per_photo: '按张',
  per_hour: '按小时',
  per_day: '按天'
}

export interface ServicePrice {
  /** null = 面议，0 = 互勉，> 0 = 计价数字 */
  price: number | null
  priceUnit?: PriceUnit | null
  /** 按张时的包含张数；空 = 按需议定 */
  photoCount?: number | null
}

/**
 * 套餐价格文案，带计价单位后缀：
 * fixed→¥599 / per_photo→¥10/张（有张数时「¥10/张 · 3张起」）/ per_hour→¥80/小时 /
 * per_day→¥599/天 / null→面议 / 0→互勉。没有后端价格汇总时不猜。
 */
export function describeServicePrice(s: ServicePrice): string {
  if (s.price == null) {
    return '面议'
  }
  if (s.price <= 0) {
    return '互勉'
  }
  const unit = s.priceUnit ?? 'fixed'
  const suffix = unit === 'per_photo' ? '/张' : unit === 'per_hour' ? '/小时' : unit === 'per_day' ? '/天' : ''
  const base = `¥${s.price}${suffix}`
  if (unit === 'per_photo' && s.photoCount != null && s.photoCount > 0) {
    return `${base} · ${s.photoCount}张起`
  }
  return base
}
