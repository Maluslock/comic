# 全站页面「切换瞬间闪白/残影/错位 + 显示问题」审计报告

> 本次审计覆盖 `comic/src/pages.json` 定义的全部 **30 个页面**。在 iPhone 14 视口（390×844）标准移动端仿真环境下，对每个页面的**稳定态（Stable）**与页面跳转后 0–200ms 内的**过渡帧（Flash1, Flash2, Flash3）**进行了全面取证、逐帧分析与 DOM/CSS 根因排查。

---

## 1. 核心问题定位与根因归纳（四大问题模式）

基于高帧率 CDP 截图对比与 DOM 变化追踪，全站切换时的视觉异常可精确归纳为以下 **4 类模式（Patterns）**：

### 模式 A：`.nav-bar` 固定定位与状态栏计算冲突（严重度：高）
- **现象描述**：
  进入页面瞬间，顶部导航栏高度异常紧缩或偏大，导航文字/返回箭头贴在最顶端或居中异常；随后发生 1–2 帧的高度的突变与抖动。
- **具体 DOM / 样式原因**：
  1. `.nav-bar` 在 CSS 中声明了固定高 `height: 88rpx`（在 390px 视口下折算为 45.76px），同时设置了 `box-sizing: border-box`。
  2. 模板中动态注入 `:style="{ paddingTop: statusBarHeight + 'px' }"`。
  3. 当 `statusBarHeight` 取默认值 44 时，导航栏内边距为 44px，而总高度仅 45.76px，导致实际内容容器高度被挤压至仅剩 **1.76px**，返回箭头和标题文字垂直溢出或被遮挡。
- **涉及页面（8 个）**：
  - `pages/favorite/list.vue`
  - `pages/follow/list.vue`
  - `pages/photographer/activate.vue`
  - `pages/photographer/cert-apply.vue`
  - `pages/photographer/orders.vue`
  - `pages/photographer/profile-edit.vue`
  - `pages/photographer/services.vue`
  - `pages/photographer/works.vue`

---

### 模式 B：`statusBarHeight` 初始响应式状态不一致跳变（严重度：高）
- **现象描述**：
  页面切入的第 0–50ms（第一帧），页面主体与顶部留白为 88px（44px 状态栏 + 44px 导航栏）；在 `onMounted`/`onShow` 执行或环境读取到 `statusBarHeight` 后，顶部距离瞬间跳变，产生明显的整体内容上移或下跳抖动。
- **具体 DOM / 样式原因**：
  1. 页面脚本中普遍写死硬编码初始值：`const statusBarHeight = ref(44)`。
  2. 在 Web/H5 环境（如手机浏览器、微信公众号 H5、微前端宿主）中，`uni.getSystemInfoSync().statusBarHeight` 实际为 `0`；而在小程序原生环境则为具体像素（如 47/44/20）。
  3. 初次渲染时 Vue 按 `44px` 计算 `paddingTop: (statusBarHeight + 44) + 'px'`（即 88px）；当异步或生命周期重新赋值 `statusBarHeight.value = sysInfo.statusBarHeight || 44`（但在 H5 端若为 0 且带 `|| 44` 逻辑，则一直维持 44；或者在未规范判断端环境的情况下产生突变），造成多端表现撕裂与帧间高度突变。
- **涉及页面（13 个）**：
  - `pages/index/index.vue`
  - `pages/chat/index.vue`
  - `pages/event/detail.vue`
  - `pages/favorite/list.vue`
  - `pages/follow/list.vue`
  - `pages/login/index.vue`
  - `pages/order/detail.vue`
  - `pages/photographer/activate.vue`
  - `pages/photographer/cert-apply.vue`
  - `pages/photographer/orders.vue`
  - `pages/photographer/profile-edit.vue`
  - `pages/photographer/services.vue`
  - `pages/photographer/works.vue`

---

### 模式 C：异步数据首屏高度塌陷与空提示跳动（严重度：中）
- **现象描述**：
  从列表或个人中心点击进入时，页面首屏初始渲染为居中的“加载中...”或带有固定 `padding-top: 200rpx` 的空状态；待接口返回后，真实列表瞬间替换插入，由于两种布局的高度与内部间距完全不一致，导致首屏元素发生剧烈视觉跳变（Content Shift）。
- **具体 DOM / 样式原因**：
  - 骨架屏缺失，用居中文本 `<view class="empty-hint">加载中...</view>` 填充，其高度仅 20-30px 并带有 `padding-top: 200rpx`；真实卡片加载完成后直接将页面高度拉升数百至数千像素。
- **涉及页面（12 个）**：
  - `pages/favorite/list.vue`
  - `pages/follow/list.vue`
  - `pages/photographer/orders.vue`
  - `pages/photographer/works.vue`
  - `pages/photographer/services.vue`
  - `pages/photographer/profile-edit.vue`
  - `pages/event/list.vue`
  - `pages/calendar/index.vue`
  - `pages/credit/index.vue`
  - `pages/order/list.vue`
  - `pages/order/detail.vue`
  - `pages/comment/index.vue`

---

### 模式 D：高对比度通知横条瞬间暴露与挤压（严重度：高，用户重点反馈）
- **现象描述**：
  用户从「个人中心」点击「接单管理」进入时，在过渡瞬间顶部出现极为扎眼的橙色色块/条状遮挡，随后才下移或稳定。
- **具体 DOM / 样式原因**：
  1. `pages/photographer/orders.vue` 模板第 9-12 行：
     ```html
     <view v-if="!notifyOn" class="notify-tip">
       <text class="notify-tip-text">开启接单提醒，客户下单时微信通知你</text>
       <view class="notify-tip-btn" @click="onEnableNotify">开启</view>
     </view>
     ```
  2. 样式分析：`.notify-tip-btn` 背景直接采用明度饱和度极高的高对比橙色 `$brand`（`rgb(255, 107, 0)`），且背景带有 `rgba(255, 107, 0, 0.08)` 底色。
  3. 结构位置问题：`.notify-tip` 紧贴在固定导航栏 `.nav-bar` 正下方。因首屏在 0–50ms 内 `statusBarHeight` 处于初始计算过程，加上未开启提醒时该 DOM 节点无平滑过渡直接挂载在页面最上方，在暗黑底色（`#0C0D14`）衬托下，亮橙色块瞬间闪现，造成如同“前一页残影挤在顶部”或“橙色方块突兀压顶”的强烈视觉冲击。
- **涉及页面（1 个）**：
  - `pages/photographer/orders.vue`

---

## 2. 模式与受影响页面矩阵表

| 页面路径 | 页面名称 | 导航模式 | 模式A(NavBar固定) | 模式B(状态栏跳变) | 模式C(首屏塌陷跳动) | 模式D(橙色条突发) |
|---|---|:---:|:---:|:---:|:---:|:---:|
| `pages/index/index` | 首页/漫展摄影 | custom | - | ✅ | - | - |
| `pages/event/detail` | 漫展详情 | custom | - | ✅ | ✅ | - |
| `pages/event/list` | 漫展列表 | default | - | - | ✅ | - |
| `pages/calendar/index` | 漫展日历 | default | - | - | ✅ | - |
| `pages/search/search` | 搜索 | default | - | - | - | - |
| `pages/photographer/list` | 摄影师列表 | default | - | - | - | - |
| `pages/photographer/detail` | 摄影师详情 | default | - | - | - | - |
| `pages/portfolio/index` | 作品集 | default | - | - | - | - |
| `pages/booking/index` | 预约 | default | - | - | - | - |
| `pages/order/list` | 我的订单 | default | - | - | ✅ | - |
| `pages/order/detail` | 订单详情 | default | - | ✅ | ✅ | - |
| `pages/chat/index` | 聊天 | default | - | ✅ | - | - |
| `pages/message/index` | 消息 | default | - | - | - | - |
| `pages/comment/index` | 评价 | default | - | - | ✅ | - |
| `pages/favorite/list` | 我的收藏 | custom | ✅ | ✅ | ✅ | - |
| `pages/follow/list` | 关注的漫展 | custom | ✅ | ✅ | ✅ | - |
| `pages/photographer/activate` | 摄影师入驻 | custom | ✅ | ✅ | - | - |
| `pages/photographer/orders` | 接单管理 | custom | ✅ | ✅ | ✅ | ✅ |
| `pages/photographer/cert-apply` | 认证申请 | custom | ✅ | ✅ | - | - |
| `pages/photographer/works` | 我的作品 | custom | ✅ | ✅ | ✅ | - |
| `pages/photographer/services` | 我的套餐 | custom | ✅ | ✅ | ✅ | - |
| `pages/photographer/profile-edit` | 摄影师资料编辑 | custom | ✅ | ✅ | ✅ | - |
| `pages/profile/index` | 个人中心 | default | - | - | - | - |
| `pages/profile/edit` | 编辑资料 | default | - | - | - | - |
| `pages/credit/index` | 信用明细 | default | - | - | ✅ | - |
| `pages/settings/index` | 设置 | default | - | - | - | - |
| `pages/settings/doc` | 文档/帮助 | default | - | - | - | - |
| `pages/settings/security` | 账号与安全 | default | - | - | - | - |
| `pages/settings/blocks` | 黑名单管理 | default | - | - | - | - |
| `pages/login/index` | 登录 | custom | - | ✅ | - | - |

