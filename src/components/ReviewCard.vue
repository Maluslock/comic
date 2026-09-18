<template>
  <view class="review-card">
    <view class="header">
      <image :src="review.userAvatar" class="avatar" mode="aspectFill" />
      <view class="user-info">
        <text class="user-name">{{ review.userName }}</text>
        <view class="rating">
          <text 
            v-for="i in 5" 
            :key="i" 
            class="star"
            :class="{ active: i <= review.rating }"
          >★</text>
        </view>
      </view>
      <text class="time">{{ formatTime(review.createdAt) }}</text>
    </view>
    <text class="content">{{ review.content }}</text>
    <view v-if="review.images?.length" class="images">
      <image 
        v-for="(img, index) in review.images" 
        :key="index" 
        :src="img" 
        class="image"
        mode="aspectFill"
      />
    </view>
  </view>
</template>

<script setup lang="ts">
import type { Review } from '@/types'

defineProps<{
  review: Review
}>()

function formatTime(timestamp: number): string {
  const now = Date.now()
  const diff = now - timestamp
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`
  
  const date = new Date(timestamp)
  return `${date.getMonth() + 1}月${date.getDate()}日`
}
</script>

<style lang="scss" scoped>
.review-card {
  background: $dark-bg-card;
  border-radius: $border-radius-md;
  padding: $spacing-md;
  margin-bottom: $spacing-md;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.25);
  border: 1rpx solid $dark-border;
}

.header {
  display: flex;
  align-items: center;
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
}

.user-info {
  flex: 1;
  margin-left: $spacing-sm;
}

.user-name {
  font-size: $font-size-base;
  font-weight: 500;
  color: $dark-text-primary;
}

.rating {
  display: flex;
  margin-top: 4rpx;
}

.star {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;

  &.active {
    color: $warning-color;
  }
}

.time {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.content {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-top: $spacing-sm;
  line-height: 1.6;
}

.images {
  display: flex;
  gap: $spacing-xs;
  margin-top: $spacing-sm;
}

.image {
  width: 160rpx;
  height: 160rpx;
  border-radius: $border-radius-sm;
}
</style>
