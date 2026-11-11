<template>
  <page-meta
    root-background-color="#0c0d14"
    background-color="#0c0d14"
    background-color-top="#0c0d14"
    background-color-bottom="#0c0d14"
  />
  <view class="page">
    <!-- Status Bar Spacer -->
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />

    <!-- Pull-to-Refresh Wrapper -->
    <scroll-view
      :scroll-y="!showCityPicker"
      class="scroll-content"
      :style="{ paddingTop: statusBarHeight + 'px' }"
      :refresher-enabled="true"
      :refresher-triggered="store.refreshing"
      :refresher-threshold="80"
      @refresherrefresh="onRefresh"
    >
      <!-- Skeleton Loading -->
      <HomeSkeleton v-if="store.loading && !store.loaded" />

      <!-- 首屏失败：给明确的错误态，而不是渲染一个空壳页面 -->
      <view v-else-if="store.error && !store.loaded" class="error-state">
        <text class="error-icon">!</text>
        <text class="error-text">{{ store.error }}</text>
        <view class="btn-retry" @click="store.fetchHomeData()">
          <text>重新加载</text>
        </view>
      </view>

      <template v-else>
        <!-- Hero Stage -->
        <view class="hero-stage">
          <image class="hero-media" :src="heroImage" mode="aspectFill" />
          <view class="hero-scrim" />
          <view class="hero-glow" />
          <view class="vf-corner vf-tl" />
          <view class="vf-corner vf-tr" />

          <view class="hero-content">
            <!--
              顶部栏单行（视觉方案 A · 极致通透）：
              左 = 纯文字城市 —— 城市不再独占一整行，也不再套玻璃胶囊（调研 9 家平台 0 家这么做）；
              右 = 假搜索框 flex:1，吃满城市按钮与微信胶囊之间的全部宽度，右边界止于胶囊左侧 8px。
              铃铛已从顶栏移除：未读由底栏「消息」Tab 角标承担（src/utils/badge.ts 的
              refreshMessageBadge → uni.setTabBarBadge(index: 2)，读的正是首页原先那个
              /v1/chat/unread 接口），顶栏再挂一个是重复曝光。
            -->
            <view class="hero-nav-bar" :style="navBarStyle">
              <view class="nav-city-btn" @click="goCitySelect">
                <text class="nav-city-name">{{ currentCity }}</text>
                <text class="nav-city-caret">▾</text>
              </view>
              <view class="nav-search-box" @click="goSearch">
                <image class="search-ico" src="/static/icons/search.svg" mode="aspectFit" />
                <text class="search-ph">搜索漫展 / 摄影师</text>
                <view class="search-action-pill">搜索</view>
              </view>
            </view>

            <view class="hero-copy">
              <text class="hero-eyebrow">米拉漫展 · NEXT UP</text>
              <text class="hero-title">{{ heroEvent ? heroEvent.name : '米拉漫展' }}</text>
              <view v-if="!heroEvent" class="hero-meta">
                <text class="hero-meta-venue">找到你的专属摄影师</text>
              </view>
            </view>
          </view>
        </view>

        <!-- Ticket dock -->
        <view v-if="nearestEvent" class="ticket" @click="goEventDetail(nearestEvent.id)">
          <view class="ticket-stub">
            <view class="ticket-stub-row">
              <text class="ticket-label">{{ nearestStarted ? '正在开展' : '距开展' }}</text>
              <text class="ticket-days">{{ nearestStarted ? '进行中' : store.countdownDays(nearestEvent.startDate) }}</text>
              <text v-if="!nearestStarted" class="ticket-unit">天</text>
            </view>
            <text class="ticket-meta">{{ store.formatDate(nearestEvent.startDate) }} · {{ nearestEvent.venue || nearestEvent.location }}</text>
          </view>
          <view class="ticket-rip" />
          <view class="ticket-right">
            <view class="ticket-cta">
              <text>查看漫展</text>
              <text class="ticket-arrow">›</text>
            </view>
          </view>
        </view>

        <!-- Quick Action Bar -->
        <view class="section-compact">
          <QuickActionBar />
        </view>

        <!-- Upcoming Events -->
        <view class="section">
          <view class="section-header">
            <text class="section-title">近期漫展</text>
            <text class="section-more" @click="goEventMore">更多 ›</text>
          </view>
          <view class="range-chips">
            <view
              v-for="opt in rangeOptions"
              :key="opt.key"
              class="range-chip"
              :class="{ active: eventRange === opt.key }"
              @click="eventRange = opt.key"
            >{{ opt.label }}</view>
          </view>
          <view v-if="!filteredEvents.length" class="empty-inline">
            <text class="empty-inline-title">{{ emptyState.title }}</text>
            <text v-if="emptyState.sub" class="empty-inline-sub" @click="onEmptyAction">{{ emptyState.sub }}</text>
          </view>
          <scroll-view v-else scroll-x class="event-scroll" :show-scrollbar="false">
            <view class="event-list">
              <view
                v-for="event in filteredEvents"
                :key="event.id"
                class="event-card"
                @click="goEventDetail(event.id)"
              >
                <image class="event-cover" lazy-load :src="event.cover" mode="aspectFill" />
                <view class="event-countdown-badge">
                  <text>{{ store.countdownLabel(event.startDate, event.endDate) }}</text>
                </view>
                <view class="event-info">
                  <text class="event-name ellipsis">{{ event.name }}</text>
                  <view class="event-meta">
                    <text class="event-location">{{ event.location }} · {{ event.venue }}</text>
                  </view>
                  <view class="event-footer">
                    <text class="event-date">{{ store.formatDate(event.startDate) }} - {{ store.formatDate(event.endDate) }}</text>
                  </view>
                </view>
              </view>
            </view>
          </scroll-view>
        </view>

        <!-- Poster Wall -->
        <view v-if="posterWall.length" class="section">
          <view class="section-header">
            <text class="section-title">漫展海报</text>
            <text class="section-more" @click="goEventMore">更多 ›</text>
          </view>
          <view class="works-grid">
            <view
              v-for="ev in posterWall"
              :key="ev.id"
              class="work-card"
              @click="goEventDetail(ev.id)"
            >
              <image class="work-img" lazy-load :src="ev.cover" mode="aspectFill" />
              <view class="work-scrim" />
              <view class="work-meta">
                <text class="work-title">{{ ev.name }}</text>
                <text class="work-author">{{ formatShortDate(ev.startDate) }} · {{ ev.location }}</text>
              </view>
            </view>
          </view>
        </view>

        <!-- Hot Tags -->
        <view class="section-compact">
          <view class="section-header">
            <text class="section-title">热门风格</text>
            <text class="section-more" @click="goSearch">更多 ›</text>
          </view>
          <view class="tags-row">
            <view
              v-for="tag in store.hotTags"
              :key="tag"
              class="tag-item"
              @click="goSearchByTag(tag)"
            >
              {{ tag }}
            </view>
          </view>
        </view>

        <!-- Recommended Photographers -->
        <view class="section">
          <view class="section-header">
            <text class="section-title">推荐摄影师</text>
            <text class="section-more" @click="goPhotographerList">更多 ›</text>
          </view>
          <view v-if="filteredPhotographers.length" class="photographer-grid">
            <PhotographerCard
              v-for="p in filteredPhotographers.slice(0, 4)"
              :key="p.id"
              :photographer="p"
              variant="grid"
            />
          </view>
          <view v-else class="empty-inline">
            <text class="empty-inline-title">{{ currentCity }} 暂无入驻摄影师</text>
            <text class="empty-inline-sub" @click="selectCity('全部')">看看全部摄影师 ›</text>
          </view>
        </view>

        <view class="bottom-space" />
      </template>
    </scroll-view>

    <!-- City Selector Popup -->
    <!-- BUG-3: mask 只 stop 冒泡防穿透，切勿再加 .prevent——它会 preventDefault 掉
         city-list 宿主（overflow:hidden auto）的原生触摸滚动，导致整列滑不动。 -->
    <view v-if="showCityPicker" class="city-popup-mask" @click="closeCityPicker" @touchmove.stop>
      <view class="city-popup" @click.stop>
        <view class="city-popup-header">
          <text class="city-popup-title">选择城市</text>
          <text class="city-popup-close" @click="closeCityPicker">✕</text>
        </view>
        <!-- 搜索框独立成行：原实现紧贴 header 的下边框，两条 1rpx 边叠在一起，
             用户看到的就是「搜索框上方多出一条线」。现在 header 无边框、搜索框自带上边框替代分隔。 -->
        <view class="city-search-row">
          <input
            v-model="cityKeyword"
            class="city-search"
            type="text"
            placeholder="搜索城市"
            placeholder-class="city-search-ph"
          />
        </view>
        <!-- mp-weixin 由 scroll-view 自身 catchtouchmove 独立承接，不依赖祖先 mask 的
             catch；H5 端仅 stopPropagation，不碰 preventDefault，滚动照常。 -->
        <scroll-view class="city-list" scroll-y @touchmove.stop>
          <!-- 热门城市：取场次最多的前 8 个（数据来自 /v1/cities 的 eventCount） -->
          <view v-if="!cityKeyword.trim() && hotCities.length" class="city-section">
            <text class="city-section-title">热门城市</text>
            <view class="city-grid">
              <view
                v-for="city in hotCities"
                :key="'hot-' + city"
                class="city-chip"
                :class="{ active: currentCity === city }"
                @click="selectCity(city)"
              >
                {{ city }}
              </view>
            </view>
          </view>
          <view v-if="!cityKeyword.trim()" class="city-section">
            <text class="city-section-title">全部城市<text class="city-section-count">· {{ cityOptionsCount }} 个</text></text>
            <view class="city-grid">
              <view
                v-for="city in restCities"
                :key="city"
                class="city-chip"
                :class="{ active: currentCity === city }"
                @click="selectCity(city)"
              >
                {{ city }}
              </view>
            </view>
          </view>
          <!-- 有搜索词时退回单列列表，扫读更快 -->
          <view v-else class="city-plain-list">
            <view
              v-for="city in filteredCityOptions"
              :key="city"
              class="city-option"
              :class="{ active: currentCity === city }"
              @click="selectCity(city)"
            >
              <text>{{ city }}</text>
              <text v-if="currentCity === city" class="city-check">✓</text>
            </view>
            <view v-if="!filteredCityOptions.length" class="city-empty">没有匹配的城市</view>
          </view>
        </scroll-view>
      </view>
    </view>

    <!-- 首启风格偏好弹层：仅登录、未选择、未跳过的用户在 onShow 触发 -->
    <StylePicker
      :visible="showStylePicker"
      @save="onStyleSave"
      @skip="onStyleSkip"
      @update:visible="showStylePicker = $event"
    />
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useStatusBar } from '@/composables/useStatusBar'
import { onShow, onReady } from '@dcloudio/uni-app'
import { storeToRefs } from 'pinia'
import PhotographerCard from '@/components/PhotographerCard.vue'
import QuickActionBar from '@/components/QuickActionBar.vue'
import HomeSkeleton from '@/components/HomeSkeleton.vue'
import { useHomeStore } from '@/stores/home'
import { useStyleStore } from '@/stores/style'
import StylePicker from '@/components/StylePicker.vue'
import { preloadPhotographerDetail } from '@/utils/preload'
import { navTo } from '@/utils/nav'