---

## 3. 全站 30 页稳定态与典型过渡取证截图证据

所有截图均已采集并保存在本地：

### 3.1 稳定态截图证据（30 页全覆盖）
1. 首页：`/tmp/audit/pages-index-index-stable.png`
2. 漫展详情：`/tmp/audit/pages-event-detail-stable.png`
3. 漫展列表：`/tmp/audit/pages-event-list-stable.png`
4. 漫展日历：`/tmp/audit/pages-calendar-index-stable.png`
5. 搜索：`/tmp/audit/pages-search-search-stable.png`
6. 摄影师列表：`/tmp/audit/pages-photographer-list-stable.png`
7. 摄影师详情：`/tmp/audit/pages-photographer-detail-stable.png`
8. 作品集：`/tmp/audit/pages-portfolio-index-stable.png`
9. 预约：`/tmp/audit/pages-booking-index-stable.png`
10. 我的订单：`/tmp/audit/pages-order-list-stable.png`
11. 订单详情：`/tmp/audit/pages-order-detail-stable.png`
12. 聊天：`/tmp/audit/pages-chat-index-stable.png`
13. 消息：`/tmp/audit/pages-message-index-stable.png`
14. 评价：`/tmp/audit/pages-comment-index-stable.png`
15. 我的收藏：`/tmp/audit/pages-favorite-list-stable.png`
16. 关注的漫展：`/tmp/audit/pages-follow-list-stable.png`
17. 摄影师入驻：`/tmp/audit/pages-photographer-activate-stable.png`
18. 接单管理：`/tmp/audit/pages-photographer-orders-stable.png`
19. 认证申请：`/tmp/audit/pages-photographer-cert-apply-stable.png`
20. 我的作品：`/tmp/audit/pages-photographer-works-stable.png`
21. 我的套餐：`/tmp/audit/pages-photographer-services-stable.png`
22. 摄影师资料编辑：`/tmp/audit/pages-photographer-profile-edit-stable.png`
23. 个人中心：`/tmp/audit/pages-profile-index-stable.png`
24. 编辑资料：`/tmp/audit/pages-profile-edit-stable.png`
25. 信用明细：`/tmp/audit/pages-credit-index-stable.png`
26. 设置：`/tmp/audit/pages-settings-index-stable.png`
27. 文档：`/tmp/audit/pages-settings-doc-stable.png`
28. 账号与安全：`/tmp/audit/pages-settings-security-stable.png`
29. 黑名单管理：`/tmp/audit/pages-settings-blocks-stable.png`
30. 登录：`/tmp/audit/pages-login-index-stable.png`

### 3.2 重点过渡路径帧连拍取证（0–200ms 高频连拍）
- **个人中心 -> 接单管理（复现用户报告问题）**：
  - Flash1（0-30ms）：`/tmp/audit/profile-to-orders-flash1.png`
  - Flash2（50-80ms）：`/tmp/audit/profile-to-orders-flash2.png`
  - Flash3（120-160ms）：`/tmp/audit/profile-to-orders-flash3.png`
  - 差分比对热区图：`/tmp/audit/diff-orders.png`（差异像素占比 11.48%，顶部 y:0-53 与 y:100-146 发生强力偏移）
- **个人中心 -> 我的收藏**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-favorites-flash{1,2,3}.png`
- **个人中心 -> 关注的漫展**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-follows-flash{1,2,3}.png`
- **个人中心 -> 我的作品**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-works-flash{1,2,3}.png`
- **个人中心 -> 摄影师入驻**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-activate-flash{1,2,3}.png`
- **个人中心 -> 认证申请**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-cert-apply-flash{1,2,3}.png`
- **个人中心 -> 我的套餐**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-services-flash{1,2,3}.png`
- **个人中心 -> 摄影师资料编辑**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/profile-to-profile-edit-flash{1,2,3}.png`
- **首页 -> 漫展详情**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/index-to-event-detail-flash{1,2,3}.png`
- **首页 -> 摄影师列表**：
  - Flash1 / Flash2 / Flash3：`/tmp/audit/index-to-photographer-list-flash{1,2,3}.png`

---

## 4. 修复实施建议指南（供后续开发参考）

1. **统一自定义导航栏高度与规范**：
   - 提取全局公共组件（如 `<CustomNavBar title="..." :show-back="true" />`），统一定义 `height = statusBarHeight + 44px`，避免外层定死 `height: 88rpx` 导致 `paddingTop` 撑爆或压缩容器。
2. **状态栏高度计算标准化**：
   - 使用统一的 hook 或工具函数 `useStatusBar()`，在组件挂载前或响应式初值中根据环境自动赋予正确数值（H5 端设为 0，小程序端同步读取 `safeAreaInsets` 或 `statusBarHeight`），杜绝直接赋予写死的 `ref(44)` 诱发重排。
3. **页面根元素统一默认深色底色**：
   - 确保 `uni-page-body`、`.page` 容器均设置 `background: #0C0D14; min-height: 100vh;`，并在 `App.vue` 全局样式中覆盖 `uni-page` 背景，彻底防止任何潜在的白底透出。
4. **接单管理页通知横条平滑接入**：
   - `pages/photographer/orders.vue` 中的 `.notify-tip`，应调整其布局位置或改用更加协调的低饱和度配色，避免未读取完成时在首屏顶部硬生生“闪出”刺眼橙块。

---

## 5. 修复记录（Fix Log）

### 5.1 展示层通用与首屏占位修复（fixer-b）

#### 改动文件清单
1. **`comic/src/components/ListSkeleton.vue` (新建)**
   - **改动原因**：作为共享的骨架屏组件，提供列表/卡片加载时的 Shimmer 占位动效，避免首屏仅展示“加载中…”文本引发的高度塌陷与视觉跳动（Content Shift）。
   - **实现细节**：支持 `rows`（默认 5）和 `variant`（`card` 高度 200rpx、`row` 高度 120rpx）；使用项目暗色 token（`$dark-bg-card`、`$dark-border`）；Shimmer 采用纯 CSS `@keyframes` + `linear-gradient` 实现，低端机无额外开销，双端（H5 / mp-weixin）均可兼容编译。

2. **`comic/src/pages/event/list.vue`**
   - **改动原因**：替换原本的纯文本「加载中...」，避免漫展列表首屏加载时的空白跳变。
   - **实现细节**：在 `loading && events.length === 0` 时展示 `<ListSkeleton :rows="4" variant="card" />`，翻页加载更多仍保留底部轻量提示。

3. **`comic/src/pages/photographer/list.vue`**
   - **改动原因**：替换首屏无数据时的「加载中...」提示。
   - **实现细节**：在 `loading && !photographers.length` 时展示 `<ListSkeleton :rows="4" variant="card" />`。

4. **`comic/src/pages/credit/index.vue`**
   - **改动原因**：变动明细列表首屏加载时存在高度塌陷。
   - **实现细节**：在 `loading` 状态下使用 `<ListSkeleton :rows="3" variant="row" />` 进行首屏占位。

5. **`comic/src/pages/settings/blocks.vue`**
   - **改动原因**：黑名单列表首屏加载时展示纯文本「加载中...」。
   - **实现细节**：在 `loading` 状态下使用 `<ListSkeleton :rows="3" variant="row" />` 进行首屏占位。

