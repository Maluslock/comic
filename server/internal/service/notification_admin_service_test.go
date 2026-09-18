package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeNotifAdminStore struct {
	broadcastID    int64
	broadcastErr   error
	broadcastCalls int
	lastType       string
	lastTitle      string
	lastContent    string
	listRows       []repository.NotificationRow
	listTotal      int64
	listErr        error
}

func (f *fakeNotifAdminStore) InsertBroadcast(_ context.Context, typ, title, content string) (int64, error) {
	f.broadcastCalls++
	f.lastType = typ
	f.lastTitle = title
	f.lastContent = content
	return f.broadcastID, f.broadcastErr
}

func (f *fakeNotifAdminStore) ListAllNotifications(_ context.Context, _, _ int) ([]repository.NotificationRow, int64, error) {
	return f.listRows, f.listTotal, f.listErr
}

func (f *fakeNotifAdminStore) DeleteNotification(_ context.Context, _ int64) error {
	return nil
}

type fakeNotifAdminNotifier struct {
	createCalls int
	lastUserID  int64
	lastType    string
}

func (f *fakeNotifAdminNotifier) Create(_ context.Context, userID int64, typ, _, _ string) (int64, error) {
	f.createCalls++
	f.lastUserID = userID
	f.lastType = typ
	return 1, nil
}

type fakeNotifUserLookup struct {
	user repository.User
	err  error
}

func (f *fakeNotifUserLookup) GetByID(_ context.Context, _ int64) (repository.User, error) {
	return f.user, f.err
}

func newNotifAdminSvc(store notifAdminStore, notifier notifAdminNotifier, users notifUserLookup) *NotificationAdminService {
	return &NotificationAdminService{store: store, notifier: notifier, users: users}
}

func TestPublish_All(t *testing.T) {
	store := &fakeNotifAdminStore{broadcastID: 7}
	svc := newNotifAdminSvc(store, &fakeNotifAdminNotifier{}, &fakeNotifUserLookup{})

	id, err := svc.Publish(context.Background(), PublishRequest{
		Type: "info", Title: "测试公告", Content: "全员必读", TargetType: "all",
	})
	if err != nil {
		t.Fatalf("Publish(all) unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("want id=7, got %d", id)
	}
	if store.broadcastCalls != 1 {
		t.Fatalf("want InsertBroadcast called once, got %d", store.broadcastCalls)
	}
	if store.lastType != "info" || store.lastTitle != "测试公告" || store.lastContent != "全员必读" {
		t.Fatalf("want (info, 测试公告, 全员必读), got (%q, %q, %q)", store.lastType, store.lastTitle, store.lastContent)
	}
}

func TestPublish_Single(t *testing.T) {
	userID := int64(42)
	store := &fakeNotifAdminStore{}
	notifier := &fakeNotifAdminNotifier{}
	svc := newNotifAdminSvc(store, notifier, &fakeNotifUserLookup{user: repository.User{ID: userID}})

	id, err := svc.Publish(context.Background(), PublishRequest{
		Type: "warning", Title: "指定公告", Content: "仅指定用户", TargetType: "single", UserID: &userID,
	})
	if err != nil {
		t.Fatalf("Publish(single) unexpected error: %v", err)
	}
	if id != 1 {
		t.Fatalf("want id=1, got %d", id)
	}
	if notifier.createCalls != 1 {
		t.Fatalf("want NotificationService.Create called once, got %d", notifier.createCalls)
	}
	if notifier.lastUserID != userID || notifier.lastType != "warning" {
		t.Fatalf("want Create(42, warning), got (%d, %q)", notifier.lastUserID, notifier.lastType)
	}
	if store.broadcastCalls != 0 {
		t.Fatalf("InsertBroadcast must not be called on single, got %d calls", store.broadcastCalls)
	}
}

func TestPublish_Single_UserNotFound(t *testing.T) {
	userID := int64(999999)
	store := &fakeNotifAdminStore{}
	notifier := &fakeNotifAdminNotifier{}
	svc := newNotifAdminSvc(store, notifier, &fakeNotifUserLookup{err: pgx.ErrNoRows})

	_, err := svc.Publish(context.Background(), PublishRequest{
		Type: "info", Title: "指定公告", Content: "x", TargetType: "single", UserID: &userID,
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
	if notifier.createCalls != 0 || store.broadcastCalls != 0 {
		t.Fatalf("no write may happen on missing user (create=%d broadcast=%d)",
			notifier.createCalls, store.broadcastCalls)
	}
}

func TestPublish_InvalidType(t *testing.T) {
	store := &fakeNotifAdminStore{}
	svc := newNotifAdminSvc(store, &fakeNotifAdminNotifier{}, &fakeNotifUserLookup{})

	_, err := svc.Publish(context.Background(), PublishRequest{
		Type: "banana", Title: "坏类型", Content: "x", TargetType: "all",
	})
	if !errors.Is(err, ErrInvalidType) {
		t.Fatalf("want ErrInvalidType, got %v", err)
	}
	if store.broadcastCalls != 0 {
		t.Fatalf("InsertBroadcast must not be called on invalid type, got %d calls", store.broadcastCalls)
	}
}

func TestPublish_InvalidTargetType(t *testing.T) {
	store := &fakeNotifAdminStore{}
	svc := newNotifAdminSvc(store, &fakeNotifAdminNotifier{}, &fakeNotifUserLookup{})

	_, err := svc.Publish(context.Background(), PublishRequest{
		Type: "info", Title: "坏目标", Content: "x", TargetType: "group",
	})
	if !errors.Is(err, ErrInvalidTargetType) {
		t.Fatalf("want ErrInvalidTargetType, got %v", err)
	}
	if store.broadcastCalls != 0 {
		t.Fatalf("InsertBroadcast must not be called on invalid target, got %d calls", store.broadcastCalls)
	}
}

func TestListHistory_Broadcast(t *testing.T) {
	broadcastUserID := (*int64)(nil)
	singleUserID := int64(1)
	store := &fakeNotifAdminStore{
		listRows: []repository.NotificationRow{
			{ID: 10, Type: "info", Title: "测试公告", Content: "全员必读", UserID: broadcastUserID, Read: false, CreatedAt: "2026-09-04T10:00:00Z"},
			{ID: 11, Type: "warning", Title: "指定公告", Content: "仅指定用户", UserID: &singleUserID, Read: false, CreatedAt: "2026-09-04T10:01:00Z"},
		},
		listTotal: 2,
	}
	svc := newNotifAdminSvc(store, &fakeNotifAdminNotifier{}, &fakeNotifUserLookup{})

	items, total, err := svc.ListHistory(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListHistory unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("want total=2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].TargetType != "all" || items[0].UserID != nil {
		t.Fatalf("want broadcast row targetType=all userID=nil, got target=%q userID=%v", items[0].TargetType, items[0].UserID)
	}
	if items[1].TargetType != "single" || items[1].UserID == nil || *items[1].UserID != 1 {
		t.Fatalf("want single row targetType=single userID=1, got target=%q userID=%v", items[1].TargetType, items[1].UserID)
	}
}
