# P2a 双角色身份体系 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 闲鱼式双角色身份体系 P2a——人人可一键开通摄影师身份、双身份共存、摄影师接单面板、移除"确认接单（演示）"与角色切换摆设。

**Architecture:** 后端 photographers 表加字段（mode/mutual_intro/certified/activated_at）+ activate/by-user/photographer-bookings API + UpdateStatus 越权校验 + login/me 带 photographerId；前端个人中心真实双身份 + 开通页 + 接单管理页。

**Tech Stack:** Go 1.22 + Gin + pgx；Vue 3 script setup + uni-app。

## Global Constraints

- Go 构建: `export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH`
- PG: `postgres://comic:comic123@127.0.0.1:5433/comic`；重启 API: `kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+')` + `(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &)`；**禁用 pkill -f**
- TypeScript strict，禁新增 `as any`；SCSS 变量/rpx/`@/` 别名；无 emoji 图标
- 身份模型: `photographers.user_id` 绑定 = 该用户是摄影师（1:1）；双身份共存（不切换）
- 认证徽章 = certified 字段（黄V/金V 语义）；蓝V 预留给未来机构——本阶段只做展示位
- 权限校验: UpdateStatus handler 严格校验操作者（coser 只改自己的单；摄影师只改自己收到的单），越权 403
- 每个任务 commit；不 push（编排者统一推 Gitee `git -c http.proxy= -c https.proxy= push gitee master`）
- demo：摄影师账号已 seed（10000000001-4 / 123456）；开通新摄影师 = POST /photographers/activate

---

## 文件结构总览

| 文件 | 职责 |
|------|------|
| `server/migrations/000010_photographer_activation.{up,down}.sql` | photographers 加 mode/mutual_intro/certified/activated_at |
| `server/internal/repository/photographers.sql.go` | InsertPhotographer + GetPhotographerByUserID + GetBookingsByPhotographer（联查 coser） |
| `server/internal/repository/querier.go` | 注册新方法 |
| `server/internal/service/booking_service.go` | ListByPhotographerBooking（photographer 侧订单）+ UpdateStatus 加操作者校验参数 |
| `server/internal/service/auth_service.go` | Login/LoginUser 加 PhotographerID |
| `server/internal/service/photographer_service.go` | Activate（幂等）+ GetByUser + 更新档案 |
| `server/internal/handler/booking_handler.go` | UpdateStatus 接 token user 校验 |
| `server/cmd/api/main.go` | 注册 /photographers/activate + /photographers/by-user + /bookings/photographer/:pid |
| `src/stores/user.ts` / `src/utils/mappers.ts` | mapLoginUser 加 photographerId |
| `src/pages/profile/index.vue` | 移除切换摆设；真实双身份标签；开通/接单管理入口 |
| `src/pages/photographer/activate.vue` | 新页面：一键开通表单 |
| `src/pages/photographer/orders.vue` | 新页面：接单管理 |
| `src/pages/order/list.vue` + `detail.vue` | 移除"确认接单（演示）"（coser 视角） |
| `src/pages/photographer/detail.vue` | mode 标签 + certified 徽章位 |
| `src/pages.json` | 注册 activate/orders 页 |

---

### Task 1: DB 迁移 — photographers 加身份字段

**Files:**
- Create: `server/migrations/000010_photographer_activation.up.sql`
- Create: `server/migrations/000010_photographer_activation.down.sql`

**Interfaces:**
- Produces: photographers 加列 `mode VARCHAR(10) NOT NULL DEFAULT 'free'`、`mutual_intro TEXT`、`certified BOOLEAN NOT NULL DEFAULT false`、`activated_at TIMESTAMPTZ`

- [ ] **Step 1: up 迁移**

```sql
ALTER TABLE photographers ADD COLUMN IF NOT EXISTS mode VARCHAR(10) NOT NULL DEFAULT 'free';
ALTER TABLE photographers ADD COLUMN IF NOT EXISTS mutual_intro TEXT;
ALTER TABLE photographers ADD COLUMN IF NOT EXISTS certified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE photographers ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ;
```

- [ ] **Step 2: down 迁移**

```sql
ALTER TABLE photographers DROP COLUMN IF EXISTS activated_at;
ALTER TABLE photographers DROP COLUMN IF EXISTS certified;
ALTER TABLE photographers DROP COLUMN IF EXISTS mutual_intro;
ALTER TABLE photographers DROP COLUMN IF EXISTS mode;
```

