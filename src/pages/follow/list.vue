<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">关注的漫展</text>
    </view>
    <view v-if="loading" class="empty-hint">加载中...</view>
    <view v-else-if="follows.length === 0" class="empty-state">
      <text class="empty-icon">◇</text>
      <text class="empty-text">还没有关注的漫展</text>
      <view class="btn-go" @click="goHome">去逛逛</view>
    </view>
    <view v-else class="list">
      <view v-for="f in follows" :key="f.eventId" class="follow-card" @click="goDetail(f.eventId)">
        <image class="cover" :src="f.coverUrl" mode="aspectFill" />
        <view class="info">
          <text class="name ellipsis">{{ f.name }}</text>
          <text class="meta">{{ f.location }} · {{ f.venue }}</text>
          <text class="date">{{ formatRange(f.startDate, f.endDate) }}</text>
          <text v-if="daysUntil(f.startDate) > 0" class="countdown">{{ daysUntil(f.startDate) }}天后开展</text>
        </view>
        <text class="unfollow" @click.stop="unfollow(f.eventId)">取消</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const follows = ref<any[]>([])
const loading = ref(true)
const userStore = useUserStore()

onShow(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    loading.value = false
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  try {
    const res = await apiGet<{ list: any[] }>(`/v1/follows/${userStore.user.id}`)
    follows.value = res.list || []
  } catch {
    follows.value = []
  } finally {
    loading.value = false
  }
})

async function unfollow(eventId: number) {
  if (!userStore.user) return
  try {
    await apiDelete(`/v1/follows/${userStore.user.id}/${eventId}`)
    follows.value = follows.value.filter(f => f.eventId !== eventId)
    uni.showToast({ title: '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}

function formatRange(start: string, end: string): string {
  const s = new Date(start)
  const e = new Date(end)
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  return `${s.getFullYear()}-${pad(s.getMonth() + 1)}-${pad(s.getDate())} 至 ${pad(e.getMonth() + 1)}-${pad(e.getDate())}`
}

function daysUntil(start: string): number {
  const ms = new Date(start).setHours(0, 0, 0, 0) - new Date().setHours(0, 0, 0, 0)
  return Math.max(0, Math.ceil(ms / 86400000))
}

function goDetail(eventId: number) {
  uni.navigateTo({ url: `/pages/event/detail?id=${eventId}` })
}

function goHome() {
  uni.switchTab({ url: '/pages/index/index' })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.list { padding: 100rpx 32rpx 32rpx; }
.follow-card { display: flex; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx; margin-bottom: 20rpx; }
.cover { width: 160rpx; height: 120rpx; border-radius: 12rpx; background: $dark-bg-secondary; flex-shrink: 0; }
.info { flex: 1; margin-left: 20rpx; overflow: hidden; }
.name { display: block; font-size: 28rpx; color: $dark-text-primary; font-weight: 600; }
.meta { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 6rpx; }
.date { display: block; font-size: 22rpx; color: $dark-text-secondary; margin-top: 6rpx; }
.countdown { display: inline-block; font-size: 20rpx; color: $neon-cyan; margin-top: 8rpx; }
.unfollow { font-size: 24rpx; color: $neon-purple; padding: 8rpx 16rpx; align-self: center; }
.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $dark-text-tertiary; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0; }
.btn-go { padding: 16rpx 48rpx; background: $neon-gradient; @include on-neon-fill; border-radius: 32rpx; font-size: 28rpx; }
.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }
</style>
