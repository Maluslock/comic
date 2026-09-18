<template>
  <view class="page">
    <view class="gallery">
      <view class="gallery-header">
        <text class="gallery-title">作品集</text>
        <text class="gallery-count">{{ works.length }} 作品</text>
      </view>
      
      <view class="gallery-grid">
        <view 
          v-for="(work, index) in works" 
          :key="work.id" 
          class="gallery-item"
          :class="{ 'full-width': work.images.length >= 2 }"
          @click="previewImage(index)"
        >
          <image :src="work.images[0]" class="gallery-image" mode="aspectFill" />
          <view class="gallery-overlay">
            <text class="gallery-title-text">{{ work.title }}</text>
            <view class="gallery-tags">
              <text 
                v-for="tag in work.tags.slice(0, 2)" 
                :key="tag" 
                class="gallery-tag"
              >{{ tag }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="works.length === 0" class="empty">
        <text class="empty-icon">◆</text>
        <text class="empty-text">暂无作品</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiGet } from '@/api/client'
import { mapWorkItem } from '@/utils/mappers'
import type { Work } from '@/types'

const works = ref<Work[]>([])

onMounted(() => {
  loadData()
})

async function loadData() {
  uni.showLoading({ title: '加载中...' })
  try {
    // 作品来自摄影师详情（首页推荐摄影师作品聚合展示）
    const pages = getCurrentPages()
    const currentPage = pages[pages.length - 1] as any
    const pid = currentPage?.options?.photographerId as string | undefined
    if (pid) {
      const res = await apiGet<any>(`/v1/photographers/${pid}`)
      works.value = (res.works || []).map(mapWorkItem)
    } else {
      // 无 pid：取首页推荐摄影师的作品聚合
      const home = await apiGet<any>('/v1/home')
      works.value = (home.featuredWorks || []).map(mapWorkItem)
    }
  } finally {
    uni.hideLoading()
  }
}

function previewImage(index: number) {
  const currentWork = works.value[index]
  uni.previewImage({
    current: currentWork.images[0],
    urls: currentWork.images
  })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.gallery {
  padding: $spacing-md;
}

.gallery-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
}

.gallery-title {
  position: relative;
  font-size: $font-size-xl;
  font-weight: 600;
  color: $dark-text-primary;
  padding-left: 20rpx;

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

.gallery-count {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-sm;
}

.gallery-item {
  position: relative;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.25);
  overflow: hidden;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.97);
    border-color: $neon-purple-glow;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }

  &.full-width {
    grid-column: span 2;
  }
}

.gallery-image {
  width: 100%;
  height: 280rpx;
  
  .full-width & {
    height: 400rpx;
  }
}

.gallery-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));
  padding: $spacing-md;
}

.gallery-title-text {
  font-size: $font-size-sm;
  color: #fff;
  font-weight: 500;
  display: block;
}

.gallery-tags {
  display: flex;
  gap: $spacing-xs;
  margin-top: $spacing-xs;
}

.gallery-tag {
  font-size: $font-size-xs;
  color: rgba(255, 255, 255, 0.8);
  padding: 2rpx 8rpx;
  background: rgba(255, 255, 255, 0.2);
  border-radius: $border-radius-sm;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2;
}

.empty-icon {
  font-size: 80rpx;
  color: $neon-purple;
  margin-bottom: $spacing-md;
}

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-tertiary;
}
</style>
