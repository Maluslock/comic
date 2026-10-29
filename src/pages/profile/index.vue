<template>
  <view class="page">
    <view class="header">
      <view class="user-card" :class="{ guest: !isLoggedIn }" @click="goLogin">
        <image :src="displayAvatar" class="avatar" mode="aspectFill" @error="avatarBroken = true" />
        <view class="user-info">
          <text class="user-name">{{ displayName }}</text>
          <view v-if="isLoggedIn" class="user-role">
            <text class="role-tag" :class="isPhotographer ? 'photographer' : 'coser'">
              {{ isPhotographer ? '摄影师' : 'Coser' }}
            </text>
            <text v-if="certified" class="cert-badge">✓ 认证摄影师</text>
          </view>
          <text class="user-desc">{{ displayDesc }}</text>
        </view>
        <text v-if="isLoggedIn" class="edit-btn" @click.stop="goEdit">编辑</text>
        <text v-if="!isLoggedIn" class="login-arrow">›</text>
      </view>

      <view class="stats-row">
        <view class="stat-item">
          <text class="stat-value">{{ stats.favorites }}</text>
          <text class="stat-label">收藏</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ stats.follows }}</text>
          <text class="stat-label">关注</text>
        </view>
        <view class="stat-divider"></view>
        <view class="stat-item">
          <text class="stat-value">{{ stats.works }}</text>
          <text class="stat-label">作品</text>
        </view>
      </view>
    </view>

    <!-- 信用分卡片 -->
    <view v-if="isLoggedIn" class="credit-card" @click="goCredit">
      <view class="credit-left">
        <text class="credit-title">信用分</text>
        <view class="credit-score-row">
          <text class="credit-score">{{ credit.score }}</text>
          <text class="credit-unit">分</text>
          <text class="credit-pill" :class="`tone-${creditTone}`">{{ creditLevelText }}</text>
        </view>
        <text v-if="creditFrozenText" class="credit-frozen">{{ creditFrozenText }}</text>
        <text v-else class="credit-desc">守时履约可以提升信用分</text>
      </view>
      <view class="credit-right">
        <view class="credit-stat">
          <text class="credit-stat-num">{{ credit.successCount }}</text>
          <text class="credit-stat-name">成功</text>
        </view>
        <view class="credit-stat">
          <text class="credit-stat-num">{{ credit.noshowCount }}</text>
          <text class="credit-stat-name">爽约</text>
        </view>
        <text class="credit-arrow">›</text>
      </view>
    </view>

    <view class="menu-section">
      <view class="menu-group">
        <view class="menu-item" @click="goOrders">
          <image class="menu-icon" src="/static/icons/menu-order.svg" mode="aspectFit" />
          <text class="menu-text">我的订单</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goFavorites">
          <image class="menu-icon" src="/static/icons/menu-fav.svg" mode="aspectFit" />
          <text class="menu-text">我的收藏</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goFollows">
          <image class="menu-icon" src="/static/icons/menu-calendar.svg" mode="aspectFit" />
          <text class="menu-text">关注的漫展</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goWorks">
          <image class="menu-icon" src="/static/icons/menu-image.svg" mode="aspectFit" />
          <text class="menu-text">我的作品</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goReviews">
          <image class="menu-icon" src="/static/icons/menu-chat.svg" mode="aspectFit" />
          <text class="menu-text">我的评价</text>
          <text class="menu-arrow">›</text>
        </view>
        <view v-if="isPhotographer" class="menu-item" @click="goPhotographerOrders">
          <image class="menu-icon" src="/static/icons/menu-camera.svg" mode="aspectFit" />
          <text class="menu-text">接单管理</text>
          <text class="menu-arrow">›</text>
        </view>
        <view v-else class="menu-item" @click="goActivate">
          <image class="menu-icon" src="/static/icons/menu-camera.svg" mode="aspectFit" />
          <text class="menu-text">我是摄影师</text>
          <text class="menu-arrow">›</text>
        </view>
      </view>

      <view class="menu-group">
        <view class="menu-item" @click="goSettings">
          <image class="menu-icon" src="/static/icons/menu-settings.svg" mode="aspectFit" />
          <text class="menu-text">设置</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="goHelp">
          <image class="menu-icon" src="/static/icons/menu-help.svg" mode="aspectFit" />
          <text class="menu-text">帮助与反馈</text>
          <text class="menu-arrow">›</text>
        </view>
        <view class="menu-item" @click="logout">
          <image class="menu-icon" src="/static/icons/menu-logout.svg" mode="aspectFit" />
          <text class="menu-text">退出登录</text>
          <text class="menu-arrow">›</text>
        </view>
      </view>
    </view>

    <view class="bottom-info">
      <text class="version" @longpress="onVersionLongPress">版本 1.0.0 · 长按切换接口</text>
      <text class="copyright">© 2024 米拉漫展</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet } from '@/api/client'
