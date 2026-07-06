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
      <view class="status-card">
        <text class="status-icon">📅</text>
        <text class="status-text">{{ getStatusText(order.status) }}</text>
        <text class="order-id">订单号：{{ order.id }}</text>
      </view>

      <view class="section">
        <view class="section-title">摄影师信息</view>
        <view class="photographer-info">
          <image :src="order.photographerAvatar" class="avatar" mode="aspectFill" />
          <view class="info">
            <text class="name">{{ order.photographerName }}</text>
            <text class="location">📍 {{ order.location }}</text>
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
            <text class="price-value">¥{{ order.totalPrice }}</text>
          </view>
          <view class="price-row total">
            <text class="price-label">总计</text>
            <text class="price-value">¥{{ order.totalPrice }}</text>
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
        v-if="order.status === 'pending'" 
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
import { ref, onMounted } from 'vue'

const statusBarHeight = ref(44)

const order = ref<any>({
  id: 'BK20240101001',
  photographerName: '光影行者',
  photographerAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=professional%20photographer%20avatar%20portrait%20studio%20lighting&image_size=square',
  location: '北京',
  serviceName: '进阶套餐',
  duration: 240,
  date: '2024-01-15',
  time: '14:00',
  remark: '希望能拍一些动感的动作',
  totalPrice: 699,
  status: 'pending',
  createTime: '2024-01-01 10:30:00'
})

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

function goBack() {
  uni.navigateBack()
}

function goChat() {
  uni.navigateTo({ url: '/pages/chat/index' })
}

function cancelOrder() {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => {
      if (res.confirm) {
        uni.showToast({ title: '已取消', icon: 'success' })
        setTimeout(() => {
          uni.navigateBack()
        }, 1500)
      }
    }
  })
}

function confirmOrder() {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => {
      if (res.confirm) {
        uni.showToast({ title: '已确认', icon: 'success' })
        setTimeout(() => {
          uni.navigateBack()
        }, 1500)
      }
    }
  })
}

function goReview() {
  uni.navigateTo({ url: '/pages/comment/index' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-sm $spacing-md;
  background: $bg-primary;
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
  color: $text-primary;
}

.header-title {
  font-size: $font-size-lg;
  font-weight: 600;
}

.placeholder {
  width: 60rpx;
}

.content {
  height: calc(100vh - 140rpx - env(safe-area-inset-top));
}

.status-card {
  background: $primary-color;
  padding: $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.status-icon {
  font-size: 80rpx;
  margin-bottom: $spacing-sm;
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
  background: $bg-primary;
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-title {
  font-size: $font-size-base;
  font-weight: 600;
  margin-bottom: $spacing-md;
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
}

.location {
  font-size: $font-size-sm;
  color: $text-secondary;
  display: block;
  margin-top: 4rpx;
}

.btn-outline {
  padding: $spacing-sm $spacing-md;
  border: 2rpx solid $primary-color;
  color: $primary-color;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
}

.service-detail {
  background: $bg-secondary;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;
  
  &:not(:last-child) {
    border-bottom: 1rpx solid $border-color;
  }
}

.detail-label {
  font-size: $font-size-sm;
  color: $text-secondary;
}

.detail-value {
  font-size: $font-size-sm;
  color: $text-primary;
}

.price-detail {
  background: $bg-secondary;
  border-radius: $border-radius-md;
  padding: $spacing-sm;
}

.price-row {
  display: flex;
  justify-content: space-between;
  padding: $spacing-sm;
  
  &.total {
    border-top: 1rpx solid $border-color;
    margin-top: $spacing-sm;
    padding-top: $spacing-md;
  }
}

.price-label {
  font-size: $font-size-sm;
  color: $text-secondary;
}

.price-value {
  font-size: $font-size-sm;
  color: $text-primary;
  
  .total & {
    font-size: $font-size-lg;
    font-weight: 600;
    color: $primary-color;
  }
}

.order-time {
  font-size: $font-size-sm;
  color: $text-tertiary;
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
  background: $bg-primary;
  box-shadow: 0 -2rpx 10rpx rgba(0, 0, 0, 0.05);
}

.btn-primary {
  padding: $spacing-sm $spacing-xl;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
}
</style>
