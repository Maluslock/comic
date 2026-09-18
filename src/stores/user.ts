import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import { apiPost } from '@/api/client'
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

  function logout() {
    user.value = null
    token.value = ''
    uni.removeStorageSync('token')
    uni.removeStorageSync('user')
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

  return { user, token, isLoggedIn, login, logout, init }
})
