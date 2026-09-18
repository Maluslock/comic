package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeCertStore struct {
	createID                  int64
	createErr                 error
	byPhotographer            *repository.CertApplication
	byPhotographerErr         error
	appByIDSeq                []*repository.AdminCertApplication
	appByIDErr                error
	reviewErr                 error
	reviewApproveTxErr        error
	reviewCalls               int
	approveTxCalls            int
	lastAction                string
	lastReason                string
	lastApproveID             int64
	lastApproveAdminID        int64
	lastApprovePhotographerID int64
}

func (f *fakeCertStore) Create(_ context.Context, _, _ int64, _ []string, _ string) (int64, error) {
	return f.createID, f.createErr
}

func (f *fakeCertStore) GetByPhotographerID(_ context.Context, _ int64) (*repository.CertApplication, error) {
	return f.byPhotographer, f.byPhotographerErr
}

func (f *fakeCertStore) ListByPhotographerID(_ context.Context, _ int64) ([]repository.CertApplication, error) {
	return nil, nil
}

func (f *fakeCertStore) List(_ context.Context, _ string, _, _ int) ([]repository.AdminCertApplication, int64, error) {
	return nil, 0, nil
}

func (f *fakeCertStore) GetByID(_ context.Context, _ int64) (*repository.AdminCertApplication, error) {
	if len(f.appByIDSeq) == 0 {
		return nil, f.appByIDErr
	}
	first := f.appByIDSeq[0]
	if len(f.appByIDSeq) > 1 {
		f.appByIDSeq = f.appByIDSeq[1:]
	}
	return first, f.appByIDErr
}

func (f *fakeCertStore) Review(_ context.Context, _ int64, action string, reason string, _ int64) error {
	f.reviewCalls++
	f.lastAction = action
	f.lastReason = reason
	return f.reviewErr
}

func (f *fakeCertStore) ReviewApproveTx(_ context.Context, id int64, adminID int64, photographerID int64) error {
	f.approveTxCalls++
	f.lastApproveID = id
	f.lastApproveAdminID = adminID
	f.lastApprovePhotographerID = photographerID
	return f.reviewApproveTxErr
}

type fakePhotographerLookup struct {
	profile repository.PhotographerWithTags
	err     error
}

func (f *fakePhotographerLookup) GetPhotographerByUserID(_ context.Context, _ int64) (repository.PhotographerWithTags, error) {
	return f.profile, f.err
}

type fakeCertNotifier struct {
	calls [][3]string
}

func (f *fakeCertNotifier) Create(_ context.Context, _ int64, typ, title, content string) (int64, error) {
	f.calls = append(f.calls, [3]string{typ, title, content})
	return 0, nil
}

func newTestSvc(store certAppStore, lookup photographerLookup, notifier certNotifier) *CertApplicationService {
	return &CertApplicationService{store: store, photographers: lookup, notifier: notifier}
}

func pendingApp() *repository.AdminCertApplication {
	return &repository.AdminCertApplication{ID: 1, UserID: 1001, PhotographerID: 2, Status: "pending"}
}

func TestCertSubmit_AlreadyApplied(t *testing.T) {
	for _, status := range []string{"pending", "approved"} {
		svc := newTestSvc(
			&fakeCertStore{byPhotographer: &repository.CertApplication{Status: status}},
			&fakePhotographerLookup{profile: repository.PhotographerWithTags{ID: 2}},
			&fakeCertNotifier{},
		)
		_, err := svc.Submit(context.Background(), 1001, []string{"https://picsum.photos/600/450"}, "desc")
		if !errors.Is(err, ErrAlreadyApplied) {
			t.Fatalf("Submit(status=%q) want ErrAlreadyApplied, got %v", status, err)
		}
	}
}

func TestCertSubmit_NewUser(t *testing.T) {
	svc := newTestSvc(
		&fakeCertStore{createID: 5, byPhotographerErr: pgx.ErrNoRows},
		&fakePhotographerLookup{profile: repository.PhotographerWithTags{ID: 2}},
		&fakeCertNotifier{},
	)
	id, err := svc.Submit(context.Background(), 1001, []string{"https://picsum.photos/600/450"}, "desc")
	if err != nil {
		t.Fatalf("Submit unexpected error: %v", err)
	}
	if id != 5 {
		t.Fatalf("want id=5, got %d", id)
	}
}

