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
        <text class="empty-icon">📷</text>
        <text class="empty-text">暂无作品</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getWorks } from '@/api/index'
import type { Work } from '@/types'

const works = ref<Work[]>([])

onMounted(() => {
  loadData()
})

async function loadData() {
  uni.showLoading({ title: '加载中...' })
  try {
    works.value = await getWorks()
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
  background: $bg-page;
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
  font-size: $font-size-xl;
  font-weight: 600;
}

.gallery-count {
  font-size: $font-size-sm;
  color: $text-tertiary;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-sm;
}

.gallery-item {
  position: relative;
  border-radius: $border-radius-md;
  overflow: hidden;
  
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
  margin-bottom: $spacing-md;
}

.empty-text {
  font-size: $font-size-base;
  color: $text-tertiary;
}
</style>
