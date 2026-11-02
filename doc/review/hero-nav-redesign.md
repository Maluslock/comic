# 首页 hero 顶部栏视觉改版方案（3 个候选 + 1 个推荐，含精确规格）

- **版本**：v1.0 (2026-11-02)
- **依赖输入**：`comic/doc/review/hero-nav-platform-research.md`（t1 调研成果）
- **设计目标**：彻底解决当前「城市胶囊独占第一行、搜索+铃铛被推到第二行贴右、第一行留白约 270px 形成巨大空洞」的用户痛点（用户反馈「丑死了」），重构出既符合主流平台心智、又能完美避让微信小程序胶囊、且与米拉漫展暗色二次元摄影基调深度契合的高品质单行顶栏。
- **约束原则**：**只做视觉方案设计与精确规格制定，不改动业务代码**。

---

## 一、核心取舍与决策（Executive Summary）

### 1. 顶部铃铛必须下沉至底部 tabBar，不再与搜索同行
- **事实依据**：
  1. 底部 tabBar 第 3 项已经是「消息」（`pages/message/index`，index=2）。
  2. `src/utils/badge.ts` 中 `refreshMessageBadge()` 已经实现 `uni.setTabBarBadge({ index: 2, text: ... })`，与首页铃铛调用完全相同的接口 `/v1/chat/unread/:userId`。
  3. t1 调研证明：业界 9 家平台中 8 家不在顶栏放消息入口（5 家底栏、得物进「我的」、美团不放、淘宝悬浮列）。唯一在顶栏放消息的大麦 H5 是因为其完全没有底部 tabBar。
- **决策**：**坚决砍掉首页顶部铃铛**。将未读提醒收敛到标准底部 tabBar 角标。这不仅去除了重复入口的认知负担，更为顶栏核心搜索区净省出至少 **44px~50px** 的宝贵物理宽度，让假搜索框真正具备舒适的输入占位体验。

### 2. 结构重构方向：从「两行畸形错位」收敛至「单行全屏贯通」
- **现状病根**：现状顶栏高度占了约 80px（胶囊底让位 40px + 第二行 32px + 间距 8rpx），吃掉首屏 820rpx hero 舞台近 1/5 垂直高度，甚至导致 `.hero-copy` 在小屏下被裁切。第一行右侧约 270px 的连续空白（药丸宽度的 3.8 倍）更是视觉灾难。
- **解法共识**：将顶栏全面压缩至**单行结构**（固定高度 64rpx / 32px 或 72rpx / 36px，与微信胶囊垂直居中对其），纵向占用从 80px 降至约 44~48px，释放至少 32px 的宝贵视觉纵深给 Hero 主海报与票根。

---

## 二、微信胶囊避让几何与坐标账本（375 / 390 / 414 三档）

### 2.1 胶囊几何基准与 API 取值
微信小程序的右上角胶囊（MenuButton）是 native 控件，位置和大小由客户端系统决定。避让时必须依赖 `uni.getMenuButtonBoundingClientRect()`：
- **胶囊右边距**：`marginRight = windowWidth - rect.right`（各机型约为 7px ~ 10px）。
- **胶囊宽度**：`rect.width`（标准约为 87px，部分机型 80px 或 88px）。
- **胶囊左侧安全边界**：`capsuleLeft = rect.left`。
- **胶囊上下居中**：顶栏内容容器的高度与垂直位置必须与胶囊 `rect.top` 和 `rect.height` 严格对齐：
  - 顶栏容器顶部 `padding-top = rect.top`（若外层已避让 statusBar，则为 `rect.top - statusBarHeight`）。
  - 顶栏交互行高度 `navHeight = rect.height`（标准为 32px）。

### 2.2 胶囊避让策略与余量定义
- **避让属性**：
  采用容器级 `padding-right: insetRight` 策略，或直接让搜索框右外边距对齐胶囊安全区：
  ```ts
  // 核心避让计算公式
  const rect = uni.getMenuButtonBoundingClientRect()
  // 胶囊到屏幕右侧的真实占用宽度 + 与胶囊的安全呼吸间隙（8px）
  const capsuleSafetyGap = 8 // px，避免贴得太死导致视觉拥挤或触摸冲突
  const rightAvoidWidth = (windowWidth - rect.left) + capsuleSafetyGap
  ```
  在 375px 下，`windowWidth - rect.left` 约为 `375 - 281 = 94px`，加上安全间隙 8px，右侧总避让宽度为 **102px**。
