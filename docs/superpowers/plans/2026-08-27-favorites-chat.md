# 交易闭环补全：收藏 + 真实聊天 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 收藏功能真实化（后端化 + 我的收藏列表页）+ 聊天真实化（会话/消息落库 + 前端接真实 API）。

**Architecture:** 后端加两张表（photographer_favorites、chat_sessions/chat_messages）+ REST API，沿用 follows 的 repository/service/handler 模式；前端收藏按钮/聊天发送/会话列表改调真实 API，移除本地状态与自动回复。

**Tech Stack:** Go 1.22 + Gin + pgx（server/）；Vue 3 script setup + uni-app（src/）。

## Global Constraints

- Go 构建: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH`
- PG: `postgres://comic:comic123@127.0.0.1:5433/comic`；重启 API: `kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')` + `(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &)`；**禁用 pkill -f**
- TypeScript strict，禁新增 `as any`；SCSS 变量/rpx/`@/` 别名；无 emoji 图标
- 表命名: `photographer_favorites`、`chat_sessions`、`chat_messages`；迁移文件 `000008_favorites_chat.up.sql`/`.down.sql`
- 收藏幂等: UNIQUE(user_id, photographer_id)；会话幂等: UNIQUE(user1_id, user2_id) 规范化 user1<user2
- 每个任务 commit；不 push（编排者统一推 Gitee `git -c http.proxy= -c https.proxy= push gitee master`）
- 按用户授权：视觉发现问题列 task 纠正；demo 无 WebSocket（刷新拉取）、未读数恒 0
- 聊天无自动回复对端（对方登录可见），移除 pickReply/replyPool/defaultReplies

---

## 文件结构总览

| 文件 | 职责 |
|------|------|
| `server/migrations/000008_favorites_chat.{up,down}.sql` | favorites + chat_sessions + chat_messages 表 |
| `server/internal/repository/favorites.sql.go` | favorites CRUD（InsertFavorite/DeleteFavorite/ListFavorites） |
| `server/internal/repository/chat.sql.go` | chat CRUD（UpsertSession/ListSessions/ListMessages/InsertMessage） |
| `server/internal/repository/querier.go` | 注册新方法 |
| `server/internal/service/favorite_service.go` | 收藏业务（幂等/校验） |
| `server/internal/service/chat_service.go` | 会话/消息业务（规范化/幂等） |
| `server/internal/handler/favorite_handler.go` | POST/DELETE/GET favorites |
| `server/internal/handler/chat_handler.go` | GET/POST sessions + GET/POST messages |
| `server/cmd/api/main.go` | 注册路由 |
| `src/api/client.ts` | 已有 apiGet/apiPost/apiDelete/apiPut 复用 |
| `src/pages/photographer/detail.vue` | 收藏按钮真实化 |
| `src/pages/favorite/list.vue` | 新页面：我的收藏 |
| `src/pages.json` | 注册 pages/favorite/list |
| `src/pages/profile/index.vue` | goFavorites 改导航 |
| `src/pages/chat/index.vue` | 真实消息收发 |
| `src/pages/message/index.vue` | 会话列表真实化 |
| `src/pages/photographer/detail.vue` | "聊天"按钮创建/进入会话 |

---

### Task 1: 后端迁移 — favorites + chat 表

**Files:**
- Create: `server/migrations/000008_favorites_chat.up.sql`
- Create: `server/migrations/000008_favorites_chat.down.sql`

**Interfaces:**
- Produces: `photographer_favorites`(id BIGSERIAL PK, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, photographer_id INTEGER NOT NULL REFERENCES photographers(id) ON DELETE CASCADE, created_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(user_id, photographer_id))；`chat_sessions`(id BIGSERIAL PK, user1_id BIGINT NOT NULL, user2_id BIGINT NOT NULL, created_at, updated_at, UNIQUE(user1_id, user2_id))；`chat_messages`(id BIGSERIAL PK, session_id BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE, sender_id BIGINT NOT NULL, content TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())

- [ ] **Step 1: 写 up 迁移**

```sql
-- Favorites (收藏摄影师)
CREATE TABLE IF NOT EXISTS photographer_favorites (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  photographer_id INTEGER NOT NULL REFERENCES photographers(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user_id, photographer_id)
);
CREATE INDEX IF NOT EXISTS idx_favorites_user ON photographer_favorites(user_id);

-- Chat sessions (用户1/用户2，规范化 user1 < user2)
CREATE TABLE IF NOT EXISTS chat_sessions (
  id BIGSERIAL PRIMARY KEY,
  user1_id BIGINT NOT NULL,
  user2_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (user1_id, user2_id)
);

-- Chat messages
CREATE TABLE IF NOT EXISTS chat_messages (
  id BIGSERIAL PRIMARY KEY,
  session_id BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  sender_id BIGINT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON chat_messages(session_id, created_at);
```

- [ ] **Step 2: 写 down 迁移**

```sql
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
DROP TABLE IF EXISTS photographer_favorites;
```