import { getMyPhotographerProfile, getMyCredit } from '@/api'
import { useUserStore } from '@/stores/user'
import { getApiBase, setApiBase, resetApiBase } from '@/utils/apiBase'
import { creditLevelText as levelTextOf, creditTone as toneOf, frozenHint } from '@/utils/credit'
import type { CreditSummary } from '@/types'

const userStore = useUserStore()

const isLoggedIn = computed(() => userStore.isLoggedIn)
const user = computed(() => userStore.user)

const displayName = computed(() => user.value?.name ?? '点击登录')
const avatarBroken = ref(false)
const DEFAULT_AVATAR = '/static/img/avatar-user.svg'
const displayAvatar = computed(() => (avatarBroken.value ? DEFAULT_AVATAR : user.value?.avatar || DEFAULT_AVATAR))
const displayDesc = computed(() => {
  if (!isLoggedIn.value) return '登录后体验完整功能'
  return user.value?.bio || '这个人很懒，什么都没写'
})
const isPhotographer = computed(() => !!userStore.user?.photographerId)
const certified = ref(false)

const stats = ref({ favorites: 0, follows: 0, works: 0 })

const credit = ref<CreditSummary>({
  score: 100,
  level: 'good',
  successCount: 0,
  cancelCount: 0,
  noshowCount: 0,
  disputeLostCount: 0,
  frozenUntil: null,
  updatedAt: null
})
const creditTone = computed(() => toneOf(credit.value.level, credit.value.score))
const creditLevelText = computed(() => levelTextOf(credit.value.level, credit.value.score))
const creditFrozenText = computed(() => frozenHint(credit.value.frozenUntil))

async function loadCredit() {
  if (!isLoggedIn.value) return
  try {
    credit.value = await getMyCredit()
  } catch {
    // 保留默认 100 分，界面仍可展示
  }
}

async function loadStats() {
  if (!isLoggedIn.value || !userStore.user) {
    stats.value = { favorites: 0, follows: 0, works: 0 }
    return
  }
  const uid = userStore.user.id

  try {
    const favs = await apiGet<any[]>(`/v1/favorites/${uid}`)
    stats.value.favorites = (favs || []).length
  } catch {
    stats.value.favorites = 0
  }

  try {
    const res = await apiGet<{ list: any[] }>(`/v1/follows/${uid}`)
    stats.value.follows = (res?.list || []).length
  } catch {
    stats.value.follows = 0
  }

  if (isPhotographer.value) {
    try {
      const works = await apiGet<any[]>('/v1/photographers/works/mine')
      stats.value.works = (works || []).length
    } catch {
      stats.value.works = 0
    }
    try {
      const profile = await getMyPhotographerProfile()
      certified.value = !!profile?.certified
    } catch {
      certified.value = false
    }
  } else {
    stats.value.works = 0
    certified.value = false
  }
}

onMounted(() => {
  userStore.init()
  loadStats()
  loadCredit()
})

onShow(() => {
  loadStats()
  loadCredit()
})

function goLogin() {
  if (isLoggedIn.value) return
  uni.navigateTo({ url: '/pages/login/index' })
}

function goEdit() {
  if (!isLoggedIn.value) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.navigateTo({ url: '/pages/profile/edit' })
}

function goActivate() {
  if (!isLoggedIn.value) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.navigateTo({ url: '/pages/photographer/activate' })
}

function goPhotographerOrders() {
  uni.navigateTo({ url: '/pages/photographer/orders' })
}

function goOrders() {
  uni.navigateTo({ url: '/pages/order/list' })
}

function goCredit() {
  if (!isLoggedIn.value) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.navigateTo({ url: '/pages/credit/index' })
}

function goFavorites() {
  uni.navigateTo({ url: '/pages/favorite/list' })
}

function goFollows() {
  uni.navigateTo({ url: '/pages/follow/list' })
}

function goWorks() {
  const url = isPhotographer.value ? '/pages/photographer/works' : '/pages/portfolio/index'
  uni.navigateTo({ url })
}

function goReviews() {
  if (!isLoggedIn.value) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.navigateTo({ url: '/pages/comment/index' })
}

function goSettings() {
  uni.navigateTo({ url: '/pages/settings/index' })
}

function goHelp() {
  uni.navigateTo({ url: '/pages/settings/doc?type=help' })
}

function logout() {
  if (!isLoggedIn.value) {
    uni.navigateTo({ url: '/pages/login/index' })
    return
  }
  uni.showModal({
    title: '确认退出',
    content: '确定要退出登录吗？',
    success: (res) => {
      if (res.confirm) {
        userStore.logout()
        uni.showToast({ title: '已退出', icon: 'success' })
      }
    }
  })
}
function onVersionLongPress() {
  const cur = getApiBase()
  uni.showModal({
    title: '接口地址',
    editable: true,
    content: cur,
    placeholderText: 'http://100.64.0.16:8088/api',
    success: (res) => {
      if (!res.confirm) return
      const v = (res.content || '').trim()
      if (!v) {
        resetApiBase()
        uni.showToast({ title: '已恢复默认地址', icon: 'none' })
        return
      }
      const saved = setApiBase(v)
      uni.showToast({ title: '已切换：' + saved, icon: 'none', duration: 2500 })
    }
  })
}

