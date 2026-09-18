package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

var (
	ErrAlreadyApplied  = errors.New("cert application already applied")
	ErrNotPhotographer = errors.New("user is not a photographer")
	ErrInvalidAction   = errors.New("invalid review action")
	ErrReasonRequired  = errors.New("reject reason required")
	ErrAlreadyReviewed = errors.New("cert application already reviewed")
)

type certAppStore interface {
	Create(ctx context.Context, userID, photographerID int64, evidenceImages []string, evidenceDesc string) (int64, error)
	GetByPhotographerID(ctx context.Context, photographerID int64) (*repository.CertApplication, error)
	ListByPhotographerID(ctx context.Context, photographerID int64) ([]repository.CertApplication, error)
	List(ctx context.Context, status string, limit, offset int) ([]repository.AdminCertApplication, int64, error)
	GetByID(ctx context.Context, id int64) (*repository.AdminCertApplication, error)
	Review(ctx context.Context, id int64, action string, reason string, adminID int64) error
	ReviewApproveTx(ctx context.Context, id int64, adminID int64, photographerID int64) error
}

type photographerLookup interface {
	GetPhotographerByUserID(ctx context.Context, userID int64) (repository.PhotographerWithTags, error)
}

type certNotifier interface {
	Create(ctx context.Context, userID int64, typ, title, content string) (int64, error)
}

type CertApplicationService struct {
	store         certAppStore
	photographers photographerLookup
	notifier      certNotifier
}

func NewCertApplicationService(store certAppStore, photographers photographerLookup, notifier certNotifier) *CertApplicationService {
	return &CertApplicationService{store: store, photographers: photographers, notifier: notifier}
}

func (s *CertApplicationService) Submit(ctx context.Context, userID int64, evidenceImages []string, evidenceDesc string) (int64, error) {
	profile, err := s.photographers.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotPhotographer
		}
		return 0, err
	}
	photographerID := int64(profile.ID)

	existing, err := s.store.GetByPhotographerID(ctx, photographerID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	if existing != nil && (existing.Status == "pending" || existing.Status == "approved") {
		return 0, ErrAlreadyApplied
	}

	id, err := s.store.Create(ctx, userID, photographerID, evidenceImages, evidenceDesc)
	if err != nil {
		// Partial unique index is the hard guarantee: one active application per
		// photographer. A race maps to ErrAlreadyApplied instead of a 500.
		if isUniqueViolation(err) {
			return 0, ErrAlreadyApplied
		}
		return 0, err
	}
	return id, nil
}

// GetMyApplication returns the caller's latest cert application (nil if never applied).
func (s *CertApplicationService) GetMyApplication(ctx context.Context, userID int64) (*repository.CertApplication, error) {
	profile, err := s.photographers.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotPhotographer
		}
		return nil, err
	}
	existing, err := s.store.GetByPhotographerID(ctx, int64(profile.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return existing, nil
}

func (s *CertApplicationService) MyApplications(ctx context.Context, userID int64) ([]repository.CertApplication, error) {
	profile, err := s.photographers.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotPhotographer
		}
		return nil, err
	}
	items, err := s.store.ListByPhotographerID(ctx, int64(profile.ID))
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []repository.CertApplication{}
	}
	return items, nil
}

func (s *CertApplicationService) List(ctx context.Context, status string, page, pageSize int) ([]repository.AdminCertApplication, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.List(ctx, status, pageSize, (page-1)*pageSize)
}

func (s *CertApplicationService) Detail(ctx context.Context, id int64) (*repository.AdminCertApplication, error) {
	return s.store.GetByID(ctx, id)
}

// Review implements the approval transaction:
//   - approve: one atomic repo tx (mark approved + photographers.certified=true),
//     then a best-effort success notification
//   - reject:  single-write mark rejected (reason required) + warning notification
//
// Notification write failures are logged and ignored — they never block the
// review result (same pattern as booking_service.notify).
func (s *CertApplicationService) Review(ctx context.Context, id int64, action string, reason string, adminID int64) error {
	if action != "approve" && action != "reject" {
		return ErrInvalidAction
	}

	app, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if app.Status != "pending" {
		return ErrAlreadyReviewed
	}
	if action == "reject" && strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}

	if action == "approve" {
		err = s.store.ReviewApproveTx(ctx, id, adminID, app.PhotographerID)
	} else {
		err = s.store.Review(ctx, id, "rejected", reason, adminID)
	}
	if err != nil {
		if errors.Is(err, repository.ErrReviewConflict) {
			return s.raceResolve(ctx, id)
		}
		return err
	}

	if action == "approve" {
		s.notify(ctx, app.UserID, "success", "认证通过", "恭喜！您的摄影师认证已通过，黄V 徽章已生效")
	} else {
		s.notify(ctx, app.UserID, "warning", "认证未通过", "很遗憾，您的认证申请未通过。本平台建议：补充更优质的作品样片后重新提交。")
	}
	return nil
}

// raceResolve re-reads after a failed conditional review update to map the
// concurrent-race case to 409 (already reviewed) instead of a misleading 404.
func (s *CertApplicationService) raceResolve(ctx context.Context, id int64) error {
	cur, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cur.Status != "pending" {
		return ErrAlreadyReviewed
	}
	return repository.ErrReviewConflict
}

func (s *CertApplicationService) notify(ctx context.Context, userID int64, typ, title, content string) {
	if _, err := s.notifier.Create(ctx, userID, typ, title, content); err != nil {
		log.Printf("cert application notify failed: %v", err)
	}
}
