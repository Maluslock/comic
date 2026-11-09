package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

// sentinelServiceScanValues 构造 GetActiveServicesByPhotographer 的一行扫描值
// （扫描序：id, name, price, description, duration, price_unit, photo_count）。
func sentinelServiceScanValues(id int64, name string, price *int32) []any {
	return []any{id, name, price, (*string)(nil), int32(120), "fixed", (*int32)(nil)}
}

// sentinelPassRows 构造 serviceId=0 放行场景的 QueryRow 序列：
// 摄影师档案 -> 时段冲突 0 -> 落库 -> 通知（失败被吞）。
func sentinelPassRows(mode string) []pgx.Row {
	profile := photographerScanValues(2, nil)
	profile[9] = mode
	return []pgx.Row{
		fakeRow{values: profile},
		fakeRow{values: []any{int64(0)}}, // CountConflictBookings
		fakeRow{values: bookingScanValues(13, "pending")},
		fakeRow{err: errors.New("no photographer profile")}, // notifyPhotographer
	}
}

// sentinelRejectRows 构造 serviceId=0 拒绝场景的 QueryRow 序列：
// 摄影师档案 -> 时段冲突 0 -> （拒后停止，不得有落库调用）。
func sentinelRejectRows(mode string) []pgx.Row {
	profile := photographerScanValues(2, nil)
	profile[9] = mode
	return []pgx.Row{
		fakeRow{values: profile},
		fakeRow{values: []any{int64(0)}}, // CountConflictBookings
	}
}

// TestMutualSentinel_Matrix_FreeZeroPackages：free + 0 上架套餐 + serviceId=0 -> 放行。
func TestMutualSentinel_Matrix_FreeZeroPackages(t *testing.T) {
	db := &bookingRecorder{
		rows:      sentinelPassRows("free"),
		queryRows: []pgx.Rows{&fakeRows{}}, // 0 个上架套餐
	}
	svc := NewBookingService(repository.New(db))
	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 0, Date: futureDate(3), Time: "09:00",
	}); err != nil {
		t.Fatalf("free+0套餐 unexpected error: %v", err)
	}
	assertMutualSentinelPricing(t, db)
}

// TestMutualSentinel_Matrix_BothZeroPackages：both + 0 上架套餐 + serviceId=0 -> 放行（本卡核心）。
// 详情页对 both+0 套餐注入虚拟互勉套餐（isMutualMode 认 both），下单必须放行，
// 否则详情承诺可约、提交 404 自相矛盾。
func TestMutualSentinel_Matrix_BothZeroPackages(t *testing.T) {
	db := &bookingRecorder{
		rows:      sentinelPassRows("both"),
		queryRows: []pgx.Rows{&fakeRows{}}, // 0 个上架套餐
	}
	svc := NewBookingService(repository.New(db))
	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 0, Date: futureDate(3), Time: "09:00",
	}); err != nil {
		t.Fatalf("both+0套餐 unexpected error: %v", err)
	}
	assertMutualSentinelPricing(t, db)
}

// TestMutualSentinel_Matrix_BothWithPackagesRejected：both + 2 上架套餐 + serviceId=0 -> 拒绝。
// 付款红线回归锁：有付费套餐的 both 绝不能凭空多出 0 元下单入口。
func TestMutualSentinel_Matrix_BothWithPackagesRejected(t *testing.T) {
	price := int32(399)
	db := &bookingRecorder{
		rows: sentinelRejectRows("both"),
		queryRows: []pgx.Rows{&fakeRows{rows: []pgx.Row{
			fakeRow{values: sentinelServiceScanValues(1, "基础套餐", &price)},
			fakeRow{values: sentinelServiceScanValues(2, "精修套餐", &price)},
		}}},
	}
	svc := NewBookingService(repository.New(db))
	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 0, Date: futureDate(1), Time: "09:00",
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("both+2套餐 want ErrInvalidReference, got %v", err)
	}
	if len(db.args) != 2 {
		t.Fatalf("both+2套餐 sentinel must stop before insert, got %d QueryRow calls", len(db.args))
	}
}

// TestMutualSentinel_Matrix_PayZeroPackagesRejected：pay + 0 上架套餐 + serviceId=0 -> 拒绝。
func TestMutualSentinel_Matrix_PayZeroPackagesRejected(t *testing.T) {
	db := &bookingRecorder{
		rows:      sentinelRejectRows("pay"),
		queryRows: []pgx.Rows{&fakeRows{}}, // 0 个上架套餐；pay 模式依然拒绝
	}
	svc := NewBookingService(repository.New(db))
	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 0, Date: futureDate(1), Time: "09:00",
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("pay+0套餐 want ErrInvalidReference, got %v", err)
	}
	if len(db.args) != 2 {
		t.Fatalf("pay+0套餐 sentinel must stop before insert, got %d QueryRow calls", len(db.args))
	}
}

// assertMutualSentinelPricing 校验哨兵放行后落库的互勉定价快照。
func assertMutualSentinelPricing(t *testing.T, db *bookingRecorder) {
	t.Helper()
	if got := createBookingInt32Arg(t, db, 2); got != 0 {
		t.Errorf("sentinel ServiceID = %d, want 0", got)
	}
	if got := createBookingInt32Arg(t, db, 6); got != 0 {
		t.Errorf("sentinel TotalPrice = %d, want 0", got)
	}
	if got := createBookingArg(t, db, 8); got != "mutual" {
		t.Errorf("sentinel PriceMode = %v, want mutual", got)
	}
	if got := createBookingArg(t, db, 9); got != "agreed" {
		t.Errorf("sentinel PriceStatus = %v, want agreed", got)
	}
	name, ok := createBookingArg(t, db, 10).(*string)
	if !ok || name == nil || *name != "互勉约拍" {
		t.Errorf("sentinel ServiceName = %v, want 互勉约拍", createBookingArg(t, db, 10))
	}
	duration, ok := createBookingArg(t, db, 11).(*int32)
	if !ok || duration == nil || *duration != 120 {
		t.Errorf("sentinel ServiceDuration = %v, want 120", createBookingArg(t, db, 11))
	}
}
