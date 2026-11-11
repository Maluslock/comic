<template>
  <view class="style-chip-grid">
    <view
      v-for="tag in tags"
      :key="tag.id"
      class="style-chip"
      :class="{ active: selected.includes(tag.id), locked: isFull && !selected.includes(tag.id) }"
      @click="onTap(tag.id)"
    >
      <text class="style-chip-label">{{ tag.name }}</text>
      <text v-if="selected.includes(tag.id)" class="style-chip-check">✓</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// 风格标签多选网格：弹层（StylePicker）与整页（profile/style-prefs）共用这一份
// chip 模板 + 交互 + SCSS，避免复制两套几乎一样的样式。
const props = withDefaults(defineProps<{
  tags: { id: number; name: string }[]
  selected: number[]
  max?: number
}>(), { max: 5 })

const emit = defineEmits<{
  (e: 'change', value: number[]): void
  (e: 'limit'): void
}>()

const isFull = computed(() => props.selected.length >= props.max)

function onTap(id: number) {
  const picked = props.selected.includes(id)
  if (!picked && isFull.value) {
    // 选满后未选中的 chip 已置灰；这里再给一次明确 toast，绝不静默失败。
    uni.showToast({ title: `最多选择 ${props.max} 个风格`, icon: 'none' })
    emit('limit')
    return
  }
  const next = picked
    ? props.selected.filter(x => x !== id)
    : [...props.selected, id]
  emit('change', next)
}
</script>

<style lang="scss" scoped>
.style-chip-grid {
  display: flex;
  flex-wrap: wrap;
  gap: $s-sm;
}

.style-chip {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $s-sm $s-md;
  border-radius: $r-full;
  background: $surface-2;
  border: 1rpx solid $border-subtle;
  color: $text-2;
  font-size: $t-sm;
  transition: background $motion-fast, border-color $motion-fast, opacity $motion-fast;

  &.active {
    background: $brand-soft;
    border-color: $border-focus;
    color: $brand;
    box-shadow: $glow-brand;
  }

  // 选满上限：未选中项置灰，点击由 onTap 拦截。
  &.locked {
    opacity: 0.45;
  }

  &:active {
    opacity: 0.8;
  }
}

.style-chip-label {
  line-height: 1.2;
}

.style-chip-check {
  margin-left: $s-2xs;
  color: $brand;
  font-size: $t-xs;
}
</style>
