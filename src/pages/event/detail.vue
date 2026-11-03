<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <scroll-view scroll-y class="scroll-content" :style="{ paddingTop: statusBarHeight + 'px' }">
      <template v-if="loading">
        <view class="sk-block" style="height: 400rpx; margin: 0 0 24rpx" />
        <view class="sk-block" style="height: 32rpx; width: 60%; margin: 0 24rpx 16rpx" />
        <view class="sk-block" style="height: 24rpx; width: 80%; margin: 0 24rpx 24rpx" />
      </template>

      <template v-else-if="event">
        <!-- Cover：三态兜底。用 <image> 而非 background-image 才能拿到 @load/@error；
             .event-cover-wrap 固定 480rpx，遮罩/返回键/标题都叠在同一个盒子上，
             图片加载失败时版面高度不变。 -->
        <view class="event-cover-wrap">
          <view
            class="event-cover"
            :class="{
              'is-loaded': coverState === 'loaded',
              'is-error': coverState === 'error',
            }"
          >
            <view v-if="coverState === 'loading'" class="cover-skeleton" />
            <image
              v-if="event.cover && coverState !== 'error'"
              class="cover-img"
              :src="event.cover"
              mode="aspectFill"
              @load="onCoverLoad"
              @error="onCoverError"
            />
            <view v-if="coverState === 'error'" class="cover-fallback">
              <text class="cover-fallback-icon">▧</text>
              <text class="cover-fallback-text">封面加载失败</text>
            </view>
          </view>
          <view class="cover-overlay" />
          <view class="back-btn" @click="goBack">
            <text class="back-arrow">‹</text>
          </view>
          <view class="cover-info">
            <text class="cover-name">{{ event.name }}</text>
            <view class="cover-status" v-if="statusText">
              <text class="status-dot" :class="event.status" />
              <text>{{ statusText }}</text>
            </view>
            <view
              class="follow-btn"
              :class="{ followed: isFollowed }"
              @click.stop="onToggleFollow"
            >
              <text>{{ isFollowed ? '已关注' : '＋ 关注' }}</text>
            </view>
          </view>
        </view>

        <view class="detail-body">
          <!-- Date & Location -->
          <view class="info-card">
            <view class="info-row">
              <text class="info-label">时间</text>
              <text class="info-value">{{ formatFullDate(event.startDate) }} - {{ formatFullDate(event.endDate) }}</text>
            </view>
            <view class="info-row">
              <text class="info-label">地点</text>
              <text class="info-value">{{ event.location }} · {{ event.venue }}</text>
            </view>
            <view v-if="event.address" class="info-row">
              <text class="info-label">地址</text>
              <text class="info-value">{{ event.address }}</text>
            </view>
            <view v-if="event.description" class="info-row">
              <text class="info-label">简介</text>
              <text class="info-value desc-text">{{ event.description }}</text>
            </view>
          </view>

          <!-- Tags -->
          <view v-if="event.tags.length" class="tags-section">
            <view v-for="tag in event.tags" :key="tag" class="tag-item">{{ tag }}</view>
          </view>

          <!-- Image Gallery -->
          <view v-if="event.imageGallery && event.imageGallery.length" class="gallery-section">
            <swiper
              class="gallery-swiper"
              :indicator-dots="true"
              :circular="false"
              :autoplay="false"
              :current="0"
            >
              <swiper-item
                v-for="(img, idx) in event.imageGallery.slice(0, 3)"
                :key="idx"
              >
                <view class="gallery-item" :class="{ 'is-loaded': galleryState(idx) === 'loaded' }">
                  <view v-if="galleryState(idx) === 'loading'" class="cover-skeleton" />
                  <image
                    v-if="img && galleryState(idx) !== 'error'"
                    :src="img"
                    class="gallery-img"
                    mode="aspectFill"
                    lazy-load
                    @load="onGalleryImgLoad(idx)"
                    @error="onGalleryImgError(idx)"
                  />
                  <view v-if="galleryState(idx) === 'error'" class="gallery-fallback">
                    <text class="gallery-fallback-icon">▧</text>
                    <text class="gallery-fallback-text">图片加载失败</text>
                  </view>
                </view>
              </swiper-item>
            </swiper>
          </view>

          <!-- Countdown -->
          <view class="countdown-card">
            <template v-if="event.status === 'upcoming'">
              <text class="countdown-num">{{ countdownDays }}</text>
              <text class="countdown-unit">天后开展</text>
            </template>
            <template v-else-if="event.status === 'ongoing'">
              <text class="countdown-live">● 正在举办</text>
            </template>
            <template v-else>
              <text class="countdown-ended">已结束</text>
            </template>
          </view>

          <!-- Photographers -->
          <view class="section">
            <view class="section-header">
              <text class="section-title">参展摄影师</text>
              <text v-if="photographersSource === 'recommended'" class="section-note">推荐</text>
              <text class="section-count">{{ photographers.length }}人</text>
            </view>
            <view class="photographer-grid" v-if="photographers.length">
              <PhotographerCard
                v-for="p in photographers"
                :key="p.id"
                :photographer="p"
                variant="grid"
              />
            </view>
            <view v-else class="empty-hint">还没有摄影师入驻</view>

            <!--
              入驻入口：摄影师能直接向本漫展提交申请；已申请则就地显示状态，
              不必再进二级页才知道自己批没批。非摄影师给出去开通的路径。
            -->
            <view v-if="userStore.user" class="apply-entry" :class="{ mine: applyStatus }" @click="goEventApply">
              <template v-if="!isPhotographer">
                <text class="apply-entry-text">成为摄影师后可申请入驻</text>
                <text class="apply-entry-arrow">›</text>
              </template>
              <template v-else-if="applyStatus">
                <view class="apply-entry-main">
                  <text class="apply-entry-text">{{ applyStatusText }}</text>
                  <text v-if="applyStatus === 'rejected' && applyReason" class="apply-entry-reason">
                    驳回理由：{{ applyReason }}
                  </text>
                </view>
                <text class="apply-entry-tag" :class="`tone-${applyStatus}`">{{ applyStatusLabel }}</text>
              </template>
              <template v-else>
                <text class="apply-entry-text">申请入驻本漫展</text>
                <text class="apply-entry-arrow">›</text>
              </template>
            </view>
            <view v-else class="apply-entry" @click="goLoginForApply">
              <text class="apply-entry-text">登录后可申请入驻本漫展</text>
              <text class="apply-entry-arrow">›</text>
            </view>
          </view>

          <!-- Works -->
          <view v-if="featuredWorks.length" class="section">
            <view class="section-header">
              <text class="section-title">精选作品</text>
            </view>
            <view class="works-grid">
              <view
                v-for="work in featuredWorks"
                :key="work.id"
                class="work-item"
              >
                <image v-if="work.images?.length" :src="work.images[0]" class="work-img" mode="aspectFill" @click.stop="previewFeaturedWork(work, 0)" />
                <view v-else class="work-img work-img--empty"><text>暂无图片</text></view>
                <view class="work-overlay" @click.stop="previewFeaturedWork(work, 0)">
                  <text class="work-title">{{ work.title }}</text>
                  <text class="work-author">by {{ work.photographerName }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </template>

      <template v-else>
        <view class="error-state">
          <text class="error-icon">!</text>
          <text class="error-text">{{ error || '漫展不存在' }}</text>
          <view class="btn-retry" @click="goBack"><text>返回</text></view>
        </view>
      </template>

      <view class="bottom-space" />
    </scroll-view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useStatusBar } from '@/composables/useStatusBar'