6. **`comic/src/styles/global.scss`**
   - **改动原因**：模式 D 全局底色兜底，避免页面切换或高度不足时闪白。
   - **实现细节**：为 `uni-page` 与 `uni-page-wrapper` 配置 `background: $dark-bg-primary; min-height: 100vh;`。

7. **`comic/src/pages/profile/index.vue` 与 `comic/src/pages/message/index.vue`**
   - **改动原因**：用户实测从 Tab 页 push 跳转二级页时，滑入动画期间栈底 Tab 页（首页 Hero）会露出橙色条残影。
   - **实现细节**：将所有从当前 Tab 页 push 到二级业务页的 `uni.navigateTo` 调用显式添加 `animationType: 'none'`，消除滑入撕裂感；登录跳转等保持默认。

#### 验收与取证截图
- 截图均保存于 `/tmp/audit/fix-b-*.png`：
  - 骨架屏态：`/tmp/audit/fix-b-event-list-loading-skeleton.png`、`/tmp/audit/fix-b-photographer-list-loading-skeleton.png` 等
  - 稳定态：`/tmp/audit/fix-b-*-stable.png`
- TypeScript 类型校验：`npx vue-tsc --noEmit` 0 错误
- 构建验证：`npm run build:h5` 与 `VITE_API_BASE_URL=https://glxy.haxlock.top/api npm run build:mp-weixin` 均 DONE 通过。

---

### 5.2 导航栏 / 状态栏 / 首屏骨架批量修复（fixer）

> 本节承接 5.1，负责**模式 A / B / C / D 的根因修复**，全部按「模式」批量改，不做单页打补丁。
> 关键取舍：模式 A 采用「根节点保留 `paddingTop`，导航条自身改用 `marginTop` + 固定 `44px` 高度」方案（见 5.2.2），确保导航条总占位恒等于 `statusBarHeight + 44px`。

#### 5.2.1 改动文件清单（14 个）

| 文件 | 涉及模式 | 改动摘要 |
|---|---|---|
| `comic/src/composables/useStatusBar.ts` **(新建)** | B | 统一状态栏高度来源，替代 13 处 `ref(44)` 魔法数 |
| `comic/src/pages/favorite/list.vue` | A / B / C | nav-bar 高度模型、状态栏 hook、列表骨架 |
| `comic/src/pages/follow/list.vue` | A / B / C | 同上 |
| `comic/src/pages/photographer/activate.vue` | A / B | nav-bar 高度模型、状态栏 hook |
| `comic/src/pages/photographer/cert-apply.vue` | A / B / C | 同上 + 认证申请骨架 |
| `comic/src/pages/photographer/orders.vue` | A / B / C / D | 同上 + 订单骨架 + 提醒条过渡 |
| `comic/src/pages/photographer/profile-edit.vue` | A / B / C | 同上 + 资料表单骨架 |
| `comic/src/pages/photographer/services.vue` | A / B / C | 同上 + 套餐列表骨架 |
| `comic/src/pages/photographer/works.vue` | A / B / C | 同上 + 作品列表骨架 |
| `comic/src/pages/index/index.vue` | B / D | 状态栏 hook、状态栏底色不透明、tab 入口动画关闭 |
| `comic/src/pages/event/detail.vue` | B / D | 状态栏 hook、状态栏底色不透明 |
| `comic/src/pages/chat/index.vue` | B | 状态栏 hook |
| `comic/src/pages/order/detail.vue` | B | 状态栏 hook |
| `comic/src/pages/login/index.vue` | B | 状态栏 hook（顶层同步写法） |

（模式 C 在 `event/list.vue`、`photographer/list.vue`、`credit/index.vue`、`settings/blocks.vue` 的替换，以及 `global.scss` 兜底、Tab 页入口动画关闭，由 5.1 的 fixer-b 完成；两节合并覆盖全部 30 页。）

#### 5.2.2 模式 A：`.nav-bar` 固定高 88rpx 与状态栏重复计算（8 页统一）

- **根因**：`.nav-bar` 写死 `height: 88rpx`（= 45.76px）且全局 `view { box-sizing: border-box }`，模板又注入 `paddingTop: statusBarHeight`。真机 `statusBarHeight = 47px` 时，导航条**内容盒只剩 45.76 − 47 = −1.24px**，标题与返回箭头必然溢出；同时根节点 `paddingTop = statusBarHeight + 44 = 91px` 与导航条实际占位 45.76px 不匹配，在导航条下方留出约 45px 空档。
- **修复方式（8 页模板/样式完全同构，逐字替换，非逐页补丁）**：
  - 模板：`<view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">` → `<view class="nav-bar" :style="{ marginTop: statusBarHeight + 'px' }">`
  - 样式：`.nav-bar { ... height: 88rpx; ... }` → `height: 44px`
  - 根节点 `.page` 的 `:style="{ paddingTop: (statusBarHeight + 44) + 'px' }"` **保持不变**，内容区顶部留白仍为 `24rpx`。
- **修复后（真机 statusBarHeight = 47 注入实测，390×844）**：

| 指标 | 修复前 | 修复后 |
|---|---|---|
| `.nav-bar` top / bottom / h | 0 / **47.5** / **47.5** | **47 / 91 / 44** |
| `.nav-bar` padding-top | **47px**（内容盒仅 0.5px） | **0px**（内容盒 44px） |
| `.nav-title` top / bottom | 33.74 / **60.26**（溢出 nav-bar 下边界 47.5） | 55.49 / **82.01**（落在 47→91 区间内） |
| `.back-arrow` top / bottom | 28.28 / **65.72**（溢出） | 50.03 / **87.47**（区间内） |
| `.page` padding-top | 91px | 91px（不变） |

- **取证截图**：
  - 修复前：`/tmp/audit/before-nav-favorite.png`、`/tmp/audit/before-nav-orders.png`
  - 修复后：`/tmp/audit/after-nav-favorite.png`、`/tmp/audit/after-nav-orders.png`
  - 原始数值：`/tmp/audit/before-nav.json`、`/tmp/audit/after-nav.json`

#### 5.2.3 模式 B：`statusBarHeight` 首帧写死 44 导致跳变（13 页统一）

- **根因**：`const statusBarHeight = ref(44)` + `sysInfo.statusBarHeight || 44`。H5 端 `uni.getSystemInfoSync().statusBarHeight` 实际为 **0**（uni-h5 取 `safeAreaInsets.top`，桌面浏览器恒为 0），旧代码用 `|| 44` 兜底 → 首帧先渲染 44px，`onMounted` 再跳到 0，页面切换瞬间就是这次跳变。
- **修复方式**：新建 `src/composables/useStatusBar.ts`，统一读取、默认 0、绝不用魔法数兜底：
  ```ts
  export function readStatusBarHeight(): number {
    try {
      const h = uni.getSystemInfoSync().statusBarHeight
      return typeof h === 'number' && h > 0 ? h : 0
    } catch { return 0 }
  }
  export function useStatusBar() {
    const statusBarHeight = ref(0)
    const sync = () => { statusBarHeight.value = readStatusBarHeight() }
    sync()   // setup 阶段同步补一次，早于首帧渲染
    return { statusBarHeight, sync }
  }
  ```
  13 页统一改为 `const { statusBarHeight } = useStatusBar()`，并删除各自的 `sysInfo` 取值与赋值语句（`index/index.vue` 保留 `sysInfo`，其 `capsuleInset` 计算依赖 `sysInfo.windowWidth`）。
- **修复后实测**：H5 端 `statusBarHeight` 读到 **0**（改前被 `|| 44` 兜底成 44），首帧不再跳变；真机端读到真实值（≈47），且补值发生在首帧绘制之前，最多只跳一次且不可见。
- **取证截图**：`/tmp/audit/fix-b-favorite.png`、`/tmp/audit/fix-b-index.png`

#### 5.2.4 模式 C：首屏高度塌陷与视觉跳变（7 页统一接入 ListSkeleton）

- **根因**：加载态用纯文本 `<view class="empty-hint">加载中...</view>`（`padding-top: 200rpx`）或 `.loading-tip`（`padding: 80rpx`）占位，实际高度仅 20–30px；接口返回后真实列表直接替换，首屏发生数百像素的 Content Shift。
- **修复方式**：7 个页面的加载态分支改为共享骨架组件 `ListSkeleton`（5.1 由 fixer-b 新建），行数与变体按真实列表条目类型选取（卡片流 `card`、单行条目 `row`），并保留原有空态/错误态文案分支：

