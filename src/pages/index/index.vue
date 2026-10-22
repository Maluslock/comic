<template>
  <view class="page">
    <!-- Status Bar Spacer -->
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <!-- Pull-to-Refresh Wrapper -->
    <scroll-view
      scroll-y
      class="scroll-content"
      :style="{ paddingTop: statusBarHeight + 'px' }"
      :refresher-enabled="true"
      :refresher-triggered="store.refreshing"
      :refresher-threshold="80"
      @refresherrefresh="onRefresh"
      @scrolltolower="onLoadMore"
    >
      <!-- Skeleton Loading -->
      <HomeSkeleton v-if="store.loading && !store.loaded" />

      <template v-else>
        <!-- Hero Stage -->
        <view class="hero-stage">
          <image class="hero-media" :src="heroImage" mode="aspectFill" />
          <view class="hero-scrim" />
          <view class="hero-glow" />
          <view class="hero-grain" />
          <view class="vf-corner vf-tl" />
          <view class="vf-corner vf-tr" />
          <view class="vf-corner vf-bl" />
          <view class="vf-corner vf-br" />

          <view class="hero-content">
            <view class="hero-top-bar">
              <view class="city-selector" @click="goCitySelect">
                <text class="city-dot" />
                <text class="city-text">{{ currentCity }}</text>
                <text class="city-arrow">▾</text>
              </view>
              <view class="hero-actions">
                <view class="hero-icon-btn" @click="goSearch">
                  <image class="hero-icon-svg" src="/static/icons/search.svg" mode="aspectFit" />
                </view>
                <view class="hero-icon-btn" @click="goMessages">
                  <image class="hero-icon-svg" src="/static/icons/bell.svg" mode="aspectFit" />
                  <view v-if="unreadCount > 0" class="badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</view>
                </view>
              </view>
            </view>

            <view class="hero-copy">
              <text class="hero-eyebrow">米拉漫展 · NEXT UP</text>
              <text class="hero-title">{{ heroEvent ? heroEvent.name : '米拉漫展' }}</text>
              <view v-if="heroEvent" class="hero-meta">
                <text class="hero-meta-strong">{{ store.formatDate(heroEvent.startDate) }}</text>
                <text class="hero-meta-dot">·</text>
                <text class="hero-meta-venue">{{ heroEvent.venue || heroEvent.location }}</text>
              </view>
              <view v-else class="hero-meta">
                <text class="hero-meta-venue">找到你的专属摄影师</text>
              </view>
            </view>
          </view>
        </view>

        <!-- Ticket dock -->
        <view v-if="nearestEvent" class="ticket" @click="goEventDetail(nearestEvent.id)">
          <view class="ticket-left">
            <text class="ticket-label">距开展</text>
            <text class="ticket-days">{{ store.countdownDays(nearestEvent.startDate) }}</text>
            <text class="ticket-unit">天</text>
          </view>
          <view class="ticket-rip" />
          <view class="ticket-right">
            <text class="ticket-cta">查看漫展</text>
            <text class="ticket-arrow">›</text>
          </view>
        </view>

        <!-- Quick Action Bar -->
        <view class="section-compact">
          <QuickActionBar @navigate="onQuickNav" />
        </view>

        <!-- Upcoming Events -->
        <view class="section">
          <view class="section-header">
            <text class="section-title">近期漫展</text>
            <text class="section-more" @click="goEventMore">更多 ›</text>
          </view>
          <view class="range-chips">
            <view
              v-for="opt in rangeOptions"
              :key="opt.key"
              class="range-chip"
              :class="{ active: eventRange === opt.key }"
              @click="eventRange = opt.key"
            >{{ opt.label }}</view>
          </view>
          <scroll-view scroll-x class="event-scroll" :show-scrollbar="false">
            <view class="event-list">
              <view
                v-for="event in filteredEvents"
                :key="event.id"
                class="event-card"
                @click="goEventDetail(event.id)"
              >
                <view
                  class="event-cover"
                  :style="{ backgroundImage: 'url(' + event.cover + ')' }"
                />
                <view class="event-countdown-badge">
                  <text>{{ store.countdownDays(event.startDate) }}天后</text>
                </view>
                <view class="event-info">
                  <text class="event-name ellipsis">{{ event.name }}</text>
                  <view class="event-meta">
                    <text class="event-location">{{ event.location }} · {{ event.venue }}</text>
                  </view>
                  <view class="event-footer">
                    <text class="event-date">{{ store.formatDate(event.startDate) }} - {{ store.formatDate(event.endDate) }}</text>
                  </view>
                </view>
              </view>
            </view>
          </scroll-view>
        </view>

        <!-- Hot Tags -->
        <view class="section-compact">
          <view class="section-header">
            <text class="section-title">热门风格</text>
            <text class="section-more" @click="goSearch">更多 ›</text>
          </view>
          <view class="tags-row">
            <view
              v-for="tag in store.hotTags"
              :key="tag"
              class="tag-item"
              @click="goSearchByTag(tag)"
            >
              {{ tag }}
            </view>
          </view>
        </view>

        <!-- Poster Wall -->
        <view v-if="posterWall.length" class="section">
          <view class="section-header">
            <text class="section-title">漫展海报</text>
            <text class="section-more" @click="goEventMore">更多 ›</text>
          </view>
          <view class="works-grid">
            <view
              v-for="ev in posterWall"
              :key="ev.id"
              class="work-card"
              @click="goEventDetail(ev)"
            >
              <image class="work-img" :src="ev.cover" mode="aspectFill" />
              <view class="work-scrim" />
              <view class="work-meta">
                <text class="work-title">{{ ev.name }}</text>
                <text class="work-author">{{ formatShortDate(ev.startDate) }} · {{ ev.location }}</text>
              </view>
            </view>
          </view>
        </view>

        <!-- Recommended Photographers -->
        <view class="section">
          <view class="section-header">
            <text class="section-title">推荐摄影师</text>
            <text class="section-more" @click="goPhotographerList">更多 ›</text>
          </view>
          <view class="photographer-grid">
            <PhotographerCard
              v-for="p in filteredPhotographers.slice(0, 4)"
              :key="p.id"
              :photographer="p"
              variant="grid"
            />
          </view>
        </view>

        <!-- Error State -->
        <view v-if="store.error && !store.loaded" class="error-state">
          <text class="error-icon">!</text>
          <text class="error-text">{{ store.error }}</text>
          <view class="btn-retry" @click="store.fetchHomeData()">
            <text>重新加载</text>
          </view>
        </view>

        <view class="bottom-space" />
      </template>
    </scroll-view>

    <!-- City Selector Popup -->
    <view v-if="showCityPicker" class="city-popup-mask" @click="closeCityPicker">
      <view class="city-popup" @click.stop>
        <view class="city-popup-header">
          <text class="city-popup-title">选择城市</text>
          <text class="city-popup-close" @click="closeCityPicker">✕</text>
        </view>
        <view class="city-list">
          <view
            v-for="city in cityOptions"
            :key="city"
            class="city-option"
            :class="{ active: currentCity === city }"
            @click="selectCity(city)"
          >
            <text>{{ city }}</text>
            <text v-if="currentCity === city" class="city-check">✓</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import PhotographerCard from '@/components/PhotographerCard.vue'
