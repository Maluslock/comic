<template>
  <view class="page">
    <template v-if="firstLoading">
      <view class="skeleton-page">
        <view class="sk-card sk-detail-header">
          <view class="sk-circle sk-detail-avatar" />
          <view class="sk-detail-info">
            <view class="sk-line sk-w-40" />
            <view class="sk-line sk-w-25" />
            <view class="sk-detail-stats">
              <view class="sk-detail-stat"><view class="sk-line sk-w-60" /><view class="sk-line sk-w-40" /></view>
              <view class="sk-detail-stat"><view class="sk-line sk-w-60" /><view class="sk-line sk-w-40" /></view>
              <view class="sk-detail-stat"><view class="sk-line sk-w-60" /><view class="sk-line sk-w-40" /></view>
            </view>
          </view>
        </view>
        <view class="sk-detail-tags"><view class="sk-block sk-detail-tag" /><view class="sk-block sk-detail-tag" /><view class="sk-block sk-detail-tag" /></view>
        <view class="sk-card sk-detail-section"><view class="sk-line sk-w-30" /><view class="sk-line sk-w-full" /><view class="sk-line sk-w-80" /></view>
        <view class="sk-card sk-detail-section"><view class="sk-line sk-w-30" /><ListSkeleton :rows="2" variant="row" /></view>
        <view class="sk-card sk-detail-section"><view class="sk-line sk-w-30" /><view class="sk-detail-works"><view class="sk-block sk-detail-work" /><view class="sk-block sk-detail-work" /><view class="sk-block sk-detail-work" /></view></view>
        <view class="sk-card sk-detail-section"><view class="sk-line sk-w-30" /><ListSkeleton :rows="2" variant="row" /></view>
      </view>
    </template>
    <template v-else>
    <scroll-view scroll-y class="content">
      <view class="header">
        <image v-if="!avatarFailed" :src="photographer?.avatar" class="avatar" mode="aspectFill" @error="onAvatarError" />
        <view v-else class="avatar avatar--fallback">
          <text>{{ (photographer?.name || '摄')[0] }}</text>
        </view>
        <view class="info">
          <view class="name-row">
            <text class="name">{{ photographer?.name }}</text>
            <text v-if="photographer?.certified" class="cert-pill">V 认证</text>
            <text class="more-btn" @click="onMore">···</text>
          </view>
          <text v-if="photographer?.location" class="name-meta">{{ photographer.location }}</text>
          <view class="stats">
            <view class="stat-item">
              <view class="stat-value-row">
                <text v-if="photographer?.reviewCount" class="stat-star">★</text>
                <text class="stat-value" :class="{ 'stat-value--warn': !!photographer?.reviewCount }">{{ ratingText }}</text>
              </view>
              <text class="stat-label">综合评分</text>
              <text v-if="creditLabel" class="credit-pill" :class="`tone-${creditTone}`">信用{{ creditLabel }}</text>
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <view class="stat-value-row">
                <text class="stat-value">{{ photographer?.reviewCount }}</text>
              </view>
              <text class="stat-label">评价</text>
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <view class="stat-value-row">
                <text class="stat-value">{{ photographer?.orderCount }}</text>
              </view>
              <text class="stat-label">接单</text>
            </view>
          </view>
        </view>
      </view>

      <view class="tags-row">
        <view v-if="photographer?.mode === 'free'" class="mode-tag">互勉</view>
        <view v-else-if="photographer?.mode === 'pay'" class="mode-tag pay">收费</view>
        <view v-else-if="photographer?.mode" class="mode-tag both">互勉 / 收费</view>
        <text 
          v-for="tag in photographer?.tags" 
          :key="tag" 
          class="tag"
        >{{ tag }}</text>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">关于我</text>
        </view>
        <text class="description">{{ photographer?.description }}</text>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">服务套餐</text>
        </view>
        <view class="service-list">
          <view 
            v-for="service in photographer?.services" 
            :key="service.id" 
            class="service-item"
            @click="selectService(service)"
          >
            <view class="service-info">
              <text class="service-name">{{ service.name }}</text>
              <text class="service-desc">{{ service.description }}{{ showServiceDuration(service) ? ' · ' + formatDuration(service.duration) : '' }}</text>
            </view>
            <view class="service-price">
              <text class="price">{{ describeServicePrice(service) }}</text>
              <text class="btn-ghost">选择</text>
            </view>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">作品展示</text>
          <text class="section-more" @click="goPortfolio">查看全部</text>
        </view>
        <scroll-view scroll-x class="work-scroll" show-scrollbar="false">
          <view class="work-list">
            <view 
              v-for="work in photographer?.works" 
              :key="work.id" 
              class="work-item"
            >
              <image v-if="work.images?.length" :src="work.images[0]" class="work-image" mode="aspectFill" lazy-load @click.stop="previewWorkImages(work, 0)" />
              <view v-else class="work-image work-image--empty"><text>暂无图片</text></view>
              <text class="work-title">{{ work.title }}</text>
            </view>
          </view>
        </scroll-view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">用户评价</text>
          <text class="section-more">{{ reviewCount }} 条</text>
        </view>
        <view class="review-list">
          <ReviewCard
            v-for="review in reviews"
            :key="review.id"
            :review="review"
          />
        </view>
        <view v-if="!reviews.length" class="review-empty">还没有评价</view>
        <view v-else-if="reviewHasMore" class="review-toggle" @click="loadReviews(false)">
          {{ reviewLoading ? '加载中…' : `加载更多（已显示 ${reviews.length}/${reviewTotal}）` }}
        </view>
        <view v-else class="review-end">已显示全部 {{ reviewTotal }} 条</view>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>

    <view class="footer">
      <view class="footer-left">
        <view class="footer-btn" @click="goChat">
          <image class="footer-icon-img" src="/static/icons/chat.png" mode="aspectFit" />
          <text>聊天</text>
        </view>
        <view class="footer-btn" @click="collect">
          <image class="footer-icon-img" :src="collected ? '/static/icons/star-filled.png' : '/static/icons/star.png'" mode="aspectFit" />
          <text>收藏</text>
        </view>
      </view>
      <view class="footer-right">
        <view class="btn-primary" @click="goBooking">立即预约</view>
      </view>
    </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ReviewCard from '@/components/ReviewCard.vue'
