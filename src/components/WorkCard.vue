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
  box-shadow: $shadow-sm;
  transition: transform 0.15s, box-shadow 0.15s;

  &:active {
    transform: scale(0.97);
    box-shadow: none;
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
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.6));
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
  background: rgba(0, 0, 0, 0.4);
}

.tag {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
}
</style>
