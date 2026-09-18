<template>
  <view class="service-card" :class="{ active: selected }" @click="$emit('select', service)">
    <view class="checkbox" :class="{ checked: selected }">
      <text v-if="selected" class="check-icon">✓</text>
    </view>
    <view class="info">
      <view class="header">
        <text class="name">{{ service.name }}</text>
        <text class="price">¥{{ service.price }}</text>
      </view>
      <text class="desc">{{ service.description }}</text>
      <text class="duration">{{ service.duration }}分钟</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import type { Service } from '@/types'

defineProps<{
  service: Service
  selected?: boolean
}>()

defineEmits<{
  select: [service: Service]
}>()
</script>

<style lang="scss" scoped>
.service-card {
  display: flex;
  align-items: flex-start;
  background: $dark-bg-card;
  border-radius: $border-radius-md;
  padding: $spacing-md;
  margin-bottom: $spacing-sm;
  border: 2rpx solid transparent;
  box-shadow: 0 4rpx 20rpx rgba(0, 0, 0, 0.2);

  &.active {
    border-color: $neon-purple-glow;
    background: $neon-purple-dim;
    box-shadow: 0 0 16rpx $neon-purple-glow;
  }
}

.checkbox {
  width: 40rpx;
  height: 40rpx;
  border: 2rpx solid $dark-border;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-right: $spacing-md;

  &.checked {
    background: $neon-purple;
    border-color: $neon-purple;
  }
}

.check-icon {
  color: #fff;
  font-size: $font-size-sm;
}

.info {
  flex: 1;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.name {
  font-size: $font-size-base;
  font-weight: 600;
  color: $dark-text-primary;
}

.price {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $neon-purple;
}

.desc {
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-top: $spacing-xs;
}

.duration {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-xs;
}
</style>