import { apiGet, apiPost, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'
import { blockUser, getPhotographerReviews } from '@/api/index'
import { mapPhotographerItem, mapWorkItem, mapReviewItem, encodeRedirect } from '@/utils/mappers'
import { describeServicePrice } from '@/utils/price'
import { previewImages } from '@/utils/preview'
import { formatDuration } from '@/utils/duration'
import { creditLevelShort, creditTone as creditToneOf } from '@/utils/credit'
import { navTo } from '@/utils/nav'
import type { Photographer, PriceUnit, Review, Service, Work } from '@/types'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

interface PhotographerDetailResponse {
  id: number
  name: string
  avatar: string
  location: string
  rating: number
  reviewCount: number
  orderCount: number
  userId: number
  mode?: string
  certified?: boolean
  /** 信用分（后端信用分接口落地后返回，缺失时前端不展示信用标签） */
  creditScore?: number
  /** 信用等级：good | fair | poor */
  creditLevel?: string
  tags: string[]
  description: string
  // priceUnit / photoCount 必须留在这里：selectService() 会把整个 service 对象
  // 序列化进 storage 传给预约页，缺这两个字段会让预约页的 isPerPhoto 恒 false，
  // 按张套餐就会被当成「整套 ¥10」直接成交。
  services: Array<{
    id: number
    name: string
    price: number | null
    description: string
    duration: number
    priceUnit?: PriceUnit
    photoCount?: number | null
  }>
  works: Array<{ id: number; title: string; images: string[]; photographerName?: string }>
  reviews: Array<{ id: number; rating: number; content: string; userName: string; userAvatar?: string; images?: string[]; createdAt: string }>
}

type PhotographerView = Photographer & {
  mode?: string
  certified?: boolean
  creditScore?: number
  creditLevel?: string
}

const userStore = useUserStore()
const photographer = ref<PhotographerView | null>(null)

// 评分按真实评价重算：没有评价时后端返回 rating=0 / reviewCount=0。
// 直接显示「0」会被读成「被打 0 分」，故无评价时给文案。
const ratingText = computed(() => ((photographer.value?.reviewCount ?? 0) > 0 ? String(photographer.value?.rating) : '暂无'))

// 信用等级标签：后端还没返回信用分时不展示（避免出现「信用良好」但没有依据的假标签）
const creditLabel = computed(() => {
  const p = photographer.value
  if (!p) return ''
  if (typeof p.creditScore !== 'number' && !p.creditLevel) return ''
  return creditLevelShort(p.creditLevel, p.creditScore)
})
const creditTone = computed(() => creditToneOf(photographer.value?.creditLevel, photographer.value?.creditScore))

const collected = ref(false)
const reviews = ref<Review[]>([])
const reviewTotal = ref(0)
const reviewHasMore = ref(false)
const reviewLoading = ref(false)
const REVIEW_PAGE_SIZE = 10
const reviewCount = computed(() => reviewTotal.value || reviews.value.length)

/** 评价走独立分页接口：详情接口里的 reviews 只作首屏兜底（老版本/接口失败）。 */
async function loadReviews(reset = false) {
  if (reviewLoading.value) return
  const pid = Number(id)
  if (!pid) return
  reviewLoading.value = true
  try {
    const offset = reset ? 0 : reviews.value.length
    const res = await getPhotographerReviews(pid, REVIEW_PAGE_SIZE, offset)
    // 分页接口返回的是后端裸 DTO（userAvatar / images 都是 /static/** 相对路径），
    // 必须过 mapReviewItem 才能进 ReviewCard，否则小程序里头像与评价附图全 404。
    const page = (res.list || []).map(mapReviewItem)
    reviews.value = reset ? page : reviews.value.concat(page)
    reviewTotal.value = res.total
    reviewHasMore.value = res.hasMore
  } catch {
    if (reset) {
      reviews.value = photographer.value?.reviews || []
      reviewTotal.value = reviews.value.length
      reviewHasMore.value = false
    }
  } finally {
    reviewLoading.value = false
  }
}

const avatarFailed = ref(false)
function onAvatarError() {
  avatarFailed.value = true
}
let id = ''

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  id = currentPage?.options?.id
  if (!id) {
    firstLoading.value = false
    uni.showToast({ title: '缺少摄影师ID', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 1500)
    return
  }
  loadData(id)
})

