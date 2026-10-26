import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Photographer, Work, ComicEvent, HomeResponse } from '@/types'
import { getHomeData } from '@/api/index'
import { apiGet } from '@/api/client'
import { mapBannerItem, mapEventItem, mapPhotographerItem, mapWorkItem } from '@/utils/mappers'

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
    const sorted = events.value
      .filter(e => e.status === 'upcoming' && e.startDate > Date.now())
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

  /** Derived from event locations — always starts with '全部' */
  const cityOptions = computed(() => {
    const cities = new Set<string>(['全部'])
    events.value.forEach(e => { if (e.location) cities.add(e.location) })
    return Array.from(cities)
  })

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
    // actions
    fetchHomeData,
    refresh,
    formatDate,
    countdownDays,
  }
})
