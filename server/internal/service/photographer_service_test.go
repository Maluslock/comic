package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

type fakeWorksStore struct {
	profile             repository.PhotographerWithTags
	profileErr          error
	updateProfileErr    error
	updateProfileID     int64
	updateProfileName   string
	updateProfileMode   string
	insertID            int64
	insertErr           error
	insertedTitles      []string
	works               []repository.Work
	worksErr            error
	workPhotographerID  int32
	workPhotographerErr error
	deleteErr           error
	deletedIDs          []int64
	updateWorkErr       error
	updatedWorkID       int64
	updatedWorkTitle    string
}

func (f *fakeWorksStore) GetPhotographerByUserID(_ context.Context, _ int64) (repository.PhotographerWithTags, error) {
	return f.profile, f.profileErr
}

func (f *fakeWorksStore) UpdatePhotographerProfile(_ context.Context, id int64, name, _, _, mode string, _ *string, _ string) error {
	f.updateProfileID = id
	f.updateProfileName = name
	f.updateProfileMode = mode
	return f.updateProfileErr
}

func (f *fakeWorksStore) InsertWork(_ context.Context, _ int64, title string, _ []string, _ string) (int64, error) {
	f.insertedTitles = append(f.insertedTitles, title)
	return f.insertID, f.insertErr
}

func (f *fakeWorksStore) GetWorksByPhotographer(_ context.Context, _ int32) ([]repository.Work, error) {
	return f.works, f.worksErr
}

func (f *fakeWorksStore) GetAllWorksByPhotographer(_ context.Context, _ int32) ([]repository.Work, error) {
	return f.works, f.worksErr
}

func (f *fakeWorksStore) GetWorkPhotographerID(_ context.Context, _ int64) (int32, error) {
	return f.workPhotographerID, f.workPhotographerErr
}

func (f *fakeWorksStore) DeleteWorkByID(_ context.Context, id int64) error {
	f.deletedIDs = append(f.deletedIDs, id)
	return f.deleteErr
}

func (f *fakeWorksStore) DeleteWorkByIDAndPhotographer(_ context.Context, id, _ int64) error {
	f.deletedIDs = append(f.deletedIDs, id)
	return f.deleteErr
}

func (f *fakeWorksStore) UpdateWorkByIDAndPhotographer(_ context.Context, id, _ int64, title string, _ []string, _ string) error {
	f.updatedWorkID = id
	f.updatedWorkTitle = title
	return f.updateWorkErr
}

func newWorkTestSvc(store worksStore) *PhotographerService {
	return &PhotographerService{works: store}
}

func TestCreateWork_NotPhotographer(t *testing.T) {
	svc := newWorkTestSvc(&fakeWorksStore{profileErr: pgx.ErrNoRows})
	_, err := svc.CreateWork(context.Background(), 1001, "测试作品", []string{"https://picsum.photos/600/450"}, "")
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("CreateWork want ErrNotPhotographer, got %v", err)
	}
}

