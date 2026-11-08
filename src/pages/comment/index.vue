<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <template v-if="firstLoading">
      <view class="skeleton-page">
        <view class="sk-line sk-w-40" />
        <ListSkeleton :rows="4" variant="row" />
      </view>
    </template>
    <template v-else>
    <view class="section">
      <view class="section-title">我的评价</view>
      <view v-if="myReviews.length === 0" class="empty">
        <text class="empty-icon">暂无</text>
        <text class="empty-text">暂无评价</text>
      </view>
      <view v-else class="review-list">
        <ReviewCard 
          v-for="review in myReviews" 
          :key="review.id" 
          :review="review"
        />
      </view>
    </view>

    <view v-if="pid" class="section">
      <view class="section-title">评价摄影师</view>
      <view class="photographer-select">
        <view v-for="p in photographers" :key="p.id" class="photographer-option">
          <image :src="p.avatar" class="avatar" mode="aspectFill" />
          <view class="info">
            <text class="name">{{ p.name }}</text>
            <text class="hint">点击评价</text>
          </view>
          <view class="btn-outline" @click="goWriteReview(p)">评价</view>
        </view>
      </view>
    </view>

    <view v-if="showWriteModal" class="modal-mask" @click="closeModal">
      <view class="modal-content" @click.stop>
        <view class="modal-header">
          <text class="modal-title">评价摄影师</text>
          <text class="modal-close" @click="closeModal">✕</text>
        </view>
        
        <view class="rating-section">
          <text class="rating-label">评分</text>
          <view class="rating-stars">
            <text 
              v-for="i in 5" 
              :key="i" 
              class="star"
              :class="{ active: i <= rating }"
              @click="setRating(i)"
            >★</text>
          </view>
        </view>

        <view class="input-section">
          <text class="input-label">评价内容</text>
          <textarea 
            class="input-textarea" 
            v-model="reviewContent"
            placeholder="请输入您的评价..."
            :maxlength="500"
          />
          <text class="input-count">{{ reviewContent.length }}/500</text>
        </view>

        <view class="modal-footer">
          <view class="btn-outline" @click="closeModal">取消</view>
          <view class="btn-primary" @click="submitReview">提交评价</view>
        </view>
      </view>
    </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import ReviewCard from '@/components/ReviewCard.vue'
import { apiGet, apiPost } from '@/api/client'
import { getMyReviews } from '@/api/index'
import { useUserStore } from '@/stores/user'
import { mapPhotographerItem, mapReviewItem } from '@/utils/mappers'
import type { Photographer } from '@/types'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

const userStore = useUserStore()

const myReviews = ref<any[]>([])
const photographers = ref<Photographer[]>([])

let pid = ''

onLoad((options) => {
  pid = options?.photographerId || ''
  loadData()
})

async function loadData() {
  try {
    const mine = await getMyReviews()
    myReviews.value = (mine || []).map(mapReviewItem)

    if (pid) {
      const res = await apiGet<any>(`/v1/photographers/${pid}`)
      photographers.value = [mapPhotographerItem(res)]
    } else {
      photographers.value = []
    }
  } catch {
    myReviews.value = []
    photographers.value = []
  } finally {
    firstLoading.value = false
  }
}

const showWriteModal = ref(false)
const rating = ref(0)
const reviewContent = ref('')

function goWriteReview(p: Photographer) {
  showWriteModal.value = true
  rating.value = 5
  reviewContent.value = ''
}

function closeModal() {
  showWriteModal.value = false
}

function setRating(r: number) {
  rating.value = r
}

async function submitReview() {
  if (!userStore.isLoggedIn) { uni.showToast({ title: '请先登录', icon: 'none' }); return }
  if (!rating.value || !reviewContent.value.trim()) {
    uni.showToast({ title: '请打分并填写内容', icon: 'none' }); return
  }
  try {
    await apiPost('/v1/reviews', {
      photographerId: Number(pid),
      userId: Number(userStore.user?.id),
      userName: userStore.user?.name || '',
      rating: rating.value,
      content: reviewContent.value.trim(),
    })
    uni.showToast({ title: '评价成功', icon: 'success' })
    closeModal()
    loadData()
  } catch {
    uni.showToast({ title: '提交失败', icon: 'none' })
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding-top: env(safe-area-inset-top);
  &::before {
    content: '';
    position: fixed;
    top: var(--window-top, 0px);
    right: 0;
    bottom: 0;
    left: 0;
    background: $surface-page;
    z-index: -1;
    pointer-events: none;
  }
}

.section {
  background: $dark-bg-card;
  margin-top: $spacing-md;
  padding: $spacing-md;
  border: 1rpx solid $dark-border;
  box-shadow: 0 2rpx 16rpx rgba(0, 0, 0, 0.15);
}

.section-title {
  position: relative;
  padding-left: 20rpx;
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;
  margin-bottom: $spacing-md;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 6rpx;
    height: 28rpx;
    background: $neon-gradient;
    border-radius: 3rpx;
  }
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2;
}