import PhotographerCard from '@/components/PhotographerCard.vue'
import type { ComicEvent, Photographer, Work } from '@/types'
import { apiGet } from '@/api/client'
import { getMyEventApplication } from '@/api/index'
import { mapEventItem, mapPhotographerItem, mapWorkItem, encodeRedirect } from '@/utils/mappers'
import { previewImages } from '@/utils/preview'
import { loadFollowedIds, toggleFollow } from '@/utils/follow'
import { useUserStore } from '@/stores/user'

interface EventDetailResponse {
  id: number
  name: string
  location: string
  venue: string
  address?: string
  startDate: string
  endDate: string
  coverUrl: string
  tags: string[]
  imageGallery?: string[]
  status: string
  typeName: string
  description: string
  /** "event" = 真实入驻；"recommended" = 该漫展还没有摄影师入驻，回退平台推荐 */
  photographersSource?: string
  photographers: Array<{
    id: number
    name: string
    avatar: string
    location: string
    rating: number
    reviewCount: number
    orderCount: number
    tags: string[]
  }>
  featuredWorks: Array<{
    id: number
    title: string
    images: string[]
    photographerName: string
  }>
}

const { statusBarHeight } = useStatusBar()
const loading = ref(true)
const error = ref('')

const event = ref<ComicEvent | null>(null)
const photographers = ref<Photographer[]>([])
const featuredWorks = ref<Work[]>([])
/** photographers 的来源，决定标题旁要不要挂「推荐」小字 */
const photographersSource = ref('')

