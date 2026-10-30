// 统一的状态栏高度来源。
//
// 为什么不再用 `ref(44)` + `sysInfo.statusBarHeight || 44`：
//   - H5 端 uni.getSystemInfoSync().statusBarHeight 实际为 0（uni-h5 取 safeAreaInsets.top，
//     桌面浏览器恒为 0）。旧代码用 `|| 44` 兜底，首帧先渲染 44px，等 onMounted 读到真实值 0
//     再跳变 —— 页面切换瞬间就是这次跳变。
//   - 真机端 statusBarHeight 是真实值（iPhone ≈ 44~47px），首帧用 0 会在首屏上方留白。
// 因此这里：默认值 0（与 H5 真实值一致，H5 端零跳变），真机端在 onMounted 里同步补一次，
// 且补值发生在首帧绘制之前（uni 的 onMounted 早于首屏渲染），真机最多只跳一次且发生在首帧前。

import { ref } from 'vue'

/** 读取系统状态栏高度（px）。拿不到时返回 0，绝不用魔法数兜底。 */
export function readStatusBarHeight(): number {
  try {
    const info = uni.getSystemInfoSync()
    const h = info.statusBarHeight
    return typeof h === 'number' && h > 0 ? h : 0
  } catch {
    return 0
  }
}

/**
 * 页面级状态栏高度。
 * @returns 响应式 statusBarHeight（px）
 */
export function useStatusBar() {
  const statusBarHeight = ref(0)

  const sync = () => {
    statusBarHeight.value = readStatusBarHeight()
  }

  // 同步补一次：uni 的 setup 阶段早于首帧渲染，真机端不会出现「先 0 后 47」的可见跳变
  sync()

  return { statusBarHeight, sync }
}
