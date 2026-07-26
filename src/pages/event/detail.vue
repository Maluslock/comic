<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <scroll-view scroll-y class="scroll-content" :style="{ paddingTop: statusBarHeight + 'px' }">
      <template v-if="loading">
        <view class="sk-block" style="height: 400rpx; margin: 0 0 24rpx" />
        <view class="sk-block" style="height: 32rpx; width: 60%; margin: 0 24rpx 16rpx" />
        <view class="sk-block" style="height: 24rpx; width: 80%; margin: 0 24rpx 24rpx" />
      </template>

      <template v-else-if="event">
        <!-- Cover -->
        <view class="event-cover-wrap">
          <image :src="event.cover" class="event-cover" mode="aspectFill" />
          <view class="cover-overlay" />
          <view class="back-btn" @click="goBack">
            <text class="back-arrow">‹</text>
          </view>
          <view class="cover-info">
            <text class="cover-name">{{ event.name }}</text>
            <view class="cover-status" v-if="statusText">
              <text class="status-dot" :class="event.status" />
              <text>{{ statusText }}</text>
            </view>
          </view>
        </view>

        <view class="detail-body">
          <!-- Date & Location -->
          <view class="info-card">
            <view class="info-row">
              <text class="info-label">时间</text>
              <text class="info-value">{{ formatFullDate(event.startDate) }} - {{ formatFullDate(event.endDate) }}</text>
            </view>
            <view class="info-row">
              <text class="info-label">地点</text>
              <text class="info-value">{{ event.location }} · {{ event.venue }}</text>
            </view>
            <view v-if="event.description" class="info-row">
              <text class="info-label">简介</text>
              <text class="info-value desc-text">{{ event.description }}</text>
            </view>
          </view>

          <!-- Tags -->
          <view v-if="event.tags.length" class="tags-section">
            <view v-for="tag in event.tags" :key="tag" class="tag-item">{{ tag }}</view>
          </view>

          <!-- Countdown -->
          <view class="countdown-card">
            <template v-if="event.status === 'upcoming'">
              <text class="countdown-num">{{ countdownDays }}</text>
              <text class="countdown-unit">天后开展</text>
            </template>
            <template v-else-if="event.status === 'ongoing'">
              <text class="countdown-live">● 正在举办</text>
            </template>
            <template v-else>
              <text class="countdown-ended">已结束</text>
            </template>
          </view>

          <!-- Photographers -->
          <view class="section">
            <view class="section-header">
              <text class="section-title">参展摄影师</text>
              <text class="section-count">{{ photographers.length }}人</text>
            </view>
            <view class="photographer-grid" v-if="photographers.length">
              <PhotographerCard
                v-for="p in photographers"
                :key="p.id"
                :photographer="p"
              />
            </view>
            <view v-else class="empty-hint">暂无摄影师报名参展</view>
          </view>

          <!-- Works -->
          <view v-if="featuredWorks.length" class="section">
            <view class="section-header">
              <text class="section-title">精选作品</text>
            </view>
            <view class="works-grid">
              <view
                v-for="work in featuredWorks"
                :key="work.id"
                class="work-item"
              >
                <image :src="work.images[0]" class="work-img" mode="aspectFill" />
                <view class="work-overlay">
                  <text class="work-title">{{ work.title }}</text>
                  <text class="work-author">by {{ work.photographerName }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </template>

      <template v-else>
        <view class="error-state">
          <text class="error-icon">!</text>
          <text class="error-text">{{ error || '漫展不存在' }}</text>
          <view class="btn-retry" @click="goBack"><text>返回</text></view>
        </view>
      </template>

      <view class="bottom-space" />
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotographerCard from '@/components/PhotographerCard.vue'
import type { ComicEvent, Photographer, Work } from '@/types'

interface EventDetailResponse {
  id: number
  name: string
  location: string
  venue: string
  startDate: string
  endDate: string
  coverUrl: string
  tags: string[]
  status: string
  typeName: string
  description: string
  photographers: Array<{
    id: number
    name: string
    avatar: string
    location: string
    rating: number
    reviewCount: number
    orderCount: number
    tags: string[]
  }>
  featuredWorks: Array<{
    id: number
    title: string
    images: string[]
    photographerName: string
  }>
}

const statusBarHeight = ref(44)
const loading = ref(true)
const error = ref('')

const event = ref<ComicEvent | null>(null)
const photographers = ref<Photographer[]>([])
const featuredWorks = ref<Work[]>([])

const statusText = ref('')
const countdownDays = ref(0)

onMounted(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  const eventId = currentPage?.options?.id as string

  if (!eventId) { error.value = '缺少漫展ID'; loading.value = false; return }

  await loadEvent(eventId)
})

