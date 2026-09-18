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
  /** Dedupe by calendar day so 6 cards cover 6 distinct dates — same-day
   *  events (nyato lists many per day) would otherwise fill all 6 slots. */
  const upcomingEvents = computed(() =>
    events.value
      .filter(e => e.status === 'upcoming' && e.startDate > Date.now())
      .sort((a, b) => a.startDate - b.startDate)
      .reduce<ComicEvent[]>((acc, e) => {
        const day = new Date(e.startDate).toDateString()
        if (!acc.some(x => new Date(x.startDate).toDateString() === day)) {
          acc.push(e)
        }
        return acc
      }, [])
      .slice(0, 6)
  )

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
  async function fetchHomeData() {
    // B1: Reset loaded and error at the start of every fetch so error state is reachable
    loading.value = true
    loaded.value = false
    error.value = null

    let data: HomeResponse

    try {
      data = await apiGet<HomeResponse>('/v1/home')
    } catch (e) {
      // Fallback to mock data during development
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
    await fetchHomeData()
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
