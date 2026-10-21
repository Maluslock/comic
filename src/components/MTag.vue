<template>
  <text class="m-tag" :class="[`variant-${variant}`, `size-${size}`]"><slot /></text>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'neutral' | 'brand' | 'accent' | 'success'
  size?: 'sm' | 'md'
}>(), {
  variant: 'neutral',
  size: 'sm'
})
</script>

<style lang="scss" scoped>
.m-tag {
  display: inline-flex;
  align-items: center;
  padding: 4rpx 14rpx;
  border-radius: $r-full;
  font-weight: 500;
  line-height: 1.2;
}

.size-sm { font-size: $t-xs; }
.size-md { font-size: $t-sm; padding: 6rpx 18rpx; }

.variant-neutral { background: rgba(255, 255, 255, 0.06); color: $text-2; }
.variant-brand { background: $brand-soft; color: $brand; }
.variant-accent { background: $accent-soft; color: $accent; }
.variant-success { background: rgba(16, 185, 129, 0.12); color: $state-success; }
</style>
