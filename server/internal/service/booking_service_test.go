package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
)

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

type bookingRecorder struct {
	rows []pgx.Row
	args [][]any
	next int
}

func (f *bookingRecorder) QueryRow(_ context.Context, _ string, a ...any) pgx.Row {
	f.args = append(f.args, a)
	if f.next >= len(f.rows) {
		return fakeRow{err: errors.New("bookingRecorder: unexpected QueryRow")}
	}
	r := f.rows[f.next]
	f.next++
	return r
}

func (f *bookingRecorder) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("bookingRecorder: Query not expected")
}

func (f *bookingRecorder) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("bookingRecorder: Exec not expected")
}

func createBookingArg(t *testing.T, db *bookingRecorder, idx int) any {
	t.Helper()
	for _, a := range db.args {
		if len(a) == 12 {
			if idx < 0 || idx >= len(a) {
				t.Fatalf("createBookingArg: idx %d out of range for %d args", idx, len(a))
			}
			return a[idx]
		}
	}
	t.Fatalf("CreateBooking call not found among %d QueryRow calls", len(db.args))
	return nil
}

func createBookingInt32Arg(t *testing.T, db *bookingRecorder, idx int) int32 {
	t.Helper()
	v, ok := createBookingArg(t, db, idx).(int32)
	if !ok {
		t.Fatalf("CreateBooking arg[%d] type = %T, want int32", idx, v)
	}
	return v
}

func TestCreateBooking_WritesServicePrice(t *testing.T) {
	owner := int64(2)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{owner}}, // GetServicePhotographerID -> 摄影师 2（本人）
		fakeRow{values: []any{int64(1), "基础套餐", int32Ptr(399), (*string)(nil), int32(120)}},
		fakeRow{values: bookingScanValues(9, "pending")},
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))

	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      1,
		Date:           "2026-10-20",
		Time:           "10:00",
	}); err != nil {
		t.Fatalf("Create unexpected error: %v", err)
	}

	if got := createBookingInt32Arg(t, db, 6); got != 399 {
		t.Errorf("CreateBooking TotalPrice = %d, want 399 (service price)", got)
	}
	if got := createBookingArg(t, db, 8); got != "fixed" {
		t.Errorf("CreateBooking PriceMode = %v, want fixed", got)
	}
	name, ok := createBookingArg(t, db, 10).(*string)
	if !ok || name == nil || *name != "基础套餐" {
		t.Errorf("CreateBooking ServiceName = %v, want 基础套餐", createBookingArg(t, db, 10))
	}
	duration, ok := createBookingArg(t, db, 11).(*int32)
	if !ok || duration == nil || *duration != 120 {
		t.Errorf("CreateBooking ServiceDuration = %v, want 120", createBookingArg(t, db, 11))
	}
}

func TestCreateBooking_ServiceMissingReturnsInvalidReference(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{err: pgx.ErrNoRows},
	}}
	svc := NewBookingService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      999,
		Date:           "2026-10-21",
		Time:           "11:00",
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("want ErrInvalidReference for a missing service, got %v", err)
	}
}

func TestCreateBooking_TransientDBErrorNotMasked(t *testing.T) {
	transient := errors.New("db connection reset")
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{err: transient},
	}}
	svc := NewBookingService(repository.New(db))

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      99,
		Date:           "2026-10-23",
		Time:           "10:00",
	})
	if !errors.Is(err, transient) {
		t.Fatalf("transient DB error = %v, want %v", err, transient)
	}
	if errors.Is(err, ErrInvalidReference) {
		t.Fatalf("transient DB error must not be masked as ErrInvalidReference")
	}
}

func TestCreateBooking_TemplateServiceRejected(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{(*int64)(nil)}}, // 无主平台模板套餐
	}}
	svc := NewBookingService(repository.New(db))
	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 1, Date: "2026-10-24", Time: "10:00",
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("want ErrInvalidReference for a template (ownerless) service, got %v", err)
	}
}

func TestCreateBooking_NegotiableService(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},                                // 拉黑检查
		fakeRow{values: []any{int64(0)}},                                                   // 冲突
		fakeRow{values: []any{int64(2)}},                                                   // GetServicePhotographerID -> 摄影师 2（本人）
		fakeRow{values: []any{int64(1), "面议套餐", (*int32)(nil), (*string)(nil), int32(60)}}, // GetServiceById -> price NULL
		fakeRow{values: bookingScanValues(11, "pending")},                                  // CreateBooking
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))
	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 99, Date: "2026-10-20", Time: "10:00",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := createBookingInt32Arg(t, db, 6); got != int32(0) {
		t.Errorf("negotiable TotalPrice = %v, want 0 (占位；展示靠 price_status)", got)
	}
	if got := createBookingArg(t, db, 8); got != "negotiable" {
		t.Errorf("negotiable PriceMode = %v, want negotiable", got)
	}
	if got := createBookingArg(t, db, 9); got != "awaiting_quote" {
		t.Errorf("negotiable PriceStatus = %v, want awaiting_quote", got)
	}
}

func TestCreateBooking_MutualService(t *testing.T) {
	zero := int32(0)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{int64(2)}}, // 套餐属于摄影师 2（本人）
		fakeRow{values: []any{int64(7), "互勉套餐", &zero, (*string)(nil), int32(45)}},
		fakeRow{values: bookingScanValues(12, "pending")},
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))
	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2, CoserID: 5, ServiceID: 7, Date: "2026-10-22", Time: "09:00",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := createBookingInt32Arg(t, db, 6); got != int32(0) {
		t.Errorf("mutual TotalPrice = %v, want 0 (占位)", got)
	}
	if got := createBookingArg(t, db, 8); got != "mutual" {
		t.Errorf("mutual PriceMode = %v, want mutual", got)
	}
	if got := createBookingArg(t, db, 9); got != "agreed" {
		t.Errorf("mutual PriceStatus = %v, want agreed", got)
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
