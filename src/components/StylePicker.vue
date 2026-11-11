<template>
  <view v-if="visible" class="style-picker-mask" @touchmove.stop>
    <!-- 点击遮罩不关闭：mask 不绑定任何 click，防误触丢掉已选风格。 -->
    <view class="style-picker-sheet" @click.stop>
      <view class="style-picker-head">
        <text class="style-picker-title">选几个你喜欢的风格</text>
        <text class="style-picker-subtitle">我们会按偏好为你推荐摄影师</text>
      </view>

      <view class="style-picker-body">
        <StyleTagChips
          :tags="styleStore.tags"
          :selected="draft"
          :max="MAX"
          @change="onDraftChange"
        />
      </view>

      <view class="style-picker-foot">
        <text class="style-picker-count">已选 {{ draft.length }}/{{ MAX }}</text>
        <view class="style-picker-actions">
          <view
            class="style-picker-btn primary"
            :class="{ disabled: draft.length === 0 }"
            @click="onSave"
          >
            <text>保存并为我推荐</text>
          </view>
          <view class="style-picker-btn ghost" @click="onSkip">
            <text>跳过</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import StyleTagChips from '@/components/StyleTagChips.vue'
import { useStyleStore } from '@/stores/style'

// 组件只读 store 里的标签全集与初始选中；不发任何请求，保存/跳过交给页面。
const MAX = 5

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  (e: 'save', tagIds: number[]): void
  (e: 'skip'): void
  (e: 'update:visible', value: boolean): void
}>()

const styleStore = useStyleStore()
const draft = ref<number[]>([])

watch(
  () => props.visible,
  (v) => { if (v) draft.value = [...styleStore.selected] },
  { immediate: true }
)

function onDraftChange(next: number[]) {
  draft.value = next
}

function onSave() {
  if (!draft.value.length) {
    uni.showToast({ title: '请至少选择 1 个风格', icon: 'none' })
    return
  }
  emit('save', [...draft.value])
}

function onSkip() {
  emit('skip')
  emit('update:visible', false)
}
</script>

<style lang="scss" scoped>
.style-picker-mask {
  position: fixed;
  inset: 0;
  // 高于城市弹层(200) 与 tabBar；沿用项目既有弹层层级体系。
  z-index: 300;
  display: flex;
  align-items: flex-end;
  // 由既有 page token 派生半透明遮罩，不引入新色值。
  background: rgba($surface-page, 0.72);
  padding-bottom: #{$tab-bar-height};
  overscroll-behavior: contain;
}

.style-picker-sheet {
  width: 100%;
  background: $surface-1;
  border-radius: $r-lg $r-lg 0 0;
  border-top: 1rpx solid $border-subtle;
  box-shadow: $elev-3;
  padding: $s-lg $s-md $s-lg;
  padding-bottom: calc(#{$s-lg} + #{$safe-area-bottom});
}

.style-picker-head {
  display: flex;
  flex-direction: column;
  margin-bottom: $s-md;
}

.style-picker-title {
  font-size: $t-2xl;
  font-weight: 700;
  color: $text-1;
}

.style-picker-subtitle {
  margin-top: $s-2xs;
  font-size: $t-sm;
  color: $text-3;
}

.style-picker-body {
  margin-bottom: $s-md;
}

.style-picker-foot {
  display: flex;
  flex-direction: column;
}

.style-picker-count {
  font-size: $t-sm;
  color: $text-2;
  margin-bottom: $s-sm;
}

.style-picker-actions {
  display: flex;
  gap: $s-sm;
}

.style-picker-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $s-md 0;
  border-radius: $r-md;
  font-size: $t-base;
  font-weight: 600;

  &.primary {
    flex: 2;
    background: $gradient-brand;
    color: $text-1;
    box-shadow: $glow-brand;

    &.disabled {
      opacity: 0.45;
      box-shadow: none;
    }
  }

  &.ghost {
    background: $surface-2;
    color: $text-2;
    border: 1rpx solid $border-subtle;
  }
}
</style>
