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
        <!-- Hero Header -->
        <view class="hero">
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

      <view class="hero-title-block">
        <text class="hero-title-main">米拉漫展</text>
        <view class="hero-divider" />
        <text class="hero-title-sub">找到你的专属摄影师</text>
      </view>

          <view class="hero-cta">
            <view class="cta-btn cta-primary" @click="goEventList">
              <text class="cta-marker">◆</text>
              <text>近期热门漫展</text>
            </view>
          </view>
        </view>

        <!-- Countdown Highlight -->
        <view v-if="nearestEvent" class="countdown-bar" @click="goEventDetail(nearestEvent.id)">
          <text class="countdown-label">距 {{ nearestEvent.name }} 开展还有</text>
          <text class="countdown-days">{{ store.countdownDays(nearestEvent.startDate) }}</text>
          <text class="countdown-label">天</text>
          <text class="countdown-arrow">→</text>
        </view>

        <!-- Banner Swiper -->
        <view class="section banner-section">
          <swiper
            class="banner-swiper"
            indicator-dots
            circular
            autoplay
            :interval="4000"
            indicator-active-color="#a855f7"
            indicator-color="rgba(255,255,255,0.2)"
          >
            <swiper-item v-for="(item, i) in store.banners" :key="i" @click="onBannerClick(i)">
              <view
                class="banner-image"
                :style="{ backgroundImage: 'url(' + item.image + ')' }"
              />
              <view class="banner-overlay">
                <text class="banner-title">{{ item.title }}</text>
              </view>
            </swiper-item>
          </swiper>
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
const unreadCount = ref(3)
const showCityPicker = ref(false)

const nearestEvent = computed(() => {
  const upcoming = filteredEvents.value
    .filter(e => e.startDate > Date.now())
    .sort((a, b) => a.startDate - b.startDate)
  return upcoming.length > 0 ? upcoming[0] : null
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
  if (currentCity.value === '全部') return store.upcomingEvents
  return store.events.filter(e => e.status === 'upcoming' && e.location === currentCity.value).slice(0, 6)
})

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

// ===== Hero =====
.hero {
  background: linear-gradient(180deg, $dark-bg-secondary 0%, $dark-bg-primary 100%);
  padding: $spacing-sm $spacing-md $spacing-lg;
}

.hero-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-lg;
}

.city-selector {
  display: flex;
  align-items: center;
  padding: 6rpx 16rpx;
  background: $dark-bg-card;
  border-radius: $border-radius-xl;
  border: 1rpx solid $dark-border;
}

.city-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: $neon-cyan;
  margin-right: 6rpx;
  box-shadow: 0 0 8rpx $neon-cyan-glow;
}

.city-icon {
  font-size: 24rpx;
  margin-right: 4rpx;
}

.city-text {
  font-size: 26rpx;
  color: $dark-text-primary;
  font-weight: 500;
}

.city-arrow {
  font-size: 20rpx;
  color: $dark-text-tertiary;
  margin-left: 4rpx;
}

.hero-actions {
  display: flex;
  gap: $spacing-sm;
}

.hero-icon-btn {
  position: relative;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: $dark-bg-card;
  border-radius: 50%;
  border: 1rpx solid $dark-border;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.hero-icon-svg {
  width: 36rpx;
  height: 36rpx;
  opacity: 0.7;
}

.badge {
  position: absolute;
  top: -4rpx;
  right: -4rpx;
  min-width: 28rpx;
  height: 28rpx;
  line-height: 28rpx;
  text-align: center;
  font-size: 18rpx;
  @include on-neon-fill;
  background: $neon-pink;
  border-radius: 14rpx;
  padding: 0 6rpx;
}

.hero-title-block {
  text-align: center;
  padding: $spacing-lg 0 $spacing-md;
}

.hero-title-main {
  display: block;
  font-size: 56rpx;
  font-weight: 900;
  letter-spacing: 4rpx;
  background: $neon-gradient;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: $spacing-sm;
}

.hero-title-sub {
  display: block;
  font-size: 28rpx;
  color: $dark-text-secondary;
  font-weight: 400;
  letter-spacing: 6rpx;
}

.hero-divider {
  width: 60rpx;
  height: 4rpx;
  margin: $spacing-md auto;
  background: $neon-gradient;
  border-radius: 2rpx;
  opacity: 0.6;
}

.hero-cta {
  display: flex;
  gap: $spacing-md;
}

.cta-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  padding: 18rpx 0;
  border-radius: $border-radius-xl;
  font-size: 26rpx;
  font-weight: 600;
  transition: all 0.2s;

  .cta-icon {
    font-size: 28rpx;
  }
}

.cta-marker {
  font-size: 20rpx;
}

.cta-primary {
  background: $neon-gradient;
  @include on-neon-fill;
  box-shadow: 0 4rpx 24rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 2rpx 12rpx $neon-purple-glow;
  }
}

.cta-secondary {
  background: $dark-bg-card;
  color: $dark-text-primary;
  border: 2rpx solid $neon-purple-glow;

  &:active {
    background: $dark-bg-card-hover;
    border-color: $neon-purple;
  }
}

// ===== Countdown Bar =====
.countdown-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  padding: 12rpx $spacing-md;
  margin: $spacing-sm $spacing-md 0;
  background: $neon-purple-dim;
  border: 1rpx solid $neon-purple-glow;
  border-radius: $border-radius-lg;
}

.countdown-label {
  font-size: 24rpx;
  color: $dark-text-secondary;
}

.countdown-days {
  font-size: 32rpx;
  font-weight: 900;
  color: $neon-purple;
  margin: 0 4rpx;
}

.countdown-arrow {
  font-size: 28rpx;
  color: $neon-purple;
  margin-left: 4rpx;
}

// ===== Banner =====
.banner-section {
  padding: 0 $spacing-md;
  margin-top: $spacing-sm;
}

.banner-swiper {
  width: 100%;
  height: 320rpx;
  border-radius: $border-radius-lg;
  overflow: hidden;
}

.banner-image {
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center top;
  background-color: $dark-bg-card;
}

.banner-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: $spacing-md;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.6));
}

.banner-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #fff;
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
  font-size: $font-size-lg;
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
  font-size: $font-size-sm;
  color: $neon-cyan;
  font-weight: 500;

  &:active { opacity: 0.6; }
}

// ===== Events =====
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
  width: 280rpx;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  overflow: hidden;
  transition: transform 0.15s, border-color 0.15s;

  &:active {
    transform: scale(0.97);
    border-color: $neon-purple-glow;
  }
}

.event-cover {
  width: 280rpx;
  height: 180rpx;
  background-size: cover;
  background-position: center top;
  background-color: $dark-bg-card;
}

.event-countdown-badge {
  position: absolute;
  top: 12rpx;
  right: 12rpx;
  padding: 4rpx 14rpx;
  background: rgba(0, 0, 0, 0.7);
  border-radius: $border-radius-sm;
  font-size: 20rpx;
  color: $neon-cyan;
  font-weight: 600;
}

.event-info {
  padding: $spacing-sm;
}

.event-name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 26rpx;
  font-weight: 600;
  color: $dark-text-primary;
}

.event-meta {
  margin-top: $spacing-xs;
}

.event-location {
  display: block;
  font-size: 22rpx;
  color: $dark-text-tertiary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-footer {
  margin-top: $spacing-xs;
}

.event-date {
  font-size: 22rpx;
  color: $neon-cyan;
  font-weight: 500;
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
  @include on-neon-fill;
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
