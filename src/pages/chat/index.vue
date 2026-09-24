<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }"></view>

    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <view class="user-info">
        <image :src="chatUser.avatar" class="avatar" mode="aspectFill" @error="onPeerAvatarError" />
        <text class="name">{{ chatUser.name }}</text>
      </view>
      <view class="placeholder"></view>
    </view>

    <scroll-view
      scroll-y
      class="chat-content"
      :scroll-into-view="scrollToId"
      :scroll-with-animation="true"
    >
      <view v-for="msg in messages" :key="msg.id" :id="'msg-' + msg.id" class="msg-item" :class="{ 'is-me': msg.isMe }">
        <image :src="msg.isMe ? myAvatar : chatUser.avatar" class="msg-avatar" mode="aspectFill" @error="onPeerAvatarError" />
        <view class="msg-wrapper">
          <view class="msg-bubble" :class="{ 'is-me': msg.isMe }">
            <text>{{ msg.content }}</text>
          </view>
          <text class="msg-time">{{ msg.time }}</text>
        </view>
      </view>

      <view v-if="typing" class="typing-indicator">
        <image :src="chatUser.avatar" class="msg-avatar" mode="aspectFill" @error="onPeerAvatarError" />
        <view class="typing-bubble">
          <view class="typing-dots">
            <view class="typing-dot"></view>
            <view class="typing-dot"></view>
            <view class="typing-dot"></view>
          </view>
        </view>
      </view>

      <view id="msg-bottom" class="scroll-anchor"></view>
    </scroll-view>

    <view class="input-bar">
      <input
        class="msg-input"
        v-model="inputValue"
        placeholder="输入消息..."
        placeholder-class="msg-input-placeholder"
        :disabled="typing"
        @confirm="sendMessage"
      />
      <view class="send-btn" :class="{ disabled: typing || !inputValue.trim() }" @click="sendMessage">
        <text>发送</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { onShow, onHide, onUnload } from '@dcloudio/uni-app'
import { apiGet, apiPost } from '@/api/client'
import { markChatSessionRead } from '@/api/index'
import { refreshMessageBadge } from '@/utils/badge'
import { useUserStore } from '@/stores/user'

interface ChatMessage {
  id: string
  content: string
  isMe: boolean
  time: string
}

const statusBarHeight = ref(44)
const inputValue = ref('')
const scrollToId = ref('')
const typing = ref(false)
const sessionId = ref('')
const loading = ref(true)

const myAvatar = '/static/img/avatar-coser-chat.jpg'
const userStore = useUserStore()

const chatUser = ref({
  name: '摄影师',
  avatar: '/static/img/avatar-photographer-chat.jpg'
})

const DEFAULT_AVATAR = '/static/img/avatar-user.svg'

function onPeerAvatarError() {
  chatUser.value.avatar = DEFAULT_AVATAR
}

const messages = ref<ChatMessage[]>([])

function formatTime(): string {
  const now = new Date()
  const h = now.getHours().toString().padStart(2, '0')
  const m = now.getMinutes().toString().padStart(2, '0')
  return `${h}:${m}`
}

let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    if (sessionId.value) loadMessages()
  }, 5000)
}

async function loadMessages() {
  if (!sessionId.value) return
  try {
    const res = await apiGet<any[]>(`/v1/chat/messages/${sessionId.value}`)
    const list = (res || []).map((m: any) => ({
      id: String(m.id),
      content: m.content,
      isMe: String(m.senderId) === String(userStore.user?.id),
      time: formatTimeFromCreated(m.createdAt)
    }))
    const changed = list.length !== messages.value.length
    messages.value = list
    if (changed) {
      await markChatSessionRead(sessionId.value).catch(() => undefined)
      await refreshMessageBadge()
      scrollToBottom()
    }
  } catch {
    if (messages.value.length === 0) messages.value = []
  }
}

