<template>
  <view class="quick-action-bar">
    <view
      v-for="item in actionItems"
      :key="item.label"
      class="action-item"
      @click="handleTap(item)"
    >
      <image class="action-icon" :src="item.icon" mode="aspectFit" />
      <text class="action-label">{{ item.label }}</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface QuickAction {
  icon: string
  label: string
  url: string
}

const props = withDefaults(defineProps<{
  items?: QuickAction[]
}>(), {
  items: () => [
    { icon: '/static/icons/action-calendar.png', label: '漫展日历', url: '/pages/calendar/index' },
    { icon: '/static/icons/action-camera.png', label: '摄影师', url: '/pages/photographer/list' },
    { icon: '/static/icons/action-tag.png', label: '风格标签', url: '/pages/search/search' },
    { icon: '/static/icons/action-order.png', label: '我的预约', url: '/pages/order/list' },
  ],
})

const emit = defineEmits<{
  navigate: [url: string]
}>()

const actionItems = computed(() => props.items)

function handleTap(item: QuickAction) {
  if (item.url.startsWith('/pages/photographer/')) {
    uni.switchTab({ url: '/pages/photographer/list' })
  } else if (item.url.startsWith('/pages/index/')) {
    uni.switchTab({ url: '/pages/index/index' })
  } else {
    uni.navigateTo({ url: item.url })
  }
  emit('navigate', item.url)
}
</script>

<style lang="scss" scoped>
.quick-action-bar {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: $spacing-sm;
  padding: 0 $spacing-md;
}

.action-icon {
  width: 44rpx;
  height: 44rpx;
  margin-bottom: 8rpx;
}

.action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20rpx $spacing-xs;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: $r-md;
  transition: all 0.2s ease;

  &:active {
    background: $dark-bg-card-hover;
    border-color: $neon-purple;
    box-shadow: 0 0 20rpx $neon-purple-glow;
  }
}

.action-icon {
  font-size: 36rpx;
  color: $neon-cyan;
  margin-bottom: $spacing-xs;
  line-height: 1;
  font-weight: 300;
}

.action-label {
  font-size: 24rpx;
  color: $dark-text-secondary;
  font-weight: 500;
}
</style>
