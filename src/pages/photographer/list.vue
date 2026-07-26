<template>
  <view class="page">
    <view class="filter-bar">
      <scroll-view scroll-x class="filter-scroll" show-scrollbar="false">
        <view class="filter-list">
          <view 
            v-for="item in filters" 
            :key="item.key" 
            class="filter-item"
            :class="{ active: currentFilter === item.key }"
            @click="setFilter(item.key)"
          >
            {{ item.label }}
            <text v-if="item.key === currentFilter" class="filter-arrow">▼</text>
          </view>
        </view>
      </scroll-view>
    </view>

    <scroll-view 
      scroll-y 
      class="content"
      @scrolltolower="loadMore"
    >
      <view class="photographer-list">
        <PhotographerCard 
          v-for="p in photographers" 
          :key="p.id" 
          :photographer="p"
        />
      </view>
      
      <view v-if="loading" class="loading">
        <text>加载中...</text>
      </view>
      
      <view v-if="!loading && photographers.length >= total" class="no-more">
        <text>没有更多了</text>
      </view>
      
      <view class="bottom-space"></view>
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotographerCard from '@/components/PhotographerCard.vue'
import { getPhotographers } from '@/api/index'
import type { Photographer } from '@/types'

const photographers = ref<Photographer[]>([])
const currentPage = ref(1)
const total = ref(0)
const loading = ref(false)
const currentFilter = ref('all')

const filters = [
  { key: 'all', label: '全部' },
  { key: 'hot', label: '热门' },
  { key: 'rating', label: '评分最高' },
  { key: 'order', label: '接单最多' },
  { key: 'new', label: '最新入驻' }
]

onMounted(() => {
  loadData()
})

async function loadData(page = 1) {
  loading.value = true
  try {
    const res = await getPhotographers({
      page,
      size: 10
    })
    if (page === 1) {
      photographers.value = res.list
    } else {
      photographers.value = [...photographers.value, ...res.list]
    }
    total.value = res.total
    currentPage.value = page
  } finally {
    loading.value = false
  }
}

function loadMore() {
  if (loading.value || photographers.value.length >= total.value) return
  loadData(currentPage.value + 1)
}

function setFilter(key: string) {
  currentFilter.value = key
  currentPage.value = 1
  photographers.value = []
  loadData()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
}

.filter-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  background: $dark-bg-secondary;
  z-index: 100;
  padding-top: env(safe-area-inset-top);
  border-bottom: 1rpx solid $dark-border;
}

.filter-scroll {
  white-space: nowrap;
}

.filter-list {
  display: inline-flex;
  padding: $spacing-sm $spacing-md;
  gap: $spacing-sm;
}

.filter-item {
  display: flex;
  align-items: center;
  font-size: $font-size-sm;
  color: $dark-text-secondary;
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-xl;
  border: 1rpx solid transparent;
  transition: all 0.15s;

  &.active {
    background: rgba($neon-purple, 0.15);
    color: $neon-purple;
    border-color: rgba($neon-purple, 0.3);
  }

  &:active {
    background: $dark-bg-card-hover;
  }
}

.filter-arrow {
  font-size: $font-size-xs;
  margin-left: 4rpx;
}

.content {
  height: 100vh;
  padding-top: calc(env(safe-area-inset-top) + 80rpx);
}

.photographer-list {
  padding: $spacing-md;
}

.loading, .no-more {
  text-align: center;
  padding: $spacing-md;
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
}

.bottom-space {
  height: 120rpx;
}
</style>
