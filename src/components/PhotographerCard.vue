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
      <text v-if="photographer.description" class="desc">{{ photographer.description }}</text>
      <view class="tags" v-if="photographer.tags.length">
        <text
          v-for="tag in photographer.tags.slice(0, 3)"
          :key="tag"
          class="tag"
        >{{ tag }}</text>
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
  background: $dark-bg-card;
  border-radius: $border-radius-lg;
  padding: $spacing-sm;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.3);
  border-left: 4rpx solid $neon-purple;
  transition: transform 0.15s, box-shadow 0.15s;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.98);
    box-shadow: 0 2rpx 10rpx rgba(0, 0, 0, 0.2);
  }
}

.avatar {
  width: 100rpx;
  height: 100rpx;
  border-radius: 50%;
  flex-shrink: 0;
  border: 3rpx solid rgba($neon-purple, 0.3);
}

.info {
  flex: 1;
  margin-left: $spacing-sm;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.name {
  display: block;
  font-size: $font-size-md;
  font-weight: 600;
  color: $dark-text-primary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: $spacing-xs;
}

.rating {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.star {
  color: $warning-color;
  font-size: 24rpx;
}

.score {
  color: $warning-color;
  font-size: 22rpx;
  margin-left: 2rpx;
}

.desc {
  font-size: 22rpx;
  color: $dark-text-secondary;
  margin-top: 4rpx;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4rpx;
  margin-top: 6rpx;
}

.tag {
  padding: 2rpx 10rpx;
  background: rgba($neon-purple, 0.15);
  color: $neon-purple;
  font-size: 20rpx;
  border-radius: $border-radius-sm;
}

.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: 6rpx;
}

.location {
  font-size: 20rpx;
  color: $dark-text-tertiary;
}

.count {
  font-size: 20rpx;
  color: $dark-text-tertiary;
}
</style>