func TestCreateWork_OK(t *testing.T) {
	store := &fakeWorksStore{
		profile:  repository.PhotographerWithTags{ID: 2},
		insertID: 7,
	}
	svc := newWorkTestSvc(store)
	id, err := svc.CreateWork(context.Background(), 1001, "测试作品", []string{"https://picsum.photos/600/450"}, "")
	if err != nil {
		t.Fatalf("CreateWork unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("want id=7, got %d", id)
	}
	if len(store.insertedTitles) != 1 || store.insertedTitles[0] != "测试作品" {
		t.Fatalf("InsertWork not called with expected title: %v", store.insertedTitles)
	}
}

func TestDeleteWork_NotOwner(t *testing.T) {
	store := &fakeWorksStore{
		profile:            repository.PhotographerWithTags{ID: 2},
		workPhotographerID: 3,
	}
	svc := newWorkTestSvc(store)
	err := svc.DeleteWork(context.Background(), 1001, 9)
	if !errors.Is(err, ErrWorkForbidden) {
		t.Fatalf("DeleteWork want ErrWorkForbidden, got %v", err)
	}
	if len(store.deletedIDs) != 0 {
		t.Fatalf("DeleteWorkByID must not run for foreign work, deletedIDs=%v", store.deletedIDs)
	}
}

func TestDeleteWork_NotFound(t *testing.T) {
	store := &fakeWorksStore{
		profile:             repository.PhotographerWithTags{ID: 2},
		workPhotographerErr: pgx.ErrNoRows,
	}
	svc := newWorkTestSvc(store)
	err := svc.DeleteWork(context.Background(), 1001, 999)
	if !errors.Is(err, ErrWorkNotFound) {
		t.Fatalf("DeleteWork want ErrWorkNotFound, got %v", err)
	}
}

func TestUpdateWork_OK(t *testing.T) {
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{ID: 2}}
	svc := newWorkTestSvc(store)
	err := svc.UpdateWork(context.Background(), 1001, 9, "新标题", []string{"/static/img/work-1.jpg"}, "")
	if err != nil {
		t.Fatalf("UpdateWork unexpected error: %v", err)
	}
	if store.updatedWorkID != 9 || store.updatedWorkTitle != "新标题" {
		t.Fatalf("UpdateWork not forwarded: id=%d title=%q", store.updatedWorkID, store.updatedWorkTitle)
	}
}

func TestUpdateWork_Forbidden(t *testing.T) {
	store := &fakeWorksStore{
		profile:            repository.PhotographerWithTags{ID: 2},
		workPhotographerID: 3,
		updateWorkErr:      repository.ErrWorkNotFound,
	}
	svc := newWorkTestSvc(store)
	err := svc.UpdateWork(context.Background(), 1001, 9, "t", []string{"x"}, "")
	if !errors.Is(err, ErrWorkForbidden) {
		t.Fatalf("UpdateWork want ErrWorkForbidden, got %v", err)
	}
}

func TestUpdateWork_NotFound(t *testing.T) {
	store := &fakeWorksStore{
		profile:             repository.PhotographerWithTags{ID: 2},
		workPhotographerErr: pgx.ErrNoRows,
		updateWorkErr:       repository.ErrWorkNotFound,
	}
	svc := newWorkTestSvc(store)
	err := svc.UpdateWork(context.Background(), 1001, 999, "t", []string{"x"}, "")
	if !errors.Is(err, ErrWorkNotFound) {
		t.Fatalf("UpdateWork want ErrWorkNotFound, got %v", err)
	}
}

func TestMyWorks_NotPhotographer(t *testing.T) {
	svc := newWorkTestSvc(&fakeWorksStore{profileErr: pgx.ErrNoRows})
	items, err := svc.MyWorks(context.Background(), 1001)
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("MyWorks want ErrNotPhotographer, got %v", err)
	}
	if items != nil {
		t.Fatalf("MyWorks error path must return nil items, got %v", items)
	}
}