- **页面左侧安全留白**：固定为 `32rpx`（16px @375），与米拉漫展全站 `$s-lg` 栅格保持一致。
- **胶囊行总可用净宽度（Content Safe Width）**：
  $$W_{\text{available}} = \text{WindowWidth} - 16\text{px (左留白)} - \text{RightAvoidWidth}$$

### 2.3 三档屏幕尺寸账本对照表

| 屏幕机型档位 | 屏宽 (px) | 状态栏 (px) | 胶囊 Rect (宽×高 / left, top) | 右避让宽度 (含8px间距) | 顶栏可用净宽 ($W_{\text{avail}}$) |
|---|---|---|---|---|---|
| **iPhone SE / 8 (375)** | 375px | 20px / 44px | 87 × 32 / left: 281, top: 24/48 | **102px** | **257px** |
| **iPhone 13/14/15/16 (390)** | 390px | 47px | 87 × 32 / left: 295, top: 51 | **103px** | **271px** |
| **iPhone Plus/ProMax (414/430)** | 414px | 48px | 87 × 32 / left: 319, top: 52 | **103px** | **295px** |

---

## 三、三个候选方案详述

---

### 方案 A：极致通透 · 纯文字城市 + 胶囊假搜索框 + 消息进底栏（推荐方案）
> **灵感样板**：猫眼 App + 大麦 H5。极度克制、聚焦核心功能。

#### 1. 结构与 DOM 层级
顶栏仅占单行，包含两大模块：左侧「纯文字城市选择器」，右侧自适应「假搜索框（`flex:1`）」，无右侧铃铛。
```html
<view class="nav-row" :style="{ paddingTop: navTop + 'px', height: navHeight + 'px', paddingRight: navRight + 'px' }">
  <!-- 模块 1：纯文字城市（无外壳边框，极度轻量） -->
  <view class="nav-city-text" @click="goCitySelect">
    <text class="city-name">{{ currentCity }}</text>
    <text class="city-arrow">▾</text>
  </view>
  <!-- 模块 2：占位搜索框（flex: 1 吸收全部剩余宽度直至胶囊左安全线） -->
  <view class="nav-search-bar" @click="goSearch">
    <image class="search-icon" src="/static/icons/search.svg" mode="aspectFit" />
    <text class="search-placeholder">搜漫展、嘉宾、摄影师、场馆</text>
    <view class="search-btn-mini">搜索</view>
  </view>
</view>
```

#### 2. 精确尺寸规格（rpx 与三档 px）

```
[16px] ┌───────┐ [8px] ┌───────────────────────────────────────────────┐ [8px] ╔══════════════╗ [7px]
左间距 │ 全部 ▾ │ 间距  │ 🔍 搜漫展、嘉宾、摄影师...             [搜索] │  安全  ║ 微信小程序胶囊 ║ 屏边
       └───────┘       └───────────────────────────────────────────────┘  间距  ╚══════════════╝
        ~48px                                flex: 1                             87 × 32 px
```

- **容器高度**：
  - 高度：`64rpx`（32px），与微信胶囊高度 1:1 严格对齐。
  - 外层内边距：左 `32rpx`（16px），右由 JS 动态计算 `navRight`（约 102px）。
- **城市选择器**：
  - 纯文字 + 图标，无背景容器、无边框。
  - 文字字号：`26rpx`（13px），`font-weight: 600`，字色 `#F8FAFC`（`$text-1`）。
  - 下拉箭头：`18rpx`，透明度 0.6，左间距 `6rpx`。
  - 整体物理宽度：两字城市约 `48px`（96rpx），三字城市约 `60px`，四字城市约 `72px`。
  - 元素右间距（与搜索框）：`16rpx`（8px）。
  - **触控热区保证**：通过 `position: relative` 与 `::after` 上下各外扩 `12rpx`、左右外扩 `10rpx`，确保触控面积达到 `44×44px`，符合微信适老化指南 3.3。
