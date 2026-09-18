<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">接单管理</text>
    </view>

    <view class="body">
      <view v-if="!userStore.user?.photographerId" class="empty-state">
        <text class="empty-icon">◈</text>
        <text class="empty-text">请先开通摄影师身份</text>
        <view class="btn-primary" @click="goActivate">去开通</view>
      </view>
      <view v-else-if="loading" class="empty-hint">加载中...</view>
      <view v-else-if="orders.length === 0" class="empty-state">
        <text class="empty-icon">◇</text>
        <text class="empty-text">暂无收到预约</text>
      </view>
      <view v-else class="order-list">
        <view v-for="order in orders" :key="order.id" class="order-card">
          <view class="order-header">
            <text class="order-date">{{ order.date }} {{ order.time }}</text>
            <text class="order-status" :class="`status-${order.status}`">{{ getStatusText(order.status) }}</text>
          </view>

          <view class="order-content">
            <image :src="order.coserAvatar || ''" class="coser-avatar" mode="aspectFill" />
            <view class="order-info">
              <text class="coser-name">{{ order.coserName || `Coser${order.coserId}` }}</text>
              <text class="coser-phone">电话：{{ order.coserPhone || '未留电话' }}</text>
              <text class="service-name">{{ order.serviceName || '服务' }} · {{ renderOrderPrice(order) }}</text>
            </view>
          </view>

          <view v-if="order.remarks" class="order-remark">备注：{{ order.remarks }}</view>

          <view class="order-footer">
            <view v-if="order.status === 'pending'" class="btn-outline danger" @click="rejectOrder(order.id)">拒绝</view>
            <view v-if="order.status === 'pending' && !isNegotiablePending(order)" class="btn-primary" @click="updateStatus(order.id, 'confirmed')">确认接单</view>
            <view v-if="canQuote(order)" class="btn-primary" @click="quoteOrder(order.id)">报价</view>
            <view v-if="order.status === 'confirmed'" class="btn-primary" @click="updateStatus(order.id, 'completed')">完成拍摄</view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { apiGet, apiPut, ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'
import type { OrderPriceFields } from '@/types'
import { renderOrderPrice, toPriceMode, toPriceStatus, promptQuote } from '@/utils/quote'

interface PhotographerOrder extends OrderPriceFields {
  id: number
  photographerId: number
  coserId: number
  serviceId: number
  date: string
  time: string
  status: string
  remarks?: string
  serviceName?: string
  coserName?: string
  coserAvatar?: string
  coserPhone?: string
}

const statusBarHeight = ref(44)
const loading = ref(true)
const orders = ref<PhotographerOrder[]>([])
const userStore = useUserStore()

onShow(() => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  loadOrders()
})

async function loadOrders() {
  if (!userStore.user?.photographerId) {
    orders.value = []
    loading.value = false
    return
  }
  try {
    const res = await apiGet<PhotographerOrder[]>(`/v1/bookings/photographer/${userStore.user.photographerId}`)
    orders.value = (res || []).map((o) => ({
      ...o,
      priceMode: toPriceMode(o.priceMode),
      priceStatus: toPriceStatus(o.priceStatus),
      quotePrice: o.quotePrice ?? null,
    }))
  } catch {
    orders.value = []
  } finally {
    loading.value = false
  }
}

function isNegotiablePending(order: PhotographerOrder): boolean {
  return order.priceMode === 'negotiable' && (order.priceStatus === 'awaiting_quote' || order.priceStatus === 'quoted')
}

function canQuote(order: PhotographerOrder): boolean {
  return order.priceMode === 'negotiable' && order.priceStatus === 'awaiting_quote'
}

function quoteOrder(id: number) {
  promptQuote(id, loadOrders)
}

async function updateStatus(id: number, status: string) {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${id}/status`, { status, actorTag: 'photographer' })
    uni.hideLoading()
    uni.showToast({ title: '操作成功', icon: 'success' })
    loadOrders()
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  }
}

function rejectOrder(id: number) {
  uni.showModal({
    title: '拒绝预约',
    content: '确定要拒绝这个预约吗？',
    success: (res) => {
      if (res.confirm) updateStatus(id, 'cancelled')
    }
  })
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

function goActivate() {
  uni.navigateTo({ url: '/pages/photographer/activate' })
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

.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $neon-cyan; text-shadow: 0 0 24rpx $neon-cyan-glow; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0 32rpx; }
.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }

.order-list { display: flex; flex-direction: column; gap: 24rpx; }
.order-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; padding: 28rpx; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.2); }
.order-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20rpx; }
.order-date { font-size: 26rpx; color: $dark-text-secondary; }
.order-status { font-size: 24rpx; font-weight: 500;
  &.status-pending { color: $neon-cyan; }
  &.status-confirmed { color: $neon-purple; }
  &.status-completed { color: $success-color; }
  &.status-cancelled { color: $dark-text-tertiary; }
}
.order-content { display: flex; align-items: center; }
.coser-avatar { width: 100rpx; height: 100rpx; border-radius: 50%; border: 2rpx solid $neon-purple-glow; background: $dark-bg-secondary; flex-shrink: 0; }
.order-info { flex: 1; margin-left: 20rpx; overflow: hidden; }
.coser-name { display: block; font-size: 30rpx; font-weight: 600; color: $dark-text-primary; }
.coser-phone { display: block; font-size: 24rpx; color: $dark-text-secondary; margin-top: 6rpx; }
.service-name { display: block; font-size: 24rpx; color: $neon-cyan; margin-top: 6rpx; }
.order-remark { font-size: 22rpx; color: $dark-text-tertiary; margin-top: 16rpx; padding-top: 16rpx; border-top: 1rpx solid $dark-border; }
.order-footer { display: flex; justify-content: flex-end; gap: 20rpx; margin-top: 24rpx; }

.btn-outline { padding: 14rpx 36rpx; border: 2rpx solid $dark-border; border-radius: 32rpx; font-size: 26rpx; color: $dark-text-secondary;
  &.danger { color: $error-color; border-color: rgba(239, 68, 68, 0.4); }
  &:active { transform: scale(0.97); background: $dark-bg-card-hover; }
}
.btn-primary { padding: 14rpx 36rpx; background: $neon-purple; color: #fff; border-radius: 32rpx; font-size: 26rpx; box-shadow: 0 0 16rpx $neon-purple-glow;
  &:active { transform: scale(0.97); box-shadow: 0 0 12rpx $neon-purple-glow; }
}
</style>