async function loadEvent(id: string) {
  try {
    const res = await uni.request({
      url: `http://localhost:8081/api/v1/events/${id}`,
      method: 'GET',
      timeout: 5000,
    })

    if (res.statusCode !== 200) {
      throw new Error(`HTTP ${res.statusCode}`)
    }

    const d = res.data as EventDetailResponse

    event.value = {
      id: String(d.id),
      name: d.name,
      location: d.location,
      venue: d.venue,
      startDate: new Date(d.startDate).getTime(),
      endDate: new Date(d.endDate).getTime(),
      cover: d.coverUrl,
      tags: d.tags,
      photographerCount: d.photographers?.length || 0,
      status: (d.status as ComicEvent['status']) || 'upcoming',
    }

    const desc = (d as any).description
    if (desc) (event.value as any).description = desc

    photographers.value = (d.photographers || []).map(p => ({
      id: String(p.id),
      name: p.name,
      avatar: p.avatar,
      role: 'photographer' as const,
      description: '',
      rating: p.rating,
      reviewCount: p.reviewCount,
      orderCount: p.orderCount,
      location: p.location,
      tags: p.tags,
      works: [],
      services: [],
      reviews: [],
      createdAt: Date.now(),
    }))

    featuredWorks.value = (d.featuredWorks || []).map(w => ({
      id: String(w.id),
      photographerId: '',
      title: w.title,
      images: w.images,
      description: '',
      tags: [],
      createdAt: Date.now(),
    }))

    if (event.value.status === 'ongoing') statusText.value = '进行中'
    else if (event.value.status === 'ended') statusText.value = '已结束'
    else if (event.value.status === 'upcoming') {
      countdownDays.value = Math.max(0, Math.ceil((event.value.startDate - Date.now()) / (1000 * 60 * 60 * 24)))
      statusText.value = ''
    }
  } catch (e: unknown) {
    console.error('Load event detail failed:', e)
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function formatFullDate(ts: number): string {
  const d = new Date(ts)
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.status-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: transparent;
}

.scroll-content {
  height: 100vh;
}

// Cover
.event-cover-wrap {
  position: relative;
  width: 100%;
  height: 480rpx;
}

.event-cover {
  width: 100%;
  height: 100%;
}

.cover-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(transparent 40%, rgba(10, 10, 26, 0.9));
}

.back-btn {
  position: absolute;
  top: 0;
  left: $spacing-md;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.back-arrow {
  font-size: 56rpx;
  color: #fff;
  font-weight: 300;
  text-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.5);
}

.cover-info {
  position: absolute;
  bottom: $spacing-lg;
  left: $spacing-md;
  right: $spacing-md;
}

.cover-name {
  display: block;
  font-size: 44rpx;
  font-weight: 900;
  color: #fff;
  text-shadow: 0 2rpx 12rpx rgba(0, 0, 0, 0.4);
}

.cover-status {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: $spacing-xs;
  font-size: 26rpx;
  color: rgba(255, 255, 255, 0.8);
}

.status-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;

  &.upcoming { background: $neon-cyan; box-shadow: 0 0 8rpx $neon-cyan-glow; }
  &.ongoing { background: $success-color; box-shadow: 0 0 8rpx rgba($success-color, 0.5); }
  &.ended { background: $dark-text-tertiary; }
}