- **假搜索框**：
  - 布局：`flex: 1`，高度 `64rpx`（32px），圆角 `32rpx`（16px 胶囊形）。
  - 内部左右 padding：左 `20rpx`，右 `8rpx`。
  - 放大镜图标：`28×28rpx`（14×14px），透明度 0.65，与文字间隙 `10rpx`。
  - 占位文案：`24rpx`（12px），字色 `#94A3B8`（`$c-slate-400`），文本溢出省略。
  - 右侧小微胶囊（可选亮点）：橙色背景 `$brand`（`#FF6B00`），字号 `20rpx`，高度 `48rpx`，内边距 `0 16rpx`，圆角 `24rpx`，强化这是漫展搜索入口。
- **三档宽度实算**：
  - **375px 档**：净可用 257px。城市占 48px + 间距 8px = 56px。**搜索框可得分得 201px（占屏宽 53.6%）**。
  - **390px 档**：净可用 271px。城市占 48px + 间距 8px = 56px。**搜索框可得分得 215px（占屏宽 55.1%）**。
  - **414px 档**：净可用 295px。城市占 48px + 间距 8px = 56px。**搜索框可得分得 239px（占屏宽 57.7%）**。

#### 3. 胶囊避让策略
- 动态获取 `rect = uni.getMenuButtonBoundingClientRect()`。
- `navRight = (windowWidth - rect.left) + 8`。
- 搜索框自适应 `flex: 1`，其右边缘自然止步于胶囊左侧安全线，无缝贴合，视觉上胶囊仿佛与搜索框属于同一条设计网格。

#### 4. 与 Hero 大图关系及材质质感
- **背景材质**：
  - 搜索框采用米拉暗房磨砂质感：`background: rgba(21, 23, 34, 0.65)`。
  - 边框：`border: 1rpx solid rgba(255, 255, 255, 0.14)`。
  - 高斯模糊：`backdrop-filter: blur(16px)`。
  - 投影：`box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.25)`。
- **随滚动变化（Header Scroll Transition）**：
  - **顶部停靠态（scrollTop = 0）**：顶栏通透浮动在 Hero 海报图之上，海报暗部渐变 scrim 提供足够对比度（白字在深色遮罩上对比度 > 7:1）。
  - **滚动上升态（scrollTop 0 -> 100px）**：随着用户向上滑动，顶栏背景由全透明线性插值为沉浸纯暗色 `$surface-page`（`#0C0D14`），并生成下边框 `border-bottom: 1rpx solid rgba(255, 255, 255, 0.08)`，常驻吸顶。

#### 5. 方案缺点
- 城市选择器由于去除了胶囊边框外壳，视觉层级相对内敛，对新用户的「可点击性（affordance）」提示主要依赖下拉小箭头 `▾`。

---

### 方案 B：一体集约 · 框内分段式搜索框（城市并入搜索框）
> **灵感样板**：大麦 App 官方首屏 + 淘宝 App。极致规整、单框吞并。

#### 1. 结构与 DOM 层级
整行只有**一个连续完整的超长胶囊容器**，城市作为搜索框内部的左侧分段，与搜索区域用细竖线隔离。
```html
<view class="nav-row" :style="{ paddingTop: navTop + 'px', height: navHeight + 'px', paddingRight: navRight + 'px' }">
  <!-- 单一超长药丸，flex: 1 吸收全部宽度 -->
  <view class="unified-search-pill">
    <!-- 框内左分段：城市选择 -->
    <view class="pill-city-segment" @click.stop="goCitySelect">
      <text class="segment-city-name">{{ currentCity }}</text>
      <text class="segment-city-arrow">▾</text>
    </view>
    <!-- 分隔竖线 -->
    <view class="pill-divider" />
    <!-- 框内右分段：搜索交互区 -->
    <view class="pill-search-segment" @click="goSearch">
      <image class="search-icon" src="/static/icons/search.svg" mode="aspectFit" />
      <text class="search-placeholder">搜漫展 / 摄影师 / 场馆</text>
    </view>
    <!-- 搜索图标或微型按钮 -->
    <view class="pill-action-icon" @click="goSearch">
      <text class="search-action-text">搜索</text>
    </view>
  </view>
</view>
```

