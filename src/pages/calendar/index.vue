<template>
  <view class="page">
    <scroll-view scroll-y class="page-scroll">
      <view class="calendar-header">
        <view class="today-btn" @click="goToday">今天</view>
        <view class="month-switcher">
          <view class="nav-arrow" @click="prevMonth">&#x2039;</view>
          <text class="month-label" @click="openYearPanel">{{ monthLabel }}</text>
          <view class="nav-arrow" @click="nextMonth">&#x203a;</view>
        </view>
        <view class="year-btn" @click="openYearPanel">▦</view>
      </view>

      <view class="week-row">
        <text v-for="d in weekDays" :key="d" class="week-day">{{ d }}</text>
      </view>

      <view class="day-grid">
        <view
          v-for="cell in grid"
          :key="cell.key"
          class="day-cell"
          :class="{
            'other-month': !cell.isCurrentMonth,
            today: cell.isToday,
            'has-events': cell.hasEvents,
            selected: selectedKey === cell.key,
          }"
          @click="selectDate(cell.key)"
        >
          <text class="day-number">{{ cell.day }}</text>
          <view v-if="cell.hasEvents" class="event-dot" />
        </view>
      </view>

      <view v-if="selectedEvents.length" class="event-panel">
        <view class="panel-header">
          <text class="panel-title">{{ selectedLabel }}</text>
          <text class="panel-close" @click="selectedKey = null">&#x2715;</text>
        </view>
        <view class="event-list">
          <view
            v-for="evt in selectedEvents"
            :key="evt.id"
            class="event-card"
            @click="goDetail(evt.id)"
          >
            <view class="event-cover" :style="{ backgroundImage: 'url(' + evt.cover + ')' }" />
            <view class="event-info">
              <text class="event-name">{{ evt.name }}</text>
              <text class="event-location">{{ evt.location }} &#183; {{ evt.venue }}</text>
              <text class="event-date">{{ formatRange(evt.startDate, evt.endDate) }}</text>
              <view class="event-footer">
                <text v-if="isFollowed(evt.id)" class="follow-hint">
                  已关注 &#183; {{ daysUntil(evt.startDate) }}天后开展
                </text>
                <view
                  class="follow-btn"
                  :class="{ followed: isFollowed(evt.id) }"
                  @click.stop="onToggleFollow(evt.id)"
                >
                  <text>{{ isFollowed(evt.id) ? '已关注' : '关注' }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <view v-if="selectedEvents.length === 0" class="empty-hint">
        <text>点击日历中的日期查看当天漫展</text>
      </view>
      <view class="bottom-space" />
    </scroll-view>

    <view v-if="showYearPanel" class="year-mask" @click="showYearPanel = false">
      <view class="year-panel" @click.stop>
        <view class="year-header">
          <view class="nav-arrow" @click="panelYear--">&#x2039;</view>
          <text class="year-label">{{ panelYear }}年</text>
          <view class="nav-arrow" @click="panelYear++">&#x203a;</view>
        </view>
        <view class="month-grid">
          <view
            v-for="m in 12"
            :key="m"
            class="month-cell"
            :class="{
              current: panelYear === current.getFullYear() && m - 1 === current.getMonth(),
              'has-events': monthHasEvents(panelYear, m),
            }"
            @click="jumpToMonth(m)"
          >
            {{ m }}月
            <view v-if="monthHasEvents(panelYear, m)" class="month-dot" />
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet } from '@/api/client'
import { loadFollowedIds, toggleFollow } from '@/utils/follow'
import { useUserStore } from '@/stores/user'

interface CalendarEvent {
  id: string
  name: string
  location: string
  venue: string
  startDate: number
  endDate: number
  cover: string
  tags: string[]
  status: string
}

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
}

interface EventListResponse {
  list: EventItem[]
  total: number
}

interface DayCell {
  key: string
  date: Date
  day: number
  isCurrentMonth: boolean
  isToday: boolean
  hasEvents: boolean
}

const userStore = useUserStore()
const weekDays = ['一', '二', '三', '四', '五', '六', '日']

const current = ref(new Date())
const events = ref<CalendarEvent[]>([])
const follows = ref<Set<string>>(new Set())
const selectedKey = ref<string | null>(null)
const loading = ref(false)

const todayKey = dateKey(new Date())

const monthLabel = computed(() => {
  return `${current.value.getFullYear()}年${current.value.getMonth() + 1}月`
})