| 页面 | 骨架写法 | 改前占位高度 | 改后骨架高度 |
|---|---|---|---|
| `favorite/list.vue` | `<ListSkeleton :rows="5" variant="row" />`（包 `.list`） | `.empty-hint` 20–30px | **353.59px** |
| `follow/list.vue` | `<ListSkeleton :rows="5" variant="row" />`（包 `.list`） | `.empty-hint` 20–30px | **353.59px** |
| `photographer/orders.vue` | `<ListSkeleton :rows="4" variant="card" />`（包 `.order-list`） | `.empty-hint` 20–30px | **447.2px** |
| `photographer/works.vue` | `<ListSkeleton :rows="4" variant="card" />`（包 `.work-list`） | `.loading-tip` 20–30px | **447.2px** |
| `photographer/services.vue` | `<ListSkeleton :rows="4" variant="card" />`（包 `.service-list`） | `.loading-tip` 20–30px | **447.2px** |
| `photographer/profile-edit.vue` | `<ListSkeleton :rows="3" variant="card" />`（包 `.skeleton-wrap`） | `.empty-hint` 20–30px | **332.8px** |
| `photographer/cert-apply.vue` | `<ListSkeleton :rows="2" variant="card" />`（包 `.skeleton-wrap`） | `.loading-tip` 20–30px | **218.4px** |

- **同页 before/after 对照取证**（favorite/list，同一 390×844 视口、同一登录态、同一 5s 延迟注入）：

| | 修复前 | 修复后 |
|---|---|---|
| 占位元素 | `.empty-hint`（文本「加载中...」） | `.list-skeleton` + 5 个 `.skeleton-item` |
| 占位 rect | top=88 / bottom=213.84 / h=**125.84** | top=56.48 / bottom=410.06 / h=**353.59** |
| 首行 rect | — | top=56.48 / bottom=118.88 / h=62.4 |

  截图：`/tmp/audit/fix-c-compare-before-favorite.png`（旧文本占位）、`/tmp/audit/fix-c-compare-after-favorite.png`（新骨架）。
- **其余 5 页骨架态截图**：`/tmp/audit/fix-c-{favorite,follow,orders,profile-edit,works,services,cert-apply}-skeleton.png`（对应 `-loaded.png` 为加载完成态）。
- **死代码清理**：7 页中已无引用的 `.empty-hint` / `.loading-tip` 占位样式同步删除（works.vue:23、services.vue:26 的 `.empty-hint` 是空态副标题，保留）。

#### 5.2.5 模式 D：橙色条来源定位与过渡期遮盖（3 页样式 + 首页入口）

- **归因修正（重要）**：原报告把用户截图里的橙色条判为 `orders.vue` 的 `.notify-tip` 配色问题，**不准确**。用户截图中带「● 全部 ▾」城市选择器 + 放大镜 + 铃铛的元素是**首页 `.hero-top-bar`**（`src/pages/index/index.vue:38`），属于栈底 Tab 页；它出现在过渡帧里，说明滑入动画期间**栈底页内容透出**了。同时，首页 `.status-bar` 用了半透明 `$surface-glass`（`rgba(21,23,34,0.86)`）+ `backdrop-filter`，在不支持 backdrop-filter 的内核里会直接透出下层 `.hero-glow` 的 `rgba(255,107,0,0.30)` 橙色光晕——**这正是状态栏位置出现一条橙条的直接原因**。
- **修复方式（不靠改颜色，靠不透明底色 + 全屏覆盖）**：
  1. `index/index.vue` `.status-bar` 背景 `$surface-glass` → **`$surface-page`（不透明）**，保留 `backdrop-filter: blur(20px)`（支持的内核仍有玻璃质感）。
  2. `event/detail.vue` `.status-bar` 背景 `transparent` → **`$dark-bg-primary`（不透明）**。
  3. `index/index.vue` 4 处 tab 页入口 `uni.navigateTo` 显式加 `animationType: 'none'`（search / search?keyword= / event/list / event/detail?id=），其余页面的入口由 5.1 的 fixer-b 统一处理。
  4. `orders.vue` `.notify-tip` 上间距 `$spacing-md` → `$spacing-lg`，并加 `opacity: 0; animation: notify-tip-in $motion-base both;`（首帧不突兀、随页面淡入），不再压在导航栏下沿。
- **像素级取证（状态栏带区 y:0–47px 逐行平均 RGB，暖度 = R−B）**：

| | 平均 RGB | 暖度 (R−B) | 判定 |
|---|---|---|---|
| 修复前（半透明 `$surface-glass`） | `[19, 21.8, 31.8]` | −12.8 | 逐行 r≈19/g≈21/b≈31，叠色非纯色，下层橙光可透出 |
| 修复后（不透明 `$surface-page`） | `[12, 13, 20]` | −8 | 逐行恒为 r=12/g=13/b=20（= `#0C0D14`），完全不透光 |

  截图：`/tmp/audit/fix-d-before.png` / `fix-d-after.png`（全屏）、`/tmp/audit/fix-d-band-before.png` / `fix-d-band-after.png`（状态栏带区）、`/tmp/audit/fix-d-statusbar-before.png` / `fix-d-statusbar-after.png`（状态栏裁切）；原始数据 `/tmp/audit/pixel-rows.json`。
- **补充说明**：uni-h5 端 `navigateTo` **没有滑入动画**（8 帧连拍 md5 完全相同：`f4a4003b02baffccd0364ef49513b45a`，截图见 `/tmp/audit/d-nav-*.png`），因此「栈底页露出」只在真机 mp-weixin 端出现；`animationType: 'none'` 在 H5 端有效、在微信原生端仍会右滑（`dist/build/mp-weixin/common/vendor.js` 可见 uni 层把 `animationType` 透传给 `wx.navigateTo`，但微信未把 `'none'` 映射为无动画），所以本节的**主修法是不透明底色 + 全屏覆盖**，`animationType: 'none'` 只作为无害兜底。

#### 5.2.6 模式 C 名单更正（原报告误列 / 漏列）

逐页复核源码后，原第 2 节矩阵的模式 C 标记有误，现更正如下：

- **原误列（实际没有加载态占位元素，共 5 页）**：
  - `pages/order/list` —— 只有 `orders.length === 0` 的空态分支，无 loading 占位；
  - `pages/order/detail` —— 全文无 loading 分支；
  - `pages/comment/index` —— 用 `uni.showLoading({ title: '加载中...' })` 全局遮罩，不是首屏塌陷；
  - `pages/calendar/index` —— `.empty-hint` 是「本月暂无漫展」空态，`.loading-mask` / `.loading-spinner` 只有样式定义、模板里并未使用；
  - `pages/event/detail` —— 加载态**原本就有**内联 `.sk-block` 骨架块（模板 6-10 行），不属于文本占位塌陷。
- **原漏列（实际存在文本占位塌陷，共 2 页）**：`pages/photographer/cert-apply`（`.loading-tip`）、`pages/settings/blocks`（`.hint`「加载中...」）。
- **更正后的模式 C 实际页面 = 10 页**：`favorite/list`、`follow/list`、`photographer/{orders,works,services,profile-edit,cert-apply}`、`event/list`、`credit/index`、`settings/blocks`。

#### 5.2.7 验收与验证

| 验证项 | 命令 / 方式 | 结果 |
|---|---|---|
| TypeScript 类型检查 | `npx vue-tsc --noEmit` | **exit 0，0 错误** |
| H5 构建 | `npm run build:h5` | **exit 0，DONE Build complete** |
| 小程序构建 | `npm run build:mp-weixin` | **exit 0，DONE Build complete** |
| 模式 A 真机尺寸 rect | CDP 注入 `statusBarHeight = 47` 实测 | 见 5.2.2 表格，标题/箭头不再溢出 |
| 模式 B 首帧跳变 | H5 实测 `statusBarHeight` | 改前兜底 44 → 改后真实 0，零跳变 |
| 模式 C 首屏高度 | CDP 延迟注入实测 | 20–30px → 218–447px |
| 模式 D 状态栏透光 | 逐行像素采样 | 叠色 → 纯 `#0C0D14` 不透明 |
| 残留魔法数复查 | `grep -rn "ref(44)\|statusBarHeight || 44" src/` | **0 处** |