#### 2. 精确尺寸规格（rpx 与三档 px）

```
[16px] ┌──────────────────────────────────────────────────────────────┐ [8px] ╔══════════════╗ [7px]
左间距 │  全部 ▾  │  🔍  搜漫展 / 摄影师 / 场馆                 [搜索] │  安全  ║ 微信小程序胶囊 ║ 屏边
       └──────────────────────────────────────────────────────────────┘  间距  ╚══════════════╝
       [~62px]   [1px]             flex: 1                      [~40px]         87 × 32 px
       <────────────────────── 整体药丸 flex: 1 ──────────────────────>
```

- **容器高度**：
  - 高度：`64rpx`（32px）。
  - 外层内边距：左 `32rpx`（16px），右由 JS 动态计算 `navRight`（约 102px）。
- **统一药丸容器（.unified-search-pill）**：
  - `flex: 1`，吃满左安全边距与右胶囊安全线之间的**全部空间**。
  - 圆角：`32rpx`（16px）。
  - 背景：`rgba(21, 23, 34, 0.72)`，边框 `1rpx solid rgba(255, 255, 255, 0.16)`。
- **框内分段细节**：
  - 城市分段：左侧 `padding: 0 16rpx 0 20rpx`，字号 `24rpx`（12px），字重 `600`，箭头间距 `4rpx`。固定占宽约 `60px~68px`。
  - 分隔细线（.pill-divider）：宽 `2rpx`（1px），高 `28rpx`（14px），背景 `rgba(255, 255, 255, 0.18)`。
  - 搜索分段：`flex: 1`，左内边距 `14rpx`，放大镜 `26×26rpx`，提示文案 `22rpx`（11px），颜色 `$c-slate-400`。
  - 右端操作钮：微型胶囊宽 `72rpx`（36px），高 `48rpx`，背景 `$brand`，圆角 `24rpx`，字号 `20rpx`。
- **三档宽度实算**：
  - **375px 档**：整框净可用宽 **257px**（占屏宽 68.5%）。其中城市占 62px，分隔线 1px，搜索交互区独享 **194px**。
  - **390px 档**：整框净可用宽 **271px**（占屏宽 69.5%）。搜索交互区独享 **208px**。
  - **414px 档**：整框净可用宽 **295px**（占屏宽 71.3%）。搜索交互区独享 **232px**。

#### 3. 胶囊避让策略
- 胶囊行仅这一个大组件，`flex: 1` 直接顶满至胶囊左边界前 8px。
- 视觉重心非常平衡，整行从左至右呈现一个强有力的横向信息条。

#### 4. 与 Hero 大图关系及材质质感
- **背景材质**：高集成度暗黑拟态，双层背景混合 `backdrop-filter: blur(20px)`，内阴影 `inset 0 1rpx 0 rgba(255, 255, 255, 0.12)`。
- **滚动表现**：滚动时胶囊框背景透明度由 0.72 渐变为 1.0，整体导航栏渐入黑底，与 Hero 舞台解耦平稳。

#### 5. 方案缺点
- **点击意图容易歧义**：用户点在左边是切城市，点在右边是去搜索，两者包在一个容器内，部分初级用户会误以为「点整个条都是去搜索」。必须通过非常清晰的分隔线与点击态高亮反馈（active 时仅城市区域高亮，或仅搜索区域高亮）来做体验防御。

---

### 方案 C：双功能并存 · 纯文字城市 + 紧凑搜索框 + 紧凑铃铛（保守保留方案）
> **灵感样板**：小红书/部分电商保守妥协方案。为强运营通知诉求保留顶栏入口。

