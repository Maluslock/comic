import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiGet, apiPut } from '@/api/client'
import { useUserStore } from '@/stores/user'

export interface StyleTag {
  id: number
  name: string
  usageCount?: number
}

interface PrefsResponse {
  tags: { id: number; name: string }[]
  chosen: boolean
}

const SKIP_KEY = 'stylePickerSkipped'

/**
 * 归一化标签列表。
 *
 * 后端 `GET /v1/tags` 是 `c.JSON(data)` 直接返回**裸数组**（`handler/tag_handler.go`，
 * 全仓没有 envelope 中间件；`pages/search/search.vue` 也是按数组取的）。
 * 曾按 `{list:[]}` 解析 → `res.list` 恒 undefined → `tags` 恒空，
 * 而且照常把 `tagsLoaded` 置 true，永久锁死重试：弹层只剩标题、编辑页一个 chip 都没有、
 * 推荐接口一次都不发 —— 静默全废，日志里连异常都没有。
 *
 * 这里同时兼容两种形状，后端将来若改成 `{list}` 也不会再静默全空。
 * 形状完全不认识时返回空数组，由调用方**不置就绪位**，下一轮还能重试。
 */
function normalizeTags(raw: unknown): StyleTag[] {
  const candidate = Array.isArray(raw) ? raw : (raw as { list?: unknown } | null | undefined)?.list
  return Array.isArray(candidate) ? (candidate as StyleTag[]) : []
}

export const useStyleStore = defineStore('style', () => {
  // --- State ---
  const tags = ref<StyleTag[]>([])
  const selected = ref<number[]>([])
  /** loadPrefs 回包里带的 id+name：/v1/tags 挂了也能还原名字，推荐链路不该被它拖死。 */
  const prefsTags = ref<{ id: number; name: string }[]>([])
  const chosen = ref(false)
  const loaded = ref(false)

  // 内部幂等闸门：首页 onShow 与 home store 的 fetchHomeData 会并行触发加载。
  // 布尔闸门挡不住并发（两个调用都看到 false 就各发一次请求），
  // 所以共享 in-flight Promise；失败时清空 Promise 让下次 onShow 能重试。
  const tagsLoaded = ref(false)
  const prefsLoaded = ref(false)
  let tagsPromise: Promise<void> | null = null
  let prefsPromise: Promise<void> | null = null

  /**
   * 「跳过」标记按用户隔离。
   * 曾经用全局键：A 点跳过后 B 首启永远不再弹，违反「每个 coser 自己选一次」的语义。
   */
  function skipKey(): string {
    const uid = useUserStore().user?.id
    return uid ? `${SKIP_KEY}:${uid}` : SKIP_KEY
  }

  // --- Actions ---
  function loadTags(): Promise<void> {
    if (tagsLoaded.value) return Promise.resolve()
    if (tagsPromise) return tagsPromise
    tagsPromise = (async () => {
      try {
        // getApiBase() 已含 /api，path 只写 /v1/...，否则会拼成 /api/api/v1/...
        const raw = await apiGet<unknown>('/v1/tags')
        const list = normalizeTags(raw)
        tags.value = list
        // 只有真的拿到标签才算就绪。空数组一律不置位：
        // 后端换形状、或临时返回空，下一次 onShow 还能再试一次。
        if (list.length > 0) tagsLoaded.value = true
      } catch (e) {
        // 公开接口失败：保持未就绪，下次 onShow 重试，不打扰用户。
        console.warn('[StyleStore] /v1/tags failed:', e instanceof Error ? e.message : String(e))
      } finally {
        tagsPromise = null
      }
    })()
    return tagsPromise
  }

  function loadPrefs(): Promise<void> {
    if (prefsLoaded.value) return Promise.resolve()
    // 游客绝不打需鉴权接口：client 的 401 会清 storage 并 redirect 登录页，
    // 首页 onShow 的探测不能把游客踢去登录。
    if (!useUserStore().isLoggedIn) return Promise.resolve()
    if (prefsPromise) return prefsPromise
    prefsPromise = (async () => {
      try {
        const res = await apiGet<PrefsResponse>('/v1/me/style-prefs')
        const list = res.tags || []
        prefsTags.value = list.map(t => ({ id: t.id, name: t.name }))
        selected.value = list.map(t => t.id)
        chosen.value = !!res.chosen
        loaded.value = true
        prefsLoaded.value = true
      } catch (e) {
        // 后端尚未就绪 / 网络异常：loaded 保持 false，下次 onShow 重试；
        // 不 toast，避免每次切页刷屏，也绝不把 chosen 误置为 true。
        console.warn('[StyleStore] /v1/me/style-prefs failed:', e instanceof Error ? e.message : String(e))
      } finally {
        prefsPromise = null
      }
    })()
    return prefsPromise
  }

  async function savePrefs(tagIds: number[]) {
    const ids = [...tagIds]
    try {
      // 空数组表示清空；成功回包不依赖，直接用本地 ids 更新 state。
      const res = await apiPut<PrefsResponse>('/v1/me/style-prefs', { tagIds: ids })
      // 回包里带 name，顺手刷新本地映射，避免保存后推荐取不到名字。
      const list = res?.tags || []
      if (list.length) prefsTags.value = list.map(t => ({ id: t.id, name: t.name }))
      selected.value = ids
      chosen.value = true
      loaded.value = true
      prefsLoaded.value = true
    } catch (e) {
      uni.showToast({
        title: e instanceof Error ? e.message : '保存失败，请稍后再试',
        icon: 'none',
      })
      throw e
    }
  }

  /** 首启弹层判据：只看「登录 + 已加载 + 未选择 + 未跳过」，
   *  绝不能用 selected.length===0 代替（用户主动清空也算已选择过）。 */
  function shouldPrompt(): boolean {
    return useUserStore().isLoggedIn === true
      && loaded.value === true
      && chosen.value === false
      && uni.getStorageSync(skipKey()) !== true
  }

  /** 跳过后不再自动弹；个人中心入口仍可随时修改。 */
  function markSkipped() {
    uni.setStorageSync(skipKey(), true)
  }

  /**
   * 清空「当前账号」的内存态。登出与 401 都必须调，否则换账号会串号：
   * B 不弹（沿用 A 的 chosen）、编辑页看到 A 的标签、
   * 甚至 B 点保存把 A 的 tagId 写进 B 账号。
   * 标签全集是公开数据，没必要清，留着省一次请求。
   */
  function reset() {
    selected.value = []
    prefsTags.value = []
    chosen.value = false
    loaded.value = false
    prefsLoaded.value = false
    prefsPromise = null
  }

  // --- Computed ---
  /** 已选 id → 标签名（优先 /v1/tags 全集，退化为 my prefs 自带的名字）。 */
  const selectedNames = computed(() => {
    const map = new Map<number, string>()
    for (const t of prefsTags.value) map.set(t.id, t.name)
    for (const t of tags.value) map.set(t.id, t.name)
    return selected.value
      .map(id => map.get(id))
      .filter((n): n is string => !!n)
  })

  return {
    tags,
    selected,
    chosen,
    loaded,
    selectedNames,
    loadTags,
    loadPrefs,
    savePrefs,
    shouldPrompt,
    markSkipped,
    reset,
  }
})
