package service

import (
	"context"
	"errors"
	"log"

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

var (
	// ErrReviewNotAllowed：没有已完成的订单不能评价 —— 评价必须对应真实发生过的拍摄。
	ErrReviewNotAllowed = errors.New("no completed booking with this photographer")
	// ErrReviewSelf：不能给自己评价。
	ErrReviewSelf = errors.New("cannot review yourself")
	// ErrReviewDuplicate：一人对一位摄影师只能有一条评价（DB 唯一索引兜底）。
	ErrReviewDuplicate = errors.New("already reviewed this photographer")
)

type ReviewService struct {
	queries *repository.Queries
}

func NewReviewService(queries *repository.Queries) *ReviewService {
	return &ReviewService{queries: queries}
}

func (s *ReviewService) Create(ctx context.Context, req CreateReviewRequest) (*ReviewItem, error) {
	// 评价必须对应真实发生过的拍摄：此前对任何人都无条件放行（没约过也能评、能无限重复）。
	completed, err := s.queries.HasCompletedBookingWith(ctx, int64(req.PhotographerID), req.UserID)
	if err != nil {
		return nil, err
	}
	if !completed {
		return nil, ErrReviewNotAllowed
	}
	// 自己不能评价自己（配合「禁止自成交」双重兜底）。
	if p, err := s.queries.GetPhotographerById(ctx, req.PhotographerID); err == nil &&
		p.UserID != nil && *p.UserID == int64(req.UserID) {
		return nil, ErrReviewSelf
	}

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
		// DB 唯一索引 uniq_reviews_user_photographer 是并发下的最终防线。
		if isUniqueViolation(err) {
			return nil, ErrReviewDuplicate
		}
		return nil, err
	}

	// 评分/评价数/单量按事实重算（此前它们是种子死数字，插评价从不更新）。
	// best-effort：评价本身已落库，重算失败只记日志（下一条评价会自我修正），
	// 不能因为派生字段让用户以为提交失败。
	if err := s.queries.RecomputePhotographerStats(ctx, int64(req.PhotographerID)); err != nil {
		log.Printf("recompute photographer stats failed: %v", err)
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
