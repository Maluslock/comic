<template>
  <view class="page">
    <view class="header">
      <view class="user-card">
        <image :src="user.avatar" class="avatar" mode="aspectFill" />
        <view class="user-info">
          <text class="user-name">{{ user.name }}</text>
          <view class="user-role">
            <text class="role-tag" :class="user.role">
              {{ user.role === 'photographer' ? '摄影师' : 'Coser' }}
            </text>
          </view>
          <text class="user-desc">{{ user.description }}</text>
        </view>
      </view>

      <view class="stats-row">
        <view class="stat-item">
          <text class="stat-value">{{ user.followers }}</text>
          <text class="stat-label">粉丝</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ user.following }}</text>
          <text class="stat-label">关注</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ user.works }}</text>
          <text class="stat-label">作品</text>
        </view>
      </view>

      <view class="role-switch">
        <text class="switch-label">切换身份：</text>
        <view class="switch-group">
          <view 
            class="switch-item" 
            :class="{ active: user.role === 'coser' }"
            @click="switchRole('coser')"
          >
            Coser
          </view>
          <view 
            class="switch-item" 
            :class="{ active: user.role === 'photographer' }"
            @click="switchRole('photographer')"
          >
            摄影师
          </view>
        </view>
      </view>
    </view>

    <view class="menu-section">
      <view class="menu-group">
        <view class="menu-item" @click="goOrders">
          <text class="menu-icon">📋</text>
          <text class="menu-text">我的订单</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goFavorites">
          <text class="menu-icon">❤️</text>
          <text class="menu-text">我的收藏</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goWorks">
          <text class="menu-icon">📷</text>
          <text class="menu-text">我的作品</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goReviews">
          <text class="menu-icon">✍️</text>
          <text class="menu-text">我的评价</text>
          <text class="menu-arrow">›</text>
        </view>
      </view>

      <view class="menu-group">
        <view class="menu-item" @click="goSettings">
          <text class="menu-icon">⚙️</text>
          <text class="menu-text">设置</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goHelp">
          <text class="menu-icon">❓</text>
          <text class="menu-text">帮助与反馈</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="logout">
          <text class="menu-icon">🚪</text>
          <text class="menu-text">退出登录</text>
          <text class="menu-arrow">›</text>
        </view>
      </view>
    </view>

    <view class="bottom-info">
      <text class="version">版本 1.0.0</text>
      <text class="copyright">© 2024 漫展摄影平台</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const user = ref({
  name: '小狐狸',
  avatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=anime%20girl%20avatar%20fox%20ears%20cute&image_size=square',
  role: 'coser' as 'coser' | 'photographer',
  description: '热爱二次元，喜欢cosplay',
  followers: 128,
  following: 64,
  works: 24
})

function switchRole(role: 'coser' | 'photographer') {
  user.value.role = role
  uni.showToast({ 
    title: `已切换为${role === 'photographer' ? '摄影师' : 'Coser'}`, 
    icon: 'none' 
  })
}

function goOrders() {
  uni.navigateTo({ url: '/pages/order/list' })
}

function goFavorites() {
  uni.showToast({ title: '我的收藏', icon: 'none' })
}

function goWorks() {
  uni.navigateTo({ url: '/pages/portfolio/index' })
}

function goReviews() {
  uni.navigateTo({ url: '/pages/comment/index' })
}

function goSettings() {
  uni.showToast({ title: '设置', icon: 'none' })
}

function goHelp() {
  uni.showToast({ title: '帮助与反馈', icon: 'none' })
}

function logout() {
  uni.showModal({
    title: '确认退出',
    content: '确定要退出登录吗？',
    success: (res) => {
      if (res.confirm) {
        uni.showToast({ title: '已退出', icon: 'success' })
      }
    }
  })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.header {
  background: linear-gradient(135deg, $primary-color 0%, $primary-light 100%);
  padding: $spacing-lg $spacing-md;
  padding-top: calc(env(safe-area-inset-top) + #{$spacing-lg});
}

.user-card {
  display: flex;
  align-items: flex-start;
}

.avatar {
  width: 160rpx;
  height: 160rpx;
  border-radius: 50%;
  border: 4rpx solid rgba(255, 255, 255, 0.3);
}

.user-info {
  flex: 1;
  margin-left: $spacing-md;
}

.user-name {
  font-size: $font-size-xl;
  font-weight: 600;
  color: #fff;
}

.user-role {
  margin-top: $spacing-xs;
}

.role-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  font-size: $font-size-xs;
  
  &.coser {
    background: rgba(255, 255, 255, 0.2);
    color: #fff;
  }
  
  &.photographer {
    background: rgba(255, 215, 0, 0.3);
    color: #fff;
  }
}

.user-desc {
  font-size: $font-size-sm;
  color: rgba(255, 255, 255, 0.8);
  margin-top: $spacing-xs;
  display: block;
}

.stats-row {
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: rgba(255, 255, 255, 0.1);
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  margin-top: $spacing-lg;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: $font-size-xl;
  font-weight: 600;
  color: #fff;
}

.stat-label {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
  margin-top: 4rpx;
}

.stat-divider {
  width: 2rpx;
  height: 48rpx;
  background: rgba(255, 255, 255, 0.2);
}

.role-switch {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: $spacing-lg;
}

.switch-label {
  font-size: $font-size-sm;
  color: rgba(255, 255, 255, 0.8);
}

.switch-group {
  display: flex;
  margin-left: $spacing-sm;
  background: rgba(255, 255, 255, 0.2);
  border-radius: $border-radius-lg;
  padding: 4rpx;
}

.switch-item {
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: rgba(255, 255, 255, 0.8);
  
  &.active {
    background: #fff;
    color: $primary-color;
    font-weight: 500;
  }
}

.menu-section {
  padding: $spacing-md;
}

.menu-group {
  background: $bg-primary;
  border-radius: $border-radius-lg;
  overflow: hidden;
  margin-bottom: $spacing-md;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: $spacing-md;
  
  &:not(:last-child) {
    border-bottom: 1rpx solid $border-color;
  }
  
  &:active {
    background: $bg-secondary;
  }
}

.menu-icon {
  font-size: $font-size-lg;
  margin-right: $spacing-md;
}

.menu-text {
  flex: 1;
  font-size: $font-size-base;
}

.menu-arrow {
  font-size: $font-size-lg;
  color: $text-tertiary;
}

.bottom-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl;
}

.version {
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.copyright {
  font-size: $font-size-xs;
  color: $text-tertiary;
  margin-top: $spacing-xs;
}
</style>