func TestMyWorks_IncludesDowned(t *testing.T) {
	store := &fakeWorksStore{
		profile: repository.PhotographerWithTags{ID: 2},
		works: []repository.Work{
			{ID: 1, Title: "上架作品", Status: "active"},
			{ID: 2, Title: "已下架作品", Status: "down"},
		},
	}
	svc := newWorkTestSvc(store)
	items, err := svc.MyWorks(context.Background(), 1001)
	if err != nil {
		t.Fatalf("MyWorks unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("MyWorks must include downed works, expected 2 items, got %d", len(items))
	}
	if items[1].Status != "down" {
		t.Errorf("downed work status not preserved, got %q", items[1].Status)
	}
}

func TestMyWorks_Empty(t *testing.T) {
	store := &fakeWorksStore{
		profile: repository.PhotographerWithTags{ID: 2},
		works:   nil,
	}
	svc := newWorkTestSvc(store)
	items, err := svc.MyWorks(context.Background(), 1001)
	if err != nil {
		t.Fatalf("MyWorks unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("MyWorks must return empty array, got nil")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestMyWorks_MapsCamelCaseDTO(t *testing.T) {
	desc := "夜景棚拍"
	store := &fakeWorksStore{
		profile: repository.PhotographerWithTags{ID: 2},
		works: []repository.Work{
			{
				ID:          1,
				Title:       "原神-雷电将军",
				Images:      []string{"https://picsum.photos/600/450"},
				Description: &desc,
				Status:      "active",
				CreatedAt:   time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	svc := newWorkTestSvc(store)
	items, err := svc.MyWorks(context.Background(), 1001)
	if err != nil {
		t.Fatalf("MyWorks unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	w := items[0]
	if w.ID != 1 || w.Title != "原神-雷电将军" {
		t.Errorf("work fields mismatch: %+v", w)
	}
	if w.Status != "active" {
		t.Errorf("Status want active, got %q", w.Status)
	}
	if w.Description != desc {
		t.Errorf("Description want %q, got %q", desc, w.Description)
	}
	if w.CreatedAt != "2026-09-01T10:00:00Z" {
		t.Errorf("CreatedAt want RFC3339 camelCase, got %q", w.CreatedAt)
	}
	if w.Images == nil || len(w.Images) != 1 {
		t.Errorf("Images not preserved: %v", w.Images)
	}
}

func TestUpdateProfile_NotPhotographer(t *testing.T) {
	svc := newWorkTestSvc(&fakeWorksStore{profileErr: pgx.ErrNoRows})
	err := svc.UpdateProfile(context.Background(), 1001, ProfileUpdate{Name: "光影行者", Mode: "both"})
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("UpdateProfile want ErrNotPhotographer, got %v", err)
	}
}

func TestUpdateProfile_InvalidMode(t *testing.T) {
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{ID: 2}}
	svc := newWorkTestSvc(store)
	err := svc.UpdateProfile(context.Background(), 1001, ProfileUpdate{Name: "光影行者", Mode: "paid"})
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("UpdateProfile want ErrInvalidMode, got %v", err)
	}
	if store.updateProfileID != 0 {
		t.Fatalf("UpdatePhotographerProfile must not run on invalid mode, called with id=%d", store.updateProfileID)
	}
}

func TestUpdateProfile_OK(t *testing.T) {
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{ID: 2}}
	svc := newWorkTestSvc(store)
	intro := "提供互勉机会"
	err := svc.UpdateProfile(context.Background(), 1001, ProfileUpdate{
		Name: "光影行者", Description: "新简介", Location: "北京", Mode: "both", MutualIntro: &intro,
	})
	if err != nil {
		t.Fatalf("UpdateProfile unexpected error: %v", err)
	}
	if store.updateProfileID != 2 {
		t.Errorf("UpdatePhotographerProfile want id=2, got %d", store.updateProfileID)
	}
	if store.updateProfileName != "光影行者" {
		t.Errorf("UpdatePhotographerProfile want name=光影行者, got %q", store.updateProfileName)
	}
	if store.updateProfileMode != "both" {
		t.Errorf("UpdatePhotographerProfile want mode=both, got %q", store.updateProfileMode)
	}
}

func TestMyProfile_IncludesModeCertified(t *testing.T) {
	intro := "提供互勉"
	store := &fakeWorksStore{
		profile: repository.PhotographerWithTags{ID: 2, Name: "光影行者", Mode: "both", MutualIntro: &intro, Certified: true},
	}
	svc := newWorkTestSvc(store)
	item, err := svc.MyProfile(context.Background(), 1001)
	if err != nil {
		t.Fatalf("MyProfile unexpected error: %v", err)
	}
	if item == nil {
		t.Fatal("MyProfile returned nil item")
	}
	if item.Mode != "both" {
		t.Errorf("Mode want both, got %q", item.Mode)
	}
	if item.MutualIntro != "提供互勉" {
		t.Errorf("MutualIntro want 提供互勉, got %q", item.MutualIntro)
	}
	if !item.Certified {
		t.Errorf("Certified want true, got false")
	}
}

func TestMyProfile_NotPhotographer(t *testing.T) {
	svc := newWorkTestSvc(&fakeWorksStore{profileErr: pgx.ErrNoRows})
	_, err := svc.MyProfile(context.Background(), 1001)
	if !errors.Is(err, ErrNotPhotographer) {
		t.Fatalf("MyProfile want ErrNotPhotographer, got %v", err)
	}
}