const store = useHomeStore()
const styleStore = useStyleStore()
const { cityOptions } = storeToRefs(store)
const { statusBarHeight } = useStatusBar()
// 首启风格偏好半屏弹层：shouldPrompt() 判据在 onShow 里异步核验后再置 true。
const showStylePicker = ref(false)
// ── 顶部栏（单行）胶囊避让 ────────────────────────────────────────────────
// 顶栏挂在 .hero-content 里面，而 .hero-content 已有 padding: 24rpx 32rpx 0。
// 因此：
//   横向 = 「搜索框右缘到屏幕右边」的目标值 navRightPx 减去容器已给的 32rpx，
//          否则内联 padding-right 会跟容器叠加成双倍右留白，整条看起来往左缩。
//   纵向 = 「胶囊中线」与「顶栏中线」的差值，直接写成 margin-top。
// 兜底（H5 / 首帧未就绪 / 老基础库）：微信胶囊根本不存在 →
//   navRightPx 归零，内联 padding-right 也归零（容器自带的 32rpx 就是视觉右边距）；
//   margin-top 归零。绝不能用一个假装有胶囊的大数字去偷搜索框宽度
//   （视觉方案文档 §6.3 里写死 102px 的那个默认值，在 H5 上就是错的）。
const NOMENU_HEIGHT_RPX = 64
// .hero-content 的内边距（rpx）：把「屏幕坐标」换算到「容器坐标」时减掉它。
const CONTENT_PAD_H_RPX = 32
const CONTENT_PAD_TOP_RPX = 24
// 胶囊与搜索框之间要留的视觉间隙（px）
const CAPSULE_GAP_PX = 8

