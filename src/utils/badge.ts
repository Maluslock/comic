import { apiGet } from '@/api/client'

const MESSAGE_TAB_INDEX = 2

export function updateMessageBadge(count: number) {
  try {
    if (count > 0) {
      uni.setTabBarBadge({ index: MESSAGE_TAB_INDEX, text: count > 99 ? '99+' : String(count) })
    } else {
      uni.removeTabBarBadge({ index: MESSAGE_TAB_INDEX })
    }
  } catch (e) {
    console.warn('[badge] update failed', e)
  }
}

export async function refreshMessageBadge() {
  const token = uni.getStorageSync('token') as string
  const stored = uni.getStorageSync('user') as string

  if (!token || !stored) {
    updateMessageBadge(0)
    return
  }

  try {
    const user = JSON.parse(stored) as { id: number }
    const res = await apiGet<{ count: number }>(`/v1/chat/unread/${user.id}`)
    updateMessageBadge(res.count || 0)
  } catch {
    updateMessageBadge(0)
  }
}
