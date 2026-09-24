<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }"></view>
    
    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <text class="header-title">订单详情</text>
      <view class="placeholder"></view>
    </view>

    <scroll-view scroll-y class="content">
      <view class="status-card" :class="order.status">
        <text class="status-icon">◆</text>
        <text class="status-text">{{ getStatusText(order.status) }}</text>
        <text class="order-id">订单号：{{ order.id }}</text>
      </view>

      <view class="section">
        <view class="section-title">摄影师信息</view>
        <view class="photographer-info">
          <image :src="order.photographerAvatar" class="avatar" mode="aspectFill" />
          <view class="info">
            <text class="name">{{ order.photographerName }}</text>
            <text class="location">{{ order.location }}</text>
          </view>
          <view class="btn-outline" @click="goChat">联系</view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">服务详情</view>
        <view class="service-detail">
          <view class="detail-row">
            <text class="detail-label">服务套餐</text>
            <text class="detail-value">{{ order.serviceName }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄时长</text>
            <text class="detail-value">{{ order.duration }}分钟</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄日期</text>
            <text class="detail-value">{{ order.date }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">拍摄时间</text>
            <text class="detail-value">{{ order.time }}</text>
          </view>
          <view class="detail-row">
            <text class="detail-label">备注信息</text>
            <text class="detail-value">{{ order.remark || '无' }}</text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">费用明细</view>
        <view class="price-detail">
          <view class="price-row">
            <text class="price-label">{{ order.serviceName }}</text>
            <text class="price-value">{{ renderOrderPrice(order) }}</text>
          </view>
          <view class="price-row total">
            <text class="price-label">总计</text>
            <text class="price-value">{{ renderOrderPrice(order) }}</text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-title">下单时间</view>
        <text class="order-time">{{ order.createTime }}</text>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>

    <view class="footer">
      <view 
        v-if="showRespondQuote" 
        class="btn-outline danger"
        @click="rejectQuote"
      >
        拒绝
      </view>
      <view 
        v-if="showRespondQuote" 
        class="btn-primary"
        @click="acceptQuote"
      >
        接受报价
      </view>
      <view 
        v-if="showQuote" 
        class="btn-primary"
        @click="quoteOrder"
      >
        报价
      </view>
      <view 
        v-if="(order.status === 'pending' || order.status === 'confirmed') && !showRespondQuote" 
        class="btn-outline"
        @click="cancelOrder"
      >
        取消预约
      </view>
      <view 
        v-if="order.status === 'confirmed'" 
        class="btn-primary"
        @click="confirmOrder"
      >
        确认完成
      </view>
      <view 
        v-if="order.status === 'completed'" 
        class="btn-primary"
        @click="goReview"
      >
        去评价
      </view>
      <view 
        v-if="order.status === 'pending' || order.status === 'confirmed'" 
        class="btn-primary"
        @click="goChat"
      >
        联系摄影师
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { apiGet, apiPost, apiPut, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import type { OrderPriceFields } from '@/types'
import { renderOrderPrice, toPriceMode, toPriceStatus, promptQuote, confirmRespondQuote } from '@/utils/quote'

interface OrderDetail extends OrderPriceFields {
  id: string
  coserId?: string
  photographerId?: string
  photographerUserId?: number
  photographerName: string
  photographerAvatar: string
  location: string
  serviceId?: string
  serviceName: string
  duration: number
  date: string
  time: string
  status: string
  remark: string
  createTime: string | number
}

const statusBarHeight = ref(44)

const defaultOrder: OrderDetail = {
  id: '',
  photographerName: '未知摄影师',
  photographerAvatar: '',
  location: '',
  serviceName: '未知服务',
  duration: 0,
  date: '',
  time: '',
  remark: '',
  totalPrice: 0,
  priceMode: 'fixed',
  quotePrice: null,
  priceStatus: 'agreed',
  status: 'pending',
  createTime: ''
}

const order = ref<OrderDetail>({ ...defaultOrder })
const userStore = useUserStore()

const bookingId = ref('')

onLoad((options: Record<string, string> | undefined) => {
  bookingId.value = options?.id || ''
  if (!bookingId.value) {
    uni.showToast({ title: '订单不存在', icon: 'none' })
    return
  }
  loadOrder()
})

async function loadOrder() {
  let found: OrderDetail | undefined

  if (userStore.isLoggedIn && userStore.user) {
    try {
      const res = await apiGet<any[]>(`/v1/bookings/${userStore.user.id}`)
      const serverOrder = (res || []).find((b: any) => String(b.id) === bookingId.value)
      if (serverOrder) {
        found = {
          id: String(serverOrder.id),
          coserId: serverOrder.coserId === undefined || serverOrder.coserId === null ? '' : String(serverOrder.coserId),
          photographerId: String(serverOrder.photographerId),
          photographerUserId: serverOrder.photographerUserId,
          photographerName: serverOrder.photographerName || `摄影师${serverOrder.photographerId}`,
          photographerAvatar: serverOrder.photographerAvatar || '',
          location: serverOrder.location || '',
          serviceId: String(serverOrder.serviceId),
          serviceName: serverOrder.serviceName || `服务${serverOrder.serviceId}`,
          duration: serverOrder.duration || 0,
          date: serverOrder.date,
          time: serverOrder.time,
          totalPrice: serverOrder.totalPrice || 0,
          priceMode: toPriceMode(serverOrder.priceMode),
          quotePrice: serverOrder.quotePrice ?? null,
          priceStatus: toPriceStatus(serverOrder.priceStatus),
          status: serverOrder.status || 'pending',
          remark: serverOrder.remarks || '',
          createTime: serverOrder.createdAt || Date.now()
        }
      }
    } catch {
      // Ignore API errors, fall through to storage
    }
  }

  if (!found) {
    try {
      const storedJson = uni.getStorageSync('bookings') || '[]'
      const storedBookings: OrderDetail[] = JSON.parse(storedJson)
      found = storedBookings.find((b: OrderDetail) => String(b.id) === bookingId.value)
    } catch {
      // Ignore parse errors
    }
  }

  if (found) {
    order.value = found
  } else {
    uni.showToast({ title: '订单不存在', icon: 'none' })
  }
}

onMounted(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
})

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待确认',
    confirmed: '待完成',
    completed: '已完成',
    cancelled: '已取消'
  }
  return map[status] || status
}