- [ ] **Step 3: 应用 + 验证**

Run: `docker exec -i comic-pg psql -U comic -d comic < server/migrations/000008_favorites_chat.up.sql` 然后 `docker exec comic-pg psql -U comic -d comic -c "\dt"`（显示 3 新表）
Expected: 3 表创建成功

- [ ] **Step 4: Commit**

```bash
git add server/migrations/000008_favorites_chat.up.sql server/migrations/000008_favorites_chat.down.sql
git commit -m "feat(db): photographer_favorites + chat_sessions + chat_messages tables"
```

---

### Task 2: 后端 repository — favorites + chat CRUD

**Files:**
- Create: `server/internal/repository/favorites.sql.go`
- Create: `server/internal/repository/chat.sql.go`
- Modify: `server/internal/repository/querier.go`

**Interfaces:**
- Consumes: `Queries`（q.db *pgxpool.Pool）
- Produces:
  - `InsertFavorite(ctx, userID int64, photographerID int32) error`（ON CONFLICT DO NOTHING）
  - `DeleteFavorite(ctx, userID int64, photographerID int32) error`
  - `ListFavorites(ctx, userID int64) ([]FavoriteRow, error)`（JOIN photographers: id/name/avatar/location/rating/added_at）
  - `UpsertSession(ctx, u1, u2 int64) (int64, error)`（ON CONFLICT (user1_id,user2_id) DO UPDATE SET updated_at=NOW() RETURNING id）
  - `ListSessions(ctx, userID int64) ([]SessionRow, error)`（JOIN 对方用户 + 最后消息）
  - `ListMessages(ctx, sessionID int64) ([]MessageRow, error)`
  - `InsertMessage(ctx, sessionID, senderID int64, content string) (int64, error)`

- [ ] **Step 1: favorites.sql.go**

```go
package repository

import (
	"context"
)

type FavoriteRow struct {
	ID             int64  `json:"id"`
	PhotographerID int32  `json:"photographerId"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	Location       string `json:"location"`
	Rating         string `json:"rating"`
	AddedAt        string `json:"addedAt"`
}

const insertFavorite = `-- name: InsertFavorite :exec
INSERT INTO photographer_favorites (user_id, photographer_id) VALUES ($1, $2)
ON CONFLICT (user_id, photographer_id) DO NOTHING
`

func (q *Queries) InsertFavorite(ctx context.Context, userID int64, photographerID int32) error {
	_, err := q.db.Exec(ctx, insertFavorite, userID, photographerID)
	return err
}

const deleteFavorite = `-- name: DeleteFavorite :exec
DELETE FROM photographer_favorites WHERE user_id = $1 AND photographer_id = $2
`

func (q *Queries) DeleteFavorite(ctx context.Context, userID int64, photographerID int32) error {
	_, err := q.db.Exec(ctx, deleteFavorite, userID, photographerID)
	return err
}

const listFavorites = `-- name: ListFavorites :many
SELECT f.id, p.id, p.name, p.avatar, COALESCE(p.location, ''), COALESCE(p.rating::text, '0'), f.created_at::text
FROM photographer_favorites f
JOIN photographers p ON p.id = f.photographer_id
WHERE f.user_id = $1
ORDER BY f.created_at DESC
`

