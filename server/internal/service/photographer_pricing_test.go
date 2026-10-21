package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

// isNilArg reports whether a recorded argument is absent. A typed nil pointer
// stored in an `any` is not `== nil`, so a plain nil comparison would wrongly
// report that the service supplied a value.
func isNilArg(v any) bool {
	if v == nil {
		return true
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return reflect.ValueOf(v).IsNil()
	}
	return false
}

// Bug: UpdateService hard-coded `active := true` and took sort_order as a plain
// int32, while the management page never sends either field. Every edit therefore
// silently re-published an unpublished package and reset its ordering to 0.
//
// The fix moves the "keep what is stored" decision into the statement via COALESCE,
// so the contract with the database is: **nil means keep**. These tests pin that
// contract down — if the service starts substituting a default again, the stored
// values get clobbered.
//
// The update statement's parameters are, in order:
//
//	$1 id, $2 photographer_id, $3 name, $4 price, $5 description, $6 duration,
//	$7 is_active, $8 sort_order
const (
	updateArgIsActive   = 6
	updateArgSortOrder  = 7
	updateArgWantLength = 8
)

func newPricingTestSvc(db *fakeDBTX) *PhotographerService {
	return newServiceTestSvc(&fakeWorksStore{profile: repository.PhotographerWithTags{ID: 7}}, db)
}

func TestUpdateService_OmitsIsActiveAndSortOrderWhenNotProvided(t *testing.T) {
	db := &fakeDBTX{execRows: 1}
	svc := newPricingTestSvc(db)

	err := svc.UpdateService(context.Background(), 1, 5, ServiceUpsertRequest{Name: "x", Duration: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(db.execArgs) != 1 {
		t.Fatalf("want 1 Exec call, got %d", len(db.execArgs))
	}
	args := db.execArgs[0]
	if len(args) != updateArgWantLength {
		t.Fatalf("want %d args, got %d: %#v", updateArgWantLength, len(args), args)
	}
	if !isNilArg(args[updateArgIsActive]) {
		t.Errorf("is_active must be nil so COALESCE keeps the stored value, got %#v", args[updateArgIsActive])
	}
	if !isNilArg(args[updateArgSortOrder]) {
		t.Errorf("sort_order must be nil so COALESCE keeps the stored value, got %#v", args[updateArgSortOrder])
	}
}

func TestUpdateService_ForwardsExplicitIsActive(t *testing.T) {
	db := &fakeDBTX{execRows: 1}
	svc := newPricingTestSvc(db)

	unpublished := false
	err := svc.UpdateService(context.Background(), 1, 5, ServiceUpsertRequest{
		Name: "x", Duration: 60, IsActive: &unpublished,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := db.execArgs[0][updateArgIsActive]
	if got == nil {
		t.Fatal("an explicit isActive must be forwarded, not dropped")
	}
	v, ok := got.(*bool)
	if !ok {
		t.Fatalf("want *bool, got %T", got)
	}
	if *v {
		t.Error("want the forwarded isActive to stay false")
	}
}

func TestUpdateService_ForwardsExplicitSortOrder(t *testing.T) {
	db := &fakeDBTX{execRows: 1}
	svc := newPricingTestSvc(db)

	order := int32(3)
	err := svc.UpdateService(context.Background(), 1, 5, ServiceUpsertRequest{
		Name: "x", Duration: 60, SortOrder: &order,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := db.execArgs[0][updateArgSortOrder]
	if got == nil {
		t.Fatal("an explicit sortOrder must be forwarded, not dropped")
	}
	v, ok := got.(*int32)
	if !ok {
		t.Fatalf("want *int32, got %T", got)
	}
	if *v != 3 {
		t.Errorf("want sortOrder 3, got %d", *v)
	}
}

// The management page renders an 上架/下架 tag from this DTO. When the endpoint
// omitted the field the page could only ever show one state, so an unpublished
// package looked published.
func TestMyServices_MapsIsActiveAndSortOrder(t *testing.T) {
	db := &fakeDBTX{queryRows: []pgx.Rows{&fakeRows{rows: []pgx.Row{
		fakeRow{values: []any{int64(1), "A", int32Ptr(399), (*string)(nil), int32(60), false, int32(7)}},
	}}}}
	svc := newPricingTestSvc(db)

	items, err := svc.MyServices(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].IsActive {
		t.Error("want IsActive=false to survive the mapping (package is unpublished)")
	}
	if items[0].SortOrder != 7 {
		t.Errorf("want SortOrder=7, got %d", items[0].SortOrder)
	}
}
