import { apiGet, apiPost, apiDelete } from '@/api/client'

export async function loadFollowedIds(userId: string): Promise<Set<string>> {
  try {
    const res = await apiGet<{ list: Array<{ eventId: number }> }>(`/v1/follows/${userId}`)
    return new Set((res.list || []).map(f => String(f.eventId)))
  } catch {
    return new Set()
  }
}

export async function toggleFollow(userId: string, eventId: string, followed: boolean): Promise<boolean> {
  if (followed) {
    await apiDelete(`/v1/follows/${userId}/${eventId}`)
    return false
  }
  await apiPost('/v1/follows', { userId, eventId: Number(eventId) })
  return true
}