func (q *Queries) ListFavorites(ctx context.Context, userID int64) ([]FavoriteRow, error) {
	rows, err := q.db.Query(ctx, listFavorites, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FavoriteRow
	for rows.Next() {
		var i FavoriteRow
		if err := rows.Scan(&i.ID, &i.PhotographerID, &i.Name, &i.Avatar, &i.Location, &i.Rating, &i.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
```

- [ ] **Step 2: chat.sql.go**

```go
package repository

import (
	"context"
)

const upsertSession = `-- name: UpsertSession :one
INSERT INTO chat_sessions (user1_id, user2_id) VALUES ($1, $2)
ON CONFLICT (user1_id, user2_id) DO UPDATE SET updated_at = NOW()
RETURNING id
`

func (q *Queries) UpsertSession(ctx context.Context, user1ID, user2ID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, upsertSession, user1ID, user2ID).Scan(&id)
	return id, err
}

const listSessions = `-- name: ListSessions :many
SELECT s.id,
       CASE WHEN s.user1_id = $1 THEN s.user2_id ELSE s.user1_id END AS peer_id,
       u.name AS peer_name, COALESCE(u.avatar, '') AS peer_avatar,
       COALESCE(m.content, '') AS last_message,
       m.created_at::text AS last_time
FROM chat_sessions s
LEFT JOIN LATERAL (
  SELECT content, created_at FROM chat_messages WHERE session_id = s.id ORDER BY created_at DESC LIMIT 1
) m ON true
LEFT JOIN users u ON u.id = CASE WHEN s.user1_id = $1 THEN s.user2_id ELSE s.user1_id END
WHERE s.user1_id = $1 OR s.user2_id = $1
ORDER BY s.updated_at DESC
`

type SessionRow struct {
	ID          int64  `json:"id"`
	PeerID      int64  `json:"peerId"`
	PeerName    string `json:"peerName"`
	PeerAvatar  string `json:"peerAvatar"`
	LastMessage string `json:"lastMessage"`
	LastTime    string `json:"lastTime"`
}

func (q *Queries) ListSessions(ctx context.Context, userID int64) ([]SessionRow, error) {
	rows, err := q.db.Query(ctx, listSessions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SessionRow
	for rows.Next() {
		var i SessionRow
		if err := rows.Scan(&i.ID, &i.PeerID, &i.PeerName, &i.PeerAvatar, &i.LastMessage, &i.LastTime); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listMessages = `-- name: ListMessages :many
SELECT id, sender_id, content, created_at::text
FROM chat_messages
WHERE session_id = $1
ORDER BY created_at ASC
`

type MessageRow struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"senderId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (q *Queries) ListMessages(ctx context.Context, sessionID int64) ([]MessageRow, error) {
	rows, err := q.db.Query(ctx, listMessages, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MessageRow
	for rows.Next() {
		var i MessageRow
		if err := rows.Scan(&i.ID, &i.SenderID, &i.Content, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const insertMessage = `-- name: InsertMessage :one
INSERT INTO chat_messages (session_id, sender_id, content) VALUES ($1, $2, $3)
RETURNING id
`

func (q *Queries) InsertMessage(ctx context.Context, sessionID, senderID int64, content string) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, insertMessage, sessionID, senderID, content).Scan(&id)
	return id, err
}
```

- [ ] **Step 3: querier.go 注册**

在 Queries 接口追加：

```go
	// Favorites
	InsertFavorite(ctx context.Context, userID int64, photographerID int32) error
	DeleteFavorite(ctx context.Context, userID int64, photographerID int32) error
	ListFavorites(ctx context.Context, userID int64) ([]FavoriteRow, error)

	// Chat
	UpsertSession(ctx context.Context, user1ID, user2ID int64) (int64, error)
	ListSessions(ctx context.Context, userID int64) ([]SessionRow, error)
	ListMessages(ctx context.Context, sessionID int64) ([]MessageRow, error)
	InsertMessage(ctx context.Context, sessionID, senderID int64, content string) (int64, error)
```

- [ ] **Step 4: 构建验证**

Run: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH && cd server && go build ./...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add server/internal/repository/favorites.sql.go server/internal/repository/chat.sql.go server/internal/repository/querier.go
git commit -m "feat(repo): favorites + chat session/message CRUD"
```

---

### Task 3: 后端 service — favorite_service + chat_service

**Files:**
- Create: `server/internal/service/favorite_service.go`
- Create: `server/internal/service/chat_service.go`

**Interfaces:**
- Consumes: Task 2 repo 方法
- Produces:
  - `FavoriteService`（NewFavoriteService(queries)）；`Add(ctx, userID int64, photographerID int32) error`；`Remove(ctx, userID int64, photographerID int32) error`；`List(ctx, userID int64) ([]FavoriteItem, error)`
  - `ChatService`（NewChatService(queries)）；`GetOrCreateSession(ctx, userID, otherUserID int64) (int64, error)`（规范化 user1<user2）；`ListSessions(ctx, userID int64) ([]SessionItem, error)`；`ListMessages(ctx, sessionID int64) ([]MessageItem, error)`；`SendMessage(ctx, sessionID, senderID int64, content string) (int64, error)`
  - 类型: FavoriteItem{PhotographerID, Name, Avatar, Location, Rating string, AddedAt string}；SessionItem{ID, PeerID, PeerName, PeerAvatar, LastMessage, LastTime}；MessageItem{ID, SenderID, Content, CreatedAt}

- [ ] **Step 1: favorite_service.go**

```go
package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type FavoriteItem struct {
	PhotographerID int32  `json:"photographerId"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	Location       string `json:"location"`
	Rating         string `json:"rating"`
	AddedAt        string `json:"addedAt"`
}

type FavoriteService struct {
	queries *repository.Queries
}

func NewFavoriteService(queries *repository.Queries) *FavoriteService {
	return &FavoriteService{queries: queries}
}

func (s *FavoriteService) Add(ctx context.Context, userID int64, photographerID int32) error {
	return s.queries.InsertFavorite(ctx, userID, photographerID)
}

func (s *FavoriteService) Remove(ctx context.Context, userID int64, photographerID int32) error {
	return s.queries.DeleteFavorite(ctx, userID, photographerID)
}

func (s *FavoriteService) List(ctx context.Context, userID int64) ([]FavoriteItem, error) {
	rows, err := s.queries.ListFavorites(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]FavoriteItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, FavoriteItem{
			PhotographerID: r.PhotographerID,
			Name:           r.Name,
			Avatar:         r.Avatar,
			Location:       r.Location,
			Rating:         r.Rating,
			AddedAt:        r.AddedAt,
		})
	}
	return items, nil
}
```

- [ ] **Step 2: chat_service.go**

```go
package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type SessionItem struct {
	ID          int64  `json:"id"`
	PeerID      int64  `json:"peerId"`
	PeerName    string `json:"peerName"`
	PeerAvatar  string `json:"peerAvatar"`
	LastMessage string `json:"lastMessage"`
	LastTime    string `json:"lastTime"`
}

type MessageItem struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"senderId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type ChatService struct {
	queries *repository.Queries
}

func NewChatService(queries *repository.Queries) *ChatService {
	return &ChatService{queries: queries}
}

func minMax(a, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}

func (s *ChatService) GetOrCreateSession(ctx context.Context, userID, otherUserID int64) (int64, error) {
	u1, u2 := minMax(userID, otherUserID)
	return s.queries.UpsertSession(ctx, u1, u2)
}

func (s *ChatService) ListSessions(ctx context.Context, userID int64) ([]SessionItem, error) {
	rows, err := s.queries.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]SessionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, SessionItem{
			ID:          r.ID,
			PeerID:      r.PeerID,
			PeerName:    r.PeerName,
			PeerAvatar:  r.PeerAvatar,
			LastMessage: r.LastMessage,
			LastTime:    r.LastTime,
		})
	}
	return items, nil
}

