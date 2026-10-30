<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }"></view>
    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <text class="header-title">设置</text>
      <view class="placeholder"></view>
    </view>

    <view class="content">
      <!-- 个人资料 -->
      <view class="section">
        <view class="section-title">个人资料</view>
        <view class="menu-group">
          <view class="profile-item" @click="goProfileEdit">
            <view class="profile-left">
              <image class="avatar" :src="user.avatar" mode="aspectFill" />
              <view class="profile-meta">
                <text class="profile-name">{{ user.name }}</text>
                <text class="profile-hint">点击编辑个人资料</text>
              </view>
            </view>
            <view class="edit-btn">编辑</view>
          </view>
        </view>
      </view>

      <!-- 通知设置 -->
      <view class="section">
        <view class="section-title">通知设置</view>
        <view class="menu-group">
          <view
            v-for="item in notificationList"
            :key="item.key"
            class="menu-item"
          >
            <view class="item-accent"></view>
            <text class="menu-text">{{ item.label }}</text>
            <switch
              class="menu-switch"
              :checked="notifications[item.key]"
              :color="SWITCH_COLOR"
              @change="handleNotificationChange(item.key, $event)"
            />
          </view>
        </view>
      </view>

      <!-- 隐私设置 -->
      <view class="section">
        <view class="section-title">隐私设置</view>
        <view class="menu-group">
          <view
            v-for="item in privacyItems"
            :key="item.label"
            class="menu-item"
            @click="item.action"
          >
            <view class="item-accent"></view>
            <text class="menu-text">{{ item.label }}</text>
            <text class="menu-arrow">›</text>
          </view>
        </view>
      </view>

      <!-- 关于我们 -->
      <view class="section">
        <view class="section-title">关于我们</view>
        <view class="menu-group">
          <view
            v-for="item in aboutItems"
            :key="item.label"
            class="menu-item"
            @click="item.action"
          >
            <view class="item-accent"></view>
            <text class="menu-text">{{ item.label }}</text>
            <text class="menu-arrow">›</text>
          </view>
        </view>
      </view>

      <!-- 退出登录 -->
      <view class="logout-section">
        <view class="logout-btn" @click="logout">
          <text class="logout-text">退出登录</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useUserStore } from '@/stores/user'
import { useStatusBar } from '@/composables/useStatusBar'

const { statusBarHeight } = useStatusBar()

const SWITCH_COLOR = '#a855f7'

type NotificationKey = 'message' | 'event' | 'marketing'

const userStore = useUserStore()
const user = computed(() => ({
  name: userStore.user?.name || '未登录',
  avatar: userStore.user?.avatar || '/static/img/avatar-user.svg'
}))

const notifications = ref<Record<NotificationKey, boolean>>({
  message: true,
  event: true,
  marketing: false
})

function loadNotificationPrefs() {
  const stored = uni.getStorageSync('notificationPrefs')
  if (stored) {
    try {
      notifications.value = { ...notifications.value, ...JSON.parse(stored) }
    } catch { /* ignore malformed */ }
  }
}

function saveNotificationPrefs() {
  uni.setStorageSync('notificationPrefs', JSON.stringify(notifications.value))
}

function goBack() {
  uni.navigateBack()
}

const notificationList: { key: NotificationKey; label: string }[] = [
  { key: 'message', label: '消息通知' },
  { key: 'event', label: '漫展提醒' },
  { key: 'marketing', label: '活动推荐' }
]

onShow(() => {
  loadNotificationPrefs()
})

function handleNotificationChange(key: NotificationKey, e: Event) {
  const detail = (e as unknown as { detail: { value: boolean } }).detail
  notifications.value[key] = detail.value
  saveNotificationPrefs()
}

const privacyItems: { label: string; action: () => void }[] = [
  {
    label: '隐私政策',
    action: () => openDoc('privacy')
  },
  {
    label: '账号与安全',
    action: () => openDoc('security')
  },
  {
    label: '黑名单管理',
    action: () => openDoc('blocks')
  }
]

