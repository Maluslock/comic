import type { Photographer, Work, Service, Review, ComicEvent } from '@/types'

export const mockPhotographers: Photographer[] = [
  {
    id: '1',
    name: '光影行者',
    avatar: '/static/img/avatar-photographer1.svg',
    role: 'photographer',
    description: '专注ACG摄影8年，擅长捕捉角色神韵，曾为多个知名coser拍摄官方宣传照。',
    tags: ['日系', '古风', '科幻'],
    location: '北京',
    rating: 4.9,
    reviewCount: 234,
    orderCount: 567,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  },
  {
    id: '2',
    name: '樱花落',
    avatar: '/static/img/avatar-photographer2.svg',
    role: 'photographer',
    description: '日系小清新风格，擅长利用自然光营造梦幻氛围，少女心满满~',
    tags: ['日系', '清新', '少女'],
    location: '上海',
    rating: 4.8,
    reviewCount: 186,
    orderCount: 423,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  },
  {
    id: '3',
    name: '暗夜骑士',
    avatar: '/static/img/avatar-photographer3.svg',
    role: 'photographer',
    description: '专攻暗黑、哥特风格，用光影诠释角色的另一面，带你进入不一样的世界。',
    tags: ['暗黑', '哥特', '朋克'],
    location: '广州',
    rating: 4.7,
    reviewCount: 156,
    orderCount: 312,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  },
  {
    id: '4',
    name: '古风公子',
    avatar: '/static/img/avatar-photographer4.svg',
    role: 'photographer',
    description: '古风摄影大师，精通汉服、仙侠题材，还原古典美学极致。',
    tags: ['古风', '汉服', '仙侠'],
    location: '杭州',
    rating: 4.9,
    reviewCount: 298,
    orderCount: 678,
    createdAt: Date.now(),
    works: [],
    services: [],
    reviews: []
  }
]

export const mockServices: Service[] = [
  { id: 's1', name: '基础套餐', price: 399, description: '2小时拍摄，10张精修', duration: 120 },
  { id: 's2', name: '进阶套餐', price: 699, description: '4小时拍摄，20张精修', duration: 240 },
  { id: 's3', name: '精品套餐', price: 1299, description: '全天拍摄，40张精修，含妆造', duration: 480 },
  { id: 's4', name: '漫展跟拍', price: 599, description: '漫展当日跟拍，15张精修', duration: 360 }
]

export const mockReviews: Review[] = [
  {
    id: 'r1',
    userId: 'u1',
    userName: '小狐狸',
    userAvatar: '/static/img/avatar-coser1.svg',
    photographerId: '1',
    rating: 5,
    content: '摄影师非常专业，拍出来的效果超出预期！沟通也很顺畅，下次还会合作~',
    createdAt: Date.now() - 86400000
  },
  {
    id: 'r2',
    userId: 'u2',
    userName: '月华',
    userAvatar: '/static/img/avatar-coser2.svg',
    photographerId: '1',
    rating: 5,
    content: '光影处理太棒了，每张照片都像海报一样！强烈推荐！',
    createdAt: Date.now() - 172800000
  }
]

export const mockWorks: Work[] = [
  {
    id: 'w1',
    photographerId: '1',
    title: '原神 - 雷电将军',
    images: [
      '/static/img/work-1.jpg',
      '/static/img/work-2.jpg'
    ],
    tags: ['原神', '雷电将军', '游戏'],
    createdAt: Date.now()
  },
  {
    id: 'w2',
    photographerId: '1',
    title: '鬼灭之刃 - 祢豆子',
    images: [
      '/static/img/work-3.jpg'
    ],
    tags: ['鬼灭之刃', '祢豆子', '动漫'],
    createdAt: Date.now() - 86400000
  },
  {
    id: 'w3',
    photographerId: '2',
    title: '魔卡少女樱',
    images: [
      '/static/img/work-4.jpg'
    ],
    tags: ['魔卡少女樱', '粉色', '魔法'],
    createdAt: Date.now()
  },
  {
    id: 'w4',
    photographerId: '4',
    title: '古风仙侠',
    images: [
      '/static/img/work-5.jpg'
    ],
    tags: ['古风', '仙侠', '汉服'],
    createdAt: Date.now()
  }
]

export const tags = [
  '日系', '古风', '暗黑', '清新', '科幻', '赛博朋克',
  '哥特', '少女', '汉服', '仙侠', '游戏', '动漫', '影视'
]

export const timeSlots = [
  '09:00', '10:00', '11:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00'
]