func (s *ChatService) ListMessages(ctx context.Context, sessionID int64) ([]MessageItem, error) {
	rows, err := s.queries.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	items := make([]MessageItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, MessageItem{
			ID:        r.ID,
			SenderID:  r.SenderID,
			Content:   r.Content,
			CreatedAt: r.CreatedAt,
		})
	}
	return items, nil
}

func (s *ChatService) SendMessage(ctx context.Context, sessionID, senderID int64, content string) (int64, error) {
	id, err := s.queries.InsertMessage(ctx, sessionID, senderID, content)
	if err == nil {
		_, _ = s.queries.UpsertSessionLastActivity(ctx, sessionID) // no-op vs — see note
	}
	return id, err
}
```

注：`UpsertSessionLastActivity` 未定义——改为**不调用**（updated_at 只在 UpsertSession 时更新，SendMessage 后会话排序由 ListSessions 的 LATERAL last_time 驱动 ORDER BY s.updated_at——demo 可接受，排序轻微滞后）。删除该行。

- [ ] **Step 3: 构建验证**

Run: `cd server && go build ./...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add server/internal/service/favorite_service.go server/internal/service/chat_service.go
git commit -m "feat(service): favorites + chat business logic"
```

---

### Task 4: 后端 handler + 路由

**Files:**
- Create: `server/internal/handler/favorite_handler.go`
- Create: `server/internal/handler/chat_handler.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**
- Consumes: FavoriteService/ChatService
- Produces:
  - `POST /api/v1/favorites` body {userId, photographerId} → 201
  - `DELETE /api/v1/favorites/:userId/:photographerId` → 204
  - `GET /api/v1/favorites/:userId` → [{photographerId,name,avatar,location,rating,addedAt}]
  - `POST /api/v1/chat/sessions` body {userId, otherUserId} → 201 {id}
  - `GET /api/v1/chat/sessions/:userId` → [{id,peerId,peerName,peerAvatar,lastMessage,lastTime}]
  - `GET /api/v1/chat/messages/:sessionId` → [{id,senderId,content,createdAt}]
  - `POST /api/v1/chat/messages` body {sessionId, senderId, content} → 201 {id}
  - auth: favorites 走 AuthRequired（userId 参数即操作者）

- [ ] **Step 1: favorite_handler.go**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

type FavoriteHandler struct {
	svc *service.FavoriteService
}

