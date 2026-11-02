import { apiGet } from '@/api/client'

const MESSAGE_TAB_INDEX = 2

/**
 * 底栏消息角标。
 *
 * 注意：uni 的 setTabBarBadge / removeTabBarBadge 在**非 tabBar 页**（搜索结果页、订单页等）
 * 会走 fail 回调并返回一个 rejected promise。try/catch 只能接住同步异常，接不住 promise 拒绝，
 * 结果就是在控制台留一条 `Uncaught (in promise)`。所以这里统一挂 .catch 吞掉——
 * 角标是纯装饰，失败不影响任何业务。
 */
export function updateMessageBadge(count: number) {
  try {
    const task =
      count > 0
        ? uni.setTabBarBadge({ index: MESSAGE_TAB_INDEX, text: count > 99 ? '99+' : String(count) })
        : uni.removeTabBarBadge({ index: MESSAGE_TAB_INDEX })
    // uni 在 H5 返回 promise，在部分端返回 undefined，两种情况都要兜住
    if (task && typeof (task as any).catch === 'function') {
      ;(task as any).catch(() => {})
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
