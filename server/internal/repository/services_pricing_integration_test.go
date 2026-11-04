package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// These exercise the SQL the pricing fixes actually live in:
//
//   - an omitted isActive / sortOrder keeps its stored value (COALESCE), and an
//     explicit one overrides it;
//   - a package that is off the shelf cannot be booked (the is_active guard),
//     while the management lookup still sees it.
//
// The unit tests in internal/service can only pin the parameters handed to the
// driver, so without these the fix itself would be unverified.
//
// Note: the DB-backed tests across this package share integrationQueries, which needs a
// live PostgreSQL and skips when TEST_DATABASE_URL is unset, so
// `go test ./...` stays runnable on a machine without a database.
func integrationQueries(t *testing.T) (*Queries, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed pricing test")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return New(conn), ctx
}

func insertPricingFixture(t *testing.T, q *Queries, ctx context.Context, active bool, order int32) int64 {
	t.Helper()
	var id int64
	err := q.db.QueryRow(ctx,
		`INSERT INTO services (name, price, description, duration, photographer_id, is_active, sort_order)
		 VALUES ('__pricing_it__', 100, '', 60, 1, $1, $2) RETURNING id`,
		active, order).Scan(&id)
	if err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = q.db.Exec(context.Background(), `DELETE FROM services WHERE id = $1`, id)
	})
	return id
}

func TestIntegration_UpdateService_KeepsStoredShelfStateWhenOmitted(t *testing.T) {
	q, ctx := integrationQueries(t)
	id := insertPricingFixture(t, q, ctx, false, 5)

	// An ordinary edit: the management page sends neither isActive nor sortOrder.
	n, err := q.UpdateService(ctx, UpdateServiceParams{
		ID: id, PhotographerID: 1, Name: "__pricing_it_renamed__", Duration: 60,
	})
	if err != nil || n != 1 {
		t.Fatalf("update: n=%d err=%v", n, err)
	}

	var active bool
	var order int32
	if err := q.db.QueryRow(ctx, `SELECT is_active, sort_order FROM services WHERE id = $1`, id).
		Scan(&active, &order); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if active {
		t.Error("is_active must stay false — an edit without isActive used to silently re-publish the package")
	}
	if order != 5 {
		t.Errorf("sort_order must stay 5, got %d — an edit without sortOrder used to reset it", order)
	}

	// An explicit value must still override.
	yes := true
	if _, err := q.UpdateService(ctx, UpdateServiceParams{
		ID: id, PhotographerID: 1, Name: "__pricing_it_renamed__", Duration: 60, IsActive: &yes,
	}); err != nil {
		t.Fatalf("update with explicit isActive: %v", err)
	}
	if err := q.db.QueryRow(ctx, `SELECT is_active FROM services WHERE id = $1`, id).Scan(&active); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !active {
		t.Error("an explicit isActive=true must be applied")
	}
}

func TestIntegration_Bookable_RejectsPackageOffTheShelf(t *testing.T) {
	q, ctx := integrationQueries(t)
	id := insertPricingFixture(t, q, ctx, false, 0)

	// Off the shelf: indistinguishable from missing, so existence is not disclosed.
	if _, err := q.GetBookableServicePhotographerID(ctx, id); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("want pgx.ErrNoRows for an inactive package, got %v", err)
	}
	// The management path must still find it — that is why the two queries differ.
	if _, err := q.GetServicePhotographerID(ctx, id); err != nil {
		t.Errorf("management lookup must still find a downed package, got %v", err)
	}

	// Back on the shelf.
	if _, err := q.db.Exec(ctx, `UPDATE services SET is_active = true WHERE id = $1`, id); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	owner, err := q.GetBookableServicePhotographerID(ctx, id)
	if err != nil {
		t.Fatalf("want the package to be bookable once active, got %v", err)
	}
	if owner == nil || *owner != 1 {
		t.Errorf("want owner 1, got %v", owner)
	}
}

