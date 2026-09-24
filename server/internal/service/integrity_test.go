package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Maluslock/comic/server/internal/repository"
)

// --- 1. 过去档期：纯函数，表驱动 ---

func TestIsPastSlot(t *testing.T) {
	now := time.Date(2026, 9, 24, 15, 30, 0, 0, time.Local)
	mk := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("bad fixture date %q: %v", s, err)
		}
		return d
	}
	cases := []struct {
		name  string
		date  string
		clock string
		want  bool
	}{
		{"昨天", "2026-09-23", "10:00", true},
		{"去年", "2025-01-01", "23:00", true},
		{"今天已经过去的时刻", "2026-09-24", "09:00", true},
		{"今天正好等于现在（不早于 → 放行）", "2026-09-24", "15:30", false},
		{"今天稍后", "2026-09-24", "16:00", false},
		{"明天零点", "2026-09-25", "00:00", false},
		{"时刻不可解析：今天放行（不误杀）", "2026-09-24", "下午", false},
		{"时刻不可解析：昨天仍拒", "2026-09-23", "下午", true},
		{"时刻带秒", "2026-09-24", "16:00:00", false},
	}
	for _, c := range cases {
		if got := isPastSlot(mk(c.date), c.clock, now); got != c.want {
			t.Errorf("%s: isPastSlot(%s %s) = %v, want %v", c.name, c.date, c.clock, got, c.want)
		}
	}
}

// --- 2. 下单：过去日期 / 自成交 / 唯一冲突 ---

// pastDateNoDBCall 证明日期校验发生在任何 DB 访问之前（否则会白白打库）。
func TestCreateBooking_RejectsPastDateBeforeTouchingDB(t *testing.T) {
	db := &bookingRecorder{}
	svc := NewBookingService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 1, Date: "2020-01-01", Time: "10:00",
	})
	if !errors.Is(err, ErrPastDate) {
		t.Fatalf("want ErrPastDate, got %v", err)
	}
	if len(db.args) != 0 {
		t.Errorf("past-date rejection must not hit the database, got %d QueryRow calls", len(db.args))
	}
}

func TestCreateBooking_RejectsSelfBooking(t *testing.T) {
	coser := int64(5)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{values: photographerScanValues(4, &coser)}, // 摄影师 4 的 owner 就是下单人自己
	}}
	svc := NewBookingService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 4, CoserID: 5, ServiceID: 1, Date: "2099-01-01", Time: "10:00",
	})
	if !errors.Is(err, ErrSelfBooking) {
		t.Fatalf("want ErrSelfBooking, got %v", err)
	}
}

// 并发下两个请求都通过了 SELECT 冲突检查时，DB 的唯一约束会在 INSERT 处挡下第二个 ——
// 必须仍然映射成 409（ErrConflict），而不是 500。
func TestCreateBooking_UniqueViolationBecomesConflict(t *testing.T) {
	owner := int64(2)
	dup := &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "uniq_bookings_active_slot"`}
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}}, // 冲突检查通过（TOCTOU 的另一半）
		fakeRow{values: []any{owner}},
		fakeRow{values: []any{int64(1), "基础套餐", int32Ptr(399), (*string)(nil), int32(120)}},
		fakeRow{err: dup}, // INSERT 被唯一索引挡下
	}}
	svc := NewBookingService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 1, Date: "2099-01-01", Time: "10:00",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict for a unique violation, got %v", err)
	}
}

// --- 3. 评价：必须对应已完成订单 / 不能自评 / 不能重复 ---

func TestReviewCreate_RequiresCompletedBooking(t *testing.T) {
	db := &fakeDBTX{rows: []pgx.Row{fakeRow{values: []any{false}}}}
	svc := NewReviewService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateReviewRequest{PhotographerID: 1, UserID: 6, Rating: 5})
	if !errors.Is(err, ErrReviewNotAllowed) {
		t.Fatalf("want ErrReviewNotAllowed without a completed booking, got %v", err)
	}
}

func TestReviewCreate_RejectsSelfReview(t *testing.T) {
	owner := int64(6)
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: []any{true}}, // 有一单已完成（自成交被单独拦住了，这里防御性再挡一次）
		fakeRow{values: photographerScanValues(1, &owner)},
	}}
	svc := NewReviewService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateReviewRequest{PhotographerID: 1, UserID: 6, Rating: 5})
	if !errors.Is(err, ErrReviewSelf) {
		t.Fatalf("want ErrReviewSelf, got %v", err)
	}
}

func TestReviewCreate_DuplicateBecomesConflict(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "uniq_reviews_user_photographer"`}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: []any{true}},
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{err: dup},
	}}
	svc := NewReviewService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateReviewRequest{PhotographerID: 1, UserID: 6, Rating: 5})
	if !errors.Is(err, ErrReviewDuplicate) {
		t.Fatalf("want ErrReviewDuplicate, got %v", err)
	}
}

// 成功路径必须触发评分重算 —— 否则评分/评价数还是那对没人维护的死数字。
func TestReviewCreate_RecomputesPhotographerRating(t *testing.T) {
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: []any{true}},
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: reviewScanValues(7, 1, 6, 4)},
	}}
	svc := NewReviewService(repository.New(db))

	if _, err := svc.Create(context.Background(), CreateReviewRequest{PhotographerID: 1, UserID: 6, Rating: 4}); err != nil {
		t.Fatalf("Create unexpected error: %v", err)
	}
	if len(db.execArgs) != 1 {
		t.Errorf("want 1 Exec (rating recompute), got %d", len(db.execArgs))
	}
}

// --- 4. 聊天：非成员不得标记已读 ---

func TestMarkSessionRead_RejectsNonParticipant(t *testing.T) {
	db := &fakeDBTX{rows: []pgx.Row{fakeRow{values: []any{int64(1), int64(5)}}}}
	svc := NewChatService(repository.New(db))

	// 用户 6 不在会话 1（参与者是 1 和 5）里。
	if err := svc.MarkSessionRead(context.Background(), 1, 6); !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden for a non-participant, got %v", err)
	}
	if len(db.execArgs) != 0 {
		t.Errorf("a rejected mark-read must not write, got %d Exec", len(db.execArgs))
	}
}

// --- 测试替身 ---

// photographerScanValues 是 GetPhotographerById 的扫描顺序（12 列 + tags 数组）。
func photographerScanValues(id int32, userID *int64) []any {
	return []any{
		id, "摄影师", (*string)(nil), (*string)(nil), (*string)(nil),
		pgtype.Numeric{}, // rating 不参与断言，只看状态位
		(*int32)(nil), (*int32)(nil), userID, "both", false, (*string)(nil),
		[]string{},
	}
}

// reviewScanValues 是 CreateReview 的扫描顺序（9 列）。
func reviewScanValues(id int64, photographerID, userID, rating int32) []any {
	return []any{
		id, photographerID, userID, (*string)(nil), (*string)(nil), rating,
		(*string)(nil), []string{}, time.Now(),
	}
}
