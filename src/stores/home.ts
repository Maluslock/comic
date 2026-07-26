import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Photographer, Work, ComicEvent, HomeResponse } from '@/types'
import { getHomeData } from '@/api/index'

export const useHomeStore = defineStore('home', () => {
  // --- State ---
  const loading = ref(true)
  const refreshing = ref(false)
  const loaded = ref(false)

  const banners = ref<{ id: number; image: string; title: string }[]>([])
  const events = ref<ComicEvent[]>([])
  const hotTags = ref<string[]>([])
  const photographers = ref<Photographer[]>([])
  const featuredWorks = ref<Work[]>([])

  const error = ref<string | null>(null)

  // --- Computed ---
  /** B3: Sort by startDate ascending before slicing to guarantee 6 earliest events */
  const upcomingEvents = computed(() =>
    events.value
      .filter(e => e.status === 'upcoming')
      .sort((a, b) => a.startDate - b.startDate)
      .slice(0, 6)
  )

  const ongoingEvents = computed(() =>
    events.value.filter(e => e.status === 'ongoing')
  )

  // --- Actions ---
  async function fetchHomeData() {
    // B1: Reset loaded and error at the start of every fetch so error state is reachable
    loading.value = true
    loaded.value = false
    error.value = null

    const BASE_URL = 'http://localhost:8081'

    let data: HomeResponse

    try {
      const res = await uni.request({
        url: `${BASE_URL}/api/v1/home`,
        method: 'GET',
        timeout: 3000,
      })

      // B2: Check HTTP status before casting — uni.request does not throw on 4xx/5xx
      if (res.statusCode === 200) {
        data = res.data as HomeResponse
      } else {
        throw new Error(`Home API returned HTTP ${res.statusCode}`)
      }
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
      banners.value = (data.banners || []).map(b => ({
        id: b.id,
        image: b.imageUrl,
        title: b.title,
      }))

      // Map events
      events.value = (data.upcomingEvents || []).map(e => ({
        id: String(e.id),
        name: e.name,
        location: e.location,
        venue: e.venue,
        startDate: new Date(e.startDate).getTime(),
        endDate: new Date(e.endDate).getTime(),
        cover: e.coverUrl,
        tags: e.tags || [],
        photographerCount: 0,
        status: (e.status as ComicEvent['status']) || 'upcoming',
      }))

      // Map hot tags
      hotTags.value = (data.hotTags || []).map(t => t.name).slice(0, 10)

      // Map photographers
      photographers.value = (data.recommendedPhotographers || []).map(p => ({
        id: String(p.id),
        name: p.name,
        avatar: p.avatar || '',
        role: 'photographer' as const,
        description: '',
        rating: p.rating,
        reviewCount: p.reviewCount,
        orderCount: p.orderCount,
        location: p.location || '',
        tags: p.tags || [],
        works: [],
        services: [],
        reviews: [],
        createdAt: Date.now(),
      }))

      // Map featured works
      featuredWorks.value = (data.featuredWorks || []).map(w => ({
        id: String(w.id),
        photographerId: '',
        title: w.title,
        images: w.images || [],
        description: '',
        tags: [],
        createdAt: Date.now(),
      }))

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
    // actions
    fetchHomeData,
    refresh,
    formatDate,
    countdownDays,
  }
})
