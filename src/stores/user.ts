import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = computed(() => !!user.value)
  
  function login(data: User) {
    user.value = data
    uni.setStorageSync('user', JSON.stringify(data))
  }
  
  function logout() {
    user.value = null
    uni.removeStorageSync('user')
  }
  
  function init() {
    const stored = uni.getStorageSync('user')
    if (stored) {
      try {
        user.value = JSON.parse(stored)
      } catch {
        logout()
      }
    }
  }
  
  return {
    user,
    isLoggedIn,
    login,
    logout,
    init
  }
})
