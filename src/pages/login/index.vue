<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <template v-if="firstLoading">
      <view class="skeleton-page">
        <view class="sk-line sk-w-60" />
        <view class="sk-line sk-w-40" />
        <view class="sk-card sk-pad"><view class="sk-line sk-w-30" /><view class="sk-line sk-w-full" /><view class="sk-line sk-w-full" /></view>
        <ListSkeleton :rows="2" variant="card" />
      </view>
    </template>
    <template v-else>
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <view class="container">
      <!-- Hero -->
      <view class="hero">
        <text class="hero-title">米拉漫展</text>
        <view class="hero-divider" />
        <text class="hero-tagline">找到你的专属摄影师</text>
      </view>

      <!-- 微信登录（仅小程序端）：头像 + 昵称都是必填，不再有「跳过」与「暂不设置」 -->
      <!-- #ifdef MP-WEIXIN -->
      <view class="wechat-card">
        <!-- 头像：微信规定 open-type="chooseAvatar" 必须由用户点击触发，所以这里是显式
             的「点击选择头像」入口，选完立即上传并回显，用户能看见自己选的是哪张。 -->
        <button class="avatar-picker" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">
          <image v-if="avatarPreview" :src="avatarPreview" class="avatar-img" mode="aspectFill" />
          <view v-else class="avatar-img avatar-empty">
            <text class="avatar-empty-mark">+</text>
          </view>
          <text class="avatar-picker-text">{{ avatarPreview ? '点击更换头像' : '点击选择微信头像' }}</text>
        </button>
        <!-- 兜底入口：chooseAvatar 要求基础库 ≥ 2.21.2，老版本微信点上面的按钮没反应。
             给一个同样能选图的相册通道，不让用户卡死在登录页。 -->
        <view class="avatar-album" @click="pickFromAlbum">从相册选择</view>

        <!-- 昵称：必须用 <input type="nickname">，微信只在原生键盘上方提供
             「使用微信昵称」按钮。2022 年后 wx.getUserProfile 已废弃，只能拿到
             「微信用户」占位名，没有别的合法途径读真实昵称。 -->
        <view class="nickname-row">
          <text class="nickname-label">昵称</text>
          <input
            v-model="nickName"
            class="nickname-input"
            type="nickname"
            maxlength="20"
            placeholder="点击输入，用键盘上方「使用微信昵称」"
            placeholder-class="nickname-placeholder"
          />
        </view>

        <button
          class="wechat-btn"
          :class="{ disabled: !canWechatLogin }"
          @click="doWeChatLogin"
        >
          <text class="wechat-btn-text">微信一键登录</text>
        </button>
        <text class="wechat-tip">头像与昵称都会带到账号资料，登录后可再修改</text>
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

    </template>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { useUserStore } from '@/stores/user'
import { decodeRedirect } from '@/utils/mappers'
import { uploadFile, updateMyAvatar } from '@/api/index'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

onMounted(() => { firstLoading.value = false })

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

// ===== 微信头像 + 昵称（两者都必填） =====
// 头像走 open-type="chooseAvatar"，昵称走 <input type="nickname">。
// 这是微信目前唯一合规的两条路：wx.getUserProfile 自 2022 年起只会返回
// 「微信用户」占位名，拿不到真实昵称。
const avatarTempPath = ref('')
const avatarPreview = ref('')
const nickName = ref('')

/** 头像与昵称都齐了才允许登录 —— 用户明确要求「默认就是需要」，不给跳过入口。 */
const canWechatLogin = computed(
  () => !!avatarTempPath.value && !!nickName.value.trim() && agreed.value
)

interface AvatarPickEvent {
  detail: { avatarUrl: string }
}

/** 相册兜底：老基础库没有 chooseAvatar 时也能选头像。 */
function pickFromAlbum() {
  uni.chooseImage({
    count: 1,
    sizeType: ['compressed'],
    sourceType: ['album'],
    success: (res) => {
      const path = (res.tempFilePaths && res.tempFilePaths[0]) || ''
      if (!path) return
      avatarTempPath.value = path
      avatarPreview.value = path
    },
    fail: (err) => {
      const raw = String((err && (err as AnyObject).errMsg) || '')
      // 用户主动取消不算失败，不弹提示
      if (raw.indexOf('cancel') >= 0 || raw.indexOf('取消') >= 0) return
      uni.showToast({ title: '选择图片失败', icon: 'none' })
    }
  })
}

/** 选完头像只做回显，不直接登录：此时昵称可能还没填。 */
function onChooseAvatar(e: AvatarPickEvent) {
  const url = e?.detail?.avatarUrl
  if (!url) return
  avatarTempPath.value = url
  avatarPreview.value = url
}

async function doWeChatLogin() {
  const fallbackNick = nickName.value.trim()
  const fallbackAvatar = avatarTempPath.value

  if (!fallbackNick) {
    uni.showToast({ title: '请填写昵称', icon: 'none' })
    return
  }
  if (!fallbackAvatar) {
    uni.showToast({ title: '请选择头像', icon: 'none' })
    return
  }
  if (!agreed.value) {
    uni.showToast({ title: '请先阅读并同意《用户协议》和《隐私政策》', icon: 'none' })
    return
  }
  await doWeChatLoginInternal({ nickName: fallbackNick, avatarTempPath: fallbackAvatar })
}

