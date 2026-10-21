<template>
  <view
    class="m-btn"
    :class="[`type-${type}`, `size-${size}`, { block, disabled, loading }]"
    :hover-class="disabled || loading ? 'none' : 'm-btn--pressed'"
    :hover-stay-time="60"
    @click="onClick"
  >
    <view v-if="loading" class="spinner" />
    <slot />
  </view>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  type?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'lg' | 'md' | 'sm'
  block?: boolean
  disabled?: boolean
  loading?: boolean
}>(), {
  type: 'primary',
  size: 'lg',
  block: false,
  disabled: false,
  loading: false
})

const emit = defineEmits<{ (e: 'click', ev: Event): void }>()

function onClick(ev: Event) {
  if (props.disabled || props.loading) return
  emit('click', ev)
}
</script>

<style lang="scss" scoped>
.m-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $s-xs;
  box-sizing: border-box;
  border-radius: $r-md;
  font-weight: 600;
  transition: background $motion-fast, transform $motion-fast, opacity $motion-fast;
}

.block { display: flex; width: 100%; }

.size-lg { height: 88rpx; padding: 0 $s-lg; font-size: $t-base; }
.size-md { height: 72rpx; padding: 0 $s-md; font-size: $t-sm; }
.size-sm { height: 56rpx; padding: 0 $s-sm; font-size: $t-xs; }

.type-primary {
  background: $brand;
  color: #fff;
  box-shadow: $glow-brand;
}
.type-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: $text-1;
  border: 1rpx solid $border-strong;
}
.type-ghost {
  background: transparent;
  color: $text-2;
  border: 1rpx solid $border-subtle;
}
.type-danger {
  background: rgba(244, 63, 94, 0.14);
  color: $state-danger;
  border: 1rpx solid rgba(244, 63, 94, 0.35);
}

.m-btn--pressed { transform: scale(0.985); opacity: 0.9; }
.type-primary.m-btn--pressed { background: $brand-active; box-shadow: none; }

.disabled {
  background: $surface-3;
  color: $text-disabled;
  border-color: transparent;
  box-shadow: none;
}

.spinner {
  width: 28rpx;
  height: 28rpx;
  border-radius: 50%;
  border: 3rpx solid rgba(255, 255, 255, 0.35);
  border-top-color: #fff;
  animation: m-spin 0.7s linear infinite;
}

@keyframes m-spin {
  to { transform: rotate(360deg); }
}
</style>