func NewFavoriteHandler(svc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

func (h *FavoriteHandler) Add(c *gin.Context) {
	var req struct {
		UserID          int64 `json:"userId" binding:"required"`
		PhotographerID  int32 `json:"photographerId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.svc.Add(c.Request.Context(), req.UserID, req.PhotographerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *FavoriteHandler) Remove(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	photographerID, err := strconv.Atoi(c.Param("photographerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	if err := h.svc.Remove(c.Request.Context(), userID, int32(photographerID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *FavoriteHandler) List(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	items, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}
```

- [ ] **Step 2: chat_handler.go**

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) CreateSession(c *gin.Context) {
	var req struct {
		UserID      int64 `json:"userId" binding:"required"`
		OtherUserID int64 `json:"otherUserId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := h.svc.GetOrCreateSession(c.Request.Context(), req.UserID, req.OtherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	items, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("sessionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	items, err := h.svc.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req struct {
		SessionID int64  `json:"sessionId" binding:"required"`
		SenderID  int64  `json:"senderId" binding:"required"`
		Content   string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := h.svc.SendMessage(c.Request.Context(), req.SessionID, req.SenderID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
```

- [ ] **Step 3: main.go 注册**

在 auth 区块附近追加（在 pool 作用域内）：

```go
		// Favorites
		favoriteSvc := service.NewFavoriteService(queries)
		favoriteH := handler.NewFavoriteHandler(favoriteSvc)
		router.POST("/api/v1/favorites", middleware.AuthRequired(userRepo), favoriteH.Add)
		router.DELETE("/api/v1/favorites/:userId/:photographerId", middleware.AuthRequired(userRepo), favoriteH.Remove)
		router.GET("/api/v1/favorites/:userId", middleware.AuthRequired(userRepo), favoriteH.List)

		// Chat
		chatSvc := service.NewChatService(queries)
		chatH := handler.NewChatHandler(chatSvc)
		router.POST("/api/v1/chat/sessions", middleware.AuthRequired(userRepo), chatH.CreateSession)
		router.GET("/api/v1/chat/sessions/:userId", middleware.AuthRequired(userRepo), chatH.ListSessions)
		router.GET("/api/v1/chat/messages/:sessionId", middleware.AuthRequired(userRepo), chatH.ListMessages)
		router.POST("/api/v1/chat/messages", middleware.AuthRequired(userRepo), chatH.SendMessage)
```

注意：userRepo 在 auth 区块定义，需在其后；确认位置在 pool/queries 初始化后、userRepo 定义之后。

- [ ] **Step 4: 构建 + 重启 + 冒烟**

Run:
```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build -o /tmp/mila-api ./cmd/api
kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2
(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
# favorites
curl -s -X POST http://127.0.0.1:8080/api/v1/favorites -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"userId":1,"photographerId":2}'
curl -s http://127.0.0.1:8080/api/v1/favorites/1 -H "Authorization: Bearer $TOKEN"
curl -s -o /dev/null -w "del: %{http_code}\n" -X DELETE http://127.0.0.1:8080/api/v1/favorites/1/2 -H "Authorization: Bearer $TOKEN"
# chat
SID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/chat/sessions -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"userId":1,"otherUserId":1}' | python3 -c "import json,sys;print(json.load(sys.stdin)['id'])")
echo "SID=$SID"
curl -s -X POST http://127.0.0.1:8080/api/v1/chat/messages -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "{\"sessionId\":$SID,\"senderId\":1,\"content\":\"hi\"}"
curl -s http://127.0.0.1:8080/api/v1/chat/messages/$SID -H "Authorization: Bearer $TOKEN"
curl -s http://127.0.0.1:8080/api/v1/chat/sessions/1 -H "Authorization: Bearer $TOKEN"
```
Expected: favorites 201 → list 含 id=2 → del 204；chat SID 创建、消息 201、messages 返回 hi、sessions 返回含 peerName

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/favorite_handler.go server/internal/handler/chat_handler.go server/cmd/api/main.go
git commit -m "feat(api): favorites + chat REST endpoints"
```

---

### Task 5: 前端 — photographer detail 收藏真实化 + 聊天入口

**Files:**
- Modify: `src/pages/photographer/detail.vue`

**Interfaces:**
- Consumes: `apiGet/apiPost/apiDelete`（client.ts）；`useUserStore`；`requireAuth`（utils/auth）
- Produces: `collect()` 改调 API；`isCollected` 初始化；`goChat()` 创建会话后跳转

- [ ] **Step 1: 重写 collect + 初始化**

```ts
import { apiGet, apiPost, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const collected = ref(false)

async function initCollected() {
  if (!userStore.isLoggedIn || !userStore.user) return
  try {
    const res = await apiGet<any[]>(`/v1/favorites/${userStore.user.id}`)
    collected.value = (res || []).some((f: any) => String(f.photographerId) === id)
  } catch { /* ignore */ }
}

async function collect() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/detail?id=' + id }), 800)
    return
  }
  const pid = Number(id)
  try {
    if (collected.value) {
      await apiDelete(`/v1/favorites/${userStore.user.id}/${pid}`)
      collected.value = false
      uni.showToast({ title: '取消收藏', icon: 'none' })
    } else {
      await apiPost('/v1/favorites', { userId: Number(userStore.user.id), photographerId: pid })
      collected.value = true
      uni.showToast({ title: '已收藏', icon: 'success' })
    }
  } catch {
    uni.showToast({ title: '操作失败', icon: 'none' })
  }
}
```

onMounted/load 完成后调用 `initCollected()`。

- [ ] **Step 2: 重写 goChat（创建会话）**

```ts
async function goChat() {
  if (!userStore.isLoggedIn || !userStore.user) {
    uni.showToast({ title: '请先登录', icon: 'none' })
    setTimeout(() => uni.navigateTo({ url: '/pages/login/index?redirect=/pages/photographer/detail?id=' + id }), 800)
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/sessions', {
      userId: Number(userStore.user.id),
      otherUserId: Number(id)   // demo: id 即摄影师用户的 userId（seed 摄影师 id=1..4 对应 users id=1..4 简化）
    })
    uni.navigateTo({ url: `/pages/chat/index?sessionId=${res.id}&peerName=${encodeURIComponent(photographer.value.name)}&peerAvatar=${encodeURIComponent(photographer.value.avatar)}` })
  } catch {
    uni.showToast({ title: '无法发起会话', icon: 'none' })
  }
}
```

注：demo 简化——photographer.id 视作对方 userId（seed 数据 1:1 映射）。若后续摄影师有独立 userId 需后端联查，标注。

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "photographer/detail" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/pages/photographer/detail.vue
git commit -m "feat(photo-detail): real favorite toggle + chat session entry"
```

---

### Task 6: 前端 — 我的收藏列表页（新页面）

**Files:**
- Create: `src/pages/favorite/list.vue`
- Modify: `src/pages.json`
- Modify: `src/pages/profile/index.vue`

**Interfaces:**
- Consumes: `apiGet('/v1/favorites/:userId')`；`PhotographerCard` 组件（可复用或简单卡片）
- Produces: 收藏列表渲染 + 空状态

- [ ] **Step 1: favorites/list.vue**

```vue
<template>
  <view class="page">
    <view class="status-bar" :style="{ height: statusBarHeight + 'px' }" />
    <view class="nav-bar" :style="{ paddingTop: statusBarHeight + 'px' }">
      <text class="back-arrow" @click="goBack">‹</text>
      <text class="nav-title">我的收藏</text>
    </view>
    <view v-if="loading" class="empty-hint">加载中...</view>
    <view v-else-if="favorites.length === 0" class="empty-state">
      <text class="empty-icon">☆</text>
      <text class="empty-text">还没有收藏摄影师</text>
      <view class="btn-go" @click="goHome">去逛逛</view>
    </view>
    <view v-else class="list">
      <view v-for="f in favorites" :key="f.photographerId" class="fav-card" @click="goDetail(f.photographerId)">
        <image class="avatar" :src="f.avatar" mode="aspectFill" />
        <view class="info">
          <text class="name">{{ f.name }}</text>
          <text class="meta">{{ f.location }} · 评分 {{ f.rating }}</text>
        </view>
        <text class="unfav" @click.stop="unfavorite(f.photographerId)">取消</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiGet, apiDelete } from '@/api/client'
import { useUserStore } from '@/stores/user'

const statusBarHeight = ref(44)
const favorites = ref<any[]>([])
const loading = ref(true)
const userStore = useUserStore()

onMounted(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44
  if (!userStore.isLoggedIn || !userStore.user) {
    loading.value = false
    uni.showToast({ title: '请先登录', icon: 'none' })
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/favorites/${userStore.user.id}`)
    favorites.value = res || []
  } catch {
    favorites.value = []
  } finally {
    loading.value = false
  }
})