- [ ] **Step 3: 应用 + 验证**

Run: `docker exec -i comic-pg psql -U comic -d comic < server/migrations/000010_photographer_activation.up.sql` 然后 `docker exec comic-pg psql -U comic -d comic -c "\d photographers" | grep -E "mode|mutual|certified|activated"`（4 列出现）

- [ ] **Step 4: Commit**

```bash
git add server/migrations/000010_photographer_activation.up.sql server/migrations/000010_photographer_activation.down.sql
git commit -m "feat(db): photographers activation fields (mode/mutual_intro/certified/activated_at)"
```

---

### Task 2: 后端 repository — insert + by-user + photographer bookings

**Files:**
- Modify: `server/internal/repository/photographers.sql.go`
- Modify: `server/internal/repository/bookings.sql.go`
- Modify: `server/internal/repository/querier.go`

**Interfaces:**
- Consumes: `Queries`（q.db *pgxpool.Pool）
- Produces:
  - `InsertPhotographer(ctx, userID int64, name string, description *string, mode string, mutualIntro *string) (PhotographerWithTags, error)` — INSERT photographers (name/description/mode/mutual_intro/user_id/activated_at) RETURNING 全字段
  - `GetPhotographerByUserID(ctx, userID int64) (PhotographerWithTags, error)` — WHERE user_id=$1
  - `GetBookingsByPhotographer(ctx, photographerID int32) ([]BookingWithCoser, error)` — bookings WHERE photographer_id=$1 JOIN users（coser name/avatar/phone）

- [ ] **Step 1: photographers.sql.go 追加**

