package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
)

// fakeEventApplyStore 覆盖 eventApplyStore 的全部方法，默认返回零值，
// 各用例按需注入行为，用来验证 service 层的状态映射（而不是 SQL）。
type fakeEventApplyStore struct {
	applyRow    repository.EventApplication
	applyErr    error
	applyCalls  int
	lastMsg     *string
	lastEvtID   int64
	lastPhotoID int64

	reviveRow   repository.EventApplication
	reviveErr   error
	reviveCalls int

	mineRow repository.EventApplication
	mineErr error

	myList []repository.MyEventApplication

	withdrawN   int64
	withdrawErr error

	approved []repository.PhotographerWithTags

	adminList      []repository.AdminEventApplication
	adminTotal     int64
	adminLastPage  int32
	adminLastSize  int32
	adminLastState string

	reviewN     int64
	reviewErr   error
	reviewCalls int
	lastStatus  string
	lastReason  string
	lastAdminID int64

	byIDSeq []repository.EventApplication
	byIDErr error

	event    repository.ComicEvent
	eventErr error
}

func (f *fakeEventApplyStore) ApplyEventPhotographer(_ context.Context, eventID, photographerID, userID int64, message *string) (repository.EventApplication, error) {
	f.applyCalls++
	f.lastEvtID = eventID
	f.lastPhotoID = photographerID
	f.lastMsg = message
	return f.applyRow, f.applyErr
}

func (f *fakeEventApplyStore) ReviveWithdrawnEventApplication(_ context.Context, _, _ int64, _ *string) (repository.EventApplication, error) {
	f.reviveCalls++
	return f.reviveRow, f.reviveErr
}

func (f *fakeEventApplyStore) GetMyEventApplication(_ context.Context, _, _ int64) (repository.EventApplication, error) {
	return f.mineRow, f.mineErr
}

func (f *fakeEventApplyStore) ListMyEventApplications(_ context.Context, _ int64) ([]repository.MyEventApplication, error) {
	return f.myList, nil
}

func (f *fakeEventApplyStore) WithdrawEventApplication(_ context.Context, _, _ int64) (int64, error) {
	return f.withdrawN, f.withdrawErr
}

func (f *fakeEventApplyStore) ListApprovedPhotographersByEvent(_ context.Context, _ int64) ([]repository.PhotographerWithTags, error) {
	return f.approved, nil
}

func (f *fakeEventApplyStore) CountApprovedByEvent(_ context.Context, _ int64) (int64, error) {
	return int64(len(f.approved)), nil
}

func (f *fakeEventApplyStore) ListEventApplications(_ context.Context, status string, page, size int32) ([]repository.AdminEventApplication, int64, error) {
	f.adminLastState = status
	f.adminLastPage = page
	f.adminLastSize = size
	return f.adminList, f.adminTotal, nil
}

func (f *fakeEventApplyStore) CountEventApplications(_ context.Context, _ string) (int64, error) {
	return f.adminTotal, nil
}

func (f *fakeEventApplyStore) ReviewEventApplication(_ context.Context, _ int64, status, reason string, adminID int64) (int64, error) {
	f.reviewCalls++
	f.lastStatus = status
	f.lastReason = reason
	f.lastAdminID = adminID
	return f.reviewN, f.reviewErr
}

func (f *fakeEventApplyStore) GetEventApplicationByID(_ context.Context, _ int64) (repository.EventApplication, error) {
	if len(f.byIDSeq) == 0 {
		return repository.EventApplication{}, f.byIDErr
	}
	first := f.byIDSeq[0]
	if len(f.byIDSeq) > 1 {
		f.byIDSeq = f.byIDSeq[1:]
	}
	return first, f.byIDErr
}

func (f *fakeEventApplyStore) GetEventById(_ context.Context, _ int64) (repository.ComicEvent, error) {
	return f.event, f.eventErr
}

type fakeEventApplyPhotos struct {
	profile repository.PhotographerWithTags
	err     error
}

func (f *fakeEventApplyPhotos) GetPhotographerByUserID(_ context.Context, _ int64) (repository.PhotographerWithTags, error) {
	return f.profile, f.err
}

type fakeEventApplyNotifier struct {
	calls int
	typ   string
	title string
}

func (f *fakeEventApplyNotifier) Create(_ context.Context, _ int64, typ, title, _ string) (int64, error) {
	f.calls++
	f.typ = typ
	f.title = title
	return 1, nil
}

func newEventApplySvc(store *fakeEventApplyStore, photos *fakeEventApplyPhotos, notifier *fakeEventApplyNotifier) *EventApplyService {
	return NewEventApplyService(store, photos, notifier)
}

func openEvent() repository.ComicEvent {
	return repository.ComicEvent{ID: 12, Name: "赤马EDC", EndDate: time.Now().Add(48 * time.Hour)}
}

