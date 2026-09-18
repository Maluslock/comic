<template>
  <view class="page">
    <view class="tab-bar">
      <view 
        v-for="tab in tabs" 
        :key="tab.key" 
        class="tab-item"
        :class="{ active: currentTab === tab.key }"
        @click="setTab(tab.key)"
      >
        {{ tab.label }}
        <view v-if="tab.key === currentTab" class="tab-indicator"></view>
      </view>
    </view>

    <scroll-view scroll-y class="content">
      <view v-if="orders.length === 0" class="empty">
        <text class="empty-icon">◇</text>
        <text class="empty-text">暂无订单</text>
        <view class="empty-btn" @click="goHome">去逛逛</view>
      </view>

      <view v-else class="order-list">
        <view v-for="order in orders" :key="order.id" class="order-card" @click="goDetail(order.id)">
          <view class="order-header">
            <text class="order-id">订单号：{{ order.id }}</text>
            <text class="order-status" :class="getStatusClass(order.status)">
              {{ getStatusText(order.status) }}
            </text>
          </view>

          <view class="order-content">
            <image :src="order.photographerAvatar" class="photographer-avatar" mode="aspectFill" />
            <view class="order-info">
              <text class="photographer-name">{{ order.photographerName }}</text>
              <text class="service-name">{{ order.serviceName }}</text>
              <text class="order-time">{{ order.date }} {{ order.time }}</text>
            </view>
            <text class="order-price">¥{{ order.totalPrice }}</text>
          </view>

          <view class="order-footer" @click.stop>
            <view v-if="order.status === 'pending' || order.status === 'confirmed'" class="btn-outline" @click="cancelOrder(order.id)">取消预约</view>
            <view v-if="order.status === 'confirmed'" class="btn-primary" @click="confirmOrder(order.id)">确认完成</view>
            <view v-if="order.status === 'completed'" class="btn-outline" @click="goReview(order)">去评价</view>
            <view v-if="order.status === 'pending' || order.status === 'confirmed'" class="btn-primary" @click="goChat(order.id)">联系摄影师</view>
          </view>
        </view>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiGet, apiPost, apiPut, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

interface DisplayOrder {
  id: string
  photographerId?: string
  photographerUserId?: number
  photographerName: string
  photographerAvatar: string
  location: string
  serviceId?: string
  serviceName: string
  duration?: number
  date: string
  time: string
  totalPrice: number
  status: string
  remark: string
  createTime: string | number
}

const currentTab = ref('all')
const fullOrders = ref<DisplayOrder[]>([])
const orders = ref<DisplayOrder[]>([])

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待确认' },
  { key: 'confirmed', label: '待完成' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' }
]

const userStore = useUserStore()

onMounted(() => {
  loadBookings()
})

function readStoredBookings(): any[] {
  const storedJson = uni.getStorageSync('bookings') || '[]'
  try {
    return JSON.parse(storedJson)
  } catch {
    return []
  }
}

async function loadBookings() {
  if (!userStore.isLoggedIn || !userStore.user) {
    fullOrders.value = readStoredBookings()
    applyFilter()
    return
  }

  try {
    const res = await apiGet<any[]>(`/v1/bookings/${userStore.user.id}`)
    const stored = readStoredBookings()

    const serverOrders = (res || []).map((b: any) => ({
      id: String(b.id),
      photographerId: String(b.photographerId),
      photographerUserId: b.photographerUserId,
      photographerName: b.photographerName || `摄影师${b.photographerId}`,
      photographerAvatar: b.photographerAvatar || '',
      location: b.location || '',
      serviceId: String(b.serviceId),
      serviceName: b.serviceName || `服务${b.serviceId}`,
      date: b.date,
      time: b.time,
      totalPrice: b.totalPrice || 0,
      status: b.status || 'pending',
      remark: b.remarks || '',
      createTime: b.createdAt || Date.now()
    }))

    const storedIds = new Set(serverOrders.map(o => o.id))
    // purge legacy local orders ("booking_4" etc.) — they have no server record
    // and would only confuse with read-only cards; server is the single source
    const stripBookingPrefix = (id: string) => id.replace(/^booking_/, '')
    const legacyLocal = stored.filter((o: any) => storedIds.has(stripBookingPrefix(String(o.id))) === false && /^booking_/.test(String(o.id)))
    const keptLocal = stored.filter((o: any) => !/^booking_/.test(String(o.id)))
    if (legacyLocal.length) {
      uni.setStorageSync('bookings', JSON.stringify(keptLocal))
    }
    fullOrders.value = [...serverOrders, ...keptLocal]
  } catch {
    fullOrders.value = readStoredBookings()
  }

  applyFilter()
}

