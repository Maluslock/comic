# 订单域核心：状态机 + 时间冲突 + 订单联查 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让预约订单从"只能查看"变成"真实流转"——后端状态机 API + 时间冲突检测 + 摄影师联查，前端订单操作改调真实 API。

**Architecture:** 后端在 bookings 域新增 UpdateStatus（状态机校验）、占用时段查询与冲突检测、列表联查 photographers/services；前端 client.ts 增 apiPut，order 两页与 booking 页改调真实 API，移除本地 storage 状态写入。

**Tech Stack:** Go 1.22 + Gin + pgx（server/）；Vue 3 script setup + uni-app（src/）。已有 middleware.AuthRequired、repository.Queries 手写 SQL 模式。

## Global Constraints

- Go 构建需显式导出: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH`
- PG: `postgres://comic:comic123@127.0.0.1:5433/comic`（docker comic-pg）；重启 API: `kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')` 后 `(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &)`；**禁用 pkill -f（会自杀）**
- TypeScript strict，禁止 `as any` 新增（已有 `apiGet<any>` 调用保持现状，新代码不得引入）
- SCSS 变量/rpx/`@/` 别名；无 emoji 图标
- 状态机合法迁移（spec 权威）：pending→{confirmed,cancelled}；confirmed→{completed,cancelled}；completed/cancelled 终态；非法迁移 409
- 每个任务完成即 commit；不 push（编排者统一推 Gitee：`git -c http.proxy= -c https.proxy= push gitee master`）
- demo 约定：coser 可直接 confirmed（"确认接单"按钮标注演示用），摄影师接单流程 P3 再做
- 本地 storage 'bookings' 保留为**只读 fallback**，不再写入状态

---

## 文件结构总览

| 文件 | 职责 |
|------|------|
| `server/internal/repository/bookings.sql.go` | 新增 SQL: GetBookingByID / GetOccupiedTimes / CountConflict / UpdateBookingStatus / GetBookingsByUserWithDetails |
| `server/internal/repository/models.go` | 新增 BookingWithDetails struct |
| `server/internal/repository/querier.go` | Queries 接口注册新方法 |
| `server/internal/service/booking_service.go` | 状态机 canTransition + UpdateStatus + Create 冲突检测 + ListByUser 联查 |
| `server/internal/service/booking_service_test.go` | 状态机纯函数测试（无 DB） |
| `server/internal/handler/booking_handler.go` | UpdateStatus handler（409/404）+ TimeSlots handler |
| `server/cmd/api/main.go` | 注册 PUT /bookings/:id/status + GET /photographers/:id/timeslots |
| `src/api/client.ts` | 新增 apiPut |
| `src/pages/order/detail.vue` | 取消/确认/确认接单改真实 API，删 updateStoredOrderStatus |
| `src/pages/order/list.vue` | 同样改真实 API，删 updateOrderStatus 本地写入 |
| `src/pages/booking/index.vue` | loadTimeSlots 改真实占用 API，删硬编码 disabledTimes |

---

### Task 1: 后端 repository — 新增 bookings SQL 查询

**Files:**
- Modify: `server/internal/repository/bookings.sql.go`
- Modify: `server/internal/repository/models.go`（新增 BookingWithDetails）
- Modify: `server/internal/repository/querier.go`

**Interfaces:**
- Consumes: 现有 `Booking` struct、`Queries`（q.db *pgxpool.Pool 手写 SQL 模式）
- Produces:
  - `GetBookingByID(ctx, id int64) (Booking, error)`
  - `GetOccupiedTimesByPhotographerDate(ctx, photographerID int32, date time.Time) ([]string, error)`
  - `CountConflictBookings(ctx, photographerID int32, date time.Time, timeStr string) (int64, error)`
  - `UpdateBookingStatus(ctx, id int64, status string) (Booking, error)`
  - `GetBookingsByUserWithDetails(ctx, coserID int32) ([]BookingWithDetails, error)`
  - `type BookingWithDetails struct { Booking; PhotographerName string; PhotographerAvatar string; ServiceName string; ServicePrice int32 }`（Go 嵌入 struct 字段平铺 JSON）

- [ ] **Step 1: 写新 SQL 查询函数**

在 bookings.sql.go 末尾追加（遵循现有手写模式）：