func photographerProfile() *fakeEventApplyPhotos {
	return &fakeEventApplyPhotos{profile: repository.PhotographerWithTags{ID: 1, Name: "光影行者"}}
}

func TestEventApply_Apply_NotPhotographer(t *testing.T) {
	store := &fakeEventApplyStore{event: openEvent()}
	photos := &fakeEventApplyPhotos{err: pgx.ErrNoRows}
	svc := newEventApplySvc(store, photos, &fakeEventApplyNotifier{})

	_, err := svc.Apply(context.Background(), 142, 12, "hi")
	if !errors.Is(err, ErrEventApplyNotPhotographer) {
		t.Fatalf("want ErrEventApplyNotPhotographer, got %v", err)
	}
	if store.applyCalls != 0 {
		t.Fatalf("must not touch store when not a photographer, applyCalls=%d", store.applyCalls)
	}
}

func TestEventApply_Apply_EventNotFound(t *testing.T) {
	store := &fakeEventApplyStore{eventErr: pgx.ErrNoRows}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	_, err := svc.Apply(context.Background(), 1, 12, "")
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("want ErrEventNotFound, got %v", err)
	}
}

func TestEventApply_Apply_EventEnded(t *testing.T) {
	store := &fakeEventApplyStore{event: repository.ComicEvent{ID: 12, EndDate: time.Now().Add(-time.Hour)}}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	_, err := svc.Apply(context.Background(), 1, 12, "")
	if !errors.Is(err, ErrEventEnded) {
		t.Fatalf("want ErrEventEnded, got %v", err)
	}
	if store.applyCalls != 0 {
		t.Fatalf("must not insert when event ended, applyCalls=%d", store.applyCalls)
	}
}

func TestEventApply_Apply_Success(t *testing.T) {
	store := &fakeEventApplyStore{
		event:    openEvent(),
		applyRow: repository.EventApplication{ID: 7, EventID: 12, PhotographerID: 1, UserID: 1, Status: "pending"},
	}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	app, err := svc.Apply(context.Background(), 1, 12, "  想入驻  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.ID != 7 {
		t.Fatalf("want id 7, got %d", app.ID)
	}
	if store.lastPhotoID != 1 {
		t.Fatalf("photographer id must come from token owner, got %d", store.lastPhotoID)
	}
	if store.lastMsg == nil || *store.lastMsg != "想入驻" {
		t.Fatalf("message must be trimmed, got %v", store.lastMsg)
	}
}

func TestEventApply_Apply_BlankMessageIsNil(t *testing.T) {
	store := &fakeEventApplyStore{event: openEvent(), applyRow: repository.EventApplication{ID: 1}}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	if _, err := svc.Apply(context.Background(), 1, 12, "   "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMsg != nil {
		t.Fatalf("blank message must be NULL, got %q", *store.lastMsg)
	}
}

func TestEventApply_Apply_DuplicateIs409(t *testing.T) {
	store := &fakeEventApplyStore{
		event:     openEvent(),
		applyErr:  &pgconn.PgError{Code: "23505"},
		reviveErr: pgx.ErrNoRows,
	}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	_, err := svc.Apply(context.Background(), 1, 12, "")
	if !errors.Is(err, ErrEventApplyDuplicate) {
		t.Fatalf("want ErrEventApplyDuplicate, got %v", err)
	}
	if store.reviveCalls != 1 {
		t.Fatalf("23505 must try to revive the withdrawn row once, got %d", store.reviveCalls)
	}
}

func TestEventApply_Apply_WithdrawnIsRevived(t *testing.T) {
	store := &fakeEventApplyStore{
		event:     openEvent(),
		applyErr:  &pgconn.PgError{Code: "23505"},
		reviveRow: repository.EventApplication{ID: 9, EventID: 12, Status: "pending"},
	}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	app, err := svc.Apply(context.Background(), 1, 12, "再来一次")
	if err != nil {
		t.Fatalf("withdraw -> reapply must succeed, got %v", err)
	}
	if app.ID != 9 {
		t.Fatalf("want revived id 9, got %d", app.ID)
	}
}

func TestEventApply_Apply_ForeignKeyIs404(t *testing.T) {
	store := &fakeEventApplyStore{
		event:    openEvent(),
		applyErr: &pgconn.PgError{Code: "23503"},
	}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	_, err := svc.Apply(context.Background(), 1, 12, "")
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("want ErrEventNotFound, got %v", err)
	}
}

func TestEventApply_Withdraw_NoActiveRow(t *testing.T) {
	store := &fakeEventApplyStore{event: openEvent(), withdrawN: 0}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	if err := svc.Withdraw(context.Background(), 1, 12); !errors.Is(err, ErrEventApplyNotFound) {
		t.Fatalf("want ErrEventApplyNotFound, got %v", err)
	}
}

