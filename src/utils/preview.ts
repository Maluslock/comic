import { resolveImage } from './mappers'

/**
 * 统一的图片大图预览入口（作品图 / 精选作品 / 其它需要点图看大图的场景）。
 *
 * 为什么要单独收口：
 * 1) urls 必须全部经过 resolveImage() —— 后端把图片托管在 /static/remote/**、
 *    /static/uploads/**，H5 靠同源代理，小程序端必须拿到 https 绝对地址，
 *    否则会被当成本地包内文件读取失败（500）。包内静态图（/static/img 等）原样保留。
 * 2) current 传「字符串 url」而不是「数字下标」：
 *    - H5（uni-h5）previewImage 内部用 `urls.indexOf(current)` 反查下标，
 *      current 传 url 即可精确定位到用户点的那一张；
 *    - 小程序端 wx.previewImage 的 current 既支持 url 也支持下标，
 *      传 url 在两端语义一致，不用各自处理「下标 0」这类边界。
 * 3) 调用方一律用 @click.stop 绑定，避免点击图片时冒泡触发外层卡片跳转。
 */
export function previewImages(images: string[] | undefined | null, index = 0): void {
  const list = (images || []).map((url) => resolveImage(url)).filter((url) => !!url)
  if (!list.length) return
  const safeIndex = Math.min(Math.max(index, 0), list.length - 1)
  uni.previewImage({
    current: list[safeIndex],
    urls: list,
    fail: () => {
      uni.showToast({ title: '图片预览失败', icon: 'none' })
    }
  })
}
