package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- API Response DTOs ---

// HomeResponse is the API response DTO for GET /api/v1/home
type HomeResponse struct {
	Banners                  []BannerItem       `json:"banners"`
	UpcomingEvents           []EventItem        `json:"upcomingEvents"`
	HotTags                  []TagItem          `json:"hotTags"`
	RecommendedPhotographers []PhotographerItem `json:"recommendedPhotographers"`
	FeaturedWorks            []WorkItem         `json:"featuredWorks"`
}

type BannerItem struct {
	ID       int64  `json:"id"`
	ImageURL string `json:"imageUrl"`
	Title    string `json:"title"`
	LinkType string `json:"linkType"`
	LinkID   *int32 `json:"linkId"`
}

type EventItem struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Location  string   `json:"location"`
	Venue     string   `json:"venue"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	CoverURL  string   `json:"coverUrl"`
	Tags      []string `json:"tags"`
	Status    string   `json:"status"`
	TypeName  string   `json:"typeName"`
}

type TagItem struct {
	Name       string `json:"name"`
	UsageCount int32  `json:"usageCount"`
}

type PhotographerItem struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Avatar      string   `json:"avatar"`
	Location    string   `json:"location"`
	Rating      float64  `json:"rating"`
	ReviewCount int32    `json:"reviewCount"`
	OrderCount  int32    `json:"orderCount"`
	Tags        []string `json:"tags"`
}

type WorkItem struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Images           []string `json:"images"`
	PhotographerName string   `json:"photographerName"`
}

// --- Service ---

// HomeService handles home page business logic
type HomeService struct {
	repo *repository.HomeRepository
}

// NewHomeService creates a new HomeService
func NewHomeService(repo *repository.HomeRepository) *HomeService {
	return &HomeService{repo: repo}
}

// GetHomeData fetches and aggregates all home page data
func (s *HomeService) GetHomeData(ctx context.Context) (*HomeResponse, error) {
	data, err := s.repo.GetHomeData(ctx)
	if err != nil {
		return nil, fmt.Errorf("home service: %w", err)
	}

	return &HomeResponse{
		Banners:                  mapBanners(data.Banners),
		UpcomingEvents:           mapEvents(data.UpcomingEvents),
		HotTags:                  mapTags(data.HotTags),
		RecommendedPhotographers: mapPhotographers(data.RecommendedPhotographers),
		FeaturedWorks:            mapWorks(data.FeaturedWorks),
	}, nil
}

// GetHomeDataParallel fetches home data using parallel goroutines for future optimization.
// Currently calls GetHomeData under the hood; individual goroutines handle mapping in parallel.
func (s *HomeService) GetHomeDataParallel(ctx context.Context) (*HomeResponse, error) {
	data, err := s.repo.GetHomeData(ctx)
	if err != nil {
		return nil, fmt.Errorf("home service: %w", err)
	}

	var (
		banners       []BannerItem
		events        []EventItem
		tags          []TagItem
		photographers []PhotographerItem
		works         []WorkItem
		mu            sync.Mutex
		wg            sync.WaitGroup
	)

	wg.Add(5)
	go func() {
		defer wg.Done()
		mu.Lock()
		banners = mapBanners(data.Banners)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		events = mapEvents(data.UpcomingEvents)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		tags = mapTags(data.HotTags)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		photographers = mapPhotographers(data.RecommendedPhotographers)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		works = mapWorks(data.FeaturedWorks)
		mu.Unlock()
	}()
	wg.Wait()

	return &HomeResponse{
		Banners:                  banners,
		UpcomingEvents:           events,
		HotTags:                  tags,
		RecommendedPhotographers: photographers,
		FeaturedWorks:            works,
	}, nil
}

// --- Mapping functions ---

func mapBanners(banners []repository.Banner) []BannerItem {
	items := make([]BannerItem, 0, len(banners))
	for _, b := range banners {
		items = append(items, BannerItem{
			ID:       b.ID,
			ImageURL: b.ImageUrl,
			Title:    derefString(b.Title),
			LinkType: derefString(b.LinkType),
			LinkID:   b.LinkID,
		})
	}
	return items
}

func mapEvents(events []repository.ComicEvent) []EventItem {
	items := make([]EventItem, 0, len(events))
	for _, e := range events {
		tags := e.Tags
		if tags == nil {
			tags = []string{}
		}
		items = append(items, EventItem{
			ID:        e.ID,
			Name:      e.Name,
			Location:  derefString(e.Location),
			Venue:     derefString(e.Venue),
			StartDate: e.StartDate.Format("2006-01-02T15:04:05Z07:00"),
			EndDate:   e.EndDate.Format("2006-01-02T15:04:05Z07:00"),
			CoverURL:  derefString(e.CoverUrl),
			Tags:      tags,
			Status:    e.Status,
			TypeName:  derefString(e.TypeName),
		})
	}
	return items
}

func mapTags(tags []repository.Tag) []TagItem {
	items := make([]TagItem, 0, len(tags))
	for _, t := range tags {
		items = append(items, TagItem{
			Name:       t.Name,
			UsageCount: derefInt32(t.UsageCount),
		})
	}
	return items
}

func mapPhotographers(photographers []repository.PhotographerWithTags) []PhotographerItem {
	items := make([]PhotographerItem, 0, len(photographers))
	for _, p := range photographers {
		tags := p.Tags
		if tags == nil {
			tags = []string{}
		}
		rating := numericToFloat64(p.Rating)
		items = append(items, PhotographerItem{
			ID:          int64(p.ID),
			Name:        p.Name,
			Avatar:      derefString(p.Avatar),
			Location:    derefString(p.Location),
			Rating:      rating,
			ReviewCount: derefInt32(p.ReviewCount),
			OrderCount:  derefInt32(p.OrderCount),
			Tags:        tags,
		})
	}
	return items
}

func mapWorks(works []repository.FeaturedWork) []WorkItem {
	items := make([]WorkItem, 0, len(works))
	for _, w := range works {
		images := w.Images
		if images == nil {
			images = []string{}
		}
		items = append(items, WorkItem{
			ID:               w.ID,
			Title:            w.Title,
			Images:           images,
			PhotographerName: w.PhotographerName,
		})
	}
	return items
}

// --- Helpers ---

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.NaN {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return f8.Float64
}