func TestCertSubmit_NotPhotographer(t *testing.T) {
	svc := newTestSvc(
		&fakeCertStore{},
		&fakePhotographerLookup{err: pgx.ErrNoRows},
		&fakeCertNotifier{},
	)
	_, err := svc.Submit(context.Background(), 9999, []string{"https://picsum.photos/600/450"}, "desc")
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("want ErrNotPhotographer, got %v", err)
	}
}

func TestGetMyApplication_NotPhotographer(t *testing.T) {
	svc := newTestSvc(
		&fakeCertStore{},
		&fakePhotographerLookup{err: pgx.ErrNoRows},
		&fakeCertNotifier{},
	)
	_, err := svc.GetMyApplication(context.Background(), 9999)
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("want ErrNotPhotographer, got %v", err)
	}
}

func TestGetMyApplication_NoApplication(t *testing.T) {
	svc := newTestSvc(
		&fakeCertStore{byPhotographerErr: pgx.ErrNoRows},
		&fakePhotographerLookup{profile: repository.PhotographerWithTags{ID: 2}},
		&fakeCertNotifier{},
	)
	app, err := svc.GetMyApplication(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetMyApplication unexpected error: %v", err)
	}
	if app != nil {
		t.Fatalf("want nil application, got %+v", app)
	}
}

func TestGetMyApplication_HasApplication(t *testing.T) {
	existing := &repository.CertApplication{ID: 7, UserID: 1001, PhotographerID: 2, Status: "pending"}
	svc := newTestSvc(
		&fakeCertStore{byPhotographer: existing},
		&fakePhotographerLookup{profile: repository.PhotographerWithTags{ID: 2}},
		&fakeCertNotifier{},
	)
	app, err := svc.GetMyApplication(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetMyApplication unexpected error: %v", err)
	}
	if app == nil || app.ID != 7 || app.Status != "pending" {
		t.Fatalf("want application id=7 status=pending, got %+v", app)
	}
}

func TestCertReview_Approve(t *testing.T) {
	store := &fakeCertStore{appByIDSeq: []*repository.AdminCertApplication{pendingApp()}}
	notifier := &fakeCertNotifier{}
	svc := newTestSvc(store, &fakePhotographerLookup{}, notifier)

	err := svc.Review(context.Background(), 1, "approve", "", 9)
	if err != nil {
		t.Fatalf("Review(approve) unexpected error: %v", err)
	}
	if store.approveTxCalls != 1 {
		t.Fatalf("want ReviewApproveTx called once, got %d", store.approveTxCalls)
	}
	if store.lastApproveID != 1 || store.lastApproveAdminID != 9 || store.lastApprovePhotographerID != 2 {
		t.Fatalf("want ReviewApproveTx(1, 9, 2), got (%d, %d, %d)",
			store.lastApproveID, store.lastApproveAdminID, store.lastApprovePhotographerID)
	}
	if store.reviewCalls != 0 {
		t.Fatalf("reject-style Review must not be called on approve, got %d calls", store.reviewCalls)
	}
	if len(notifier.calls) != 1 || notifier.calls[0][0] != "success" || notifier.calls[0][1] != "认证通过" {
		t.Fatalf("want success notification, got %v", notifier.calls)
	}
}

func TestCertReview_ApproveTxError_NoPartialState(t *testing.T) {
	txErr := errors.New("tx failed")
	store := &fakeCertStore{
		appByIDSeq:         []*repository.AdminCertApplication{pendingApp()},
		reviewApproveTxErr: txErr,
	}
	notifier := &fakeCertNotifier{}
	svc := newTestSvc(store, &fakePhotographerLookup{}, notifier)

	err := svc.Review(context.Background(), 1, "approve", "", 9)
	if !errors.Is(err, txErr) {
		t.Fatalf("want tx error propagated, got %v", err)
	}
	if store.approveTxCalls != 1 {
		t.Fatalf("want ReviewApproveTx called once, got %d", store.approveTxCalls)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("no notification may fire on failed approve tx, got %v", notifier.calls)
	}
}

func TestCertReview_InvalidAction(t *testing.T) {
	store := &fakeCertStore{appByIDSeq: []*repository.AdminCertApplication{pendingApp()}}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "banana", "", 9)
	if !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("want ErrInvalidAction, got %v", err)
	}
	if store.reviewCalls != 0 || store.approveTxCalls != 0 {
		t.Fatalf("no store write may happen on invalid action (review=%d approveTx=%d)",
			store.reviewCalls, store.approveTxCalls)
	}
}