// 一律用 px 存（胶囊 rect 就是 px），渲染时再跟 rpx 换算出来的容器内边距对减。
const navRightPx = ref(0)
const navHeightPx = ref(0)
const navOffsetTopPx = ref(0)
// 750rpx 对应的实际像素宽度，用于把上面两个 rpx 常量换成 px
const windowWidth = ref(375)

const navBarStyle = computed(() => {
  const toPx = (rpx: number) => (rpx * windowWidth.value) / 750
  // 容器自身已经给了 32rpx 右内边距，内联这一层只补差额
  const extraRight = Math.max(navRightPx.value - toPx(CONTENT_PAD_H_RPX), 0)
  // 兜底环境 navHeightPx=0 → 直接用 64rpx 的自然高度
  const height = navHeightPx.value > 0 ? navHeightPx.value : toPx(NOMENU_HEIGHT_RPX)
  return {
    paddingRight: `${extraRight}px`,
    height: `${height}px`,
    marginTop: `${navOffsetTopPx.value}px`,
  }
})
const currentCity = ref('全部')
const eventRange = ref<'all' | 'week' | 'month'>('all')
const rangeOptions = [
  { key: 'all' as const, label: '全部' },
  { key: 'week' as const, label: '本周' },
  { key: 'month' as const, label: '本月' },
]
const showCityPicker = ref(false)
const cityKeyword = ref('')

const filteredCityOptions = computed(() => {
  const kw = cityKeyword.value.trim()
  const list = (store.cityOptions || []) as string[]
  return kw ? list.filter(c => c.includes(kw)) : list
})

/** '全部' 之外的候选城市（热门/全部两组都从它切） */
const cityChoices = computed(() => {
  const list = (store.cityOptions || []) as string[]
  return list.filter(c => c !== '全部')
})

/** 城市总数：弹层标题里的「73 个」用它，让用户一眼看出库有多大 */
const cityOptionsCount = computed(() => cityChoices.value.length)

/** 热门城市：按在办场次降序取前 8。走接口时 store.cityEventCounts 有值；
 *  接口失败回落 events 推导时它为空对象，此时退化为原顺序前 8 个。 */
const hotCities = computed(() => {
  const counts = store.cityEventCounts || {}
  return cityChoices.value
    .slice()
    .sort((a, b) => (counts[b] || 0) - (counts[a] || 0))
    .slice(0, 8)
})

/** 全部城市：去掉热门那 8 个，避免同一城市在弹层里出现两次 */
const restCities = computed(() => {
  const hot = new Set(hotCities.value)
  return cityChoices.value.filter(c => !hot.has(c))
})

const nearestEvent = computed(() => {
  // 用 endDate 判断「还没结束」：进行中的多日漫展已经在办，但仍应出现在「距开展」票根上
  // （此前按 startDate 过滤，开场当天票根直接消失）。startDate 在过去时倒计时为 0 天。
  const now = Date.now()
  const upcoming = filteredEvents.value
    .filter(e => e.endDate >= now)
    .sort((a, b) => a.startDate - b.startDate)
  return upcoming.length > 0 ? upcoming[0] : null
})

const heroEvent = computed(() => nearestEvent.value || (filteredEvents.value.length ? filteredEvents.value[0] : null))

/** 票根文案开关：最近的这场是否已经开场（还在办）。 */
const nearestStarted = computed(() => !!nearestEvent.value && nearestEvent.value.startDate <= Date.now())

const heroImage = computed(() => {
  if (heroEvent.value?.cover) return heroEvent.value.cover
  if (store.banners.length) return store.banners[0].image
  // 不再兜底到不存在的 /static/img/banner-1.jpg（死链）：
  // 无封面时返回空串，image 组件渲染占位而非加载失败。
  return ''
})

/**
 * 量一次微信胶囊，算出顶栏的右边界、高度与竖向偏移。
 * H5 / 首帧未就绪（rect 全 0）→ 全部归零走兜底：右边界就是页面自身的 32rpx 边距。
 */
