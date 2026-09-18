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

func createBookingTotalPriceArg(t *testing.T, db *bookingRecorder) int32 {
	t.Helper()
	for _, a := range db.args {
		if len(a) == 8 {
			v, ok := a[6].(int32)
			if !ok {
				t.Fatalf("TotalPrice arg type = %T, want int32", a[6])
			}
			return v
		}
	}
	t.Fatalf("CreateBooking call not found among %d QueryRow calls", len(db.args))
	return 0
}

func TestCreateBooking_WritesServicePrice(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
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

	if got := createBookingTotalPriceArg(t, db); got != 399 {
		t.Errorf("CreateBooking TotalPrice = %d, want 399 (service price)", got)
	}
}

func TestCreateBooking_ServiceLookupFailsFallsBackToZero(t *testing.T) {
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{err: errors.New("service not found")},
		fakeRow{values: bookingScanValues(10, "pending")},
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))

	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      999,
		Date:           "2026-10-21",
		Time:           "11:00",
	}); err != nil {
		t.Fatalf("Create must not fail when the service row is missing, got %v", err)
	}

	if got := createBookingTotalPriceArg(t, db); got != 0 {
		t.Errorf("CreateBooking TotalPrice = %d, want 0 fallback", got)
	}
}
