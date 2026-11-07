/**
 * 统一跳转出口：navTo / navBack
 *
 * 【为什么所有 navigateTo 都必须走这里】
 * 小程序真机（iOS / Android 的 webview）每次 navigateTo 打开新页面，webview 的首帧
 * 默认是一整屏白底，而转场动画（默认 pop-in，约 200~300ms）期间新页面往往还没绘制完成，
 * 于是用户在深色主题下会看到「切页白屏/闪白」。H5 端没有这个问题（同一路径逐帧采样
 * 262 帧白帧为 0），所以只能在小程序侧收口：所有页面跳转统一 animationType: 'none'，
 * 新页面绘制完成前不做位移/淡入，白帧就不会露出来。
 *
 * 顺带兜住两件事：
 * 1. 页面栈满（小程序上限 10 层）时 navigateTo 直接失败，用户点了像没反应 —— 自动回落
 *    redirectTo 顶掉当前页，保证「点了能走」；
 * 2. 以后新增页面忘了写 animationType，默认值就是 'none'，不会再把白屏放回来。
 *
 * 【不接管】switchTab / reLaunch / navigateBack：
 * - switchTab 是 tab 切换，本身没有转场动画语义，也没有 animationType 字段；
 * - reLaunch 会重建整个页面栈，同样不适用；
 * - navigateBack 的动画类型（slide-out-*）与前进方向无关，不在这里收口。
 */

type NavAnimation = 'none' | 'pop-in' | 'fade-in'

export interface NavOptions {
  /** 转场动画，默认 'none'（见文件头：默认值就是防白屏的那道闸） */
  animationType?: NavAnimation
  /** true 时走 uni.redirectTo（不新增页面栈，用于替换当前页） */
  redirect?: boolean
  /** 透传给 uni.navigateTo 的页面间通信通道 */
  events?: any
  /** 调用方自定义成功回调（原样透传） */
  success?: (res: any) => void
  /** 调用方自定义失败回调（原样透传，但栈满时先走 redirectTo 兜底） */
  fail?: (err: any) => void
  /** 调用方自定义结束回调（成功/失败都执行，只会被调用一次） */
  complete?: (res: any) => void
}

/**
 * 判断是否为「页面栈已满」导致的 navigateTo 失败。
 *
 * 微信不同基础库版本的 errMsg 不一样，必须同时兼容：
 * - 旧版：`navigateTo:fail page limit exceeded`
 * - 新版：`navigateTo:fail webview count limit exceed`（2025 年起常见）
 * - 部分端：`exceed max page`
 * 只匹配其中一种，会在另一种版本上让兜底彻底失效 —— 用户点卡片「没反应」。
 */
function isPageLimitError(err: any): boolean {
  const msg = err && typeof err === 'object' ? String(err.errMsg || err.message || '') : String(err || '')
  const lower = msg.toLowerCase()
  return (
    lower.indexOf('page limit') !== -1 ||
    lower.indexOf('webview count limit') !== -1 ||
    lower.indexOf('exceed max page') !== -1
  )
}

/**
 * 统一跳转。默认 animationType: 'none'，默认 navigateTo。
 * 返回 Promise：成功 resolve(true)，失败（含兜底后仍失败）resolve(false)。
 * 刻意不 reject：绝大多数调用点都是「点一下就走」的同步写法，没人 await，
 * reject 只会变成一堆 unhandled rejection 噪音，失败信息交给调用方传进来的 fail。
 */
export function navTo(url: string, options: NavOptions = {}): Promise<boolean> {
  const { animationType = 'none', redirect = false, events, success, fail, complete } = options

  return new Promise<boolean>((resolve) => {
    const finish = (ok: boolean, payload: any) => {
      if (complete) complete(payload)
      resolve(ok)
    }

    // redirectTo 没有 animationType，也不存在栈满问题
    if (redirect) {
      uni.redirectTo({
        url,
        success: (res: any) => {
          if (success) success(res)
          finish(true, res)
        },
        fail: (err: any) => {
          if (fail) fail(err)
          finish(false, err)
        }
      })
      return
    }

    const params: UniNamespace.NavigateToOptions = { url, animationType }
    if (events) params.events = events
    params.success = (res: any) => {
      if (success) success(res)
      finish(true, res)
    }
    params.fail = (err: any) => {
      // 栈满兜底：redirectTo 顶掉当前页，用户至少能进到目标页
      if (isPageLimitError(err)) {
        uni.redirectTo({
          url,
          success: (res: any) => finish(true, res),
          fail: (err2: any) => {
            if (fail) fail(err2)
            finish(false, err2)
          }
        })
        return
      }
      if (fail) fail(err)
      finish(false, err)
    }

    uni.navigateTo(params)
  })
}

/** 返回上一页（默认 1 层）。navigationBack 没有防白屏诉求，保留系统默认动画。 */
export function navBack(delta = 1): void {
  uni.navigateBack({ delta })
}
