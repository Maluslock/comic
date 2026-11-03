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
        <image :src="msg.isMe ? myAvatar : chatUser.avatar" class="msg-avatar" mode="aspectFill" @error="msg.isMe ? onMyAvatarError() : onPeerAvatarError()" />
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

    <view class="input-bar" :style="inputBarStyle">
      <input
        class="msg-input"
        v-model="inputValue"
        placeholder="输入消息..."
        placeholder-class="msg-input-placeholder"
        :disabled="typing"
        :adjust-position="false"
        :cursor-spacing="16"
        @confirm="sendMessage"
      />
      <view class="send-btn" :class="{ disabled: typing || !inputValue.trim() }" @click="sendMessage">
        <text>发送</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { onShow, onHide, onUnload } from '@dcloudio/uni-app'
import { apiGet, apiPost } from '@/api/client'
import { markChatSessionRead } from '@/api/index'
import { refreshMessageBadge } from '@/utils/badge'
import { useUserStore } from '@/stores/user'
import { resolveImage } from '@/utils/mappers'

interface ChatMessage {
  id: string
  content: string
  isMe: boolean
  time: string
}

const { statusBarHeight } = useStatusBar()
const inputValue = ref('')
const scrollToId = ref('')
const typing = ref(false)
const sessionId = ref('')
const loading = ref(true)
const keyboardHeight = ref(0)

// 输入栏基础内边距（rpx）。必须换算成 px 再进 calc：H5 的 calc() 不认 rpx，
// `calc(12rpx + 10px)` 会被整条判为无效声明并回落 0，键盘弹起时反而丢了下边距。
const INPUT_BAR_PAD_RPX = 12

function rpxToPx(rpx: number): number {
  try {
    const fn = (uni as unknown as { upx2px?: (v: number) => number }).upx2px
    if (typeof fn === 'function') return fn(rpx)
  } catch {
    /* 忽略：降级到 375pt 目测换算 */
  }
  return Math.round(rpx * 0.5)
}

const inputBarStyle = computed(() => {
  const base = `calc(${rpxToPx(INPUT_BAR_PAD_RPX)}px + env(safe-area-inset-bottom)`
  return {
    paddingBottom: keyboardHeight.value > 0 ? `${base} + ${keyboardHeight.value}px)` : `${base})`
  }
})

let onKeyboardChangeHandler: ((res: { height: number }) => void) | null = null

// H5 没有 uni.onKeyboardHeightChange（uni-h5 未实现该 API），键盘弹起时
// 只有 visualViewport 会变矮。不补这一段，H5 的键盘避让就是空转。
//
// 但 visualViewport 变矮也可能是浏览器工具栏收缩 / 窗口尺寸变化，不能一律当键盘。
// 判据：①当前有可编辑元素获得焦点；②收缩量 > 120px（键盘最低也在 200px 上下）。
// 两条不满足就当作布局变化，重设基准高度，避免把窗口缩放误算成 143px 键盘。
let h5ViewportHandler: (() => void) | null = null
let h5WindowHandler: (() => void) | null = null
let h5BaseHeight = 0

function hasFocusedEditable(): boolean {
  if (typeof document === 'undefined') return false
  const el = document.activeElement as HTMLElement | null
  if (!el) return false
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || el.isContentEditable
}

function bindH5KeyboardFallback() {
  if (typeof window === 'undefined') return
  const viewport = window.visualViewport
  if (!viewport || h5ViewportHandler) return
  // 基准取 visualViewport 自身高度：无键盘时它才是内容真正可用高度，
  // 用 innerHeight 当基准会把浏览器工具栏高度算成键盘。
  const resetBase = () => {
    keyboardHeight.value = 0
    h5BaseHeight = viewport.height
  }
  resetBase()

  h5ViewportHandler = () => {
    const shrink = Math.round(h5BaseHeight - viewport.height)
    if (shrink > 120 && hasFocusedEditable()) {
      keyboardHeight.value = shrink
      scrollToBottom()
      return
    }
    if (shrink <= 0 || !hasFocusedEditable()) resetBase()
  }
  viewport.addEventListener('resize', h5ViewportHandler)

  // 窗口尺寸变化时重置基准，否则后面的键盘高度会算进这段差值
  h5WindowHandler = () => {
    if (hasFocusedEditable()) return
    resetBase()
  }
  window.addEventListener('resize', h5WindowHandler)
}

