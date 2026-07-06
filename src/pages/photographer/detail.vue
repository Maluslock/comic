<template>
  <view class="page">
    <scroll-view scroll-y class="content">
      <view class="header">
        <image :src="photographer?.avatar" class="avatar" mode="aspectFill" />
        <view class="info">
          <text class="name">{{ photographer?.name }}</text>
          <view class="stats">
            <view class="stat-item">
              <text class="stat-value">{{ photographer?.rating }}</text>
              <text class="stat-label">评分</text>
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <text class="stat-value">{{ photographer?.reviewCount }}</text>
              <text class="stat-label">评价</text>
            </view>
            <view class="stat-divider"></view>
            <view class="stat-item">
              <text class="stat-value">{{ photographer?.orderCount }}</text>
              <text class="stat-label">接单</text>
            </view>
          </view>
          <view class="location">
            <text class="location-icon">📍</text>
            <text>{{ photographer?.location }}</text>
          </view>
        </view>
      </view>

      <view class="tags-row">
        <text 
          v-for="tag in photographer?.tags" 
          :key="tag" 
          class="tag"
        >{{ tag }}</text>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">关于我</text>
        </view>
        <text class="description">{{ photographer?.description }}</text>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">服务套餐</text>
        </view>
        <view class="service-list">
          <view 
            v-for="service in photographer?.services" 
            :key="service.id" 
            class="service-item"
            @click="selectService(service)"
          >
            <view class="service-info">
              <text class="service-name">{{ service.name }}</text>
              <text class="service-desc">{{ service.description }}</text>
              <text class="service-duration">{{ service.duration }}分钟</text>
            </view>
            <view class="service-price">
              <text class="price">¥{{ service.price }}</text>
              <text class="btn">选择</text>
            </view>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">作品展示</text>
          <text class="section-more" @click="goPortfolio">查看全部</text>
        </view>
        <scroll-view scroll-x class="work-scroll" show-scrollbar="false">
          <view class="work-list">
            <view 
              v-for="work in photographer?.works" 
              :key="work.id" 
              class="work-item"
            >
              <image :src="work.images[0]" class="work-image" mode="aspectFill" />
              <text class="work-title">{{ work.title }}</text>
            </view>
          </view>
        </scroll-view>
      </view>

      <view class="section">
        <view class="section-header">
          <text class="section-title">用户评价</text>
          <text class="section-more" @click="goReviews">查看全部</text>
        </view>
        <view class="review-list">
          <ReviewCard 
            v-for="review in photographer?.reviews" 
            :key="review.id" 
            :review="review"
          />
        </view>
      </view>

      <view class="bottom-space"></view>
    </scroll-view>

    <view class="footer">
      <view class="footer-left">
        <view class="footer-btn" @click="goChat">
          <text class="footer-icon">💬</text>
          <text>聊天</text>
        </view>
        <view class="footer-btn" @click="collect">
          <text class="footer-icon">{{ collected ? '❤️' : '🤍' }}</text>
          <text>收藏</text>
        </view>
      </view>
      <view class="footer-right">
        <view class="btn-primary" @click="goBooking">立即预约</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ReviewCard from '@/components/ReviewCard.vue'
import { getPhotographerById } from '@/api/index'
import type { Photographer, Service } from '@/types'

const photographer = ref<Photographer | null>(null)
const collected = ref(false)

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  const id = currentPage?.options?.id || '1'
  loadData(id)
})

async function loadData(id: string) {
  uni.showLoading({ title: '加载中...' })
  try {
    photographer.value = await getPhotographerById(id)
  } finally {
    uni.hideLoading()
  }
}

function selectService(service: Service) {
  uni.setStorageSync('selectedService', JSON.stringify(service))
  uni.navigateTo({ url: '/pages/booking/index' })
}

function goChat() {
  uni.navigateTo({ url: '/pages/chat/index' })
}

function collect() {
  collected.value = !collected.value
  uni.showToast({
    title: collected.value ? '已收藏' : '取消收藏',
    icon: 'none'
  })
}

function goBooking() {
  uni.navigateTo({ url: '/pages/booking/index' })
}

function goPortfolio() {
  uni.navigateTo({ url: '/pages/portfolio/index' })
}

function goReviews() {
  uni.navigateTo({ url: '/pages/comment/index' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.content {
  height: 100vh;
  padding-bottom: 140rpx;
}

.header {
  display: flex;
  background: $bg-primary;
  padding: $spacing-lg $spacing-md;
}

.avatar {
  width: 180rpx;
  height: 180rpx;
  border-radius: $border-radius-lg;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name {
  font-size: $font-size-xl;
  font-weight: 600;
}

.stats {
  display: flex;
  align-items: center;
  margin-top: $spacing-sm;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.stat-value {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $primary-color;
}

.stat-label {
  font-size: $font-size-xs;
  color: $text-tertiary;
  margin-top: 4rpx;
}

.stat-divider {
  width: 2rpx;
  height: 48rpx;
  background: $border-color;
}

.location {
  display: flex;
  align-items: center;
  margin-top: $spacing-sm;
  font-size: $font-size-sm;
  color: $text-secondary;
}

.location-icon {
  font-size: $font-size-sm;
  margin-right: 4rpx;
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
  padding: $spacing-sm $spacing-md;
  background: $bg-primary;
  border-bottom: 1rpx solid $border-color;
}

.tag {
  padding: 6rpx 16rpx;
  background: rgba($primary-color, 0.08);
  color: $primary-color;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
}

.section {
  background: $bg-primary;
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
}

.section-more {
  font-size: $font-size-sm;
  color: $text-tertiary;
}

.description {
  font-size: $font-size-sm;
  color: $text-secondary;
  line-height: 1.6;
}

.service-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.service-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md;
  background: $bg-secondary;
  border-radius: $border-radius-md;
}

.service-info {
  flex: 1;
}

.service-name {
  font-size: $font-size-base;
  font-weight: 500;
}

.service-desc {
  font-size: $font-size-xs;
  color: $text-secondary;
  margin-top: 4rpx;
}

.service-duration {
  font-size: $font-size-xs;
  color: $text-tertiary;
  margin-top: 4rpx;
}

.service-price {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.price {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $primary-color;
}

.btn {
  padding: 8rpx 24rpx;
  background: $primary-color;
  color: #fff;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
  margin-top: $spacing-xs;
}

.work-scroll {
  white-space: nowrap;
}

.work-list {
  display: inline-flex;
  gap: $spacing-sm;
}

.work-item {
  width: 200rpx;
}

.work-image {
  width: 200rpx;
  height: 280rpx;
  border-radius: $border-radius-md;
}

.work-title {
  font-size: $font-size-xs;
  color: $text-secondary;
  margin-top: $spacing-xs;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.review-list {
  margin-top: $spacing-sm;
}

.bottom-space {
  height: $spacing-xl;
}

.footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding: $spacing-sm $spacing-md;
  padding-bottom: calc(#{$spacing-sm} + env(safe-area-inset-bottom));
  background: $bg-primary;
  box-shadow: 0 -2rpx 10rpx rgba(0, 0, 0, 0.05);
}

.footer-left {
  display: flex;
  gap: $spacing-lg;
}

.footer-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: $font-size-xs;
  color: $text-secondary;
}

.footer-icon {
  font-size: $font-size-lg;
}

.footer-right {
  flex: 1;
  margin-left: $spacing-md;
}

.btn-primary {
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  padding: $spacing-sm;
  text-align: center;
  font-size: $font-size-base;
  font-weight: 500;
}
</style>
