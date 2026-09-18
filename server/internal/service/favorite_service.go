package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type FavoriteItem struct {
	PhotographerID int32  `json:"photographerId"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	Location       string `json:"location"`
	Rating         string `json:"rating"`
	AddedAt        string `json:"addedAt"`
}

type FavoriteService struct {
	queries *repository.Queries
}

func NewFavoriteService(queries *repository.Queries) *FavoriteService {
	return &FavoriteService{queries: queries}
}

func (s *FavoriteService) Add(ctx context.Context, userID int64, photographerID int32) error {
	if err := s.queries.InsertFavorite(ctx, userID, photographerID); err != nil {
		if isForeignKeyViolation(err) {
			return ErrInvalidReference
		}
		return err
	}
	return nil
}

func (s *FavoriteService) Remove(ctx context.Context, userID int64, photographerID int32) error {
	return s.queries.DeleteFavorite(ctx, userID, photographerID)
}

func (s *FavoriteService) List(ctx context.Context, userID int64) ([]FavoriteItem, error) {
	rows, err := s.queries.ListFavorites(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]FavoriteItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, FavoriteItem{
			PhotographerID: r.PhotographerID,
			Name:           r.Name,
			Avatar:         r.Avatar,
			Location:       r.Location,
			Rating:         r.Rating,
			AddedAt:        r.AddedAt,
		})
	}
	return items, nil
}
