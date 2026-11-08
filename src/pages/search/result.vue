<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <view class="search-bar">
      <view class="search-input-wrap">
        <image class="search-icon-svg" src="/static/icons/search.svg" mode="aspectFit" />
        <input 
          class="search-input" 
          v-model="keyword"
          placeholder="搜索摄影师、标签"
          placeholder-class="search-placeholder"
          @confirm="handleSearch"
        />
        <text v-if="keyword" class="clear-btn" @click="clearKeyword">✕</text>
      </view>
      <text class="cancel-btn" @click="goBack">取消</text>
    </view>

    <view class="tags-row" v-if="selectedTags.length">
      <view class="tag-chip" v-for="tag in selectedTags" :key="tag" @click="removeTag(tag)">
        {{ tag }} <text class="remove-icon">✕</text>
      </view>
    </view>

    <!-- 首屏骨架屏：仅首屏加载且没有现有数据时展示 -->
    <view v-if="loading && currentPage === 1" class="content">
      <ListSkeleton variant="card" :rows="4" />
    </view>
    <!-- 无搜索条件（keyword与tags均为空）时的引导态，不发全站列表请求 -->
    <view v-else-if="!keyword && !selectedTags.length" class="empty">
      <text class="empty-text">请输入关键词或选择标签</text>
      <view class="retry-btn" @click="goBack">返回</view>
    </view>
    <view v-else-if="error" class="empty">
      <text class="empty-text">{{ error }}</text>
      <view class="retry-btn" @click="handleSearch">重试</view>
    </view>
    <view v-else-if="photographers.length" class="result-list">
      <view class="result-header">
        <text class="result-count">{{ resultTitle }}</text>
      </view>
      <PhotographerCard 
        v-for="p in photographers" 
        :key="p.id" 
        :photographer="p"
      />
      <view v-if="hasMore" class="load-more" :class="{ disabled: loadingMore }" @click="loadMore">
        {{ loadingMore ? '加载中...' : '加载更多' }}
      </view>
    </view>
    <view v-else class="empty">
      <text class="empty-text">没搜到相关摄影师，换个关键词试试</text>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import PhotographerCard from '@/components/PhotographerCard.vue'
import ListSkeleton from '@/components/ListSkeleton.vue'
import { apiGet } from '@/api/client'
import { mapPhotographerItem } from '@/utils/mappers'
import type { Photographer } from '@/types'

const keyword = ref('')
const selectedTags = ref<string[]>([])
const photographers = ref<Photographer[]>([])
const total = ref(0)
const currentPage = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const hasMore = ref(false)

const resultTitle = computed(() => {
  if (keyword.value) {
    return `「${keyword.value}」· 找到 ${total.value} 位摄影师`
  }
  if (selectedTags.value.length) {
    return `标签：${selectedTags.value.join('、')} · 找到 ${total.value} 位摄影师`
  }
  return `找到 ${total.value} 位摄影师`
})

function safeDecode(val: string) {
  try {
    return val.includes('%') ? decodeURIComponent(val) : val
  } catch {
    return val
  }
}

onLoad((options: any) => {
  if (options.keyword) keyword.value = safeDecode(options.keyword)
  if (options.tags) selectedTags.value = safeDecode(options.tags).split(',').filter(Boolean)
  if (!keyword.value && !selectedTags.value.length) {
    return
  }
  handleSearch()
})

async function handleSearch() {
  if (!keyword.value && !selectedTags.value.length) {
    photographers.value = []
    total.value = 0
    hasMore.value = false
    return
  }
  currentPage.value = 1
  await fetchPhotographers(false)
}

async function fetchPhotographers(isLoadMore = false) {
  if (isLoadMore) {
    loadingMore.value = true
  } else {
    loading.value = true
  }
  error.value = ''
  try {
    const params: Record<string, string> = {
      page: String(currentPage.value),
      size: '10'
    }
    if (keyword.value) params.keyword = keyword.value
    if (selectedTags.value.length) params.tags = selectedTags.value.join(',')
    
    const res = await apiGet<{ list: any[]; total: number }>('/v1/photographers', params)
    const mapped = (res.list || []).map(mapPhotographerItem)
    photographers.value = currentPage.value === 1 ? mapped : [...photographers.value, ...mapped]
    total.value = res.total
    hasMore.value = photographers.value.length < total.value
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    if (isLoadMore) {
      loadingMore.value = false
    } else {
      loading.value = false
    }
  }
}

function removeTag(tag: string) {
  selectedTags.value = selectedTags.value.filter(t => t !== tag)
  handleSearch()
}

function clearKeyword() {
  keyword.value = ''
  handleSearch()
}

function goBack() {
  uni.navigateBack()
}

function loadMore() {
  if (loading.value || loadingMore.value || !hasMore.value) return
  currentPage.value++
  fetchPhotographers(true)
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
  padding-bottom: $spacing-lg;
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

.search-bar {
  display: flex;
  align-items: center;
  padding: $spacing-md;
  background: transparent;
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  height: 64rpx;
  background: rgba(21, 23, 34, 0.68);
  border: 1rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 32rpx;
  padding: 0 20rpx;
  backdrop-filter: blur(16px);
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.25);
}

.search-icon-svg {
  width: 28rpx;
  height: 28rpx;
  margin-right: 12rpx;
  opacity: 0.65;
}

.search-input {
  flex: 1;
  font-size: $font-size-base;
  color: $dark-text-primary;
}

.clear-btn {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  padding: $spacing-xs;
}

.cancel-btn {
  font-size: $font-size-base;
  color: $text-2;
  margin-left: $spacing-md;

  &:active {
    opacity: 0.7;
  }
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  // padding-top 是给 chip 的 ::after 热区留的（见 .tag-chip），视觉上和原来一样贴着搜索栏
  padding: 18rpx $spacing-md $spacing-md;
}

.tag-chip {
  position: relative; // 给 ::after 热区扩展层做定位基准
  padding: 8rpx 20rpx;
  background: $dark-bg-card;
  border: 1rpx solid $brand;
  border-radius: $border-radius-xl;
  color: $dark-text-primary;
  font-size: $font-size-sm;
  display: flex;
  align-items: center;

  // 视觉高度 29px，热区要拉到 ≥44px（微信适老化 3.3 的最小可点区域）。
  // 上下各 -18rpx(≈9.4px) + .tags-row 预留的 18rpx padding-top = 顶到搜索栏下沿但不重叠；
  // 左右各 -8rpx 正好是相邻 chip 间距(gap 16rpx)的一半，两个热区相切不交叠。
  &::after {
    content: '';
    position: absolute;
    top: -18rpx;
    bottom: -18rpx;
    left: -8rpx;
    right: -8rpx;
  }

  &:active {
    opacity: 0.8;
  }
}

.remove-icon {
  margin-left: 8rpx;
  font-size: 20rpx;
  color: $brand;
}

.result-list {
  padding: 0 $spacing-md;
}

.result-header {
  padding: $spacing-md 0;
  color: $dark-text-secondary;
  font-size: $font-size-sm;
}

.load-more {
  text-align: center;
  padding: $spacing-md;
  color: $dark-text-tertiary;
  font-size: $font-size-sm;

  &.disabled {
    opacity: 0.5;
  }
}

.empty {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 400rpx;
  color: $dark-text-tertiary;
  gap: 24rpx;
}

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
}

.retry-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 180rpx;
  height: 88rpx;
  padding: 0 40rpx;
  background: $brand;
  color: #fff;
  font-size: $font-size-base;
  font-weight: 500;
  border-radius: 44rpx;

  &:active {
    opacity: 0.85;
  }
}
</style>
