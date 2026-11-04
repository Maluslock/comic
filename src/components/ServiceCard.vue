<template>
  <view class="service-card" :class="{ active: selected }" @click="$emit('select', service)">
    <view class="checkbox" :class="{ checked: selected }">
      <text v-if="selected" class="check-icon">✓</text>
    </view>
    <view class="info">
      <view class="header">
        <text class="name">{{ service.name }}</text>
        <text class="price">{{ priceText }}</text>
      </view>
      <text class="desc">{{ service.description }}{{ showDuration ? ' · ' + formatDuration(service.duration) : '' }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import type { Service } from '@/types'
import { describeServicePrice } from '@/utils/price'
import { formatDuration } from '@/utils/duration'
import { computed } from 'vue'

const props = defineProps<{
  service: Service
  selected?: boolean
}>()

// 描述里已写「N 小时拍摄」时不再重复一行时长（避免「2小时拍摄」下一行又是「2 小时」）
const showDuration = computed(() => {
  const minutes = props.service.duration
  if (!minutes) return false
  const desc = props.service.description || ''
  // duration 单位是分钟（120 → 2 小时），描述里已是「2小时拍摄」时不再重复一行
  const hours = minutes % 60 === 0 ? String(minutes / 60) : (minutes / 60).toFixed(1)
  return !new RegExp(`${hours}\\s*小时`).test(desc)
})

defineEmits<{
  select: [service: Service]
}>()

// 价格必须带单位：¥10 在按张套餐里是「一张 10 块」，写成裸 ¥10 会被读成一整套 10 块。
// 复用 utils/price 的 describeServicePrice，和摄影师详情页/套餐管理页保持同一套文案：
// per_photo 有张数时还会补「· N张起」，自己拼字符串必然漂移。
const priceText = computed(() =>
  describeServicePrice({
    price: props.service.price ?? null,
    priceUnit: props.service.priceUnit ?? null,
    photoCount: props.service.photoCount ?? null,
  })
)
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
  display: block;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  margin-top: $spacing-xs;
}

.duration {
  display: block;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-xs;
}
</style>