function updateNavMetrics() {
  let winWidth = 375
  try {
    winWidth = uni.getSystemInfoSync().windowWidth || 375
  } catch (e) {
    // 拿不到就按 375 算，只影响 rpx 换算精度，不影响兜底安全性
  }
  windowWidth.value = winWidth

  // 兜底：无胶囊环境。padding-right 归零（容器自带的 32rpx 就是视觉右边距），
  // margin-top 归零（顶栏就待在 .hero-content 的 24rpx 上边距里，绝不顶歪）。
  navRightPx.value = 0
  navHeightPx.value = 0
  navOffsetTopPx.value = 0

  try {
    const rect = uni.getMenuButtonBoundingClientRect()
    // width/left 全 0 是首帧未就绪的典型表现，不能当成「胶囊贴在左边」来算，
    // 否则 navRightPx 会变成 (屏宽 - 0) + 8 = 满屏宽，顶栏被挤成 0 宽。
    if (!rect || !(rect.width > 0) || !(rect.left > 0)) return

    // 横向：胶囊左边界到屏幕右边的整段都让出去，再加 8px 视觉间隙。
    navRightPx.value = Math.max(winWidth - rect.left + CAPSULE_GAP_PX, CAPSULE_GAP_PX)
    if (rect.height > 0) navHeightPx.value = rect.height

    // 纵向：让顶栏中线与胶囊中线齐平。
    // hero-stage 紧跟状态栏之下，.hero-content 又自带 24rpx 上边距，
    // 两者都要从「屏幕坐标」里减掉，差值才是在容器内的偏移。
    const toPx = (rpx: number) => (rpx * winWidth) / 750
    const barTop = statusBarHeight.value + toPx(CONTENT_PAD_TOP_RPX)
    const capsuleCenter = rect.top + rect.height / 2
    const delta = capsuleCenter - barTop - navHeightPx.value / 2
    // 夹在 ±12px：个别机型把 rect 报得很离谱时也不会把内容顶出屏幕
    navOffsetTopPx.value = Math.min(Math.max(delta, -12), 12)
  } catch (e) {
    // 取不到就走上面的兜底值，不抛出、不影响首屏渲染
  }
}

// --- 首屏摄影师卡预载（C 档）-----------------------------------------------
// 模块级节流闸门：整个首页生命周期只放行一次；onShow 再次进入不重复预载。
// <script setup> 顶层 let 编译进 setup()，tab 页实例常驻 → 跨多次 onShow 保持。
let preloadedOnce = false
let preloadArmed = false

/** 摄影师列表数据到达后，对首屏前 2 张卡发起预载（仅一次，H5 端为空操作）。 */
function preloadFirstTwoOnce() {
  if (preloadedOnce || !preloadArmed) return
  const firstTwo = store.photographers.slice(0, 2)
  if (!firstTwo.length) return
  preloadedOnce = true
  firstTwo.forEach(p => preloadPhotographerDetail(p.id))
}

onShow(() => {
  updateNavMetrics()
  // 节流：只在第一次 onShow 放行；从详情页返回再次 onShow 时不重复预载。
  if (!preloadedOnce) {
    preloadArmed = true
    preloadFirstTwoOnce() // 数据已加载（暖启动/再次进入）时立即补一次
  }
  // 首启风格偏好：判据是「登录 + 已加载偏好 + 未选择 + 未跳过」，
  // 异步核验（标签全集 + 我的偏好）后才置 visible。游客 loadPrefs 直接返回，
  // 绝不会打鉴权接口、绝不会弹；已选/已跳过的用户进来也不会重复弹。
  void (async () => {
    await Promise.all([styleStore.loadTags(), styleStore.loadPrefs()])
    if (styleStore.shouldPrompt()) showStylePicker.value = true
  })()
})

// 冷启动 onShow 早于 onMounted 的 fetchHomeData：列表异步到达后再补触发一次，
// 避免「首次放行时列表为空、一次性闸门被空数据锁死」。
watch(() => store.photographers, () => preloadFirstTwoOnce())

onReady(() => {
  updateNavMetrics()
})

onMounted(() => {
  updateNavMetrics()
  if (!store.loaded) {
    store.fetchHomeData()
  }
  // 城市清单独立拉取：它比首页数据大（73 城），失败也不影响首屏
  store.fetchCities()
})

async function onRefresh() {
  await store.refresh()
  if (store.error) {
    uni.showToast({ title: '刷新失败，请稍后再试', icon: 'none' })
  }
}

// 首启风格偏好：组件只发事件，请求/持久化留在 store，刷新推荐区留在页面。
async function onStyleSave(tagIds: number[]) {
  try {
    await styleStore.savePrefs(tagIds)
  } catch {
    return // 失败提示已在 store 内统一 toast，保留弹层与当前选择
  }
  uni.showToast({ title: '已为你记录风格偏好', icon: 'none' })
  showStylePicker.value = false
  // fetchHomeData 末尾会按新偏好重新拉摄影师列表覆盖推荐区；
  // 用 refresh()（preserveLoaded）保住已加载内容，不闪骨架屏。
  await store.refresh()
}

function onStyleSkip() {
  styleStore.markSkipped()
  showStylePicker.value = false
}

function goSearch() {
  navTo('/pages/search/search')
}

function goSearchByTag(tag: string) {
  // 这里必须走 tags 通道：后端 keyword 只对 p.name / p.description 做 ILIKE，不碰标签表，
  // 而「日系」「哥特」这类词几乎只作为标签存在 → 走 keyword 会命中 0（用户实测的「没有结果」）。
  // 直达结果页而不是先进入口页：入口页 onLoad 会立刻 redirectTo，用户会看到一次白帧/闪烁。
  navTo(`/pages/search/result?tags=${encodeURIComponent(tag)}`)
}

function goPhotographerList() {
  uni.switchTab({ url: '/pages/photographer/list' })
}

function goEventMore() {
  navTo('/pages/event/list')
}

function goEventDetail(id: string) {
  navTo(`/pages/event/detail?id=${id}`)
}

function goCitySelect() {
  showCityPicker.value = true
}

function closeCityPicker() {
  cityKeyword.value = ''
  showCityPicker.value = false
}

function selectCity(city: string) {
  currentCity.value = city
  closeCityPicker()
}

const filteredEvents = computed(() => {
  const now = Date.now()
  const base = currentCity.value === '全部'
    ? store.upcomingEvents
    : store.events
      // 与后端 / store 同一口径：endDate 判定「还没结束」，进行中的场次照样可见
      .filter(e => e.status === 'upcoming' && e.endDate >= now && e.location === currentCity.value)
      .sort((a, b) => a.startDate - b.startDate)
      .slice(0, 8)
  if (eventRange.value === 'all') return base
  const days = eventRange.value === 'week' ? 7 : 30
  const limit = now + days * 24 * 3600 * 1000
  // 时间窗是「开场时间落在窗口内」；已经开场仍在办的场次（startDate 在过去）保留，
  // 否则「本周」在最忙的那天会突然空掉。
  return base.filter(e => e.startDate <= limit)
})