export const mockEvents: ComicEvent[] = [
  {
    id: 'e1',
    name: '第39届萤火虫漫展',
    location: '广州',
    venue: '保利世贸博览馆',
    startDate: new Date('2026-07-17').getTime(),
    endDate: new Date('2026-07-20').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC39%E5%B1%8A%E8%90%A4%E7%81%AB%E8%99%AB%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%B9%BF%E5%B7%9E%20%C2%B7%20%E4%BF%9D%E5%88%A9%E4%B8%96%E8%B4%B8%E5%8D%9A%E8%A7%88%E9%A6%86%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['动漫', '游戏', 'cosplay', '同人'],
    photographerCount: 35,
    status: 'ended'
  },
  {
    id: 'e2',
    name: 'CP32 综合同人展',
    location: '杭州',
    venue: '杭州大会展中心',
    startDate: new Date('2026-05-01').getTime(),
    endDate: new Date('2026-05-05').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3ECP32%20%E7%BB%BC%E5%90%88%E5%90%8C%E4%BA%BA%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%9D%AD%E5%B7%9E%20%C2%B7%20%E6%9D%AD%E5%B7%9E%E5%A4%A7%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['同人', '创作', 'cosplay'],
    photographerCount: 42,
    status: 'ended'
  },
  {
    id: 'e3',
    name: 'CCG EXPO 2026',
    location: '上海',
    venue: '上海跨国采购会展中心',
    startDate: new Date('2026-07-04').getTime(),
    endDate: new Date('2026-07-06').getTime(),
    cover: '/static/img/cover-ccg.jpg',
    tags: ['综合', '游戏', '动漫'],
    photographerCount: 30,
    status: 'ended'
  },
  {
    id: 'e4',
    name: '第22届中国国际动漫节',
    location: '杭州',
    venue: '白马湖国际会展中心',
    startDate: new Date('2026-06-17').getTime(),
    endDate: new Date('2026-06-21').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC22%E5%B1%8A%E4%B8%AD%E5%9B%BD%E5%9B%BD%E9%99%85%E5%8A%A8%E6%BC%AB%E8%8A%82%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E6%9D%AD%E5%B7%9E%20%C2%B7%20%E7%99%BD%E9%A9%AC%E6%B9%96%E5%9B%BD%E9%99%85%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['综合', '动画', '漫画', '游戏'],
    photographerCount: 50,
    status: 'ended'
  },
  {
    id: 'e5',
    name: '第28届IJOY国际动漫游戏狂欢节',
    location: '北京',
    venue: '北京国家会议中心',
    startDate: new Date('2026-05-01').getTime(),
    endDate: new Date('2026-05-03').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC28%E5%B1%8AIJOY%E5%9B%BD%E9%99%85%E5%8A%A8%E6%BC%AB%E6%B8%B8%E6%88%8F%E7%8B%82%E6%AC%A2%E8%8A%82%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%8C%97%E4%BA%AC%20%C2%B7%20%E5%8C%97%E4%BA%AC%E5%9B%BD%E5%AE%B6%E4%BC%9A%E8%AE%AE%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['综合', '国潮', 'cosplay'],
    photographerCount: 25,
    status: 'ended'
  },
  {
    id: 'e6',
    name: '第40届萤火虫漫展',
    location: '广州',
    venue: '保利世贸博览馆',
    startDate: new Date('2026-08-14').getTime(),
    endDate: new Date('2026-08-17').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E7%AC%AC40%E5%B1%8A%E8%90%A4%E7%81%AB%E8%99%AB%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E5%B9%BF%E5%B7%9E%20%C2%B7%20%E4%BF%9D%E5%88%A9%E4%B8%96%E8%B4%B8%E5%8D%9A%E8%A7%88%E9%A6%86%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['动漫', '游戏', 'cosplay'],
    photographerCount: 38,
    status: 'upcoming'
  },
  {
    id: 'e7',
    name: 'ChinaJoy 2026',
    location: '上海',
    venue: '上海新国际博览中心',
    startDate: new Date('2026-08-01').getTime(),
    endDate: new Date('2026-08-04').getTime(),
    cover: '/static/img/cover-chinajoy.jpg',
    tags: ['游戏', '数码', 'cosplay', '电竞'],
    photographerCount: 60,
    status: 'upcoming'
  },
  {
    id: 'e8',
    name: '西安第二十六届梦乡动漫展',
    location: '西安',
    venue: '西安国际会展中心',
    startDate: new Date('2026-09-11').getTime(),
    endDate: new Date('2026-09-14').getTime(),
    cover: 'data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22750%22%20height%3D%22360%22%3E%3Cdefs%3E%3ClinearGradient%20id%3D%22g%22%20x1%3D%220%25%22%20y1%3D%220%25%22%20x2%3D%22100%25%22%20y2%3D%22100%25%22%3E%3Cstop%20offset%3D%220%25%22%20stop-color%3D%22%23a855f7%22%2F%3E%3Cstop%20offset%3D%22100%25%22%20stop-color%3D%22%2306b6d4%22%2F%3E%3C%2FlinearGradient%3E%3C%2Fdefs%3E%3Crect%20fill%3D%22%230a0a1a%22%20width%3D%22750%22%20height%3D%22360%22%2F%3E%3Crect%20fill%3D%22url%28%23g%29%22%20x%3D%2240%22%20y%3D%22150%22%20width%3D%228%22%20height%3D%2244%22%20rx%3D%224%22%2F%3E%3Ctext%20fill%3D%22%23e2e8f0%22%20font-size%3D%2232%22%20font-family%3D%22system-ui%2Csans-serif%22%20font-weight%3D%22bold%22%20x%3D%2270%22%20y%3D%22182%22%3E%E8%A5%BF%E5%AE%89%E7%AC%AC%E4%BA%8C%E5%8D%81%E5%85%AD%E5%B1%8A%E6%A2%A6%E4%B9%A1%E5%8A%A8%E6%BC%AB%E5%B1%95%3C%2Ftext%3E%3Ctext%20fill%3D%22%2364748b%22%20font-size%3D%2216%22%20font-family%3D%22system-ui%2Csans-serif%22%20x%3D%2270%22%20y%3D%22218%22%3E%E8%A5%BF%E5%AE%89%20%C2%B7%20%E8%A5%BF%E5%AE%89%E5%9B%BD%E9%99%85%E4%BC%9A%E5%B1%95%E4%B8%AD%E5%BF%83%3C%2Ftext%3E%3C%2Fsvg%3E',
    tags: ['动漫', '同人', '汉服'],
    photographerCount: 20,
    status: 'upcoming'
  }
]
