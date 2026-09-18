package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationItem struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"createdAt"`
	LinkType  *string `json:"linkType"`
	LinkID    *int64  `json:"linkId"`
}

type NotificationService struct {
	queries *repository.Queries
}

func NewNotificationService(queries *repository.Queries) *NotificationService {
	return &NotificationService{queries: queries}
}

func (s *NotificationService) Create(ctx context.Context, userID int64, typ, title, content string) (int64, error) {
	return s.queries.InsertNotification(ctx, userID, typ, title, content, nil, nil)
}

func (s *NotificationService) CreateLinked(ctx context.Context, userID int64, typ, title, content, linkType string, linkID int64) (int64, error) {
	lt, lid := linkType, linkID
	return s.queries.InsertNotification(ctx, userID, typ, title, content, &lt, &lid)
}

func (s *NotificationService) List(ctx context.Context, userID int64) ([]NotificationItem, error) {
	rows, err := s.queries.ListNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]NotificationItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, NotificationItem{
			ID:        r.ID,
			Type:      r.Type,
			Title:     r.Title,
			Content:   r.Content,
			Read:      r.Read,
			CreatedAt: r.CreatedAt,
			LinkType:  r.LinkType,
			LinkID:    r.LinkID,
		})
	}
	return items, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, id int64) error {
	if err := s.queries.MarkNotificationRead(ctx, id, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotificationNotFound
		}
		return err
	}
	return nil
}