async function loadData(pageId: string) {
  try {
    const res = await apiGet<PhotographerDetailResponse>(`/v1/photographers/${pageId}`)
    photographer.value = {
      ...mapPhotographerItem(res),
      mode: res.mode,
      certified: res.certified,
      creditScore: res.creditScore,
      creditLevel: res.creditLevel,
      services: (res.services || []).map(s => ({
        id: String(s.id),
        name: s.name,
        price: s.price,
        description: s.description,
        duration: s.duration,
        // 计价方式与起拍张数必须带下去（B1）：预约页靠 priceUnit 判断是否显示张数步进器，
        // 靠 photoCount 展示「N张起」。这里丢掉 → 按张套餐按整套价成交。
        priceUnit: (s.priceUnit || 'fixed') as PriceUnit,
        photoCount: typeof s.photoCount === 'number' ? s.photoCount : null,
      })),
      works: (res.works || []).map(mapWorkItem),
      reviews: (res.reviews || []).map(mapReviewItem),
    }
    initCollected()
    loadReviews(true)
  } catch (e) {
    uni.showToast({ title: '加载失败，请稍后再试', icon: 'none' })
  } finally {
    firstLoading.value = false
  }
}

async function initCollected() {
  if (!userStore.isLoggedIn || !userStore.user) return
  try {
    const res = await apiGet<any[]>(`/v1/favorites/${userStore.user.id}`)
    collected.value = (res || []).some((f: any) => String(f.photographerId) === id)
  } catch { /* ignore */ }
}