// insertPhotographerFixture creates a throwaway photographer (only `name` is required)
// so the roll-up tests never touch the demo rows.
func insertPhotographerFixture(t *testing.T, q *Queries, ctx context.Context) int64 {
	t.Helper()
	var id int64
	if err := q.db.QueryRow(ctx,
		`INSERT INTO photographers (name) VALUES ('__pricing_it_photographer__') RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("insert photographer fixture: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		// 顺序要紧：bookings/reviews 都指向摄影师，先删子行再删摄影师，否则外键挡住 DELETE，
		// 夹具会永久留在真库里（曾经真的漏掉过一批）。
		_, _ = q.db.Exec(bg, `DELETE FROM bookings WHERE photographer_id = $1`, id)
		_, _ = q.db.Exec(bg, `DELETE FROM reviews WHERE photographer_id = $1`, id)
		_, _ = q.db.Exec(bg, `DELETE FROM services WHERE photographer_id = $1`, id)
		_, _ = q.db.Exec(bg, `DELETE FROM photographers WHERE id = $1`, id)
	})
	return id
}

func insertServiceFixture(t *testing.T, q *Queries, ctx context.Context, photographerID int64, price *int32, active bool) int64 {
	t.Helper()
	return insertServiceFixtureWithUnit(t, q, ctx, photographerID, price, active, "fixed", nil)
}

func insertServiceFixtureWithUnit(t *testing.T, q *Queries, ctx context.Context, photographerID int64, price *int32, active bool, priceUnit string, photoCount *int32) int64 {
	t.Helper()
	var id int64
	if err := q.db.QueryRow(ctx,
		`INSERT INTO services (name, price, description, duration, photographer_id, is_active, price_unit, photo_count)
		 VALUES ('__pricing_it__', $1, '', 60, $2, $3, $4, $5) RETURNING id`,
		price, photographerID, active, priceUnit, photoCount).Scan(&id); err != nil {
		t.Fatalf("insert service fixture: %v", err)
	}
	return id
}

// 全是面议（price IS NULL）时 min_price 这一列整体为 NULL。这曾经让列表接口直接 500
// ——「摄影师只挂面议套餐」是完全合法的用法，所以这条必须钉住。
func TestIntegration_GetServicePriceSummaries_AllNegotiable(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	insertServiceFixture(t, q, ctx, pid, nil, true)
	insertServiceFixture(t, q, ctx, pid, nil, true)
	insertServiceFixture(t, q, ctx, pid, nil, false) // 下架不计入

	sums, err := q.GetServicePriceSummaries(ctx, []int64{pid})
	if err != nil {
		t.Fatalf("GetServicePriceSummaries: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("want 1 summary row, got %d", len(sums))
	}
	s := sums[0]
	if s.PhotographerID != pid {
		t.Errorf("want photographer %d, got %d", pid, s.PhotographerID)
	}
	if s.MinPrice != nil {
		t.Errorf("want NULL min price when every package is negotiable, got %d", *s.MinPrice)
	}
	if s.HasFree {
		t.Error("want hasFree=false (no zero-priced package)")
	}
	if !s.HasNegotiable {
		t.Error("want hasNegotiable=true")
	}
	if s.ActiveCount != 2 {
		t.Errorf("want activeCount 2 (downed package excluded), got %d", s.ActiveCount)
	}
}

// 互勉（0 元）不能把最低价拉成 0，也不能漏掉 hasFree。
func TestIntegration_GetServicePriceSummaries_FreeDoesNotLowerMinPrice(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	zero := int32(0)
	cheap := int32(399)
	insertServiceFixture(t, q, ctx, pid, &zero, true)
	insertServiceFixture(t, q, ctx, pid, &cheap, true)

	sums, err := q.GetServicePriceSummaries(ctx, []int64{pid})
	if err != nil {
		t.Fatalf("GetServicePriceSummaries: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("want 1 summary row, got %d", len(sums))
	}
	s := sums[0]
	if !s.HasFree {
		t.Error("want hasFree=true")
	}
	if s.MinPrice == nil || *s.MinPrice != 399 {
		t.Errorf("want min price 399 (the 0-priced package must not win), got %v", s.MinPrice)
	}
	if s.HasNegotiable {
		t.Error("want hasNegotiable=false")
	}
	if s.ActiveCount != 2 {
		t.Errorf("want activeCount 2, got %d", s.ActiveCount)
	}
}

// 按张计价绝不能污染「¥399 起」：¥10/张 与 ¥399 一整套的量纲不同，并进 min_price
// 会让列表显示「¥10 起」，点进去最低消费其实是 399（或 10×N）。这条用例就是
// 那个 bug 的回归锁。
func TestIntegration_GetServicePriceSummaries_PerPhotoDoesNotPolluteMinPrice(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	flat := int32(399)
	perShot := int32(10)
	insertServiceFixtureWithUnit(t, q, ctx, pid, &flat, true, "fixed", nil)
	insertServiceFixtureWithUnit(t, q, ctx, pid, &perShot, true, "per_photo", nil)

	sums, err := q.GetServicePriceSummaries(ctx, []int64{pid})
	if err != nil {
		t.Fatalf("GetServicePriceSummaries: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("want 1 summary row, got %d", len(sums))
	}
	s := sums[0]
	if s.MinPrice == nil || *s.MinPrice != 399 {
		t.Errorf("min_price must stay 399 (整套价 only), got %v", s.MinPrice)
	}
	if s.MinPricePerPhoto == nil || *s.MinPricePerPhoto != 10 {
		t.Errorf("min_price_per_photo must be 10, got %v", s.MinPricePerPhoto)
	}
	if s.HasNegotiable || s.HasFree {
		t.Errorf("unexpected flags: %+v", s)
	}
	if s.ActiveCount != 2 {
		t.Errorf("activeCount = %d, want 2", s.ActiveCount)
	}
}

// 只有按张套餐时，min_price 必须整体为 NULL（而不是 10）—— 前端据此显示
// 「¥10/张」而不是「¥10 起」。
func TestIntegration_GetServicePriceSummaries_PerPhotoOnlyLeavesMinPriceNull(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	perShot := int32(10)
	insertServiceFixtureWithUnit(t, q, ctx, pid, &perShot, true, "per_photo", nil)

	sums, err := q.GetServicePriceSummaries(ctx, []int64{pid})
	if err != nil {
		t.Fatalf("GetServicePriceSummaries: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("want 1 summary row, got %d", len(sums))
	}
	s := sums[0]
	if s.MinPrice != nil {
		t.Errorf("min_price must be NULL when only per-photo packages exist, got %d", *s.MinPrice)
	}
	if s.MinPricePerPhoto == nil || *s.MinPricePerPhoto != 10 {
		t.Errorf("min_price_per_photo = %v, want 10", s.MinPricePerPhoto)
	}
}

// 时价/日价暂不参与任何「起价」展示：它们既不进 min_price 也不进 min_price_per_photo，
// 但仍然是上架套餐（计入 active_count）。
func TestIntegration_GetServicePriceSummaries_HourlyPriceNotShownAsMinPrice(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	hourly := int32(80)
	insertServiceFixtureWithUnit(t, q, ctx, pid, &hourly, true, "per_hour", nil)

	sums, err := q.GetServicePriceSummaries(ctx, []int64{pid})
	if err != nil {
		t.Fatalf("GetServicePriceSummaries: %v", err)
	}
	if len(sums) != 1 {
		t.Fatalf("want 1 summary row, got %d", len(sums))
	}
	s := sums[0]
	if s.MinPrice != nil || s.MinPricePerPhoto != nil {
		t.Errorf("hourly price must not masquerade as a package starting price: %+v", s)
	}
	if s.ActiveCount != 1 {
		t.Errorf("activeCount = %d, want 1", s.ActiveCount)
	}
}

// 新列必须真的被读出来 —— UpdateService 的 COALESCE 语义（省略则保持原值）也要覆盖到
// price_unit/photo_count，否则老客户端编辑套餐名会把按张套餐悄悄降级成整套价。
func TestIntegration_ServiceRoundTripsPriceUnit(t *testing.T) {
	q, ctx := integrationQueries(t)
	var pid int64
	if err := q.db.QueryRow(ctx, `INSERT INTO photographers (name) VALUES ('__pricing_it_photographer__') RETURNING id`).Scan(&pid); err != nil {
		t.Fatalf("insert photographer fixture: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = q.db.Exec(bg, `DELETE FROM bookings WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM reviews WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM services WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM photographers WHERE id = $1`, pid)
	})

	price := int32(10)
	shots := int32(3)
	id, err := q.InsertService(ctx, InsertServiceParams{
		Name: "__pricing_it_per_photo__", Price: &price, Duration: 30,
		PhotographerID: pid, IsActive: true, SortOrder: 0,
		PriceUnit: "per_photo", PhotoCount: &shots,
	})
	if err != nil {
		t.Fatalf("insert per_photo service: %v", err)
	}

	got, err := q.GetServiceById(ctx, id)
	if err != nil {
		t.Fatalf("GetServiceById: %v", err)
	}
	if got.PriceUnit != "per_photo" {
		t.Errorf("PriceUnit = %q, want per_photo", got.PriceUnit)
	}
	if got.PhotoCount == nil || *got.PhotoCount != 3 {
		t.Errorf("PhotoCount = %v, want 3", got.PhotoCount)
	}

	// 编辑时不带 priceUnit/photoCount：两列都必须保持原值。
	n, err := q.UpdateService(ctx, UpdateServiceParams{
		ID: id, PhotographerID: pid, Name: "__pricing_it_per_photo_renamed__", Price: &price, Duration: 30,
	})
	if err != nil || n != 1 {
		t.Fatalf("update: n=%d err=%v", n, err)
	}
	again, err := q.GetServiceById(ctx, id)
	if err != nil {
		t.Fatalf("GetServiceById after update: %v", err)
	}
	if again.PriceUnit != "per_photo" {
		t.Errorf("PriceUnit after edit = %q, want per_photo kept", again.PriceUnit)
	}
	if again.PhotoCount == nil || *again.PhotoCount != 3 {
		t.Errorf("PhotoCount after edit = %v, want 3 kept", again.PhotoCount)
	}

	// 显式改回整套价时必须能改成功。
	fixed := "fixed"
	if _, err := q.UpdateService(ctx, UpdateServiceParams{
		ID: id, PhotographerID: pid, Name: again.Name, Price: &price, Duration: 30, PriceUnit: &fixed,
	}); err != nil {
		t.Fatalf("switch back to fixed: %v", err)
	}
	back, err := q.GetServiceById(ctx, id)
	if err != nil {
		t.Fatalf("GetServiceById after switch: %v", err)
	}
	if back.PriceUnit != "fixed" {
		t.Errorf("PriceUnit = %q, want fixed", back.PriceUnit)
	}
}

// 数据库层的 CHECK 是最后一道防线：按张 0 元、张数 0 都必须被 DB 拒绝，
// 即使有人绕过 service 层直接写 SQL。
func TestIntegration_PerPhotoConstraintsRejectBadRows(t *testing.T) {
	q, ctx := integrationQueries(t)
	var pid int64
	if err := q.db.QueryRow(ctx, `INSERT INTO photographers (name) VALUES ('__pricing_it_photographer__') RETURNING id`).Scan(&pid); err != nil {
		t.Fatalf("insert photographer fixture: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = q.db.Exec(bg, `DELETE FROM bookings WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM reviews WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM services WHERE photographer_id = $1`, pid)
		_, _ = q.db.Exec(bg, `DELETE FROM photographers WHERE id = $1`, pid)
	})

	zero := int32(0)
	badCount := int32(0)

	if _, err := q.InsertService(ctx, InsertServiceParams{
		Name: "__pricing_it_bad__", Price: &zero, Duration: 30, PhotographerID: pid,
		IsActive: true, PriceUnit: "per_photo",
	}); err == nil {
		t.Error("per_photo with 0 price must violate services_per_photo_price_check")
	}

	if _, err := q.InsertService(ctx, InsertServiceParams{
		Name: "__pricing_it_bad2__", Price: int32PtrLocal(10), Duration: 30, PhotographerID: pid,
		IsActive: true, PriceUnit: "per_photo", PhotoCount: &badCount,
	}); err == nil {
		t.Error("photo_count 0 must violate services_photo_count_check")
	}

	if _, err := q.InsertService(ctx, InsertServiceParams{
		Name: "__pricing_it_bad3__", Price: &zero, Duration: 30, PhotographerID: pid,
		IsActive: true, PriceUnit: "per_kg",
	}); err == nil {
		t.Error("unknown price_unit must violate services_price_unit_check")
	}

	// 0 元整套价依然合法（互勉），别把这条限制扩到 fixed。
	if _, err := q.InsertService(ctx, InsertServiceParams{
		Name: "__pricing_it_mutual__", Price: &zero, Duration: 30, PhotographerID: pid,
		IsActive: true, PriceUnit: "fixed",
	}); err != nil {
		t.Errorf("0-priced fixed package must stay legal: %v", err)
	}
}

func int32PtrLocal(v int32) *int32 { return &v }
