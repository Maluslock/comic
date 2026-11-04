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

      <!-- 微信一键登录（仅小程序端） -->
      <!-- #ifdef MP-WEIXIN -->
      <view class="wechat-card">
        <view class="wechat-btn" :class="{ disabled: !agreed }" @click="onWeChatLogin">
          <text class="wechat-btn-text">微信一键登录</text>
        </view>
        <text class="wechat-tip">使用微信身份快速登录，无需注册</text>
      </view>
      <!-- #endif -->

      <!-- Login Form（仅 H5/开发预览保留手机号登录；小程序端只用微信一键登录） -->
      <!-- #ifndef MP-WEIXIN -->
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

      </view>
      <!-- #endif -->

      <!-- 协议：两种登录方式共用 -->
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

      <!-- Footer -->
      <view class="footer">
        <!-- #ifdef MP-WEIXIN -->
        <text class="footer-hint">微信授权登录，无需手机号</text>
        <!-- #endif -->
        <!-- #ifndef MP-WEIXIN -->
        <text class="footer-hint">Mock 登录，无需真实手机号</text>
        <!-- #endif -->
        <text class="footer-copyright">© 2024 米拉漫展</text>
      </view>
    </view>

    <!-- 完善资料（仅小程序端）：本地昵称为空或占位「微信用户」时，登录前先取微信昵称与头像 -->
    <!-- #ifdef MP-WEIXIN -->
    <view v-if="profileStep" class="profile-mask">
      <view class="profile-sheet">
        <text class="profile-title">完善资料</text>
        <text class="profile-hint">填入微信昵称和头像，让摄影师认出你</text>
        <button class="avatar-btn" open-type="chooseAvatar" @chooseavatar="onPickWxAvatar">
          <image v-if="avatarTempPath" :src="avatarTempPath" class="avatar-img" mode="aspectFill" />
          <view v-else class="avatar-empty">
            <text class="avatar-empty-mark">◇</text>
            <text class="avatar-empty-text">选择微信头像</text>
          </view>
        </button>
        <input
          v-model="nickName"
          class="nick-input"
          type="nickname"
          placeholder="点输入框填入微信昵称"
          placeholder-style="color: #64748b"
          maxlength="20"
        />
        <view class="profile-actions">
          <view class="profile-btn ghost" @click="skipProfile">
            <text class="profile-btn-text ghost">跳过</text>
          </view>
          <view class="profile-btn primary" @click="confirmProfile">
            <text class="profile-btn-text">保存并登录</text>
          </view>
        </view>
        <text class="profile-tip">可跳过，之后也能在资料页补填</text>
      </view>
    </view>
    <!-- #endif -->
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { useUserStore } from '@/stores/user'
import { decodeRedirect } from '@/utils/mappers'
import { isPlaceholderName } from '@/utils/placeholder-name'
import { uploadFile, updateUserProfile } from '@/api/index'

const userStore = useUserStore()

const phone = ref('')
const code = ref('')
const agreed = ref(false)
const counting = ref(false)
const countDown = ref(60)
let timer: ReturnType<typeof setInterval> | null = null

const { statusBarHeight } = useStatusBar()

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

// ===== 完善资料（微信昵称 + 头像，仅小程序端弹层用） =====
const profileStep = ref(false)
const nickName = ref('')
const avatarTempPath = ref('')

interface AvatarPickEvent {
  detail: { avatarUrl: string }
}

function onPickWxAvatar(e: AvatarPickEvent) {
  const url = e?.detail?.avatarUrl
  if (url) avatarTempPath.value = url
}

/** 本地还没有真实昵称（未登录 / 为空 / 占位名）→ 登录前先弹完善资料 */
function needProfile(): boolean {
  const u = userStore.user
  // 占位判定与后端 isPlaceholderName() 同源：'11'、'摄影师15' 这类系统值不算真名
  return !u || isPlaceholderName(u.name)
}

async function onWeChatLogin() {
  if (!agreed.value) {
    uni.showToast({ title: '请先阅读并同意《用户协议》和《隐私政策》', icon: 'none' })
    return
  }
  if (needProfile()) {
    profileStep.value = true
    return
  }
  await doWeChatLogin()
}

function skipProfile() {
  // 跳过：清掉已填内容，维持原有登录行为
  profileStep.value = false
  nickName.value = ''
  avatarTempPath.value = ''
  doWeChatLogin()
}

function confirmProfile() {
  profileStep.value = false
  const profile =
    nickName.value.trim() || avatarTempPath.value
      ? { nickName: nickName.value.trim(), avatarTempPath: avatarTempPath.value }
      : undefined
  doWeChatLogin(profile)
}