async function unfavorite(pid: number) {
  if (!userStore.user) return
  try {
    await apiDelete(`/v1/favorites/${userStore.user.id}/${pid}`)
    favorites.value = favorites.value.filter(f => f.photographerId !== pid)
    uni.showToast({ title: '已取消', icon: 'none' })
  } catch {
    uni.showToast({ title: '操作失败', icon: 'none' })
  }
}

function goDetail(pid: number) {
  uni.navigateTo({ url: `/pages/photographer/detail?id=${pid}` })
}

function goHome() {
  uni.switchTab({ url: '/pages/index/index' })
}

function goBack() {
  uni.navigateBack()
}
</script>

<style lang="scss" scoped>
.page { min-height: 100vh; background: $dark-bg-primary; }
.status-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 100; }
.nav-bar { position: fixed; top: 0; left: 0; right: 0; z-index: 99; height: 88rpx; display: flex; align-items: center; background: $dark-bg-primary; border-bottom: 1rpx solid $dark-border; }
.back-arrow { font-size: 48rpx; color: $dark-text-primary; padding: 0 24rpx; }
.nav-title { font-size: 34rpx; font-weight: 600; color: $dark-text-primary; }
.list { padding: 100rpx 32rpx 32rpx; }
.fav-card { display: flex; align-items: center; background: $dark-bg-card; border: 1rpx solid $dark-border; border-radius: 16rpx; padding: 24rpx; margin-bottom: 20rpx; }
.avatar { width: 96rpx; height: 96rpx; border-radius: 50%; margin-right: 24rpx; }
.info { flex: 1; }
.name { display: block; font-size: 30rpx; color: $dark-text-primary; font-weight: 600; }
.meta { display: block; font-size: 24rpx; color: $dark-text-tertiary; margin-top: 8rpx; }
.unfav { font-size: 24rpx; color: $neon-purple; padding: 8rpx 16rpx; }
.empty-state { display: flex; flex-direction: column; align-items: center; padding-top: 200rpx; }
.empty-icon { font-size: 100rpx; color: $dark-text-tertiary; }
.empty-text { font-size: 28rpx; color: $dark-text-secondary; margin: 24rpx 0; }
.btn-go { padding: 16rpx 48rpx; background: $neon-gradient; color: #fff; border-radius: 32rpx; font-size: 28rpx; }
.empty-hint { text-align: center; padding-top: 200rpx; color: $dark-text-tertiary; }
</style>
```

- [ ] **Step 2: pages.json 注册**

```json
{
  "path": "pages/favorite/list",
  "style": {
    "navigationStyle": "custom"
  }
}
```

- [ ] **Step 3: profile goFavorites**

```ts
function goFavorites() {
  uni.navigateTo({ url: '/pages/favorite/list' })
}
```

替换原 toast。

- [ ] **Step 4: 类型检查 + H5 实测**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "favorite" | head -5` — 无错误
Playwright: 登录 → 收藏一个摄影师 → profile 我的收藏 → 列表显示；取消 → 消失。

- [ ] **Step 5: Commit**

```bash
git add src/pages/favorite/list.vue src/pages.json src/pages/profile/index.vue
git commit -m "feat(favorites): my favorites page wired to real API"
```

---

### Task 7: 前端 — chat 页真实化

**Files:**
- Modify: `src/pages/chat/index.vue`

**Interfaces:**
- Consumes: `apiGet/apiPost`；`useUserStore`；路由参数 sessionId/peerName/peerAvatar
- Produces: 消息从 API 加载；发送落库；移除 pickReply/replyPool/defaultReplies

- [ ] **Step 1: 重写 script 核心逻辑（保留模板结构）**

```ts
import { ref, onMounted, nextTick } from 'vue'
import { apiGet, apiPost } from '@/api/client'
import { useUserStore } from '@/stores/user'

interface ChatMessage {
  id: string
  content: string
  isMe: boolean
  time: string
}

const statusBarHeight = ref(44)
const inputValue = ref('')
const scrollToId = ref('')
const typing = ref(false)
const sessionId = ref('')
const loading = ref(true)

const myAvatar = 'https://picsum.photos/seed/FemaleCoser/150/150'
const userStore = useUserStore()

const chatUser = ref({
  name: '摄影师',
  avatar: 'https://picsum.photos/seed/PhotographerLight/150/150'
})

const messages = ref<ChatMessage[]>([])

function formatTime(): string {
  const now = new Date()
  const h = now.getHours().toString().padStart(2, '0')
  const m = now.getMinutes().toString().padStart(2, '0')
  return `${h}:${m}`
}

onMounted(async () => {
  const sysInfo = uni.getSystemInfoSync()
  statusBarHeight.value = sysInfo.statusBarHeight || 44

  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1] as any
  sessionId.value = currentPage?.options?.sessionId as string || ''
  const peerName = currentPage?.options?.peerName as string
  const peerAvatar = currentPage?.options?.peerAvatar as string
  if (peerName) chatUser.value.name = decodeURIComponent(peerName)
  if (peerAvatar) chatUser.value.avatar = decodeURIComponent(peerAvatar)

  if (!sessionId.value) {
    loading.value = false
    uni.showToast({ title: '会话不存在', icon: 'none' })
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/chat/messages/${sessionId.value}`)
    messages.value = (res || []).map((m: any) => ({
      id: String(m.id),
      content: m.content,
      isMe: String(m.senderId) === String(userStore.user?.id),
      time: formatTimeFromCreated(m.createdAt)
    }))
  } catch {
    messages.value = []
  } finally {
    loading.value = false
    scrollToBottom()
  }
})