async function collect() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => navTo('/pages/login/index?redirect=' + encodeRedirect('/pages/photographer/detail?id=' + id)), 800)
    return
  }
  const pid = Number(id)
  try {
    if (collected.value) {
      await apiDelete(`/v1/favorites/${userStore.user.id}/${pid}`)
      collected.value = false
      uni.showToast({ title: '取消收藏', icon: 'none' })
    } else {
      await apiPost('/v1/favorites', { userId: Number(userStore.user.id), photographerId: pid })
      collected.value = true
      uni.showToast({ title: '已收藏', icon: 'success' })
    }
  } catch {
    uni.showToast({ title: '操作失败', icon: 'none' })
  }
}

async function blockPhotographer() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => navTo('/pages/login/index?redirect=' + encodeRedirect('/pages/photographer/detail?id=' + id)), 800)
    return
  }
  const targetUserId = photographer.value?.userId
  if (!targetUserId) {
    uni.showToast({ title: '无法拉黑该摄影师', icon: 'none' })
    return
  }
  uni.showModal({
    title: '加入黑名单',
    content: `确定将「${photographer.value?.name || '该摄影师'}」加入黑名单吗？对方将不再出现在你的搜索与推荐中，且无法互相发起聊天或预约。`,
    success: async (res) => {
      if (!res.confirm) return
      try {
        await blockUser(Number(targetUserId))
        uni.showToast({ title: '已加入黑名单', icon: 'success' })
        setTimeout(() => uni.navigateBack(), 800)
      } catch {
        uni.showToast({ title: '操作失败', icon: 'none' })
      }
    }
  })
}

async function goChat() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => navTo('/pages/login/index?redirect=' + encodeRedirect('/pages/photographer/detail?id=' + id)), 800)
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(photographer.value?.userId || id)
    })
    navTo(`/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(photographer.value?.name || '')}&peerAvatar=${encodeURIComponent(photographer.value?.avatar || '')}`)
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}

function selectService(service: Service) {
  const pid = photographer.value?.id || ''
  uni.setStorageSync('selectedService', JSON.stringify(service))
  uni.setStorageSync('bookingPhotographerId', pid || '1')
  navTo(`/pages/booking/index?photographerId=${pid}`)
}

function goBooking() {
  const pid = photographer.value?.id || ''
  uni.setStorageSync('bookingPhotographerId', pid || '1')
  navTo(`/pages/booking/index?photographerId=${pid}`)
}

function goPortfolio() {
  const pid = photographer.value?.id ?? id
  navTo(`/pages/portfolio/index?photographerId=${pid}`)
}

// 作品大图预览：work.images 可能是多图，点第几张就从第几张开始看
function previewWorkImages(work: Work, index = 0) {
  previewImages(work.images, index)
}

// duration 单位分钟；描述里已写「N小时拍摄」时不再重复一行时长
function showServiceDuration(service: { duration: number; description: string }) {
  if (!service.duration) return false
  const hours = service.duration % 60 === 0 ? String(service.duration / 60) : (service.duration / 60).toFixed(1)
  return !new RegExp(`${hours}\\s*小时`).test(service.description || '')
}

function onMore() {
  uni.showActionSheet({
    itemList: ['拉黑该摄影师'],
    success: (res) => {
      if (res.tapIndex === 0) blockPhotographer()
    }
  })
}

</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.footer-icon-img {
  width: 34rpx;
  height: 34rpx;
  margin-bottom: 4rpx;
}

.content {
  height: 100vh;
  padding-bottom: 240rpx;
}

.header {
  display: flex;
  background: $dark-bg-card;
  padding: $spacing-lg $spacing-md;
  border-bottom: 1rpx solid $dark-border;
}

.avatar {
  width: 180rpx;
  height: 180rpx;
  border-radius: $border-radius-lg;
  background: $surface-2;
}

.avatar--fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 64rpx;
  font-weight: 700;
  color: $brand;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}
.name {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $dark-text-primary;
}
.cert-pill {
  padding: 2rpx 12rpx;
  border-radius: 8rpx;
  font-size: 20rpx;
  color: #FFB02E;
  background: rgba(255, 176, 46, 0.12);
  border: 1rpx solid rgba(255, 176, 46, 0.35);
}
.name-meta {
  margin-top: 6rpx;
  font-size: 24rpx;
  color: $text-3;
}
.more-btn {
  margin-left: auto;
  padding: 0 16rpx;
  min-height: 88rpx;
  line-height: 88rpx;
  font-size: 32rpx;
  color: $dark-text-tertiary;
  letter-spacing: 2rpx;
}

