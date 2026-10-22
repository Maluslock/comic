// H5 走 vite 代理用相对路径 /api；小程序没有同源概念，需要在构建时给绝对地址，
// 例如：VITE_API_BASE_URL=http://192.168.124.7:8088/api npm run dev:mp-weixin
const BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api'

function authHeader(): Record<string, string> {
  const token = uni.getStorageSync('token') as string
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function handleUnauthorized(res: { statusCode: number }) {
  if (res.statusCode === 401) {
    uni.removeStorageSync('token')
    uni.removeStorageSync('user')
    const pages = getCurrentPages()
    const cur = pages[pages.length - 1]?.route
    if (cur && cur !== 'pages/login/index') {
      uni.navigateTo({ url: `/pages/login/index?redirect=/${cur}` })
    }
  }
}

export async function apiGet<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = BASE + path + (params ? '?' + new URLSearchParams(params).toString() : '')
  const res = await uni.request({ url, method: 'GET', timeout: 5000, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}

export async function apiPost<T>(path: string, body: AnyObject): Promise<T> {
  const res = await uni.request({ url: BASE + path, method: 'POST', data: body, timeout: 5000, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode !== 200 && res.statusCode !== 201) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}

export async function apiPut<T>(path: string, body: AnyObject): Promise<T> {
  const res = await uni.request({ url: BASE + path, method: 'PUT', data: body, timeout: 5000, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}

export async function apiDelete<T>(path: string): Promise<T> {
  const res = await uni.request({ url: BASE + path, method: 'DELETE', timeout: 5000, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200 && res.statusCode !== 204) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message) }
}
export class NotFoundError extends ApiError {
  constructor() { super(404, 'not found') }
}