async function doWeChatLoginInternal(profile: { nickName: string; avatarTempPath: string }) {
  uni.showLoading({ title: '登录中...' })
  try {
    const loginRes = await new Promise<UniApp.LoginRes>((resolve, reject) => {
      uni.login({ provider: 'weixin', success: resolve, fail: reject })
    })
    if (!loginRes.code) throw new Error('no code')

    // 头像要换正式 URL 才能落库（临时路径下次启动即失效）。
    //
    // 关键时序：/api/v1/upload 需要 Bearer token，而登录前根本没有 token ——
    // 之前先传头像再登录，上传必然 401，被 catch 吞掉后只在控制台留一行 warn，
    // 用户看到的就是「接口调用正常，但头像没上去」。现在改成：
    //   先 login 拿到 token → 再用 token 上传 → 再 PUT /me/avatar。
    // 头像失败不影响登录成功，只补一条提示。
    let avatarUrl = ''
    if (profile?.avatarTempPath) {
      // 先登录，拿到 token
      // （loginWeChat 内部会写 storage 里的 token，uploadFile 正是从那里取）
    }

    const ok = await userStore.loginWeChat(loginRes.code, {
      nickName: profile?.nickName || '',
      avatar: ''
    })
    uni.hideLoading()
    if (!ok) {
      uni.showToast({ title: '微信登录失败，请重试', icon: 'none' })
      return
    }

    // 登录成功后（token 已就位）再上传头像并落库。
    // 两个后端点都要打：/me/avatar 写 users.avatar 并同步 photographers.avatar。
    if (profile?.avatarTempPath) {
      try {
        avatarUrl = await uploadFile(profile.avatarTempPath)
        await updateMyAvatar(avatarUrl)
        if (userStore.user) {
          userStore.user.avatar = avatarUrl
          uni.setStorageSync('user', JSON.stringify(userStore.user))
        }
      } catch (e) {
        console.warn('[Login] avatar upload failed after login:', e)
        uni.showToast({ title: '头像暂未保存，可稍后在资料页重设', icon: 'none' })
      }
    }

    uni.showToast({ title: '登录成功', icon: 'success' })
    setTimeout(() => goAfterLogin(), 600)
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
  setTimeout(() => goAfterLogin(), 800)
}

/**
 * 登录成功后的去向：有 redirect 就回跳原页面，否则回「我的」。
 * 手机号登录与微信登录共用同一套逻辑 —— 此前微信登录硬写 switchTab，
 * 从「我的订单」跳来登录的用户登录完落不到订单页。
 */
function goAfterLogin() {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  const rawRedirect = currentPage?.options?.redirect as string | undefined
  // 跳转方已对 redirect 整体 encodeURIComponent（内层 id= 不会再被当成登录页参数），这里配对解码。
  // 老跳转方未编码时 decode 会原样返回（无 % 序列），向后兼容。
  const redirect = rawRedirect ? decodeRedirect(rawRedirect) : undefined

  if (redirect) {
    uni.redirectTo({ url: redirect })
  } else {
    uni.switchTab({ url: '/pages/profile/index' })
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  &::before {
    content: '';
    position: fixed;
    top: var(--window-top, 0px);
    right: 0;
    bottom: 0;
    left: 0;
    background: $surface-page;
    z-index: -1;
    pointer-events: none;
  }
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

/* 头像选择：整块是一个 button（open-type 只能挂在 button 上），
   点哪里都触发微信头像选择，热区远大于一个 96rpx 的小圆圈。 */
.avatar-picker {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  padding: $spacing-lg 0;
  margin-bottom: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;

  &::after {
    border: none;
  }
}

.avatar-img {
  width: 160rpx;
  height: 160rpx;
  border-radius: 50%;
  background: $dark-bg-secondary;
}

.avatar-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx dashed rgba(255, 107, 0, 0.5);
}

.avatar-empty-mark {
  font-size: 56rpx;
  line-height: 1;
  color: #ff8a3d;
}

.avatar-picker-text {
  margin-top: $spacing-md;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.avatar-album {
  margin-top: $spacing-sm;
  padding: 12rpx 24rpx;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

/* 昵称：必须用 <input type="nickname">，微信只在键盘上方给「使用微信昵称」 */
.nickname-row {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 0 $spacing-md;
  margin-bottom: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
}

.nickname-label {
  flex-shrink: 0;
  width: 88rpx;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.nickname-input {
  flex: 1;
  height: 88rpx;
  font-size: $font-size-md;
  color: $dark-text-primary;
}

.nickname-placeholder {
  color: $dark-text-tertiary;
  font-size: $font-size-sm;
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
/* ── 首屏深色骨架：切页不再露白（B 档）── */
.skeleton-page { min-height: 100vh; padding: $spacing-md; box-sizing: border-box; display: flex; flex-direction: column; gap: 24rpx; }
.sk-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; box-sizing: border-box; }
.sk-block { background: $surface-2; border-radius: 12rpx; }
.sk-line { height: 28rpx; background: $surface-2; border-radius: 8rpx; }
.sk-circle { background: $surface-2; border-radius: 50%; flex-shrink: 0; }
.sk-row { display: flex; align-items: center; gap: 24rpx; }
.sk-pad { padding: 28rpx; display: flex; flex-direction: column; gap: 24rpx; }
.sk-avatar-120 { width: 120rpx; height: 120rpx; }
.sk-avatar-88 { width: 88rpx; height: 88rpx; }
.sk-hero { height: 160rpx; }
.sk-grid { display: flex; flex-wrap: wrap; gap: 16rpx; }
.sk-grid-item { width: calc(50% - 8rpx); height: 320rpx; border-radius: 16rpx; }
.sk-w-25 { width: 25%; } .sk-w-30 { width: 30%; } .sk-w-40 { width: 40%; }
.sk-w-60 { width: 60%; } .sk-w-80 { width: 80%; } .sk-w-full { width: 100%; }
</style>
