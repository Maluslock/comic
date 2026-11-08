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
          confirm-type="search"
          @confirm="submitSearch"
        />
        <text v-if="keyword" class="clear-btn" @click="clearKeyword">✕</text>
      </view>
      <text class="cancel-btn" @click="goBack">取消</text>
    </view>

    <view class="hot-search">
      <view class="section-header">
        <text class="section-title">热门搜索</text>
      </view>
      <view class="hot-list">
        <view
          v-for="(item, index) in hotKeywords"
          :key="index"
          class="hot-item"
          @click="goResult({ tags: [item] })"
        >
          {{ item }}
        </view>
      </view>

      <view class="section-header mt-lg">
        <text class="section-title">热门标签</text>
        <text class="multi-toggle" :class="{ active: multiSelect }" @click="toggleMultiMode">
          {{ multiSelect ? '退出多选' : '多选' }}
        </text>
      </view>
      <view class="tags-container">
        <view
          v-for="tag in tags"
          :key="tag"
          class="tag-item"
          :class="{ active: selectedTags.includes(tag) }"
          @click="onTapTag(tag)"
        >
          {{ tag }}
        </view>
      </view>

      <view v-if="multiSelect" class="action-row">
        <view
          class="action-btn"
          :class="{ disabled: selectedTags.length === 0 }"
          @click="submitTags"
        >
          按 {{ selectedTags.length }} 个标签搜索
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { apiGet } from '@/api/client'
import { navTo } from '@/utils/nav'

const keyword = ref('')
const tags = ref<string[]>([])
const selectedTags = ref<string[]>([])
const multiSelect = ref(false)

/**
 * 热门搜索词表：这 6 个词全部是库里真实存在、且确有摄影师引用的标签。
 * 后端 keyword 通道（photographers.sql.go:91）只对 p.name / p.description 做 ILIKE，不碰标签表，
 * 所以只作为标签存在的词（原神、鬼灭之刃、动漫、影视、游戏…）走 keyword 必然 0 命中；
 * 而 tags 通道是 `t2.name = ANY(string_to_array($3, ','))` 精确匹配，命中率 100%。
 * 因此这批热词点击时统一走标签通道 goResult({ tags: [item] })，不走 keyword 通道。
 */
const hotKeywords = ['古风', '日系', '暗黑', '仙侠', '哥特', '汉服']
// /v1/tags 取不到时的兜底：只放确认存在于标签库的名字，入口页不允许点出 0 结果
const fallbackTags = ['古风', '日系', '暗黑', '仙侠', '哥特', '汉服', '少女', '清新']

/**
 * 小程序 navigateTo 的 options 是**未解码原串**（H5 hash 路由由框架自动解码），
 * 所以这里统一只解一层：值里含 % 才 decodeURIComponent，失败返回原串。
 */
function safeDecode(value?: string): string {
  const raw = (value ?? '').trim()
  if (!raw.includes('%')) return raw
  try {
    return decodeURIComponent(raw)
  } catch {
    return raw
  }
}

/** `tags` 是标签名用逗号拼接后整体编码：先整体解一层，再按逗号切，各段不再二次解码 */
function parseTags(value?: string): string[] {
  return safeDecode(value)
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.length > 0)
}

/** 结果页 URL：keyword / tags 都由这里统一编码，避免调用点各写一套 */
function buildResultUrl(kw: string, tagList: string[]): string {
  const query: string[] = []
  if (kw) query.push(`keyword=${encodeURIComponent(kw)}`)
  if (tagList.length > 0) query.push(`tags=${encodeURIComponent(tagList.join(','))}`)
  return `/pages/search/result?${query.join('&')}`
}

onLoad((options: Record<string, string> | undefined) => {
  // 小程序 options 是未解码原串，H5 hash 路由已被框架解过一层；safeDecode 只解一层，两端一致
  const kw = safeDecode(options?.keyword)
  const tagList = parseTags(options?.tags)

  // 首页「热门风格」等带参入口：条件已经定死，直接落到独立结果页（redirectTo 不涨页面栈，返回即回首页）
  if (kw || tagList.length > 0) {
    keyword.value = kw
    selectedTags.value = tagList
    multiSelect.value = tagList.length > 1
    const url = buildResultUrl(kw, tagList)
    // redirectTo 失败时退回 navigateTo，保证带参入口不会停在空白入口页
    uni.redirectTo({ url, fail: () => navTo(url) })
    return
  }

  void loadTags()
})

async function loadTags() {
  try {
    const res = await apiGet<Array<{ id: number; name: string }>>('/v1/tags')
    const names = (res || []).map((t) => t.name).filter(Boolean)
    tags.value = names.length > 0 ? names : fallbackTags
  } catch {
    tags.value = fallbackTags
  }
}

/** 搜索入口页只负责收集条件，结果一律在独立结果页展示 */
function goResult(params: { keyword?: string; tags?: string[] }) {
  const kw = (params.keyword ?? '').trim()
  const tagList = params.tags ?? []
  if (!kw && tagList.length === 0) {
    uni.showToast({ title: '请输入关键词或选择标签', icon: 'none' })
    return
  }
  navTo(buildResultUrl(kw, tagList))
}

function submitSearch() {
  goResult({ keyword: keyword.value, tags: selectedTags.value })
}

function submitTags() {
  if (selectedTags.value.length === 0) {
    uni.showToast({ title: '请先选择标签', icon: 'none' })
    return
  }
  goResult({ tags: [...selectedTags.value] })
}

function onTapTag(tag: string) {
  // 非多选模式：点标签直接出结果页
  if (!multiSelect.value) {
    goResult({ tags: [tag] })
    return
  }
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
}

function toggleMultiMode() {
  multiSelect.value = !multiSelect.value
  if (!multiSelect.value) selectedTags.value = []
}

function clearKeyword() {
  keyword.value = ''
  selectedTags.value = []
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  background: $dark-bg-primary;
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
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  // 与首页 .nav-search-box 保持一致：磨砂玻璃输入框，靠边框与页面/条带拉出边界
  background: rgba(21, 23, 34, 0.68);
  border: 1rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 32rpx;
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
  color: $text-2;
  margin-left: $spacing-md;

  &:active {
    opacity: 0.6;
  }
}

.hot-search {
  padding: $spacing-md;
  padding-bottom: calc(#{$spacing-xl} + env(safe-area-inset-bottom));
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.multi-toggle {
  font-size: $font-size-sm;
  color: $dark-text-tertiary;
  padding: $spacing-xs $spacing-sm;
  border: 1rpx solid $dark-border;
  border-radius: $border-radius-xl;

  &.active {
    color: $neon-purple;
    border-color: $neon-purple;
    background: $neon-purple-dim;
  }

  &:active {
    opacity: 0.7;
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

.action-row {
  margin-top: $spacing-lg;
}

.action-btn {
  text-align: center;
  padding: $spacing-md;
  border-radius: $border-radius-lg;
  font-size: $font-size-base;
  font-weight: 500;
  color: #fff;
  background: $neon-purple;

  &.disabled {
    color: $dark-text-tertiary;
    background: $dark-bg-card;
    border: 1rpx solid $dark-border;
  }

  &:active {
    opacity: 0.85;
  }
}
</style>