#### 1. 结构与 DOM 层级
顶栏同一行容纳三件套：左「纯文字城市」+ 中「紧凑搜索框（`flex:1`）」+ 右「紧凑铃铛圆钮」。
```html
<view class="nav-row" :style="{ paddingTop: navTop + 'px', height: navHeight + 'px', paddingRight: navRight + 'px' }">
  <!-- 1. 城市 -->
  <view class="nav-city-text" @click="goCitySelect">
    <text class="city-name">{{ currentCity }}</text>
    <text class="city-arrow">▾</text>
  </view>
  <!-- 2. 搜索框（被两头夹紧，宽度压缩） -->
  <view class="nav-search-bar-compact" @click="goSearch">
    <image class="search-icon" src="/static/icons/search.svg" mode="aspectFit" />
    <text class="search-placeholder">搜漫展/嘉宾</text>
  </view>
  <!-- 3. 保留消息铃铛（与微信胶囊同行） -->
  <view class="nav-bell-btn" @click="goMessages">
    <image class="bell-icon" src="/static/icons/bell.svg" mode="aspectFit" />
    <view v-if="unreadCount > 0" class="badge-mini">{{ unreadCount > 99 ? '99+' : unreadCount }}</view>
  </view>
</view>
```

#### 2. 精确尺寸规格（rpx 与三档 px）

```
[16px] ┌───────┐ [6px] ┌───────────────────────────┐ [6px] ┌────┐ [8px] ╔══════════════╗ [7px]
左间距 │ 全部 ▾ │ 间距  │ 🔍 搜漫展/嘉宾...          │ 间距  │ 🔔 │  安全  ║ 微信小程序胶囊 ║ 屏边
       └───────┘       └───────────────────────────┘       └────┘  间距  ╚══════════════╝
        ~46px                      flex: 1                  30px         87 × 32 px
```

- **容器高度**：
  - 高度：`64rpx`（32px）。
  - 外层内边距：左 `32rpx`（16px），右由 JS 动态计算 `navRight`（约 102px）。
- **三构件尺寸分配**：
  - 城市文字：紧凑型排版，字号 `24rpx`，占宽约 `46px`。
  - 右侧铃铛按钮：
    - 外框圆角按钮：直径 `60rpx`（30px），圆角 `50%`。
    - 图标大小：`28×28rpx`。
    - 独立红点：`16×16rpx`，带数字时高 `24rpx`、字号 `18rpx`。
    - 占宽：`30px`。
  - 中间搜索框：
    - `flex: 1`，高度 `64rpx`（32px）。
    - 左右间距各自只有 `12rpx`（6px）。
    - 内部文字只能放极短 placeholder（如「搜漫展」），无法容纳右侧「搜索」文字微按钮。
- **三档宽度实算**：
  - **375px 档**：净可用宽 257px。城市 46px + 间距 6px + 铃铛 30px + 间距 6px = 88px。**搜索框仅剩 169px（仅占屏宽 45%）**。
  - **390px 档**：净可用宽 271px。**搜索框得 183px（占屏宽 46.9%）**。
  - **414px 档**：净可用宽 295px。**搜索框得 207px（占屏宽 50.0%）**。

#### 3. 胶囊避让策略
- 铃铛右边缘与微信胶囊左边缘保持 8px 绝对物理间距。
- 必须确保铃铛按钮在胶囊左侧有清晰的轮廓边界，不可发生重叠。

#### 4. 与 Hero 大图关系及材质质感
- 与方案 A 类似，采用暗色磨砂玻璃，但由于元素数量达 3 个，顶部视觉碎片化增加，在海报背景杂乱时视觉燥点偏多。

#### 5. 方案缺点
- **严重拥挤与热区隐患**：在 375px 屏幕上，中间搜索框被压缩到仅 169px，文案稍微放长就会截断；城市、搜索框、铃铛三者之间的间隙被压缩至仅 6px，根据微信适老化指南 3.3「元素间视觉距离至少 2A」「最大热区不超过容器本身」，极易发生触控误触（想点铃铛点到胶囊或搜索框）。
- **认知冗余**：底部 tabBar 明明有「消息」和角标，顶部还放铃铛，浪费顶栏寸土寸金的空间。

---

## 四、三候选方案横向综合比对矩阵