const aboutItems: { label: string; action: () => void }[] = [
  {
    label: '关于米拉漫展',
    action: () => openDoc('about')
  },
  {
    label: '用户协议',
    action: () => openDoc('terms')
  },
  {
    label: '版本信息',
    action: () => uni.showToast({ title: '版本 1.0.0', icon: 'none' })
  }
]

function openDoc(type: 'privacy' | 'terms' | 'about' | 'help' | 'security' | 'blocks') {
  if (type === 'security' || type === 'blocks') {
    if (!userStore.isLoggedIn) {
      uni.navigateTo({ url: '/pages/login/index' })
      return
    }
    uni.navigateTo({ url: `/pages/settings/${type}` })
    return
  }
  uni.navigateTo({ url: `/pages/settings/doc?type=${type}` })
}

function goProfileEdit() {
  if (!userStore.isLoggedIn) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.navigateTo({ url: '/pages/profile/edit' })
}

function logout() {
  if (!userStore.isLoggedIn) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.showModal({
    title: '确认退出',
    content: '确定要退出登录吗？',
    success: (res) => {
      if (res.confirm) {
        userStore.logout()
        uni.showToast({ title: '已退出登录', icon: 'success' })
        setTimeout(() => uni.navigateBack(), 800)
      }
    }
  })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.status-bar {
  background: #7c3aed;
}

/* 品牌感保留：状态栏与页头连成一条紫色渐变带，再向页面底色收尾。
   高度仍是 44px，不把内容顶下去，也不吃回「大橙头」那套高度。 */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  box-sizing: border-box;
  padding: 0 $spacing-md;
  background: linear-gradient(180deg, #7c3aed 0%, #3b2456 58%, $dark-bg-secondary 100%);
  border-bottom: 1rpx solid $dark-border;
}

.back-btn {
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.back-icon {
  font-size: 48rpx;
  color: $dark-text-primary;
}

.placeholder {
  width: 60rpx;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.content {
  padding: $spacing-md;
}

.section {
  margin-bottom: $spacing-lg;
}

.section-title {
  position: relative;
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
  padding-left: 20rpx;
  margin-bottom: $spacing-md;

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

.menu-group {
  background: $dark-bg-card;
  border-radius: $border-radius-lg;
  overflow: hidden;
  box-shadow: $shadow-md;
  border: 1rpx solid $dark-border;
}

.profile-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-md;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.profile-left {
  display: flex;
  align-items: center;
}

.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 50%;
  border: 3rpx solid $neon-purple;
}

.profile-meta {
  margin-left: $spacing-md;
  display: flex;
  flex-direction: column;
}

.profile-name {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.profile-hint {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  margin-top: 4rpx;
}

.edit-btn {
  padding: $spacing-xs $spacing-md;
  border: 2rpx solid $neon-purple;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  color: $neon-purple;

  &:active {
    background: $neon-purple-dim;
  }
}

.menu-item {
  display: flex;
  align-items: center;
  padding: $spacing-md;

  &:not(:last-child) {
    border-bottom: 1rpx solid $dark-border;
  }

  &:active {
    background: $dark-bg-card-hover;
  }
}

.item-accent {
  width: 6rpx;
  height: 28rpx;
  background: $neon-gradient;
  border-radius: 3rpx;
  margin-right: $spacing-md;
  flex-shrink: 0;
}

.menu-text {
  flex: 1;
  font-size: $font-size-base;
  color: $dark-text-primary;
}

.menu-arrow {
  font-size: $font-size-lg;
  color: $dark-text-tertiary;
}

.menu-switch {
  transform: scale(0.8);
  transform-origin: right center;
}

.logout-section {
  margin-top: $spacing-lg;
  margin-bottom: $spacing-xl;
}

.logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid rgba($neon-pink, 0.3);
  border-radius: $border-radius-lg;
  box-shadow: $shadow-md;

  &:active {
    background: rgba($neon-pink, 0.1);
  }
}

.logout-text {
  font-size: $font-size-base;
  font-weight: 500;
  color: $neon-pink;
}
</style>