onMounted(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  sessionId.value = currentPage?.options?.sessionId as string || ''
  const peerName = currentPage?.options?.peerName as string
  const peerAvatar = currentPage?.options?.peerAvatar as string
  if (peerName) chatUser.value.name = decodeURIComponent(peerName)
  if (peerAvatar) chatUser.value.avatar = decodeURIComponent(peerAvatar)

  if (!sessionId.value) {
    loading.value = false
    uni.showToast({ title: '会话不存在', icon: 'none' })
    return
  }
  await loadMessages()
  loading.value = false
  scrollToBottom()
  startPolling()
})

onShow(() => {
  if (!sessionId.value) return
  loadMessages()
  startPolling()
})

onHide(stopPolling)
onUnload(stopPolling)

function formatTimeFromCreated(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
  } catch { return '' }
}

function scrollToBottom() {
  nextTick(() => { scrollToId.value = 'msg-bottom' })
}

async function sendMessage() {
  if (!inputValue.value.trim() || typing.value) return
  const content = inputValue.value.trim()
  if (!sessionId.value || !userStore.user) {
    uni.showToast({ title: '无法发送', icon: 'none' })
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/messages', {
      sessionId: Number(sessionId.value),
      senderId: Number(userStore.user.id),
      content
    })
    messages.value.push({ id: String(res.id), content, isMe: true, time: formatTime() })
    inputValue.value = ''
    scrollToBottom()
  } catch {
    uni.showToast({ title: '发送失败', icon: 'none' })
  }
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-secondary;
  border-bottom: 1rpx solid $dark-border;
}

.back-btn {
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.back-icon {
  font-size: 48rpx;
  color: $dark-text-primary;
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
  color: $dark-text-primary;
}

.placeholder {
  width: 60rpx;
}

.chat-content {
  flex: 1;
  box-sizing: border-box;
  padding: $spacing-md;
}

.scroll-anchor {
  height: 0;
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

.msg-wrapper {
  display: flex;
  flex-direction: column;
  max-width: 70%;

  .is-me & {
    align-items: flex-end;
    margin-right: $spacing-sm;
  }

  .msg-item:not(.is-me) & {
    margin-left: $spacing-sm;
  }
}

.msg-bubble {
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
  line-height: 1.5;
  color: $dark-text-primary;
  word-break: break-all;

  &:not(.is-me) {
    background: $dark-bg-card;
    border-top-left-radius: 4rpx;
  }

  &.is-me {
    background: $neon-purple;
    color: #fff;
    border-top-right-radius: 4rpx;
    box-shadow: 0 0 16rpx $neon-purple-glow;
  }
}

.msg-time {
  font-size: 20rpx;
  color: $dark-text-tertiary;
  margin-top: 6rpx;
}

.typing-indicator {
  display: flex;
  align-items: flex-end;
  margin-bottom: $spacing-md;
}

.typing-bubble {
  margin-left: $spacing-sm;
  padding: $spacing-sm $spacing-lg;
  background: $dark-bg-card;
  border-radius: $border-radius-lg;
  border-top-left-radius: 4rpx;
}

.typing-dots {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.typing-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: $dark-text-tertiary;
  animation: typing-bounce 1.4s infinite ease-in-out both;

  &:nth-child(1) {
    animation-delay: 0s;
  }

  &:nth-child(2) {
    animation-delay: 0.2s;
  }

  &:nth-child(3) {
    animation-delay: 0.4s;
  }
}

@keyframes typing-bounce {
  0%, 80%, 100% {
    transform: scale(0.6);
    opacity: 0.4;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.input-bar {
  display: flex;
  align-items: center;
  padding: $spacing-sm $spacing-md;
  padding-bottom: calc(#{$spacing-sm} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
}

.msg-input {
  flex: 1;
  background: $dark-bg-secondary;
  border-radius: $border-radius-lg;
  padding: $spacing-sm $spacing-md;
  font-size: $font-size-base;
  color: $dark-text-primary;

  &-placeholder {
    color: $dark-text-tertiary;
  }
}

.send-btn {
  padding: $spacing-sm $spacing-lg;
  background: $neon-gradient;
  @include on-neon-fill;
  border-radius: $border-radius-lg;
  margin-left: $spacing-md;
  font-size: $font-size-base;
  font-weight: 500;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }

  &.disabled {
    opacity: 0.5;
    pointer-events: none;
  }
}
</style>
