<template>
  <view class="page">
    <template v-if="firstLoading">
      <view class="skeleton-page">
        <view class="sk-line sk-w-40" />
        <view class="sk-grid"><view class="sk-block sk-grid-item" /><view class="sk-block sk-grid-item" /><view class="sk-block sk-grid-item" /><view class="sk-block sk-grid-item" /></view>
        <ListSkeleton :rows="2" variant="card" />
      </view>
    </template>
    <template v-else>
    <view class="gallery">
      <view class="gallery-header">
        <!-- 标题必须是动态的：从摄影师详情带 photographerId 进来时看的是「该摄影师的作品」，
             从「我的 → 我的作品」进来才是「我的作品」。此前无条件写死「摄影作品」，
             而数据在无 pid 时拉的是首页 featuredWorks（其他摄影师的作品），
             用户看到的就是「我的作品里全是不认识的人拍的照片」。 -->
        <text class="gallery-title">{{ pageTitle }}</text>
        <text class="gallery-count">{{ works.length }} 作品</text>
      </view>
      
      <view class="gallery-grid">
        <view 
          v-for="(work, index) in works" 
          :key="work.id" 
          class="gallery-item"
          :class="{ 'full-width': (work.images?.length || 0) >= 2 }"
          @click="previewImage(index)"
        >
          <image v-if="work.images?.length" :src="work.images[0]" class="gallery-image" mode="aspectFill" lazy-load />
          <view v-else class="gallery-image gallery-image--empty"><text>暂无图片</text></view>
          <view class="gallery-overlay">
            <text class="gallery-title-text">{{ work.title }}</text>
            <view class="gallery-tags">
              <text 
                v-for="tag in (work.tags || []).slice(0, 2)" 
                :key="tag" 
                class="gallery-tag"
              >{{ tag }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="works.length === 0" class="empty">
        <text class="empty-icon">◆</text>
        <text class="empty-text">{{ emptyHint }}</text>
      </view>
    </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { apiGet } from '@/api/client'
import { mapWorkItem } from '@/utils/mappers'
import { useUserStore } from '@/stores/user'
import type { Work } from '@/types'
import ListSkeleton from '@/components/ListSkeleton.vue'

/** 首屏深色骨架：数据就绪前置 true，消除切页白屏（B 档） */
const firstLoading = ref(true)

const works = ref<Work[]>([])
// 空态文案要说清楚「为什么空」，别只说「暂无作品」
const emptyHint = ref('暂无作品')
// 是不是「某个摄影师的作品集」视图（由 photographerId 参数决定）
const scopedPhotographerId = ref<string>('')
// 是否从「我的 → 我的作品」进来（无 pid 且已登录，展示自己的作品）
const isMine = ref(false)

const pageTitle = computed(() => {
  if (scopedPhotographerId.value) return '摄影师作品'
  if (isMine.value) return '我的作品'
  return '摄影作品'
})

onMounted(() => {
  loadData()
})

async function loadData() {
  try {
    const pages = getCurrentPages()
    const currentPage = pages[pages.length - 1] as any
    const pid = currentPage?.options?.photographerId as string | undefined
    if (pid) {
      scopedPhotographerId.value = pid
      const res = await apiGet<any>(`/v1/photographers/${pid}`)
      works.value = (res.works || []).map(mapWorkItem)
      return
    }

    // 无 pid = 「我的作品」入口：必须拿到自己的摄影师档案 id，再拉自己的作品。
    // 绝不回落到 /v1/home 的 featuredWorks —— 那是别人的作品，拿来当「我的作品」
    // 就是数据串台（用户原话「我还没登录为什么会有我的作品」）。
    isMine.value = true
    const userStore = useUserStore()
    const myId = userStore.user?.photographerId
    if (!userStore.isLoggedIn || !myId) {
      // 已登录但没有摄影师档案，或压根没登录：一句话说清楚，不给假数据。
      works.value = []
      emptyHint.value = userStore.isLoggedIn
        ? '你还没有摄影师档案，也没有发布过作品'
        : '登录后查看你的作品'
      return
    }
    const res = await apiGet<any>(`/v1/photographers/${myId}`)
    works.value = (res.works || []).map(mapWorkItem)
    if (!works.value.length) emptyHint.value = '你还没有发布作品'
  } catch {
    uni.showToast({ title: '加载失败，请稍后再试', icon: 'none' })
  } finally {
    firstLoading.value = false
  }
}

function previewImage(index: number) {
  const currentWork = works.value[index]
  if (!currentWork?.images?.length) return
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
  display: block;
  width: 100%;
  aspect-ratio: 3 / 4;
  background: $surface-2;

  /* 双图作品横跨两列，用更宽的取景比例，避免 3:4 拉成巨幅 */
  .full-width & {
    aspect-ratio: 4 / 3;
  }
}

/* 老安卓 WebView 不支持 aspect-ratio 时给固定高度兜底，避免塌成 0 高 */
@supports not (aspect-ratio: 1 / 1) {
  .gallery-image {
    height: 440rpx;
    .full-width & { height: 330rpx; }
  }
}
.gallery-image--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24rpx;
  color: $text-3;
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