const isOwnOrder = computed(() => {
  if (!userStore.user) return false
  // Stored/local rows carry no coserId but are still the user's own order.
  if (!order.value.coserId) return true
  return order.value.coserId === String(userStore.user.id)
})

const showRespondQuote = computed(() => (
  order.value.priceMode === 'negotiable' && order.value.priceStatus === 'quoted' && isOwnOrder.value
))

const showQuote = computed(() => (
  order.value.priceMode === 'negotiable' &&
  order.value.priceStatus === 'awaiting_quote' &&
  userStore.user?.photographerId != null &&
  order.value.photographerId === String(userStore.user.photographerId)
))

function acceptQuote() {
  if (!order.value.id) return
  confirmRespondQuote(order.value.id, true, loadOrder)
}

function rejectQuote() {
  if (!order.value.id) return
  confirmRespondQuote(order.value.id, false, loadOrder)
}

function quoteOrder() {
  if (!order.value.id) return
  promptQuote(order.value.id, loadOrder)
}

function goBack() {
  uni.navigateBack()
}

async function goChat() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/order/list' }), 800)
    return
  }
  if (!order.value.photographerId) {
    uni.showToast({ title: '订单信息缺失', icon: 'none' })
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(order.value.photographerUserId || order.value.photographerId)
    })
    uni.navigateTo({
      url: `/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(order.value.photographerName)}&peerAvatar=${encodeURIComponent(order.value.photographerAvatar || '')}`
    })
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}

async function updateStatus(newStatus: 'confirmed' | 'completed' | 'cancelled') {
  if (!order.value.id) return
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${order.value.id}/status`, { status: newStatus })
    order.value = { ...order.value, status: newStatus }
    uni.showToast({ title: '操作成功', icon: 'success' })
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function cancelOrder() {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => { if (res.confirm) updateStatus('cancelled') }
  })
}

function confirmOrder() {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => { if (res.confirm) updateStatus('completed') }
  })
}

function goReview() {
  uni.navigateTo({ url: `/pages/comment/index?photographerId=${order.value.photographerId}` })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
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
}

.back-icon {
  font-size: 48rpx;
  color: $dark-text-primary;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
}

.placeholder {
  width: 60rpx;
}

.content {
  height: calc(100vh - 140rpx - env(safe-area-inset-top));
}

.status-card {
  padding: $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: center;

  &.confirmed {
    background: $neon-gradient;
  }

  &.pending {
    background: $neon-cyan;
  }

  &.completed {
    background: $success-color;
  }

  &.cancelled {
    background: $dark-bg-secondary;
    border-bottom: 1rpx solid $dark-border;
  }
}

.status-icon {
  font-size: 80rpx;
  margin-bottom: $spacing-sm;
  color: rgba(255, 255, 255, 0.9);
}

.status-text {
  font-size: $font-size-xl;
  font-weight: 600;
  color: #fff;
}

.order-id {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
  margin-top: $spacing-xs;
}

.section {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  margin: $spacing-md;
  padding: $spacing-md;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.section-title {
  font-size: $font-size-base;
  font-weight: 600;
  margin-bottom: $spacing-md;
  color: $dark-text-primary;
  padding-left: 20rpx;
  position: relative;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 6rpx;
    height: 28rpx;
    background: $neon-gradient;
    border-radius: 3rpx;
  }
}

.photographer-info {
  display: flex;
  align-items: center;
}

.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: $border-radius-md;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name {
  font-size: $font-size-lg;
  font-weight: 500;
  display: block;
  color: $dark-text-primary;
}

.location {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  display: block;
  margin-top: 4rpx;
}

.btn-outline {
  padding: $spacing-sm $spacing-md;
  border: 2rpx solid $neon-purple;
  color: $neon-purple;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;

  &.danger {
    border-color: rgba(239, 68, 68, 0.4);
    color: $error-color;
  }

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.service-detail {
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;

  &:not(:last-child) {
    border-bottom: 1rpx solid $dark-border;
  }
}

.detail-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.detail-value {
  font-size: $font-size-sm;
  color: $dark-text-primary;
}

.price-detail {
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.price-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;

  &.total {
    border-top: 1rpx solid $dark-border;
    margin-top: $spacing-sm;
    padding-top: $spacing-md;
  }
}

.price-label {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
}

.price-value {
  font-size: $font-size-sm;
  color: $dark-text-primary;

  .total & {
    font-size: $font-size-lg;
    font-weight: 600;
    color: $neon-purple;
  }
}

.order-time {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.bottom-space {
  height: 140rpx;
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
  padding: $spacing-md;
  padding-bottom: calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $dark-bg-secondary;
  border-top: 1rpx solid $dark-border;
  box-shadow: 0 -4rpx 16rpx rgba(0, 0, 0, 0.2);
}

.btn-primary {
  padding: $spacing-sm $spacing-xl;
  background: $neon-gradient;
  @include on-neon-fill;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}
</style>
