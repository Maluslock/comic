# 摄影师自助定价 + 单轮报价 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让摄影师自助维护套餐与价格（支持互勉/面议），并支持面议套餐的单轮报价（摄影师报价 → coser 接受/拒绝）。

**Architecture:** 复用现有 `services` 表加 `photographer_id` 实现"一表两用"（NULL=平台模板，非 NULL=摄影师套餐），`bookings` 加价格语义与快照字段承载三种定价模式；报价作为 `bookings` 上的状态机扩展（不改现有 `canTransition`）。

**Tech Stack:** Go 1.22（Gin + pgx，手写 repository 层）/ PostgreSQL / Vue3 + uni-app（TS + Pinia）/ SCSS 变量主题。

## Global Constraints

- **Go 版本**：必须用 `GOROOT=/home/user/go-sdk/go1.22`（系统默认 go 是 1.18，会编译失败）
- **门禁命令**（每个任务结束必跑）：
  - 后端：`cd server && go build ./... && go vet ./... && go test ./...`
  - 前端：`npx vue-tsc --noEmit`
- **本仓库不是 git 仓库** → 计划中的"Commit"步骤改为**验证检查点**（跑门禁 + 确认输出）
- **迁移必须幂等**：全部用 `IF NOT EXISTS` / 条件 UPDATE，因 `deploy/migrate.sh` 会全量重放
- **迁移同时应用到 dev 与 prod**：`comic-postgres`（dev）/ `deploy-postgres-1`（prod）
- **设计文档**：`docs/superpowers/specs/2026-09-17-photographer-pricing-design.md`（本计划的唯一需求来源）
- **不可回退的既有行为**：拉黑拦截、时段冲突校验、鉴权先于业务规则（第九轮已实现，改造中不得破坏）
- **命名约定**：repo 层手写 SQL 常量（小驼峰）+ `Queries` 方法；service 层哨兵错误集中定义；handler 放 `internal/handler/`
- **价格语义（唯一口径）**：`price IS NULL`=面议 / `price = 0`=互勉 / `price > 0`=固定价

---

## File Structure

| 文件 | 动作 | 责任 |
|------|------|------|
| `server/migrations/000026_photographer_services.{up,down}.sql` | 创建 | services 加 photographer_id/is_active/sort_order + price 放开 |
| `server/migrations/000027_booking_pricing.{up,down}.sql` | 创建 | bookings 加 price_mode/quote_price/price_status/快照 |
| `server/internal/repository/bookings.sql.go` | 改 | 6 处 SQL + 6 处 Scan 加 5 列；新增报价相关 UPDATE |
| `server/internal/repository/models.go` | 改 | `Booking` 结构体加 5 字段 |
| `server/internal/repository/services.sql.go` | 改 | 按摄影师取套餐 + 套餐增删改 + 归属查询 |
| `server/internal/repository/querier.go` | 改 | 接口同步新增方法 |
| `server/internal/service/photographer_service.go` | 改 | `GetDetail` 取该摄影师套餐；套餐 CRUD；模板列表 |
| `server/internal/service/booking_service.go` | 改 | 下单推导 price_mode + 归属校验 + 快照；报价状态机 |
| `server/internal/handler/photographer_service_handler.go` | 创建 | 套餐 CRUD 4 个端点 |
| `server/internal/handler/booking_handler.go` | 改 | 报价 2 个端点 |
| `server/cmd/api/main.go` | 改 | 注册路由 |
| `server/internal/service/*_test.go` | 改/增 | 同步 fake 行序 + 新增定价/报价单测 |
| `src/types/index.ts` | 改 | `Service.price` 可空；`Booking` 加价格状态 |
| `src/api/index.ts` | 改 | 套餐 CRUD + 报价 API |
| `src/pages/photographer/services.vue` | 创建 | 我的套餐管理 |
| `src/pages/photographer/detail.vue` | 改 | 价格渲染 |
| `src/pages/booking/index.vue` | 改 | 价格渲染 + 面议提示 |
| `src/components/ServiceCard.vue` | 改 | 价格渲染（面议/互勉） |
| `src/pages/photographer/list.vue` | 改 | 新增价格展示 |
| `src/pages/order/list.vue`、`order/detail.vue` | 改 | 报价状态 + 报价/接受/拒绝交互 |
| `src/pages.json` | 改 | 注册 `photographer/services` |
| `server/scripts/smoke.sh` | 改 | 新增套餐/报价用例组 |

---

## Task 1: 数据库迁移（services + bookings 扩展）

**Files:**
- Create: `server/migrations/000026_photographer_services.up.sql` / `.down.sql`
- Create: `server/migrations/000027_booking_pricing.up.sql` / `.down.sql`

**Interfaces:**
- Consumes: 无
- Produces: `services.photographer_id/is_active/sort_order`（`price` 可空）、`bookings.price_mode/quote_price/price_status/service_name/service_duration`

- [ ] **Step 1: 写 000026 up**

```sql
ALTER TABLE services ADD COLUMN IF NOT EXISTS photographer_id BIGINT;
ALTER TABLE services ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE services ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE services ALTER COLUMN price DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_services_photographer ON services(photographer_id);

INSERT INTO services (name, price, description, duration, photographer_id, is_active, sort_order)
SELECT s.name, s.price, s.description, s.duration, p.id, true, s.id
FROM services s
CROSS JOIN photographers p
WHERE s.photographer_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM services x WHERE x.photographer_id = p.id);
```

- [ ] **Step 2: 写 000026 down**

```sql
DELETE FROM services WHERE photographer_id IS NOT NULL;
DROP INDEX IF EXISTS idx_services_photographer;
ALTER TABLE services DROP COLUMN IF EXISTS sort_order;
ALTER TABLE services DROP COLUMN IF EXISTS is_active;
ALTER TABLE services DROP COLUMN IF EXISTS photographer_id;
```

- [ ] **Step 3: 写 000027 up**

```sql
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_mode VARCHAR(10) NOT NULL DEFAULT 'fixed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS quote_price INTEGER;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS price_status VARCHAR(20) NOT NULL DEFAULT 'agreed';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_name VARCHAR(100);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS service_duration INTEGER;

UPDATE bookings b
SET service_name = COALESCE(b.service_name, s.name),
    service_duration = COALESCE(b.service_duration, s.duration)
FROM services s
WHERE s.id = b.service_id AND (b.service_name IS NULL OR b.service_duration IS NULL);
```

- [ ] **Step 4: 写 000027 down**

```sql
ALTER TABLE bookings DROP COLUMN IF EXISTS service_duration;
ALTER TABLE bookings DROP COLUMN IF EXISTS service_name;
ALTER TABLE bookings DROP COLUMN IF EXISTS price_status;
ALTER TABLE bookings DROP COLUMN IF EXISTS quote_price;
ALTER TABLE bookings DROP COLUMN IF EXISTS price_mode;
```

