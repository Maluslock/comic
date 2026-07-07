<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <scroll-view scroll-y class="content" :style="{ paddingTop: statusBarHeight + 'px' }">
      <view class="header">
        <text class="header-title">漫展摄影</text>
      </view>

      <view class="section search-section">
        <view class="search-entry" @click="goSearch">
          <text class="search-icon">🔍</text>
          <text class="search-text">搜索摄影师、漫展</text>
        </view>
      </view>

      <view class="section banner-section">
        <swiper class="banner-swiper" indicator-dots circular autoplay :interval="4000" indicator-active-color="#6366f1">
          <swiper-item v-for="(item, index) in bannerList" :key="index" @click="onBannerClick(index)">
            <image :src="item.image" class="banner-image" mode="aspectFill" @error="onImgError" />
          </swiper-item>
        </swiper>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">近期漫展</text>
          <text class="section-more" @click="goSearch">更多</text>
        </view>
        <scroll-view scroll-x class="event-scroll" :show-scrollbar="false">
          <view class="event-list">
            <view
              v-for="event in events"
              :key="event.id"
              class="event-card"
              @click="goEventDetail(event.id)"
            >
              <image :src="event.cover" class="event-cover" mode="aspectFill" @error="onImgError" />
              <view class="event-info">
                <text class="event-name ellipsis">{{ event.name }}</text>
                <view class="event-meta">
                  <text class="event-location">{{ event.location }} · {{ event.venue }}</text>
                </view>
                <view class="event-footer">
                  <text class="event-date">{{ formatDate(event.startDate) }} - {{ formatDate(event.endDate) }}</text>
                  <text class="event-badge">{{ event.photographerCount }}位摄影师</text>
                </view>
              </view>
            </view>
          </view>
        </scroll-view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">热门风格</text>
          <text class="section-more" @click="goSearch">更多</text>
        </view>
        <view class="tags-row">
          <view
            v-for="tag in tags"
            :key="tag"
            class="tag-item"
            @click="goSearchByTag(tag)"
          >{{ tag }}</view>
        </view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">推荐摄影师</text>
          <text class="section-more" @click="goPhotographerList">查看全部</text>
        </view>
        <view class="photographer-list">
          <PhotographerCard
            v-for="p in photographers"
            :key="p.id"
            :photographer="p"
          />
        </view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">精选作品</text>
        </view>
        <view class="work-grid">
          <WorkCard
            v-for="work in works"
            :key="work.id"
            :work="work"
          />
        </view>
      </view>

      <view class="bottom-space" />
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotographerCard from '@/components/PhotographerCard.vue'
import WorkCard from '@/components/WorkCard.vue'
import type { Photographer, Work, ComicEvent, HomeResponse } from '@/types'

const statusBarHeight = ref(44)
const photographers = ref<Photographer[]>([])
const works = ref<Work[]>([])
const tags = ref<string[]>([])
const events = ref<ComicEvent[]>([])
const bannerList = ref<{ image: string; title: string }[]>([])

onMounted(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  loadData()
})

const BASE_URL = ''

