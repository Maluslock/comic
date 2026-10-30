/**
 * 微信订阅消息（一次性订阅）。
 *
 * 模板 ID 来自 mp.weixin.qq.com → 功能 → 订阅消息。一次性订阅的语义是
 * 「用户点一次允许 = 一条额度」，所以必须挂在真实的点击事件里调，
 * 且只在 mp-weixin 端有意义（H5 端直接返回空对象）。
 */
export const TMPL = {
  /** 收到预约通知 —— 发给摄影师 */
  newBooking: 'Eke1CcEgA7cYDGoZhpStVnOIuokPv3kPupxo0yikD4s',
  /** 预约结果通知 —— 发给 coser */
  result: 'Fp2RKAffs9pbkaHlr8iUUw9w-H1rYrcXkGBUIXSCyyo',
  /** 相册交付状态通知 —— 发给 coser */
  delivered: 'X8aHjhqaEfq2UeU8AIGpoWSIaLH5Cy0Gjdz0LCXS3A8',
} as const

export type SubscribeResult = Record<string, 'accept' | 'reject' | 'ban' | string>

/**
 * 请求订阅。永远 resolve（失败返回 {}），调用方不需要 try/catch，
 * 因为「没授权」不该阻断下单/接单这类主流程。
 */
export function requestSubscribe(tmplIds: string[]): Promise<SubscribeResult> {
  return new Promise((resolve) => {
    // #ifdef MP-WEIXIN
    const ids = tmplIds.filter(Boolean).slice(0, 3)
    if (!ids.length) {
      resolve({})
      return
    }
    uni.requestSubscribeMessage({
      tmplIds: ids,
      success: (res) => resolve(res as unknown as SubscribeResult),
      fail: () => resolve({}),
    })
    return
    // #endif
    // #ifndef MP-WEIXIN
    resolve({})
    // #endif
  })
}

const KEY = (id: string) => `wxsub:${id}`

export function markSubscribed(id: string) {
  try {
    uni.setStorageSync(KEY(id), Date.now())
  } catch (e) {
    // 存储失败不影响功能，忽略
  }
}

export function isSubscribed(id: string): boolean {
  try {
    return !!uni.getStorageSync(KEY(id))
  } catch (e) {
    return false
  }
}