const statusText = ref('')
const countdownDays = ref(0)

// --- 封面三态（与 event/list.vue、calendar/index.vue 同一套口径）---
// 状态完全由 <image> 的 @load/@error 驱动，不额外请求探测图片是否存在。
type CoverState = 'loading' | 'loaded' | 'error'
const coverResult = ref<'loaded' | 'error' | null>(null)

const coverState = computed<CoverState>(() => {
  // 详情页一进来 event 为 null，模板在 loading 分支里，不渲染封面区
  if (!event.value?.cover) return 'error'
  return coverResult.value ?? 'loading'
})

function onCoverLoad() {
  coverResult.value = 'loaded'
}

function onCoverError() {
  coverResult.value = 'error'
}

// --- 相册三态 ---
// loadedGallery / galleryFailed 都是按下标记录；两者都没有 = 仍在加载（显示骨架）。
const loadedGallery = ref<Set<number>>(new Set())
const galleryFailed = ref<Set<number>>(new Set())

function galleryState(idx: number): CoverState {
  if (galleryFailed.value.has(idx)) return 'error'
  return loadedGallery.value.has(idx) ? 'loaded' : 'loading'
}

function onGalleryImgLoad(idx: number) {
  const next = new Set(loadedGallery.value)
  next.add(idx)
  loadedGallery.value = next
}

function onGalleryImgError(idx: number) {
  // 失败时换成占位图标+文案，不再静默留一个空盒子
  const next = new Set(galleryFailed.value)
  next.add(idx)
  galleryFailed.value = next
}

const userStore = useUserStore()
const isFollowed = ref(false)

/** 我在本漫展的入驻申请状态：'' = 未申请，其余为 pending/approved/rejected/withdrawn */
const applyStatus = ref('')
const applyReason = ref('')

const isPhotographer = computed(() => !!userStore.user?.photographerId)

const APPLY_STATUS_LABEL: Record<string, string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '未通过',
  withdrawn: '已撤回'
}

const applyStatusLabel = computed(() => APPLY_STATUS_LABEL[applyStatus.value] || applyStatus.value)

const applyStatusText = computed(() => {
  if (applyStatus.value === 'approved') return '已入驻本漫展'
  if (applyStatus.value === 'rejected') return '入驻申请未通过'
  if (applyStatus.value === 'withdrawn') return '已撤回申请，可重新申请'
  return '入驻申请审核中'
})

let id = ''   // 路由 eventId（string），onMounted 中赋值

onMounted(async () => {

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  id = (currentPage?.options?.id as string) || ''

  if (!id) { error.value = '缺少漫展ID'; loading.value = false; return }

  await loadEvent(id)
  if (event.value) {
    await Promise.all([initFollowState(), loadMyApplication()])
  }
})

