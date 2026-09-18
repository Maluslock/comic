package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

type FollowRequest struct {
	UserID  string `json:"userId" binding:"required"`
	EventID int64  `json:"eventId" binding:"required"`
}

type SubscribeRequest struct {
	UserID     string `json:"userId" binding:"required"`
	EventID    int64  `json:"eventId" binding:"required"`
	TemplateID string `json:"templateId"`
}

type FollowItem struct {
	ID        int64  `json:"id"`
	EventID   int64  `json:"eventId"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	Venue     string `json:"venue"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	CoverUrl  string `json:"coverUrl"`
	Status    string `json:"status"`
}

type FollowService struct {
	queries *repository.Queries
}

func NewFollowService(queries *repository.Queries) *FollowService {
	return &FollowService{queries: queries}
}

func (s *FollowService) Follow(ctx context.Context, req FollowRequest) (bool, error) {
	_, err := s.queries.InsertFollow(ctx, repository.InsertFollowParams{
		UserID:  req.UserID,
		EventID: req.EventID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil // already followed — idempotent
		}
		return false, err
	}
	return true, nil
}

func (s *FollowService) Unfollow(ctx context.Context, req FollowRequest) (bool, error) {
	err := s.queries.DeleteFollow(ctx, repository.DeleteFollowParams{
		UserID:  req.UserID,
		EventID: req.EventID,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *FollowService) ListFollows(ctx context.Context, userID string) ([]FollowItem, error) {
	rows, err := s.queries.ListFollows(ctx, repository.ListFollowsParams{
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]FollowItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, FollowItem{
			ID:        r.ID,
			EventID:   r.EventID,
			Name:      r.Name,
			Location:  derefString(r.Location),
			Venue:     derefString(r.Venue),
			StartDate: r.StartDate.Format("2006-01-02T15:04:05Z07:00"),
			EndDate:   r.EndDate.Format("2006-01-02T15:04:05Z07:00"),
			CoverUrl:  derefString(r.CoverUrl),
			Status:    r.Status,
		})
	}
	return items, nil
}

func (s *FollowService) Subscribe(ctx context.Context, req SubscribeRequest) (bool, error) {
	templateID := req.TemplateID
	if templateID == "" {
		templateID = ""
	}
	_, err := s.queries.InsertSubscription(ctx, repository.InsertSubscriptionParams{
		UserID:     req.UserID,
		EventID:    req.EventID,
		TemplateID: templateID,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