const eventMap = computed(() => {
  const map = new Map<string, CalendarEvent[]>()
  for (const e of events.value) {
    const key = dateKey(new Date(e.startDate))
    const list = map.get(key) || []
    list.push(e)
    map.set(key, list)
  }
  return map
})

const grid = computed<DayCell[]>(() => {
  const year = current.value.getFullYear()
  const month = current.value.getMonth()
  const first = new Date(year, month, 1)
  const offset = (first.getDay() + 6) % 7
  const start = new Date(year, month, 1 - offset)
  const cells: DayCell[] = []
  for (let i = 0; i < 42; i++) {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i)
    const key = dateKey(d)
    cells.push({
      key,
      date: d,
      day: d.getDate(),
      isCurrentMonth: d.getMonth() === month,
      isToday: key === todayKey,
      hasEvents: (eventMap.value.get(key)?.length ?? 0) > 0,
    })
  }
  return cells
})

const selectedEvents = computed(() => {
  if (!selectedKey.value) return []
  return eventMap.value.get(selectedKey.value) || []
})

const selectedLabel = computed(() => {
  if (!selectedKey.value) return ''
  const [y, m, d] = selectedKey.value.split('-').map(Number)
  return `${y}年${m}月${d}日`
})

function dateKey(d: Date): string {
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function prevMonth() {
  const d = current.value
  current.value = new Date(d.getFullYear(), d.getMonth() - 1, 1)
}

function nextMonth() {
  const d = current.value
  current.value = new Date(d.getFullYear(), d.getMonth() + 1, 1)
}

function selectDate(key: string) {
  selectedKey.value = key
}

const showYearPanel = ref(false)
const panelYear = ref(current.value.getFullYear())

function openYearPanel() {
  panelYear.value = current.value.getFullYear()
  showYearPanel.value = true
}

function goToday() {
  current.value = new Date()
  selectedKey.value = todayKey
}

function jumpToMonth(m: number) {
  current.value = new Date(panelYear.value, m - 1, 1)
  showYearPanel.value = false
}

function monthHasEvents(year: number, month: number): boolean {
  const key = `${year}-${month < 10 ? '0' + month : month}`
  for (const e of events.value) {
    const d = new Date(e.startDate)
    const dk = dateKey(d).slice(0, 7)
    if (dk === key) return true
  }
  return false
}

function isFollowed(id: string): boolean {
  return follows.value.has(id)
}

async function onToggleFollow(id: string) {
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/calendar/index' }), 800)
    return
  }
  try {
    const nowFollowed = follows.value.has(id)
    const newState = await toggleFollow(userStore.user.id, id, nowFollowed)
    if (newState) follows.value.add(id)
    else follows.value.delete(id)
    uni.showToast({ title: newState ? '关注成功' : '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}

function formatRange(start: number, end: number): string {
  const s = new Date(start)
  const e = new Date(end)
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  return `${s.getFullYear()}-${pad(s.getMonth() + 1)}-${pad(s.getDate())} 至 ${pad(e.getMonth() + 1)}-${pad(e.getDate())}`
}

function daysUntil(start: number): number {
  const ms = new Date(start).setHours(0, 0, 0, 0) - new Date().setHours(0, 0, 0, 0)
  return Math.max(0, Math.ceil(ms / 86400000))
}

function goDetail(id: string) {
  uni.navigateTo({ url: `/pages/event/detail?id=${id}` })
}

async function loadEvents() {
  try {
    const res = await apiGet<EventListResponse>('/v1/events', { status: 'upcoming' })
    events.value = (res.list || []).map(item => ({
      id: String(item.id),
      name: item.name,
      location: item.location,
      venue: item.venue,
      startDate: new Date(item.startDate).getTime(),
      endDate: new Date(item.endDate).getTime(),
      cover: item.coverUrl,
      tags: item.tags || [],
      status: item.status,
    }))
  } catch (e: unknown) {
    events.value = []
  }
}

async function loadFollows() {
  if (!userStore.user) { follows.value = new Set(); return }
  follows.value = await loadFollowedIds(userStore.user.id)
}

onShow(() => {
  loading.value = true
  Promise.all([loadEvents(), loadFollows()]).finally(() => {
    loading.value = false
  })
})
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.page-scroll {
  height: 100vh;
}

.calendar-header {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-md $spacing-lg;
  background: $dark-bg-secondary;
  border-bottom: 1rpx solid $dark-border;
}

.nav-arrow {
  width: 56rpx;
  height: 56rpx;
  line-height: 56rpx;
  text-align: center;
  font-size: 40rpx;
  color: $neon-cyan;
  background: $dark-bg-card;
  border-radius: 50%;
  border: 1rpx solid $dark-border;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.month-label {
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
  padding: $spacing-xs $spacing-sm;
}

.today-btn {
  font-size: $font-size-sm;
  color: $neon-cyan;
  padding: $spacing-xs $spacing-md;
  border: 1rpx solid $neon-cyan-glow;
  border-radius: $border-radius-xl;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.year-btn {
  width: 56rpx;
  height: 56rpx;
  line-height: 56rpx;
  text-align: center;
  font-size: 32rpx;
  color: $neon-purple;
  background: $dark-bg-card;
  border-radius: 50%;
  border: 1rpx solid $dark-border;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.month-switcher {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.year-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
}

.year-panel {
  width: 80%;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-lg;
  box-shadow: 0 8rpx 40rpx rgba(0, 0, 0, 0.5);
}

.year-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-lg;
  margin-bottom: $spacing-lg;
}

.year-label {
  font-size: $font-size-xl;
  font-weight: 700;
  color: $dark-text-primary;
  min-width: 140rpx;
  text-align: center;
}

.month-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: $spacing-sm;
}

.month-cell {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: $spacing-md 0;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: $dark-text-primary;

  &.current {
    border-color: $neon-cyan;
    color: $neon-cyan;
    box-shadow: 0 0 12rpx $neon-cyan-glow;
  }

  &:active {
    background: $dark-bg-card-hover;
  }
}

.month-dot {
  width: 8rpx;
  height: 8rpx;
  border-radius: 50%;
  background: $neon-purple;
  margin-top: 4rpx;
  box-shadow: 0 0 6rpx $neon-purple-glow;
}

.week-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-secondary;
  border-bottom: 1rpx solid $dark-border;
}

.week-day {
  text-align: center;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.day-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 12rpx;
  padding: $spacing-sm $spacing-md;
}

.day-cell {
  aspect-ratio: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-md;
  border: 2rpx solid transparent;
  background: $dark-bg-card;

  &:active {
    background: $dark-bg-card-hover;
  }

  &.other-month {
    opacity: 0.4;
  }

  &.today {
    border-color: $neon-cyan;
    box-shadow: 0 0 12rpx $neon-cyan-glow;
  }

  &.selected {
    border-color: $neon-purple;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.day-number {
  font-size: $font-size-base;
  color: $dark-text-primary;
  font-weight: 500;
}

.event-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: $neon-purple;
  margin-top: 8rpx;
  box-shadow: 0 0 8rpx $neon-purple-glow;
}

.event-panel {
  margin: $spacing-md;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  border-bottom: 1rpx solid $dark-border;
}

.panel-title {
  font-size: $font-size-md;
  font-weight: 700;
  color: $dark-text-primary;
}

.panel-close {
  font-size: 32rpx;
  color: $dark-text-tertiary;
  padding: $spacing-xs;
}

.event-list {
  padding: $spacing-md;
}

.event-card {
  display: flex;
  gap: $spacing-md;
  padding: $spacing-md;
  margin-bottom: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;

  &:active {
    border-color: $neon-purple-glow;
  }
}

.event-cover {
  width: 180rpx;
  height: 180rpx;
  flex-shrink: 0;
  border-radius: $border-radius-md;
  background-size: cover;
  background-position: center top;
  background-color: $dark-bg-card;
}

.event-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 0;
}

.event-name {
  font-size: $font-size-md;
  font-weight: 600;
  color: $dark-text-primary;
}

.event-location {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.event-date {
  font-size: $font-size-sm;
  color: $neon-cyan;
}

.event-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.follow-hint {
  font-size: $font-size-xs;
  color: $neon-purple;
}

.follow-btn {
  padding: 8rpx 24rpx;
  background: $neon-purple;
  color: #fff;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  font-weight: 500;

  &:active {
    opacity: 0.8;
  }

  &.followed {
    background: transparent;
    color: $neon-purple;
    border: 1rpx solid $neon-purple;
  }
}

.loading-mask {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba($dark-bg-primary, 0.7);
  z-index: 50;
}

.loading-spinner {
  width: 48rpx;
  height: 48rpx;
  border-radius: 50%;
  border: 4rpx solid $dark-border;
  border-top-color: $neon-cyan;
  animation: spin 1s linear infinite;
}

.empty-hint {
  text-align: center;
  padding: $spacing-xl 0;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.bottom-space {
  height: $spacing-xl;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