- [ ] **Step 5: 应用到 dev 并验证**

Run:
```bash
cd /home/user/comic/server
docker exec -i comic-postgres psql -U comic -d comic -v ON_ERROR_STOP=1 < migrations/000026_photographer_services.up.sql
docker exec -i comic-postgres psql -U comic -d comic -v ON_ERROR_STOP=1 < migrations/000027_booking_pricing.up.sql
docker exec comic-postgres psql -U comic -d comic -tAc "SELECT column_name,is_nullable FROM information_schema.columns WHERE table_name='services' AND column_name IN ('photographer_id','price','is_active');"
docker exec comic-postgres psql -U comic -d comic -tAc "SELECT count(*) FROM services WHERE photographer_id IS NOT NULL;"
```
Expected: `price|YES`；`photographer_id` 存在；非模板套餐行数 = `摄影师数 × 4`

- [ ] **Step 6: 幂等复验（重放两次）**

Run: 重复 Step 5 的两条 psql，再查 count 不变
Expected: 无报错，count 不增长

> ✅ **已预演验证**（2026-09-17，在一次性库 `mig_test` 上，未触碰 dev/prod）：
> `000026 up` 应用后 → 列全部就位、`price` 可空、每位摄影师复制到 `模板数` 份；**重放 → `INSERT 0 0`**（幂等）。
> `000027 up` 应用后 → 5 列就位、存量订单**全部回填快照**（实测 `UPDATE 2`）、默认 `fixed/agreed`。
> 两个 `down` 回滚后 schema **精确还原**（`services` 回到 6 列、`bookings` 回到 11 列）。
> 临时库已 `DROP`，`pg_database` 计数 0。
> 因此 Task 1 的 SQL 逻辑本身已被证实可用，执行时只需在 dev/prod 实际落库。

---

## Task 2: repository 层——6 处 SQL/Scan 加列 + 套餐查询

**Files:**
- Modify: `server/internal/repository/models.go`（`Booking` 结构体 + **`Service` 结构体**）
- Modify: `server/internal/repository/bookings.sql.go`（6 处 SQL + 6 处 Scan）
- Modify: `server/internal/repository/services.sql.go`（**2 处既有 Scan 改 `Price` 为指针** + 新增查询）
- Modify: `server/internal/repository/querier.go`
- Test: `server/internal/service/admin_manage_service_test.go`（`bookingScanValues`）

**Interfaces:**
- Consumes: Task 1 的列
- Produces:
  - `repository.Booking` 新增 `PriceMode string` / `QuotePrice *int32` / `PriceStatus string` / `ServiceName *string` / `ServiceDuration *int32`
  - **`repository.Service.Price` 由 `int32` 改为 `*int32`**（面议 = NULL；现为值类型，必须改）
  - `(*Queries).GetActiveServicesByPhotographer(ctx, photographerID int32) ([]Service, error)`
  - `(*Queries).GetAllServicesByPhotographer(ctx, photographerID int32) ([]Service, error)`
  - `(*Queries).GetServicePhotographerID(ctx, id int64) (*int64, error)`
  - `(*Queries).InsertService(ctx, arg InsertServiceParams) (int64, error)`
  - `(*Queries).UpdateService(ctx, arg UpdateServiceParams) (int64, error)`
  - `(*Queries).DeleteService(ctx, id, photographerID int64) (int64, error)`
  - `(*Queries).QuoteBooking(ctx, id int64, price int32, photographerID int32) (int64, error)`
  - `(*Queries).RespondBookingQuote(ctx, id int64, coserID int32, accept bool) (int64, error)`

- [ ] **Step 1: 结构体加字段**

`models.go` 的 `Booking` 在 `UpdatedAt time.Time` 前插入：

```go
	PriceMode       string  `json:"price_mode"`
	QuotePrice      *int32  `json:"quote_price"`
	PriceStatus     string  `json:"price_status"`
	ServiceName     *string `json:"service_name"`
	ServiceDuration *int32  `json:"service_duration"`
```

**同时**把 `Service.Price` 改为指针（`models.go:65`）：

```go
type Service struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Price       *int32  `json:"price"`
	Description *string `json:"description"`
	Duration    int32   `json:"duration"`
}
```

并同步 `services.sql.go` 的 **2 处既有 Scan**（行 27 `GetServices`、行 53 `GetServiceById`）：`&i.Price` 无需改动（`scan` 对 `*int32` 目标传 `&i.Price` 即为 `**int32`，pgx 支持 NULL）——**无需改代码，但必须回归验证 `GetServices` 仍能扫出 NULL 与非 NULL 两种行**。

- [ ] **Step 1.5: 验证指针化不破坏既有读取**

Run:
```bash
cd /home/user/comic/server
export GOROOT=/home/user/go-sdk/go1.22; export PATH=$GOROOT/bin:$PATH
go build ./...                                   # 1) 编译
pkill -x comic-api; sleep 1                      # 2) 必须重启，否则 curl 打的是旧进程
CGO_ENABLED=0 go build -tags timetzdata -ldflags="-s -w" -o build/comic-api ./cmd/api
cp build/comic-api /tmp/comic-api
(setsid /tmp/comic-api > /tmp/comic-api.log 2>&1 < /dev/null &) ; sleep 3
curl -s http://127.0.0.1:8088/api/v1/photographers/1 | jq '.services[0].price'   # 3) 验证运行时
```
Expected: 编译通过；**重启后** price 输出为数字（如 `399`）。
> ⚠️ 只跑 `go build` 不重启是无效验证——`curl` 会打到旧进程。（初稿漏了重启，已修正。）

- [ ] **Step 2: 6 处 SQL 补列**

对 `bookings.sql.go` 的 **行 13 / 56 / 95 / 142 / 153 / 188** 六处，在列清单末尾 `updated_at` 后统一追加：

```
, price_mode, quote_price, price_status, service_name, service_duration
```
（行 153/188 是 join 语句，用 `b.` 前缀：`, b.price_mode, b.quote_price, b.price_status, b.service_name, b.service_duration`）

- [ ] **Step 3: 6 处 Scan 同步**

对行 **39 / 71 / 102 / 148 / 172 / 205** 的 `Scan(...)`，在 `&i.UpdatedAt,` 后插入：

```go
		&i.PriceMode,
		&i.QuotePrice,
		&i.PriceStatus,
		&i.ServiceName,
		&i.ServiceDuration,
```
（行 172/205 的 join 变体：插入位置在 `&i.UpdatedAt,` 与后续 join 字段之间）

- [ ] **Step 4: 新增套餐查询**