import QuickActionBar from '@/components/QuickActionBar.vue'
import HomeSkeleton from '@/components/HomeSkeleton.vue'
import { useHomeStore } from '@/stores/home'

const store = useHomeStore()
const { cityOptions } = storeToRefs(store)
const statusBarHeight = ref(44)
const currentCity = ref('全部')
const eventRange = ref<'all' | 'week' | 'month'>('all')
const rangeOptions = [
  { key: 'all' as const, label: '全部' },
  { key: 'week' as const, label: '本周' },
  { key: 'month' as const, label: '本月' },
]
const unreadCount = ref(3)
const showCityPicker = ref(false)

const nearestEvent = computed(() => {
  const upcoming = filteredEvents.value
    .filter(e => e.startDate > Date.now())
    .sort((a, b) => a.startDate - b.startDate)
  return upcoming.length > 0 ? upcoming[0] : null
})

const heroEvent = computed(() => nearestEvent.value || (filteredEvents.value.length ? filteredEvents.value[0] : null))
const heroImage = computed(() => {
  if (heroEvent.value && (heroEvent.value as any).cover) return (heroEvent.value as any).cover
  if (store.banners.length) return store.banners[0].image
  return '/static/img/banner-1.jpg'
})

onMounted(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!store.loaded) {
    store.fetchHomeData()
  }
})