function applyFilter() {
  if (currentTab.value === 'all') {
    orders.value = [...fullOrders.value]
  } else {
    orders.value = fullOrders.value.filter(o => o.status === currentTab.value)
  }
}

function setTab(key: string) {
  currentTab.value = key
  applyFilter()
}

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待确认',
    confirmed: '待完成',
    completed: '已完成',
    cancelled: '已取消'
  }
  return map[status] || status
}

function getStatusClass(status: string): string {
  return `status-${status}`
}

async function updateStatus(id: string, newStatus: 'confirmed' | 'completed' | 'cancelled') {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${id}/status`, { status: newStatus })
    uni.hideLoading()
    uni.showToast({ title: '操作成功', icon: 'success' })
    loadBookings()
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  }
}

function cancelOrder(id: string) {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => { if (res.confirm) updateStatus(id, 'cancelled') }
  })
}

function confirmOrder(id: string) {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => { if (res.confirm) updateStatus(id, 'completed') }
  })
}

function goDetail(id: string) {
  uni.navigateTo({ url: `/pages/order/detail?id=${id}` })
}

function goReview(order: any) {
  uni.navigateTo({ url: `/pages/comment/index?photographerId=${order.photographerId}` })
}

async function goChat(id: string) {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/order/list' }), 800)
    return
  }
  const order = fullOrders.value.find(o => o.id === id)
  if (!order) {
    uni.showToast({ title: '订单不存在', icon: 'none' })
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(order.photographerUserId || order.photographerId)
    })
    uni.navigateTo({
      url: `/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(order.photographerName)}&peerAvatar=${encodeURIComponent(order.photographerAvatar || '')}`
    })
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}

function goHome() {
  uni.switchTab({ url: '/pages/index/index' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.tab-bar {
  display: flex;
  background: $dark-bg-card;
  border-bottom: 1rpx solid $dark-border;
  padding-top: env(safe-area-inset-top);
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-md;
  position: relative;
  font-size: $font-size-sm;
  color: $dark-text-secondary;

  &.active {
    color: $neon-purple;
    font-weight: 500;
  }
}

.tab-indicator {
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 48rpx;
  height: 6rpx;
  background: $neon-gradient;
  border-radius: 3rpx;
}

.content {
  height: calc(100vh - env(safe-area-inset-top) - 100rpx);
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 3;
}

.empty-icon {
  font-size: 100rpx;
  color: $neon-purple;
  margin-bottom: $spacing-md;
}

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-tertiary;
  margin-bottom: $spacing-lg;
}

.empty-btn {
  padding: $spacing-sm $spacing-xl;
  background: $neon-purple;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
  box-shadow: 0 0 20rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.order-list {
  padding: $spacing-md;
}

.order-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  margin-bottom: $spacing-md;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2);

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
}

.order-id {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.order-status {
  font-size: $font-size-sm;
  font-weight: 500;

  &.status-pending {
    color: $neon-cyan;
  }

  &.status-confirmed {
    color: $neon-purple;
  }

  &.status-completed {
    color: $success-color;
  }

  &.status-cancelled {
    color: $dark-text-tertiary;
  }
}

.order-content {
  display: flex;
  align-items: center;
  padding-bottom: $spacing-md;
  border-bottom: 1rpx solid $dark-border;
}

.photographer-avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: $border-radius-md;
  border: 2rpx solid $dark-border;
}

.order-info {
  flex: 1;
  margin-left: $spacing-md;
}

.photographer-name {
  font-size: $font-size-base;
  font-weight: 500;
  color: $dark-text-primary;
  display: block;
}

.service-name {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  display: block;
  margin-top: 4rpx;
}

.order-time {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  display: block;
  margin-top: 4rpx;
}

.order-price {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $neon-purple;
}

.order-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
  margin-top: $spacing-md;
}

.btn-outline {
  padding: $spacing-sm $spacing-lg;
  border: 2rpx solid $dark-border;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: $dark-text-secondary;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.btn-primary {
  padding: $spacing-sm $spacing-lg;
  background: $neon-purple;
  color: #fff;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  box-shadow: 0 0 16rpx $neon-purple-glow;

  &:active {
    transform: scale(0.97);
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

.bottom-space {
  height: 60rpx;
}
</style>
