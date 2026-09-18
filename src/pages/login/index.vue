<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <view class="container">
      <!-- Hero -->
      <view class="hero">
        <text class="hero-title">米拉漫展</text>
        <view class="hero-divider" />
        <text class="hero-tagline">找到你的专属摄影师</text>
      </view>

      <!-- Login Form -->
      <view class="form-card">
        <view class="input-group">
          <text class="input-label">手机号</text>
          <view class="input-wrap">
            <text class="input-prefix">+86</text>
            <input
              v-model="phone"
              class="input-field"
              type="number"
              maxlength="11"
              placeholder="请输入手机号"
              placeholder-class="input-placeholder"
            />
          </view>
        </view>

        <view class="input-group">
          <text class="input-label">验证码</text>
          <view class="input-wrap">
            <input
              v-model="code"
              class="input-field code-field"
              type="number"
              maxlength="6"
              placeholder="请输入验证码"
              placeholder-class="input-placeholder"
            />
            <view
              class="code-btn"
              :class="{ disabled: counting || !canSend }"
              @click="sendCode"
            >
              <text class="code-btn-text">{{ codeBtnText }}</text>
            </view>
          </view>
        </view>

        <view
          class="login-btn"
          :class="{ disabled: !canLogin }"
          @click="onLogin"
        >
          <text class="login-btn-text">登 录</text>
        </view>

        <view class="agreement-row">
          <view
            class="checkbox"
            :class="{ checked: agreed }"
            @click="agreed = !agreed"
          >
            <text v-if="agreed" class="check-mark">✓</text>
          </view>
          <text class="agreement-text">
            我已阅读并同意
            <text class="agreement-link" @click.stop="showMock('用户协议')">《用户协议》</text>
            和
            <text class="agreement-link" @click.stop="showMock('隐私政策')">《隐私政策》</text>
          </text>
        </view>
      </view>

      <!-- Footer -->
      <view class="footer">
        <text class="footer-hint">Mock 登录，无需真实手机号</text>
        <text class="footer-copyright">© 2024 米拉漫展</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const phone = ref('')
const code = ref('')
const agreed = ref(false)
const counting = ref(false)
const countDown = ref(60)
let timer: ReturnType<typeof setInterval> | null = null

const statusBarHeight = ref(44)
const sysInfo = uni.getSystemInfoSync()
statusBarHeight.value = sysInfo.statusBarHeight || 44

const canSend = computed(() => /^1[3-9]\d{9}$/.test(phone.value) || /^100\d{8}$/.test(phone.value))
const canLogin = computed(() => canSend.value && /^\d{4,6}$/.test(code.value) && agreed.value)

const codeBtnText = computed(() => {
  if (counting.value) return `${countDown.value}s后重发`
  return '获取验证码'
})

function sendCode() {
  if (!canSend.value || counting.value) return
  uni.showToast({ title: '验证码已发送：123456', icon: 'none' })
  counting.value = true
  countDown.value = 60
  timer = setInterval(() => {
    countDown.value--
    if (countDown.value <= 0) {
      if (timer) clearInterval(timer)
      counting.value = false
    }
  }, 1000)
}

function showMock(title: string) {
  uni.showToast({ title: `${title}（模拟）`, icon: 'none' })
}

async function onLogin() {
  if (!canLogin.value) return

  uni.showLoading({ title: '登录中...' })
  const ok = await userStore.login(phone.value, code.value)
  uni.hideLoading()

  if (!ok) {
    uni.showToast({ title: '登录失败，验证码应为1234开头', icon: 'none' })
    return
  }

  uni.showToast({ title: '登录成功', icon: 'success' })

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  const redirect = currentPage?.options?.redirect as string | undefined

  setTimeout(() => {
    if (redirect) {
      uni.redirectTo({ url: redirect })
    } else {
      uni.switchTab({ url: '/pages/profile/index' })
    }
  }, 800)
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.status-bar {
  width: 100%;
}

.container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  padding: 0 $spacing-lg;
  padding-bottom: calc(env(safe-area-inset-bottom) + #{$spacing-lg});
}

// ===== Hero =====
.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: $spacing-xl * 2;
  padding-bottom: $spacing-xl;
}