追加到 `services.sql.go`：

```go
const getServicesByPhotographer = `-- name: GetServicesByPhotographer :many
SELECT id, name, price, description, duration
FROM services
WHERE photographer_id = $1 AND is_active = true
ORDER BY sort_order ASC, id ASC
`

func (q *Queries) GetActiveServicesByPhotographer(ctx context.Context, photographerID int32) ([]Service, error) {
	rows, err := q.db.Query(ctx, getServicesByPhotographer, photographerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Service, 0)
	for rows.Next() {
		var i Service
		if err := rows.Scan(&i.ID, &i.Name, &i.Price, &i.Description, &i.Duration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const getAllServicesByPhotographer = `-- name: GetAllServicesByPhotographer :many
SELECT id, name, price, description, duration
FROM services
WHERE photographer_id = $1
ORDER BY sort_order ASC, id ASC
`

func (q *Queries) GetAllServicesByPhotographer(ctx context.Context, photographerID int32) ([]Service, error) {
	rows, err := q.db.Query(ctx, getAllServicesByPhotographer, photographerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Service, 0)
	for rows.Next() {
		var i Service
		if err := rows.Scan(&i.ID, &i.Name, &i.Price, &i.Description, &i.Duration); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const getServicePhotographerID = `-- name: GetServicePhotographerID :one
SELECT photographer_id FROM services WHERE id = $1
`

func (q *Queries) GetServicePhotographerID(ctx context.Context, id int64) (*int64, error) {
	var pid *int64
	err := q.db.QueryRow(ctx, getServicePhotographerID, id).Scan(&pid)
	return pid, err
}

const insertService = `-- name: InsertService :one
INSERT INTO services (name, price, description, duration, photographer_id, is_active, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`

type InsertServiceParams struct {
	Name           string
	Price          *int32
	Description    *string
	Duration       int32
	PhotographerID int64
	IsActive       bool
	SortOrder      int32
}

func (q *Queries) InsertService(ctx context.Context, arg InsertServiceParams) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, insertService,
		arg.Name, arg.Price, arg.Description, arg.Duration,
		arg.PhotographerID, arg.IsActive, arg.SortOrder,
	).Scan(&id)
	return id, err
}

const updateService = `-- name: UpdateService :execrows
UPDATE services
SET name = $3, price = $4, description = $5, duration = $6, is_active = $7, sort_order = $8
WHERE id = $1 AND photographer_id = $2
`

type UpdateServiceParams struct {
	ID             int64
	PhotographerID int64
	Name           string
	Price          *int32
	Description    *string
	Duration       int32
	IsActive       bool
	SortOrder      int32
}

func (q *Queries) UpdateService(ctx context.Context, arg UpdateServiceParams) (int64, error) {
	tag, err := q.db.Exec(ctx, updateService,
		arg.ID, arg.PhotographerID, arg.Name, arg.Price,
		arg.Description, arg.Duration, arg.IsActive, arg.SortOrder,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const deleteService = `-- name: DeleteService :execrows
DELETE FROM services WHERE id = $1 AND photographer_id = $2
`

func (q *Queries) DeleteService(ctx context.Context, id, photographerID int64) (int64, error) {
	tag, err := q.db.Exec(ctx, deleteService, id, photographerID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
```

- [ ] **Step 5: 报价 UPDATE（原子条件更新）**

追加到 `bookings.sql.go`：

```go
const quoteBooking = `-- name: QuoteBooking :execrows
UPDATE bookings SET quote_price = $2, price_status = 'quoted', updated_at = NOW()
WHERE id = $1 AND photographer_id = $3 AND price_mode = 'negotiable' AND price_status = 'awaiting_quote'
`

func (q *Queries) QuoteBooking(ctx context.Context, id int64, price int32, photographerID int32) (int64, error) {
	tag, err := q.db.Exec(ctx, quoteBooking, id, price, photographerID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const acceptBookingQuote = `-- name: AcceptBookingQuote :execrows
UPDATE bookings SET total_price = quote_price, price_status = 'agreed', status = 'confirmed', updated_at = NOW()
WHERE id = $1 AND coser_id = $2 AND price_status = 'quoted'
`

const rejectBookingQuote = `-- name: RejectBookingQuote :execrows
UPDATE bookings SET price_status = 'rejected', status = 'cancelled', updated_at = NOW()
WHERE id = $1 AND coser_id = $2 AND price_status = 'quoted'
`

func (q *Queries) RespondBookingQuote(ctx context.Context, id int64, coserID int32, accept bool) (int64, error) {
	stmt := rejectBookingQuote
	if accept {
		stmt = acceptBookingQuote
	}
	tag, err := q.db.Exec(ctx, stmt, id, coserID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
```

- [ ] **Step 6: 同步 querier.go**

在接口中追加 Task 2 Produces 列出的全部新方法签名。

- [ ] **Step 7: 同步测试助手**

`server/internal/service/admin_manage_service_test.go:133` 的 `bookingScanValues` 在 `time.Date(...)` 后追加 5 个值：

```go
		"fixed", (*int32)(nil), "agreed", (*string)(nil), (*int32)(nil),
```

- [ ] **Step 8: 跑门禁**

Run: `cd /home/user/comic/server && GOROOT=/home/user/go-sdk/go1.22 PATH=/home/user/go-sdk/go1.22/bin:$PATH go build ./... && go test ./...`
Expected: 全部 PASS（若 `TestSetOrderStatus_*` 报 scan 数量不符，回到 Step 7 补齐）

---

## Task 3: 后端——详情按摄影师取套餐 + 套餐 CRUD

**Files:**
- Modify: `server/internal/service/photographer_service.go`
- Create: `server/internal/handler/photographer_service_handler.go`
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/photographer_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `GetActiveServicesByPhotographer` / `GetAllServicesByPhotographer` / `InsertService` / `UpdateService` / `DeleteService`；既有 `GetPhotographerByUserID` / `ErrForbidden`
- Produces:
  - `ServiceItem` 新增 `Price *int32`（由 `int32` 改为指针）
  - `(*PhotographerService).MyServices(ctx, userID int64) ([]ServiceItem, error)`
  - `(*PhotographerService).CreateService(ctx, userID int64, req ServiceUpsertRequest) (int64, error)`
  - `(*PhotographerService).UpdateService(ctx, userID int64, serviceID int64, req ServiceUpsertRequest) error`
  - `(*PhotographerService).DeleteService(ctx, userID int64, serviceID int64) error`
  - `(*PhotographerService).ListTemplates(ctx) ([]ServiceItem, error)`
  - `type ServiceUpsertRequest struct { Name string; Price *int32; Description string; Duration int32; IsActive bool; SortOrder int32 }`
  - 哨兵：`ErrServiceNotFound`、`ErrServiceForbidden`、`ErrInvalidPrice`

- [ ] **Step 1: 写失败测试**

在 `photographer_service_test.go` 追加：

```go
> ⚠️ **实测校正**（初稿写错，已按真实代码改）：
> - 既有 fake 名是 **`fakeWorksStore`**（不是 `fakeWorkStore`），其 `GetPhotographerByUserID` 返回 **`profile` / `profileErr`** 字段
> - `newWorkTestSvc(store)` = `&PhotographerService{works: store}`（`queries` 为 nil）→ 需要 `queries` 的测试**必须直接构造结构体**
> - `worksStore` 接口**不含**套餐方法 → 套餐查询必须走 `s.queries`，测试需同时提供 `fakeDBTX`
> - `UpdateService`/`DeleteService` 是 `:execrows` → 走 **`Exec`**，`fakeDBTX.Exec` 当前直接返错 → **需先扩展**

`admin_manage_service_test.go` 的 `fakeDBTX` 增加：

```go
type fakeDBTX struct {
	rows     []pgx.Row
	next     int
	execRows int64
}

func (f *fakeDBTX) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", f.execRows)), nil
}
```
（保留既有 `QueryRow`/`Query` 实现不变；`execRows` 默认 0）

- [ ] **Step 1: 写失败测试**

在 `photographer_service_test.go` 追加：

```go
func newServiceTestSvc(store worksStore, db repository.DBTX) *PhotographerService {
	return &PhotographerService{queries: repository.New(db), works: store}
}