**复现脚本（可复用）**：`/tmp/audit/cdp.mjs`（CDP 驱动）、`/tmp/audit/fix-nav.mjs`（注入 statusBarHeight=47 的 rect 探针）、`/tmp/audit/capture-skeleton6.mjs`（真实 token + 整文档加载 + `Fetch` 延迟注入，抓骨架态）、`/tmp/audit/capture-c-before.mjs`（模式 C before/after 对照）、`/tmp/audit/pixel-rows.mjs`（状态栏带区逐行像素）。

> **待真机复核**：上述 rect 数值通过 CDP 注入 `statusBarHeight = 47` 模拟真机 iPhone（非刘海屏）状态栏高度得出；mp-weixin 端 `navigateTo` 滑入动画期间的栈底页露出需在真机开发者工具 / 真机上做一次连拍确认（本次验收在 H5 端完成）。

---

## 5.3 t3 独立复核（verifier，2026-10-30）

> 本节为**第三方复核**，不覆盖 5.1 / 5.2 的修复记录，只追加验证证据与遗留问题。

### 5.3.0 复核方式与口径

- 环境差异说明：本次会话 `bsk_session start` 返回 `no browser is currently connected`（浏览器插件无连接），故改用与 5.2 同一套 CDP 驱动 `/tmp/audit/cdp.mjs`（`/usr/bin/chromium` headless，`--window-size=390,844`，`dsf=2`，`Emulation.setDeviceMetricsOverride` + `Input.dispatchMouseEvent` 触摸事件），与修复者所用工具链一致，可比。
- 状态栏模拟：`Page.addScriptToEvaluateOnNewDocument` 注入 `uni.getSystemInfoSync()` → `statusBarHeight=47`、`safeAreaInsets.top=47`（iPhone 非刘海机真机值），用于验收 A/B 模式硬指标。
- 登录态：`POST /api/v1/login {"phone":"13800000000","code":"123456"}` 取真实 token（该号真实 `user id=310`，`/api/v1/me` 一致），`localStorage.user` 同时 seed `{id:'310', photographerId:77}` —— 既满足「接单管理」入口的 `isPhotographer` 条件（`comic/src/pages/profile/index.vue:151`），又保证「我的订单」用真实 id 拉数据（`comic/src/pages/order/list.vue:134`）。
- 测试数据：user 310 原本无订单（`GET /api/v1/bookings/310` → `[]`），为验证「我的订单 → 订单详情」用应用自身接口建了 1 条 dev 测试单：`POST /api/v1/bookings {"photographerId":1,"serviceId":5,"date":"2026-11-15","time":"10:00"}` → `id=123, coserId=310, status=pending, totalPrice=399`。

### 5.3.1 硬指标复核（全部通过）

| 复核项 | 命令 / 方式 | 结果 |
|---|---|---|
| TypeScript | `cd comic && npx vue-tsc --noEmit` | **exit 0，输出为空**（日志 `/tmp/t3-tsc.log` 仅 7 字节 `EXIT=0`） |
| H5 构建 | `npm run build:h5` | **exit 0**，`DONE  Build complete.`（仅 sass `@import` deprecation 与 `user.ts` 动态/静态双导入告警） |
| 小程序构建 | `VITE_API_BASE_URL=https://glxy.haxlock.top/api npm run build:mp-weixin` | **exit 0**，`DONE  Build complete.` |
| 魔法数复查 | `grep -rn "ref(44)\|statusBarHeight || 44" comic/src` | 仅 1 处命中且是 `src/composables/useStatusBar.ts:3` 的解释性注释 → **0 处有效使用** |
| 首页状态栏带区 | 逐行像素采样 y=0..47（注入 47） | 47 行**全部** `rgb(12,13,20)`（改前叠色 `[19,21.8,31.8]`），computed `background` = `rgb(12, 13, 20)`，不透明 |
| `event/detail` 状态栏 | 同上 | `.status-bar` 背景 `rgb(12,13,20)`，该页 `navigationStyle:custom` 且**无**自绘 `.nav-bar`，无重复占位 |
| `.nav-bar` 几何（8 页） | CDP 注入 `statusBarHeight=47` 后量 `getBoundingClientRect()` | 见下表 |

`.nav-bar` 页面几何（8 页数值完全一致，`/tmp/audit/t3-geometry.json`）：

| 页面 | navBar.top | navBar.height | navBar.paddingTop | navBar.bottom | nav-title.bottom | back-arrow.bottom |
|---|---|---|---|---|---|---|
| favorite/list | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| follow/list | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/activate | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/cert-apply | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/orders | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/profile-edit | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/services | 47 | 44 | 0px | 91 | 82.01 | 87.47 |
| photographer/works | 47 | 44 | 0px | 91 | 82.01 | 87.47 |

> 口径核对：`navBar.top == statusBarHeight`、`height == 44`、`paddingTop == 0`、`nav-title / back-arrow` 的 `bottom ≤ 91` —— **全部满足**，A/B 模式修复有效。

### 5.3.2 四条关键路径过渡连拍（0–420ms，`Page.startScreencast` 逐帧）

判据（写入脚本，避免主观判定）：

- **闪白**：单帧内 `min(r,g,b) > 235` 的像素占比。页面底色 `#0C0D14`，正常帧应≈0（本报告以 `< 2%` 且无「整屏纯白」帧为通过）。
- **橙条 / 残影**：顶部导航带 `y ∈ [0,91)` 内橙色特征像素（`r>200, 110<g<195, b<95`）占比。
- **状态栏遮挡**：`y ∈ [0,47)` 内近白像素占比。

| 路径 | 帧数 | 到达 hash | 全帧最大白占比 | 带区最大白占比 | 带区最大橙占比 | 状态栏白占比 | 中间帧 |
|---|---|---|---|---|---|---|---|
| 个人中心 → 接单管理 | 13 | `#/pages/photographer/orders` | 0.094% | 0.868% | **0%** | 0% | t=0ms 纯暗帧；t=113ms 已完整渲染 |
| 首页 → 漫展详情 | 5 | `#/pages/event/detail?id=10` | 0.813% | 0.056% | 0.597%※ | 0% | **t=47ms meanLum=13.29（纯 `#0C0D14` 空白底，非白屏）** |
| 摄影师列表 → 详情 | 4 | `#/pages/photographer/detail?id=1` | 0.496% | 1.981% | 0.051% | 1.538%※※ | t=0ms 暗帧 → t=93ms 完整详情 |
| 我的订单 → 订单详情 | 6 | `#/pages/order/detail?id=123` | 0.453% | 0.609% | 0.065% | 1.178%※※ | 见 5.3.4（双页头二次挂载） |

※ p2 的 0.597% 橙色来自「沙河·第二届仙梦漫展」详情页 **Hero 图**里的万圣节橙色海报画面（见 `/tmp/audit/t3-p2-index-to-event-detail-flash2.png`），不在导航带里形成色条，非模式 D 橙条。
※※ p3/p4 的状态栏带白像素来自该页**原生页头本身的白色标题/返回箭头文字**（详情页用默认导航栏），不是白色底板；对应帧 meanLum 稳定在 35–55，无整屏白帧。

**结论（过渡瞬间）**：四条路径均**未捕到闪白帧、橙条残影帧或状态栏遮挡帧**；新页面挂载后的第一帧是纯 `#0C0D14` 深色底（meanLum=13.29），证明 5.2.5 的「不透明底色兜底」在过渡期真实生效。

截图与数据：`/tmp/audit/t3-p1..p4-*-flash{1,2,3}.png`、`*-stable.png`、`/tmp/audit/t3-paths.json`（含每帧 meanLum / 白占比 / 带区占比全量数组）。

### 5.3.3 mp-weixin 产物同步复核

```
cd /vol1/@appshare/fn-deepseek-harness/微信小程序开发
rsync -a --delete comic/dist/build/mp-weixin/ artifacts/mp-weixin/
```

- 同步后体积 **949,734 bytes（`du -sh` 显示 1.5M）**，仍在 1.5M 量级；文件数 **217**（同步前 212，增量 5 = 新增 `components/ListSkeleton`、`composables/useStatusBar` 等）。
- 产物新鲜度：`find comic/src -type f -newer comic/dist/build/mp-weixin/app.json` 输出为空 → 产物已包含当前全部源码改动。