func TestEventApply_Withdraw_Success(t *testing.T) {
	store := &fakeEventApplyStore{event: openEvent(), withdrawN: 1}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	if err := svc.Withdraw(context.Background(), 1, 12); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventApply_Review_InvalidStatus(t *testing.T) {
	svc := newEventApplySvc(&fakeEventApplyStore{}, photographerProfile(), &fakeEventApplyNotifier{})
	if err := svc.Review(context.Background(), 1, "maybe", "", 1); !errors.Is(err, ErrEventApplyInvalidStatus) {
		t.Fatalf("want ErrEventApplyInvalidStatus, got %v", err)
	}
}

func TestEventApply_Review_ReasonRequired(t *testing.T) {
	svc := newEventApplySvc(&fakeEventApplyStore{}, photographerProfile(), &fakeEventApplyNotifier{})
	if err := svc.Review(context.Background(), 1, "rejected", "   ", 1); !errors.Is(err, ErrEventApplyReasonRequired) {
		t.Fatalf("want ErrEventApplyReasonRequired, got %v", err)
	}
}

func TestEventApply_Review_NotFound(t *testing.T) {
	store := &fakeEventApplyStore{byIDErr: pgx.ErrNoRows}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})
	if err := svc.Review(context.Background(), 404, "approved", "", 1); !errors.Is(err, ErrEventApplyNotFound) {
		t.Fatalf("want ErrEventApplyNotFound, got %v", err)
	}
}

func TestEventApply_Review_AlreadyReviewed(t *testing.T) {
	store := &fakeEventApplyStore{byIDSeq: []repository.EventApplication{{ID: 1, Status: "approved"}}}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})
	if err := svc.Review(context.Background(), 1, "rejected", "x", 1); !errors.Is(err, ErrEventApplyAlreadyReviewed) {
		t.Fatalf("want ErrEventApplyAlreadyReviewed, got %v", err)
	}
	if store.reviewCalls != 0 {
		t.Fatalf("must not update a reviewed row, reviewCalls=%d", store.reviewCalls)
	}
}

func TestEventApply_Review_ConcurrentConflictIs409(t *testing.T) {
	// 第一次读到 pending，条件更新返回 0 行，重读发现已被别人改成 approved。
	store := &fakeEventApplyStore{
		byIDSeq: []repository.EventApplication{
			{ID: 1, Status: "pending"},
			{ID: 1, Status: "approved"},
		},
		reviewN: 0,
	}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})
	if err := svc.Review(context.Background(), 1, "approved", "", 9); !errors.Is(err, ErrEventApplyAlreadyReviewed) {
		t.Fatalf("want ErrEventApplyAlreadyReviewed, got %v", err)
	}
}

func TestEventApply_Review_ApproveNotifies(t *testing.T) {
	store := &fakeEventApplyStore{
		byIDSeq: []repository.EventApplication{{ID: 1, UserID: 42, Status: "pending"}},
		reviewN: 1,
	}
	notifier := &fakeEventApplyNotifier{}
	svc := newEventApplySvc(store, photographerProfile(), notifier)

	if err := svc.Review(context.Background(), 1, "approved", "", 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastAdminID != 9 {
		t.Fatalf("admin id must be recorded, got %d", store.lastAdminID)
	}
	if notifier.calls != 1 || notifier.typ != "success" {
		t.Fatalf("approve must notify once with success, got calls=%d typ=%q", notifier.calls, notifier.typ)
	}
}

func TestEventApply_Review_RejectNotifiesWarning(t *testing.T) {
	store := &fakeEventApplyStore{
		byIDSeq: []repository.EventApplication{{ID: 1, UserID: 42, Status: "pending"}},
		reviewN: 1,
	}
	notifier := &fakeEventApplyNotifier{}
	svc := newEventApplySvc(store, photographerProfile(), notifier)

	if err := svc.Review(context.Background(), 1, "rejected", "样片不足", 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.calls != 1 || notifier.typ != "warning" {
		t.Fatalf("reject must notify once with warning, got calls=%d typ=%q", notifier.calls, notifier.typ)
	}
	if store.lastReason != "样片不足" {
		t.Fatalf("reason must reach the store, got %q", store.lastReason)
	}
}

func TestEventApply_AdminList_PaginationNormalised(t *testing.T) {
	store := &fakeEventApplyStore{}
	svc := newEventApplySvc(store, photographerProfile(), &fakeEventApplyNotifier{})

	if _, _, err := svc.AdminList(context.Background(), "pending", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.adminLastPage != 1 || store.adminLastSize != 20 || store.adminLastState != "pending" {
		t.Fatalf("want page=1 size=20 status=pending, got page=%d size=%d status=%q",
			store.adminLastPage, store.adminLastSize, store.adminLastState)
	}

	if _, _, err := svc.AdminList(context.Background(), "", 2, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.adminLastPage != 2 || store.adminLastSize != 20 {
		t.Fatalf("oversized pageSize must clamp to 20, got page=%d size=%d", store.adminLastPage, store.adminLastSize)
	}
}
