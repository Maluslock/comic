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
          <!-- 封面三态兜底：loading=中性骨架 / loaded=<image> 淡入 / error=图标+文案。
               用 <image> 而不是 background-image，才能拿到小程序端的 @load / @error 回调。
               .cover-wrap 固定 320rpx，三态共用同一个盒子，高度不随图片加载状态变化。 -->
          <view
            class="cover-wrap"
            :class="{
              'is-loaded': coverState(event) === 'loaded',
              'is-error': coverState(event) === 'error',
            }"
          >
            <view v-if="coverState(event) === 'loading'" class="cover-skeleton" />
            <image
              v-if="event.cover && coverState(event) !== 'error'"
              class="event-cover"
              :src="event.cover"
              mode="aspectFill"
              lazy-load
              @load="onCoverLoad(event.id)"
              @error="onCoverError(event.id)"
            />
            <view v-if="coverState(event) === 'error'" class="cover-fallback">
              <text class="cover-fallback-icon">▧</text>
              <text class="cover-fallback-text">{{ coverFallbackText(event) }}</text>
            </view>
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

      <ListSkeleton v-if="loading && events.length === 0" :rows="4" variant="card" />

      <view v-if="loading && events.length > 0" class="loading">
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
import ListSkeleton from '@/components/ListSkeleton.vue'
import { mockEvents } from '@/data/mock'
import type { ComicEvent } from '@/types'
import { apiGet } from '@/api/client'
// 统一走 mapping 层：cover 由 resolveImage 补成绝对地址（小程序端没有 /static 代理，
// 后端相对路径 /static/remote/** 会被当成包内资源读取失败，露出深色卡片底）。
import { mapEventItem, type EventDTO } from '@/utils/mappers'

interface EventListResponse {
  list: EventDTO[]
  total: number
}

const events = ref<ComicEvent[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const currentStatus = ref<'upcoming' | 'ongoing' | 'ended'>('upcoming')

// --- 封面三态 ---
// 只记录「已经出结果」的卡片（loaded / error）。没有记录 = 还在加载。
// 不做任何额外的图片探测请求：状态完全由 <image> 自身的 @load / @error 驱动。
type CoverState = 'loading' | 'loaded' | 'error'
const coverStates = ref<Record<string, 'loaded' | 'error'>>({})

function coverState(event: ComicEvent): CoverState {
  // 后端/mock 没给封面时直接进兜底态，避免永远停在骨架上
  if (!event.cover) return 'error'
  return coverStates.value[event.id] ?? 'loading'
}

function coverFallbackText(event: ComicEvent): string {
  return event.cover ? '封面加载失败' : '暂无封面'
}

function onCoverLoad(id: string) {
  coverStates.value = { ...coverStates.value, [id]: 'loaded' }
}

function onCoverError(id: string) {
  coverStates.value = { ...coverStates.value, [id]: 'error' }
}

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

    // 统一经 mapEventItem：cover 由 resolveImage 补成绝对地址后再进模板，
    // 模板只做 `url(${event.cover})` 拼接，不再拼后端相对路径。
    list = (res.list || []).map(mapEventItem)
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
    background: rgba($neon-purple, 0.15);
    color: $neon-purple;
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
  overflow: hidden;
  // 中性占位底：$surface-2(#1d202f) 而不是 $dark-bg-card(#151722)。
  // 弱网时这一层会先露出来，用近黑会继续被读成「海报全黑」。
  background: $surface-2;
}

// 加载中骨架：低对比灰 + 缓慢 shimmer，与 HomeSkeleton 同一套写法。
// 颜色只到 rgba(255,255,255,.04~.08) 量级（设计系统里没有对应 token，
// 这是骨架动效的标准做法：纯白低透明度叠在 $surface-2 上，不会读成黑块）。
.cover-skeleton {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background-color: $surface-2;
  background-image: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.04) 25%,
    rgba(255, 255, 255, 0.08) 50%,
    rgba(255, 255, 255, 0.04) 75%
  );
  background-size: 200% 100%;
  // shimmer 写在基础规则上：小程序端 wxss 不支持 prefers-reduced-motion，
  // 放进媒体查询里动画永远不会跑。H5 端由下方 reduce 分支关掉。
  animation: cover-shimmer 1.5s ease-in-out infinite;
}

@keyframes cover-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

// H5：系统开启「减少动效」时停掉 shimmer，骨架保持静止（仍可见，不影响理解）。
// 小程序端忽略该媒体查询，动画照常。
@media (prefers-reduced-motion: reduce) {
  .cover-skeleton {
    animation: none;
  }
}

.event-cover {
  display: block;
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
  // 底色与骨架一致，避免图片解码瞬间闪出近黑
  background-color: $surface-2;
  opacity: 0;
  transition: opacity $motion-base;
}

.cover-wrap.is-loaded .event-cover {
  opacity: 1;
}

// 兜底态：图标 + 文案，居中；尺寸与加载态完全一致（同一个 320rpx 盒子）
.cover-fallback {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $s-xs;
  background: $surface-2;
}

.cover-fallback-icon {
  font-size: $t-2xl;
  line-height: 1;
  color: $text-3;
}

.cover-fallback-text {
  font-size: $font-size-xs;
  color: $text-3;
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