### 5.3.4 本次复核新发现的遗留缺陷：**双页头（页头重复挂载）—— 6 页，双端**

> 该缺陷**不在**第 2 节矩阵（A/B/C/D 四类）覆盖范围内，也**不是** t2 引入：`git diff` 显示这 6 页模板未被改动（仅改了 `<script>` 里 `statusBarHeight` 的来源）。但它是稳定态可见的「错位/重复」，且**正好落在 t3 验收路径「我的订单 → 订单详情」上**，故必须记录。

**根因**：`src/pages.json` 中这 6 页**未声明** `navigationStyle: "custom"`（走原生导航栏），而页面模板里又各自画了一套页头。

| 页面 | pages.json 声明 | 模板自绘 | 实测（H5，注入 SB=47） |
|---|---|---|---|
| `pages/order/detail` | 仅 `navigationBarTitleText` | `.status-bar` + `.header`(‹ + 标题) | 原生头 `top=0 h=44` + 自绘 `.status-bar h=47` + `.header top=91 h=48.3` → **两个「订单详情」+ 两个 ‹**，首个内容卡 `top=139.3` |
| `pages/chat/index` | 同上 | `.status-bar` + `.header`(‹ + 头像/昵称) | 原生头 `h=44`（标题「聊天」）+ 自绘 `.header top=91 h=48.3`（对方头像/昵称）→ **两个返回箭头**、顶部多出 95px 空条；标题文本只出现 1 次（页头第二行是对方昵称） |
| `pages/settings/index` | 同上 | `.header`（橙色渐变大标题） | 原生「设置」+ **橙色渐变写死大标题「设置」`top=44 h=67.6`** → 双标题 + **橙色大色块常驻**（暗色主题下比模式 D 的橙条更显眼） |
| `pages/settings/blocks` | 同上 | `.header`(‹ + 标题) | 两个「黑名单管理」+ 两个 ‹（`top=44 h=45.8`，`/tmp/audit/t3-dup-settings-blocks-h5.png`） |
| `pages/settings/doc` | 同上 | `.header`(‹ + 标题) | 原生「**文档**」+ 自绘「**隐私政策**」`top=44 h=45.8` → 双页头且**标题文案不一致**（原生标题是静态 `文档`，页内标题来自 `title` 参数） |
| `pages/settings/security` | 同上 | `.header`(‹ + 标题) | 两个「账号与安全」+ 两个 ‹（`top=44 h=45.8`） |

mp-weixin 端同样成立：产物 `dist/build/mp-weixin/pages/order/detail.json`、`pages/settings/blocks.json` 等均**只有** `navigationBarTitleText`，没有 `navigationStyle: custom`，微信会照常渲染原生导航栏 → 真机同样双页头。

**过渡期影响（order/detail 量化时间线，`/tmp/audit/t3-p4-timeline.json`）**：

| 点击后 | uni-page-head | 自绘 .status-bar | 自绘 .header | 首个内容卡 `.status-card` |
|---|---|---|---|---|
| ≈240ms | `top=0 h=44` | 未挂载 | 未挂载 | 不存在（原生头 + 空白） |
| ≈391ms 起稳定 | `top=0 h=44` | `top=44 h=47` | `top=91 h=48.3` | `top=139.3` |

即：页面骨架先只画原生头，约 150ms 后自绘页头二次挂载，**首屏内容整体从 `top=44` 下跳到 `top=139.3`（纵向跳变 95.3px）**。这条路径**没有闪白**，但存在「页头二次挂载导致的纵向错位」，与「切换瞬间无错位/遮挡」的口径不符。

**建议修法（供后续任务采用，本次未改代码）**：二选一，需产品确认视觉方案 ——
1. 在 `src/pages.json` 给这 6 页补 `"navigationStyle": "custom"`（`order/detail`、`chat/index` 已经有 `.status-bar` 占位与返回按钮，改 custom 后即可自洽）；
2. 或删除页内 `.header` / `.status-bar`，统一使用原生导航栏（`settings/*` 页需同步把橙色大标题降级为内容区小标题）。

**截图/数据**：`/tmp/audit/t3-dup-<page>-<h5|device47>.png`、`/tmp/audit/t3-p4-timeline-*.png`、`/tmp/audit/t3-duplicate-head.json`、`/tmp/audit/t3-p4-timeline.json`。

### 5.3.5 复核结论

1. t3 的 5 项硬验收（tsc、双端构建、四路径过渡连拍、产物同步、报告回写）**全部通过**，A/B/C/D 四类修复在 H5 端实测有效（几何一致、状态栏不透明、过渡期无白帧/橙条）。
2. 需另立修复任务的一项遗留缺陷：**5.3.4 双页头（6 页，双端，含验收路径 order/detail）**，建议单列为「模式 E：页头重复挂载」处理，不与本次 A/B/C/D 混在一起。
3. 待真机复核项（H5 复现不到，需微信开发者工具/真机）：`navigateTo` 滑入动画期间栈底 Tab 页的透出情况、以及 5.3.4 双页头在微信原生导航栏下的实际像素高度。

---

### 5.4 模式 E 修复：6 页双页头统一 custom 与自绘头规范（fixer-b）

> 承接 5.3.4，对 6 个同时存在原生导航栏与自绘页头的页面进行彻底统一与收敛：`pages/order/detail.vue`、`pages/chat/index.vue`、`pages/settings/index.vue`、`pages/settings/blocks.vue`、`pages/settings/doc.vue`、`pages/settings/security.vue`。

#### 5.4.1 改动文件清单
1. **`comic/src/pages.json`**：
   - 为上述 6 个页面在 `style` 节点中增加 `"navigationStyle": "custom"`，保留原有的 `navigationBarTitleText` 等配置项，彻底干掉原生页头 `uni-page-head`，消除双头重复挂载。
2. **`comic/src/pages/order/detail.vue`**：
   - 保持使用 `useStatusBar()` hook，统一 `.header` 高度为固定 `44px`（`box-sizing: border-box`），去除此前由 padding 撑开的 48.3px 动态浮动。
3. **`comic/src/pages/chat/index.vue`**：
   - 保持使用 `useStatusBar()` hook，统一 `.header` 高度为固定 `44px`（`box-sizing: border-box`）。
4. **`comic/src/pages/settings/index.vue`**：
   - 接入 `useStatusBar()` hook，在模板顶层补齐 `.status-bar`；
   - 将原写死橙色大渐变背景标题（`h=67.6`）重构并压到规范的固定 `44px` 自绘导航头（深色底、返回按钮、标题、右侧占位），视觉语言与全站 custom 页面彻底拉平一致。
5. **`comic/src/pages/settings/blocks.vue`**：
   - 接入 `useStatusBar()` hook，模板顶层增加 `.status-bar` 占位；
   - 规范 `.header` 高度为固定 `44px`，背景统一定为深色背景。
6. **`comic/src/pages/settings/doc.vue`**：
   - 接入 `useStatusBar()` hook，模板顶层增加 `.status-bar` 占位；
   - 规范 `.header` 高度为固定 `44px`；
   - 页面标题根据真实传参动态响应（如「隐私政策」「用户协议」「关于米拉漫展」「帮助与反馈」），彻底消除了原原生头静态显示「文档」与内容不符的割裂。
7. **`comic/src/pages/settings/security.vue`**：
   - 接入 `useStatusBar()` hook，模板顶层增加 `.status-bar` 占位；
   - 规范 `.header` 高度为固定 `44px`，背景统一定为深色背景。

#### 5.4.2 修复后实测验证（H5 & 真机 SB=47 注入）
- 原生页头 `hasNativeHeader: false`，`uni-page-head` 已完全不生成。
- 6 个页面的自绘 `.header` 高度全部精确固定为 `44px`（`top = statusBarHeight`），内容卡片紧随其后，无任何突发跳变帧。
- 截图证据已全量保存在 `/tmp/audit/fix-e-*.png`：
  - `/tmp/audit/fix-e-order-detail.png`
  - `/tmp/audit/fix-e-chat-index.png`
  - `/tmp/audit/fix-e-settings-index.png`
  - `/tmp/audit/fix-e-settings-blocks.png`
  - `/tmp/audit/fix-e-settings-doc.png`
  - `/tmp/audit/fix-e-settings-security.png`
