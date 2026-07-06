<template>
  <view class="photographer-card" @click="goDetail">
    <image :src="photographer.avatar" class="avatar" mode="aspectFill" />
    <view class="info">
      <view class="header">
        <text class="name">{{ photographer.name }}</text>
        <view class="rating">
          <text class="star">★</text>
          <text class="score">{{ photographer.rating }}</text>
        </view>
      </view>
      <text class="desc">{{ photographer.description }}</text>
      <view class="tags">
        <text 
          v-for="tag in photographer.tags.slice(0, 3)" 
          :key="tag" 
          class="tag"
        >
          {{ tag }}
        </text>
      </view>
      <view class="footer">
        <text class="location">{{ photographer.location }}</text>
        <text class="count">{{ photographer.orderCount }}单已接</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import type { Photographer } from '@/types'

const props = defineProps<{
  photographer: Photographer
}>()

function goDetail() {
  uni.navigateTo({
    url: `/pages/photographer/detail?id=${props.photographer.id}`
  })
}
</script>

<style lang="scss" scoped>
.photographer-card {
  display: flex;
  background: $bg-primary;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
  margin-bottom: $spacing-md;
  box-shadow: $shadow-md;
  border-left: 6rpx solid $primary-color;
  transition: transform 0.15s, box-shadow 0.15s;
  
  &:active {
    background: $bg-secondary;
    transform: scale(0.98);
    box-shadow: $shadow-sm;
  }
}

.avatar {
  width: 144rpx;
  height: 144rpx;
  border-radius: 50%;
  flex-shrink: 0;
  border: 4rpx solid rgba($primary-color, 0.15);
}

.info {
  flex: 1;
  margin-left: $spacing-md;
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.name {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $text-primary;
}

.rating {
  display: flex;
  align-items: center;
}

.star {
  color: $warning-color;
  font-size: $font-size-base;
}

.score {
  color: $warning-color;
  font-size: $font-size-sm;
  margin-left: 4rpx;
}

.desc {
  font-size: $font-size-sm;
  color: $text-secondary;
  margin-top: $spacing-xs;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
  margin-top: $spacing-sm;
}

.tag {
  padding: 4rpx 12rpx;
  background: rgba($primary-color, 0.08);
  color: $primary-color;
  font-size: $font-size-xs;
  border-radius: $border-radius-sm;
}

.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: $spacing-sm;
}

.location {
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.count {
  font-size: $font-size-xs;
  color: $text-tertiary;
}
</style>