function onRefresh() {
  store.refresh()
}

function onLoadMore() {}

function goSearch() {
  uni.navigateTo({ url: '/pages/search/search' })
}

function goSearchByTag(tag: string) {
  uni.navigateTo({ url: `/pages/search/search?keyword=${encodeURIComponent(tag)}` })
}

function goPhotographerList() {
  uni.switchTab({ url: '/pages/photographer/list' })
}

function goEventList() {
  uni.navigateTo({ url: '/pages/calendar/index' })
}

function goEventMore() {
  uni.navigateTo({ url: '/pages/event/list' })
}

function goEventDetail(id: string) {
  uni.navigateTo({ url: `/pages/event/detail?id=${id}` })
}

function goWork(work: { photographerId: string }) {
  uni.navigateTo({ url: `/pages/photographer/works?id=${work.photographerId}` })
}

function goMessages() {
  uni.switchTab({ url: '/pages/message/index' })
}

function goCitySelect() {
  showCityPicker.value = true
  lockPageScroll(true)
}

function closeCityPicker() {
  showCityPicker.value = false
  lockPageScroll(false)
}

function selectCity(city: string) {
  currentCity.value = city
  closeCityPicker()
}

function lockPageScroll(lock: boolean) {
  // #ifdef H5
  const pageEl = document.querySelector('.scroll-content') as HTMLElement | null
  if (pageEl) {
    pageEl.style.overflow = lock ? 'hidden' : ''
  }
  // #endif
}

const filteredEvents = computed(() => {
  const base = currentCity.value === '全部'
    ? store.upcomingEvents
    : store.events.filter(e => e.status === 'upcoming' && e.location === currentCity.value).slice(0, 6)
  if (eventRange.value === 'all') return base
  const days = eventRange.value === 'week' ? 7 : 30
  const limit = Date.now() + days * 24 * 3600 * 1000
  return base.filter(e => e.startDate <= limit)
})

const posterWall = computed(() => {
  const list = store.events.filter(e => e.status === 'upcoming')
  const rest = list.length > 6 ? list.slice(6) : list
  return rest.slice(0, 4)
})

const formatShortDate = (ts: number) => {
  const d = new Date(ts)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const filteredPhotographers = computed(() => {
  if (currentCity.value === '全部') return store.photographers
  return store.photographers.filter(p => p.location === currentCity.value)
})

function onBannerClick(index: number) {
  const banner = store.banners[index]
  if (banner && banner.linkId) {
    const event = store.events.find(e => String(e.id) === String(banner.linkId))
    if (event) {
      goEventDetail(event.id)
    }
  }
}

function onQuickNav(_url: string) {}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $surface-page;
}

.status-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: $surface-glass;
  backdrop-filter: blur(20px);
}

.scroll-content {
  height: 100vh;
}

// ===== Hero Stage =====
.hero-stage {
  position: relative;
  height: 900rpx;
  overflow: hidden;
  background: $surface-page;
}

