<template>
  <view class="page" :style="{ paddingTop: (statusBarHeight + 44) + 'px' }">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ marginTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">我的收藏</text>
    </view>
    <view v-if="loading" class="list">
      <ListSkeleton :rows="5" variant="row" />
    </view>
    <view v-else-if="favorites.length === 0" class="empty-state">
      <text class="empty-icon">☆</text>
      <text class="empty-text">还没有收藏摄影师</text>
      <view class="btn-go" @click="goHome">去逛逛</view>
    </view>
    <view v-else class="list">
      <view v-for="f in favorites" :key="f.photographerId" class="fav-card" @click="goDetail(f.photographerId)">
        <image class="avatar" :src="f.avatar" mode="aspectFill" />
        <view class="info">
          <text class="name">{{ f.name }}</text>
          <text class="meta">{{ f.location }} · 评分 {{ f.rating }}</text>
        </view>
        <text class="unfav" @click.stop="unfavorite(f.photographerId)">取消</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { apiGet, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'
import { resolveImage } from '@/utils/mappers'

/** 收藏列表 DTO：后端 GET /v1/favorites/:userId 返回裸数组，字段见 service.FavoriteItem */
interface FavoriteDTO {
  photographerId: number
  name: string
  avatar?: string
  location?: string
  rating?: string | number
}

/** 模板视图：avatar 已由 resolveImage 补成绝对地址（/static/remote、/static/uploads 在小程序端没有同源代理） */
interface FavoriteView {
  photographerId: number
  name: string
  avatar: string
  location: string
  rating: string
}

const { statusBarHeight } = useStatusBar()
const favorites = ref<FavoriteView[]>([])
const loading = ref(true)
const userStore = useUserStore()

onMounted(async () => {
  if (!userStore.isLoggedIn || !userStore.user) {
    loading.value = false
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  try {
    const res = await apiGet<FavoriteDTO[]>(`/v1/favorites/${userStore.user.id}`)
    // 头像必须走 resolveImage：后端存的是 /static/remote/**、/static/uploads/** 相对路径，
    // 小程序端会被当包内资源读取（404 → 只剩深色底，看起来是一块黑方块）
    favorites.value = (res || []).map((f) => ({
      photographerId: f.photographerId,
      name: f.name,
      avatar: resolveImage(f.avatar || ''),
      location: f.location || '',
      rating: f.rating === undefined || f.rating === null ? '0' : String(f.rating),
    }))
  } catch {
    favorites.value = []
  } finally {
    loading.value = false
  }
})

async function unfavorite(pid: number) {
  if (!userStore.user) return
  try {
    await apiDelete(`/v1/favorites/${userStore.user.id}/${pid}`)
    favorites.value = favorites.value.filter(f => f.photographerId !== pid)
    uni.showToast({ title: '已取消', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败', icon: 'none' })
  }
}

function goDetail(pid: number) {
  uni.navigateTo({ url: `/pages/photographer/detail?id=${pid}` })
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
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 44px; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.list { padding: 24rpx 32rpx 32rpx; }
.fav-card { display: flex; align-items: center; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 24rpx; margin-bottom: 20rpx; }
.avatar { width: 96rpx; height: 96rpx; border-radius: 50%; margin-right: 24rpx; }
.info { flex: 1; }
.name { display: block; font-size: 30rpx; color: $dark-text-primary; font-weight: 600; }
.meta { display: block; font-size: 24rpx; color: $dark-text-tertiary; margin-top: 8rpx; }
.unfav { font-size: 24rpx; color: $neon-purple; padding: 8rpx 16rpx; }
.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $dark-text-tertiary; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0; }
.btn-go { padding: 16rpx 48rpx; background: $neon-gradient; color: #fff; border-radius: 32rpx; font-size: 28rpx; }
</style>