func TestCreateService_NotPhotographer(t *testing.T) {
	svc := newServiceTestSvc(&fakeWorksStore{profileErr: pgx.ErrNoRows}, &fakeDBTX{})
	_, err := svc.CreateService(context.Background(), 99, ServiceUpsertRequest{Name: "x", Duration: 60})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestCreateService_NegativePrice(t *testing.T) {
	neg := int32(-1)
	svc := newServiceTestSvc(
		&fakeWorksStore{profile: repository.PhotographerWithTags{ID: 3}},
		&fakeDBTX{},
	)
	_, err := svc.CreateService(context.Background(), 9, ServiceUpsertRequest{Name: "x", Price: &neg, Duration: 60})
	if !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("want ErrInvalidPrice, got %v", err)
	}
}

func TestUpdateService_NotOwner(t *testing.T) {
	owner := int64(42)
	svc := newServiceTestSvc(
		&fakeWorksStore{profile: repository.PhotographerWithTags{ID: 7}},
		&fakeDBTX{
			execRows: 0, // UPDATE 命中 0 行（归属不符）
			rows:     []pgx.Row{fakeRow{values: []any{owner}}}, // GetServicePhotographerID -> 42
		},
	)
	err := svc.UpdateService(context.Background(), 99, 5, ServiceUpsertRequest{Name: "x", Duration: 60})
	if !errors.Is(err, ErrServiceForbidden) {
		t.Fatalf("want ErrServiceForbidden, got %v", err)
	}
}
```

> 注：`repository.DBTX` 是既有接口类型；若未导出，改用 `newServiceTestSvc(store, db any)` + 内部断言，或直接内联构造 `&PhotographerService{queries: repository.New(db), works: store}`。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd /home/user/comic/server && GOROOT=/home/user/go-sdk/go1.22 PATH=$GOROOT/bin:$PATH go test ./internal/service/ -run 'Service' -v`
Expected: 编译失败（方法未定义）

- [ ] **Step 3: 改 `ServiceItem` 与 `mapServiceItems`**

`photographer_service.go`：

```go
type ServiceItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Price       *int32 `json:"price"`
	Description string `json:"description"`
	Duration    int32  `json:"duration"`
}
```
`mapServiceItems` 内部：`Price: s.Price`；`s.Description` 为 `*string` 时用既有 nil 安全取值（参照文件内 `derefString` 等价写法）。

- [ ] **Step 4: 详情改取该摄影师套餐**

`GetDetail` 中把：

```go
	services, err := s.queries.GetServices(ctx)
```
改为：

```go
	services, err := s.queries.GetActiveServicesByPhotographer(ctx, id)
```

- [ ] **Step 5: 实现 5 个 service 方法 + 哨兵**

```go
var (
	ErrServiceNotFound  = errors.New("service not found")
	ErrServiceForbidden = errors.New("cannot modify another photographer's service")
	ErrInvalidPrice     = errors.New("invalid price")
)

type ServiceUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Price       *int32 `json:"price"`
	Description string `json:"description"`
	Duration    int32  `json:"duration"`
	IsActive    *bool  `json:"isActive"`
	SortOrder   int32  `json:"sortOrder"`
}

func (s *PhotographerService) MyServices(ctx context.Context, userID int64) ([]ServiceItem, error) {
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return nil, ErrForbidden
	}
	rows, err := s.queries.GetAllServicesByPhotographer(ctx, int32(p.ID))
	if err != nil {
		return nil, err
	}
	return mapServiceItems(rows), nil
}

func (s *PhotographerService) CreateService(ctx context.Context, userID int64, req ServiceUpsertRequest) (int64, error) {
	if req.Price != nil && *req.Price < 0 {
		return 0, ErrInvalidPrice
	}
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return 0, ErrForbidden
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return s.queries.InsertService(ctx, repository.InsertServiceParams{
		Name:           req.Name,
		Price:          req.Price,
		Description:    strPtr(req.Description),
		Duration:       req.Duration,
		PhotographerID: p.ID,
		IsActive:       active,
		SortOrder:      req.SortOrder,
	})
}

func (s *PhotographerService) UpdateService(ctx context.Context, userID int64, serviceID int64, req ServiceUpsertRequest) error {
	if req.Price != nil && *req.Price < 0 {
		return ErrInvalidPrice
	}
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	n, err := s.queries.UpdateService(ctx, repository.UpdateServiceParams{
		ID:             serviceID,
		PhotographerID: p.ID,
		Name:           req.Name,
		Price:          req.Price,
		Description:    strPtr(req.Description),
		Duration:       req.Duration,
		IsActive:       active,
		SortOrder:      req.SortOrder,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		if pid, err := s.queries.GetServicePhotographerID(ctx, serviceID); err == nil && pid != nil {
			return ErrServiceForbidden
		}
		return ErrServiceNotFound
	}
	return nil
}

func (s *PhotographerService) DeleteService(ctx context.Context, userID int64, serviceID int64) error {
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	n, err := s.queries.DeleteService(ctx, serviceID, p.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		if pid, err := s.queries.GetServicePhotographerID(ctx, serviceID); err == nil && pid != nil {
			return ErrServiceForbidden
		}
		return ErrServiceNotFound
	}
	return nil
}

func (s *PhotographerService) ListTemplates(ctx context.Context) ([]ServiceItem, error) {
	rows, err := s.queries.GetServices(ctx)
	if err != nil {
		return nil, err
	}
	return mapServiceItems(rows), nil
}
```