.empty-icon {
  font-size: 48rpx;
  margin-bottom: $spacing-md;
  color: $dark-text-tertiary;
}

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
}

.review-list {
  margin-top: $spacing-sm;
}

.photographer-select {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.photographer-option {
  display: flex;
  align-items: center;
  padding: $spacing-md;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;

  &:active {
    background: $dark-bg-card-hover;
  }
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  border: 2rpx solid $dark-border;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
  color: $dark-text-primary;
}

.hint {
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
}

.btn-outline {
  padding: $spacing-xs $spacing-md;
  border: 2rpx solid $neon-purple;
  color: $neon-purple;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;

  &:active {
    background: $neon-purple-dim;
    border-color: $neon-purple-glow;
  }
}

.modal-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 90%;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-xl;
  padding: $spacing-lg;
  box-shadow: 0 8rpx 40rpx rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-lg;
}

.modal-title {
  font-size: $font-size-xl;
  font-weight: 600;
  color: $dark-text-primary;
}

.modal-close {
  font-size: $font-size-xl;
  color: $dark-text-tertiary;
  padding: $spacing-xs;

  &:active {
    opacity: 0.6;
  }
}

.rating-section {
  margin-bottom: $spacing-lg;
}

.rating-label {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
  color: $dark-text-primary;
  margin-bottom: $spacing-sm;
}

.rating-stars {
  display: flex;
  gap: $spacing-sm;
}

.star {
  font-size: 60rpx;
  color: $dark-text-tertiary;

  &.active {
    color: $warning-color;
  }
}

.input-section {
  margin-bottom: $spacing-lg;
}

.input-label {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
  color: $dark-text-primary;
  margin-bottom: $spacing-sm;
}

.input-textarea {
  width: 100%;
  height: 240rpx;
  padding: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  font-size: $font-size-base;
  color: $dark-text-primary;

  &::placeholder {
    color: $dark-text-tertiary;
  }
}

.input-count {
  display: block;
  text-align: right;
  font-size: $font-size-xs;
  color: $dark-text-tertiary;
  margin-top: $spacing-xs;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
}

.btn-primary {
  padding: $spacing-sm $spacing-xl;
  background: $neon-gradient;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
  box-shadow: 0 4rpx 16rpx $neon-purple-glow;

  &:active {
    opacity: 0.85;
  }
}
/* ── 首屏深色骨架：切页不再露白（B 档）── */
.skeleton-page { min-height: 100vh; padding: $spacing-md; box-sizing: border-box; display: flex; flex-direction: column; gap: 24rpx; }
.sk-card { background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 20rpx; box-sizing: border-box; }
.sk-block { background: $surface-2; border-radius: 12rpx; }
.sk-line { height: 28rpx; background: $surface-2; border-radius: 8rpx; }
.sk-circle { background: $surface-2; border-radius: 50%; flex-shrink: 0; }
.sk-row { display: flex; align-items: center; gap: 24rpx; }
.sk-pad { padding: 28rpx; display: flex; flex-direction: column; gap: 24rpx; }
.sk-avatar-120 { width: 120rpx; height: 120rpx; }
.sk-avatar-88 { width: 88rpx; height: 88rpx; }
.sk-hero { height: 160rpx; }
.sk-grid { display: flex; flex-wrap: wrap; gap: 16rpx; }
.sk-grid-item { width: calc(50% - 8rpx); height: 320rpx; border-radius: 16rpx; }
.sk-w-25 { width: 25%; } .sk-w-30 { width: 30%; } .sk-w-40 { width: 40%; }
.sk-w-60 { width: 60%; } .sk-w-80 { width: 80%; } .sk-w-full { width: 100%; }
</style>