// Body
.detail-body {
  padding: 0 $spacing-md;
  margin-top: -32rpx;
  position: relative;
  z-index: 5;
}

// Info Card
.info-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
}

.info-row {
  display: flex;
  gap: $spacing-sm;
  padding: $spacing-xs 0;

  & + & {
    border-top: 1rpx solid $dark-border;
  }
}

.info-label {
  font-size: 26rpx;
  color: $neon-cyan;
  font-weight: 600;
  flex-shrink: 0;
  width: 64rpx;
}

.info-value {
  font-size: 26rpx;
  color: $dark-text-primary;
  line-height: 1.5;
}

.desc-text {
  color: $dark-text-secondary;
  font-size: 24rpx;
}

// Tags
.tags-section {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-md;
}

.tag-item {
  padding: 6rpx 20rpx;
  font-size: 22rpx;
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 1rpx solid rgba($neon-purple, 0.3);
  border-radius: $border-radius-xl;
}

// Countdown
.countdown-card {
  display: flex;
  align-items: baseline;
  justify-content: center;
  padding: $spacing-lg 0;
  margin-top: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $neon-purple-glow;
  border-radius: $border-radius-lg;
}

.countdown-num {
  font-size: 72rpx;
  font-weight: 900;
  color: $neon-purple;
  line-height: 1;
}

.countdown-unit {
  font-size: 32rpx;
  color: $dark-text-primary;
  margin-left: $spacing-xs;
  font-weight: 600;
}

.countdown-live {
  font-size: 36rpx;
  color: $success-color;
  font-weight: 700;
}

.countdown-ended {
  font-size: 32rpx;
  color: $dark-text-tertiary;
  font-weight: 600;
}

// Sections
.section {
  margin-top: $spacing-xl;
}

.section-header {
  display: flex;
  align-items: flex-end;
  gap: $spacing-sm;
  margin-bottom: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
  padding-left: 20rpx;
  position: relative;

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

.section-count {
  font-size: 24rpx;
  color: $dark-text-tertiary;
  padding-bottom: 2rpx;
}

.photographer-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
}

.empty-hint {
  text-align: center;
  padding: $spacing-xl;
  color: $dark-text-tertiary;
  font-size: $font-size-sm;
}

// Works
.works-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-sm;
}

.work-item {
  position: relative;
  border-radius: $border-radius-md;
  overflow: hidden;
  aspect-ratio: 4/3;

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border: 1rpx solid rgba($neon-purple, 0.12);
    border-radius: $border-radius-md;
    pointer-events: none;
    z-index: 2;
  }
}

.work-img {
  width: 100%;
  height: 100%;
}

.work-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: $spacing-sm;
  background: linear-gradient(transparent, rgba(10, 10, 26, 0.85));
}

.work-title {
  display: block;
  font-size: 24rpx;
  color: #fff;
  font-weight: 600;
}

.work-author {
  display: block;
  font-size: 20rpx;
  color: rgba(255, 255, 255, 0.6);
  margin-top: 2rpx;
}

// Error
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2 $spacing-md;
}

.error-icon {
  width: 80rpx;
  height: 80rpx;
  line-height: 80rpx;
  text-align: center;
  font-size: 48rpx;
  font-weight: 900;
  color: $dark-text-tertiary;
  border: 2rpx solid $dark-border;
  border-radius: 50%;
  margin-bottom: $spacing-md;
}

.error-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
  margin-bottom: $spacing-lg;
}

.btn-retry {
  padding: 14rpx 48rpx;
  background: $neon-gradient;
  color: #fff;
  border-radius: $border-radius-xl;
  font-size: $font-size-base;
  font-weight: 600;

  &:active { opacity: 0.8; }
}

// Skeleton
.sk-block {
  background: rgba(255, 255, 255, 0.04);
  border-radius: 16rpx;
}

.bottom-space {
  height: 48rpx;
}
</style>