> 复用**既有** helper（不要新造）：`strPtr(s string) *string`（`event_sync.go:137`，空串→nil）与 `derefString(*string) string`（`home_service.go:267`，nil 安全）。

- [ ] **Step 6: 写 handler**

`photographer_service_handler.go`：

```go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type PhotographerServiceHandler struct {
	svc *service.PhotographerService
}

func NewPhotographerServiceHandler(svc *service.PhotographerService) *PhotographerServiceHandler {
	return &PhotographerServiceHandler{svc: svc}
}

func (h *PhotographerServiceHandler) mapServiceErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
	case errors.Is(err, service.ErrServiceForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify another photographer's service"})
	case errors.Is(err, service.ErrServiceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
	case errors.Is(err, service.ErrInvalidPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be >= 0"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (h *PhotographerServiceHandler) MyServices(c *gin.Context) {
	items, err := h.svc.MyServices(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": items})
}

func (h *PhotographerServiceHandler) Create(c *gin.Context) {
	var req service.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	id, err := h.svc.CreateService(c.Request.Context(), middleware.UserID(c), req)
	if err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *PhotographerServiceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	var req service.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if err := h.svc.UpdateService(c.Request.Context(), middleware.UserID(c), id, req); err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerServiceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), middleware.UserID(c), id); err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerServiceHandler) Templates(c *gin.Context) {
	items, err := h.svc.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}
```

- [ ] **Step 7: 注册路由**

`main.go` 在摄影师相关路由块内追加：

```go
		svcH := handler.NewPhotographerServiceHandler(photographerSvc)
		router.GET("/api/v1/photographers/services/mine", middleware.AuthRequired(userRepo), svcH.MyServices)
		router.POST("/api/v1/photographers/services", middleware.AuthRequired(userRepo), svcH.Create)
		router.PUT("/api/v1/photographers/services/:id", middleware.AuthRequired(userRepo), svcH.Update)
		router.DELETE("/api/v1/photographers/services/:id", middleware.AuthRequired(userRepo), svcH.Delete)
		router.GET("/api/v1/services/templates", svcH.Templates)
```

> ✅ `photographerSvc` **已存在**（`main.go:137`）——直接用，勿新增 `Svc()` 访问器。
> ✅ 路由顺序**已实测可行**：现有 `/photographers/by-user/:userId` 与 `/photographers/:id` 共存且静态段优先（`by-user` 401 / `:id` 200），所以 `services/*` 按同样方式注册即可；仍建议放在 `:id` 之前。

- [ ] **Step 8: 跑测试 + curl 验证**

Run:
```bash
cd /home/user/comic/server && GOROOT=/home/user/go-sdk/go1.22 PATH=$GOROOT/bin:$PATH go build ./... && go test ./...
# 重建 dev API 后：
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:8088/api/v1/photographers/services/mine | jq '.list|length'
curl -s http://127.0.0.1:8088/api/v1/photographers/2 | jq '.services|length'
```
Expected: 我的套餐 = 4（迁移复制的）；摄影师 2 详情 services = 4

---

## Task 4: 后端——下单推导定价模式 + 归属校验 + 快照

**Files:**
- Modify: `server/internal/service/booking_service.go`
- Test: `server/internal/service/booking_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `GetServicePhotographerID` / `GetServiceById`；Task 3 的价格语义
- Produces: `Create` 写入 `price_mode`/`price_status`/`total_price`/快照；归属不符返回 `ErrInvalidReference`

- [ ] **Step 1: 写失败测试**

```go
func TestCreateBooking_NegotiableService(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},                    // 拉黑检查
		fakeRow{values: []any{int64(0)}},                                       // 冲突
		fakeRow{values: []any{int64(1), (*int64)(nil)}},                        // GetServicePhotographerID -> NULL(模板,跳过归属校验用)
		fakeRow{values: []any{int64(1), "面议套餐", (*int32)(nil), (*string)(nil), int32(60)}}, // GetServiceById
		fakeRow{values: bookingScanValues(11, "pending")},                      // CreateBooking
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))
	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 99, Date: "2026-10-20", Time: "10:00",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := createBookingArg(t, db, 6); got != nil {
		t.Errorf("negotiable TotalPrice = %v, want nil", got)
	}
}

func TestCreateBooking_ServiceNotOwnedByPhotographer(t *testing.T) {
	other := int64(42)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{other}}, // 套餐属于摄影师 42
	}}
	svc := NewBookingService(repository.New(db))
	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 99, Date: "2026-10-20", Time: "10:00",
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("want ErrInvalidReference, got %v", err)
	}
}
```

> `createBookingArg(t, db, idx)`：从参数个数=12 的 `CreateBooking` 调用中取第 idx 个实参（现有 `createBookingTotalPriceArg` 的泛化版，参数增至 12 个后索引 6 = TotalPrice）。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/service/ -run TestCreateBooking -v`
Expected: FAIL（`GetServicePhotographerID` 未被调用 / TotalPrice 仍写 0）

- [ ] **Step 3: 实现**

`Create` 中，`GetServiceById` 之后、`CreateBooking` 之前插入：

```go
	serviceID64 := int64(req.ServiceID)
	ownerID, err := s.queries.GetServicePhotographerID(ctx, serviceID64)
	if err != nil {
		return nil, ErrInvalidReference
	}
	if ownerID != nil && int64(req.PhotographerID) != *ownerID {
		return nil, ErrInvalidReference
	}

	var pricePtr *int32
	priceMode := "fixed"
	priceStatus := "agreed"
	var snapName *string
	var snapDuration *int32
	if svc, err := s.queries.GetServiceById(ctx, serviceID64); err == nil {
		pricePtr = svc.Price
		snapDuration = &svc.Duration
		name := svc.Name
		snapName = &name
		switch {
		case svc.Price == nil:
			priceMode, priceStatus = "negotiable", "awaiting_quote"
		case *svc.Price == 0:
			priceMode = "mutual"
		}
	}
```

`CreateBookingParams` 调用补：