.hero-media {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.hero-scrim {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(12, 13, 20, 0.74) 0%, rgba(12, 13, 20, 0.10) 30%, rgba(12, 13, 20, 0) 44%),
    linear-gradient(0deg, #0C0D14 0%, rgba(12, 13, 20, 0.97) 30%, rgba(12, 13, 20, 0.86) 48%, rgba(12, 13, 20, 0.46) 68%, rgba(12, 13, 20, 0.12) 84%, rgba(12, 13, 20, 0) 94%);
}

.hero-glow {
  position: absolute;
  left: -10%;
  right: -10%;
  top: -6%;
  height: 72%;
  background:
    radial-gradient(52% 40% at 78% 12%, rgba(255, 107, 0, 0.30) 0%, rgba(255, 107, 0, 0) 70%),
    radial-gradient(40% 30% at 6% 86%, rgba(0, 210, 211, 0.12) 0%, rgba(0, 210, 211, 0) 72%);
  mix-blend-mode: screen;
  pointer-events: none;
}

.hero-grain {
  position: absolute;
  inset: 0;
  opacity: 0.05;
  pointer-events: none;
  background-image: radial-gradient(rgba(255, 255, 255, 0.6) 0.5rpx, transparent 0.7rpx);
  background-size: 3rpx 3rpx;
}

.vf-corner {
  position: absolute;
  width: 36rpx;
  height: 36rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.55);
  pointer-events: none;
}

.vf-tl { top: 24rpx; left: 24rpx; border-right: 0; border-bottom: 0; border-radius: 4rpx 0 0 0; opacity: .45; }
.vf-tr { top: 24rpx; right: 24rpx; border-left: 0; border-bottom: 0; border-radius: 0 4rpx 0 0; opacity: .6; }
.vf-bl { bottom: 24rpx; left: 24rpx; border-right: 0; border-top: 0; border-radius: 0 0 0 4rpx; opacity: .6; }
.vf-br { bottom: 24rpx; right: 24rpx; border-left: 0; border-top: 0; border-radius: 0 0 4rpx 0; opacity: .75; }

.hero-content {
  position: relative;
  z-index: 2;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24rpx 32rpx 0;
}

.hero-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.city-selector {
  display: flex;
  align-items: center;
  height: 72rpx;
  padding: 0 24rpx;
  background: rgba(21, 23, 34, 0.66);
  border-radius: 36rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(16px);
}

.city-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: $neon-cyan;
  margin-right: 10rpx;
  box-shadow: 0 0 8rpx $neon-cyan-glow;
}

.city-text {
  font-size: 26rpx;
  color: $dark-text-primary;
  font-weight: 500;
}

.city-arrow {
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-left: 8rpx;
}

.hero-actions {
  display: flex;
  gap: 16rpx;
}

.hero-icon-btn {
  position: relative;
  width: 80rpx;
  height: 80rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(21, 23, 34, 0.66);
  border-radius: 50%;
  border: 1rpx solid rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(16px);

  &:active {
    background: rgba(29, 32, 47, 0.9);
  }
}

.hero-icon-svg {
  width: 36rpx;
  height: 36rpx;
  opacity: 0.8;
}

.badge {
  position: absolute;
  top: -4rpx;
  right: -4rpx;
  min-width: 28rpx;
  height: 28rpx;
  line-height: 28rpx;
  text-align: center;
  font-size: 20rpx;
  color: #fff;
  background: $neon-pink;
  border-radius: 14rpx;
  padding: 0 6rpx;
}

.hero-copy {
  margin-top: auto;
  padding-bottom: 116rpx;
}

.hero-eyebrow {
  display: block;
  font-size: 22rpx;
  letter-spacing: 4rpx;
  color: $neon-cyan;
  font-weight: 600;
  margin-bottom: 16rpx;
}

.hero-title {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 56rpx;
  font-weight: 900;
  line-height: 1.16;
  letter-spacing: -1rpx;
  color: #fff;
  text-shadow: 0 2rpx 2rpx rgba(0, 0, 0, 0.35), 0 8rpx 32rpx rgba(0, 0, 0, 0.65);
  margin-bottom: 16rpx;
}

