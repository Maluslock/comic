# 米拉漫展 H5 首页 · 首屏英雄区重构方案

> 结论先说：首屏不"惊艳"的第一原因不是配色不够亮，是**首屏一张影像都没有**。一个卖"现场感"的生意，把最贵的 540px 全给了文字。
>
> 记忆点设计：**「取景框里的下一场」**——首屏是一张被取景框框住的海报，底部压一张橙色票根。
> 全站唯一品牌装置 = 取景框（四角 L 线 + 对焦十字）× 票根（打孔缺口 + 虚线撕线）。不做霓虹描边、不做发光大字、不堆玻璃拟态。

---

## 1. 现状诊断（5 条，按杀伤力排序）

**① 首屏 0 张图，前 230px 全是字。**
城市胶囊 → 主标题 → 装饰分隔线 → 副标题 → CTA → 倒计时，一路文字；真正的影像要到 banner（第 4 段）才出现，而 banner 只有 340rpx 高、还套在圆角边框里。用户点开一个漫展 App，第一屏没见过一张现场图。

**② 10 个可点击目标，其中 4 个通向同一处。**
首屏可点数：城市 / 搜索 / 通知 / CTA / 倒计时 / banner / 4 个快捷入口 = **10**。
其中「近期热门漫展」CTA、倒计时 pill、banner、快捷入口「漫展日历」全部指向漫展列表或详情。三个"主 CTA"并列，等于没有主 CTA——用户不选，直接滑走。

**③ 一条一条往下堆，没有任何重叠或越界。**
hero、倒计时、banner、四宫格、横滑卡：全部 `padding: 0 24rpx`、同宽、同 20rpx 圆角、同 `$surface-1` 底、同 1rpx 描边。没有全幅、没有压叠、没有尺寸跳变。眼睛读到的是**列表**，不是**页面**——这就是"信息罗列"的机械成因，换色换字体都治不好。

**④ 品牌位被浪费在说废话。**
「米拉漫展」56rpx 渐变字 + 居中 + 60rpx 装饰线 + 副标题，占掉首屏最贵的 ~90px，但说的是用户已经知道的事（他知道自己打开了米拉）。真正的信息（哪一场、几号、在哪）全在折叠线以下。这是**开屏页版式**误用成了首页。

**⑤ 自己的规范自己没守（可量化，会被测试抓到）。**
- 字号 < 24rpx 共 9 处：badge 18rpx、城市箭头 / cta-marker / 倒计时徽标 20rpx、地点 22rpx；
- 顶栏三个点击区 60–64rpx，低于 88rpx 下限（CTA 也仅 ~76rpx）；
- 主 CTA 白字压 `#FF6B00` = **2.86:1**（WCAG AA 需 4.5:1，深色底上尤其糊）；
- `$text-3 #64748B` 在卡片底 `#151722` 上 = **3.75:1**，页面底 = 4.07:1，辅助正文全部不达标。
- 另：代码里横滑卡实际是 280rpx 宽（不是 240），信息区白底上黑字，电商列表味很重。

---

## 2. 首屏英雄区重构：分层 DOM + 尺寸 + CSS

### 2.1 层级结构（文字版）

```
.hero（relative / height 1080rpx / overflow hidden / 底部与 page 硬接）
├─ L0 .hero__media        <image mode=aspectFill>  一张展现场图，全幅出血
├─ L1 .hero__scrim        绝对定位，双段渐变遮罩
├─ L2 .hero__glow         绝对定位，橙主光 + 青副光（mix-blend-mode: screen）
├─ L3 .hero__grain        绝对定位，颗粒噪点（opacity .05 / overlay）
├─ L4 .hero__vf           四角 L 形角标 + 中心对焦十字（取景框装置）
├─ L5 .hero__content      内容层（relative，padding-top = statusBar + 88rpx）
│   ├─ 顶栏：城市胶囊 + 搜索胶囊(flex:1) + 通知
│   ├─ 标题：本场展名，左对齐，2 行内
│   └─ 元信息：日期块（11.23 + 周六）｜场馆
└─ L6 .ticket             橙色票根，margin-top: -64rpx，z-index: 3
```

关键点：**L6 必须压在 L0 的底边上**。这一处"越界"是整个首屏从"列表"变成"页面"的开关，成本 1 行 margin，收益最大。

### 2.2 每层作用与尺寸