```go
		PriceMode:       priceMode,
		PriceStatus:     priceStatus,
		ServiceName:     snapName,
		ServiceDuration: snapDuration,
```
（`TotalPrice` 改为 `pricePtr`）

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/service/ -run TestCreateBooking -v`
Expected: PASS

---

## Task 5: 后端——报价状态机

**Files:**
- Modify: `server/internal/service/booking_service.go`
- Modify: `server/internal/handler/booking_handler.go`
- Modify: `server/cmd/api/main.go`
- Test: `server/internal/service/booking_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `QuoteBooking` / `RespondBookingQuote` / `GetBookingByID`
- Produces:
  - `(*BookingService).Quote(ctx, bookingID int64, actorUserID int64, price int32) (*BookingItem, error)`
  - `(*BookingService).RespondQuote(ctx, bookingID int64, actorUserID int64, accept bool) (*BookingItem, error)`
  - 哨兵复用 `ErrForbidden` / `ErrBookingNotFound` / **`ErrInvalidPrice`（Task 3 已定义，勿重复声明）**；新增 `ErrQuoteNotAllowed`
  - `BookingItem` 新增 `PriceMode string` / `QuotePrice *int32` / `PriceStatus string`

- [ ] **Step 1: 写失败测试**

```go
func TestQuote_NotNegotiable(t *testing.T) {
	// GetBookingByID 返回 price_mode=fixed → 应拒绝
}

func TestQuote_OK(t *testing.T) {
	// 摄影师身份匹配 + price_status=awaiting_quote → RowsAffected=1 → 返回更新后的 item
}

func TestRespondQuote_Accept(t *testing.T) {
	// coser + price_status=quoted + accept=true → status=confirmed, total_price=quote_price
}
```

（每个测试用 `fakeDBTX` 按真实调用顺序给行：`GetBookingByID` → `GetPhotographerByUserID`(摄影师时) → `QuoteBooking`/`RespondBookingQuote` → `GetBookingByID`）

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/service/ -run 'Quote' -v`
Expected: 编译失败（方法未定义）

- [ ] **Step 3: 实现 service**

```go
var (
	ErrQuoteNotAllowed = errors.New("quote not allowed in current state")
)

func (s *BookingService) Quote(ctx context.Context, bookingID int64, actorUserID int64, price int32) (*BookingItem, error) {
	if price <= 0 || price > 99999 {
		return nil, ErrInvalidPrice
	}
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	profile, err := s.queries.GetPhotographerByUserID(ctx, actorUserID)
	if err != nil || int64(profile.ID) != int64(b.PhotographerID) {
		return nil, ErrForbidden
	}
	n, err := s.queries.QuoteBooking(ctx, bookingID, price, b.PhotographerID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrQuoteNotAllowed
	}
	updated, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return bookingToItem(updated), nil
}

func (s *BookingService) RespondQuote(ctx context.Context, bookingID int64, actorUserID int64, accept bool) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if int64(b.CoserID) != actorUserID {
		return nil, ErrForbidden
	}
	n, err := s.queries.RespondBookingQuote(ctx, bookingID, b.CoserID, accept)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrQuoteNotAllowed
	}
	updated, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return bookingToItem(updated), nil
}
```

- [ ] **Step 4: `bookingToItem` 补字段**

```go
		PriceMode:   b.PriceMode,
		QuotePrice:  b.QuotePrice,
		PriceStatus: b.PriceStatus,
