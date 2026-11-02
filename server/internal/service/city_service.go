package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

// CityItem 是城市选择器的一行。
type CityItem struct {
	Name       string `json:"name"`
	EventCount int64  `json:"eventCount"`
}

// CityListResponse 是 GET /api/v1/cities 的响应体。
// total 显式给出，前端不必再 array.length 判断「是否拿到全量」。
type CityListResponse struct {
	List  []CityItem `json:"list"`
	Total int        `json:"total"`
}

// cityStore 是城市清单需要的数据访问面，由 *repository.CityRepo 实现，
// 单测用 fake 注入（同 event_source_service.go 的 eventSourceStore 写法）。
type cityStore interface {
	List(ctx context.Context) ([]repository.CityCount, error)
}

// CityService 提供城市清单。缓存由 handler 负责，这里保持纯函数语义。
type CityService struct {
	repo cityStore
}

func NewCityService(repo cityStore) *CityService {
	return &CityService{repo: repo}
}

// List 返回全部有在办漫展的城市。
//
// 排序与场次数由仓储层保证（eventCount DESC, name ASC）；
// 这里只做「空切片不能是 nil」这一层保证 —— 前端 v-for 拿到 null 会直接报错。
func (s *CityService) List(ctx context.Context) (CityListResponse, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return CityListResponse{}, err
	}
	items := make([]CityItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, CityItem{Name: r.Name, EventCount: r.EventCount})
	}
	return CityListResponse{List: items, Total: len(items)}, nil
}