| 层 | 作用 | 尺寸 | 要点 |
|---|---|---|---|
| L0 影像 | 提供"现场感"，承担 LCP | 750×1080rpx，`aspectFill` | 用 `<image>` 不用 `background-image`：可被 LCP 统计、可做 decode 后淡入 |
| L1 遮罩 | 保证底部文字可读 + 与页面无缝 | 全幅 | 底部 22% 直接压到纯 `#0C0D14`，衔接下一屏零接缝 |
| L2 氛围光 | 让画面"有温度"，唯一允许的 glow | 120%×70%，左上溢出 | 只用 8%–34% 透明度，超过就是廉价霓虹 |
| L3 颗粒 | 消除大面积渐变的塑料感 | 64×64 平铺 | opacity .05，overlay；小程序不支持时降级为纯 opacity |
| L4 取景框 | **品牌记忆点** | 角标 36rpx，线宽 2rpx，距边 24rpx | 只在首屏出现一次；下方分节不重复，否则变装饰 |
| L5 内容 | 回答"哪一场、几号、在哪" | 左右 gutter 32rpx | 左对齐，绝不居中 |
| L6 票根 | 把"倒计时 + 主 CTA"合成一个物体 | 高 132rpx，左右 margin 32rpx | 深墨字压橙底（6.79:1），不用白字 |

### 2.3 关键 CSS 参数

**遮罩（L1）**
```scss
background:
  linear-gradient(180deg, rgba(12,13,20,.78) 0%, rgba(12,13,20,0) 22%),
  linear-gradient(0deg, #0C0D14 0%, rgba(12,13,20,.94) 22%,
                  rgba(12,13,20,.45) 52%, rgba(12,13,20,0) 78%);
```

**氛围光（L2）**
```scss
background:
  radial-gradient(52% 40% at 76% 16%, rgba(255,107,0,.34) 0%, rgba(255,107,0,0) 68%),
  radial-gradient(40% 30% at 8% 88%, rgba(0,210,211,.14) 0%, rgba(0,210,211,0) 70%);
mix-blend-mode: screen; opacity: .9;
```
橙做主光（右上，压在标题对侧），青只做 14% 的对位副光。两色永不同权重——这是"高级"和"赛博"的分界线。

**颗粒（L3）**
```scss
background: url('/static/noise.png') repeat;  /* 64×64 单通道 PNG，<1KB */
background-size: 128rpx 128rpx;
opacity: .05; mix-blend-mode: overlay; pointer-events: none;
```
无图兜底：两层 2rpx `radial-gradient` 交错 + `background-size: 3px 3px`，观感略糙但零资源。

**取景框（L4）**
```scss
.vf-corner { position:absolute; width:36rpx; height:36rpx;
  border:2rpx solid rgba(255,255,255,.55); }
.vf-tl { top:24rpx; left:24rpx;  border-right:0; border-bottom:0; border-radius:4rpx 0 0 0; }
/* 其余三角镜像；四角透明度各自 .45/.55/.55/.7，制造光从左上来 */
.vf-reticle { width:64rpx; height:64rpx; left:50%; top:38%;
  border:1rpx solid rgba(0,210,211,.7); }   /* 只在入场时闪一次，常驻时 opacity:0 */
```

**票根（L6）— 已实现版本**
```scss
.ticket {
  margin: -64rpx auto 0; width: calc(100% - 48rpx); height: 132rpx; border-radius: 24rpx;
  background: linear-gradient(90deg, rgba(36,40,59,.98) 0%, rgba(26,28,41,.98) 58%, rgba(19,21,31,.99) 100%);
  border: 1rpx solid rgba(255,107,0,.18);        /* 边缘只留品牌暗示，橙色留给数字和 CTA */
  box-shadow: 0 12rpx 28rpx rgba(0,0,0,.30), 0 3rpx 8rpx rgba(0,0,0,.16),
              inset 0 1rpx 0 rgba(255,255,255,.055);
}
.ticket-rip {                                    /* 撕口：贯穿卡高 */
  align-self: stretch; width: 0; margin: 0 4rpx;
  border-left: 2rpx dashed rgba(248,250,252,.16);
}
```
- **不做左右半圆缺口**：票根上移 64rpx 后，缺口正好横跨 hero 底边（上半压图、下半压页面底色），纯色圆片对不上底图必留接缝；`mask-composite` 在微信 webview（iOS WKWebView / Android X5）不可靠，失效时整张卡片会被裁空。缺口需下移到纯色区域才值得做。
- 撕口不放"打孔小圆点"：脱离缺口后读起来就是两个游离黑点。
- hero 取景框只保留上面两个角标（`.vf-tl` / `.vf-tr`）：下面两个会被票根压住并露出灰色细线。

内容：左「距开展 **1** 天」（数字 52rpx/900/tabular-nums），虚线，右橙色胶囊「查看漫展」+ 箭头。

