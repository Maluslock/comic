<template>
  <view class="page">
    <view class="header">
      <view class="user-card" :class="{ guest: !isLoggedIn }" @click="goLogin">
        <image :src="displayAvatar" class="avatar" mode="aspectFill" @error="avatarBroken = true" />
        <view class="user-info">
          <view class="user-name-row">
            <text class="user-name">{{ displayName }}</text>
            <!-- 占位名提示从「下划线蓝字」改成小圆角胶囊：下划线是网页链接的样子，
                 深色卡片里既不像可点、又显得廉价（用户原话「丑的要死」）。
                 参考奶茶/外卖 App 的通行做法：名字正常展示，旁边挂一个低饱和度的行动胶囊。 -->
            <text v-if="isLoggedIn && isPlaceholderName" class="name-fix-pill" @click.stop="goEdit">去完善昵称</text>
          </view>
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
        <!--
          定价入口以前只藏在「开通摄影师」页里，摄影师自己也找不到（用户原话：
          「三天去三场漫展，我自己怎么定价？看半天没看到入口」）。这里把它提到
          接单管理正下方常驻可见 —— 紧邻接单入口，符合"接单 → 定价"的操作顺序。
        -->
        <view v-if="isPhotographer" class="menu-item" @click="goPhotographerServices">
          <image class="menu-icon" src="/static/icons/action-tag.png" mode="aspectFit" />
          <text class="menu-text">我的套餐 · 定价</text>
          <text class="menu-arrow">›</text>
        </view>
        <view v-if="isPhotographer" class="menu-item" @click="goPhotographerEventApply">
          <image class="menu-icon" src="/static/icons/action-calendar.png" mode="aspectFit" />
          <text class="menu-text">我的漫展入驻</text>
          <text class="menu-arrow">›</text>
        </view>
        <view v-if="!isPhotographer" class="menu-item" @click="goActivate">
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
import { isPlaceholderName as isPlaceholderNameValue } from '@/utils/placeholder-name'
import type { CreditSummary } from '@/types'

const userStore = useUserStore()

const isLoggedIn = computed(() => userStore.isLoggedIn)
const user = computed(() => userStore.user)

// 占位名（空 /「微信用户」/「11」这类纯数字 /「摄影师15」兜底名）不当真名展示：
// 判定直接复用 isPlaceholderName，与后端 isPlaceholderName() 同源，避免两边漂移。
const isPlaceholderName = computed(() => {
  if (!isLoggedIn.value) return false
  return isPlaceholderNameValue(user.value?.name)
})
const displayName = computed(() => {
  if (!isLoggedIn.value) return '点击登录'
  // 占位名不再在名字位置显示「点击完善昵称」——那是把提示当名字，读起来像报错。
  // 名字位显示账号真昵称（哪怕是「微信用户」），行动统一交给右边的胶囊。
  return user.value?.name || '微信用户'
})
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
  if (needLogin('/pages/profile/edit')) return
  uni.navigateTo({ url: '/pages/profile/edit', animationType: 'none' })
}

// 名字行原本整体可点（占位名时直达资料编辑）。改版后占位引导交给旁边的
// 「去完善昵称」胶囊，名字本身不再绑定点击 —— 普通用户点自己名字没有任何动作
// 才是符合直觉的（奶茶 App 也是这么做的）。旧的 onNameClick 因此删除。

function goActivate() {
  if (needLogin('/pages/photographer/activate')) return
  uni.navigateTo({ url: '/pages/photographer/activate', animationType: 'none' })
}

function goPhotographerOrders() {
  if (needLogin('/pages/photographer/orders')) return
  uni.navigateTo({ url: '/pages/photographer/orders', animationType: 'none' })
}

// 定价入口：从「我的套餐」页直接改价/建套餐（此前只有 activate.vue 一个入口）。
function goPhotographerServices() {
  uni.navigateTo({ url: '/pages/photographer/services', animationType: 'none' })
}

// 漫展入驻：tab=mine 让申请页直接落在「我的申请」而不是申请表单。
function goPhotographerEventApply() {
  uni.navigateTo({ url: '/pages/photographer/event-apply?tab=mine', animationType: 'none' })
}

