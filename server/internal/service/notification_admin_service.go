package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

var (
	ErrInvalidType       = errors.New("invalid notification type")
	ErrInvalidTargetType = errors.New("invalid target type")
	ErrUserNotFound      = errors.New("user not found")
	ErrEmptyContent      = errors.New("title and content are required")
)

type PublishRequest struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	TargetType string `json:"targetType"`
	UserID     *int64 `json:"userId"`
}

type HistoryItem struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	UserID     *int64 `json:"userId"`
	TargetType string `json:"targetType"`
	CreatedAt  string `json:"createdAt"`
}

type notifAdminStore interface {
	InsertBroadcast(ctx context.Context, typ, title, content string) (int64, error)
	ListAllNotifications(ctx context.Context, limit, offset int) ([]repository.NotificationRow, int64, error)
	DeleteNotification(ctx context.Context, id int64) error
}

type notifAdminNotifier interface {
	Create(ctx context.Context, userID int64, typ, title, content string) (int64, error)
}

type notifUserLookup interface {
	GetByID(ctx context.Context, id int64) (repository.User, error)
}

type NotificationAdminService struct {
	store    notifAdminStore
	notifier notifAdminNotifier
	users    notifUserLookup
}

func NewNotificationAdminService(queries *repository.Queries, notificationSvc *NotificationService, userRepo *repository.UserRepo) *NotificationAdminService {
	return &NotificationAdminService{store: queries, notifier: notificationSvc, users: userRepo}
}

func (s *NotificationAdminService) Publish(ctx context.Context, req PublishRequest) (int64, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		return 0, ErrEmptyContent
	}
	if req.Type != "success" && req.Type != "info" && req.Type != "warning" {
		return 0, ErrInvalidType
	}
	switch req.TargetType {
	case "all":
		return s.store.InsertBroadcast(ctx, req.Type, req.Title, req.Content)
	case "single":
		if req.UserID == nil {
			return 0, ErrInvalidTargetType
		}
		if _, err := s.users.GetByID(ctx, *req.UserID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, ErrUserNotFound
			}
			return 0, err
		}
		return s.notifier.Create(ctx, *req.UserID, req.Type, req.Title, req.Content)
	default:
		return 0, ErrInvalidTargetType
	}
}

func (s *NotificationAdminService) Delete(ctx context.Context, id int64) error {
	if err := s.store.DeleteNotification(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotificationNotFound
		}
		return err
	}
	return nil
}

func (s *NotificationAdminService) ListHistory(ctx context.Context, page, pageSize int) ([]HistoryItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, total, err := s.store.ListAllNotifications(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	items := make([]HistoryItem, 0, len(rows))
	for _, r := range rows {
		target := "single"
		if r.UserID == nil {
			target = "all"
		}
		items = append(items, HistoryItem{
			ID:         r.ID,
			Type:       r.Type,
			Title:      r.Title,
			Content:    r.Content,
			UserID:     r.UserID,
			TargetType: target,
			CreatedAt:  r.CreatedAt,
		})
	}
	return items, total, nil
}