**标题（L5）**
```scss
font-size: 64rpx; font-weight: 900; line-height: 1.12;
letter-spacing: -1.5rpx;              /* 中文负字距 = 海报感，不加字体依赖 */
text-shadow: 0 4rpx 24rpx rgba(0,0,0,.45);
```
中文没法定制字体（不引依赖），就用**字重 900 + 负字距 + 超大字阶**做海报感，比渐变字更有骨架。

---

## 3. 信息架构调整

| 模块 | 现状 | 调整 | 理由 |
|---|---|---|---|
| 「米拉漫展」主标题 | 首屏 56rpx 居中 | 降为顶栏 24rpx 字标；首屏标题位让给**本场展名** | 品牌位不该说废话 |
| 「近期热门漫展」CTA | 首屏最大色块 | **删除**，功能并入票根 | 与倒计时/banner/日历共 4 重跳同目的地 |
| 倒计时 pill | 独立一条 | **并入票根左侧** | 它霸占了"第二英雄"位 |
| Banner 轮播 | 首屏第 4 段 | **升级为英雄区底图**（复用 `store.banners`） | 首屏终于有影像；零接口改动 |
| 四宫格 | banner 下 | **无卡片图标条**，4→3 项；「我的预约」归「我的」tab | 4 个同权方卡 = 仪表盘感 |
| 近期漫展横滑 | 第 5 段 | **提到第 2 屏首位**，按「本周 / 下周」分段（前端 computed） | 主业务前置 |
| 热门风格胶囊 | 横滑之后 | **后置**到摄影师之后，或并入搜索页 | 弱意图，别挡在路中间 |
| 推荐摄影师 | 末位 | 上移到第 3 屏，与作品区**混排**（人+片） | 差异化是"约拍"，不是"展讯" |

**首屏只留三块**：顶栏（城市 / 搜索 / 通知）+ 英雄海报（图 / 展名 / 日期 / 场馆）+ 票根。
第 2 屏开头露出横滑卡的第一张（约 160px），制造"下面还有"的下滑钩子。

---

## 4. 动效与交互

### 4.1 入场编排（一次性，只编一次，不要逐区块淡入）

| 阶段 | 延迟 | 时长 | 缓动 | 属性 |
|---|---|---|---|---|
| 影像 reveal | 0 | 900ms | `cubic-bezier(.16,1,.3,1)` | `scale 1.06→1` |
| 遮罩 | 80ms | 600ms | `linear` | `opacity 0→1` |
| 取景角标 | 140ms | 420ms | `cubic-bezier(.2,.8,.2,1)` | `scale 1.35→1` + 描边显现 |
| 标题 | 220ms | 520ms | `cubic-bezier(.16,1,.3,1)` | `translateY(24rpx)→0` + opacity |
| 元信息 | 290ms | 420ms | 同上 | 同上（stagger 70ms） |
| 票根 | 360ms | 560ms | `cubic-bezier(.2,.9,.25,1)` | `translateY(48rpx)→0` + 阴影收紧 |
| 对焦闪光 | 620ms | 420ms | `ease-out` | 十字 `scale 1.6→1`、`opacity .85→0` |
| 票根扫光 | 720ms | 800ms | `linear` | 高光带 `translateX(-120%)→120%` |

全程 ≈1.15s，一串镜头语言。**所有 keyframes 放 `global.scss`**：uni-app scoped 样式里的 `@keyframes` 在部分平台会被加 `[data-v]` 前缀而不生效。

### 4.2 滚动视差（只做 2 层，不做逐卡）

- 触发：`scroll-view` 的 `@scroll`，读 `e.detail.scrollTop`，节流 16ms；
- 影像层：`translate3d(0, min(scrollTop*0.35, 220px), 0)`，同时 `scale(1 + min(scrollTop,220)*0.0004)`；
- 票根：`translateY(-scrollTop*0.12)`，最大 -40px（与影像拉开层次）；
- `scrollTop > 560px` 后置 flag 停止写入，避免长列表持续计算；
- 只写 `transform`，不读布局，不用 `document/window`。

### 4.3 点击反馈

| 目标 | 按下 | 释放 | 备注 |
|---|---|---|---|
| 票根 | `scale(.985) translateY(2rpx)` + `filter:brightness(1.05)`，90ms | 220ms 回弹 | 加 `uni.vibrateShort({type:'light'})`，try/catch 静默 |
| 横滑卡 | 卡 `scale(.97)`，图 `scale(1.04)` 反向 | 240ms | 父缩子放 = 实体按压感 |
| 顶栏图标 | `background: rgba(255,255,255,.10)`，120ms | 同 | 保留 88rpx 热区 |