const posterWall = computed(() => {
  // 只展示当前城市、且没被「近期漫展」实际展示过的漫展，避免同一场出现两次。
  // 按 id 去重而不是按日期：数据只有几个档期时按日期会把整段海报墙清空。
  const now = Date.now()
  const shownIds = new Set(filteredEvents.value.map(e => e.id))
  return store.events
    .filter(e => e.status === 'upcoming' && e.endDate >= now && (currentCity.value === '全部' || e.location === currentCity.value))
    .filter(e => !shownIds.has(e.id))
    .sort((a, b) => a.startDate - b.startDate)
    .slice(0, 4)
})

/** 空态文案：必须分清「这个城市本来就没有漫展」和「有时间窗 / 都已结束」，
 *  否则会把「数据过期」说成「城市没漫展」，用户会以为是城市选错了。
 *  三种情形：① 本周/本月筛空 → 说时间；② 城市一场都没有 → 说城市；
 *  ③ 城市有但都不在「即将开展」的区间（多为已结束）→ 说「近期」。 */
type EmptyAction = 'resetRange' | 'resetCity' | 'none'
const emptyState = computed<{ title: string; sub: string; action: EmptyAction }>(() => {
  const now = Date.now()
  const cityScoped = currentCity.value !== '全部'
  const cityName = currentCity.value
  // 当前城市在库里的全部漫展（不看时间窗，用来判断「这个城市到底有没有漫展」）
  const cityAll = cityScoped ? store.events.filter(e => e.location === cityName) : store.events

  if (eventRange.value !== 'all') {
    return {
      title: cityScoped ? `${cityName}近期暂无即将开展的漫展` : '近期暂无即将开展的漫展',
      sub: '看看全部漫展 ›',
      action: 'resetRange',
    }
  }
  if (!cityAll.length) {
    return {
      title: cityScoped ? `${cityName}暂无漫展` : '近期暂无即将开展的漫展',
      sub: cityScoped ? '看全部城市的漫展 ›' : '',
      action: cityScoped ? 'resetCity' : 'none',
    }
  }
  // 城市有漫展但都被时间窗滤掉了（绝大多数是已经结束）—— 文案落在「近期」，不落在城市
  const hasEnded = cityAll.some(e => e.endDate < now || e.status === 'ended')
  return {
    title: cityScoped && hasEnded ? `${cityName}近期暂无即将开展的漫展` : '近期暂无即将开展的漫展',
    sub: cityScoped ? '看全部城市的漫展 ›' : '',
    action: cityScoped ? 'resetCity' : 'none',
  }
})

function onEmptyAction() {
  if (emptyState.value.action === 'resetRange') eventRange.value = 'all'
  else if (emptyState.value.action === 'resetCity') selectCity('全部')
}

const formatShortDate = (ts: number) => {
  const d = new Date(ts)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const filteredPhotographers = computed(() => {
  if (currentCity.value === '全部') return store.photographers
  return store.photographers.filter(p => p.location === currentCity.value)
})

function onQuickNav(_url: string) {}
</script>

<style lang="scss" scoped>
/*
 * 页面根用「固定定位 + 内部唯一滚动容器」锁死高度。
 *
 * 旧写法 min-height:100vh + .scroll-content{height:100vh} + 内联 padding-top:statusBarHeight
 * 会让文档实际高度溢出视口：content-box 下 100vh 不含 padding，再加 47px 状态栏 padding，
 * 又叠加 uni-tabbar 撑出的 50px，document.documentElement.scrollHeight 变成 941（视口 844），
 * 于是**整个 window 能被滚 97px**。而 .status-bar 是 position:fixed 不跟随位移，
 * 滚到底再回顶时顶部导航被顶出视口、压在状态栏下面 —— 用户报的「搜索框和『全部』不见了」。
 *
 * 改成 position:fixed;inset:0 后页面自成一块视口大小的盒子，window 不再参与滚动，
 * 位移全部收敛到内部 scroll-view，fixed 的状态栏与顶部导航永远贴在原位。
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
  background: $surface-page;
}

/*
 * 状态栏底色必须不透明。
 * 旧写法 background: $surface-glass（rgba(21,23,34,0.86)）+ backdrop-filter，
 * 在不支持 backdrop-filter 的内核里会直接透出下层 hero 的橙色光晕
 * （.hero-glow 的 rgba(255,107,0,0.30)），视觉上就是状态栏位置出现一条橙条。
 * 这里改成不透明底色 + 保留 backdrop-filter（支持的机器上仍有玻璃质感）。
 */
.status-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: $surface-page;
  backdrop-filter: blur(20px);
}

/*
 * 唯一滚动容器：吃掉父容器剩余高度。
 * 必须显式写 min-height:0 —— flex 子项默认 min-height:auto，内容一多就把父容器顶破，
 * 页面又会被撑出视口。box-sizing:border-box 让内联的 padding-top（状态栏高度）算进自身高度，
 * 否则 100vh + padding 又会溢出（见 .page 注释）。
 */
.scroll-content {
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
  box-sizing: border-box;
}

// ===== Hero Stage =====
.hero-stage {
  position: relative;
  height: 820rpx;
  overflow: hidden;
  background: $surface-page;
}

