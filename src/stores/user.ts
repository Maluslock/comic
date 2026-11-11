import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import { apiGet, apiPost } from '@/api/client'
import { mapLoginUser, type LoginUserDTO } from '@/utils/mappers'

interface LoginResponse {
  token: string
  user: LoginUserDTO
}

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const token = ref('')
  const isLoggedIn = computed(() => !!user.value && !!token.value)

  async function login(phone: string, code: string): Promise<boolean> {
    try {
      const res = await apiPost<LoginResponse>('/v1/login', { phone, code })
      token.value = res.token
      user.value = mapLoginUser(res.user)
      uni.setStorageSync('token', res.token)
      uni.setStorageSync('user', JSON.stringify(user.value))
      return true
    } catch (e) {
      console.error('[UserStore] login failed:', e)
      return false
    }
  }

  // 微信小程序登录：code 由 uni.login 获取，一次性有效。
  async function loginWeChat(code: string, profile?: { nickName?: string; avatar?: string }): Promise<boolean> {
    try {
      const res = await apiPost<LoginResponse>('/v1/auth/wechat/login', {
        code,
        nickName: profile?.nickName || '',
        avatar: profile?.avatar || '',
      })
      token.value = res.token
      user.value = mapLoginUser(res.user)
      uni.setStorageSync('token', res.token)
      uni.setStorageSync('user', JSON.stringify(user.value))
      return true
    } catch (e) {
      console.error('[UserStore] wechat login failed:', e)
      return false
    }
  }

  // refreshMe 从 /v1/me 补齐本地 user 里可能缺失的服务端字段（photographerId /
  // photographerName / bio / phone）。必要性：photographerId 只在登录响应里下发，
  // 且登录时写进 storage；升级前就登录、之后没重新登录的用户，storage 里的 user
  // 没有这个字段 —— 于是「编辑资料页对摄影师显示艺名提示」永远不显示，
  // 「已有档案」的判断也会漏。失败静默（离线/401 都不该阻塞页面）。
  async function refreshMe(): Promise<void> {
    if (!token.value) return
    try {
      const me = await apiGet<{ id: number; name: string; phone?: string; avatar?: string; bio?: string; photographerId?: number; photographerName?: string }>('/v1/me')
      if (!user.value) return
      // 只补/更新服务端权威字段，不覆盖前端在本地算出来的 tags / createdAt 等。
      user.value = {
        ...user.value,
        name: me.name || user.value.name,
        avatar: me.avatar || user.value.avatar,
        phone: me.phone ?? user.value.phone,
        bio: me.bio ?? user.value.bio,
        photographerId: me.photographerId,
      }
      uni.setStorageSync('user', JSON.stringify(user.value))
    } catch {
      // best-effort：拿不到就沿用本地缓存，提示晚一步出现而已。
    }
  }

  function logout() {
    user.value = null
    token.value = ''
    uni.removeStorageSync('token')
    uni.removeStorageSync('user')
    // 风格偏好按账号隔离，登出必须清内存态：否则 A 登出 B 登录后沿用 A 的
    // selected/chosen（B 不弹首启、编辑页看到 A 的标签、保存把 A 的 tagId 写进 B）。
    try {
      // 动态 import 避免 user ↔ style 形成循环依赖（style 顶层就 import user）。
      void import('@/stores/style').then(({ useStyleStore }) => useStyleStore().reset())
    } catch { /* pinia 未就绪时忽略 */ }
  }

  function init() {
    const storedToken = uni.getStorageSync('token') as string
    const storedUser = uni.getStorageSync('user')
    if (storedToken && storedUser) {
      try {
        token.value = storedToken
        user.value = JSON.parse(storedUser)
      } catch {
        logout()
      }
    }
  }

  return { user, token, isLoggedIn, login, loginWeChat, logout, init, refreshMe }
})
