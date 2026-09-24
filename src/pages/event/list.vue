<template>
  <view class="page">
    <view class="filter-bar">
      <view class="filter-list">
        <view
          v-for="item in statusTabs"
          :key="item.key"
          class="filter-item"
          :class="{ active: currentStatus === item.key }"
          @click="setStatus(item.key)"
        >
          {{ item.label }}
        </view>
      </view>
    </view>

    <scroll-view
      scroll-y
      class="content"
      @scrolltolower="loadMore"
    >
      <view class="event-list">
        <view
          v-for="event in events"
          :key="event.id"
          class="event-card"
          @click="goDetail(event.id)"
        >
          <view class="cover-wrap">
            <view
              class="event-cover"
              :style="{ backgroundImage: 'url(' + event.cover + ')' }"
            />
            <view class="status-badge" :class="event.status">
              <text>{{ statusLabel(event.status) }}</text>
            </view>
          </view>
          <view class="event-info">
            <text class="event-name">{{ event.name }}</text>
            <view class="event-meta">
              <text class="event-location">{{ event.location }} · {{ event.venue }}</text>
            </view>
            <view class="event-date">{{ formatDateRange(event.startDate, event.endDate) }}</view>
          </view>
        </view>
      </view>

      <view v-if="loading" class="loading">
        <text>加载中...</text>
      </view>

      <view v-if="!loading && events.length >= total && events.length > 0" class="no-more">
        <text>没有更多了</text>
      </view>

      <view v-if="!loading && events.length === 0" class="empty-state">
        <text class="empty-icon">!</text>
        <text class="empty-text">暂无{{ currentLabel }}漫展</text>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { mockEvents } from '@/data/mock'
import type { ComicEvent } from '@/types'
import { apiGet } from '@/api/client'

interface EventItem {
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
  description?: string
}

interface EventListResponse {
  list: EventItem[]
  total: number
}

const events = ref<ComicEvent[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const currentStatus = ref<'upcoming' | 'ongoing' | 'ended'>('upcoming')

const statusTabs = [
  { key: 'upcoming', label: '即将开始' },
  { key: 'ongoing', label: '进行中' },
  { key: 'ended', label: '已结束' }
] as const

const currentLabel = computed(() => {
  const found = statusTabs.find(t => t.key === currentStatus.value)
  return found ? found.label : ''
})

onMounted(() => {
  loadEvents()
})

async function loadEvents(targetPage = 1) {
  loading.value = true
  try {
    const res = await apiGet<EventListResponse>('/v1/events', {
      status: currentStatus.value,
      page: String(targetPage),
      size: '10',
    })

    let list: ComicEvent[] = []
    let fetchedTotal = 0

      list = (res.list || []).map(item => ({
        id: String(item.id),
        name: item.name,
        location: item.location,
        venue: item.venue,
        startDate: new Date(item.startDate).getTime(),
        endDate: new Date(item.endDate).getTime(),
        cover: item.coverUrl,
        tags: item.tags || [],
        photographerCount: 0,
        status: item.status as ComicEvent['status'],
        description: item.description
      }))
      fetchedTotal = res.total || 0

    if (list.length === 0) {
      const fallback = mockEvents
        .filter(e => e.status === currentStatus.value)
        .sort((a, b) => a.startDate - b.startDate)
      const size = 10
      fetchedTotal = fallback.length
      list = fallback.slice((targetPage - 1) * size, targetPage * size)
    }

    if (targetPage === 1) {
      events.value = list
    } else {
      events.value = [...events.value, ...list]
    }
    total.value = fetchedTotal
    page.value = targetPage
  } catch (e: unknown) {
    console.error('Load events failed:', e)
    const fallback = mockEvents
      .filter(e => e.status === currentStatus.value)
      .sort((a, b) => a.startDate - b.startDate)
    const size = 10
    const list = fallback.slice((targetPage - 1) * 10, targetPage * 10)
    if (targetPage === 1) {
      events.value = list
    } else {
      events.value = [...events.value, ...list]
    }
    total.value = fallback.length
    page.value = targetPage
  } finally {
    loading.value = false
  }
}

function loadMore() {
  if (loading.value || events.value.length >= total.value) return
  loadEvents(page.value + 1)
}

function setStatus(status: 'upcoming' | 'ongoing' | 'ended') {
  currentStatus.value = status
  page.value = 1
  total.value = 0
  events.value = []
  loadEvents()
}

function statusLabel(status: ComicEvent['status']) {
  if (status === 'ongoing') return '进行中'
  if (status === 'ended') return '已结束'
  return '即将开始'
}

function formatDateRange(start: number, end: number): string {
  const s = new Date(start)
  const e = new Date(end)
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  return `${s.getFullYear()}-${pad(s.getMonth() + 1)}-${pad(s.getDate())} 至 ${pad(e.getMonth() + 1)}-${pad(e.getDate())}`
}

function goDetail(id: string) {
  uni.navigateTo({ url: `/pages/event/detail?id=${id}` })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.filter-bar {
  position: sticky;
  top: 0;
  background: $dark-bg-secondary;
  z-index: 100;
  border-bottom: 1rpx solid $dark-border;
}

.filter-list {
  display: flex;
  padding: $spacing-sm $spacing-md;
  gap: $spacing-sm;
}

.filter-item {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  padding: $spacing-xs 0;
  border-radius: $border-radius-xl;
  border: 1rpx solid transparent;
  transition: all 0.15s;

  &.active {
    @include neon-pill;
    border-color: rgba($neon-purple, 0.3);
  }

  &:active {
    background: $dark-bg-card-hover;
  }
}

.content {
  height: 100vh;
}

.event-list {
  padding: $spacing-md;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.event-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  overflow: hidden;
  transition: transform 0.15s, border-color 0.15s;

  &:active {
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 20rpx $neon-purple-glow;
  }
}

.cover-wrap {
  position: relative;
  width: 100%;
  height: 320rpx;
}

.event-cover {
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center top;
  background-color: $dark-bg-card;
}

.status-badge {
  position: absolute;
  top: $spacing-sm;
  right: $spacing-sm;
  padding: 6rpx 18rpx;
  border-radius: $border-radius-sm;
  font-size: $font-size-xs;
  font-weight: 600;
  background: rgba(0, 0, 0, 0.5);
  border: 1rpx solid currentColor;

  &.upcoming {
    color: $neon-cyan;
    background: rgba($neon-cyan, 0.12);
    border-color: rgba($neon-cyan, 0.3);
  }

  &.ongoing {
    color: $success-color;
    background: rgba($success-color, 0.12);
    border-color: rgba($success-color, 0.3);
  }

  &.ended {
    color: $dark-text-tertiary;
    background: rgba($dark-text-tertiary, 0.12);
    border-color: rgba($dark-text-tertiary, 0.3);
  }
}

.event-info {
  padding: $spacing-md;
}

.event-name {
  display: block;
  font-size: $font-size-base;
  font-weight: 700;
  color: $dark-text-primary;
  line-height: 1.4;
}

.event-meta {
  margin-top: $spacing-xs;
}

.event-location {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  line-height: 1.4;
}

.event-date {
  margin-top: $spacing-xs;
  font-size: $font-size-sm;
  color: $neon-cyan;
  font-weight: 500;
}

.loading,
.no-more {
  text-align: center;
  padding: $spacing-md;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2 $spacing-md;
}

.empty-icon {
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

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
}

.bottom-space {
  height: 120rpx;
}
</style>