### 4.4 兼容与降级

| 手法 | H5 | 小程序 | 兜底 |
|---|---|---|---|
| `mix-blend-mode` | ✔ | 基础库 2.9+ 部分 | 纯 opacity |
| `backdrop-filter` | ✔ | 部分机型无 | `@supports` 内加半透明底 |
| `clip-path` | ✔ | ✖ | **不使用** |
| `background-attachment: fixed` | 移动端 ✖ | ✖ | 用 transform 视差 |
| `prefers-reduced-motion` | ✔ | — | 关闭编排与视差，仅留 120ms opacity |

**明确不做**：霓虹描边、发光文字、第二套字体、逐卡 reveal、任何需要 `document/window` 的滚动库。

---

## 5. 实施顺序（改动小 → 视觉收益大）

### P0 · 首屏换骨（0.5–1 天，吃掉 70% 的收益）
1. `src/pages/index/index.vue`：删 `hero-title-block` / `hero-cta` / `countdown-bar` / `banner-section` 四段（约 -90 行），插入 hero 六层 + 票根（约 +70 行模板 / +140 行样式）。
2. 同文件结构调整（易漏）：`scroll-content` 的 `paddingTop` 要去掉、`.status-bar` 的 `$surface-glass` 要改成透明渐变——否则英雄图无法出血到状态栏下，顶部会压一条灰带。
3. `src/styles/tokens/_semantic.scss`：新增 `$hero-scrim / $hero-glow-warm / $hero-glow-cool / $ticket-gradient / $ink-on-brand`；把 `$text-3` 由 `#64748B` 改 `#8A94A8`（3.75 → **5.84:1**）。
4. `src/styles/variables.scss`：补 `$gutter: 32rpx`（首屏与下方分节统一 32rpx，替换现在的 24rpx）。
5. 同批必改 `HomeSkeleton.vue`，否则加载瞬间高度从旧骨架跳到 1080rpx，CLS 直接爆。

### P1 · 动效（0.5 天）
`src/styles/global.scss` 加 `@keyframes` + `.enter-*` 工具类；`index.vue` 在 `onMounted` 里按表加 class（一次性），`prefers-reduced-motion` 时直接加终态。

### P2 · 横滑卡改"图卡"（0.5 天）
`index.vue` 的 `.event-card`：280×~400rpx → **300×420rpx**；去掉 `$surface-1` 底和 1rpx 描边，图占满整卡，信息压图上（底部 60% 渐变）；左上角放真实日期数字块（不是装饰编号）。卡片之间 `gap: 20rpx`，第一张左偏 32rpx 出血，制造"横向可滑"的视觉断口。

### P3 · 导航降噪（0.5 天）
`src/components/QuickActionBar.vue`：4 个卡 → 3 项无卡图标条（icon 44rpx、label 24rpx、热区 88rpx），删掉重复定义的 `.action-icon`（现文件里写了两遍，后者覆盖前者）。

### P4 · 视差 + 分节 reveal（0.5 天）
`@scroll` 的 2 层视差；`uni.createIntersectionObserver`（threshold .15、once）给 3 个分节各一次 `translateY(32rpx)→0`，**不对列表卡做 stagger**。

### P5 · 摄影师区（可选，1 天）
`PhotographerCard.vue` 的 `variant="grid"`：去边框、头像 96→112rpx、名字 28rpx、简介 1 行截断；需回归列表页（`variant="row"` 共用）。

---

## 6. 验收标准（3 条可量化）

| # | 指标 | 目标 | 测法 |
|---|---|---|---|
| 1 | **首屏视觉区块数 / 可点击目标数** | 区块 **3** 个（顶栏 / 英雄 / 票根，现 6 个）；可点目标 **≤5** 且目的地互不重复（现 10 个、4 个同目的地） | 375×812 真机截图，逐个数 |
| 2 | **对比度** | 首屏最低文本对比度 **≥4.5:1**（白字橙底 2.86:1 → 墨字橙底 **6.79:1**）；`$text-3` 3.75:1 → **5.84:1**；无 <24rpx 文字 | axe DevTools / 自写对比度脚本跑 token 表 |
| 3 | **加载与稳定** | 4G 模拟 LCP **≤1.8s**，CLS **≤0.05**，首屏图片总重 **≤180KB**（WebP），骨架→内容无高度跳变 | Lighthouse mobile + 录屏核对 |

附加（不占主指标）：首屏渲染图片 **1 张**；顶栏所有热区 **≥88rpx**；断网/慢网下票根倒计时不出现 `NaN`。
