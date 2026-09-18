<template>
  <view class="page">
    <view class="header">
      <text class="back" @click="goBack">‹</text>
      <text class="header-title">账号与安全</text>
    </view>

    <view class="content">
      <view class="section">
        <view class="section-title">账号信息</view>
        <view class="info-card">
          <view class="info-row">
            <text class="info-label">绑定手机号</text>
            <text class="info-value">{{ maskedPhone }}</text>
          </view>
          <view class="info-row">
            <text class="info-label">昵称</text>
            <text class="info-value">{{ userStore.user?.name || '未设置' }}</text>
          </view>
          <view class="info-row">
            <text class="info-label">账号状态</text>
            <text class="info-value ok">正常</text>
          </view>
        </view>
        <text class="hint">换绑手机号请联系客服 support@mira-comic.example</text>
      </view>

      <view class="section">
        <view class="section-title">登录状态</view>
        <view class="danger-card" @click="onLogoutAll">
          <text class="danger-title">退出所有设备</text>
          <text class="danger-desc">使所有已登录设备立即失效（含当前设备），需要重新登录</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { logoutAllDevices } from '@/api/index'

const userStore = useUserStore()

const maskedPhone = computed(() => {
  const p = userStore.user?.phone || ''
  if (p.length < 7) return p || '未绑定'
  return `${p.slice(0, 3)}****${p.slice(-4)}`
})

function goBack() {
  uni.navigateBack()
}

function onLogoutAll() {
  if (!userStore.isLoggedIn) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.showModal({
    title: '退出所有设备',
    content: '所有设备（含当前设备）都将退出登录，确定继续吗？',
    success: async (res) => {
      if (!res.confirm) return
      try {
        await logoutAllDevices()
      } catch {
        uni.showToast({ title: '操作失败，请稍后重试', icon: 'none' })
        return
      }
      userStore.logout()
      uni.showToast({ title: '已退出所有设备', icon: 'success' })
      setTimeout(() => uni.reLaunch({ url: '/pages/login/index' }), 800)
    }
  })
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

.section {
  margin-bottom: $spacing-xl;
}

.section-title {
  display: block;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  margin-bottom: $spacing-sm;
}

.info-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-md;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-sm 0;

  & + & {
    border-top: 1rpx solid $dark-border;
  }
}

.info-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.info-value {
  font-size: $font-size-sm;
  color: $dark-text-primary;

  &.ok {
    color: $neon-cyan;
  }
}

.hint {
  display: block;
  margin-top: $spacing-sm;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.danger-card {
  background: $dark-bg-card;
  border: 1rpx solid rgba(239, 68, 68, 0.35);
  border-radius: $border-radius-md;
  padding: $spacing-md;

  &:active {
    background: rgba(239, 68, 68, 0.08);
  }
}

.danger-title {
  display: block;
  font-size: $font-size-md;
  color: $error-color;
  font-weight: 600;
}

.danger-desc {
  display: block;
  margin-top: 8rpx;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  line-height: 1.6;
}
</style>
