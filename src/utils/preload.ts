/**
 * 首页首屏摄影师卡 → 详情页预载。
 *
 * 背景：当前 uni-app（3.0.0-4080720251210001）在小程序端没有封装该能力，
 * mp-weixin 的 uni.api.esm.js 命中 0；H5 侧那几处命中是另一套实现，不复用。
 * 因此微信小程序端必须条件编译手调原生接口；其余端整体编译为空操作。
 *
 * 约束：预载会触发目标页 onLoad 发请求，故只预载首屏前 2 个、同一 id 一次、
 * 集合上限 8；任何失败都静默，绝不影响首页主流程。
 */

// 只在此块内声明微信原生全局：`@dcloudio/types` 未提供全局 wx 类型，
// 这里给出最小结构化声明（declare 不生成运行时代码，真实对象由小程序运行时注入）。
// #ifdef MP-WEIXIN
declare const wx: {
  preloadPage?: (options: { url: string }) => void
} | undefined
// #endif

/** 已预载 id 集合（模块级单例）：跨调用去重，首页一次性闸门之外的第二层保险。 */
const preloaded = new Set<string>()
/** 集合上限：达到上限后清空重来，避免长会话里无序增长。 */
const MAX_PRELOADED = 8

function doPreload(id: number | string): void {
  // #ifdef MP-WEIXIN
  // 能力探测优先于 wx.canIUse('preloadPage')：部分基础库 canIUse 会返回 false，
  // 但接口实际存在可调。故以「全局存在 + 方法确为 function」双条件为准，
  // 不把 canIUse 当作否决条件，避免漏掉本可预载的基础库。
  if (typeof wx !== 'undefined' && typeof wx.preloadPage === 'function') {
    wx.preloadPage({ url: '/pages/photographer/detail?id=' + id })
  }
  // #endif
  // #ifndef MP-WEIXIN
  void id // 非微信端：整体编译为空操作
  // #endif
}

/**
 * 预载摄影师详情页。全函数静默：预载是纯优化，任何异常都不阻断首页。
 * @param id 摄影师 id（number 或 string，内部统一按字符串去重）
 */
export function preloadPhotographerDetail(id: number | string): void {
  try {
    const key = String(id)
    if (preloaded.has(key)) return
    if (preloaded.size >= MAX_PRELOADED) preloaded.clear()
    doPreload(id)
    preloaded.add(key)
  } catch {
    // 静默：预载失败绝不影响主流程
  }
}
