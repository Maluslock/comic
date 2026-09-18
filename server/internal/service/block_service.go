package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type BlockService struct {
	queries *repository.Queries
}

func NewBlockService(queries *repository.Queries) *BlockService {
	return &BlockService{queries: queries}
}

type BlockItem struct {
	UserID    int64  `json:"userId"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	CreatedAt string `json:"createdAt"`
}

func (s *BlockService) Block(ctx context.Context, userID, blockedUserID int64) error {
	if userID == blockedUserID {
		return ErrInvalidReference
	}
	exists, err := s.queries.UserExists(ctx, blockedUserID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return s.queries.BlockUser(ctx, userID, blockedUserID)
}

func (s *BlockService) Unblock(ctx context.Context, userID, blockedUserID int64) error {
	return s.queries.UnblockUser(ctx, userID, blockedUserID)
}

func (s *BlockService) List(ctx context.Context, userID int64) ([]BlockItem, error) {
	rows, err := s.queries.ListBlockedUsers(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]BlockItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, BlockItem{
			UserID:    r.UserID,
			Name:      r.Name,
			Avatar:    r.Avatar,
			CreatedAt: r.CreatedAt,
		})
	}
	return items, nil
}

func (s *BlockService) IsBlocked(ctx context.Context, userA, userB int64) (bool, error) {
	return s.queries.IsBlockedPair(ctx, userA, userB)
}