.hero-meta {
  display: flex;
  align-items: center;
  gap: 12rpx;
  font-size: 24rpx;
  color: #9AA0B4;
}

.hero-meta-strong {
  color: #fff;
  font-weight: 700;
  font-size: 28rpx;
}

.hero-meta-dot {
  opacity: 0.5;
}

.hero-meta-venue {
  max-width: 460rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

// ===== Ticket dock =====
.ticket {
  position: relative;
  z-index: 3;
  margin: -64rpx auto 0;
  width: calc(100% - 48rpx);
  height: 132rpx;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  border-radius: 24rpx;
  background: linear-gradient(105deg, #FF8C33 0%, #FF6B00 46%, #E05A00 100%);
  box-shadow: 0 18rpx 48rpx rgba(255, 107, 0, 0.28), inset 0 1rpx 0 rgba(255, 255, 255, 0.35);
  color: #0C0D14;

  &:active {
    transform: scale(0.985) translateY(2rpx);
  }

  &::before,
  &::after {
    content: '';
    position: absolute;
    top: 50%;
    width: 24rpx;
    height: 24rpx;
    border-radius: 50%;
    background: $surface-page;
    transform: translateY(-50%);
  }

  &::before { left: -12rpx; }
  &::after { right: -12rpx; }
}

.ticket-left {
  padding-left: 40rpx;
  display: flex;
  align-items: baseline;
  padding: 0 28rpx;
}

.ticket-label {
  font-size: 24rpx;
  font-weight: 600;
  opacity: 0.75;
  margin-right: 8rpx;
}

.ticket-days {
  font-size: 52rpx;
  font-weight: 900;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.ticket-unit {
  font-size: 24rpx;
  font-weight: 700;
  margin-left: 6rpx;
}

.ticket-rip {
  width: 0;
  height: 72rpx;
  border-left: 2rpx dashed rgba(12, 13, 20, 0.28);
}

.ticket-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 28rpx;
}

.ticket-cta {
  font-size: 30rpx;
  font-weight: 800;
  letter-spacing: 1rpx;
}

.ticket-arrow {
  font-size: 40rpx;
  font-weight: 700;
  line-height: 1;
}

// ===== Sections (Shared) =====
.section {
  padding: 0 $spacing-md;
  margin-top: $spacing-lg;
}

.section-compact {
  padding: 0 $spacing-md;
  margin-top: $spacing-md;
}

.section-first {
  margin-top: $spacing-md;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: $spacing-md;
}

.section-title {
  position: relative;
  font-size: $t-2xl;
  font-weight: 700;
  color: $dark-text-primary;
  padding-left: 20rpx;

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
  font-size: $t-sm;
  color: $accent;
  font-weight: 500;

  &:active { opacity: 0.6; }
}

// ===== Events =====
.range-chips {
  display: flex;
  gap: 12rpx;
  padding: 0 $spacing-md $spacing-sm;
}

.range-chip {
  padding: 8rpx 24rpx;
  border-radius: 999rpx;
  border: 1rpx solid $border-subtle;
  background: rgba(255, 255, 255, 0.04);
  font-size: 22rpx;
  color: $text-3;
  line-height: 1.2;
}

.range-chip.active {
  border-color: rgba(255, 107, 0, 0.55);
  background: rgba(255, 107, 0, 0.14);
  color: #FFB27A;
  font-weight: 600;
}

.event-scroll {
  white-space: nowrap;
  padding-bottom: $spacing-xs;
}

.event-list {
  display: inline-flex;
  gap: $spacing-md;

  &::after {
    content: '';
    width: $spacing-md;
    flex-shrink: 0;
  }
}

.event-card {
  position: relative;
  width: 300rpx;
  height: 420rpx;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: 24rpx;
  overflow: hidden;
  transition: transform 0.16s ease, border-color 0.16s ease;

  &:active {
    transform: scale(0.975);
    border-color: rgba(255, 107, 0, 0.5);
  }
}

.event-cover {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  background-color: $dark-bg-card;
}

.event-card::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(12, 13, 20, 0) 38%, rgba(12, 13, 20, 0.62) 64%, rgba(12, 13, 20, 0.97) 100%);
  pointer-events: none;
}