func TestCertReview_Reject_NotifyContent(t *testing.T) {
	store := &fakeCertStore{appByIDSeq: []*repository.AdminCertApplication{pendingApp()}}
	notifier := &fakeCertNotifier{}
	svc := newTestSvc(store, &fakePhotographerLookup{}, notifier)

	err := svc.Review(context.Background(), 1, "reject", "作品太少", 9)
	if err != nil {
		t.Fatalf("Review(reject) unexpected error: %v", err)
	}
	if store.reviewCalls != 1 || store.lastAction != "rejected" || store.lastReason != "作品太少" {
		t.Fatalf("want Review(rejected, 作品太少), got action=%q reason=%q", store.lastAction, store.lastReason)
	}
	if store.approveTxCalls != 0 {
		t.Fatalf("ReviewApproveTx must not be called on reject, got %d calls", store.approveTxCalls)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("want exactly one notification, got %v", notifier.calls)
	}
	want := [3]string{"warning", "认证未通过", "很遗憾，您的认证申请未通过。本平台建议：补充更优质的作品样片后重新提交。"}
	if notifier.calls[0] != want {
		t.Fatalf("want %v, got %v", want, notifier.calls[0])
	}
}

func TestCertReview_Reject_NoReason(t *testing.T) {
	store := &fakeCertStore{appByIDSeq: []*repository.AdminCertApplication{pendingApp()}}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "reject", "", 9)
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("want ErrReasonRequired, got %v", err)
	}
	if store.reviewCalls != 0 {
		t.Fatalf("Review must not be called on missing reason, got %d calls", store.reviewCalls)
	}
}

func TestCertReview_NonPending(t *testing.T) {
	store := &fakeCertStore{
		appByIDSeq: []*repository.AdminCertApplication{{ID: 1, Status: "approved"}},
	}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "approve", "", 9)
	if !errors.Is(err, ErrAlreadyReviewed) {
		t.Fatalf("want ErrAlreadyReviewed, got %v", err)
	}
	if store.approveTxCalls != 0 {
		t.Fatalf("ReviewApproveTx must not be called on non-pending application, got %d calls", store.approveTxCalls)
	}
}

func TestCertReview_Race_AlreadyReviewed(t *testing.T) {
	// Approve loses a concurrent review: pre-check sees pending, the conditional
	// update misses, re-read shows approved → 409, not 404.
	store := &fakeCertStore{
		appByIDSeq: []*repository.AdminCertApplication{
			pendingApp(),
			{ID: 1, UserID: 1001, PhotographerID: 2, Status: "approved"},
		},
		reviewApproveTxErr: repository.ErrReviewConflict,
	}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "approve", "", 9)
	if !errors.Is(err, ErrAlreadyReviewed) {
		t.Fatalf("want ErrAlreadyReviewed (409), got %v", err)
	}
}

func TestCertReview_Reject_Race_AlreadyReviewed(t *testing.T) {
	store := &fakeCertStore{
		appByIDSeq: []*repository.AdminCertApplication{
			pendingApp(),
			{ID: 1, UserID: 1001, PhotographerID: 2, Status: "rejected"},
		},
		reviewErr: repository.ErrReviewConflict,
	}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "reject", "作品太少", 9)
	if !errors.Is(err, ErrAlreadyReviewed) {
		t.Fatalf("want ErrAlreadyReviewed (409), got %v", err)
	}
}

func TestCertReview_Race_Deleted(t *testing.T) {
	// Application deleted between pre-check and update → 404 via ErrManageNotFound.
	store := &fakeCertStore{
		appByIDSeq: []*repository.AdminCertApplication{
			pendingApp(),
			nil,
		},
		appByIDErr:         repository.ErrManageNotFound,
		reviewApproveTxErr: repository.ErrReviewConflict,
	}
	svc := newTestSvc(store, &fakePhotographerLookup{}, &fakeCertNotifier{})

	err := svc.Review(context.Background(), 1, "approve", "", 9)
	if !errors.Is(err, repository.ErrManageNotFound) {
		t.Fatalf("want ErrManageNotFound (404), got %v", err)
	}
}
