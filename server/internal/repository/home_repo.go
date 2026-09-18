package repository

import (
	"context"
	"fmt"
)

// HomeData is the aggregate response for the home page API
type HomeData struct {
	Banners                  []Banner               `json:"banners"`
	UpcomingEvents           []ComicEvent           `json:"upcomingEvents"`
	HotTags                  []Tag                  `json:"hotTags"`
	RecommendedPhotographers []PhotographerWithTags `json:"recommendedPhotographers"`
	FeaturedWorks            []FeaturedWork         `json:"featuredWorks"`
}

// HomeRepository composes all home page queries
type HomeRepository struct {
	queries *Queries
}

// NewHomeRepository creates a new HomeRepository
func NewHomeRepository(queries *Queries) *HomeRepository {
	return &HomeRepository{queries: queries}
}

// GetHomeData fetches all home page data
func (r *HomeRepository) GetHomeData(ctx context.Context) (*HomeData, error) {
	data := &HomeData{}

	// Fetch banners
	banners, err := r.queries.GetActiveBanners(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get banners: %w", err)
	}
	data.Banners = banners

	// Fetch upcoming events
	events, err := r.queries.GetUpcomingEvents(ctx, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	data.UpcomingEvents = events

	// Fetch hot tags
	tags, err := r.queries.GetHotTags(ctx, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	data.HotTags = tags

	// Fetch recommended photographers
	photographers, err := r.queries.GetRecommendedPhotographers(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to get photographers: %w", err)
	}
	data.RecommendedPhotographers = photographers

	// Fetch featured works
	works, err := r.queries.GetFeaturedWorks(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to get works: %w", err)
	}
	data.FeaturedWorks = works

	return data, nil
}
