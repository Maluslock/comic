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
  background: $bg-primary;
  border-radius: $border-radius-md;
  padding: $spacing-md;
  margin-bottom: $spacing-sm;
  border: 2rpx solid transparent;
  
  &.active {
    border-color: $primary-color;
    background: rgba($primary-color, 0.05);
  }
}

.checkbox {
  width: 40rpx;
  height: 40rpx;
  border: 2rpx solid $border-color;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-right: $spacing-md;
  
  &.checked {
    background: $primary-color;
    border-color: $primary-color;
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
}

.price {
  font-size: $font-size-lg;
  font-weight: 600;
  color: $primary-color;
}

.desc {
  font-size: $font-size-sm;
  color: $text-secondary;
  margin-top: $spacing-xs;
}

.duration {
  font-size: $font-size-xs;
  color: $text-tertiary;
  margin-top: $spacing-xs;
}
</style>