| 比对维度 | 方案 A：纯文字城市 + 假搜索框（消息进底栏） | 方案 B：城市并入搜索框（一体化单框） | 方案 C：城市 + 搜索框 + 铃铛（三件套） |
|---|---|---|---|
| **视觉纯净度与呼吸感** | ⭐⭐⭐⭐⭐ 极简、通透、呼吸感最强 | ⭐⭐⭐⭐ 工业集约感强、极规整 | ⭐⭐ 拥挤、破碎、小屏拥堵 |
| **搜索框宽度（375px基准）** | **201px（53.6% 屏宽）**，极宽敞 | **194px（框体整体 257px / 68.5%）** | **169px（45.0% 屏宽）**，局促 |
| **微信胶囊避让从容度** | ⭐⭐⭐⭐⭐ 完美消化，自然贴合 | ⭐⭐⭐⭐⭐ 完美消化，浑然一体 | ⭐⭐⭐ 铃铛与胶囊紧挨，易局促 |
| **触控热区与无障碍合规** | ⭐⭐⭐⭐⭐ 间距充足，各组件热区独立达标 | ⭐⭐⭐⭐ 需做框内分段热区隔离 | ⭐⭐ 间距仅 6px，濒临误触边缘 |
| **城市入口可发现性** | ⭐⭐⭐⭐ 独立成组，带小箭头，辨识度高 | ⭐⭐⭐ 嵌在框内，初次使用可能稍弱 | ⭐⭐⭐⭐ 独立成组 |
| **顶栏垂直占用释放** | 从 80px 压至 **44~48px**（释放 32px+） | 从 80px 压至 **44~48px**（释放 32px+） | 从 80px 压至 **44~48px**（释放 32px+） |
| **业务贴合度与开发成本** | ⭐⭐⭐⭐⭐ 去除重复消息，动线顺畅 | ⭐⭐⭐⭐ 需重写框内分段点击交互 | ⭐⭐⭐ 仍保留顶栏冗余消息轮询 |

---

## 五、最终推荐方案与核心设计规范（方案 A）

### 5.1 最终推荐结论
**强烈推荐采用「方案 A：纯文字城市 + 胶囊假搜索框 + 消息下沉底栏」**。

### 5.2 核心理由与关键取舍回答
1. **彻底解决「丑」的心理根因**：
   现状的「丑」是因为第一行放了一个小胶囊后，右侧留白了 270px 的大黑洞，第二行又孤零零漂着两个小图标。方案 A 将顶栏精简为**纯单行**，且用 `flex: 1` 的假搜索框把胶囊让出的全部宽度直接「吃满」，右侧不再有空洞，完全符合大麦、猫眼等标杆产品的成熟视觉法则。
2. **坚决砍掉顶部铃铛的正当性**：
   - 底部已有 TabBar 消息页（Tab 2），`src/utils/badge.ts` 原生角标联动已非常成熟。
   - 砍掉顶部铃铛，让搜索框在 375px 屏幕下直接扩容 **32px+**，可以从容展示「搜漫展、嘉宾、摄影师、场馆」完整 Placeholder 与微型「搜索」CTA，点击意图极强。
3. **释放宝贵垂直空间**：
   顶栏从原本畸形的两行（约 80px）收缩至标杆单行（44~48px），为 Hero 海报的大片氛围感以及下方极其关键的「票根（Ticket Stub）」组件争取到了至少 32px 的纵向安全空间，彻底消除小屏设备上 Hero CTA 与票面被下沉裁切的风险。

---

## 六、推荐方案（方案 A）实施级精确样式规格单

为方便后续下游研发人员直接落地，特提供完整的样式参数与代码级规范：

### 6.1 节点结构与布局
```html
<view class="hero-nav-bar" :style="navBarStyle">
  <!-- 城市入口 -->
  <view class="nav-city-btn" @click="goCitySelect">
    <text class="nav-city-name">{{ currentCity }}</text>
    <text class="nav-city-caret">▾</text>
  </view>

  <!-- 搜索框本体 -->
  <view class="nav-search-box" @click="goSearch">
    <image class="search-ico" src="/static/icons/search.svg" mode="aspectFit" />
    <text class="search-ph">搜漫展、嘉宾、摄影师、场馆</text>
    <view class="search-action-pill">搜索</view>
  </view>
</view>
```

