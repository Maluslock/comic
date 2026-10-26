<template>
  <view class="page">
    <scroll-view scroll-y class="content">
      <view class="header">
        <image :src="photographer?.avatar" class="avatar" mode="aspectFill" @error="onAvatarError" />
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
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <text class="stat-value">{{ photographer?.reviewCount }}</text>
              <text class="stat-label">评价</text>
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <text class="stat-value">{{ photographer?.orderCount }}</text>
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
              <text class="service-desc">{{ service.description }}</text>
              <text class="service-duration">{{ formatDuration(service.duration) }}</text>
            </view>
            <view class="service-price">
              <text class="price">{{ formatPrice(service.price) }}</text>
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
              <image v-if="work.images?.length" :src="work.images[0]" class="work-image" mode="aspectFill" />
              <view v-else class="work-image work-image--empty"><text>暂无图片</text></view>
              <text class="work-title">{{ work.title }}</text>
            </view>
          </view>
        </scroll-view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">用户评价</text>
          <text class="section-more" @click="goReviews">查看全部</text>
        </view>
        <view class="review-list">
          <ReviewCard 
            v-for="review in photographer?.reviews" 
            :key="review.id" 
            :review="review"
          />
        </view>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>

    <view class="footer">
      <view class="footer-left">
        <view class="footer-btn" @click="goChat">
          <image class="footer-icon-img" src="/static/icons/chat.svg" mode="aspectFit" />
          <text>聊天</text>
        </view>
        <view class="footer-btn" @click="collect">
          <image class="footer-icon-img" :src="collected ? '/static/icons/star-filled.svg' : '/static/icons/star.svg'" mode="aspectFit" />
          <text>收藏</text>
        </view>
      </view>
      <view class="footer-right">
        <view class="btn-primary" @click="goBooking">立即预约</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ReviewCard from '@/components/ReviewCard.vue'
import { apiGet, apiPost, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'
import { blockUser } from '@/api/index'
import { mapPhotographerItem, mapWorkItem, mapReviewItem, formatPrice } from '@/utils/mappers'
import { formatDuration } from '@/utils/duration'
import type { Photographer, Service } from '@/types'

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
  tags: string[]
  description: string
  services: Array<{ id: number; name: string; price: number | null; description: string; duration: number }>
  works: Array<{ id: number; title: string; images: string[]; photographerName?: string }>
  reviews: Array<{ id: number; rating: number; content: string; userName: string; userAvatar?: string; createdAt: string }>
}

type PhotographerView = Photographer & { mode?: string; certified?: boolean }

const userStore = useUserStore()
const photographer = ref<PhotographerView | null>(null)

// 评分按真实评价重算：没有评价时后端返回 rating=0 / reviewCount=0。
// 直接显示「0」会被读成「被打 0 分」，故无评价时给文案。
const ratingText = computed(() => ((photographer.value?.reviewCount ?? 0) > 0 ? String(photographer.value?.rating) : '暂无'))

const collected = ref(false)
const DEFAULT_AVATAR = '/static/img/avatar-user.svg'

function onAvatarError() {
  if (photographer.value) photographer.value.avatar = DEFAULT_AVATAR
}
let id = ''

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  id = currentPage?.options?.id
  if (!id) {
    uni.showToast({ title: '缺少摄影师ID', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 1500)
    return
  }
  loadData(id)
})

async function loadData(pageId: string) {
  uni.showLoading({ title: '加载中...' })
  try {
    const res = await apiGet<PhotographerDetailResponse>(`/v1/photographers/${pageId}`)
    photographer.value = {
      ...mapPhotographerItem(res),
      mode: res.mode,
      certified: res.certified,
      services: (res.services || []).map(s => ({
        id: String(s.id),
        name: s.name,
        price: s.price,
        description: s.description,
        duration: s.duration,
      })),
      works: (res.works || []).map(mapWorkItem),
      reviews: (res.reviews || []).map(mapReviewItem),
    }
    initCollected()
  } finally {
    uni.hideLoading()
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
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/detail?id=' + id }), 800)
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
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/detail?id=' + id }), 800)
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
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/detail?id=' + id }), 800)
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(photographer.value?.userId || id)
    })
    uni.navigateTo({ url: `/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(photographer.value?.name || '')}&peerAvatar=${encodeURIComponent(photographer.value?.avatar || '')}` })
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}

function selectService(service: Service) {
  const pid = photographer.value?.id || ''
  uni.setStorageSync('selectedService', JSON.stringify(service))
  uni.setStorageSync('bookingPhotographerId', pid || '1')
  uni.navigateTo({ url: `/pages/booking/index?photographerId=${pid}` })
}

function goBooking() {
  const pid = photographer.value?.id || ''
  uni.setStorageSync('bookingPhotographerId', pid || '1')
  uni.navigateTo({ url: `/pages/booking/index?photographerId=${pid}` })
}

function goPortfolio() {
  const pid = photographer.value?.id ?? id
  uni.navigateTo({ url: `/pages/portfolio/index?photographerId=${pid}` })
}

function onMore() {
  uni.showActionSheet({
    itemList: ['拉黑该摄影师'],
    success: (res) => {
      if (res.tapIndex === 0) blockPhotographer()
    }
  })
}

function goReviews() {
  uni.navigateTo({ url: `/pages/comment/index?photographerId=${id}` })
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
  min-height: 56rpx;
  line-height: 56rpx;
  font-size: 32rpx;
  color: $dark-text-tertiary;
  letter-spacing: 2rpx;
}

.stats {
  display: flex;
  align-items: center;
  margin-top: $spacing-sm;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.stat-value-row {
  display: flex;
  align-items: baseline;
  gap: 6rpx;
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

.section-more {
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
</style>