.hero-title {
  font-size: 72rpx;
  font-weight: 900;
  letter-spacing: 8rpx;
  background: $neon-gradient;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-divider {
  width: 80rpx;
  height: 4rpx;
  margin: $spacing-md auto;
  background: $neon-gradient;
  border-radius: 2rpx;
  opacity: 0.6;
}

.hero-tagline {
  font-size: $font-size-md;
  color: $dark-text-secondary;
  letter-spacing: 6rpx;
}

// ===== Form =====
.form-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-lg;
  box-shadow: 0 8rpx 32rpx rgba(0, 0, 0, 0.25);
}

.input-group {
  margin-bottom: $spacing-lg;

  &:last-of-type {
    margin-bottom: $spacing-xl;
  }
}

.input-label {
  display: block;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-bottom: $spacing-xs;
}

.input-wrap {
  display: flex;
  align-items: center;
  height: 96rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: 0 $spacing-md;
  transition: border-color 0.2s;

  &:focus-within {
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.input-prefix {
  font-size: $font-size-base;
  color: $dark-text-secondary;
  margin-right: $spacing-sm;
  padding-right: $spacing-sm;
  border-right: 1rpx solid $dark-border;
}

.input-field {
  flex: 1;
  height: 100%;
  font-size: $font-size-md;
  color: $dark-text-primary;
  background: transparent;
  border: none;
  outline: none;
}

.input-placeholder {
  color: $dark-text-tertiary;
}

.code-field {
  flex: 1;
}

.code-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 168rpx;
  height: 64rpx;
  margin-left: $spacing-sm;
  padding: 0 $spacing-sm;
  background: $neon-purple;
  border-radius: $border-radius-sm;
  box-shadow: 0 0 12rpx $neon-purple-glow;

  &:active {
    background: rgba($neon-purple, 0.85);
    transform: scale(0.98);
  }

  &.disabled {
    background: $dark-bg-card-hover;
    box-shadow: none;
  }
}

.code-btn-text {
  font-size: $font-size-sm;
  color: $dark-text-primary;
  white-space: nowrap;
}

// ===== Login Button =====
.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 96rpx;
  background: $neon-gradient;
  border-radius: $border-radius-xl;
  box-shadow: 0 8rpx 32rpx $neon-purple-glow;
  margin-bottom: $spacing-md;

  &:active {
    transform: scale(0.98);
    box-shadow: 0 4rpx 20rpx $neon-purple-glow;
  }

  &.disabled {
    opacity: 0.45;
    box-shadow: none;
  }
}

.login-btn-text {
  font-size: $font-size-md;
  font-weight: 700;
  color: $dark-text-primary;
  letter-spacing: 8rpx;
}

// ===== Agreement =====
.agreement-row {
  display: flex;
  align-items: flex-start;
}

.checkbox {
  width: 32rpx;
  height: 32rpx;
  margin-top: 4rpx;
  margin-right: $spacing-sm;
  display: flex;
  align-items: center;
  justify-content: center;
  background: $dark-bg-secondary;
  border: 2rpx solid $dark-border;
  border-radius: 8rpx;

  &.checked {
    background: $neon-purple;
    border-color: $neon-purple;
    box-shadow: 0 0 8rpx $neon-purple-glow;
  }
}

.check-mark {
  font-size: $font-size-sm;
  color: $dark-text-primary;
  font-weight: 700;
}

.agreement-text {
  flex: 1;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  line-height: 1.6;
}

.agreement-link {
  color: $neon-cyan;

  &:active {
    opacity: 0.7;
  }
}

// ===== Footer =====
.footer {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: $spacing-xl;
}

.footer-hint {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  margin-bottom: $spacing-xs;
}

.footer-copyright {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}
</style>