</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.header {
  background: linear-gradient(135deg, $neon-purple 40%, $dark-bg-secondary 100%);
  padding: $spacing-lg $spacing-md;
  padding-top: calc(env(safe-area-inset-top) + #{$spacing-lg});
}

.user-card {
  display: flex;
  align-items: flex-start;

  &.guest {
    align-items: center;
  }

  &:active {
    opacity: 0.9;
  }
}

.login-arrow {
  font-size: 48rpx;
  color: rgba(255, 255, 255, 0.6);
  margin-left: $spacing-sm;
}

.edit-btn {
  flex-shrink: 0;
  padding: 8rpx 28rpx;
  border-radius: 999rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.5);
  font-size: $font-size-sm;
  color: #fff;
  margin-left: $spacing-sm;

  &:active {
    background: rgba(255, 255, 255, 0.2);
  }
}

.avatar {
  width: 160rpx;
  height: 160rpx;
  border-radius: 50%;
  border: 4rpx solid rgba(255, 255, 255, 0.3);
}

.user-info {
  flex: 1;
  margin-left: $spacing-md;
}

.user-name {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $dark-text-primary;
}

.user-role {
  margin-top: $spacing-xs;
  display: flex;
  align-items: center;
}

.cert-badge {
  margin-left: $spacing-xs;
  padding: 2rpx 12rpx;
  border: 2rpx solid #f59e0b;
  background: rgba(245, 158, 11, 0.18);
  color: #fbbf24;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
  box-shadow: 0 0 8rpx rgba(245, 158, 11, 0.35);
}

.role-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  font-size: $font-size-xs;
  
  &.coser {
    background: rgba(255, 255, 255, 0.2);
    color: #fff;
  }
  
  &.photographer {
    background: rgba(255, 215, 0, 0.3);
    color: #fff;
  }
}

.user-desc {
  font-size: $font-size-sm;
  color: rgba(255, 255, 255, 0.8);
  margin-top: $spacing-xs;
  display: block;
}

.stats-row {
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: rgba(255, 255, 255, 0.1);
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  margin-top: $spacing-lg;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: $font-size-xl;
  font-weight: 600;
  color: #fff;
}

.stat-label {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
  margin-top: 4rpx;
}

.stat-divider {
  width: 2rpx;
  height: 48rpx;
  background: rgba(255, 255, 255, 0.2);
}

.credit-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: $spacing-md $spacing-md 0;
  padding: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);

  &:active {
    opacity: 0.9;
  }
}

.credit-left {
  flex: 1;
}

.credit-title {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.credit-score-row {
  display: flex;
  align-items: baseline;
  margin-top: 4rpx;
}

.credit-score {
  font-size: $font-size-xl;
  font-weight: 700;
  color: $dark-text-primary;
}

.credit-unit {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-left: 4rpx;
  margin-right: 12rpx;
}

.credit-pill {
  font-size: 22rpx;
  padding: 4rpx 14rpx;
  border-radius: 999rpx;
  border: 1rpx solid $dark-border;
  background: rgba(255, 255, 255, 0.04);

  &.tone-good {
    color: $success-color;
    border-color: rgba(34, 197, 94, 0.4);
  }

  &.tone-fair {
    color: $warning-color;
    border-color: rgba(250, 204, 21, 0.4);
  }

  &.tone-poor {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.4);
  }
}

.credit-desc {
  display: block;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: 6rpx;
}

.credit-frozen {
  display: block;
  font-size: $font-size-xs;
  color: $warning-color;
  margin-top: 6rpx;
}

.credit-right {
  display: flex;
  align-items: center;
}

.credit-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-left: $spacing-md;
}

.credit-stat-num {
  font-size: $font-size-md;
  font-weight: 600;
  color: $dark-text-primary;
}

.credit-stat-name {
  font-size: 22rpx;
  color: $dark-text-tertiary;
  margin-top: 2rpx;
}

.credit-arrow {
  font-size: 40rpx;
  color: $dark-text-tertiary;
  margin-left: $spacing-sm;
}

.menu-section {
  padding: $spacing-md;
}

.menu-group {
  background: $dark-bg-card;
  border-radius: $border-radius-lg;
  overflow: hidden;
  margin-bottom: $spacing-md;
  box-shadow: 0 2rpx 16rpx rgba(0, 0, 0, 0.15);
  border: 1rpx solid $dark-border;
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

.menu-icon {
  width: 36rpx;
  height: 36rpx;
  margin-right: $spacing-sm;
  flex-shrink: 0;
}

.menu-marker {
  font-size: 28rpx;
  margin-right: $spacing-md;
  color: $neon-purple;

  &.fav {
    color: $warning-color;
  }

  &.cam {
    color: $neon-cyan;
  }

  &.exit {
    color: $error-color;
  }
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

.bottom-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl;
  color: $dark-text-tertiary;
}

.version {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.copyright {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-xs;
}
</style>