function unbindH5KeyboardFallback() {
  if (typeof window === 'undefined') return
  const viewport = window.visualViewport
  if (viewport && h5ViewportHandler) viewport.removeEventListener('resize', h5ViewportHandler)
  if (h5WindowHandler) window.removeEventListener('resize', h5WindowHandler)
  h5ViewportHandler = null
  h5WindowHandler = null
  h5BaseHeight = 0
}

const DEFAULT_AVATAR = '/static/img/avatar-user.svg'
const myAvatarFallback = ref(false)
const userStore = useUserStore()

// 「我」的头像与「我的」页面同源：userStore.user.avatar，为空/加载失败时降级默认头像
const myAvatar = computed(() =>
  myAvatarFallback.value ? DEFAULT_AVATAR : resolveImage(userStore.user?.avatar || DEFAULT_AVATAR)
)

const chatUser = ref({
  name: '摄影师',
  avatar: resolveImage('/static/img/avatar-photographer-chat.jpg')
})

function onMyAvatarError() {
  myAvatarFallback.value = true
}

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

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  sessionId.value = currentPage?.options?.sessionId as string || ''
  const peerName = currentPage?.options?.peerName as string
  const peerAvatar = currentPage?.options?.peerAvatar as string
  if (peerName) chatUser.value.name = decodeURIComponent(peerName)
  if (peerAvatar) chatUser.value.avatar = resolveImage(decodeURIComponent(peerAvatar))

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
  if (typeof uni.onKeyboardHeightChange === 'function' && !onKeyboardChangeHandler) {
    onKeyboardChangeHandler = (res: { height: number }) => {
      keyboardHeight.value = res.height || 0
      if (res.height > 0) {
        scrollToBottom()
      }
    }
    uni.onKeyboardHeightChange(onKeyboardChangeHandler)
  }
  bindH5KeyboardFallback()
  if (!sessionId.value) return
  loadMessages()
  startPolling()
})

onHide(() => {
  stopPolling()
  if (typeof uni.offKeyboardHeightChange === 'function' && onKeyboardChangeHandler) {
    uni.offKeyboardHeightChange(onKeyboardChangeHandler)
    onKeyboardChangeHandler = null
  }
  unbindH5KeyboardFallback()
  keyboardHeight.value = 0
})

onUnload(() => {
  stopPolling()
  if (typeof uni.offKeyboardHeightChange === 'function' && onKeyboardChangeHandler) {
    uni.offKeyboardHeightChange(onKeyboardChangeHandler)
    onKeyboardChangeHandler = null
  }
  unbindH5KeyboardFallback()
  keyboardHeight.value = 0
})

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
  height: 100vh;
  overflow: hidden;
  background: $dark-bg-primary;
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  box-sizing: border-box;
  padding: 0 $spacing-md;
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
  /* flex 子项默认 min-height:auto，消息一多就会撑破容器把输入栏顶出视口；
     .page 现在是 height:100vh + overflow:hidden，撑出去就再也滚不到输入栏。 */
  min-height: 0;
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
  padding: 12rpx $spacing-md;
  box-sizing: border-box;
  /* 抬到 surface-1：与页面底 #0c0d14 拉开一档，配合 border-strong 描边，
     让输入区有独立分层，而不是和背景糊成一片。 */
  background: $surface-1;
  border-top: 1rpx solid $border-strong;
  transition: padding-bottom 0.15s ease;
}

.msg-input {
  flex: 1;
  height: 72rpx;
  line-height: 72rpx;
  box-sizing: border-box;
  background: rgba(21, 23, 34, 0.68);
  border: 1rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 36rpx;
  padding: 0 $spacing-md;
  font-size: $font-size-base;
  color: $dark-text-primary;

  &-placeholder {
    color: $dark-text-tertiary;
  }
}

.send-btn {
  height: 72rpx;
  line-height: 72rpx;
  box-sizing: border-box;
  padding: 0 $spacing-md;
  background: $brand;
  color: #fff;
  border-radius: 36rpx;
  margin-left: $spacing-md;
  font-size: $font-size-sm;
  font-weight: 500;
  display: flex;
  align-items: center;
  justify-content: center;

  &:active {
    transform: scale(0.97);
  }

  &.disabled {
    opacity: 0.45;
    pointer-events: none;
  }
}
</style>