async function loadEvent(id: string) {
  try {
    const d = await apiGet<EventDetailResponse>(`/v1/events/${id}`)

    event.value = mapEventItem(d)

    photographers.value = (d.photographers || []).map(mapPhotographerItem)

    photographersSource.value = d.photographersSource || ''

    featuredWorks.value = (d.featuredWorks || []).map(mapWorkItem)

    if (event.value.status === 'ongoing') statusText.value = '进行中'
    else if (event.value.status === 'ended') statusText.value = '已结束'
    else if (event.value.status === 'upcoming') {
      countdownDays.value = Math.max(0, Math.ceil((event.value.startDate - Date.now()) / (1000 * 60 * 60 * 24)))
      statusText.value = ''
    }
  } catch (e: unknown) {
    console.error('Load event detail failed:', e)
    // Fallback: try mock data from store
    const store = (await import('@/stores/home')).useHomeStore()
    if (store.events.length === 0) {
      await store.fetchHomeData()
    }
    const found = store.events.find(ev => String(ev.id) === id)
    if (found) {
      event.value = found
      photographers.value = store.photographers
      featuredWorks.value = store.featuredWorks
      countdownDays.value = Math.max(0, Math.ceil((found.startDate - Date.now()) / 86400000))
      statusText.value = found.status === 'ongoing' ? '进行中' : found.status === 'ended' ? '已结束' : ''
    } else {
      error.value = e instanceof Error ? e.message : '加载失败'
    }
  } finally {
    loading.value = false
  }
}

async function initFollowState() {
  if (!userStore.user) { isFollowed.value = false; return }
  const set = await loadFollowedIds(userStore.user.id)
  isFollowed.value = set.has(id)   // id 是路由 eventId（string）
}

/**
 * 我在本漫展的入驻状态。后端对非摄影师返回 403（mine 也走 resolvePhotographer），
 * 这不是错误 —— 恰恰是「还不是摄影师」这个正常分支，所以按未申请处理，不弹错。
 */
async function loadMyApplication() {
  if (!userStore.user) { applyStatus.value = ''; applyReason.value = ''; return }
  try {
    const res = await getMyEventApplication(id)
    const app = res.application
    applyStatus.value = app?.status || ''
    applyReason.value = app?.reviewReason || ''
  } catch {
    applyStatus.value = ''
    applyReason.value = ''
  }
}

function goEventApply() {
  if (!userStore.user) { goLoginForApply(); return }
  uni.navigateTo({ url: `/pages/photographer/event-apply?id=${id}`, animationType: 'none' })
}

function goLoginForApply() {
  uni.navigateTo({
    url: '/pages/login/index?redirect=' + encodeRedirect('/pages/event/detail?id=' + id)
  })
}