// ── 登录守卫 ───────────────────────────────────────────────────────────────
// 用户原话：「我还没登录为什么会有我的作品」「在我的页面，点击几个东西应该都会跳转
// 微信登录页面才对」。此前 goOrders/goFavorites/goFollows/goWorks/goSettings 五个入口
// 完全没有守卫：游客点进去，下游页面要么只弹一句「请先登录」的 toast 然后留在空白页，
// 要么（作品页）直接拿首页推荐位的数据冒充「我的作品」。
//
// 统一收敛到 needLogin()：未登录 → 带上 redirect 去登录页，登录成功后原地回来。
// 五个入口各自写一份判断的写法本身就是缺陷来源，这里一次性收口。
function needLogin(redirect?: string): boolean {
  if (isLoggedIn.value) return false
  const target = redirect ? `?redirect=${encodeURIComponent(redirect)}` : ''
  uni.navigateTo({ url: `/pages/login/index${target}` })
  return true
}

function goOrders() {
  if (needLogin('/pages/order/list')) return
  uni.navigateTo({ url: '/pages/order/list', animationType: 'none' })
}

function goCredit() {
  if (needLogin('/pages/credit/index')) return
  uni.navigateTo({ url: '/pages/credit/index', animationType: 'none' })
}

function goFavorites() {
  if (needLogin('/pages/favorite/list')) return
  uni.navigateTo({ url: '/pages/favorite/list', animationType: 'none' })
}

function goFollows() {
  if (needLogin('/pages/follow/list')) return
  uni.navigateTo({ url: '/pages/follow/list', animationType: 'none' })
}

function goWorks() {
  // 摄影师看自己的作品列表，Coser 看「我的作品」（自己收藏/投稿的展示）。
  // 无论哪种：没登录就不该有内容 —— 之前游客进 portfolio 会拉到首页 featuredWorks，
  // 屏幕上出现别人的作品但标题写着「我的作品」，这是数据串台的硬伤。
  const url = isPhotographer.value ? '/pages/photographer/works' : '/pages/portfolio/index'
  if (needLogin(url)) return
  uni.navigateTo({ url, animationType: 'none' })
}

function goReviews() {
  if (needLogin('/pages/comment/index')) return
  uni.navigateTo({ url: '/pages/comment/index', animationType: 'none' })
}

function goSettings() {
  // 设置页里有退出登录、账号安全、拉黑名单等，游客进来只有一堆空洞的开关，
  // 且退出登录按钮会直接再跳一次登录页 —— 统一改成先登录。
  if (needLogin('/pages/settings/index')) return
  uni.navigateTo({ url: '/pages/settings/index', animationType: 'none' })
}

function goHelp() {
  uni.navigateTo({ url: '/pages/settings/doc?type=help', animationType: 'none' })
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

.user-name-row {
  display: flex;
  align-items: center;
  min-width: 0;
}

.user-name {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $dark-text-primary;
  // 占位名不再变色+下划线：名字本身按正常用户展示，
  // 「去补昵称」的引导交给旁边的 .name-fix-pill 胶囊。
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

// 占位名行动胶囊：深色半透明底 + 白字。
// 不能沿用品牌橙（#FF6B00 文字压 #FF6B00 页头 = 对比度约 1:1，实测截图上几乎隐形，
// 用户原话就是嫌原来的蓝字下划线丑、又不能又造出一个看不见的元素）。
// 深底 #0C0D14 以 55% 叠在橙头上约 #83360B，白字对比度 ≈8.5:1（AA/AAA 均过），
// 与右侧「编辑」按钮同属浅色文字系，视觉上不抢名字。
.name-fix-pill {
  flex-shrink: 0;
  margin-left: 12rpx;
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.45);
  background: rgba(12, 13, 20, 0.55);
  font-size: $font-size-xs;
  line-height: 32rpx;
  color: #fff;

  &:active {
    background: rgba(12, 13, 20, 0.75);
  }
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