.hero-media {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.hero-scrim {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(12, 13, 20, 0.74) 0%, rgba(12, 13, 20, 0.10) 30%, rgba(12, 13, 20, 0) 44%),
    linear-gradient(0deg, #0C0D14 0%, rgba(12, 13, 20, 0.97) 32%, rgba(12, 13, 20, 0.92) 50%, rgba(12, 13, 20, 0.62) 70%, rgba(12, 13, 20, 0.18) 86%, rgba(12, 13, 20, 0) 96%);
}

.hero-glow {
  position: absolute;
  left: -10%;
  right: -10%;
  top: -6%;
  height: 72%;
  background:
    radial-gradient(52% 40% at 78% 12%, rgba(255, 107, 0, 0.30) 0%, rgba(255, 107, 0, 0) 70%),
    radial-gradient(40% 30% at 6% 86%, rgba(0, 210, 211, 0.12) 0%, rgba(0, 210, 211, 0) 72%);
  opacity: 0.9;
  pointer-events: none;
}

.vf-corner {
  position: absolute;
  width: 36rpx;
  height: 36rpx;
  border: 2rpx solid rgba(255, 255, 255, 0.55);
  pointer-events: none;
}

.vf-tl { top: 24rpx; left: 24rpx; border-right: 0; border-bottom: 0; border-radius: 4rpx 0 0 0; opacity: .45; }
.vf-tr { top: 24rpx; right: 24rpx; border-left: 0; border-bottom: 0; border-radius: 0 4rpx 0 0; opacity: .6; }

.hero-content {
  position: relative;
  z-index: 2;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24rpx 32rpx 0;
}

// ── 顶部栏（单行）────────────────────────────────────────────────────────
// 高度和 paddingRight 都由脚本按胶囊实测算出后内联注入；这里只管横向排布。
.hero-nav-bar {
  position: relative;
  z-index: 10;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  width: 100%;
  padding-left: 0;
}

// 城市：纯文字 + 小三角，不套玻璃胶囊。城市只是「筛选条件」，
// 给它一个药丸外框会让它在视觉权重上跟搜索框平级，反而抢戏。
.nav-city-btn {
  position: relative;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  height: 64rpx;
  margin-right: 16rpx;
  transition: opacity 0.2s ease;

  // 热区扩展：上下各 24rpx（宽松，顶栏本身给足了竖向空间），
  // 右侧只扩 8rpx —— 必须 ≤ 间距的一半（16rpx / 2），
  // 否则会压进搜索框的点击区，点搜索时误触发城市选择器。
  &::after {
    content: '';
    position: absolute;
    top: -24rpx;
    bottom: -24rpx;
    left: -12rpx;
    right: -8rpx;
  }

  &:active {
    opacity: 0.75;
  }
}

.nav-city-name {
  max-width: 132rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 28rpx;
  font-weight: 600;
  color: $text-1;
  letter-spacing: 0.5rpx;
  text-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.45);
}

.nav-city-caret {
  margin-left: 6rpx;
  font-size: 18rpx;
  line-height: 1;
  color: rgba(255, 255, 255, 0.6);
  transform: translateY(1rpx);
}

// 假搜索框：flex:1 吃满城市按钮与胶囊之间的全部宽度。
// 它只是「看起来像输入框的按钮」，点击直接跳搜索页，所以内部没有真实 input。
.nav-search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
  height: 64rpx;
  padding-left: 20rpx;
  padding-right: 8rpx;
  background: rgba(21, 23, 34, 0.68);
  border: 1rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 32rpx;
  backdrop-filter: blur(16px);
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.25);
  transition: all 0.2s ease;

  &:active {
    background: rgba(29, 32, 47, 0.85);
    border-color: rgba(255, 107, 0, 0.4);
  }

  // 竖向热区补足：搜索框自身 64rpx（H5 ≈33px）不足 44px，上下各外扩 12rpx
  // 补到 ≈45px。左右不外扩 —— 左边是城市按钮，扩了会抢它的点击。
  &::after {
    content: '';
    position: absolute;
    top: -12rpx;
    bottom: -12rpx;
    left: 0;
    right: 0;
  }
}

.search-ico {
  width: 28rpx;
  height: 28rpx;
  margin-right: 12rpx;
  opacity: 0.65;
  flex-shrink: 0;
}

.search-ph {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 24rpx;
  color: $text-2;
  letter-spacing: 0.3rpx;
}

// 右侧「搜索」小药丸：给假搜索框一个明确的行动暗示，同时把右端收住，
// 避免文字拉成一条长线显得更空。
.search-action-pill {
  flex-shrink: 0;
  height: 48rpx;
  display: flex;
  align-items: center;
  padding: 0 16rpx;
  background: $brand;
  border-radius: 24rpx;
  font-size: 20rpx;
  font-weight: 600;
  color: #fff;
  letter-spacing: 0.5rpx;
  box-shadow: 0 2rpx 8rpx rgba(255, 107, 0, 0.35);
}

.hero-copy {
  margin-top: auto;
  padding-bottom: 116rpx;
}

.hero-eyebrow {
  display: block;
  font-size: 24rpx;
  letter-spacing: 4rpx;
  color: $neon-cyan;
  font-weight: 600;
  margin-bottom: 16rpx;
}

.hero-title {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 56rpx;
  font-weight: 900;
  line-height: 1.16;
  letter-spacing: -1rpx;
  color: #fff;
  text-shadow: 0 2rpx 2rpx rgba(0, 0, 0, 0.35), 0 8rpx 32rpx rgba(0, 0, 0, 0.65);
  margin-bottom: 16rpx;
}

.hero-meta {
  display: flex;
  align-items: center;
  gap: 12rpx;
  font-size: 24rpx;
  color: #9AA0B4;
}

.hero-meta-strong {
  color: #fff;
  font-weight: 700;
  font-size: 28rpx;
}

.hero-meta-dot {
  opacity: 0.5;
}