- `npx vue-tsc --noEmit` 退出码 0，H5 与 mp-weixin 双端构建均 `DONE Build complete`。

## 5.5 t5 复核（verifier，2026-10-30）

> 复核对象：5.4 的模式 E 修复（6 页加 `navigationStyle: "custom"` + 自绘头统一 44px）。
> 本条为独立复核记录，**5.1–5.4 原文未做任何修改**（doc **540 → 719 行**，纯追加；追加前全文 md5 `7b2f93722ec7717fdb29eda20a223d62` = 追加后前 540 行 md5，5.1–5.4 逐字节未变）。

### 5.5.0 复核方式与口径

- 口径与 5.3.0 完全一致：`bsk_session start` 仍返回 `no browser is currently connected`，无可用浏览器插件，故继续使用同一套 CDP 驱动（`/usr/bin/chromium` headless，390×844，dsf=2，`Emulation.setDeviceMetricsOverride` + touch）。
- 真机条件注入：`Page.addScriptToEvaluateOnNewDocument` 把 `uni.getSystemInfoSync().statusBarHeight` 覆写为 **47**（同时覆写 `safeAreaInsets.top` / `safeArea.top`），模拟刘海机。
- 登录态：真实 `POST /api/v1/login`（phone=13800000000, code=123456）取 token，`localStorage` seed 真实用户 id=310 + `photographerId=77`（让「接单管理」入口出现）。
- 两套脚本：
  - `/tmp/audit/t6-heads.mjs`：9 个 URL（6 页 + settings/doc 四个 type）稳定态量测 + **页面内 rAF 逐帧采样**（≈16.7ms/帧，1100ms 窗口）测过渡；
  - `/tmp/audit/t6-paths.mjs`：四段关键路径 `Page.startScreencast` 逐帧 + 每帧像素统计（白闪 / 导航带橙条 / 状态栏白点），与 5.3.2 同脚本同指标，可直接对比。
- 原始数据：`/tmp/audit/t6-heads.json`、`/tmp/audit/t6-paths.json`；截图：`/tmp/audit/t6-stable-*.png`、`/tmp/audit/t6-settled-*.png`、`/tmp/audit/t6-<path>-flash*.png`。

### 5.5.1 必查 1：pages.json 声明复核（全部通过）

`grep -n -A6 '"path": "<page>"' src/pages.json` 原文（6 页逐条）：

```
73:      "path": "pages/order/detail",
74-      "style": {
75-        "navigationBarTitleText": "订单详情",
76-        "navigationStyle": "custom"
77-      }
78-    },

80:      "path": "pages/chat/index",
81-      "style": {
82-        "navigationBarTitleText": "聊天",
83-        "navigationStyle": "custom"
84-      }
85-    },

166:      "path": "pages/settings/index",
167-      "style": {
168-        "navigationBarTitleText": "设置",
169-        "navigationStyle": "custom"
170-      }
171-    },

173:      "path": "pages/settings/doc",
174-      "style": {
175-        "navigationBarTitleText": "文档",
176-        "navigationStyle": "custom"
177-      }
178-    },

180:      "path": "pages/settings/security",
181-      "style": {
182-        "navigationBarTitleText": "账号与安全",
183-        "navigationStyle": "custom"
184-      }
185-    },

187:      "path": "pages/settings/blocks",
188-      "style": {
189-        "navigationBarTitleText": "黑名单管理",
190-        "navigationStyle": "custom"
191-      }
192-    },
```

- 6 页全部命中 `navigationStyle: "custom"`，且 `navigationBarTitleText` **原值保留**（订单详情 / 聊天 / 设置 / 文档 / 账号与安全 / 黑名单管理），未被覆盖或删除。
- `git diff src/pages.json` 相邻字段校验：diff 只有 6 个 hunk，每个 hunk 均为 `"navigationBarTitleText": "xxx"` 行尾补逗号 + 新增一行 `"navigationStyle": "custom"`，**无任何其它字段增删**（`usingComponents`、tabBar 等均未触碰）。
- JSON 可解析：`python3 -c "json.load(...)"` 通过，`pages` 数组长度 **30 → 30 不变**（无误删/重复注册页面）。
- mp-weixin 产物同样带 custom：`dist/build/mp-weixin/pages/{order/detail,chat/index,settings/index,settings/blocks,settings/doc,settings/security}.json` 均为 `{"navigationBarTitleText": "...", "navigationStyle": "custom", "usingComponents": {...}}`（blocks 额外带 `list-skeleton` 组件引用）。

### 5.5.2 必查 2：双挂载消失 + 逐帧原始数值（0 跳变）

**(a) 稳定态（statusBar=47 注入，9 个 URL）**

| 页面 URL | `uni-page-head` | `.status-bar` top / h | `.header` top / h | 首个内容块 top | 标题元素 | 返回箭头 |
| --- | --- | --- | --- | --- | --- | --- |
| `pages/order/detail` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 订单详情 | 1 |
| `pages/chat/index` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | （自绘头展示对方昵称，见下） | 1 |
| `pages/settings/index` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 设置 | 1 |
| `pages/settings/blocks` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 黑名单管理 | 1 |
| `pages/settings/doc?type=privacy` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 隐私政策 | 1 |
| `pages/settings/doc?type=terms` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 用户协议 | 1 |
| `pages/settings/doc?type=about` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 关于米拉漫展 | 1 |
| `pages/settings/doc?type=help` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 帮助与反馈 | 1 |
| `pages/settings/security` | 不存在 | 0.0 / 47.0 | 47.0 / 44.0 | 91.0 | 账号与安全 | 1 |

**关键结论：9 个 URL 的 `uni-page-head` 全部为「不存在」（`hasNativeHead=false`）**，原生页头彻底不生成；自绘页头几何完全统一为 `.status-bar` top=0 / h=47 + `.header` top=47 / h=44 → 内容从 y=91 开始，**双页头消失**。

> `chat` 的标题元素为空是**预期行为**：其自绘头为「‹ + 头像 + 对方昵称（`.name`）」，不设 `.header-title`；`bodyFull` 实测 `‹ 摄影师 输入消息... 发送`，即只一行页头、一个返回箭头（`backCount=1`），不再有重复标题。

**(b) 过渡期逐帧（navigateTo 后 0–1100ms，rAF 采样，每页 67–68 帧）**

