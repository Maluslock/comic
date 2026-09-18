# 交易闭环补全：收藏 + 真实聊天 — 设计文档

**日期**: 2026-08-27
**状态**: 设计定稿（用户确认方案后开始执行）
**范围**: 米拉漫展小程序 coser 侧闭环补全——收藏功能（后端化 + 我的收藏页）+ 真实聊天（会话/消息表 + API + 前端接真实数据）

## 1. 背景与问题

视觉审阅（agent-browser 全站截图）确认的缺口：

1. **收藏是假的**：
   - `src/pages/photographer/detail.vue:147` — `collected = ref(false)` 纯本地状态，`collect()` 只翻转本地 boolean + toast，无持久化
   - `src/pages/profile/index.vue:153` — `goFavorites()` 只弹 toast，无"我的收藏"页面
   - 后端无 favorites 表/API
2. **聊天是假的**：
   - `src/pages/chat/index.vue:93` — `messages` 为本地自定义 `ChatMessage[]`，`pickReply()` 关键词自动回复（纯前端 demo）
   - `src/pages/message/index.vue:58` — 通知/会话硬编码数组
   - `src/stores/chat.ts` — 有 store 骨架但从未被消费（AGENTS.md 标注 "NOT consumed"）
   - 后端无会话/消息表/API

## 2. 目标

1. 收藏功能真实化：收藏/取消收藏持久化到后端，新增"我的收藏"列表页
2. 聊天真实化：会话与消息落库，chat 页发送/接收真实消息，消息页会话列表接真实数据（demo 标注保留用于尚未接真实通知的部分）

**明确不做**（YAGNI）：WebSocket 实时推送（轮询/刷新拉取，demo 够用）；微信订阅消息真推送；聊天图片发送（仅 text）。

## 3. 架构决策

### 3.1 收藏（photographer favorites）

**后端**:
- 迁移 `000008_photographer_favorites.up.sql`: `photographer_favorites` 表（id BIGSERIAL PK, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, photographer_id INTEGER NOT NULL REFERENCES photographers(id) ON DELETE CASCADE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(user_id, photographer_id)）
- API：
  - `POST /api/v1/favorites` body `{userId, photographerId}` → 201（幂等，重复收藏返回 200）
  - `DELETE /api/v1/favorites/:userId/:photographerId` → 204
  - `GET /api/v1/favorites/:userId` → `[{photographerId, name, avatar, location, rating, addedAt}]`（JOIN photographers）
  - 校验：photographer 存在（404），user 存在（404）

**前端**:
- `photographer/detail.vue`：`collect()` 改调 `apiPost('/v1/favorites')` / `apiDelete('/v1/favorites/:userId/:photographerId')`；onLoad 时若已登录调 `GET /favorites/:userId` 初始化 `collected`（需从列表查 or 新增单查接口——demo 用列表查 contains 简化）；未登录点收藏 → 跳登录
- 新增 `src/pages/favorite/list.vue`：收藏摄影师列表页（复用 PhotographerCard 样式），空状态 + 去逛逛；注册到 pages.json（`pages/favorite/list`）
- `profile/index.vue`：`goFavorites()` 改 `uni.navigateTo('/pages/favorite/list')`

### 3.2 聊天（chat sessions + messages）

**后端**:
- 迁移 `000008` 同一文件追加（或单独 000008_chat.up.sql，选用单独文件保持单一职责）：`chat_sessions`（id BIGSERIAL PK, user1_id, user2_id, created_at, updated_at, UNIQUE(user1_id, user2_id) 规范化 user1<user2）、`chat_messages`（id BIGSERIAL PK, session_id BIGINT REFERENCES chat_sessions(id) ON DELETE CASCADE, sender_id BIGINT, content TEXT, created_at）
- API：
  - `GET /api/v1/chat/sessions/:userId` → 会话列表（JOIN 对方 user + 最后一条消息 + 未读数；未读数先置 0，demo 简化）
  - `POST /api/v1/chat/sessions` body `{userId, otherUserId}` → 201（upsert，规范化 user1<user2）返回 sessionId
  - `GET /api/v1/chat/messages/:sessionId` → 消息列表（按 created_at ASC）
  - `POST /api/v1/chat/messages` body `{sessionId, senderId, content}` → 201

**前端**:
- `chat/index.vue`：onLoad 读路由参数 `sessionId`（摄影师详情"聊天"按钮 → 先 `POST /chat/sessions` 拿/建 session，navigateTo chat?sessionId=xxx）；`messages` 从 `GET /chat/messages/:sessionId` 加载；`sendMessage` 改 `POST /chat/messages` 后追加本地；移除 `pickReply` 自动回复（保留占位提示"对方未在线，消息已发送"）
  - 需展示对方昵称/头像：路由传 `peerName/peerAvatar` 或 GET session 详情——demo 用路由参数传最简
- `message/index.vue`：会话列表改调 `GET /chat/sessions/:userId`（登录用户）；未登录显示空 + 引导登录；通知部分保留 demo 标注（通知表接入属 P3）
- `stores/chat.ts`：现有骨架适配（或页面直接调 API 不经 store——demo 选直接 API，减少复杂度）

### 3.3 Demo 简化决策

- 聊天无真实对端自动应答：发送后仅落库+显示"已发送"，对方登录即看到——demo 阶段双端手动测
- 未读数恒 0
- 收藏状态未登录时点击 → 跳登录（复用 requireAuth 模式）

## 4. 数据流

```
收藏:
  detail 页点击 → apiPost/DELETE /favorites → 刷新 collected → toast
  profile 我的收藏 → GET /favorites/:userId → 列表渲染

聊天:
  摄影师详情"聊天" → POST /chat/sessions → sessionId → navigateTo chat?sessionId
  chat 页 onLoad → GET /chat/messages/:sessionId → 渲染
  发送 → POST /chat/messages → 追加本地消息
  消息页 → GET /chat/sessions/:userId → 会话列表
```

## 5. 错误处理

- 401 未登录 → client.ts 统一跳登录
- 404（摄影师/用户不存在）→ toast
- 会话不存在 → toast + 返回
- 网络失败 → toast

## 6. 测试策略

- 后端: Go 测试——favorites upsert 幂等、DELETE 幂等、sessions upsert 幂等（规范化 user1<user2）、messages 追加顺序
- 前端: vue-tsc 类型检查 + Playwright H5 实测——收藏→我的收藏页显示；聊天发送→刷新消息仍在
- 回归: 全站主要页面截图对比（视觉审阅），go test 全绿

## 7. 里程碑

- M1: 后端迁移 + favorites API + chat API（session/messages）
- M2: 前端收藏（detail + 新列表页 + profile 入口）
- M3: 前端聊天（chat 页真实化 + 消息页会话真实化）
- M4: 全链路验证（Playwright 收藏/聊天实测）+ AGENTS.md 更新 + 推送

## 8. 假设记录

1. 用户已确认批 1 方案（聊天 + 收藏），视觉发现问题自行列 task 纠正（用户授权）
2. 无 WebSocket，轮询/刷新拉取
3. 未读数恒 0，无消息推送（P3 接通知表）
4. 收藏列表页新增页面，注册 pages.json
5. 每次里程碑提交推送 Gitee
