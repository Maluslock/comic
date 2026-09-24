<template>
  <view class="page">
    <view class="section">
      <view class="section-title">系统通知</view>
      <view class="notification-list">
        <view 
          v-for="item in notifications" 
          :key="item.id" 
          class="notification-item"
          @click="readNotification(item)"
        >
          <view class="notification-icon" :class="item.type">
            {{ item.icon }}
          </view>
          <view class="notification-info">
            <text class="notification-title">{{ item.title }}</text>
            <text class="notification-content">{{ item.content }}</text>
            <text class="notification-time">{{ item.time }}</text>
          </view>
          <view v-if="!item.read" class="unread-dot"></view>
        </view>
      </view>
    </view>

    <view class="section">
      <view class="section-title">聊天会话</view>
      <view class="chat-list">
        <view 
          v-for="session in sessions" 
          :key="session.id" 
          class="chat-item"
          @click="goChat(session)"
        >
          <image :src="session.userAvatar" class="avatar" mode="aspectFill" @error="onSessionAvatarError(session)" />
          <view class="chat-info">
            <view class="chat-header">
              <text class="chat-name">{{ session.userName }}</text>
              <text class="chat-time">{{ session.updatedAt }}</text>
            </view>
            <text class="chat-last">{{ session.lastMessage }}</text>
          </view>
          <view v-if="session.unreadCount > 0" class="unread-count">
            {{ session.unreadCount }}
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet } from '@/api/client'
import { markNotificationRead } from '@/api/index'
import { updateMessageBadge } from '@/utils/badge'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const notifications = ref<any[]>([])

const sessions = ref<any[]>([])

onShow(() => {
  loadSessions()
  loadNotifications()
})

async function loadNotifications() {
  if (!userStore.isLoggedIn || !userStore.user) {
    notifications.value = []
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/notifications/${userStore.user.id}`)
    notifications.value = (res || []).map((n: any) => ({
      id: String(n.id),
      type: n.type,
      icon: notifyIcon(n.type),
      title: n.title,
      content: n.content,
      read: n.read,
      linkType: n.linkType || '',
      linkId: n.linkId != null ? String(n.linkId) : '',
      time: formatNotifyTime(n.createdAt)
    }))
  } catch {
    notifications.value = []
  }
}

function notifyIcon(type: string): string {
  if (type === 'success') return '\u2713'
  if (type === 'warning') return '\u26A0'
  return '\u2139'
}

function formatNotifyTime(iso: string): string {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return ''
    const MM = (d.getMonth() + 1).toString().padStart(2, '0')
    const DD = d.getDate().toString().padStart(2, '0')
    const HH = d.getHours().toString().padStart(2, '0')
    const mm = d.getMinutes().toString().padStart(2, '0')
    return `${MM}/${DD} ${HH}:${mm}`
  } catch { return '' }
}

async function loadSessions() {
  if (!userStore.isLoggedIn || !userStore.user) {
    sessions.value = []
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/chat/sessions/${userStore.user.id}`)
    sessions.value = (res || []).map((s: any) => ({
      id: String(s.id),
      userId: String(s.peerId),
      userName: s.peerName,
      userAvatar: s.peerAvatar,
      lastMessage: s.lastMessage,
      unreadCount: s.unreadCount || 0,
      updatedAt: formatSessionTime(s.lastTime)
    }))
  } catch {
    sessions.value = []
  }
  updateMessageBadge(sessions.value.reduce((n, s) => n + (s.unreadCount || 0), 0))
}

function formatSessionTime(iso: string): string {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return ''
    const MM = (d.getMonth() + 1).toString().padStart(2, '0')
    const DD = d.getDate().toString().padStart(2, '0')
    const HH = d.getHours().toString().padStart(2, '0')
    const mm = d.getMinutes().toString().padStart(2, '0')
    return `${MM}/${DD} ${HH}:${mm}`
  } catch { return '' }
}

async function readNotification(item: any) {
  try {
    await markNotificationRead(item.id)
    item.read = true
  } catch {
    item.read = false
  }
  if (item.linkType === 'order' && item.linkId) {
    uni.navigateTo({ url: `/pages/order/detail?id=${item.linkId}` })
    return
  }
  if (item.linkType === 'photographer_orders') {
    uni.navigateTo({ url: '/pages/photographer/orders' })
    return
  }
  uni.showToast({ title: item.title, icon: 'none' })
}

function onSessionAvatarError(session: any) {
  session.userAvatar = '/static/img/avatar-user.svg'
}

function goChat(session: any) {
  uni.navigateTo({ url: `/pages/chat/index?sessionId=${session.id}&peerName=${encodeURIComponent(session.userName)}&peerAvatar=${encodeURIComponent(session.userAvatar)}` })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding-top: $spacing-md;
}

.section {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  box-shadow: 0 2rpx 16rpx rgba(0, 0, 0, 0.15);
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
  margin-bottom: $spacing-md;
  position: relative;
  padding-left: 20rpx;

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

.notification-list {
  display: flex;
  flex-direction: column;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  padding: $spacing-md 0;

  &:not(:last-child) {
    border-bottom: 1rpx solid $dark-border;
  }

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.notification-icon {
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $font-size-sm;
  flex-shrink: 0;

  &.success {
    @include neon-pill;
  }

  &.info {
    background: rgba($neon-cyan, 0.15);
    color: $neon-cyan;
  }

  &.warning {
    background: rgba($neon-pink, 0.15);
    color: $neon-pink;
  }
}

.notification-info {
  flex: 1;
  margin-left: $spacing-md;
}

.notification-title {
  font-size: $font-size-base;
  font-weight: 500;
  color: $dark-text-primary;
  display: block;
}

.notification-content {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  display: block;
  margin-top: 4rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  display: block;
  margin-top: 4rpx;
}

.unread-dot {
  width: 16rpx;
  height: 16rpx;
  background: $neon-pink;
  border-radius: 50%;
  margin-top: $spacing-sm;
  box-shadow: 0 0 8rpx rgba($neon-pink, 0.5);
}

.chat-list {
  display: flex;
  flex-direction: column;
}

.chat-item {
  display: flex;
  align-items: center;
  padding: $spacing-md 0;

  &:not(:last-child) {
    border-bottom: 1rpx solid $dark-border;
  }

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: 50%;
  border: 2rpx solid $dark-border;
}

.chat-info {
  flex: 1;
  margin-left: $spacing-md;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chat-name {
  font-size: $font-size-base;
  font-weight: 500;
  color: $dark-text-primary;
}

.chat-time {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.chat-last {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-top: 4rpx;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.unread-count {
  background: $neon-pink;
  @include on-neon-fill;
  font-size: $font-size-xs;
  min-width: 36rpx;
  height: 36rpx;
  border-radius: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8rpx;
  box-shadow: 0 0 8rpx rgba($neon-pink, 0.3);
}
</style>
