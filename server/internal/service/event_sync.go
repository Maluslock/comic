package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Maluslock/comic/server/internal/config"
	"github.com/Maluslock/comic/server/internal/repository"
)

// AllcppEvent is the raw JSON structure from allcpp.cn API.
type AllcppEvent struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	CityName      string `json:"cityName"`
	AreaName      string `json:"areaName"`
	ProvName      string `json:"provName"`
	EnterAddress  string `json:"enterAddress"`
	EnterTime     int64  `json:"enterTime"`
	EndTime       int64  `json:"endTime"`
	AppLogoPicUrl string `json:"appLogoPicUrl"`
	Tag           string `json:"tag"`
	Enabled       int    `json:"enabled"`
	TypeName      string `json:"typeName"`
}

type allcppResponse struct {
	Result struct {
		PageCount   int           `json:"pageCount"`
		Total       int           `json:"total"`
		CurrentPage int           `json:"currentPage"`
		List        []AllcppEvent `json:"list"`
	} `json:"result"`
	Message   string `json:"message"`
	IsSuccess bool   `json:"isSuccess"`
}

// SyncResult summarizes an event sync operation.
type SyncResult struct {
	Total   int
	New     int
	Updated int
	Errors  int
}

// EventSyncer fetches events from allcpp.cn and upserts them into the database.
type EventSyncer struct {
	queries    *repository.Queries
	baseURL    string
	httpClient *http.Client
	cdnBase    string
}

// NewEventSyncer creates a new EventSyncer.
func NewEventSyncer(queries *repository.Queries, cfg *config.Config) *EventSyncer {
	return &EventSyncer{
		queries:    queries,
		baseURL:    cfg.AllcppBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cdnBase:    "https://imagecdn3.allcpp.cn/upload",
	}
}

// Sync fetches all events from allcpp.cn and upserts them into the database.
func (s *EventSyncer) Sync(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch first page to get total pages
	resp, err := s.fetchPage(ctx, 1, 50)
	if err != nil {
		return nil, fmt.Errorf("fetch page 1: %w", err)
	}

	events := resp.Result.List
	totalPages := resp.Result.PageCount

	// Fetch remaining pages
	for page := 2; page <= totalPages; page++ {
		pageResp, err := s.fetchPage(ctx, page, 50)
		if err != nil {
			result.Errors++
			continue // skip errored pages, continue syncing
		}
		events = append(events, pageResp.Result.List...)
	}

	result.Total = len(events)

	// Upsert each event
	for _, e := range events {
		params := s.mapToParams(e)
		_, err := s.queries.UpsertEvent(ctx, params)
		if err != nil {
			result.Errors++
			continue
		}
		result.New++ // Upsert handles both insert and update
	}

	return result, nil
}

func (s *EventSyncer) fetchPage(ctx context.Context, page, size int) (*allcppResponse, error) {
	url := fmt.Sprintf("%s/allcpp/event/getList.do?order=1&sort=0&pageindex=%d&pagesize=%d",
		s.baseURL, page, size)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result allcppResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("allcpp api error: %s", result.Message)
	}

	return &result, nil
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *EventSyncer) mapToParams(e AllcppEvent) repository.UpsertEventParams {
	// Map enabled to status
	status := "upcoming"
	switch e.Enabled {
	case 5:
		status = "cancelled"
	case 6, 7:
		status = "ended"
	}

	// Build location from city + province
	location := e.CityName
	if e.ProvName != "" && e.ProvName != e.CityName {
		location = e.ProvName + e.CityName
	}

	// Build cover URL
	coverURL := e.AppLogoPicUrl
	if coverURL != "" && !strings.HasPrefix(coverURL, "http") {
		coverURL = s.cdnBase + coverURL
	}

	// Parse tags from pipe-separated string
	var tags []string
	if e.Tag != "" {
		for _, t := range strings.Split(e.Tag, "|") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Marshal raw data
	rawData, _ := json.Marshal(e)

	return repository.UpsertEventParams{
		AllcppID:  int32(e.ID),
		Name:      e.Name,
		Location:  strPtr(location),
		Venue:     strPtr(e.EnterAddress),
		StartDate: time.UnixMilli(e.EnterTime),
		EndDate:   time.UnixMilli(e.EndTime),
		CoverUrl:  strPtr(coverURL),
		Tags:      tags,
		TypeName:  strPtr(e.TypeName),
		Status:    status,
		RawData:   rawData,
	}
}
