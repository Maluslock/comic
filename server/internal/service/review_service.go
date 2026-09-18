package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type CreateReviewRequest struct {
	PhotographerID int32    `json:"photographerId" binding:"required"`
	UserID         int32    `json:"userId"`
	UserName       string   `json:"userName"`
	UserAvatar     string   `json:"userAvatar"`
	Rating         int32    `json:"rating" binding:"required,min=1,max=5"`
	Content        string   `json:"content"`
	Images         []string `json:"images"`
}

type ReviewService struct {
	queries *repository.Queries
}

func NewReviewService(queries *repository.Queries) *ReviewService {
	return &ReviewService{queries: queries}
}

func (s *ReviewService) Create(ctx context.Context, req CreateReviewRequest) (*ReviewItem, error) {
	var userName, userAvatar, content *string
	if req.UserName != "" {
		userName = &req.UserName
	}
	if req.UserAvatar != "" {
		userAvatar = &req.UserAvatar
	}
	if req.Content != "" {
		content = &req.Content
	}

	images := req.Images
	if images == nil {
		images = []string{}
	}

	review, err := s.queries.CreateReview(ctx, repository.CreateReviewParams{
		PhotographerID: req.PhotographerID,
		UserID:         req.UserID,
		UserName:       userName,
		UserAvatar:     userAvatar,
		Rating:         req.Rating,
		Content:        content,
		Images:         images,
	})
	if err != nil {
		return nil, err
	}

	img := review.Images
	if img == nil {
		img = []string{}
	}

	return &ReviewItem{
		ID:             review.ID,
		PhotographerID: review.PhotographerID,
		UserID:         review.UserID,
		UserName:       derefString(review.UserName),
		UserAvatar:     derefString(review.UserAvatar),
		Rating:         review.Rating,
		Content:        derefString(review.Content),
		Images:         img,
		CreatedAt:      review.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *ReviewService) ListByPhotographer(ctx context.Context, photographerID int32) ([]ReviewItem, error) {
	reviews, err := s.queries.GetReviewsByPhotographer(ctx, photographerID)
	if err != nil {
		return nil, err
	}
	return mapReviewItems(reviews), nil
}

func (s *ReviewService) ListByUser(ctx context.Context, userID int32) ([]ReviewItem, error) {
	reviews, err := s.queries.ListReviewsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapReviewItems(reviews), nil
}