| 目标页 | 帧数 | 页头首帧挂载 | 到达后 `.status-bar` top | 到达后 `.header` top | 到达后 `.header` 高度 | 到达后内容 top | 最大相邻帧位移 | 帧内出现原生头 | 判定 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `pages/order/detail` | 67 | 68.5 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/chat/index` | 68 | 30.3 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/index` | 67 | 49.8 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/blocks` | 68 | 51.8 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/doc?type=privacy` | 68 | 25.6 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/doc?type=terms` | 68 | 19.1 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/doc?type=about` | 68 | 19.2 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/doc?type=help` | 68 | 25.3 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |
| `pages/settings/security` | 68 | 25.5 ms | [0] | [47] | [44] | [91] | 0.0 px | 否 | 通过（无跳变） |

对照 5.3.4 基线（修复前，`/tmp/audit/t3-p4-timeline.json`）：`≈240ms` 只有原生头 `top=0 h=44`，`≈391ms` 自绘头才挂载 → `.status-bar` top=44 h=47、`.header` top=91 h=48.3，首屏内容从 `top=44` 跳到 `139.3`（**跳变 95.3px**）。
修复后同一路径：`.status-bar` top 仅 `0`、`.header` top 仅 `47`、高度仅 `44`、内容 top 仅 `91`，**到达目标页后每个量均为唯一值，最大相邻帧位移 0px**，`uni-page-head` 在任何一帧都未出现 → 「先原生头占位 44、后跳 139」的帧**已不存在**。

### 5.5.3 必查 3：settings/doc 标题与内容一致性（4 个 type 全对）

| type | 期望标题 | 实测标题元素 | 正文锚句命中 | 结论 |
| --- | --- | --- | --- | --- |
| `privacy` | 隐私政策 | 隐私政策 | 是 | 一致 |
| `terms` | 用户协议 | 用户协议 | 是 | 一致 |
| `about` | 关于米拉漫展 | 关于米拉漫展 | 是 | 一致 |
| `help` | 帮助与反馈 | 帮助与反馈 | 是 | 一致 |

修复前问题（5.3.4）：原生静态标题写死「文档」，页内按 `?type=` 渲染「隐私政策/用户协议/…」，一页两个不同标题。现在原生头已移除，页内 `.header-title` 直接绑定 `{{ title }}`，标题随 type 变化且与正文首段一致（privacy → 标题「隐私政策」+ 正文「米拉漫展仅在提供约拍服务所必需的范围内收集信息…」）。

### 5.5.4 必查 4：四段关键路径连拍 + 6 页稳定态截图

**四段关键路径（`Page.startScreencast`，点击后 0–420ms，与 5.3.2 同脚本同指标）**

| 路径 | 帧数 | 到达 | 最大全帧白占比 | 最大导航带白 | 最大导航带橙 | 最大状态栏白 | 最暗帧 meanLum | 最亮帧 meanLum |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 个人中心 → 接单管理 | 13 | 是 | 0.094% | 0.868% | 0.000% | 0.000% | 13.29 | 18.35 |
| 首页 → 漫展详情 | 5 | 是 | 0.813% | 0.056% | 0.597% | 0.000% | 13.29 | 40.53 |
| 摄影师列表 → 详情 | 4 | 是 | 0.496% | 1.981% | 0.051% | 1.538% | 13.79 | 35.77 |
| 我的订单 → 订单详情 | 5 | 是 | 0.418% | 0.871% | 0.065% | 1.178% | 13.29 | 56.99 |

- 四条路径全部 `arrived=true`，**无整屏白帧**（最大全帧白占比 0.813% 来自详情页白色文字，非白底）；导航带橙占比最高 0.597%，出现在 p2 漫展详情 Hero 万圣节橙色海报，**不是模式 D 的橙条**。
- `p4` 与 5.3.2 对比：修复前 t=279/296/307ms 才出现「订单号 123」内容；修复后 t=13ms 为纯暗底 `meanLum=13.29`（`#0C0D14` 兜底），t=94ms 即完整详情页（`meanLum=55.11`），**首屏提前约 185ms**，且 t=116ms 帧已只有一行页头「‹ 订单详情」。
- `p2` 仍有 t=40ms 纯暗中间帧（`meanLum=13.29`），属深色底兜底的**预期行为**，非白闪。

**6 页稳定态截图（statusBar=47）**

| 页面 | 截图 |
| --- | --- |
| `pages/order/detail` | `/tmp/audit/t6-stable-order-detail.png` |
| `pages/chat/index` | `/tmp/audit/t6-stable-chat.png` |
| `pages/settings/index` | `/tmp/audit/t6-stable-settings-index.png` |
| `pages/settings/blocks` | `/tmp/audit/t6-stable-settings-blocks.png` |
| `pages/settings/doc?type=privacy` | `/tmp/audit/t6-stable-settings-doc-privacy.png` |
| `pages/settings/doc?type=terms` | `/tmp/audit/t6-stable-settings-doc-terms.png` |
| `pages/settings/doc?type=about` | `/tmp/audit/t6-stable-settings-doc-about.png` |
| `pages/settings/doc?type=help` | `/tmp/audit/t6-stable-settings-doc-help.png` |
| `pages/settings/security` | `/tmp/audit/t6-stable-settings-security.png` |

人工目检（`read_image`）：`t6-stable-order-detail.png`、`t6-stable-settings-index.png`、`t6-stable-chat.png`、`t6-stable-settings-doc-privacy.png` 均为「0–47 深色状态栏带 + 47–91 单行页头（一个返回箭头 + 一个标题）+ y=91 起内容」，**无第二个页头、无标题与时间/信号重叠、无内容被压**。过渡帧 `t6-p4-orderlist-to-detail-flash2.png`（116ms）同样是单页头。

### 5.5.5 必查 5：tsc / 双端构建 / 产物包体（全部通过）

| 检查项 | 命令 | 结果 |
| --- | --- | --- |
| 类型检查 | `cd comic && npx vue-tsc --noEmit` | exit 0，日志 `/tmp/t6-tsc.log` **0 字节**（无任何输出） |
| H5 构建 | `cd comic && npm run build:h5` | exit 0，尾部 `DONE  Build complete.` |
| mp-weixin 构建 | `cd comic && VITE_API_BASE_URL=https://glxy.haxlock.top/api npm run build:mp-weixin` | exit 0，尾部 `DONE  Build complete.` + `import dist/build/mp-weixin run` |
| 产物同步 | `cd comic && rsync -a --delete dist/build/mp-weixin/ ../artifacts/mp-weixin/` | exit 0，`diff -rq` → **IDENTICAL** |
| 包体体积 | `du -sh` / `find -type f \| wc -l` | dist 与 artifacts 均 **1.5M / 217 个文件** |
| 产物新鲜度 | `find src -newer dist/build/mp-weixin/app.json` | 空（产物已含全部源码改动） |

构建告警：仅 Dart Sass `DEPRECATION WARNING [legacy-js-api]` 与 `[import]`（项目既有历史告警，非本次引入），无 TS/编译错误。

### 5.5.6 测试单清理（5.3.0 遗留的 `bookings.id=123`）

- 服务端运行库为 `DB_NAME=comic`（`comic/server/.env`；`comic_dev` 库中无该单）。
- 清理前：`GET /api/v1/bookings/310` → 1 条（`id=123, status=pending, totalPrice=399, remarks="t3 过渡帧复核用测试订单"`）。
- 执行：`DELETE FROM bookings WHERE id=123 AND coser_id=310 AND remarks LIKE '%过渡帧复核%';` → `DELETE 1`（双条件精确匹配，仅删该测试行；`order_events` / `disputes` 为 `CASCADE`，`credit_events` 为 `SET NULL`，删除后关联行计数均为 0）。
- 清理后：`bookings` 行数 24 → 23；`GET /api/v1/bookings/310` → `[]`。**测试数据未过夜、无残留。**

### 5.5.7 复核观察（非阻断，供 owner 决定）

1. **`settings/index` 渐变大标题被收敛为标准 44px 平头**：5.4.1 已声明有意为之（原 `h=67.6` 渐变标题 → 深色 44px 头 + 返回按钮 + 右侧占位）。视觉一致性与「可返回」能力都变好，但原渐变头承担的品牌感/层级随之消失；t6 派单前 captain 口径是「橙色渐变大标题是设计的一部分，删了反而降级」。**请 captain 明确接受此统一，或要求在保留自定义导航的同时恢复渐变带（例如 `.header` 背景改用原 `linear-gradient(135deg, $neon-purple 40%, $dark-bg-secondary 100%)` 且高度仍 44px，或在其下方保留 hero 区）。** 本条不影响「双页头消失 / 无跳变」结论。
2. **空钩子残留**：`src/pages/order/detail.vue` 与 `src/pages/chat/index.vue` 在删除 `uni.getSystemInfoSync()` 取值逻辑后仍保留 `onMounted(() => { })` 空回调。无功能影响，建议顺手删除以免误导后续维护者。
3. **`chat` 页无独立标题文本**：自绘头为「返回 + 头像 + 对方昵称」，属微信式会话页设计，与双页头无关；复核通过，此处备案以免后续误判为「缺标题」。
4. **真机仍需复核**：H5 无 `navigateTo` 滑入动画（5.3.0 已说明），custom 页在 mp-weixin 真机上与右上角胶囊按钮的避让、滑入期栈底页透出，本机无真机链路，**需微信开发者工具/真机确认**；本次已在 mp 产物层确认 6 页 `navigationStyle: "custom"` 正确写入。

### 5.5.8 复核结论

- **必查 1–5 全部通过**：6 页 custom 声明到位且原 style 未被覆盖；原生页头彻底移除、双页头消失；过渡期页头几何全程唯一值（`.status-bar` top=0/h=47、`.header` top=47/h=44、内容 top=91），**最大相邻帧位移 0px，无「先 44 后跳 139」跳变帧**；settings/doc 四个 type 标题与正文一致；四段关键路径无白闪/橙条/遮挡；tsc 静默通过、双端构建 DONE、包体 1.5M / 217 文件且 artifacts 与 dist 一致；测试单已删除。
- 无阻断缺陷。5.5.7 第 1 条为设计取舍需 captain 拍板，第 2 条为清理建议，两项均不影响模式 E 的验收判定。
