<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }"></view>
    
    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <view class="user-info">
        <image :src="chatUser.avatar" class="avatar" mode="aspectFill" />
        <text class="name">{{ chatUser.name }}</text>
      </view>
      <view class="placeholder"></view>
    </view>

    <scroll-view 
      scroll-y 
      class="chat-content"
      :scroll-into-view="scrollToId"
      scroll-with-animation
    >
      <view v-for="msg in messages" :key="msg.id" :id="'msg-' + msg.id" class="msg-item">
        <image :src="msg.isMe ? myAvatar : chatUser.avatar" class="msg-avatar" mode="aspectFill" />
        <view class="msg-bubble" :class="{ 'is-me': msg.isMe }">
          <text>{{ msg.content }}</text>
        </view>
      </view>
    </scroll-view>

    <view class="input-bar">
      <input 
        class="msg-input" 
        v-model="inputValue"
        placeholder="输入消息..."
        @confirm="sendMessage"
      />
      <view class="send-btn" @click="sendMessage">
        <text>发送</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'

const statusBarHeight = ref(44)
const inputValue = ref('')
const scrollToId = ref('')

const myAvatar = 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=anime%20girl%20avatar%20cute%20style&image_size=square'

const chatUser = ref({
  name: '光影行者',
  avatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=professional%20photographer%20avatar%20portrait%20studio%20lighting&image_size=square'
})

const messages = ref<any[]>([
  { id: '1', content: '您好，请问想约什么时间拍摄呢？', isMe: false },
  { id: '2', content: '您好！我想约下周六下午可以吗？', isMe: true },
  { id: '3', content: '下周六下午可以的，您想约哪个套餐呢？', isMe: false },
  { id: '4', content: '进阶套餐就可以', isMe: true },
  { id: '5', content: '好的，那我帮您确认一下档期，稍后给您回复~', isMe: false }
])

onMounted(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  scrollToBottom()
})

function scrollToBottom() {
  nextTick(() => {
    if (messages.value.length > 0) {
      scrollToId.value = 'msg-' + messages.value[messages.value.length - 1].id
    }
  })
}

function sendMessage() {
  if (!inputValue.value.trim()) return
  
  const newMsg = {
    id: Date.now().toString(),
    content: inputValue.value,
    isMe: true
  }
  
  messages.value.push(newMsg)
  inputValue.value = ''
  scrollToBottom()
  
  setTimeout(() => {
    const replyMsg = {
      id: (Date.now() + 1).toString(),
      content: '收到，我这边记下了~',
      isMe: false
    }
    messages.value.push(replyMsg)
    scrollToBottom()
  }, 1000)
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-sm $spacing-md;
  background: $bg-primary;
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
  color: $text-primary;
}

.user-info {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar {
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
}

.name {
  font-size: $font-size-lg;
  font-weight: 500;
  margin-left: $spacing-sm;
}

.placeholder {
  width: 60rpx;
}

.chat-content {
  flex: 1;
  padding: $spacing-md;
}

.msg-item {
  display: flex;
  margin-bottom: $spacing-md;
  
  &.is-me {
    flex-direction: row-reverse;
  }
}

.msg-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 50%;
  flex-shrink: 0;
}

.msg-bubble {
  max-width: 70%;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
  line-height: 1.5;
  
  &:not(.is-me) {
    background: $bg-primary;
    margin-left: $spacing-sm;
    border-top-left-radius: 4rpx;
  }
  
  &.is-me {
    background: $primary-color;
    color: #fff;
    margin-right: $spacing-sm;
    border-top-right-radius: 4rpx;
  }
}

.input-bar {
  display: flex;
  align-items: center;
  padding: $spacing-sm $spacing-md;
  padding-bottom: calc(#{$spacing-sm} + env(safe-area-inset-bottom));
  background: $bg-primary;
}

.msg-input {
  flex: 1;
  background: $bg-tertiary;
  border-radius: $border-radius-lg;
  padding: $spacing-sm $spacing-md;
  font-size: $font-size-base;
}

.send-btn {
  padding: $spacing-sm $spacing-lg;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  margin-left: $spacing-md;
  font-size: $font-size-base;
}
</style>
