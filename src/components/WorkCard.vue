<template>
  <view class="work-card" @click="goDetail">
    <image :src="work.images[0]" class="cover" mode="aspectFill" />
    <view class="overlay">
      <text class="title">{{ work.title }}</text>
    </view>
    <view class="tags">
      <text v-for="tag in work.tags.slice(0, 2)" :key="tag" class="tag">{{ tag }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import type { Work } from '@/types'

defineProps<{
  work: Work
}>()

function goDetail() {
  uni.showToast({
    title: '查看作品',
    icon: 'none'
  })
}
</script>

<style lang="scss" scoped>
.work-card {
  position: relative;
  border-radius: $border-radius-lg;
  overflow: hidden;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.25);
  transition: transform 0.15s, box-shadow 0.15s, border-color 0.15s;

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: $border-radius-lg;
    border: 1rpx solid rgba($neon-purple, 0.15);
    pointer-events: none;
    z-index: 2;
  }

  &:active {
    transform: scale(0.97);
    box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.15);
    border-color: $neon-purple-glow;
  }
}

.cover {
  width: 100%;
  height: 240rpx;
}

.overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: linear-gradient(transparent, rgba(10, 10, 26, 0.85));
  padding: $spacing-md $spacing-sm $spacing-sm;
}

.title {
  font-size: $font-size-sm;
  color: #fff;
  font-weight: 500;
}

.tags {
  display: flex;
  gap: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  background: rgba($neon-purple-dim, 0.8);
  border-top: 1rpx solid $neon-purple-glow;
}

.tag {
  font-size: $font-size-xs;
  color: $neon-cyan;
  padding: 2rpx 12rpx;
  background: rgba($neon-cyan, 0.12);
  border-radius: $border-radius-sm;
}
</style>