.stats {
  // 三列等高：首列多一条信用 pill，stretch 让三列顶部（数字重心）严格对齐，
  // 底部留白由最高的首列决定，不会把数字挤到不同基线。
  display: flex;
  align-items: stretch;
  margin-top: $spacing-sm;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.stat-value-row {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 6rpx;
  // 主数字行只放评分/数字，绝不与 pill 抢同一行；禁止换行兜底窄屏。
  white-space: nowrap;
}
.stat-star {
  font-size: 24rpx;
  color: #FFB02E;
}
.stat-value--warn {
  color: #FFB02E;
}
.stat-value {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.credit-pill {
  // 独立成行：v-if 控制，creditLabel 为空时不渲染 → 无空 pill、无多余间距。
  font-size: 20rpx;
  line-height: 1.4;
  padding: 2rpx 10rpx;
  margin-top: 4rpx;
  border-radius: 999rpx;
  border: 1rpx solid $dark-border;
  background: rgba(255, 255, 255, 0.04);
  align-self: center;
  white-space: nowrap;
  flex-shrink: 0;

  &.tone-good {
    color: $success-color;
    border-color: rgba(34, 197, 94, 0.4);
  }

  &.tone-fair {
    color: $warning-color;
    border-color: rgba(250, 204, 21, 0.4);
  }

  &.tone-poor {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.4);
  }
}

.stat-label {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.stat-divider {
  width: 2rpx;
  height: 48rpx;
  background: $dark-border;
}

.location {
  display: flex;
  align-items: center;
  margin-top: $spacing-sm;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.location-icon {
  font-size: $font-size-sm;
  margin-right: 4rpx;
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-card;
  border-bottom: 1rpx solid $dark-border;
}

.mode-tag {
  padding: 6rpx 16rpx;
  border: 1rpx solid rgba(255, 107, 0, 0.5);
  background: rgba(255, 107, 0, 0.10);
  color: $brand;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;

  &.pay,
  &.both {
    border-color: rgba(255, 107, 0, 0.5);
    background: rgba(255, 107, 0, 0.10);
    color: $brand;
  }
}

.cert-badge {
  padding: 6rpx 16rpx;
  border: 1rpx solid rgba(251, 191, 36, 0.45);
  background: rgba(251, 191, 36, 0.12);
  color: #FBBF24;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
}

.tag {
  padding: 6rpx 16rpx;
  background: rgba(255, 255, 255, 0.06);
  border: 1rpx solid rgba(255, 255, 255, 0.08);
  color: $dark-text-secondary;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
}

.section {
  background: $dark-bg-card;
  margin-top: $spacing-md;
  padding: $spacing-md;
  border: 1rpx solid $dark-border;
  box-shadow: 0 2rpx 16rpx rgba(0, 0, 0, 0.15);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
}

.section-title {
  position: relative;
  padding-left: 20rpx;
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 6rpx;
    height: 28rpx;
    background: $neon-gradient;
    border-radius: 3rpx;
  }
}

.review-toggle {
  margin-top: 12rpx;
  padding: 24rpx 0;
  text-align: center;
  font-size: $font-size-sm;
  color: $brand;

  &:active { opacity: .7; }
}

.review-empty,
.review-end {
  padding: 28rpx 0;
  text-align: center;
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.section-more {
  /* 热区 ≥88rpx：纯文字链接在小屏上极易点空 */
  display: flex;
  align-items: center;
  min-height: 88rpx;
  padding: 0 8rpx;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.description {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  line-height: 1.6;
}

.service-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.service-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.service-info {
  flex: 1;
}

.service-name {
  display: block;
  font-size: $font-size-base;
  font-weight: 500;
  color: $dark-text-primary;
}

.service-desc {
  display: block;
  font-size: $font-size-xs;
  color: $dark-text-secondary;
  margin-top: 4rpx;
}

.service-duration {
  display: block;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.service-price {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.price {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $neon-purple;
}

.btn-ghost {
  padding: 8rpx 24rpx;
  border: 1rpx solid rgba(255, 107, 0, 0.45);
  color: $brand;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
  margin-top: $spacing-xs;
}

.work-scroll {
  white-space: nowrap;
}

.work-list {
  display: inline-flex;
  gap: $spacing-sm;
}

.work-item {
  width: 200rpx;
  position: relative;

  &::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 200rpx;
    height: 280rpx;
    background: linear-gradient(to bottom, rgba(0, 0, 0, 0.1), rgba(0, 0, 0, 0.4));
    border-radius: $border-radius-md;
    pointer-events: none;
  }

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.work-image {
  width: 200rpx;
  height: 280rpx;
  border-radius: $border-radius-md;
  background: $surface-2;
}
.work-image--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22rpx;
  color: $text-3;
}

.work-title {
  font-size: $font-size-xs;
  color: $dark-text-secondary;
  margin-top: $spacing-xs;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.review-list {
  margin-top: $spacing-sm;
}

.bottom-space {
  height: $spacing-xl;
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding: $spacing-sm $spacing-md;
  padding-bottom: calc(#{$spacing-sm} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.4);
}

.footer-left {
  display: flex;
  gap: $spacing-lg;
}

.footer-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 96rpx;
  padding: 8rpx 12rpx;
  font-size: $font-size-xs;
  color: $dark-text-secondary;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.footer-icon {
  font-size: $font-size-lg;
}

.footer-right {
  flex: 1;
  margin-left: $spacing-md;
}

.btn-primary {
  background: $neon-purple;
  color: #fff;
  border-radius: $border-radius-lg;
  padding: $spacing-sm;
  text-align: center;
  font-size: $font-size-base;
  font-weight: 500;
}
/* ── 首屏深色骨架：切页不再露白（B 档）── */
.skeleton-page { min-height: 100vh; padding: $spacing-md; box-sizing: border-box; display: flex; flex-direction: column; gap: 24rpx; }
.sk-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; box-sizing: border-box; }
.sk-block { background: $surface-2; border-radius: 12rpx; }
.sk-line { height: 28rpx; background: $surface-2; border-radius: 8rpx; }
.sk-circle { background: $surface-2; border-radius: 50%; flex-shrink: 0; }
.sk-row { display: flex; align-items: center; gap: 24rpx; }
.sk-pad { padding: 28rpx; display: flex; flex-direction: column; gap: 24rpx; }
.sk-avatar-120 { width: 120rpx; height: 120rpx; }
.sk-avatar-88 { width: 88rpx; height: 88rpx; }
.sk-hero { height: 160rpx; }
.sk-grid { display: flex; flex-wrap: wrap; gap: 16rpx; }
.sk-grid-item { width: calc(50% - 8rpx); height: 320rpx; border-radius: 16rpx; }
.sk-w-25 { width: 25%; } .sk-w-30 { width: 30%; } .sk-w-40 { width: 40%; }
.sk-w-60 { width: 60%; } .sk-w-80 { width: 80%; } .sk-w-full { width: 100%; }

/* 摄影师详情骨架：对齐真实六区块高度，加载完不跳 */
.sk-detail-header { display: flex; gap: 24rpx; padding: 32rpx; }
.sk-detail-avatar { width: 180rpx; height: 180rpx; }
.sk-detail-info { flex: 1; display: flex; flex-direction: column; gap: 16rpx; padding-top: 10rpx; }
.sk-detail-stats { display: flex; justify-content: space-between; margin-top: 20rpx; }
.sk-detail-stat { flex: 1; display: flex; flex-direction: column; gap: 12rpx; align-items: center; }
.sk-detail-stat .sk-line { height: 24rpx; }
.sk-detail-tags { display: flex; gap: 16rpx; }
.sk-detail-tag { width: 120rpx; height: 48rpx; border-radius: 24rpx; }
.sk-detail-section { padding: 28rpx; display: flex; flex-direction: column; gap: 20rpx; }
.sk-detail-works { display: flex; gap: 16rpx; }
.sk-detail-work { width: 200rpx; height: 280rpx; border-radius: 12rpx; flex-shrink: 0; }
</style>
