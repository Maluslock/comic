export interface AuthUser {
  id: string
  name: string
  phone: string
  avatar?: string
}

export function getAuthUser(): AuthUser | null {
  const raw = uni.getStorageSync('user')
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function requireAuth(): boolean {
  if (getAuthUser()) return true
  uni.showToast({ title: '请先登录', icon: 'none' })
  setTimeout(() => uni.navigateTo({ url: '/pages/login/index' }), 800)
  return false
}
