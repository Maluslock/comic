<template>
  <text v-if="text" class="m-price" :class="[`tone-${tone}`, `size-${size}`]">{{ text }}</text>
</template>

<script setup lang="ts">
/**
 * 价格文案唯一出口：¥{min} 起 / 互勉 / 面议 / 暂未设置
 * tone 由 describePrice() 给出（price / free / negotiable / muted）
 */
withDefaults(defineProps<{
  text?: string
  tone?: 'price' | 'free' | 'negotiable' | 'muted'
  size?: 'sm' | 'md' | 'lg'
}>(), {
  text: '',
  tone: 'price',
  size: 'md'
})
</script>

<style lang="scss" scoped>
.m-price {
  font-family: 'DIN Alternate', 'Trebuchet MS', -apple-system, sans-serif;
  font-weight: 700;
  letter-spacing: 0.2rpx;
  white-space: nowrap;
}

.size-sm { font-size: $t-sm; }
.size-md { font-size: $t-base; }
.size-lg { font-size: $t-2xl; }

.tone-price { color: $brand; }
.tone-free { color: $state-success; }
.tone-negotiable { color: $accent; }
.tone-muted { color: $text-3; font-weight: 500; }
</style>
