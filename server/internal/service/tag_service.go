package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type TagDetail struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	UsageCount int32  `json:"usageCount"`
}

type TagService struct {
	queries *repository.Queries
}

func NewTagService(queries *repository.Queries) *TagService {
	return &TagService{queries: queries}
}

func (s *TagService) GetAll(ctx context.Context) ([]TagDetail, error) {
	tags, err := s.queries.GetAllTags(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]TagDetail, 0, len(tags))
	for _, t := range tags {
		items = append(items, TagDetail{
			ID:         t.ID,
			Name:       t.Name,
			UsageCount: derefInt32(t.UsageCount),
		})
	}
	return items, nil
}
