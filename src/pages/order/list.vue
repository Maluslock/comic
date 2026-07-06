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
        <text class="empty-icon">📋</text>
        <text class="empty-text">暂无订单</text>
        <view class="empty-btn" @click="goHome">去逛逛</view>
      </view>

      <view v-else class="order-list">
        <view v-for="order in orders" :key="order.id" class="order-card">
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

          <view class="order-footer">
            <view 
              v-if="order.status === 'pending'" 
              class="btn-outline"
              @click="cancelOrder(order.id)"
            >
              取消预约
            </view>
            <view 
              v-if="order.status === 'confirmed'" 
              class="btn-primary"
              @click="confirmOrder(order.id)"
            >
              确认完成
            </view>
            <view 
              v-if="order.status === 'completed'" 
              class="btn-outline"
              @click="goReview(order.id)"
            >
              去评价
            </view>
            <view 
              v-if="order.status === 'pending' || order.status === 'confirmed'" 
              class="btn-primary"
              @click="goChat(order.id)"
            >
              联系摄影师
            </view>
          </view>
        </view>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const currentTab = ref('all')
const orders = ref<any[]>([])

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待确认' },
  { key: 'confirmed', label: '待完成' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' }
]

onMounted(() => {
  loadOrders()
})

function loadOrders() {
  orders.value = [
    {
      id: 'BK20240101001',
      photographerName: '光影行者',
      photographerAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=professional%20photographer%20avatar%20portrait%20studio%20lighting&image_size=square',
      serviceName: '进阶套餐',
      date: '2024-01-15',
      time: '14:00',
      totalPrice: 699,
      status: 'pending'
    },
    {
      id: 'BK20240101002',
      photographerName: '古风公子',
      photographerAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=chinese%20ancient%20style%20photographer%20avatar%20elegant&image_size=square',
      serviceName: '精品套餐',
      date: '2024-01-20',
      time: '10:00',
      totalPrice: 1299,
      status: 'confirmed'
    },
    {
      id: 'BK20240101003',
      photographerName: '樱花落',
      photographerAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=female%20photographer%20avatar%20pink%20hair%20cute%20style&image_size=square',
      serviceName: '基础套餐',
      date: '2024-01-10',
      time: '15:00',
      totalPrice: 399,
      status: 'completed'
    }
  ]
}

function setTab(key: string) {
  currentTab.value = key
  if (key === 'all') {
    loadOrders()
  } else {
    orders.value = orders.value.filter(o => o.status === key)
  }
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

function cancelOrder(id: string) {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => {
      if (res.confirm) {
        uni.showToast({ title: '已取消', icon: 'success' })
        loadOrders()
      }
    }
  })
}

function confirmOrder(id: string) {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => {
      if (res.confirm) {
        uni.showToast({ title: '已确认', icon: 'success' })
        loadOrders()
      }
    }
  })
}

function goReview(id: string) {
  uni.navigateTo({ url: '/pages/comment/index' })
}

function goChat(id: string) {
  uni.navigateTo({ url: '/pages/chat/index' })
}

function goHome() {
  uni.switchTab({ url: '/pages/index/index' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.tab-bar {
  display: flex;
  background: $bg-primary;
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
  color: $text-secondary;
  
  &.active {
    color: $primary-color;
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
  background: $primary-color;
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
  margin-bottom: $spacing-md;
}

.empty-text {
  font-size: $font-size-base;
  color: $text-tertiary;
  margin-bottom: $spacing-lg;
}

.empty-btn {
  padding: $spacing-sm $spacing-xl;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
}

.order-list {
  padding: $spacing-md;
}

.order-card {
  background: $bg-primary;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  margin-bottom: $spacing-md;
}

.order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
}

.order-id {
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.order-status {
  font-size: $font-size-sm;
  font-weight: 500;
  
  &.status-pending {
    color: $warning-color;
  }
  
  &.status-confirmed {
    color: $primary-color;
  }
  
  &.status-completed {
    color: $success-color;
  }
  
  &.status-cancelled {
    color: $text-tertiary;
  }
}

.order-content {
  display: flex;
  align-items: center;
  padding-bottom: $spacing-md;
  border-bottom: 1rpx solid $border-color;
}

.photographer-avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: $border-radius-md;
}

.order-info {
  flex: 1;
  margin-left: $spacing-md;
}

.photographer-name {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
}

.service-name {
  font-size: $font-size-sm;
  color: $text-secondary;
  display: block;
  margin-top: 4rpx;
}

.order-time {
  font-size: $font-size-xs;
  color: $text-tertiary;
  display: block;
  margin-top: 4rpx;
}

.order-price {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $primary-color;
}

.order-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
  margin-top: $spacing-md;
}

.btn-outline {
  padding: $spacing-sm $spacing-lg;
  border: 2rpx solid $border-color;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  color: $text-secondary;
}

.btn-primary {
  padding: $spacing-sm $spacing-lg;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
}

.bottom-space {
  height: 60rpx;
}
</style>