```go
const insertPhotographer = `-- name: InsertPhotographer :one
INSERT INTO photographers (name, description, mode, mutual_intro, user_id, activated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING id, name, avatar, description, location, rating, review_count, order_count, user_id
`

type InsertPhotographerParams struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Mode        string  `json:"mode"`
	MutualIntro *string `json:"mutual_intro"`
	UserID      *int64  `json:"user_id"`
}

func (q *Queries) InsertPhotographer(ctx context.Context, arg InsertPhotographerParams) (PhotographerWithTags, error) {
	row := q.db.QueryRow(ctx, insertPhotographer, arg.Name, arg.Description, arg.Mode, arg.MutualIntro, arg.UserID)
	var i PhotographerWithTags
	err := row.Scan(&i.ID, &i.Name, &i.Avatar, &i.Description, &i.Location, &i.Rating, &i.ReviewCount, &i.OrderCount, &i.UserID)
	return i, err
}

const getPhotographerByUserID = `-- name: GetPhotographerByUserID :one
SELECT id, name, avatar, description, location, rating, review_count, order_count, user_id
FROM photographers WHERE user_id = $1
`

func (q *Queries) GetPhotographerByUserID(ctx context.Context, userID int64) (PhotographerWithTags, error) {
	row := q.db.QueryRow(ctx, getPhotographerByUserID, userID)
	var i PhotographerWithTags
	err := row.Scan(&i.ID, &i.Name, &i.Avatar, &i.Description, &i.Location, &i.Rating, &i.ReviewCount, &i.OrderCount, &i.UserID)
	return i, err
}
```

- [ ] **Step 2: bookings.sql.go 追加 BookingWithCoser**

```go
type BookingWithCoser struct {
	Booking
	CoserName   string `json:"coser_name"`
	CoserAvatar string `json:"coser_avatar"`
	CoserPhone  string `json:"coser_phone"`
}

const getBookingsByPhotographer = `-- name: GetBookingsByPhotographer :many
SELECT b.id, b.photographer_id, b.coser_id, b.service_id, b.date, b.time, b.status, b.total_price, b.remarks, b.created_at, b.updated_at,
       COALESCE(u.name, '') AS coser_name, COALESCE(u.avatar, '') AS coser_avatar, COALESCE(u.phone, '') AS coser_phone
FROM bookings b
LEFT JOIN users u ON u.id = b.coser_id
WHERE b.photographer_id = $1
ORDER BY b.created_at DESC
`

func (q *Queries) GetBookingsByPhotographer(ctx context.Context, photographerID int32) ([]BookingWithCoser, error) {
	rows, err := q.db.Query(ctx, getBookingsByPhotographer, photographerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookingWithCoser
	for rows.Next() {
		var i BookingWithCoser
		if err := rows.Scan(&i.ID, &i.PhotographerID, &i.CoserID, &i.ServiceID, &i.Date, &i.Time, &i.Status, &i.TotalPrice, &i.Remarks, &i.CreatedAt, &i.UpdatedAt, &i.CoserName, &i.CoserAvatar, &i.CoserPhone); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
```

- [ ] **Step 3: querier.go 注册**

```go
	// Photographer activation
	InsertPhotographer(ctx context.Context, arg InsertPhotographerParams) (PhotographerWithTags, error)
	GetPhotographerByUserID(ctx context.Context, userID int64) (PhotographerWithTags, error)
	GetBookingsByPhotographer(ctx context.Context, photographerID int32) ([]BookingWithCoser, error)
```

- [ ] **Step 4: 构建验证**

Run: `cd server && go build ./...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add server/internal/repository/photographers.sql.go server/internal/repository/bookings.sql.go server/internal/repository/querier.go
git commit -m "feat(repo): photographer insert/by-user + bookings-by-photographer with coser join"
```

---

### Task 3: 后端 service — activate/by-user/photographer bookings + auth photographerId + 越权校验

**Files:**
- Modify: `server/internal/service/photographer_service.go`
- Modify: `server/internal/service/auth_service.go`
- Modify: `server/internal/service/booking_service.go`

**Interfaces:**
- Consumes: Task 2 repo 方法
- Produces:
  - `PhotographerService.Activate(ctx, userID int64, name string, mode string, intro string) (int64, error)` — 幂等（已存在返回现有 id）+ InsertPhotographer
  - `PhotographerService.GetByUser(ctx, userID int64) (*PhotographerItem, error)` — 无则 nil
  - `BookingService.ListByPhotographer(ctx, photographerID int32) ([]BookingItemWithCoser, error)`
  - `BookingItemWithCoser`（BookingItem 字段 + CoserName/CoserAvatar/CoserPhone）
  - `LoginUser` 加 `PhotographerID *int64 json:"photographerId"` — Login 时查 GetPhotographerByUserID
  - `UpdateStatus(ctx, bookingID, newStatus, actorTag string)` — actorTag 校验：'photographer'（须是 photographer_id 对应档案，userID 匹配 photographer.user_id）或 'coser'（booking.coser_id == actorUserID）；不匹配 → ErrForbidden

- [ ] **Step 1: 错误变量 + BookingItemWithCoser**

booking_service.go 加:

```go
var ErrForbidden = errors.New("forbidden")

type BookingItemWithCoser struct {
	BookingItem
	CoserName   string `json:"coserName"`
	CoserAvatar string `json:"coserAvatar"`
	CoserPhone  string `json:"coserPhone"`
}
```

- [ ] **Step 2: ListByPhotographer**

```go
func (s *BookingService) ListByPhotographer(ctx context.Context, photographerID int32) ([]BookingItemWithCoser, error) {
	rows, err := s.queries.GetBookingsByPhotographer(ctx, photographerID)
	if err != nil {
		return nil, err
	}
	items := make([]BookingItemWithCoser, 0, len(rows))
	for _, b := range rows {
		item := bookingToItem(b.Booking)
		item.PhotographerName = b.CoserName // 误用? 不——photographer 侧看到的是 coser；用专用结构
		items = append(items, BookingItemWithCoser{
			BookingItem: *item,
			CoserName:   b.CoserName,
			CoserAvatar: b.CoserAvatar,
			CoserPhone:  b.CoserPhone,
		})
	}
	return items, nil
}
```

（注：bookingToItem 的 PhotographerName 字段在摄影师视角无意义；结构体含 CoserName 正确。）

- [ ] **Step 3: Activate + GetByUser（photographer_service.go）**

```go
var ErrAlreadyActivated = errors.New("photographer already activated")

func (s *PhotographerService) Activate(ctx context.Context, userID int64, name string, mode string, intro string) (int64, error) {
	existing, err := s.queries.GetPhotographerByUserID(ctx, userID)
	if err == nil {
		return int64(existing.ID), nil // idempotent
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	var desc *string
	if name != "" { desc = &name }
	var introPtr *string
	if intro != "" { introPtr = &intro }
	p, err := s.queries.InsertPhotographer(ctx, repository.InsertPhotographerParams{
		Name: name, Description: desc, Mode: mode, MutualIntro: introPtr, UserID: &userID,
	})
	if err != nil { return 0, err }
	return int64(p.ID), nil
}
```

（description 用 name 简通——若 name 为空则默认"摄影师"；mode 校验 free/pay/both。）

- [ ] **Step 4: auth LoginUser 加 PhotographerID**

auth_service.go `LoginUser` 加字段；Login 里查：

```go
photographerID, _ := s.users.GetPhotographerIDByUser(ctx, u.ID) // 需 UserRepo 加方法 或 queries
```

注：AuthService 现只有 UserRepo——**加查询**：repository/users.sql.go 加 `GetPhotographerByUserID` 委托 queries，或 AuthService 补 queries。**采用**：AuthService 构造加 `queries *repository.Queries`（main.go 更新 NewAuthService 调用），Login 里 `s.queries.GetPhotographerByUserID(ctx, u.ID)`，ErrNoRows → PhotographerID=nil。

```go
type LoginUser struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Avatar        string `json:"avatar"`
	PhotographerID *int64 `json:"photographerId"`
}
```

- [ ] **Step 5: UpdateStatus 越权校验**

booking_service.go UpdateStatus 签名改 `UpdateStatus(ctx, bookingID int64, newStatus string, actorUserID int64, actorTag string)`:
- GetBookingByID 后：
  - actorTag == 'photographer': 查 photographer by user_id 得 pid；pid != booking.PhotographerID → ErrForbidden
  - actorTag == 'coser': booking.CoserID != actorUserID → ErrForbidden
- 其余逻辑不变

- [ ] **Step 6: 构建验证**

Run: `cd server && go build ./...`
Expected: 无错误

- [ ] **Step 7: Commit**

```bash
git add server/internal/service/photographer_service.go server/internal/service/auth_service.go server/internal/service/booking_service.go server/internal/repository/users.sql.go
git commit -m "feat(service): activate/by-user + photographer bookings + auth photographerId + UpdateStatus actor check"
```

---

### Task 4: 后端 handler + 路由

**Files:**
- Modify: `server/internal/handler/photographer_handler.go`
- Modify: `server/internal/handler/booking_handler.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**
- Consumes: Task 3 service 方法
- Produces:
  - `POST /api/v1/photographers/activate`（AuthRequired）body {name, mode, intro} → 201 {photographerId}
  - `GET /api/v1/photographers/by-user/:userId`（AuthRequired）→ PhotographerItem 或 404
  - `GET /api/v1/bookings/photographer/:photographerId`（AuthRequired）→ []BookingItemWithCoser
  - `UpdateStatus` handler 读 `middleware.UserID(c)`（actor）+ body 加 `actorTag?: string`（'photographer'|'coser'，默认 'coser'）

- [ ] **Step 1: photographer_handler.go 加 3 方法**

```go
func (h *PhotographerHandler) Activate(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Mode string `json:"mode" binding:"required"`
		Intro string `json:"intro"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"}); return
	}
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return
	}
	pid, err := h.svc.Activate(c.Request.Context(), userID, req.Name, req.Mode, req.Intro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"}); return
	}
	c.JSON(http.StatusCreated, gin.H{"photographerId": pid})
}

func (h *PhotographerHandler) GetByUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return
	}
	item, err := h.svc.GetByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"}); return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not activated"}); return
	}
	c.JSON(http.StatusOK, item)
}
```

- [ ] **Step 2: booking_handler.go 加 ListByPhotographer + UpdateStatus 越权**

```go
func (h *BookingHandler) ListByPhotographer(c *gin.Context) {
	pid, err := strconv.Atoi(c.Param("photographerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"}); return
	}
	data, err := h.svc.ListByPhotographer(c.Request.Context(), int32(pid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"}); return
	}
	c.JSON(http.StatusOK, data)
}
```

UpdateStatus handler: body 加 actorTag（默认 'coser'），actorUserID = middleware.UserID(c)，传 svc.UpdateStatus；ErrForbidden → 403。

```go
	var req struct {
		Status   string `json:"status" binding:"required"`
		ActorTag string `json:"actorTag"`
	}
	...
	if req.ActorTag == "" { req.ActorTag = "coser" }
	data, err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status, middleware.UserID(c), req.ActorTag)
	...
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
```

- [ ] **Step 3: main.go 注册路由**

```go
		router.POST("/api/v1/photographers/activate", middleware.AuthRequired(userRepo), photographerH.Activate)
		router.GET("/api/v1/photographers/by-user/:userId", middleware.AuthRequired(userRepo), photographerH.GetByUser)
		router.GET("/api/v1/bookings/photographer/:photographerId", middleware.AuthRequired(userRepo), bookingH.ListByPhotographer)
```

auth 区块: `authSvc := service.NewAuthService(userRepo, queries)`（Task 3 Step 4 改的构造）。

- [ ] **Step 4: 构建 + 重启 + 冒烟**

Run:
```bash
export GOROOT=/home/Haxlock/go GOPATH=/home/Haxlock/gopath PATH=/home/Haxlock/go/bin:$PATH
cd server && go build -o /tmp/mila-api ./cmd/api
kill $(ss -tlnp | grep 8080 | grep -oP 'pid=\K[0-9]+'); sleep 2
(setsid /tmp/mila-api > /tmp/mila-api.log 2>&1 &); sleep 3
# 登录 coser
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
echo "me:"; curl -s http://127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
# 开通摄影师
curl -s -X POST http://127.0.0.1:8080/api/v1/photographers/activate -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"name":"测试摄影师","mode":"both","intro":"互勉可约"}'
# by-user
curl -s http://127.0.0.1:8080/api/v1/photographers/by-user/1 -H "Authorization: Bearer $TOKEN"
# 摄影师账号登录
PTOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"10000000001","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
curl -s http://127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $PTOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('photographerId:', d.get('photographerId'))"
# 摄影师订单列表
curl -s http://127.0.0.1:8080/api/v1/bookings/photographer/1 -H "Authorization: Bearer $PTOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('photographer bookings:', len(d))"
```
Expected: me 无 photographerId；activate 201；by-user 返回档案；摄影师 me 带 photographerId=1；bookings/photographer/1 → 列表

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/photographer_handler.go server/internal/handler/booking_handler.go server/cmd/api/main.go
git commit -m "feat(api): activate/by-user/photographer bookings routes + UpdateStatus actor check"
```

---

### Task 5: 前端 — 个人中心双身份 + 开通页 + 接单管理页

**Files:**
- Modify: `src/stores/user.ts` + `src/utils/mappers.ts`（photographerId）
- Modify: `src/pages/profile/index.vue`（移除切换摆设；双身份标签；开通/接单入口）
- Create: `src/pages/photographer/activate.vue`
- Create: `src/pages/photographer/orders.vue`
- Modify: `src/pages.json`

**Interfaces:**
- Consumes: `apiPost('/v1/photographers/activate')`、`apiGet('/v1/photographers/by-user/:userId')`、`apiGet('/v1/bookings/photographer/:pid')`、`apiPut('/v1/bookings/:id/status')`（actorTag）
- Produces: `userStore.user.photographerId?`（mapLoginUser 映射）；个人中心"我是摄影师"入口（未开通）/"接单管理"（已开通）

- [ ] **Step 1: mappers + user store**

mappers.ts `LoginUserDTO` 加 `photographerId?: number`；`mapLoginUser` 加 `photographerId: u.photographerId`。User 类型（types/index.ts）加 `photographerId?: number`。

login 成功后（stores/user.ts）用 me 刷新? ——简化：LoginResponse.user 已带 photographerId（后端 login 返回），mapLoginUser 映射即可。

- [ ] **Step 2: profile 改双身份**

profile/index.vue:
- 删 `displayRole`/`switchRole`/`role-switch` 区块（摆设）
- user-card 角色标签改为：显示 `user.photographerId ? '摄影师' : 'Coser'`（真实身份）+ 若 photographerId 有接单管理入口
- 菜单区加：
  - 未开通（无 photographerId）→ "我是摄影师"（navigateTo activate）
  - 已开通 → "接单管理"（navigateTo photographer/orders）

```ts
const isPhotographer = computed(() => !!userStore.user?.photographerId)
function goActivate() { uni.navigateTo({ url: '/pages/photographer/activate' }) }
function goPhotographerOrders() { uni.navigateTo({ url: '/pages/photographer/orders' }) }
```

- [ ] **Step 3: activate.vue（开通页）**

参照 dark 霓虹表单（booking 页风格）。字段：昵称/风格标签(多选 tags)/简介/接单模式（radio: 互勉 free / 收费 pay / 两者 both）/互勉说明。提交 → apiPost('/v1/photographers/activate', {name, mode, intro}) → 成功 toast + 更新 userStore.user.photographerId + 返回。

```ts
const form = reactive({ name: '', mode: 'both', intro: '' })
async function submit() {
  if (!form.name.trim()) { uni.showToast({ title: '请填写昵称', icon: 'none' }); return }
  uni.showLoading({ title: '提交中...' })
  try {
    const res = await apiPost<{ photographerId: number }>('/v1/photographers/activate', form)
    if (userStore.user) {
      userStore.user.photographerId = res.photographerId
      uni.setStorageSync('user', JSON.stringify(userStore.user))
    }
    uni.showToast({ title: '开通成功', icon: 'success' })
    setTimeout(() => uni.navigateBack(), 800)
  } catch { uni.showToast({ title: '开通失败', icon: 'none' }) }
  finally { uni.hideLoading() }
}
```

- [ ] **Step 4: orders.vue（接单管理）**

onShow 加载 `apiGet('/v1/bookings/photographer/:photographerId')`（photographerId 从 userStore.user.photographerId）；订单卡（coser 名/头像/电话/时间/服务/状态）+ 按钮：pending → [确认接单 actorTag=photographer][拒绝→cancelled actorTag=photographer]；confirmed → [完成→completed actorTag=photographer]。

```ts
async function loadOrders() {
  if (!userStore.user?.photographerId) { orders.value = []; return }
  const res = await apiGet<any[]>(`/v1/bookings/photographer/${userStore.user.photographerId}`)
  orders.value = res || []
}
async function updateStatus(id: string, status: string) {
  await apiPut(`/v1/bookings/${id}/status`, { status, actorTag: 'photographer' })
  loadOrders()
}
```

- [ ] **Step 5: pages.json 注册**

```json
{ "path": "pages/photographer/activate", "style": { "navigationStyle": "custom" } },
{ "path": "pages/photographer/orders", "style": { "navigationStyle": "custom" } }
```

- [ ] **Step 6: 类型检查**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "profile|activate|orders" | head -6`
Expected: 无新错误

- [ ] **Step 7: Commit**

```bash
git add src/stores/user.ts src/utils/mappers.ts src/types/index.ts src/pages/profile/index.vue src/pages/photographer/activate.vue src/pages/photographer/orders.vue src/pages.json
git commit -m "feat(ui): dual-role identity — activate page, photographer orders panel, profile real badges"
```

---

### Task 6: 前端 — 移除演示按钮 + 详情页徽章/模式标签

**Files:**
- Modify: `src/pages/order/list.vue`（移除"确认接单"按钮——coser 视角不该有）
- Modify: `src/pages/order/detail.vue`（移除"确认接单（演示）"）
- Modify: `src/pages/photographer/detail.vue`（mode 标签 + certified 徽章位）

**Interfaces:**
- Consumes: photographer detail API（返回 mode/certified —— 需后端 PhotographerItem 加 mode/certified 或此任务仅前端展示已给字段）
- Produces: coser 视角订单无"确认接单"；详情页显示互勉/收费/认证徽章

- [ ] **Step 1: order/list.vue 移除确认接单**

删模板 `v-if="order.status === 'pending'" @click="confirmAccept"` 按钮行 + script confirmAccept 函数。

- [ ] **Step 2: order/detail.vue 移除**

删"确认接单（演示）"按钮 + confirmAccept 函数。

- [ ] **Step 3: photographer/detail.vue 加 mode/认证徽章**

后端 PhotographerItem 加 `Mode string json:"mode"` + `Certified bool json:"certified"`（Task 3 漏——**补充**：repository PhotographerWithTags 加 Mode/Certified、SQL SELECT 加 mode/certified、service 映射）。然后前端详情页在 tags 区旁显示：

```html
<view v-if="photographer?.mode === 'free'" class="mode-tag">互勉</view>
<view v-else-if="photographer?.mode === 'pay'" class="mode-tag pay">收费</view>
<view v-else class="mode-tag both">互勉/收费</view>
<view v-if="photographer?.certified" class="cert-badge">认证摄影师</view>
```

（后端字段补充纳入此任务：photographers.sql.go 3 处 SELECT 加 `p.mode, p.certified`；PhotographerWithTags 加 Mode/Certified；Scan 加 2 项；PhotographerItem 加 Mode/Certified；mapPhotographers + GetDetail 映射。）

- [ ] **Step 4: 类型检查 + build:h5**

Run: `npx vue-tsc --noEmit 2>&1 | grep -E "order|photographer" | head -6`（无新错误）；`npm run build:h5 2>&1 | tail -2`（成功）

- [ ] **Step 5: Commit**

```bash
git add src/pages/order/list.vue src/pages/order/detail.vue src/pages/photographer/detail.vue server/internal/repository/photographers.sql.go server/internal/repository/models.go server/internal/service/home_service.go server/internal/service/photographer_service.go
git commit -m "feat: remove 确认接单 demo from coser view + photographer mode/certified badge"
```

---

### Task 7: 全链路验证 + 视觉复查 + AGENTS.md + 推送

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: 后端冒烟（完整双角色链路）**

```bash
# coser 开通
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
curl -s -X POST http://127.0.0.1:8080/api/v1/photographers/activate -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"name":"测试摄影师","mode":"both","intro":"互勉可约"}'
curl -s http://127.0.0.1:8080/api/v1/photographers/by-user/1 -H "Authorization: Bearer $TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('by-user:', d.get('name'), d.get('mode'))"
# 摄影师账号看接单
PTOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login -H 'Content-Type: application/json' -d '{"phone":"10000000001","code":"123456"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
curl -s http://127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $PTOKEN" | python3 -c "import json,sys; print('photographerId:', json.load(sys.stdin).get('photographerId'))"
curl -s http://127.0.0.1:8080/api/v1/bookings/photographer/1 -H "Authorization: Bearer $PTOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin); print('orders:', len(d))"
# 越权：coser 试着确认摄影师订单（id=2）→ 403
curl -s -o /dev/null -w "cross-actor: %{http_code}\n" -X PUT http://127.0.0.1:8080/api/v1/bookings/2/status -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"status":"confirmed","actorTag":"photographer"}'
```
Expected: by-user 返回；photographerId=1；orders 有数据；cross-actor 403（coser 不能冒充摄影师确认）

- [ ] **Step 2: Playwright H5 全链路**

coser 登录 → 个人中心"我是摄影师"→ 开通表单提交 → 双身份标签出现 + "接单管理"入口 → 登出 → 摄影师账号登录 → 接单管理看到订单 → 确认接单 → 订单状态变化。coser 视角订单列表无"确认接单（演示）"按钮。

- [ ] **Step 3: 回归**

Run: `cd server && go test ./... 2>&1 | tail -3`（全绿除 pre-existing）；`npx vue-tsc --noEmit 2>&1 | grep -cE "error"`（≤1）；`npm run build:h5 2>&1 | tail -2`

- [ ] **Step 4: 视觉复查**

agent-browser 截图：个人中心双身份、开通页表单、接单管理页、订单列表（coser 视角无确认接单）、摄影师详情页模式标签/徽章位。发现问题列 task 纠正。

- [ ] **Step 5: AGENTS.md 更新**

新增"双角色身份体系（P2a）"：photographers 激活字段 + activate/by-user/photographer-bookings API + UpdateStatus 越权校验 + 前端开通页/接单管理 + click-shot 借鉴说明。

- [ ] **Step 6: 提交推送**

```bash
git add AGENTS.md
git commit -m "docs: AGENTS.md — dual-role identity P2a"
git -c http.proxy= -c https.proxy= push gitee master
```

---

## 自审记录

- **Spec 覆盖**: 3.1 身份模型 → Task 3/5；3.2 DB → Task 1；3.3 API → Task 2/3/4；3.4 前端 → Task 5/6；3.5 权限 → Task 3/4；M3 → Task 7。
- **占位符**: 无；每任务含具体代码/命令。
- **类型一致**: `InsertPhotographer`/`GetPhotographerByUserID`/`GetBookingsByPhotographer` Task 2 定义 → Task 3/4 用；`BookingItemWithCoser` Task 3 定义 → Task 4/5 用；`Activate(ctx, userID, name, mode, intro)` Task 3 → Task 4 handler；`UpdateStatus(..., actorUserID, actorTag)` Task 3 → Task 4/5/6 调用（actorTag）；PhotographerItem 加 Mode/Certified Task 6 补充并同步 SQL/结构。
- **已知取舍**: AuthService 构造加 queries 参数（main.go 同步）；mode 校验 free/pay/both；description 用 name 简通；certified 徽章 P2b 做申请设置（本阶段展示位）。
