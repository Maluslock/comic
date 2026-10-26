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
  setTimeout(() => {
    // 启动期/重复点击都可能触发；已经在登录页就不再压栈，避免路由栈异常
    const pages = getCurrentPages()
    const top = pages[pages.length - 1] as unknown as { route?: string } | undefined
    if ((top?.route || '').includes('pages/login')) return
    uni.navigateTo({ url: '/pages/login/index' })
  }, 800)
  return false
}
