<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="header">
      <view class="back-btn" @click="goBack">
        <text class="back-icon">‹</text>
      </view>
      <text class="header-title">风格偏好</text>
      <view class="placeholder" />
    </view>

    <view class="content">
      <view class="intro-card">
        <text class="intro-title">选几个你喜欢的风格</text>
        <text class="intro-desc">最多 5 个，我们会按偏好为你推荐摄影师；随时可以回来修改或清空。</text>
      </view>

      <view class="chip-card">
        <!-- 与首启弹层共用同一份 chip 子组件 + SCSS -->
        <StyleTagChips
          :tags="styleStore.tags"
          :selected="draft"
          :max="MAX"
          @change="onDraftChange"
        />
        <text class="chip-count">已选 {{ draft.length }}/{{ MAX }}</text>
      </view>

      <view class="actions">
        <view
          class="btn-save"
          :class="{ disabled: draft.length === 0 }"
          @click="onSave"
        >
          <text>保存偏好</text>
        </view>
        <view class="btn-clear" @click="onClear">
          <text>清空偏好</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useStatusBar } from '@/composables/useStatusBar'
import { navBack, navTo } from '@/utils/nav'
import { useUserStore } from '@/stores/user'
import { useStyleStore } from '@/stores/style'
import StyleTagChips from '@/components/StyleTagChips.vue'

const MAX = 5
const { statusBarHeight } = useStatusBar()
const userStore = useUserStore()
const styleStore = useStyleStore()
const draft = ref<number[]>([])

onShow(() => { void syncPrefs() })

async function syncPrefs() {
  if (!userStore.isLoggedIn) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => navTo('/pages/login/index?redirect=/pages/profile/style-prefs'), 800)
    return
  }
  await Promise.all([styleStore.loadTags(), styleStore.loadPrefs()])
  draft.value = [...styleStore.selected]
}

function onDraftChange(next: number[]) {
  draft.value = next
}

function goBack() {
  navBack()
}

async function onSave() {
  if (!draft.value.length) {
    uni.showToast({ title: '请至少选择 1 个风格', icon: 'none' })
    return
  }
  try {
    await styleStore.savePrefs(draft.value)
  } catch {
    return // 失败提示已在 store 内统一 toast，保留当前选择
  }
  uni.showToast({ title: '已保存风格偏好', icon: 'none' })
  setTimeout(() => navBack(), 500)
}

async function onClear() {
  try {
    // 空数组 = 清空；chosen 仍为 true，不会重新触发首启弹层。
    await styleStore.savePrefs([])
  } catch {
    return
  }
  draft.value = []
  uni.showToast({ title: '已清空风格偏好', icon: 'none' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $surface-page;
}

.status-bar {
  width: 100%;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $s-sm $s-md;
}

.back-btn {
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.back-icon {
  font-size: 48rpx;
  color: $text-1;
}

.placeholder {
  width: 60rpx;
}

.header-title {
  font-size: $t-2xl;
  font-weight: 600;
  color: $text-1;
}

.content {
  padding: $s-md;
}

.intro-card {
  display: flex;
  flex-direction: column;
  background: $surface-1;
  border-radius: $r-md;
  border: 1rpx solid $border-subtle;
  padding: $s-md;
  margin-bottom: $s-md;
}

.intro-title {
  font-size: $t-xl;
  font-weight: 700;
  color: $text-1;
}

.intro-desc {
  margin-top: $s-2xs;
  font-size: $t-sm;
  color: $text-3;
  line-height: 1.5;
}

.chip-card {
  background: $surface-1;
  border-radius: $r-md;
  border: 1rpx solid $border-subtle;
  padding: $s-md;
  margin-bottom: $s-lg;
}

.chip-count {
  display: block;
  margin-top: $s-md;
  font-size: $t-sm;
  color: $text-2;
}

.actions {
  display: flex;
  flex-direction: column;
  gap: $s-sm;
}

.btn-save,
.btn-clear {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $s-md 0;
  border-radius: $r-md;
  font-size: $t-base;
  font-weight: 600;
}

.btn-save {
  background: $gradient-brand;
  color: $text-1;
  box-shadow: $glow-brand;

  &.disabled {
    opacity: 0.45;
    box-shadow: none;
  }
}

.btn-clear {
  background: $surface-2;
  color: $text-2;
  border: 1rpx solid $border-subtle;
}
</style>