function formatTimeFromCreated(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
  } catch { return '' }
}

function scrollToBottom() {
  nextTick(() => { scrollToId.value = 'msg-bottom' })
}

async function sendMessage() {
  if (!inputValue.value.trim() || typing.value) return
  const content = inputValue.value.trim()
  if (!sessionId.value || !userStore.user) {
    uni.showToast({ title: '无法发送', icon: 'none' })
    return
  }
  try {
    const res = await apiPost<{ id: number }>('/v1/chat/messages', {
      sessionId: Number(sessionId.value),
      senderId: Number(userStore.user.id),
      content
    })
    messages.value.push({ id: String(res.id), content, isMe: true, time: formatTime() })
    inputValue.value = ''
    scrollToBottom()
  } catch {
    uni.showToast({ title: '发送失败', icon: 'none' })
  }
}

function goBack() {
  uni.navigateBack()
}
```

删除 `replyPool`、`defaultReplies`、`pickReply`、初始 `messages` 数组。模板中 `v-for msg in messages` 结构不变，`is-me` class 依赖 `msg.isMe` 保留。

- [ ] **Step 2: 类型检查 + H5 实测**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "chat" | head -5` — 无错误
Playwright: 摄影师详情"聊天" → chat 页加载空会话 → 发送"你好" → 刷新消息仍在（落库验证）。

- [ ] **Step 3: Commit**

```bash
git add src/pages/chat/index.vue
git commit -m "feat(chat): real session messages via API, remove auto-reply"
```

---

### Task 8: 前端 — message 页会话列表真实化

**Files:**
- Modify: `src/pages/message/index.vue`

**Interfaces:**
- Consumes: `apiGet('/v1/chat/sessions/:userId')`；`useUserStore`
- Produces: sessions 从 API 加载；通知部分保留 demo 横幅（P3 再接通知表）

- [ ] **Step 1: 会话列表真实化**