.hero-meta-venue {
  max-width: 460rpx;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

// ===== Ticket dock =====
.ticket {
  position: relative;
  z-index: 3;
  margin: -64rpx auto 0;
  width: calc(100% - 48rpx);
  min-height: 132rpx;
  box-sizing: border-box;
  display: flex;
  align-items: stretch;
  border-radius: 24rpx;
  /* 一层淡暖底：把「票」和下面纯色四宫格区分开；主渐变仍走冷色，避免整卡发橙 */
  background:
    linear-gradient(90deg, rgba(255, 107, 0, 0.07) 0%, rgba(255, 107, 0, 0.02) 46%, rgba(255, 107, 0, 0) 72%),
    linear-gradient(90deg, rgba(36, 40, 59, 0.98) 0%, rgba(26, 28, 41, 0.98) 58%, rgba(19, 21, 31, 0.99) 100%);
  border: 1rpx solid rgba(255, 107, 0, 0.18);
  box-shadow: 0 12rpx 28rpx rgba(0, 0, 0, 0.3), 0 3rpx 8rpx rgba(0, 0, 0, 0.16), inset 0 1rpx 0 rgba(255, 255, 255, 0.055);
  color: $text-1;

  &:active {
    transform: scale(0.985) translateY(2rpx);
  }
}

/* 票根下半段间距单独放宽，让卡和四宫格各自成组（上紧下松） */
.ticket + .section-compact {
  margin-top: 44rpx;
}

.ticket-stub {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 18rpx 0 18rpx 34rpx;
}

.ticket-stub-row {
  display: flex;
  align-items: baseline;
}

.ticket-label {
  font-size: 24rpx;
  font-weight: 600;
  color: rgba(248, 250, 252, 0.62);
  margin-right: 8rpx;
}

.ticket-days {
  font-size: 52rpx;
  font-weight: 900;
  line-height: 1;
  color: $brand;
  font-variant-numeric: tabular-nums;
}

.ticket-unit {
  font-size: 24rpx;
  font-weight: 700;
  color: $brand;
  margin-left: 6rpx;
}

.ticket-meta {
  margin-top: 6rpx;
  font-size: 24rpx;
  color: rgba(248, 250, 252, 0.5);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.ticket-rip {
  /* 撕口：3rpx 暖色齿线，上下各内缩 20rpx 不触边；
     用 repeating-gradient 控制 8:8 的 dash/gap——border dashed 的间距由浏览器定，且贴边时读作分栏线。 */
  align-self: center;
  width: 3rpx;
  height: 92rpx;
  margin: 0 10rpx;
  background: repeating-linear-gradient(
    180deg,
    rgba(255, 107, 0, 0.34) 0,
    rgba(255, 107, 0, 0.34) 8rpx,
    rgba(255, 107, 0, 0) 8rpx,
    rgba(255, 107, 0, 0) 16rpx
  );
}

.ticket-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ticket-cta {
  display: flex;
  align-items: center;
  font-size: 26rpx;
  font-weight: 800;
  letter-spacing: 1rpx;
  color: #0C0D14;
  background: $brand;
  padding: 14rpx 24rpx 14rpx 28rpx;
  border-radius: 999rpx;
}

.ticket-arrow {
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1;
  margin-left: 10rpx;
  color: rgba(12, 13, 20, 0.72);
}

// ===== Sections (Shared) =====
.section {
  padding: 0 $spacing-md;
  margin-top: $spacing-lg;
}

.section-compact {
  padding: 0 $spacing-md;
  margin-top: $spacing-md;
}

.section-first {
  margin-top: $spacing-md;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: $spacing-md;
}

.section-title {
  position: relative;
  font-size: $t-2xl;
  font-weight: 700;
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

.section-more {
  font-size: $t-sm;
  color: $accent;
  font-weight: 500;

  &:active { opacity: 0.6; }
}

// ===== Events =====
.range-chips {
  display: flex;
  gap: 12rpx;
  padding: 0 0 $spacing-sm;
}

.range-chip {
  display: flex;
  align-items: center;
  min-height: 88rpx;
  padding: 0 32rpx;
  border-radius: 999rpx;
  border: 1rpx solid $border-subtle;
  background: rgba(255, 255, 255, 0.04);
  font-size: 24rpx;
  color: $text-3;
  line-height: 1.2;
}

.range-chip.active {
  border-color: rgba(255, 107, 0, 0.55);
  background: rgba(255, 107, 0, 0.14);
  color: #FFB27A;
  font-weight: 600;
}

.event-scroll {
  white-space: nowrap;
  padding-bottom: $spacing-xs;
}

.event-list {
  display: inline-flex;
  gap: $spacing-md;

  &::after {
    content: '';
    width: $spacing-md;
    flex-shrink: 0;
  }
}

.event-card {
  position: relative;
  width: 300rpx;
  height: 420rpx;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: 24rpx;
  overflow: hidden;
  transition: transform 0.16s ease, border-color 0.16s ease;

  &:active {
    transform: scale(0.975);
    border-color: rgba(255, 107, 0, 0.5);
  }
}

.event-cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  background-color: $dark-bg-card;
}

.event-card::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(12, 13, 20, 0) 38%, rgba(12, 13, 20, 0.62) 64%, rgba(12, 13, 20, 0.97) 100%);
  pointer-events: none;
}

.event-countdown-badge {
  position: absolute;
  top: 16rpx;
  left: 16rpx;
  z-index: 2;
  padding: 6rpx 16rpx;
  background: rgba(12, 13, 20, 0.62);
  border: 1rpx solid rgba(255, 255, 255, 0.16);
  border-radius: 999rpx;
  font-size: 24rpx;
  letter-spacing: 0.04em;
  color: #FFB27A;
  backdrop-filter: blur(12rpx);
}

.event-info {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  padding: 20rpx;
}

.event-name {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: normal;
  font-size: 28rpx;
  line-height: 1.3;
  font-weight: 650;
  color: #F8FAFC;
}

.event-meta {
  margin-top: 8rpx;
}

.event-location {
  display: block;
  font-size: 24rpx;
  color: rgba(226, 232, 240, 0.72);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-footer {
  margin-top: 6rpx;
}

.event-date {
  font-size: 24rpx;
  color: #FF9A4D;
  font-weight: 500;
  letter-spacing: 0.02em;
}

// ===== Tags =====
.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.tag-item {
  display: flex;
  align-items: center;
  min-height: 88rpx;
  padding: 0 32rpx;
  font-size: 24rpx;
  color: $neon-purple;
  background: $neon-purple-dim;
  border: 2rpx solid rgba($neon-purple, 0.4);
  border-radius: $border-radius-xl;
  transition: all 0.2s;

  &:active {
    background: $neon-purple-dim;
    border-color: $neon-purple;
    box-shadow: 0 0 16rpx $neon-purple-glow;
  }
}

// ===== Works =====
.works-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: $spacing-md;
}

