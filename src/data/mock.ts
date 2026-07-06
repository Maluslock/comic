import type { Photographer, Work, Service, Review, ComicEvent } from '@/types'

export const mockPhotographers: Photographer[] = [
  {
    id: '1',
    name: '光影行者',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer1&backgroundColor=b6e3f4',
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
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer2&backgroundColor=ffd5dc',
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
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer3&backgroundColor=c0aede',
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
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=photographer4&backgroundColor=d1d4f9',
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
    userAvatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=coser1&backgroundColor=ffdfbf',
    photographerId: '1',
    rating: 5,
    content: '摄影师非常专业，拍出来的效果超出预期！沟通也很顺畅，下次还会合作~',
    createdAt: Date.now() - 86400000
  },
  {
    id: 'r2',
    userId: 'u2',
    userName: '月华',
    userAvatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=coser2&backgroundColor=c9e9f6',
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
      'https://picsum.photos/seed/coswork1/600/450',
      'https://picsum.photos/seed/coswork2/600/450'
    ],
    tags: ['原神', '雷电将军', '游戏'],
    createdAt: Date.now()
  },
  {
    id: 'w2',
    photographerId: '1',
    title: '鬼灭之刃 - 祢豆子',
    images: [
      'https://picsum.photos/seed/coswork3/600/450'
    ],
    tags: ['鬼灭之刃', '祢豆子', '动漫'],
    createdAt: Date.now() - 86400000
  },
  {
    id: 'w3',
    photographerId: '2',
    title: '魔卡少女樱',
    images: [
      'https://picsum.photos/seed/coswork4/600/450'
    ],
    tags: ['魔卡少女樱', '粉色', '魔法'],
    createdAt: Date.now()
  },
  {
    id: 'w4',
    photographerId: '4',
    title: '古风仙侠',
    images: [
      'https://picsum.photos/seed/coswork5/600/450'
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
    name: '上海 CP30',
    location: '上海',
    venue: '国家会展中心',
    startDate: Date.now() + 86400000 * 7,
    endDate: Date.now() + 86400000 * 9,
    cover: 'https://picsum.photos/seed/comic1/750/360',
    tags: ['综合', '同人', 'cosplay'],
    photographerCount: 28,
    status: 'upcoming'
  },
  {
    id: 'e2',
    name: '成都 CD28',
    location: '成都',
    venue: '世纪城新国际会展中心',
    startDate: Date.now() + 86400000 * 14,
    endDate: Date.now() + 86400000 * 15,
    cover: 'https://picsum.photos/seed/comic2/750/360',
    tags: ['综合', '游戏', '音乐'],
    photographerCount: 15,
    status: 'upcoming'
  },
  {
    id: 'e3',
    name: '广州萤火虫',
    location: '广州',
    venue: '保利世贸博览馆',
    startDate: Date.now() + 86400000 * 3,
    endDate: Date.now() + 86400000 * 5,
    cover: 'https://picsum.photos/seed/comic3/750/360',
    tags: ['动漫', '游戏', '同人'],
    photographerCount: 22,
    status: 'upcoming'
  },
  {
    id: 'e4',
    name: '北京 IDO42',
    location: '北京',
    venue: '国家会议中心',
    startDate: Date.now() + 86400000 * 21,
    endDate: Date.now() + 86400000 * 22,
    cover: 'https://picsum.photos/seed/comic4/750/360',
    tags: ['综合', '原创', '汉服'],
    photographerCount: 18,
    status: 'upcoming'
  },
  {
    id: 'e5',
    name: '杭州 CJ漫展',
    location: '杭州',
    venue: '白马湖国际会展中心',
    startDate: Date.now() + 86400000 * 10,
    endDate: Date.now() + 86400000 * 11,
    cover: 'https://picsum.photos/seed/comic5/750/360',
    tags: ['动漫', 'cosplay', '游戏'],
    photographerCount: 12,
    status: 'upcoming'
  }
]