```ts
import { apiGet } from '@/api/client'
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()

// onMounted 加载 sessions
async function loadSessions() {
  if (!userStore.isLoggedIn || !userStore.user) {
    sessions.value = []
    return
  }
  try {
    const res = await apiGet<any[]>(`/v1/chat/sessions/${userStore.user.id}`)
    sessions.value = (res || []).map((s: any) => ({
      id: String(s.id),
      userId: String(s.peerId),
      userName: s.peerName,
      userAvatar: s.peerAvatar,
      lastMessage: s.lastMessage,
      unreadCount: 0,
      updatedAt: s.lastTime || Date.now()
    }))
  } catch {
    sessions.value = []
  }
}
```

模板 `sessions` v-for 的点击跳转改带 peer 参数：

```ts
function goChat(session: any) {
  uni.navigateTo({ url: `/pages/chat/index?sessionId=${session.id}&peerName=${encodeURIComponent(session.userName)}&peerAvatar=${encodeURIComponent(session.userAvatar)}` })
}
```

notifications 硬编码数组保留 + demo 横幅保留（P3 再接）。

- [ ] **Step 2: 类型检查 + H5 实测**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "message" | head -5` — 无错误
Playwright: 发过消息后进消息页 → 会话列表显示真实会话。

- [ ] **Step 3: Commit**

```bash
git add src/pages/message/index.vue
git commit -m "feat(message): real chat sessions list, keep notification demo"
```

---

### Task 9: 全链路验证 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: 后端全链路冒烟（收藏+聊天）**

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
# 收藏
curl -s -X POST http://127.0.0.1:8080/api/v1/favorites -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"userId":1,"photographerId":3}'
curl -s http://127.0.0.1:8080/api/v1/favorites/1 -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; print('favs:', len(json.load(sys.stdin)))"
# 会话+消息
SID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/chat/sessions -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"userId":1,"otherUserId":2}' | python3 -c "import json,sys;print(json.load(sys.stdin)['id'])")
curl -s -X POST http://127.0.0.1:8080/api/v1/chat/messages -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "{\"sessionId\":$SID,\"senderId\":1,\"content\":\"你好，想约拍\"}"
curl -s "http://127.0.0.1:8080/api/v1/chat/messages/$SID" -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('msgs:', len(d), '| last:', d[-1]['content'] if d else 'none')"
curl -s "http://127.0.0.1:8080/api/v1/chat/sessions/1" -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('sessions:', len(d), '| peer:', d[0]['peerName'] if d else 'none')"
```
Expected: favs≥1；msgs≥1 且 last=你好，想约拍；sessions 含 peerName

- [ ] **Step 2: Playwright H5 全链路**

登录 → 摄影师详情（收藏★）→ 我的收藏页（列表含该摄影师）→ 聊天（发送消息）→ 消息页（会话列表含该会话）。断言每步。

- [ ] **Step 3: 回归**

Run: `cd server && go test ./... 2>&1 | tail -5`（全绿）；`npx vue-tsc --noEmit 2>&1 | grep -cE "error"`（≤1 pre-existing）
Run: `npm run build:h5 2>&1 | tail -3`（构建成功）

- [ ] **Step 4: 视觉复查（用户授权）**

用 agent-browser 全站截图对比修复前后：重点 chat 页（不再是假消息）、我的收藏页（新页面）、消息页（真实会话）。发现问题列 task 纠正。

- [ ] **Step 5: AGENTS.md 更新**

新增：favorites 表 + /api/v1/favorites 三接口；chat_sessions/messages 表 + /api/v1/chat/* 四接口；前端收藏页/聊天真实化说明。

- [ ] **Step 6: 提交推送**

```bash
git add AGENTS.md
git commit -m "docs: AGENTS.md — favorites + chat modules"
git -c http.proxy= -c https.proxy= push gitee master
```

---

## 自审记录

- **Spec 覆盖**: 3.1 收藏（迁移/API/前端 detail+新页+profile）→ Task 1/2/3/4/5/6；3.2 聊天（迁移/API/前端 chat+message）→ Task 1/2/3/4/7/8；M4 验证 → Task 9。spec "demo 简化"（未读数 0、路由传 peer 参数、photographer.id 视作 userId）在各任务注明。
- **占位符**: 无；每任务含具体代码/命令。
- **类型一致**: `InsertFavorite(ctx, int64, int32)` Task 2 定义、Task 3 service 用；`UpsertSession(ctx, u1, u2)` → `GetOrCreateSession`；favorite_service 的 FavoriteItem{PhotographerID,Name,Avatar,Location,Rating,AddedAt} 与 handler JSON 一致；chat_service 的 SessionItem/MessageItem 与 handler 一致。Task 3 note 修正了误引用的 `UpsertSessionLastActivity`（改为不调用）。
- **已知取舍**: 会话 updated_at 不随发消息更新（排序略滞后，demo 接受）；photographer.id 与 users.id 1:1 映射假设（seed 数据成立，标注）；未读数恒 0。