.work-card {
  position: relative;
  height: 420rpx;
  border-radius: 24rpx;
  overflow: hidden;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
}

.work-img {
  width: 100%;
  height: 100%;
}

.work-scrim {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(12, 13, 20, 0) 24%,
    rgba(12, 13, 20, 0.55) 46%,
    rgba(12, 13, 20, 0.94) 72%,
    rgba(12, 13, 20, 1) 100%
  );
}

.work-meta {
  position: absolute;
  left: 20rpx;
  right: 20rpx;
  bottom: 18rpx;
  display: flex;
  flex-direction: column;
}

.work-title {
  font-size: 26rpx;
  font-weight: 600;
  color: #F8FAFC;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.work-author {
  margin-top: 4rpx;
  font-size: 24rpx;
  color: rgba(226, 232, 240, 0.68);
}

// ===== Photographers =====
/* minmax(0, 1fr)：避免超长昵称的 min-content 撑宽轨道、挤窄另一列 */
.photographer-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: stretch;
  gap: $spacing-md;
  padding-bottom: $spacing-sm;
}

// ===== Error =====
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl * 2 $spacing-md;
}

.error-icon {
  font-size: 80rpx;
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
  color: #0C0D14;
  border-radius: $border-radius-xl;
  font-size: $font-size-base;
  font-weight: 600;

  &:active {
    opacity: 0.8;
  }
}

.bottom-space {
  height: 120rpx;
}

// ===== City Popup =====
.city-popup-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 200;
  display: flex;
  align-items: flex-end;
  padding-bottom: #{$tab-bar-height};
  overscroll-behavior: contain;
}

.city-popup {
  width: 100%;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  background: $dark-bg-secondary;
  border-radius: $border-radius-lg $border-radius-lg 0 0;
  padding-bottom: env(safe-area-inset-bottom);
}

.city-popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md $spacing-lg $spacing-sm;
  /* 不再画下边框：它和下面 .city-search 自己的 1rpx 上边框只隔 8rpx，
     两条浅色线在深色底上会合成一条看起来"多出来的"线（用户截图里那条）。 */
}

.city-popup-title {
  font-size: $font-size-lg;
  font-weight: 700;
  color: $dark-text-primary;
}

.city-popup-close {
  font-size: 36rpx;
  color: $dark-text-tertiary;
  padding: $spacing-xs;
}

/* 搜索框独占一行，用上边框 + 内边距当分隔，替代原 header 的 border-bottom */
.city-search-row {
  padding: $spacing-sm $spacing-lg $spacing-xs;
  border-top: 1rpx solid $dark-border;
}

.city-search {
  width: 100%;
  box-sizing: border-box;
  padding: 0 24rpx;
  height: 80rpx;
  line-height: 80rpx;
  background: rgba(255, 255, 255, 0.06);
  border: 1rpx solid $border-subtle;
  border-radius: 40rpx;
  font-size: $font-size-sm;
  color: $text-primary;
}

.city-search-ph {
  color: $text-tertiary;
}

.city-list {
  /* BUG-3: uni-scroll-view 宿主默认 content-box，左右 padding 会把 390 撑成 423；
     改为 border-box 后宽度归位 390，弹层不出屏、无横向滚动。overflow 千万别动
     ——H5 真正吃触摸滚动的就是这个宿主（overflow:hidden auto）。 */
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: $spacing-sm $spacing-lg $spacing-lg;
}

.city-section {
  margin-bottom: $spacing-md;
}

.city-section-title {
  display: block;
  font-size: $font-size-xs;
  color: $text-tertiary;
  padding: $spacing-xs 0 $spacing-sm;
  letter-spacing: 0.5rpx;
}

.city-section-count {
  margin-left: $spacing-xs;
  color: $border-subtle;
}

/* 双列网格：73 个城市用单列要滚 5 屏，双列一屏半就能看完 */
.city-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16rpx;
}

.city-chip {
  height: 76rpx;
  line-height: 76rpx;
  padding: 0 20rpx;
  border-radius: 16rpx;
  background: rgba(255, 255, 255, 0.04);
  border: 1rpx solid rgba(255, 255, 255, 0.08);
  font-size: $font-size-sm;
  color: $dark-text-primary;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: background 0.15s, border-color 0.15s;

  &:active {
    background: $dark-bg-card-hover;
  }

  &.active {
    color: $brand;
    border-color: rgba($brand, 0.5);
    background: rgba($brand, 0.12);
    font-weight: 600;
  }
}

/* 有搜索词时退回单列长列表（一眼扫全，不用在网格里找） */
.city-plain-list {
  padding-top: $spacing-xs;
}

.city-empty {
  padding: $spacing-xl 0;
  text-align: center;
  font-size: $font-size-sm;
  color: $text-tertiary;
}

.city-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  font-size: $font-size-md;
  color: $dark-text-primary;
  transition: background 0.15s;

  &:active {
    background: $dark-bg-card-hover;
  }

  &.active {
    color: $neon-cyan;
    background: rgba($neon-cyan, 0.06);
  }
}

.city-check {
  color: $neon-cyan;
  font-size: $font-size-md;
  font-weight: 700;
}

.empty-inline {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12rpx;
  padding: 56rpx 32rpx;
  background: $surface-1;
  border: 1rpx solid $border-subtle;
  border-radius: $r-lg;
}
.empty-inline-title {
  font-size: 28rpx;
  color: $text-secondary;
}
.empty-inline-sub {
  font-size: 26rpx;
  color: $brand;
}

/* 低端机/旧内核不支持 backdrop-filter 时退回不透明底，避免文字压图不可读 */
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .status-bar,
  .nav-search-box,
  .event-countdown-badge {
    background: rgba(12, 13, 20, 0.94);
  }
}
</style>
