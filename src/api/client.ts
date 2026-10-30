// H5 走 vite 代理用相对路径 /api；小程序没有同源概念：
// 构建时 VITE_API_BASE_URL 给默认绝对地址，运行时可在个人中心长按版本号改（storage: apiBase）
import { getApiBase } from '@/utils/apiBase'

function authHeader(): Record<string, string> {
  const token = uni.getStorageSync('token') as string
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function handleUnauthorized(res: { statusCode: number }) {
  if (res.statusCode === 401) {
    uni.removeStorageSync('token')
    uni.removeStorageSync('user')
    // storage 清了 Pinia 也要清，否则 isLoggedIn 仍为 true（半登录态）
    void import('@/stores/user').then(({ useUserStore }) => {
      try {
        const s = useUserStore()
        s.user = null
        s.token = ''
      } catch { /* pinia 未安装时忽略 */ }
    })
    const pages = getCurrentPages()
    const top = pages[pages.length - 1]
    const cur = top?.route
    if (cur && cur !== 'pages/login/index') {
      // redirect 必须带上当前页自身的 query（如 detail?id=12）并整体编码，
      // 否则 401 → 登录 → 跳回时丢 id，落到「缺少摄影师ID」死页。
      // options 的取法与各页面 onMounted 里 getCurrentPages() 一致。
      const opts = ((top as unknown as { options?: Record<string, string> })?.options || {}) as Record<string, string>
      const qs = Object.keys(opts)
        .filter((k) => opts[k] !== undefined && opts[k] !== '')
        .map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(String(opts[k]))}`)
        .join('&')
      const back = `/${cur}${qs ? '?' + qs : ''}`
      uni.navigateTo({ url: `/pages/login/index?redirect=${encodeURIComponent(back)}` })
    }
  }
}

// 调试期把基址打出来：真机上没有 Network 面板时，vConsole 里这行就是证据
let baseLogged = false

type ReqOpts = {
  url: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: AnyObject
  header?: Record<string, string>
}

// uni.request 在部分端/版本里返回的不是 Promise，直接 await 会拿到 Task 或 [err, res]。
// 统一包成回调式 Promise，行为在所有端一致。
function request(opts: ReqOpts): Promise<UniApp.RequestSuccessCallbackResult> {
  return new Promise((resolve, reject) => {
    uni.request({
      ...opts,
      timeout: 8000,
      success: (res) => resolve(res as UniApp.RequestSuccessCallbackResult),
      fail: (err) => reject(new Error(String((err && (err as AnyObject).errMsg) || 'request failed')))
    })
  })
}

// 小程序基础库没有 URLSearchParams（鸿蒙容器直接 ReferenceError），手写序列化。
// 症状：带 query 的接口一律「加载失败」，且请求根本没发出（服务端零日志）。
function toQuery(params?: Record<string, string>): string {
  if (!params) return ''
  const parts: string[] = []
  Object.keys(params).forEach((k) => {
    const v = params[k]
    if (v === undefined || v === null || v === '') return
    parts.push(encodeURIComponent(k) + '=' + encodeURIComponent(String(v)))
  })
  return parts.length ? '?' + parts.join('&') : ''
}

export async function apiGet<T>(path: string, params?: Record<string, string>): Promise<T> {
  const base = getApiBase()
  if (!baseLogged) { baseLogged = true; console.log('[api] base =', base) }
  const url = base + path + toQuery(params)
  const res = await request({ url, method: 'GET', header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200) throw new ApiError(res.statusCode, serverError(res))
  return res.data as T
}

export async function apiPost<T>(path: string, body: AnyObject): Promise<T> {
  const res = await request({ url: getApiBase() + path, method: 'POST', data: body, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode !== 200 && res.statusCode !== 201) throw new ApiError(res.statusCode, serverError(res))
  return res.data as T
}

export async function apiPut<T>(path: string, body: AnyObject): Promise<T> {
  const res = await request({ url: getApiBase() + path, method: 'PUT', data: body, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200) throw new ApiError(res.statusCode, serverError(res))
  return res.data as T
}

export async function apiDelete<T>(path: string): Promise<T> {
  const res = await request({ url: getApiBase() + path, method: 'DELETE', header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode !== 200 && res.statusCode !== 204) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message) }
}

/** 后端统一用 {"error": "..."} 表达失败原因，能拿到就透传给用户，别吞成「请求失败」。 */
function serverError(res: { data?: unknown }): string {
  const data = res.data as { error?: unknown } | undefined
  const msg = data && typeof data.error === 'string' ? data.error : ''
  return msg || 'request failed'
}
export class NotFoundError extends ApiError {
  constructor() { super(404, 'not found') }
}