```go
const getBookingByID = `-- name: GetBookingByID :one
SELECT id, photographer_id, coser_id, service_id, date, time, status, total_price, remarks, created_at, updated_at
FROM bookings WHERE id = $1
`

func (q *Queries) GetBookingByID(ctx context.Context, id int64) (Booking, error) {
	row := q.db.QueryRow(ctx, getBookingByID, id)
	var i Booking
	err := row.Scan(&i.ID, &i.PhotographerID, &i.CoserID, &i.ServiceID, &i.Date, &i.Time, &i.Status, &i.TotalPrice, &i.Remarks, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const getOccupiedTimes = `-- name: GetOccupiedTimesByPhotographerDate :many
SELECT time FROM bookings
WHERE photographer_id = $1 AND date = $2 AND status <> 'cancelled'
`

func (q *Queries) GetOccupiedTimesByPhotographerDate(ctx context.Context, photographerID int32, date time.Time) ([]string, error) {
	rows, err := q.db.Query(ctx, getOccupiedTimes, photographerID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var times []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		times = append(times, t)
	}
	return times, rows.Err()
}

const countConflict = `-- name: CountConflictBookings :one
SELECT COUNT(*) FROM bookings
WHERE photographer_id = $1 AND date = $2 AND time = $3 AND status <> 'cancelled'
`

func (q *Queries) CountConflictBookings(ctx context.Context, photographerID int32, date time.Time, timeStr string) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, countConflict, photographerID, date, timeStr).Scan(&n)
	return n, err
}

const updateBookingStatus = `-- name: UpdateBookingStatus :one
UPDATE bookings SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, photographer_id, coser_id, service_id, date, time, status, total_price, remarks, created_at, updated_at
`

func (q *Queries) UpdateBookingStatus(ctx context.Context, id int64, status string) (Booking, error) {
	row := q.db.QueryRow(ctx, updateBookingStatus, id, status)
	var i Booking
	err := row.Scan(&i.ID, &i.PhotographerID, &i.CoserID, &i.ServiceID, &i.Date, &i.Time, &i.Status, &i.TotalPrice, &i.Remarks, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const getBookingsByUserWithDetails = `-- name: GetBookingsByUserWithDetails :many
SELECT b.id, b.photographer_id, b.coser_id, b.service_id, b.date, b.time, b.status, b.total_price, b.remarks, b.created_at, b.updated_at,
       p.name AS photographer_name, p.avatar AS photographer_avatar,
       s.name AS service_name, COALESCE(s.price, 0) AS service_price
FROM bookings b
LEFT JOIN photographers p ON p.id = b.photographer_id
LEFT JOIN services s ON s.id = b.service_id
WHERE b.coser_id = $1
ORDER BY b.created_at DESC
`

func (q *Queries) GetBookingsByUserWithDetails(ctx context.Context, coserID int32) ([]BookingWithDetails, error) {
	rows, err := q.db.Query(ctx, getBookingsByUserWithDetails, coserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookingWithDetails
	for rows.Next() {
		var i BookingWithDetails
		if err := rows.Scan(&i.ID, &i.PhotographerID, &i.CoserID, &i.ServiceID, &i.Date, &i.Time, &i.Status, &i.TotalPrice, &i.Remarks, &i.CreatedAt, &i.UpdatedAt, &i.PhotographerName, &i.PhotographerAvatar, &i.ServiceName, &i.ServicePrice); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
```

- [ ] **Step 2: models.go 加 BookingWithDetails**

```go
// BookingWithDetails embeds Booking and adds joined photographer/service fields.
type BookingWithDetails struct {
	Booking
	PhotographerName   string `json:"photographer_name"`
	PhotographerAvatar string `json:"photographer_avatar"`
	ServiceName        string `json:"service_name"`
	ServicePrice       int32  `json:"service_price"`
}
```

- [ ] **Step 3: querier.go 接口注册**

在 `// Bookings` 区块追加：

```go
	GetBookingByID(ctx context.Context, id int64) (Booking, error)
	GetOccupiedTimesByPhotographerDate(ctx context.Context, photographerID int32, date time.Time) ([]string, error)
	CountConflictBookings(ctx context.Context, photographerID int32, date time.Time, timeStr string) (int64, error)
	UpdateBookingStatus(ctx context.Context, id int64, status string) (Booking, error)
	GetBookingsByUserWithDetails(ctx context.Context, coserID int32) ([]BookingWithDetails, error)
```