### 6.2 SCSS 精确样式表（符合米拉漫展暗房规范）
```scss
// ===== 顶栏容器 =====
.hero-nav-bar {
  position: relative;
  z-index: 10;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  padding-left: 32rpx; // 16px 左栅格
  // padding-right, padding-top, height 由 inline style 动态注入
  width: 100%;
  transition: background-color 0.25s ease, border-color 0.25s ease;
}

// ===== 纯文字城市选择器 =====
.nav-city-btn {
  position: relative;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  height: 64rpx;
  margin-right: 16rpx;
  cursor: pointer;

  // 热区外扩，确保符合 >= 44×44px 适老化触控规范
  &::after {
    content: '';
    position: absolute;
    top: -12rpx;
    bottom: -12rpx;
    left: -12rpx;
    right: -12rpx;
  }

  &:active {
    opacity: 0.75;
    transform: scale(0.97);
  }
}

.nav-city-name {
  font-size: 26rpx;
  font-weight: 600;
  color: #F8FAFC; // $c-slate-50
  letter-spacing: 0.5rpx;
  max-width: 120rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nav-city-caret {
  margin-left: 6rpx;
  font-size: 18rpx;
  color: rgba(255, 255, 255, 0.6);
  transform: translateY(1rpx);
}

// ===== 胶囊假搜索框 =====
.nav-search-box {
  flex: 1;
  display: flex;
  align-items: center;
  height: 64rpx; // 32px，与微信胶囊高度严格 1:1 对齐
  box-sizing: border-box;
  padding-left: 20rpx;
  padding-right: 8rpx;
  background: rgba(21, 23, 34, 0.68); // 磨砂深色底
  border: 1rpx solid rgba(255, 255, 255, 0.14);
  border-radius: 32rpx; // 完全胶囊圆角
  backdrop-filter: blur(16px);
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.25);
  transition: all 0.2s cubic-bezier(0.2, 0, 0, 1);

  &:active {
    background: rgba(29, 32, 47, 0.85);
    border-color: rgba(255, 107, 0, 0.4); // 琥珀橙聚焦提示
  }
}

.search-ico {
  width: 28rpx;
  height: 28rpx;
  margin-right: 12rpx;
  opacity: 0.65;
  flex-shrink: 0;
}

.search-ph {
  flex: 1;
  font-size: 24rpx;
  color: #94A3B8; // $c-slate-400
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-action-pill {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 48rpx;
  padding: 0 16rpx;
  background: $brand; // #FF6B00 琉珀橙
  border-radius: 24rpx;
  font-size: 20rpx;
  font-weight: 600;
  color: #FFFFFF;
  box-shadow: 0 2rpx 8rpx rgba(255, 107, 0, 0.35);
}
```

### 6.3 脚本计算与环境适配逻辑（TypeScript）
```ts
// 状态栏与胶囊避让响应式计算
const navBarStyle = computed(() => {
  let navTop = 44 // 默认兜底（px）
  let navHeight = 32 // 默认胶囊高（px）
  let navRight = 102 // 默认胶囊避让右内边距（px）

  try {
    const rect = uni.getMenuButtonBoundingClientRect()
    const sys = uni.getSystemInfoSync()
    if (rect && rect.top && rect.height) {
      navTop = rect.top
      navHeight = rect.height
      // 屏幕右侧避让距离 = 胶囊右留白 + 胶囊宽 + 8px 安全呼吸间距
      navRight = (sys.windowWidth - rect.left) + 8
    }
  } catch (e) {
    // 降级兜底保持默认
  }

  return {
    paddingTop: `${navTop}px`,
    height: `${navHeight}px`,
    paddingRight: `${navRight}px`
  }
})
```

---

## 七、交付结论总结

本方案依据主流平台调研（t1）与米拉漫展品牌基调，给出了 3 个经过严格尺寸账本计算的候选方案，并最终推荐了兼具极简美感与工程健壮性的**方案 A**。
- **砍掉顶栏铃铛**：消除与底部 tabBar 消息页的重复曝光，为搜索框释放关键横向空间。
- **单行化重构**：从两行 ≈80px 压至单行 ≈48px，挽救被挤压的 Hero 主视觉与票根。
- **精确吸收胶囊让位**：通过 `flex:1` 假搜索框从容吸收胶囊避让区，彻底告别 270px 视觉黑洞。
- **严格遵循代码冻结约束**：本文档完全作为设计规范产出在 `comic/doc/review/hero-nav-redesign.md`，未改动任何源码文件。