```

- [ ] **Step 5: handler + 路由**

`booking_handler.go` 追加：

```go
func (h *BookingHandler) Quote(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Price int32 `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price is required"})
		return
	}
	data, err := h.svc.Quote(c.Request.Context(), id, middleware.UserID(c), req.Price)
	if err != nil {
		h.mapQuoteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) RespondQuote(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	data, err := h.svc.RespondQuote(c.Request.Context(), id, middleware.UserID(c), req.Accept)
	if err != nil {
		h.mapQuoteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) mapQuoteErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, service.ErrQuoteNotAllowed):
		c.JSON(http.StatusConflict, gin.H{"error": "quote not allowed in current state"})
	case errors.Is(err, service.ErrInvalidPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be between 1 and 99999"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
```

`main.go`：

```go
		router.POST("/api/v1/bookings/:id/quote", middleware.AuthRequired(userRepo), bookingH.Quote)
		router.POST("/api/v1/bookings/:id/quote/respond", middleware.AuthRequired(userRepo), bookingH.RespondQuote)
```

- [ ] **Step 6: 跑门禁**

Run: `cd /home/user/comic/server && GOROOT=/home/user/go-sdk/go1.22 PATH=$GOROOT/bin:$PATH go build ./... && go vet ./... && go test ./...`
Expected: 全 PASS

---

## Task 6: 前端——类型 + API + 我的套餐管理页

**Files:**
- Modify: `src/types/index.ts`
- Modify: `src/api/index.ts`
- Create: `src/pages/photographer/services.vue`
- Modify: `src/pages.json`

**Interfaces:**
- Consumes: Task 3 的 5 个端点
- Produces: `interface MyService { id: number; name: string; price: number | null; description: string; duration: number }`；`getMyServices()` / `createService()` / `updateService()` / `deleteService()` / `getServiceTemplates()`

- [ ] **Step 1: 类型改可空 + 新增**

`src/types/index.ts`：

```ts
export interface Service {
  id: string
  name: string
  price: number | null
  description: string
  duration: number
}
```

- [ ] **Step 2: API 层**

`src/api/index.ts`：

```ts
export interface MyService {
  id: number
  name: string
  price: number | null
  description: string
  duration: number
}

export function getMyServices() {
  return apiGet<{ list: MyService[] }>('/v1/photographers/services/mine')
}
export function getServiceTemplates() {
  return apiGet<MyService[]>('/v1/services/templates')
}
export function createService(payload: {
  name: string; price: number | null; description: string; duration: number
}) {
  return apiPost<{ id: number }>('/v1/photographers/services', payload)
}
export function updateService(id: number, payload: {
  name: string; price: number | null; description: string; duration: number
}) {
  return apiPut<{ ok: boolean }>(`/v1/photographers/services/${id}`, payload)
}
export function deleteService(id: number) {
  return apiDelete<{ ok: boolean }>(`/v1/photographers/services/${id}`)
}
```

- [ ] **Step 3: 我的套餐页**

`src/pages/photographer/services.vue`（暗色霓虹，遵循 `$dark-*`/`$neon-*`；结构参照 `photographer/works.vue` 的 list/form 双模式）：
- list 模式：`共 N 个套餐` + `＋ 新增套餐` + `一键预填平台模板`；每项显示 名称 / 价格渲染（`¥399` / `互勉` / `面议`）/ 时长 / 上架状态 + 编辑/删除
- form 模式：名称（必填 ≤50）/ 价格（数字输入，**留空=面议**，`0=互勉`，提示文案说明）/ 时长（分钟）/ 说明（≤200）
- 删除二次确认「删除后不可恢复，历史订单不受影响」
- 403 → toast「仅摄影师可管理套餐」并返回

- [ ] **Step 4: 注册路由 + 入口**

`src/pages.json` 追加：

```json
    {
      "path": "pages/photographer/services",
      "style": { "navigationBarTitleText": "我的套餐" }
    },
```
`src/pages/photographer/activate.vue` 已激活卡加入口：
```html
<view class="btn-services" @click="goServices">套餐管理</view>
```

- [ ] **Step 5: 验证**

Run: `npx vue-tsc --noEmit`（EXIT=0）
再用 agent-browser 打开 `#/pages/photographer/services`，确认列表渲染且新增/删除可用

---

## Task 7: 前端——价格渲染（详情/预约/ServiceCard/列表）

**Files:**
- Modify: `src/components/ServiceCard.vue`
- Modify: `src/pages/photographer/detail.vue`
- Modify: `src/pages/booking/index.vue`
- Modify: `src/pages/photographer/list.vue`

**Interfaces:**
- Consumes: Task 6 的 `Service.price: number | null`
- Produces: 统一价格渲染函数 `formatPrice(price: number | null): string`（返回 `¥399` / `互勉` / `面议`）

- [ ] **Step 1: 抽公共渲染函数**

`src/utils/mappers.ts`（或同目录 utils）追加：

```ts
export function formatPrice(price: number | null | undefined): string {
  if (price === null || price === undefined) return '面议'
  if (price === 0) return '互勉'
  return `¥${price}`
}
```

- [ ] **Step 2: ServiceCard**

`src/components/ServiceCard.vue` 第 9 行 `¥{{ service.price }}` 改为 `{{ formatPrice(service.price) }}` 并 import。

- [ ] **Step 3: 摄影师详情 + 预约页**

两处 `<text class="price">¥{{ service.price }}</text>` 改为 `{{ formatPrice(service.price) }}`。
预约页底部总价：`¥{{ selectedService?.price || 0 }}` 改为：

```html
<text class="total-price">{{ selectedService?.price === null ? '面议' : formatPrice(selectedService?.price) }}</text>
```
并在选择面议套餐时显示提示「提交后等待摄影师报价」。

- [ ] **Step 4: 摄影师列表新增价格**

`photographer/list.vue` 卡片内新增一行价格（规则见 spec 第 7 节）：后端列表接口不返回价格 → **本步先只做占位不做聚合**，改为在卡片不显示价格（YAGNI，避免为列表加聚合查询）。**此项标注为"暂不做"，在计划中显式记录为已评估。**

- [ ] **Step 5: 验证**

Run: `npx vue-tsc --noEmit` + agent-browser 打开摄影师详情与预约页，确认三种价格渲染正确

---

## Task 8: 前端——订单页报价交互

**Files:**
- Modify: `src/pages/order/list.vue`
- Modify: `src/pages/order/detail.vue`
- Modify: `src/types/index.ts`（订单类型加 `priceMode`/`quotePrice`/`priceStatus`）

**Interfaces:**
- Consumes: Task 5 的报价端点；`formatPrice`
- Produces: 面议订单的报价/接受/拒绝交互

- [ ] **Step 1: 订单记录映射补字段**

`order/list.vue` 与 `order/detail.vue` 的映射函数加入：

```ts
      priceMode: b.priceMode || 'fixed',
      quotePrice: b.quotePrice ?? null,
      priceStatus: b.priceStatus || 'agreed',
```

- [ ] **Step 2: 金额渲染按状态**

订单金额渲染改为：

```ts
function renderOrderPrice(o: any): string {
  if (o.priceMode === 'negotiable') {
    if (o.priceStatus === 'awaiting_quote') return '待报价'
    if (o.priceStatus === 'quoted') return `已报价 ¥${o.quotePrice}`
    if (o.priceStatus === 'agreed') return `¥${o.totalPrice}`
    return '已拒绝'
  }
  return formatPrice(o.priceMode === 'mutual' ? 0 : o.totalPrice)
}
```

- [ ] **Step 3: 报价按钮（摄影师）**

摄影师接单面板 `photographer/orders.vue` 与订单详情：当 `priceMode==='negotiable' && priceStatus==='awaiting_quote'` 显示「报价」按钮 → `uni.showModal` 内嵌输入（或跳转输入页）→ `POST /v1/bookings/:id/quote`。
> `uni.showModal` 不支持输入框 → 用 `editable: true` 的 `uni.showModal`（H5/小程序均支持）。

- [ ] **Step 4: 接受/拒绝（coser）**

当 `priceMode==='negotiable' && priceStatus==='quoted' && 是本人订单` 显示「接受报价 / 拒绝」→ `POST /v1/bookings/:id/quote/respond` `{accept}`。

- [ ] **Step 5: 验证**

Run: `npx vue-tsc --noEmit`；agent-browser 走通：摄影师建面议套餐 → coser 下单 → 摄影师报价 → coser 接受 → 订单显示 `¥X` 且状态 confirmed

---

## Task 9: 回归——smoke.sh 新增用例组 + 双环境

**Files:**
- Modify: `server/scripts/smoke.sh`

**Interfaces:**
- Consumes: 全部端点
- Produces: 新增「J. 套餐与报价」用例组

- [ ] **Step 1: 新增用例组 J**

```bash
section "J. 套餐自助定价 + 单轮报价"
# 摄影师：读我的套餐
expect 200 "读我的套餐" -H "Authorization: Bearer $T_P2" "$BASE_URL/api/v1/photographers/services/mine"
# 摄影师：建面议套餐（price=null）
NEW_SVC=$(curl -s -X POST "$BASE_URL/api/v1/photographers/services" -H "Authorization: Bearer $T_P2" \
  -H 'Content-Type: application/json' -d '{"name":"SMOKE-面议","price":null,"description":"","duration":60}')
SVC_ID=$(echo "$NEW_SVC" | jq -r '.id // empty')
expect_true "建面议套餐" "$([ -n "$SVC_ID" ] && echo ok)" "ok"
# 越权：coser 改摄影师的套餐 → 403
expect 403 "非本人改套餐" -X PUT "$BASE_URL/api/v1/photographers/services/$SVC_ID" \
  -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' \
  -d '{"name":"hack","price":1,"description":"","duration":60}'
# 面议下单 → 待报价
QB=$(curl -s -X POST "$BASE_URL/api/v1/bookings" -H "Authorization: Bearer $T_COSER" -H 'Content-Type: application/json' \
  -d "{\"photographerId\":$PID,\"serviceId\":$SVC_ID,\"date\":\"2027-02-01\",\"time\":\"10:00\",\"remarks\":\"SMOKE-nego\"}")
QBID=$(echo "$QB" | jq -r '.id // empty')
expect_true "面议下单 price_status=awaiting_quote" "$(echo "$QB" | jq -r '.priceStatus')" "awaiting_quote"
expect_true "面议下单 totalPrice=null" "$(echo "$QB" | jq -r '.totalPrice // "null"')" "null"
# 报价
expect 200 "摄影师报价" -X POST "$BASE_URL/api/v1/bookings/$QBID/quote" \
  -H "Authorization: Bearer $T_P2" -H 'Content-Type: application/json' -d '{"price":888}'
expect 400 "报价越界" -X POST "$BASE_URL/api/v1/bookings/$QBID/quote" \
  -H "Authorization: Bearer $T_P2" -H 'Content-Type: application/json' -d '{"price":0}'
expect 403 "他摄影师报价" -X POST "$BASE_URL/api/v1/bookings/$QBID/quote" \
  -H "Authorization: Bearer $T_P5" -H 'Content-Type: application/json' -d '{"price":1}'
# 接受
exports=$(curl -s -X POST "$BASE_URL/api/v1/bookings/$QBID/quote/respond" -H "Authorization: Bearer $T_COSER" \
  -H 'Content-Type: application/json' -d '{"accept":true}')
expect_true "接受后 totalPrice=888" "$(echo "$exports" | jq -r '.totalPrice')" "888"
expect_true "接受后 status=confirmed" "$(echo "$exports" | jq -r '.status')" "confirmed"
# 清理
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc \
  "DELETE FROM services WHERE name LIKE 'SMOKE-%';" >/dev/null 2>&1
```

- [ ] **Step 2: 清理函数补 services**

在 `cleanup()` 中追加：

```bash
       DELETE FROM services WHERE name LIKE 'SMOKE-%';
```

- [ ] **Step 3: 双环境跑**

Run:
```bash
cd /home/user/comic
bash server/scripts/smoke.sh
BASE_URL=http://127.0.0.1 DB_CONTAINER=deploy-postgres-1 bash server/scripts/smoke.sh
```
Expected: 两次均 EXIT=0，新增用例全绿

- [ ] **Step 4: 应用迁移到 prod + 重建**

Run:
```bash
cd /home/user/comic
bash deploy/migrate.sh                       # 应 EXIT=0
docker compose -f deploy/docker-compose.prod.yml build api web && docker compose -f deploy/docker-compose.prod.yml up -d
```

---

## Self-Review

**1. Spec coverage**

| spec 章节 | 对应任务 |
|-----------|----------|
| 3.1 services 改造 | Task 1 |
| 3.2 bookings 扩展 | Task 1 |
| 4.1 面议显示 ¥0 | Task 8 Step 2 |
| 4.2 服务名实时读表 | Task 1 Step 3（快照回填）+ Task 4 Step 3（写快照） |
| 4.3 6 处 SQL/Scan | Task 2 Step 2-3 |
| 4.4 测试助手 | Task 2 Step 7 |
| 4.5 price NOT NULL | Task 1 Step 1 |
| 4.6 套餐归属校验 | Task 4 Step 3 |
| 4.7/4.8 无影响点 | 无需任务（已确认不改） |
| 5 状态机 | Task 5 |
| 6.1 套餐 CRUD | Task 3 |
| 6.2 报价 | Task 5 |
| 6.3 改造详情/下单 | Task 3 Step 4 + Task 4 |
| 7 前端改动 | Task 6/7/8 |
| 8 迁移与兼容 | Task 1 + Task 9 Step 4 |
| 9 边界 | Task 3（价格校验）/ Task 4（归属）/ Task 5（409/400） |
| 10 测试策略 | 各任务 TDD + Task 9 |
| 12 实施顺序 | Task 1→9 顺序一致 |

**2. Placeholder scan**：无 TBD/TODO；所有代码步骤含可执行代码。Task 7 Step 4 的"列表页价格"**显式标注为"暂不做"**（YAGNI），非占位。

**3. Type consistency 核对**
- `repository.Service.Price` 实为 **`int32`（值类型）** → 已在 Task 2 Step 1 补上"改为 `*int32`"与回归验证步骤（初稿误写"本就是 `*int32`"，已更正）✓
- `CreateBookingParams` 新增 5 字段名与 Task 2 Step 2 的 SQL 列名一致 ✓
- `bookingScanValues` 追加 5 值与 Task 2 Step 1 结构体字段顺序一致（PriceMode→QuotePrice→PriceStatus→ServiceName→ServiceDuration）✓
- `formatPrice` 在 Task 7 定义，Task 8 复用同一名字 ✓
- `ErrInvalidPrice` 在 Task 3 与 Task 5 **各定义一次** → ⚠️ **冲突**：两者同在 `service` 包。**已修正**：只在 Task 3 定义（文案统一为 `invalid price`），Task 5 只声明 `ErrQuoteNotAllowed` 并复用前者；两处 HTTP 文案由各自 handler 给出。✓

**4. 二次核验（对照真实代码逐条实测，修正 7 处）**

| # | 计划初稿的错 | 实测真相 | 修正 |
|---|--------------|----------|------|
| 1 | `repository.Service.Price` 已是 `*int32` | 实为 `int32`（`models.go:65`） | Task 2 Step 1 补指针化 + 回归验证 |
| 2 | 测试用 `fakeWorkStore` | 真实名为 **`fakeWorksStore`**，返回 `profile`/`profileErr` | Task 3 Step 1 重写 |
| 3 | `NewPhotographerService(queries, store, nil)` | 签名只接 `queries`；fake 经 `works` 字段注入 | 改为直接构造 `&PhotographerService{queries, works}` |
| 4 | 测试用 `s.queries.GetPhotographerByUserID` | 既有 seam 是 `s.workStore()`，测试才能注入 fake | Task 3 的 4 个方法改走 `workStore()` |
| 5 | `:execrows` 测试可直接跑 | `Exec` 走 `fakeDBTX.Exec`，当前**直接返错** | 补 `execRows` 字段 + `Exec` 实现 |
| 6 | 路由用 `photographerH.Svc()` | 无该访问器；`main.go:137` 已有 `photographerSvc` | 直接用既有变量 |
| 7 | 用虚构 helper `strPtrOrNil` | 既有 `strPtr`（空串→nil）/`derefString`（nil 安全） | 改用既有 helper |
| 8 | Task 2 验证步骤只 `go build` 就 curl | 不重启则 curl 打的是**旧进程**，验证不到本次改动 | 补 `pkill` + 重建 + 重启 + 再 curl |

另：路由静态段优先**已实测**（`/photographers/by-user/:userId` → 401，`/photographers/:id` → 200），故 `services/*` 同法注册可行，无需"若冲突则…"的兜底猜测。

---

## 执行前置条件（未满足不得开工）

1. 用户已复核并批准 `docs/superpowers/specs/2026-09-17-photographer-pricing-design.md`
2. 用户已批准本计划
3. 已确认执行方式（subagent-driven / inline）
