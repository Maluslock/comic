// 小程序没有同源代理，API 基址必须在运行时决定：
// 1) 构建时 VITE_API_BASE_URL 打的默认值（H5 走 /api 相对路径）
// 2) 运行时用 storage 覆盖 —— 换网络（内网/Tailscale/frp 域名）不必重新打包
const KEY = 'apiBase'
let defaultBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api'
// #ifdef MP-WEIXIN
// 小程序包内没有 /static 代理，默认必须是绝对地址，否则 remote 图会被当成本地文件（500）。
// 换网络时在「个人中心 → 长按版本号」改基址，不必重新打包。
defaultBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || 'http://100.64.0.16:8088/api'
// #endif
const DEFAULT_BASE = defaultBase

/** 规范化用户输入：去空白、去尾部斜杠，缺 /api 时补上 */
export function normalizeBase(raw: string): string {
  let v = (raw || '').trim().replace(/\/+$/, '')
  if (!v) return DEFAULT_BASE
  // 用户可能只填 host:port，小程序里必须带协议才能取图
  if (/^[\w.-]+(:\d+)?$/.test(v)) v = 'http://' + v
  if (!/\/api$/.test(v)) v += '/api'
  return v
}

export function getApiBase(): string {
  try {
    const v = uni.getStorageSync(KEY)
    if (typeof v === 'string' && v) return normalizeBase(v)
  } catch { /* storage 不可用时退回默认 */ }
  return DEFAULT_BASE
}

export function setApiBase(raw: string): string {
  const v = normalizeBase(raw)
  uni.setStorageSync(KEY, v)
  return v
}

export function resetApiBase() {
  uni.removeStorageSync(KEY)
}

export function defaultApiBase(): string {
  return DEFAULT_BASE
}