.event-countdown-badge {
  position: absolute;
  top: 16rpx;
  left: 16rpx;
  z-index: 2;
  padding: 6rpx 16rpx;
  background: rgba(12, 13, 20, 0.62);
  border: 1rpx solid rgba(255, 255, 255, 0.16);
  border-radius: 999rpx;
  font-size: 20rpx;
  letter-spacing: 0.04em;
  color: #FFB27A;
  backdrop-filter: blur(12rpx);
}

.event-info {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  padding: 20rpx;
}

.event-name {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: normal;
  font-size: 28rpx;
  line-height: 1.3;
  font-weight: 650;
  color: #F8FAFC;
}

.event-meta {
  margin-top: 8rpx;
}

.event-location {
  display: block;
  font-size: 22rpx;
  color: rgba(226, 232, 240, 0.72);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-footer {
  margin-top: 6rpx;
}

.event-date {
  font-size: 21rpx;
  color: #FF9A4D;
  font-weight: 500;
  letter-spacing: 0.02em;
}

// ===== Tags =====
.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.tag-item {
  padding: 10rpx 28rpx;
  font-size: $font-size-sm;
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 2rpx solid rgba($neon-purple, 0.4);
  border-radius: $border-radius-xl;
  transition: all 0.2s;

  &:active {
    background: $neon-purple-dim;
    border-color: $neon-purple;
    box-shadow: 0 0 16rpx $neon-purple-glow;
  }
}

// ===== Works =====
.works-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
}

.work-card {
  position: relative;
  height: 420rpx;
  border-radius: 24rpx;
  overflow: hidden;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
}

.work-img {
  width: 100%;
  height: 100%;
}

.work-scrim {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(12, 13, 20, 0) 24%,
    rgba(12, 13, 20, 0.55) 46%,
    rgba(12, 13, 20, 0.94) 72%,
    rgba(12, 13, 20, 1) 100%
  );
}

.work-meta {
  position: absolute;
  left: 20rpx;
  right: 20rpx;
  bottom: 18rpx;
  display: flex;
  flex-direction: column;
}

.work-title {
  font-size: 26rpx;
  font-weight: 600;
  color: #F8FAFC;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.work-author {
  margin-top: 4rpx;
  font-size: 21rpx;
  color: rgba(226, 232, 240, 0.68);
}

// ===== Photographers =====
.photographer-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
  padding-bottom: $spacing-sm;
}

// ===== Error =====
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2 $spacing-md;
}

.error-icon {
  font-size: 80rpx;
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

  &:active {
    opacity: 0.8;
  }
}

.bottom-space {
  height: 120rpx;
}

// ===== City Popup =====
.city-popup-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 200;
  display: flex;
  align-items: flex-end;
  padding-bottom: #{$tab-bar-height};
  overscroll-behavior: contain;
}

.city-popup {
  width: 100%;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  background: $dark-bg-secondary;
  border-radius: $border-radius-lg $border-radius-lg 0 0;
  padding-bottom: env(safe-area-inset-bottom);
}

.city-popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  border-bottom: 1rpx solid $dark-border;
}

.city-popup-title {
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
}

.city-popup-close {
  font-size: 36rpx;
  color: $dark-text-tertiary;
  padding: $spacing-xs;
}

.city-list {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: $spacing-sm 0;
}

.city-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  font-size: $font-size-md;
  color: $dark-text-primary;
  transition: background 0.15s;

  &:active {
    background: $dark-bg-card-hover;
  }

  &.active {
    color: $neon-cyan;
    background: rgba($neon-cyan, 0.06);
  }
}

.city-check {
  color: $neon-cyan;
  font-size: $font-size-md;
  font-weight: 700;
}
</style>