- [ ] **Step 4: 构建验证**

Run: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH && cd server && go build ./...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add server/internal/repository/bookings.sql.go server/internal/repository/models.go server/internal/repository/querier.go
git commit -m "feat(repo): booking status update + occupied times + conflict count + details join queries"
```

---

### Task 2: 后端 service — 状态机 + UpdateStatus + 冲突检测 + 联查

**Files:**
- Modify: `server/internal/service/booking_service.go`
- Create: `server/internal/service/booking_service_test.go`

**Interfaces:**
- Consumes: Task 1 的 5 个 repo 方法；`middleware.AuthRequired`（main.go 已用）
- Produces:
  - `var ErrConflict = errors.New("conflict")`、`var ErrInvalidTransition = errors.New("invalid status transition")`、`var ErrBookingNotFound = errors.New("booking not found")`
  - `func canTransition(from, to string) bool`（纯函数，测试目标）
  - `(s *BookingService) UpdateStatus(ctx, bookingID int64, newStatus string) (*BookingItem, error)` — GetBookingByID（pgx.ErrNoRows→ErrBookingNotFound）→ canTransition（false→ErrInvalidTransition）→ UpdateBookingStatus → 映射 BookingItem
  - `(s *BookingService) GetOccupiedTimes(ctx, photographerID int32, dateStr string) ([]string, error)` — dateStr "2006-01-02"
  - `(s *BookingService) Create` 改造：date 解析后先 `CountConflictBookings`，>0 → ErrConflict
  - `(s *BookingService) ListByUser` 改造：改调 `GetBookingsByUserWithDetails`，BookingItem 映射新字段（见下）

- [ ] **Step 1: 写状态机失败测试**

```go
package service

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "confirmed", true},
		{"pending", "cancelled", true},
		{"pending", "completed", false},
		{"confirmed", "completed", true},
		{"confirmed", "cancelled", true},
		{"confirmed", "pending", false},
		{"completed", "cancelled", false},
		{"completed", "pending", false},
		{"cancelled", "pending", false},
		{"cancelled", "completed", false},
		{"", "confirmed", false},
		{"pending", "", false},
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.to); got != c.want {
			t.Errorf("canTransition(%q, %q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd server && go test ./internal/service/ -run TestCanTransition`
Expected: FAIL — `undefined: canTransition`

- [ ] **Step 3: 实现 canTransition + 错误变量 + UpdateStatus + GetOccupiedTimes**

在 booking_service.go 追加：

```go
import (
	"errors"
	"github.com/jackc/pgx/v5"
)

var (
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrBookingNotFound   = errors.New("booking not found")
)

func canTransition(from, to string) bool {
	switch from {
	case "pending":
		return to == "confirmed" || to == "cancelled"
	case "confirmed":
		return to == "completed" || to == "cancelled"
	default:
		return false
	}
}

func (s *BookingService) UpdateStatus(ctx context.Context, bookingID int64, newStatus string) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if !canTransition(b.Status, newStatus) {
		return nil, ErrInvalidTransition
	}
	updated, err := s.queries.UpdateBookingStatus(ctx, bookingID, newStatus)
	if err != nil {
		return nil, err
	}
	return bookingToItem(updated), nil
}

func (s *BookingService) GetOccupiedTimes(ctx context.Context, photographerID int32, dateStr string) ([]string, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	times, err := s.queries.GetOccupiedTimesByPhotographerDate(ctx, photographerID, date)
	if err != nil {
		return nil, err
	}
	if times == nil {
		times = []string{}
	}
	return times, nil
}
```

- [ ] **Step 4: Create 加冲突检测**

在 `Create` 中 date 解析成功之后、`CreateBooking` 之前插入：

```go
	conflicts, err := s.queries.CountConflictBookings(ctx, req.PhotographerID, date, req.Time)
	if err != nil {
		return nil, err
	}
	if conflicts > 0 {
		return nil, ErrConflict
	}
```

- [ ] **Step 5: ListByUser 改联查 + 抽取 bookingToItem**

将现有两处 Booking→BookingItem 映射抽取为 `bookingToItem(b repository.Booking) *BookingItem`（现有字段映射不变），`ListByUser` 改调 `GetBookingsByUserWithDetails`，映射后填充：

```go
type BookingItem struct {
	// ...现有字段不变...
	PhotographerName   string `json:"photographerName"`
	PhotographerAvatar string `json:"photographerAvatar"`
	ServiceName        string `json:"serviceName"`
	// 注: ServicePrice 暂不进 BookingItem（前端用 serviceName 展示）
}
```

在 ListByUser 循环里：

```go
	item := bookingToItem(b.Booking)
	item.PhotographerName = b.PhotographerName
	item.PhotographerAvatar = b.PhotographerAvatar
	item.ServiceName = b.ServiceName
	items = append(items, *item)
```

- [ ] **Step 6: 跑测试确认通过**

Run: `cd server && go test ./internal/service/ -run TestCanTransition && go build ./...`
Expected: PASS + 构建无错误

- [ ] **Step 7: Commit**

```bash
git add server/internal/service/booking_service.go server/internal/service/booking_service_test.go
git commit -m "feat(service): booking state machine + conflict detection + photographer join"
```

---

### Task 3: 后端 handler + 路由 — UpdateStatus / TimeSlots API

**Files:**
- Modify: `server/internal/handler/booking_handler.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**
- Consumes: `BookingService.UpdateStatus / GetOccupiedTimes`；`middleware.AuthRequired`
- Produces:
  - `PUT /api/v1/bookings/:id/status`（AuthRequired）请求体 `{"status":"confirmed|completed|cancelled"}`；200 返回 BookingItem；409 ErrInvalidTransition/ErrConflict；404 ErrBookingNotFound
  - `GET /api/v1/photographers/:id/timeslots?date=YYYY-MM-DD` 返回 `{"date":"...","occupied":["10:00",...]}`

- [ ] **Step 1: handler 追加两个方法**

```go
func (h *BookingHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	data, err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBookingNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		case errors.Is(err, service.ErrInvalidTransition):
			c.JSON(http.StatusConflict, gin.H{"error": "invalid status transition"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) TimeSlots(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	times, err := h.svc.GetOccupiedTimes(c.Request.Context(), int32(id), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"date": date, "occupied": times})
}
```

handler 顶部 import 加 `"errors"`。

- [ ] **Step 2: Create handler 错误分支补 409**

将 `Create` 的 err 分支改为：

```go
	data, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "该时段已被预约"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
```

- [ ] **Step 3: main.go 注册路由**

在 bookings 区块追加：

```go
		router.PUT("/api/v1/bookings/:id/status", middleware.AuthRequired(userRepo), bookingH.UpdateStatus)
		router.GET("/api/v1/photographers/:id/timeslots", photographerH.TimeSlots)
```

注意：`photographerH` 是 `handler.PhotographerHandler`，需在其上新增 `TimeSlots` 方法——改为在 `booking_handler.go` 中不实现，而是给 `PhotographerHandler` 加方法（文件 `server/internal/handler/photographer_handler.go`）：

```go
func (h *PhotographerHandler) TimeSlots(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	times, err := h.bookingSvc.GetOccupiedTimes(c.Request.Context(), int32(id), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"date": date, "occupied": times})
}
```

为此 `PhotographerHandler` 需加字段 `bookingSvc *service.BookingService`，`NewPhotographerHandler` 增加参数。main.go 构造处改为 `handler.NewPhotographerHandler(photographerSvc, redisCache, bookingSvc)`。

- [ ] **Step 4: 构建 + 重启 + 实测**

Run:
```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build -o /tmp/mila-api ./cmd/api
kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2
(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3
# 登录拿 token
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
# 创建预约
curl -s -X POST http://127.0.0.1:8080/api/v1/bookings -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"photographerId":1,"coserId":1,"serviceId":1,"date":"2026-09-01","time":"10:00","remarks":"test"}'
# 同时段重复预约 → 409
curl -s -o /dev/null -w "conflict: %{http_code}\n" -X POST http://127.0.0.1:8080/api/v1/bookings -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"photographerId":1,"coserId":1,"serviceId":1,"date":"2026-09-01","time":"10:00"}'
# 占用时段
curl -s "http://127.0.0.1:8080/api/v1/photographers/1/timeslots?date=2026-09-01" -H "Authorization: Bearer $TOKEN"
# 状态流转: 合法 + 非法
curl -s -X PUT http://127.0.0.1:8080/api/v1/bookings/<ID>/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"confirmed"}'
curl -s -o /dev/null -w "invalid transition: %{http_code}\n" -X PUT http://127.0.0.1:8080/api/v1/bookings/<ID>/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"pending"}'
```
Expected: 创建 201；重复预约 409；timeslots 返回 `occupied:["10:00"]`；confirmed 200；confirmed→pending 409。（<ID> 用第一条创建返回的 id）

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/booking_handler.go server/internal/handler/photographer_handler.go server/cmd/api/main.go
git commit -m "feat(api): PUT /bookings/:id/status state machine + GET /photographers/:id/timeslots"
```

---

### Task 4: 前端 client.ts — 新增 apiPut

**Files:**
- Modify: `src/api/client.ts`

**Interfaces:**
- Consumes: authHeader/handleUnauthorized（已有）
- Produces: `apiPut<T>(path: string, body: AnyObject): Promise<T>` — 同 apiPost 模式，method 'PUT'，409 抛 `ApiError(409)`（调用方判断 statusCode===409 显示冲突 toast）

- [ ] **Step 1: 写 apiPut**

```ts
export async function apiPut<T>(path: string, body: AnyObject): Promise<T> {
  const res = await uni.request({ url: BASE + path, method: 'PUT', data: body, timeout: 5000, header: authHeader() })
  handleUnauthorized(res)
  if (res.statusCode === 404) throw new NotFoundError()
  if (res.statusCode !== 200) throw new ApiError(res.statusCode, 'request failed')
  return res.data as T
}
```

- [ ] **Step 2: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "client.ts" | head -3`
Expected: client.ts 无新错误

- [ ] **Step 3: Commit**

```bash
git add src/api/client.ts
git commit -m "feat(client): add apiPut for booking status updates"
```

---

### Task 5: 前端 order/detail.vue — 状态操作真实化

**Files:**
- Modify: `src/pages/order/detail.vue`

**Interfaces:**
- Consumes: `apiPut`（Task 4）；`apiGet` 已有；`useUserStore`
- Produces: `updateStatus(newStatus: 'confirmed'|'completed'|'cancelled')` — 调 `apiPut('/v1/bookings/' + order.value.id + '/status', {status})`，成功 toast + `order.value.status = newStatus`；409 → toast "操作不允许"；失败 → toast "操作失败"

- [ ] **Step 1: 替换 cancelOrder / confirmOrder / 新增 confirmAccept**

```ts
import { apiGet, apiPut } from '@/api/client'

async function updateStatus(newStatus: 'confirmed' | 'completed' | 'cancelled') {
  if (!order.value.id) return
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${order.value.id}/status`, { status: newStatus })
    order.value = { ...order.value, status: newStatus }
    uni.showToast({ title: '操作成功', icon: 'success' })
  } catch (e) {
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function confirmAccept() {
  uni.showModal({
    title: '确认接单（演示）',
    content: '模拟摄影师确认接单？',
    success: (res) => { if (res.confirm) updateStatus('confirmed') }
  })
}

function cancelOrder() {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => { if (res.confirm) updateStatus('cancelled') }
  })
}

function confirmOrder() {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => { if (res.confirm) updateStatus('completed') }
  })
}
```

删除 `updateStoredOrderStatus` 函数（不再本地写入）。

- [ ] **Step 2: 模板按钮更新（spec 按钮规则）**

替换 action 按钮区块为：

```html
        <view v-if="order.status === 'pending'" class="btn-outline" @click="confirmAccept">确认接单（演示）</view>
        <view v-if="order.status === 'pending' || order.status === 'confirmed'" class="btn-outline" @click="cancelOrder">取消预约</view>
        <view v-if="order.status === 'confirmed'" class="btn-primary" @click="confirmOrder">确认完成</view>
        <view v-if="order.status === 'completed'" class="btn-primary" @click="goReview">去评价</view>
```

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "order/detail" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/pages/order/detail.vue
git commit -m "feat(order-detail): real status API (cancel/confirm/accept) replaces local storage writes"
```

---

### Task 6: 前端 order/list.vue — 状态操作真实化

**Files:**
- Modify: `src/pages/order/list.vue`

**Interfaces:**
- Consumes: `apiPut`（Task 4）；`loadBookings`（现有）
- Produces: `updateStatus(id, newStatus)` 调真实 API 后 `loadBookings()` 刷新；移除 `updateOrderStatus` 本地写入

- [ ] **Step 1: 替换 cancelOrder / confirmOrder / 新增 confirmAccept**

```ts
import { apiGet, apiPut } from '@/api/client'

async function updateStatus(id: string, newStatus: 'confirmed' | 'completed' | 'cancelled') {
  uni.showLoading({ title: '提交中...' })
  try {
    await apiPut(`/v1/bookings/${id}/status`, { status: newStatus })
    uni.hideLoading()
    uni.showToast({ title: '操作成功', icon: 'success' })
    loadBookings()
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '操作不允许' : '操作失败', icon: 'none' })
  }
}

function confirmAccept(id: string) {
  uni.showModal({
    title: '确认接单（演示）',
    content: '模拟摄影师确认接单？',
    success: (res) => { if (res.confirm) updateStatus(id, 'confirmed') }
  })
}

function cancelOrder(id: string) {
  uni.showModal({
    title: '确认取消',
    content: '确定要取消这个预约吗？',
    success: (res) => { if (res.confirm) updateStatus(id, 'cancelled') }
  })
}

function confirmOrder(id: string) {
  uni.showModal({
    title: '确认完成',
    content: '确认拍摄已完成？',
    success: (res) => { if (res.confirm) updateStatus(id, 'completed') }
  })
}
```

删除 `updateOrderStatus` 函数。同时 `readStoredBookings` 保留（只读 fallback），但 `loadBookings` 中不再把本地订单与服务器合并写回 storage——保持现有合并展示逻辑不动（只读），仅删状态写入路径。

- [ ] **Step 2: 模板按钮更新**

按钮区块改为：

```html
            <view v-if="order.status === 'pending'" class="btn-outline" @click="confirmAccept(order.id)">确认接单</view>
            <view v-if="order.status === 'pending' || order.status === 'confirmed'" class="btn-outline" @click="cancelOrder(order.id)">取消预约</view>
            <view v-if="order.status === 'confirmed'" class="btn-primary" @click="confirmOrder(order.id)">确认完成</view>
            <view v-if="order.status === 'completed'" class="btn-outline" @click="goReview(order.id)">去评价</view>
            <view v-if="order.status === 'pending' || order.status === 'confirmed'" class="btn-primary" @click="goChat(order.id)">联系摄影师</view>
```

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "order/list" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/pages/order/list.vue
git commit -m "feat(order-list): real status API replaces local storage writes"
```

---

### Task 7: 前端 booking/index.vue — 时间槽真实化

**Files:**
- Modify: `src/pages/booking/index.vue`

**Interfaces:**
- Consumes: `apiGet`；`GET /api/v1/photographers/:id/timeslots?date=` 返回 `{date, occupied: string[]}`
- Produces: `loadTimeSlots(date)` 真实请求填充 `disabledTimes`；删除硬编码 `['10:00','15:00']` 与 `generateTimeSlots` 静态列表由前端生成（保留生成静态 8 时段 + 后端占用禁用）

- [ ] **Step 1: 替换 loadTimeSlots + 移除硬编码**

```ts
const ALL_SLOTS = ['09:00','10:00','11:00','13:00','14:00','15:00','16:00','17:00']

async function loadTimeSlots(date: string) {
  timeSlots.value = ALL_SLOTS
  disabledTimes.value = []
  if (!photographerId.value) return
  try {
    const res = await apiGet<{ occupied: string[] }>(`/v1/photographers/${photographerId.value}/timeslots`, { date })
    disabledTimes.value = res.occupied || []
  } catch {
    // 后端不可达时不展示禁用状态（不阻塞预约提交——后端仍会做冲突兜底）
  }
}
```

删除 `generateTimeSlots` 函数；`onMounted` 中 `timeSlots.value = generateTimeSlots(...)` 改为 `loadTimeSlots(selectedDate.value)`（async 调用不 await 亦可）；`onDateChange` 中同样改调 `loadTimeSlots(e.detail.value)`。

- [ ] **Step 2: submitBooking 错误处理补 409**

将 catch 分支改为：

```ts
  } catch (e) {
    uni.hideLoading()
    const code = e instanceof ApiError ? e.status : 0
    uni.showToast({ title: code === 409 ? '该时段已被预约' : '预约失败', icon: 'none' })
    return
  }
```

确保 `ApiError` 已 import（若未 import 加 `import { apiGet, apiPost, ApiError } from '@/api/client'`）。

- [ ] **Step 3: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "booking" | head -5`
Expected: 无新错误

- [ ] **Step 4: Commit**

```bash
git add src/pages/booking/index.vue
git commit -m "feat(booking): real occupied timeslots + 409 conflict toast"
```

---

### Task 8: 全链路验证 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: 后端冒烟（完整状态流转链路）**

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
# 创建
BID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/bookings -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"photographerId":1,"coserId":1,"serviceId":1,"date":"2026-09-05","time":"14:00"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['id'])")
echo "BID=$BID"
# 冲突
curl -s -o /dev/null -w "dup: %{http_code}\n" -X POST http://127.0.0.1:8080/api/v1/bookings -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"photographerId":1,"coserId":1,"serviceId":1,"date":"2026-09-05","time":"14:00"}'
# 流转 pending→confirmed→completed
curl -s -o /dev/null -w "confirm: %{http_code}\n" -X PUT http://127.0.0.1:8080/api/v1/bookings/$BID/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"confirmed"}'
curl -s -o /dev/null -w "complete: %{http_code}\n" -X PUT http://127.0.0.1:8080/api/v1/bookings/$BID/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"completed"}'
curl -s -o /dev/null -w "invalid: %{http_code}\n" -X PUT http://127.0.0.1:8080/api/v1/bookings/$BID/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"pending"}'
# 列表联查
curl -s http://127.0.0.1:8080/api/v1/bookings/1 -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); b=d[0] if d else {}; print('name:', b.get('photographerName'), '| svc:', b.get('serviceName'), '| status:', b.get('status'))"
```
Expected: dup 409; confirm 200; complete 200; invalid 409; 列表含 photographerName/serviceName

- [ ] **Step 2: Playwright H5 实测**

预约页选时间 → 提交 → 订单列表出现 → 确认接单（演示）→ 确认完成 → 去评价入口可见；重复时段显示禁用。
（复用 chromium: `/home/Haxlock/.cache/ms-playwright/chromium-1234/chrome-linux64/chrome`，NODE_PATH=/home/Haxlock/.npm/_npx/e41f203b7505f1fb/node_modules）

- [ ] **Step 3: 回归**

Run: `cd server && go test ./... 2>&1 | tail -5`（全绿）；`npx vue-tsc --noEmit 2>&1 | grep -cE "error"`（≤1，仅 profile/index.vue pre-existing）

- [ ] **Step 4: AGENTS.md 更新**

在 COMMANDS 后新增"订单状态机"小节：PUT /bookings/:id/status（状态迁移表）、GET /photographers/:id/timeslots、冲突检测 409、联查字段 photographerName/avatar/serviceName、前端 apiPut。

- [ ] **Step 5: 提交推送**

```bash
git add AGENTS.md
git commit -m "docs: AGENTS.md — order state machine + timeslots + conflict detection"
git -c http.proxy= -c https.proxy= push gitee master
```
Expected: push 成功，远端含全部 8 个 commit

---

## 自审记录（writing-plans 完成后）

- **Spec 覆盖**: 3.1 状态机 → Task 2/3；3.2 冲突检测+timeslots → Task 1/3/7；3.3 联查 → Task 1/2；3.4 前端真实化 → Task 4/5/6；里程碑 M1-M3 → Task 1-8。spec 的"确认接单（演示）"按钮规则 → Task 5/6 模板。
- **占位符**: 无 TBD/TODO；每个任务含具体代码与命令。
- **类型一致**: `apiPut` 签名 Task 4 定义，Task 5/6 使用；`canTransition`/`ErrConflict`/`ErrInvalidTransition`/`ErrBookingNotFound` Task 2 定义，Task 3 handler 使用；`GetOccupiedTimes` Task 2 定义，Task 3 的 PhotographerHandler.TimeSlots 使用；`BookingWithDetails` Task 1 定义，Task 2 使用；`NewPhotographerHandler` 参数变更在 Task 3 明确。
- **已知取舍**: TimeSlots 路由挂在 photographer 资源下但由 BookingService 提供数据——PhotographerHandler 增 bookingSvc 字段（Task 3 已写明）；demo 允许 coser 直接 confirmed（spec 假设 2）。
