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
          <image :src="session.userAvatar" class="avatar" mode="aspectFill" />
          <view class="chat-info">
            <view class="chat-header">
              <text class="chat-name">{{ session.userName }}</text>
              <text class="chat-time">{{ session.time }}</text>
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
import { ref, onMounted } from 'vue'

const notifications = ref<any[]>([
  {
    id: 'n1',
    type: 'success',
    icon: '✓',
    title: '预约成功',
    content: '您预约的「光影行者」拍摄服务已确认',
    time: '今天 14:30',
    read: false
  },
  {
    id: 'n2',
    type: 'info',
    icon: 'ℹ',
    title: '新消息',
    content: '摄影师「古风公子」给您发来了消息',
    time: '今天 10:20',
    read: true
  },
  {
    id: 'n3',
    type: 'warning',
    icon: '⚠',
    title: '订单提醒',
    content: '您的订单「BK20240101002」即将开始',
    time: '昨天 18:00',
    read: true
  }
])

const sessions = ref<any[]>([
  {
    id: 's1',
    userName: '光影行者',
    userAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=professional%20photographer%20avatar%20portrait%20studio%20lighting&image_size=square',
    lastMessage: '好的，那我帮您确认一下档期',
    time: '14:30',
    unreadCount: 2
  },
  {
    id: 's2',
    userName: '古风公子',
    userAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=chinese%20ancient%20style%20photographer%20avatar%20elegant&image_size=square',
    lastMessage: '您看这个时间可以吗？',
    time: '昨天',
    unreadCount: 0
  },
  {
    id: 's3',
    userName: '樱花落',
    userAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=female%20photographer%20avatar%20pink%20hair%20cute%20style&image_size=square',
    lastMessage: '收到，期待合作~',
    time: '周一',
    unreadCount: 0
  }
])

onMounted(() => {
})

function readNotification(item: any) {
  item.read = true
  uni.showToast({ title: item.title, icon: 'none' })
}

function goChat(session: any) {
  session.unreadCount = 0
  uni.navigateTo({ url: '/pages/chat/index' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
  padding-top: env(safe-area-inset-top);
}

.section {
  background: $bg-primary;
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
  margin-bottom: $spacing-md;
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
    border-bottom: 1rpx solid $border-color;
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
    background: rgba($success-color, 0.1);
    color: $success-color;
  }
  
  &.info {
    background: rgba($info-color, 0.1);
    color: $info-color;
  }
  
  &.warning {
    background: rgba($warning-color, 0.1);
    color: $warning-color;
  }
}

.notification-info {
  flex: 1;
  margin-left: $spacing-md;
}

.notification-title {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
}

.notification-content {
  font-size: $font-size-sm;
  color: $text-secondary;
  display: block;
  margin-top: 4rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  font-size: $font-size-xs;
  color: $text-tertiary;
  display: block;
  margin-top: 4rpx;
}

.unread-dot {
  width: 16rpx;
  height: 16rpx;
  background: $error-color;
  border-radius: 50%;
  margin-top: $spacing-sm;
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
    border-bottom: 1rpx solid $border-color;
  }
}

.avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: 50%;
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
}

.chat-time {
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.chat-last {
  font-size: $font-size-sm;
  color: $text-secondary;
  margin-top: 4rpx;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.unread-count {
  background: $error-color;
  color: #fff;
  font-size: $font-size-xs;
  min-width: 36rpx;
  height: 36rpx;
  border-radius: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8rpx;
}
</style>
