<template>
  <view class="page">
    <view class="header">
      <text class="back" @click="goBack">‹</text>
      <text class="header-title">黑名单管理</text>
    </view>

    <view class="content">
      <view v-if="loading" class="hint">加载中...</view>

      <view v-else-if="blocks.length === 0" class="empty">
        <text class="empty-icon">▪</text>
        <text class="empty-text">黑名单是空的</text>
        <text class="empty-desc">在摄影师主页点击「加入黑名单」后，对方不会再出现在你的搜索与推荐中</text>
      </view>

      <view v-else class="block-list">
        <view v-for="b in blocks" :key="b.userId" class="block-item">
          <image class="avatar" :src="b.avatar || DEFAULT_AVATAR" mode="aspectFill" @error="onAvatarError(b)" />
          <view class="block-info">
            <text class="block-name">{{ b.name || ('用户' + b.userId) }}</text>
            <text class="block-sub">已拉黑</text>
          </view>
          <text class="unblock-btn" @click="onUnblock(b)">解除</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { getBlocks, unblockUser, type BlockedUser } from '@/api/index'
import { useUserStore } from '@/stores/user'

const DEFAULT_AVATAR = '/static/img/avatar-user.svg'

const userStore = useUserStore()
const blocks = ref<BlockedUser[]>([])
const loading = ref(true)

onShow(load)

async function load() {
  if (!userStore.isLoggedIn) {
    blocks.value = []
    loading.value = false
    return
  }
  loading.value = true
  try {
    const res = await getBlocks()
    blocks.value = res?.list || []
  } catch {
    blocks.value = []
  } finally {
    loading.value = false
  }
}

function onAvatarError(b: BlockedUser) {
  b.avatar = DEFAULT_AVATAR
}

function onUnblock(b: BlockedUser) {
  uni.showModal({
    title: '解除拉黑',
    content: `确定将「${b.name || '该用户'}」移出黑名单吗？`,
    success: async (res) => {
      if (!res.confirm) return
      try {
        await unblockUser(b.userId)
        uni.showToast({ title: '已解除', icon: 'success' })
        load()
      } catch {
        uni.showToast({ title: '操作失败', icon: 'none' })
      }
    }
  })
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

.header {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 $spacing-md;
  border-bottom: 1rpx solid $dark-border;
}

.back {
  font-size: 44rpx;
  color: $dark-text-primary;
  width: 60rpx;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.content {
  padding: $spacing-lg $spacing-md;
}

.hint {
  display: block;
  text-align: center;
  padding-top: 120rpx;
  color: $dark-text-tertiary;
  font-size: $font-size-sm;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 160rpx;
}

.empty-icon {
  font-size: 80rpx;
  color: $dark-text-tertiary;
}

.empty-text {
  margin-top: $spacing-md;
  font-size: $font-size-md;
  color: $dark-text-secondary;
}

.empty-desc {
  margin-top: $spacing-sm;
  padding: 0 $spacing-xl;
  text-align: center;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  line-height: 1.7;
}

.block-list {
  display: flex;
  flex-direction: column;
}

.block-item {
  display: flex;
  align-items: center;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-md;
  margin-bottom: $spacing-sm;
}

.avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 50%;
  border: 3rpx solid $neon-purple;
}

.block-info {
  flex: 1;
  margin-left: $spacing-md;
  display: flex;
  flex-direction: column;
}

.block-name {
  font-size: $font-size-md;
  color: $dark-text-primary;
}

.block-sub {
  margin-top: 6rpx;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.unblock-btn {
  padding: 10rpx 28rpx;
  border: 2rpx solid $neon-cyan;
  border-radius: $border-radius-sm;
  color: $neon-cyan;
  font-size: $font-size-sm;
  box-shadow: 0 0 10rpx $neon-cyan-glow;
}
</style>
