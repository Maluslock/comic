<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">开通摄影师</text>
    </view>

    <view class="body">
      <view class="hero">
        <text class="hero-icon">◈</text>
        <text class="hero-title">成为摄影师</text>
        <text class="hero-sub">一次开通，即可接单约拍</text>
      </view>

      <view v-if="isPhotographer" class="activated-tip">
        <text class="tip-text">你已是摄影师</text>
        <view class="btn-go" @click="goOrders">去接单管理</view>
        <view class="btn-edit" @click="goProfileEdit">主页管理</view>
        <view class="btn-works" @click="goWorks">作品管理</view>
        <view class="btn-services" @click="goServices">套餐管理</view>
        <view class="btn-cert" :class="{ 'is-cert': certified }" @click="goCertApply">
          <text>{{ certified ? '✓ 已认证摄影师' : '申请认证' }}</text>
        </view>
      </view>

      <view v-else class="form-card">
        <view class="field">
          <text class="field-label">摄影师昵称</text>
          <input
            v-model="form.name"
            class="field-input"
            placeholder="例如：阿米摄影"
            placeholder-style="color: #64748b"
            maxlength="20"
          />
        </view>

        <view class="field">
          <text class="field-label">接单模式</text>
          <view class="mode-row">
            <view
              v-for="m in modes"
              :key="m.value"
              class="mode-item"
              :class="{ active: form.mode === m.value }"
              @click="form.mode = m.value"
            >
              {{ m.label }}
            </view>
          </view>
          <text class="field-hint">{{ modeHint }}</text>
        </view>

        <view class="field">
          <text class="field-label">个人简介</text>
          <textarea
            v-model="form.intro"
            class="field-textarea"
            placeholder="介绍你的拍摄风格、器材、经验…"
            placeholder-style="color: #64748b"
            maxlength="200"
          />
        </view>

        <view class="btn-submit" @click="submit">立即开通</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet, apiPost } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const userStore = useUserStore()
const certified = ref(false)

const modes = [
  { value: 'free', label: '互勉' },
  { value: 'pay', label: '收费' },
  { value: 'both', label: '两者' }
]

const form = reactive({ name: '', mode: 'both', intro: '' })

const isPhotographer = computed(() => !!userStore.user?.photographerId)

const modeHint = computed(() => {
  const map: Record<string, string> = {
    free: '互勉：免费互相创作，积累作品',
    pay: '收费：有偿接单约拍',
    both: '两者：互勉与收费都可接受'
  }
  return map[form.mode] || ''
})

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => {
      uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/activate' })
    }, 800)
  } else {
    loadCertified()
  }
})

async function loadCertified() {
  const pid = userStore.user?.photographerId
  if (!pid) return
  try {
    const res = await apiGet<{ certified?: boolean }>(`/v1/photographers/${pid}`)
    certified.value = !!res.certified
  } catch {
    certified.value = false
  }
}

async function submit() {
  if (!form.name.trim()) {
    uni.showToast({ title: '请填写昵称', icon: 'none' })
    return
  }
  uni.showLoading({ title: '提交中...' })
  try {
    const res = await apiPost<{ photographerId: number }>('/v1/photographers/activate', form)
    if (userStore.user) {
      userStore.user.photographerId = res.photographerId
      uni.setStorageSync('user', JSON.stringify(userStore.user))
    }
    uni.showToast({ title: '开通成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 800)
  } catch {
    uni.showToast({ title: '开通失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function goOrders() {
  uni.navigateTo({ url: '/pages/photographer/orders' })
}

function goProfileEdit() {
  uni.navigateTo({ url: '/pages/photographer/profile-edit' })
}

function goWorks() {
  uni.navigateTo({ url: '/pages/photographer/works' })
}

function goServices() {
  uni.navigateTo({ url: '/pages/photographer/services' })
}

function goCertApply() {
  uni.navigateTo({ url: '/pages/photographer/cert-apply' })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.body { padding: 100rpx 32rpx 48rpx; }

.hero { display: flex; flex-direction: column; align-items: center; padding: 48rpx 0 40rpx; }
.hero-icon { font-size: 96rpx; color: $neon-cyan; text-shadow: 0 0 24rpx $neon-cyan-glow; }
.hero-title { font-size: 44rpx; font-weight: 700; color: $dark-text-primary; margin-top: 16rpx; }
.hero-sub { font-size: 24rpx; color: $dark-text-tertiary; margin-top: 8rpx; }

.activated-tip { display: flex; flex-direction: column; align-items: center; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 48rpx 32rpx; }
.tip-text { font-size: 28rpx; color: $dark-text-secondary; margin-bottom: 24rpx; }
.btn-go { padding: 16rpx 48rpx; background: $neon-gradient; @include on-neon-fill; border-radius: 32rpx; font-size: 28rpx; box-shadow: 0 0 20rpx $neon-purple-glow; }
.btn-edit { margin-top: 24rpx; padding: 14rpx 48rpx; background: $dark-bg-secondary; border: 2rpx solid $neon-purple; color: $neon-purple; border-radius: 32rpx; font-size: 28rpx; }
.btn-works { margin-top: 24rpx; padding: 14rpx 48rpx; background: rgba(6, 182, 212, 0.1); border: 2rpx solid $neon-cyan; color: $neon-cyan; border-radius: 32rpx; font-size: 28rpx; }
.btn-services { margin-top: 24rpx; padding: 14rpx 48rpx; background: rgba(236, 72, 153, 0.1); border: 2rpx solid $neon-pink; color: $neon-pink; border-radius: 32rpx; font-size: 28rpx; }
.btn-cert { margin-top: 24rpx; padding: 14rpx 48rpx; background: $neon-purple-dim; border: 2rpx solid $neon-purple; color: $neon-purple; border-radius: 32rpx; font-size: 28rpx;
  &.is-cert { background: rgba(245, 158, 11, 0.12); border-color: $warning-color; color: $warning-color; box-shadow: 0 0 12rpx rgba(245, 158, 11, 0.25); }
}

.form-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 24rpx; padding: 40rpx 32rpx; box-shadow: 0 4rpx 24rpx rgba(0, 0, 0, 0.25); }
.field { margin-bottom: 36rpx; }
.field-label { display: block; font-size: 26rpx; color: $dark-text-secondary; margin-bottom: 16rpx; }
.field-input { height: 88rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 0 24rpx; font-size: 28rpx; color: $dark-text-primary; }
.field-textarea { width: 100%; height: 200rpx; background: $dark-bg-secondary; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 20rpx 24rpx; font-size: 28rpx; color: $dark-text-primary; box-sizing: border-box; }
.field-hint { display: block; font-size: 22rpx; color: $dark-text-tertiary; margin-top: 12rpx; }

.mode-row { display: flex; gap: 16rpx; }
.mode-item { flex: 1; text-align: center; padding: 20rpx 0; background: $dark-bg-secondary; border: 2rpx solid $dark-border; border-radius: 16rpx; font-size: 28rpx; color: $dark-text-secondary;
  &.active { border-color: $neon-purple; color: $neon-purple; background: $neon-purple-dim; box-shadow: 0 0 12rpx $neon-purple-glow; font-weight: 500; }
}

.btn-submit { margin-top: 48rpx; text-align: center; padding: 24rpx 0; background: $neon-gradient; @include on-neon-fill; border-radius: 44rpx; font-size: 30rpx; font-weight: 600; box-shadow: 0 0 24rpx $neon-purple-glow;
  &:active { transform: scale(0.97); box-shadow: 0 0 12rpx $neon-purple-glow; }
}
</style>