async function onToggleFollow() {
  if (!userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=' + encodeRedirect('/pages/event/detail?id=' + id) }), 800)
    return
  }
  try {
    const newState = await toggleFollow(userStore.user.id, id, isFollowed.value)
    isFollowed.value = newState
    uni.showToast({ title: newState ? '关注成功' : '已取消关注', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}

function formatFullDate(ts: number): string {
  const d = new Date(ts)
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

function goBack() {
  uni.navigateBack()
}

// 精选作品大图预览：单 cover 就传单元素数组，多图可左右滑动
function previewFeaturedWork(work: Work, index = 0) {
  previewImages(work.images, index)
}
</script>

<style lang="scss" scoped>
/*
 * 与首页同构的修复：固定定位 + 内部唯一滚动容器。
 * 旧写法 min-height:100vh + .scroll-content{height:100vh} + 内联 padding-top:statusBarHeight
 * 在 content-box 下会让文档高度溢出视口（视口 844 而 docH 941），window 可被滚 97px，
 * 而状态栏与顶部返回栏是 fixed 不位移 → 滑到底再回顶时它们被顶出屏幕。
 */
.page {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  background: $dark-bg-primary;
}

/*
 * 状态栏底色不透明：旧写法 background: transparent 会让栈底页（首页 hero）
 * 在滑入动画期间透过状态栏区域露出来（用户截图里的橙条就是这么来的）。
 */
.status-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: $dark-bg-primary;
}

/*
 * 唯一滚动容器：吃掉父容器剩余高度（flex 子项必须显式 min-height:0，否则内容撑破父容器）；
 * box-sizing:border-box 让内联的 padding-top（状态栏高度）算进自身高度，不再产生溢出。
 */
.scroll-content {
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
  box-sizing: border-box;
}

// Cover
.event-cover-wrap {
  position: relative;
  width: 100%;
  height: 480rpx;
}

.event-cover {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  // 中性占位底：$surface-2(#1d202f)。原来这里是 $dark-bg-secondary(同色)，
  // 但弱网下整块 480rpx 接近黑会被读成「海报全黑」；骨架层再叠一层浅灰解决。
  background-color: $surface-2;
}

// 图片绝对定位铺满，三态高度恒为 480rpx
.cover-img {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: $surface-2;
  opacity: 0;
  transition: opacity $motion-base;
}

.event-cover.is-loaded .cover-img {
  opacity: 1;
}

// 加载中骨架：低对比灰 + shimmer。纯白 0.04~0.08 叠在 $surface-2 上，
// 沿用 HomeSkeleton.vue 取值（设计系统无骨架专用 token），不新造硬编码色。
.cover-skeleton {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background-color: $surface-2;
  background-image: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.04) 25%,
    rgba(255, 255, 255, 0.08) 50%,
    rgba(255, 255, 255, 0.04) 75%
  );
  background-size: 200% 100%;
  // shimmer 写在基础规则上：小程序 wxss 不支持 prefers-reduced-motion，
  // 放进媒体查询里动画永远不跑。H5 由下方 reduce 分支关闭。
  animation: cover-shimmer 1.5s ease-in-out infinite;
}

@keyframes cover-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

// H5：系统开启「减少动效」时停掉 shimmer（骨架静态仍可见）。小程序端忽略此查询。
@media (prefers-reduced-motion: reduce) {
  .cover-skeleton {
    animation: none;
  }
}

// 兜底：居中图标 + 文案，压在 .cover-overlay/z-index 之下但可见
.cover-fallback {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $s-xs;
  background: $surface-2;
}

.cover-fallback-icon {
  font-size: $t-3xl;
  line-height: 1;
  color: $text-3;
}

.cover-fallback-text {
  font-size: $font-size-sm;
  color: $text-3;
}

.cover-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(transparent 40%, rgba(10, 10, 26, 0.9));
}

.back-btn {
  position: absolute;
  top: 0;
  left: $spacing-md;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.back-arrow {
  font-size: 56rpx;
  color: #fff;
  font-weight: 300;
  text-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.5);
}

.cover-info {
  position: absolute;
  bottom: $spacing-lg;
  left: $spacing-md;
  right: $spacing-md;
}

.cover-name {
  display: block;
  font-size: 44rpx;
  font-weight: 900;
  color: #fff;
  text-shadow: 0 2rpx 12rpx rgba(0, 0, 0, 0.4);
}

.cover-status {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: $spacing-xs;
  font-size: 26rpx;
  color: rgba(255, 255, 255, 0.8);
}

.status-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;

  &.upcoming { background: $neon-cyan; box-shadow: 0 0 8rpx $neon-cyan-glow; }
  &.ongoing { background: $success-color; box-shadow: 0 0 8rpx rgba($success-color, 0.5); }
  &.ended { background: $dark-text-tertiary; }
}

