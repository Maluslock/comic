import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Photographer, Work, ComicEvent, HomeResponse } from '@/types'
import { getHomeData, getCities, type CityItem } from '@/api/index'
import { apiGet } from '@/api/client'
import { mapBannerItem, mapEventItem, mapPhotographerItem, mapWorkItem } from '@/utils/mappers'
import type { PhotographerDTO } from '@/utils/mappers'
import { useStyleStore } from '@/stores/style'

export const useHomeStore = defineStore('home', () => {
  // --- State ---
  const loading = ref(true)
  const refreshing = ref(false)
  const loaded = ref(false)

  const banners = ref<{ id: number; image: string; title: string; linkId?: number | null }[]>([])
  const events = ref<ComicEvent[]>([])
  const hotTags = ref<string[]>([])
  const photographers = ref<Photographer[]>([])
  const featuredWorks = ref<Work[]>([])

  const error = ref<string | null>(null)

  // --- Computed ---
  /** 按开场时间升序，且每天最多 ceil(8/档期数) 张 —— 顺序与覆盖同时成立。
   *  「先按日去重再把同日补到末尾」会把顺序打乱成 1天/7天/1天（用户实测报告）；
   *  纯按时间排又会被同一天的 12 场占满。配额法两头都兼顾，结果天然单调。 */
  const upcomingEvents = computed(() => {
    // 时间窗口径与后端 GetUpcomingEvents 一致：用 endDate 判断「还没结束」，
    // 而不是 startDate「还没开场」。多日漫展开场之后仍在办，按 startDate 过滤会让
    // 进行中的场次从首页整体消失（首页曾因此在开场当天直接落空态）。
    const now = Date.now()
    const sorted = events.value
      .filter(e => e.status === 'upcoming' && e.endDate >= now)
      .sort((a, b) => a.startDate - b.startDate)
    if (!sorted.length) return []

    const dayOf = (e: ComicEvent) => new Date(e.startDate).toDateString()
    const distinctDays = new Set(sorted.map(dayOf)).size
    const perDay = Math.max(1, Math.ceil(8 / distinctDays))

    const dayCount = new Map<string, number>()
    const picked: ComicEvent[] = []
    for (const e of sorted) {
      const day = dayOf(e)
      const n = dayCount.get(day) ?? 0
      if (n < perDay) {
        dayCount.set(day, n + 1)
        picked.push(e)
      }
      if (picked.length === 8) break
    }

    // 档期极少时配额也补不满（例如只剩 2 天），再从剩余里按时间补，补完重排保持单调
    if (picked.length < 8) {
      const chosen = new Set(picked.map(e => e.id))
      const rest = sorted.filter(e => !chosen.has(e.id))
      picked.push(...rest.slice(0, 8 - picked.length))
      picked.sort((a, b) => a.startDate - b.startDate)
    }
    return picked.slice(0, 8)
  })

  const ongoingEvents = computed(() =>
    events.value.filter(e => e.status === 'ongoing')
  )

  // --- City list ---
  // 城市选择器要的是「全库还有未结束漫展的城市」（73 个），首页 events 只有 50 场、
  // 前端推导只能拿到 20 来个。所以走 /v1/cities 接口；接口挂了才退回 events 推导，
  // 保证断网时弹层至少还能选到眼前这几场所在的城市。
  const cityList = ref<CityItem[]>([])
  const cityListLoaded = ref(false)

  /** 弹层用的城市名数组，'全部' 永远在第一位 */
  const cityOptions = computed(() => {
    if (cityList.value.length) {
      return ['全部', ...cityList.value.map(c => c.name)]
    }
    const cities = new Set<string>(['全部'])
    events.value.forEach(e => { if (e.location) cities.add(e.location) })
    return Array.from(cities)
  })

  /** 城市 → 在办场次数。走接口时用它给弹层显示「N 场」，回落推导时为空对象。 */
  const cityEventCounts = computed<Record<string, number>>(() => {
    const map: Record<string, number> = {}
    cityList.value.forEach(c => { map[c.name] = c.eventCount })
    return map
  })

  async function fetchCities() {
    if (cityListLoaded.value) return
    try {
      const res = await getCities()
      cityList.value = res?.list || []
      cityListLoaded.value = true
    } catch (e) {
      // 城市清单是辅助功能：失败不置全局 error，也不打断首页，只留上下文供排查。
      const msg = e instanceof Error ? e.message : String(e)
      console.warn('[HomeStore] /v1/cities failed, fallback to events:', msg)
    }
  }

  // --- Actions ---
  async function fetchHomeData(preserveLoaded = false) {
    // B1: 首屏加载要能进入错误态，所以置 loaded=false；但下拉刷新必须保住已加载内容，
    // 否则整页被骨架屏替换、失败还会盖掉列表（刷新不等于重开）。
    loading.value = true
    if (!preserveLoaded) loaded.value = false
    error.value = null

    let data: HomeResponse

    try {
      data = await apiGet<HomeResponse>('/v1/home')
    } catch (e) {
      // 只有开发环境才回退 mock：生产环境展示假漫展/假摄影师会比报错更糟
      if (!import.meta.env.DEV) {
        const msg = e instanceof Error ? e.message : String(e)
        error.value = msg || '加载失败'
        loading.value = false
        refreshing.value = false
        return
      }
      console.log('[HomeStore] Backend unavailable, using mock data')
      try {
        data = await getHomeData()
      } catch (mockErr) {
        console.error('[HomeStore] Mock data also failed:', mockErr)
        const msg = mockErr instanceof Error ? mockErr.message : String(mockErr)
        error.value = msg || '加载失败'
        loading.value = false
        refreshing.value = false
        return
      }
    }

    try {
      // Map banners
      banners.value = (data.banners || []).map(mapBannerItem)
      events.value = (data.upcomingEvents || []).map(mapEventItem)
      hotTags.value = (data.hotTags || []).map(t => t.name).slice(0, 10)
      // /v1/home 的推荐先落底：后面按偏好覆盖失败/空结果时直接保住这份，
      // 绝不出现白屏或 0 卡。
      photographers.value = (data.recommendedPhotographers || []).map(mapPhotographerItem)
      featuredWorks.value = (data.featuredWorks || []).map(mapWorkItem)

      loaded.value = true
    } catch (e: unknown) {
      // B5: Narrow unknown error type instead of any
      const msg = e instanceof Error ? e.message : String(e)
      console.error('[HomeStore] Failed to map response:', msg)
      error.value = msg || '加载失败'
    } finally {
      loading.value = false
      refreshing.value = false
    }

    // 有登录用户风格偏好时，推荐区改调按标签过滤的摄影师列表；
    // 任何失败（网络 / 非 200 / 空结果）都保留上面的 /v1/home 推荐位。
    // 不改 /v1/home 后端契约：这里只是前端展示层再拉一次列表覆盖 photographers。
    try {
      const styleStore = useStyleStore()
      // 幂等：已加载则立即返回；游客 loadPrefs 直接 return，不会打鉴权接口。
      await Promise.all([styleStore.loadTags(), styleStore.loadPrefs()])
      const names = styleStore.selectedNames
      if (names.length) {
        // 分页参数名必须是 size：后端 photographer_handler.go 只读
        // strconv.Atoi(c.DefaultQuery("size","10"))，传 limit 会被整条忽略、静默落到默认 10 条。
        // 实测 6 个摄影师时 ?limit=1 返回 6 条、?size=1 返回 1 条。
        const res = await apiGet<{ list: PhotographerDTO[] }>('/v1/photographers', {
          tags: names.join(','),
          size: '6',
        })
        // 兼容 { list: [...] } 与裸数组两种返回体；空结果不覆盖，保住 fallback。
        const raw = res as unknown
        const list: PhotographerDTO[] = Array.isArray(raw)
          ? (raw as PhotographerDTO[])
          : ((res && res.list) || [])
        if (list.length) {
          photographers.value = list.map(mapPhotographerItem)
        }
      }
    } catch (e) {
      console.warn(
        '[HomeStore] style recommendation failed, keep /v1/home:',
        e instanceof Error ? e.message : String(e),
      )
    }
  }

  async function refresh() {
    refreshing.value = true
    await fetchHomeData(true)
  }

  function formatDate(ts: number): string {
    const d = new Date(ts)
    return `${d.getMonth() + 1}/${d.getDate()}`
  }

  function countdownDays(ts: number): number {
    const now = Date.now()
    return Math.max(0, Math.ceil((ts - now) / (1000 * 60 * 60 * 24)))
  }

  /** 卡片角标文案：已开场但没结束的场次显示「进行中」，
   *  否则「0天后」会被读成「今天要开但还没开」，和实际状态不符。 */
  function countdownLabel(startDate: number, endDate?: number): string {
    const now = Date.now()
    if (startDate <= now && (endDate === undefined || endDate >= now)) return '进行中'
    return `${countdownDays(startDate)}天后`
  }

  return {
    // state
    loading,
    refreshing,
    loaded,
    banners,
    events,
    hotTags,
    photographers,
    featuredWorks,
    error,
    // computed
    upcomingEvents,
    ongoingEvents,
    cityOptions,
    cityEventCounts,
    cityList,
    // actions
    fetchHomeData,
    fetchCities,
    refresh,
    formatDate,
    countdownDays,
    countdownLabel,
  }
})
