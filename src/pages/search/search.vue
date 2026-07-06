<template>
  <view class="page">
    <view class="search-bar">
      <view class="search-input-wrap">
        <text class="search-icon">🔍</text>
        <input 
          class="search-input" 
          v-model="keyword"
          placeholder="搜索摄影师、标签"
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

    <view v-else class="search-result">
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
        <text class="empty-icon">📭</text>
        <text class="empty-text">没有找到相关摄影师</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotographerCard from '@/components/PhotographerCard.vue'
import { getPhotographers, getTags } from '@/api/index'
import type { Photographer } from '@/types'

const keyword = ref('')
const tags = ref<string[]>([])
const selectedTags = ref<string[]>([])
const photographers = ref<Photographer[]>([])

const hotKeywords = ['原神', '鬼灭之刃', '古风', '日系', '暗黑', '漫展跟拍']

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
  const res = await getTags()
  tags.value = res
}

async function handleSearch(kw?: string) {
  const searchKeyword = kw || keyword.value
  if (!searchKeyword && selectedTags.value.length === 0) return
  
  uni.showLoading({ title: '搜索中...' })
  try {
    const res = await getPhotographers({
      keyword: searchKeyword,
      tags: selectedTags.value.length > 0 ? selectedTags.value : undefined
    })
    photographers.value = res.list
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
  photographers.value = []
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $bg-page;
}

.search-bar {
  display: flex;
  align-items: center;
  padding: $spacing-md;
  background: $bg-primary;
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  background: $bg-tertiary;
  border-radius: $border-radius-lg;
  padding: $spacing-sm $spacing-md;
}

.search-icon {
  font-size: $font-size-base;
  margin-right: $spacing-xs;
}

.search-input {
  flex: 1;
  font-size: $font-size-base;
}

.clear-btn {
  font-size: $font-size-sm;
  color: $text-tertiary;
  padding: $spacing-xs;
}

.cancel-btn {
  font-size: $font-size-base;
  color: $text-secondary;
  margin-left: $spacing-md;
}

.hot-search {
  padding: $spacing-md;
}

.section-header {
  margin-bottom: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 600;
}

.hot-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.hot-item {
  padding: $spacing-sm $spacing-md;
  background: $bg-primary;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  color: $text-secondary;
  
  &:active {
    background: rgba($primary-color, 0.1);
    color: $primary-color;
  }
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.tag-item {
  padding: $spacing-sm $spacing-md;
  background: $bg-primary;
  border-radius: $border-radius-xl;
  font-size: $font-size-sm;
  color: $text-secondary;
  
  &.active {
    background: $primary-color;
    color: #fff;
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
  color: $text-tertiary;
}

.photographer-list {
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
}

.empty-text {
  font-size: $font-size-base;
  color: $text-tertiary;
}
</style>