.follow-btn {
  display: inline-flex;
  align-items: center;
  margin-top: $spacing-sm;
  padding: 8rpx 28rpx;
  font-size: 24rpx;
  color: #fff;
  background: rgba(168, 85, 247, 0.25);
  border: 1rpx solid $neon-purple;
  border-radius: $border-radius-xl;
  transition: all 0.2s;

  &.followed {
    background: $neon-gradient;
    border-color: transparent;
    box-shadow: 0 0 12rpx $neon-purple-glow;
  }
}

// Body
.detail-body {
  padding: 0 $spacing-md;
  margin-top: -32rpx;
  position: relative;
  z-index: 5;
}

// Info Card
.info-card {
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-lg;
  padding: $spacing-md;
}

.info-row {
  display: flex;
  gap: $spacing-sm;
  padding: $spacing-xs 0;

  & + & {
    border-top: 1rpx solid $dark-border;
  }
}

.info-label {
  font-size: 26rpx;
  color: $neon-cyan;
  font-weight: 600;
  flex-shrink: 0;
  width: 64rpx;
}

.info-value {
  font-size: 26rpx;
  color: $dark-text-primary;
  line-height: 1.5;
}

.desc-text {
  color: $dark-text-secondary;
  font-size: 24rpx;
}

// Tags
.tags-section {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-md;
}

.tag-item {
  padding: 6rpx 20rpx;
  font-size: 22rpx;
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 1rpx solid rgba($neon-purple, 0.3);
  border-radius: $border-radius-xl;
}

// Gallery
.gallery-section {
  margin-top: $spacing-md;
}

.gallery-swiper {
  height: 240rpx;
  border-radius: $border-radius-md;
  overflow: hidden;
  border: 1rpx solid $dark-border;
}

.gallery-item {
  position: relative;
  width: 100%;
  height: 100%;
  // 中性占位而非近黑（$surface-2 = #1d202f）
  background: $surface-2;
}

.gallery-img {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: $surface-2;
  opacity: 0;
  transition: opacity $motion-base;
}

.gallery-item.is-loaded .gallery-img {
  opacity: 1;
}

.gallery-fallback {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $s-xs;
  background: $surface-2;
}

.gallery-fallback-icon {
  font-size: $t-2xl;
  line-height: 1;
  color: $text-3;
}

.gallery-fallback-text {
  font-size: $font-size-xs;
  color: $text-3;
}

// Countdown
.countdown-card {
  display: flex;
  align-items: baseline;
  justify-content: center;
  padding: $spacing-lg 0;
  margin-top: $spacing-md;
  background: $dark-bg-card;
  border: 1rpx solid $neon-purple-glow;
  border-radius: $border-radius-lg;
}

.countdown-num {
  font-size: 72rpx;
  font-weight: 900;
  color: $neon-purple;
  line-height: 1;
}

.countdown-unit {
  font-size: 32rpx;
  color: $dark-text-primary;
  margin-left: $spacing-xs;
  font-weight: 600;
}

.countdown-live {
  font-size: 36rpx;
  color: $success-color;
  font-weight: 700;
}

.countdown-ended {
  font-size: 32rpx;
  color: $dark-text-tertiary;
  font-weight: 600;
}

// Sections
.section {
  margin-top: $spacing-xl;
}

.section-header {
  display: flex;
  align-items: flex-end;
  gap: $spacing-sm;
  margin-bottom: $spacing-md;
}

.section-title {
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
  padding-left: 20rpx;
  position: relative;

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

.section-count {
  font-size: 24rpx;
  color: $dark-text-tertiary;
  padding-bottom: 2rpx;
}

/* 两列卡片网格：显式 stretch，保证同一排卡片等高；
 * 卡片本身带统一 min-height（见 PhotographerCard 的 .variant-grid），
 * 因此跨排也保持一致的高度下限，内容超出时同排自动拉齐。
 * minmax(0, 1fr)：1fr 轨道的最小尺寸默认是 auto(=min-content)，
 * 遇到超长昵称会把所在轨道撑宽、把另一列挤窄，导致同排两列宽度不一致；
 * 用 minmax(0, 1fr) 打断 min-content 下限，溢出交给卡片内部省略号处理。 */
.photographer-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: stretch;
  gap: $spacing-md;
}