async function doWeChatLogin(profile?: { nickName?: string; avatarTempPath?: string }) {
  uni.showLoading({ title: '登录中...' })
  try {
    const loginRes = await new Promise<UniApp.LoginRes>((resolve, reject) => {
      uni.login({ provider: 'weixin', success: resolve, fail: reject })
    })
    if (!loginRes.code) throw new Error('no code')

    // 头像临时路径必须先经 /api/v1/upload 换正式 URL 再传后端（临时路径下次启动即失效）。
    // 登录前可能拿不到 token 导致上传失败 —— 不阻塞昵称保存，登录成功后带着 token 重试补一次。
    let avatarUrl = ''
    let avatarUploadFailed = false
    if (profile?.avatarTempPath) {
      try {
        avatarUrl = await uploadFile(profile.avatarTempPath)
      } catch {
        avatarUploadFailed = true
      }
    }

    const ok = await userStore.loginWeChat(loginRes.code, {
      nickName: profile?.nickName || '',
      avatar: avatarUrl
    })
    uni.hideLoading()
    if (!ok) {
      uni.showToast({ title: '微信登录失败，请重试', icon: 'none' })
      return
    }

    // 登录前上传失败（多为首次登录无 token）：登录成功后重试，失败仅记日志不阻塞
    if (avatarUploadFailed && profile?.avatarTempPath) {
      try {
        const url = await uploadFile(profile.avatarTempPath)
        await updateUserProfile({
          name: userStore.user?.name || '微信用户',
          avatar: url,
          bio: userStore.user?.bio || ''
        })
        if (userStore.user) {
          userStore.user.avatar = url
          uni.setStorageSync('user', JSON.stringify(userStore.user))
        }
      } catch (e) {
        console.warn('[Login] avatar upload failed after login:', e)
      }
    }

    uni.showToast({ title: '登录成功', icon: 'success' })
    setTimeout(() => uni.switchTab({ url: '/pages/profile/index' }), 600)
  } catch (e) {
    uni.hideLoading()
    console.error('[Login] wechat login error:', e)
    uni.showToast({ title: '微信登录已取消', icon: 'none' })
  }
}

function showMock(title: string) {
  uni.showToast({ title: `${title}（模拟）`, icon: 'none' })
}

async function onLogin() {
  if (!canLogin.value) {
    // 此前这里静默 return：填完手机号+验证码点「登录」毫无反应，用户无法得知
    // 自己漏勾了协议、还是验证码格式不对。逐项说明原因。
    if (!canSend.value) {
      uni.showToast({ title: '请输入正确的手机号', icon: 'none' })
    } else if (!/^\d{4,6}$/.test(code.value)) {
      uni.showToast({ title: '请输入 4-6 位验证码', icon: 'none' })
    } else {
      uni.showToast({ title: '请先阅读并同意《用户协议》和《隐私政策》', icon: 'none' })
    }
    return
  }

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
  const rawRedirect = currentPage?.options?.redirect as string | undefined
  // 跳转方已对 redirect 整体 encodeURIComponent（内层 id= 不会再被当成登录页参数），这里配对解码。
  // 老跳转方未编码时 decode 会原样返回（无 % 序列），向后兼容。
  const redirect = rawRedirect ? decodeRedirect(rawRedirect) : undefined

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

// ===== WeChat =====
.wechat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: $spacing-lg;
}

.wechat-btn {
  width: 100%;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-lg;
  background: #07c160;
  box-shadow: 0 8rpx 24rpx rgba(7, 193, 96, 0.28);

  &.disabled {
    opacity: 0.5;
  }
}

.wechat-btn-text {
  font-size: $font-size-md;
  font-weight: 600;
  color: #ffffff;
}

.wechat-tip {
  margin-top: $spacing-sm;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
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

// ===== 完善资料弹层（微信昵称 + 头像，仅小程序端） =====
.profile-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.72);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

.profile-sheet {
  width: 620rpx;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: 32rpx;
  padding: 48rpx 40rpx 36rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 8rpx 48rpx rgba(0, 0, 0, 0.45);
}

.profile-title {
  font-size: 36rpx;
  font-weight: 600;
  color: $dark-text-primary;
}

.profile-hint {
  font-size: 24rpx;
  color: $dark-text-secondary;
  margin-top: 12rpx;
}

.avatar-btn {
  width: 220rpx;
  height: 220rpx;
  margin-top: 40rpx;
  padding: 0;
  background: $dark-bg-secondary;
  border: 2rpx dashed $neon-purple;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;

  &::after {
    border: none;
  }
}

.avatar-img {
  width: 216rpx;
  height: 216rpx;
  border-radius: 50%;
}

.avatar-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.avatar-empty-mark {
  font-size: 64rpx;
  color: $neon-purple;
  line-height: 1;
}

.avatar-empty-text {
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-top: 12rpx;
}

.nick-input {
  width: 100%;
  height: 88rpx;
  margin-top: 36rpx;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: 16rpx;
  padding: 0 24rpx;
  font-size: 28rpx;
  color: $dark-text-primary;
  box-sizing: border-box;
}

.profile-actions {
  width: 100%;
  display: flex;
  margin-top: 40rpx;
}

.profile-btn {
  flex: 1;
  height: 80rpx;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;

  &.ghost {
    border: 1rpx solid $dark-border;
    margin-right: 20rpx;
  }

  &.primary {
    background: $neon-gradient;
    box-shadow: 0 0 24rpx $neon-purple-glow;
  }

  &:active {
    transform: scale(0.97);
  }
}

.profile-btn-text {
  font-size: 28rpx;
  color: #fff;

  &.ghost {
    color: $dark-text-secondary;
  }
}

.profile-tip {
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-top: 24rpx;
}
</style>
