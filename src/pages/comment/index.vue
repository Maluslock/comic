<template>
  <view class="page">
    <view class="section">
      <view class="section-title">我的评价</view>
      <view v-if="myReviews.length === 0" class="empty">
        <text class="empty-icon">✍️</text>
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

    <view class="section">
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
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ReviewCard from '@/components/ReviewCard.vue'
import type { Photographer } from '@/types'

const myReviews = ref<any[]>([
  {
    id: 'r1',
    userId: 'u1',
    userName: '小狐狸',
    userAvatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=anime%20girl%20avatar%20fox%20ears%20cute&image_size=square',
    photographerId: '1',
    rating: 5,
    content: '摄影师非常专业，拍出来的效果超出预期！沟通也很顺畅，下次还会合作~',
    createdAt: Date.now() - 86400000
  }
])

const photographers = ref<Photographer[]>([
  {
    id: '2',
    name: '古风公子',
    avatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=chinese%20ancient%20style%20photographer%20avatar%20elegant&image_size=square',
    role: 'photographer',
    description: '',
    tags: [],
    location: '',
    rating: 0,
    reviewCount: 0,
    orderCount: 0,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  },
  {
    id: '3',
    name: '樱花落',
    avatar: 'https://neeko-copilot.bytedance.net/api/text_to_image?prompt=female%20photographer%20avatar%20pink%20hair%20cute%20style&image_size=square',
    role: 'photographer',
    description: '',
    tags: [],
    location: '',
    rating: 0,
    reviewCount: 0,
    orderCount: 0,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  }
])

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

function submitReview() {
  if (rating.value === 0) {
    uni.showToast({ title: '请选择评分', icon: 'none' })
    return
  }
  if (!reviewContent.value.trim()) {
    uni.showToast({ title: '请输入评价内容', icon: 'none' })
    return
  }
  
  uni.showToast({ title: '评价成功', icon: 'success' })
  closeModal()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
  padding-top: env(safe-area-inset-top);
}

.section {
  background: $bg-primary;
  margin-top: $spacing-md;
  padding: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
  margin-bottom: $spacing-md;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2;
}

.empty-icon {
  font-size: 80rpx;
  margin-bottom: $spacing-md;
}

.empty-text {
  font-size: $font-size-base;
  color: $text-tertiary;
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
  background: $bg-secondary;
  border-radius: $border-radius-md;
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
}

.info {
  flex: 1;
  margin-left: $spacing-md;
}

.name {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
}

.hint {
  font-size: $font-size-xs;
  color: $text-tertiary;
}

.btn-outline {
  padding: $spacing-xs $spacing-md;
  border: 2rpx solid $primary-color;
  color: $primary-color;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
}

.modal-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 90%;
  background: $bg-primary;
  border-radius: $border-radius-xl;
  padding: $spacing-lg;
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
}

.modal-close {
  font-size: $font-size-xl;
  color: $text-tertiary;
}

.rating-section {
  margin-bottom: $spacing-lg;
}

.rating-label {
  font-size: $font-size-base;
  font-weight: 500;
  display: block;
  margin-bottom: $spacing-sm;
}

.rating-stars {
  display: flex;
  gap: $spacing-sm;
}

.star {
  font-size: 60rpx;
  color: #ddd;
  
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
  margin-bottom: $spacing-sm;
}

.input-textarea {
  width: 100%;
  height: 240rpx;
  padding: $spacing-md;
  background: $bg-secondary;
  border-radius: $border-radius-md;
  font-size: $font-size-base;
}

.input-count {
  display: block;
  text-align: right;
  font-size: $font-size-xs;
  color: $text-tertiary;
  margin-top: $spacing-xs;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
}

.btn-primary {
  padding: $spacing-sm $spacing-xl;
  background: $primary-color;
  color: #fff;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
}
</style>
