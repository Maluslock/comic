package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("fakeRow.Scan: %d dest vs %d values", len(dest), len(r.values))
	}
	for i := range dest {
		if err := assignScan(dest[i], r.values[i]); err != nil {
			return err
		}
	}
	return nil
}

// fakeRows serves a canned result set for `:many` statements. Scan delegates to
// the underlying pgx.Row so the same value-assignment rules apply.
type fakeRows struct {
	rows []pgx.Row
	next int
	err  error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }

func (r *fakeRows) Next() bool {
	if r.next >= len(r.rows) {
		return false
	}
	r.next++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.next == 0 || r.next > len(r.rows) {
		return errors.New("fakeRows.Scan: called outside Next()")
	}
	return r.rows[r.next-1].Scan(dest...)
}

func assignScan(dest any, val any) error {
	switch d := dest.(type) {
	case *int64:
		v, ok := val.(int64)
		if !ok {
			return fmt.Errorf("assignScan: want int64 for %T", dest)
		}
		*d = v
	case *int32:
		v, ok := val.(int32)
		if !ok {
			return fmt.Errorf("assignScan: want int32 for %T", dest)
		}
		*d = v
	case **int32:
		if val == nil {
			*d = nil
			return nil
		}
		v, ok := val.(*int32)
		if !ok {
			return fmt.Errorf("assignScan: want *int32 for %T", dest)
		}
		*d = v
	case *string:
		v, ok := val.(string)
		if !ok {
			return fmt.Errorf("assignScan: want string for %T", dest)
		}
		*d = v
	case **string:
		if val == nil {
			*d = nil
			return nil
		}
		v, ok := val.(*string)
		if !ok {
			return fmt.Errorf("assignScan: want *string for %T", dest)
		}
		*d = v
	case **int64:
		if val == nil {
			*d = nil
			return nil
		}
		if v, ok := val.(*int64); ok {
			*d = v
			return nil
		}
		v, ok := val.(int64)
		if !ok {
			return fmt.Errorf("assignScan: want int64 or *int64 for %T", dest)
		}
		*d = &v
	case *time.Time:
		v, ok := val.(time.Time)
		if !ok {
			return fmt.Errorf("assignScan: want time.Time for %T", dest)
		}
		*d = v
	case *bool:
		v, ok := val.(bool)
		if !ok {
			return fmt.Errorf("assignScan: want bool for %T", dest)
		}
		*d = v
	case *pgtype.Numeric:
		v, ok := val.(pgtype.Numeric)
		if !ok {
			return fmt.Errorf("assignScan: want pgtype.Numeric for %T", dest)
		}
		*d = v
	case *[]string:
		if val == nil {
			*d = nil
			return nil
		}
		v, ok := val.([]string)
		if !ok {
			return fmt.Errorf("assignScan: want []string for %T", dest)
		}
		*d = v
	default:
		return fmt.Errorf("assignScan: unsupported dest %T", dest)
	}
	return nil
}

type fakeDBTX struct {
	rows     []pgx.Row
	next     int
	execRows int64

	// execArgs records each Exec call's argument list so a test can assert on the
	// contract a statement has with the database (e.g. nil in a COALESCE slot means
	// "keep the stored value"). Read only by the tests that need it.
	execArgs [][]any
	// queryRows serves `:many` statements. When empty, Query keeps its historical
	// behaviour of reporting an unexpected call.
	queryRows []pgx.Rows
	queryNext int
}

func (f *fakeDBTX) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	if f.next >= len(f.rows) {
		return fakeRow{err: errors.New("fakeDBTX: unexpected QueryRow call")}
	}
	r := f.rows[f.next]
	f.next++
	return r
}

func (f *fakeDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if f.queryNext >= len(f.queryRows) {
		return nil, errors.New("fakeDBTX: Query not expected")
	}
	r := f.queryRows[f.queryNext]
	f.queryNext++
	return r, nil
}

func (f *fakeDBTX) Exec(_ context.Context, _ string, a ...any) (pgconn.CommandTag, error) {
	f.execArgs = append(f.execArgs, a)
	return pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", f.execRows)), nil
}

type fakeManageStore struct {
	certifiedErr error
}

func (f *fakeManageStore) ListPhotographers(_ context.Context, _ *bool) ([]repository.AdminPhotographer, error) {
	return nil, nil
}

func (f *fakeManageStore) GetPhotographerDetail(_ context.Context, _ int64) (repository.AdminPhotographerDetail, error) {
	return repository.AdminPhotographerDetail{}, nil
}

func (f *fakeManageStore) SetCertified(_ context.Context, _ int64, _ bool) error {
	return f.certifiedErr
}

func (f *fakeManageStore) ListOrders(_ context.Context, _ string, _ int32, _ int32) ([]repository.AdminOrder, int64, error) {
	return nil, 0, nil
}

func (f *fakeManageStore) ListEvents(_ context.Context) ([]repository.AdminEvent, error) {
	return nil, nil
}

func (f *fakeManageStore) SetEventStatus(_ context.Context, _ int64, _ bool) error {
	return nil
}

func bookingScanValues(id int64, status string) []any {
	return []any{
		id, int32(1), int32(2), int32(3),
		time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		"14:00", status, int32(999),
		(*string)(nil),
		time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		"fixed", (*int32)(nil), "agreed", (*string)(nil), (*int32)(nil),
	}
}

func TestSetOrderStatus_ValidTransition(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
	}{
		{"pending to confirmed", "pending", "confirmed"},
		{"confirmed to completed", "confirmed", "completed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDBTX{rows: []pgx.Row{
				fakeRow{values: bookingScanValues(7, tc.from)},
				fakeRow{values: bookingScanValues(7, tc.to)},
				fakeRow{values: []any{int64(1)}},
			}}
			svc := &AdminManageService{
				store:      &fakeManageStore{},
				bookingSvc: NewBookingService(repository.New(db)),
			}
			item, err := svc.SetOrderStatus(context.Background(), 7, tc.to)
			if err != nil {
				t.Fatalf("SetOrderStatus(%q) unexpected error: %v", tc.to, err)
			}
			if item.ID != 7 || item.Status != tc.to {
				t.Errorf("got id=%d status=%q, want id=7 status=%q", item.ID, item.Status, tc.to)
			}
		})
	}
}

func TestSetOrderStatus_InvalidTransition(t *testing.T) {
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: bookingScanValues(7, "completed")},
	}}
	svc := &AdminManageService{
		store:      &fakeManageStore{},
		bookingSvc: NewBookingService(repository.New(db)),
	}
	_, err := svc.SetOrderStatus(context.Background(), 7, "pending")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestSetCertified_NotFound(t *testing.T) {
	svc := &AdminManageService{
		store:      &fakeManageStore{certifiedErr: repository.ErrManageNotFound},
		bookingSvc: NewBookingService(repository.New(&fakeDBTX{})),
	}
	err := svc.SetCertified(context.Background(), 99999999, true)
	if !errors.Is(err, repository.ErrManageNotFound) {
		t.Fatalf("want ErrManageNotFound, got %v", err)
	}
}