async function loadData() {
  uni.showLoading({ title: '加载中...' })
  try {
    const res = await uni.request({
      url: `${BASE_URL}/api/v1/home`,
      method: 'GET'
    })
    const data = res.data as HomeResponse

    photographers.value = data.recommendedPhotographers.map(p => ({
      id: String(p.id),
      name: p.name,
      avatar: p.avatar || '',
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
    works.value = data.featuredWorks.slice(0, 4).map(w => ({
      id: String(w.id),
      photographerId: '',
      title: w.title,
      images: w.images,
      description: '',
      tags: [],
      createdAt: Date.now(),
    }))
    tags.value = data.hotTags.map(t => t.name).slice(0, 8)
    events.value = data.upcomingEvents.slice(0, 5).map(e => ({
      id: String(e.id),
      name: e.name,
      location: e.location,
      venue: e.venue,
      startDate: new Date(e.startDate).getTime(),
      endDate: new Date(e.endDate).getTime(),
      cover: e.coverUrl,
      tags: e.tags,
      photographerCount: 0,
      status: (e.status as ComicEvent['status']) || 'upcoming',
    }))
    bannerList.value = data.banners.slice(0, 3).map(b => ({
      image: b.imageUrl,
      title: b.title,
    }))
  } catch (e) {
    console.error('Failed to load home data:', e)
    uni.showToast({ title: '加载失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function formatDate(ts: number): string {
  const d = new Date(ts)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

function goSearch() {
  uni.navigateTo({ url: '/pages/search/search' })
}

function goSearchByTag(tag: string) {
  uni.navigateTo({ url: `/pages/search/search?keyword=${tag}` })
}

function goPhotographerList() {
  uni.switchTab({ url: '/pages/photographer/list' })
}

function onBannerClick(index: number) {
  const event = events.value[index]
  if (event) goEventDetail(event.id)
}

function goEventDetail(id: string) {
  uni.showToast({ title: '漫展详情开发中', icon: 'none' })
}

function onImgError(e: any) {
  const el = e?.target || e?.detail?.target
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.status-bar {
  background: $bg-primary;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
}

.header {
  background: $bg-primary;
  padding: $spacing-md;
}

.header-title {
  font-size: 40rpx;
  font-weight: 800;
  color: $text-primary;
}

.search-section {
  margin-top: $spacing-md;
}

.search-entry {
  display: flex;
  align-items: center;
  padding: 14rpx $spacing-md;
  background: $bg-primary;
  border-radius: $border-radius-xl;
  box-shadow: $shadow-sm;

  &:active {
    background: $bg-secondary;
  }
}

.search-icon {
  font-size: 28rpx;
}

.search-text {
  margin-left: $spacing-xs;
  font-size: $font-size-sm;
  color: $text-placeholder;
}

.content {
  height: 100vh;
}

.section {
  padding: 0 $spacing-md;
  margin-top: $spacing-lg;
}

.banner-section {
  padding: 0 $spacing-md;
  margin-top: $spacing-md;
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
  background: linear-gradient(135deg, #e0e7ff, #ddd6fe);
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
  letter-spacing: 1rpx;
  color: $text-primary;
  padding-left: 20rpx;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 6rpx;
    height: 28rpx;
    background: $primary-color;
    border-radius: 3rpx;
  }
}

.section-more {
  font-size: $font-size-sm;
  color: $primary-color;
  font-weight: 500;
  padding-bottom: 2rpx;
  transition: opacity 0.2s;

  &:active { opacity: 0.6; }
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
  width: 280rpx;
  background: $bg-primary;
  border-radius: $border-radius-lg;
  overflow: hidden;
  box-shadow: $shadow-md;
  transition: transform 0.15s, box-shadow 0.15s;

  &:active {
    transform: scale(0.97);
    box-shadow: $shadow-sm;
  }
}

.event-info {
  padding: $spacing-sm;
}

.event-cover {
  width: 280rpx;
  height: 180rpx;
  background: linear-gradient(135deg, #fce7f3, #e0e7ff);
}

.event-badge {
  font-size: 22rpx;
  color: $primary-color;
  background: rgba($primary-color, 0.1);
  padding: 2rpx 12rpx;
  border-radius: $border-radius-sm;
}

.event-name {
  font-size: 28rpx;
  font-weight: 600;
  color: $text-primary;
}

.event-meta {
  display: flex;
  align-items: center;
  margin-top: $spacing-xs;
}

.event-location {
  font-size: 22rpx;
  color: $text-tertiary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: $spacing-xs;
}

.event-date {
  font-size: 22rpx;
  color: $primary-color;
  font-weight: 500;
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.tag-item {
  padding: 8rpx 24rpx;
  font-size: $font-size-sm;
  color: $primary-color;
  background: rgba($primary-color, 0.08);
  border-radius: $border-radius-xl;
  border: 2rpx solid rgba($primary-color, 0.12);

  &:active {
    background: rgba($primary-color, 0.15);
  }
}

.photographer-list {
  padding-bottom: $spacing-sm;
}

.work-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
  padding-bottom: $spacing-sm;
}

.bottom-space {
  height: 100rpx;
}
</style>
