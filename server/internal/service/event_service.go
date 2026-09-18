package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type EventListResponse struct {
	List     []EventItem `json:"list"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type EventDetail struct {
	EventItem
	Photographers []PhotographerItem `json:"photographers"`
	FeaturedWorks []WorkItem         `json:"featuredWorks"`
}

type EventService struct {
	queries *repository.Queries
}

func NewEventService(queries *repository.Queries) *EventService {
	return &EventService{queries: queries}
}

func (s *EventService) List(ctx context.Context, location, status string, page, size int) (*EventListResponse, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size

	var loc, st *string
	if location != "" {
		loc = &location
	}
	if status != "" {
		st = &status
	}

	events, err := s.queries.SearchEvents(ctx, repository.SearchEventsParams{
		Query:  loc,
		Status: st,
		Limit:  int32(size + 1),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(events) > size
	if hasMore {
		events = events[:size]
	}

	items := mapEvents(events)

	total := offset + len(events)
	if hasMore {
		total = offset + size + 1
	}

	return &EventListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}

func (s *EventService) GetDetail(ctx context.Context, id int64) (*EventDetail, error) {
	event, err := s.queries.GetEventById(ctx, id)
	if err != nil {
		return nil, err
	}

	photographers, err := s.queries.GetRecommendedPhotographers(ctx, 4)
	if err != nil {
		photographers = []repository.PhotographerWithTags{}
	}

	works, err := s.queries.GetFeaturedWorks(ctx, 4)
	if err != nil {
		works = []repository.FeaturedWork{}
	}

	return &EventDetail{
		EventItem:     mapSingleEvent(event),
		Photographers: mapPhotographers(photographers),
		FeaturedWorks: mapFeaturedWorks(works),
	}, nil
}

func mapSingleEvent(e repository.ComicEvent) EventItem {
	tags := e.Tags
	if tags == nil {
		tags = []string{}
	}
	imageGallery := e.ImageGallery
	if imageGallery == nil {
		imageGallery = []string{}
	}
	return EventItem{
		ID:           e.ID,
		Name:         e.Name,
		Location:     derefString(e.Location),
		Venue:        derefString(e.Venue),
		Address:      derefString(e.Address),
		StartDate:    e.StartDate.Format("2006-01-02T15:04:05Z07:00"),
		EndDate:      e.EndDate.Format("2006-01-02T15:04:05Z07:00"),
		CoverURL:     derefString(e.CoverUrl),
		Tags:         tags,
		ImageGallery: imageGallery,
		Status:       e.Status,
		TypeName:     derefString(e.TypeName),
	}
}

func mapFeaturedWorks(works []repository.FeaturedWork) []WorkItem {
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
