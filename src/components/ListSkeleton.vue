<template>
  <view class="list-skeleton" :class="[`variant-${variant}`]">
    <view
      v-for="index in rows"
      :key="index"
      class="skeleton-item"
    >
      <view class="shimmer" />
    </view>
  </view>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    rows?: number
    variant?: 'card' | 'row'
  }>(),
  {
    rows: 5,
    variant: 'card'
  }
)
</script>

<style lang="scss" scoped>
.list-skeleton {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  width: 100%;
}

.skeleton-item {
  position: relative;
  overflow: hidden;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  box-sizing: border-box;
}

.variant-card .skeleton-item {
  height: 200rpx;
  border-radius: 20rpx;
}

.variant-row .skeleton-item {
  height: 120rpx;
  border-radius: 16rpx;
}

.shimmer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(255, 255, 255, 0.05) 50%,
    rgba(255, 255, 255, 0) 100%
  );
  animation: skeleton-shimmer 1.6s infinite ease-in-out;
  transform: translateX(-100%);
}

@keyframes skeleton-shimmer {
  100% {
    transform: translateX(100%);
  }
}
</style>