.empty-hint {
  text-align: center;
  padding: $spacing-xl;
  color: $dark-text-tertiary;
  font-size: $font-size-sm;
}

/* 「推荐」小字：photographersSource === 'recommended' 时提示这不是真实入驻名单，
 * 而是平台推荐位，避免用户误以为该漫展已经有摄影师报名。 */
.section-note {
  font-size: 22rpx;
  color: $neon-cyan;
  border: 1rpx solid rgba(6, 182, 212, 0.4);
  border-radius: 8rpx;
  padding: 0 10rpx;
  line-height: 30rpx;
  margin-bottom: 2rpx;
}

// 入驻入口
.apply-entry {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-md;
  padding: 24rpx 28rpx;
  background: $dark-bg-card;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-md;
  transition: background 0.2s ease;

  &:active {
    background: $dark-bg-card-hover;
    transform: scale(0.99);
  }
}

.apply-entry-main {
  flex: 1;
  min-width: 0;
}

.apply-entry-text {
  font-size: 26rpx;
  color: $neon-cyan;
  font-weight: 600;
}

.apply-entry-reason {
  display: block;
  font-size: 22rpx;
  color: $error-color;
  line-height: 1.5;
  margin-top: 6rpx;
}

.apply-entry-arrow {
  font-size: 32rpx;
  color: $dark-text-tertiary;
  line-height: 1;
}

.apply-entry-tag {
  flex-shrink: 0;
  font-size: 22rpx;
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  border: 1rpx solid $dark-border;

  &.tone-pending {
    color: $neon-cyan;
    border-color: rgba(6, 182, 212, 0.45);
  }

  &.tone-approved {
    color: $success-color;
    border-color: rgba(34, 197, 94, 0.45);
  }

  &.tone-rejected {
    color: $error-color;
    border-color: rgba(239, 68, 68, 0.45);
  }

  &.tone-withdrawn {
    color: $dark-text-tertiary;
  }
}

// Works
.works-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-sm;
}

.work-item {
  position: relative;
  border-radius: $border-radius-md;
  overflow: hidden;
  aspect-ratio: 4/3;

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border: 1rpx solid rgba($neon-purple, 0.12);
    border-radius: $border-radius-md;
    pointer-events: none;
    z-index: 2;
  }
}

.work-img {
  width: 100%;
  height: 100%;
}

.work-img--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22rpx;
  color: $text-3;
}

.work-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: $spacing-sm;
  background: linear-gradient(transparent, rgba(10, 10, 26, 0.85));
}

.work-title {
  display: block;
  font-size: 24rpx;
  color: #fff;
  font-weight: 600;
}

.work-author {
  display: block;
  font-size: 20rpx;
  color: rgba(255, 255, 255, 0.6);
  margin-top: 2rpx;
}

// Error
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2 $spacing-md;
}

.error-icon {
  width: 80rpx;
  height: 80rpx;
  line-height: 80rpx;
  text-align: center;
  font-size: 48rpx;
  font-weight: 900;
  color: $dark-text-tertiary;
  border: 2rpx solid $dark-border;
  border-radius: 50%;
  margin-bottom: $spacing-md;
}

.error-text {
  font-size: $font-size-base;
  color: $dark-text-secondary;
  margin-bottom: $spacing-lg;
}

.btn-retry {
  padding: 14rpx 48rpx;
  background: $neon-gradient;
  color: #fff;
  border-radius: $border-radius-xl;
  font-size: $font-size-base;
  font-weight: 600;

  &:active { opacity: 0.8; }
}

// Skeleton
.sk-block {
  background: rgba(255, 255, 255, 0.04);
  border-radius: 16rpx;
}

.bottom-space {
  height: 48rpx;
}
</style>
