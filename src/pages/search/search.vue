<template>
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

    <view v-if="!keyword" class="hot-search">
      <view class="section-header">
        <text class="section-title">热门搜索</text>
      </view>
      <view class="hot-list">
        <view 
          v-for="(item, index) in hotKeywords" 
          :key="index" 
          class="hot-item"
          @click="handleSearch(item)"
        >
          {{ item }}
        </view>
      </view>

      <view class="section-header mt-lg">
        <text class="section-title">热门标签</text>
      </view>
      <view class="tags-container">
        <view 
          v-for="tag in tags" 
          :key="tag" 
          class="tag-item"
          :class="{ active: selectedTags.includes(tag) }"
          @click="toggleTag(tag)"
        >
          {{ tag }}
        </view>
      </view>
    </view>

    <view v-if="searched" class="search-result">
      <view class="result-header">
        <text class="result-count">找到 {{ photographers.length }} 位摄影师</text>
      </view>
      <view class="photographer-list">
        <PhotographerCard 
          v-for="p in photographers" 
          :key="p.id" 
          :photographer="p"
        />
      </view>
      <view v-if="photographers.length === 0" class="empty">
        <text class="empty-icon">暂无结果</text>
        <text class="empty-text">没有找到相关摄影师</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotographerCard from '@/components/PhotographerCard.vue'
import { apiGet } from '@/api/client'
import { mapPhotographerItem } from '@/utils/mappers'
import type { Photographer } from '@/types'

const keyword = ref('')
const tags = ref<string[]>([])
const selectedTags = ref<string[]>([])
const photographers = ref<Photographer[]>([])

const hotKeywords = ['原神', '鬼灭之刃', '古风', '日系', '暗黑', '漫展跟拍']
const searched = ref(false)

onMounted(() => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  if (currentPage?.options?.keyword) {
    keyword.value = currentPage.options.keyword
    handleSearch(currentPage.options.keyword)
  }
  loadTags()
})

async function loadTags() {
  try {
    const res = await apiGet<Array<{ id: number; name: string }>>('/v1/tags')
    tags.value = (res || []).map(t => t.name)
  } catch {
    tags.value = []
  }
}

async function handleSearch(kw?: string) {
  if (kw) keyword.value = kw
  const searchKeyword = keyword.value
  if (!searchKeyword && selectedTags.value.length === 0) {
    searched.value = false
    photographers.value = []
    return
  }
  uni.showLoading({ title: '搜索中...' })
  try {
    const params: Record<string, string> = {}
    if (searchKeyword) params.keyword = searchKeyword
    if (selectedTags.value.length) params.tags = selectedTags.value.join(',')
    const res = await apiGet<{ list: any[] }>('/v1/photographers', params)
    photographers.value = (res.list || []).map(mapPhotographerItem)
    searched.value = true
  } finally {
    uni.hideLoading()
  }
}

function toggleTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
  handleSearch()
}

function clearKeyword() {
  keyword.value = ''
  selectedTags.value = []
  photographers.value = []
  searched.value = false
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.search-bar {
  display: flex;
  align-items: center;
  padding: $spacing-md;
  background: $dark-bg-secondary;
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  background: $dark-bg-secondary;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-sm $spacing-md;
}

.search-icon-svg {
  width: 36rpx;
  height: 36rpx;
  margin-right: $spacing-xs;
}

.search-input {
  flex: 1;
  font-size: $font-size-base;
  color: $dark-text-primary;
}

.search-placeholder {
  color: $dark-text-tertiary;
}

.clear-btn {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  padding: $spacing-xs;
}

.cancel-btn {
  font-size: $font-size-base;
  color: $neon-cyan;
  margin-left: $spacing-md;

  &:active {
    opacity: 0.6;
  }
}

.hot-search {
  padding: $spacing-md;
}

.section-header {
  margin-bottom: $spacing-md;
}

.section-title {
  position: relative;
  padding-left: 20rpx;
  font-size: $font-size-lg;
  font-weight: 600;
  color: $dark-text-primary;

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

.hot-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.hot-item {
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  color: $dark-text-secondary;

  &:active {
    background: $neon-purple-dim;
    color: $neon-purple;
  }
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.tag-item {
  padding: $spacing-sm $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  color: $dark-text-secondary;

  &.active {
    background: $neon-purple;
    color: #fff;
    border-color: $neon-purple;
  }

  &:active {
    opacity: 0.8;
  }
}

.search-result {
  padding: $spacing-md;
}

.result-header {
  margin-bottom: $spacing-md;
}

.result-count {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.photographer-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding-bottom: $spacing-xl;
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
  color: $dark-text-tertiary;
}

.empty-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
}
</style>
