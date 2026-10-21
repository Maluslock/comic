<template>
  <view class="skeleton-page">
    <!-- Header -->
    <view class="sk-row sk-header">
      <view class="sk-block" style="width: 160rpx; height: 36rpx" />
    </view>

    <!-- Hero -->
    <view class="sk-hero">
      <view class="sk-block sk-hero-fill" />
      <view class="sk-hero-copy">
        <view class="sk-block" style="width: 240rpx; height: 24rpx" />
        <view class="sk-block" style="width: 360rpx; height: 56rpx; margin-top: 16rpx" />
        <view class="sk-block" style="width: 280rpx; height: 24rpx; margin-top: 16rpx" />
      </view>
    </view>

    <!-- Ticket -->
    <view class="sk-row sk-ticket">
      <view class="sk-block" style="width: 100%; height: 132rpx; border-radius: 24rpx" />
    </view>

    <!-- Quick actions -->
    <view class="sk-row sk-actions">
      <view v-for="i in 4" :key="'a' + i" class="sk-action-item">
        <view class="sk-block" style="width: 72rpx; height: 72rpx; border-radius: 50%" />
        <view class="sk-block" style="width: 60rpx; height: 20rpx; margin-top: 12rpx" />
      </view>
    </view>

    <!-- Section title -->
    <view class="sk-row sk-section-title">
      <view class="sk-block" style="width: 160rpx; height: 32rpx" />
      <view class="sk-block" style="width: 80rpx; height: 24rpx" />
    </view>

    <!-- Horizontal event cards -->
    <view class="sk-row sk-events">
      <view v-for="i in 3" :key="'e' + i" class="sk-event-card">
        <view class="sk-block" style="width: 300rpx; height: 420rpx; border-radius: 24rpx" />
      </view>
    </view>

    <!-- Tag pills -->
    <view class="sk-row sk-section-title">
      <view class="sk-block" style="width: 140rpx; height: 32rpx" />
    </view>
    <view class="sk-row sk-tags">
      <view v-for="i in 6" :key="'t' + i" class="sk-block" style="width: 100rpx; height: 48rpx; border-radius: 48rpx" />
    </view>

    <!-- Photographer cards grid -->
    <view class="sk-row sk-section-title">
      <view class="sk-block" style="width: 180rpx; height: 32rpx" />
      <view class="sk-block" style="width: 80rpx; height: 24rpx" />
    </view>
    <view class="sk-row sk-photographers">
      <view v-for="i in 4" :key="'p' + i" class="sk-photographer-card">
        <view class="sk-block" style="width: 80rpx; height: 80rpx; border-radius: 50%" />
        <view class="sk-block" style="width: 120rpx; height: 24rpx; margin-top: 12rpx" />
        <view class="sk-block" style="width: 80rpx; height: 20rpx; margin-top: 8rpx" />
      </view>
    </view>

  </view>
</template>

<script setup lang="ts">
// Skeleton component — no logic needed
</script>

<script lang="ts">
export default { name: 'HomeSkeleton' }
</script>

<style lang="scss" scoped>
.skeleton-page {
  min-height: 100vh;
  background: $surface-page;
  padding-top: calc(var(--status-bar-height, 44px) + $spacing-md);
  padding-bottom: 120rpx;
}

.sk-hero {
  position: relative;
  height: 780rpx;
  margin-bottom: 0;
}

.sk-hero-fill {
  position: absolute;
  inset: 0;
  border-radius: 0;
}

.sk-hero-copy {
  position: absolute;
  left: $spacing-md;
  right: $spacing-md;
  bottom: 140rpx;
}

.sk-ticket {
  margin-top: -64rpx;
  position: relative;
  z-index: 2;
}

.sk-row {
  padding: 0 $spacing-md;
  margin-bottom: $spacing-md;
}

.sk-block {
  background: rgba(255, 255, 255, 0.04);
  border-radius: 16rpx;
  background-size: 200% 100%;
  background-image: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.04) 25%,
    rgba(255, 255, 255, 0.08) 50%,
    rgba(255, 255, 255, 0.04) 75%
  );
  animation: shimmer 1.5s ease-in-out infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.sk-header {
  padding: $spacing-md;
}

.sk-search {
  margin-top: $spacing-sm;
}

.sk-banner {
  margin-top: $spacing-sm;
}

.sk-actions {
  display: flex;
  justify-content: space-around;
  margin-top: $spacing-lg;
}

.sk-action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.sk-section-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: $spacing-lg;
}

.sk-events {
  display: flex;
  gap: $spacing-md;
  overflow: hidden;
}

.sk-event-card {
  flex-shrink: 0;
}

.sk-tags {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.sk-photographers {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
}

.sk-photographer-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: $r-lg;
  padding: $s-md;
}

.sk-works {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
}

</style>
