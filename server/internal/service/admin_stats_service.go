package service

import (
	"context"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
)

type Trends struct {
	Date        []string `json:"date"`
	NewUsers    []int64  `json:"newUsers"`
	ActiveUsers []int64  `json:"activeUsers"`
	Orders      []int64  `json:"orders"`
}

type DashboardData struct {
	Totals         repository.Totals `json:"totals"`
	Trends         Trends            `json:"trends"`
	OrdersByStatus map[string]int64  `json:"ordersByStatus"`
}

type AdminStatsService struct {
	repo *repository.AdminStatsRepo
}

func NewAdminStatsService(repo *repository.AdminStatsRepo) *AdminStatsService {
	return &AdminStatsService{repo: repo}
}

func (s *AdminStatsService) Dashboard(ctx context.Context) (*DashboardData, error) {
	totals, err := s.repo.Totals(ctx)
	if err != nil {
		return nil, err
	}
	days, err := s.repo.Past7Days(ctx)
	if err != nil {
		return nil, err
	}
	byStatus, err := s.repo.OrdersByStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure all four statuses appear; absent statuses count as 0.
	normalized := map[string]int64{
		"pending":   byStatus["pending"],
		"confirmed": byStatus["confirmed"],
		"completed": byStatus["completed"],
		"cancelled": byStatus["cancelled"],
	}

	return &DashboardData{
		Totals:         totals,
		Trends:         build7DaySeries(days),
		OrdersByStatus: normalized,
	}, nil
}

// build7DaySeries projects stats onto a fixed 7-day "MM-DD" sequence ending today.
func build7DaySeries(days []repository.DayStat) Trends {
	byDate := make(map[string]repository.DayStat, len(days))
	for _, d := range days {
		byDate[d.Date] = d
	}

	t := Trends{
		Date:        make([]string, 0, 7),
		NewUsers:    make([]int64, 0, 7),
		ActiveUsers: make([]int64, 0, 7),
		Orders:      make([]int64, 0, 7),
	}
	for _, date := range last7Days() {
		d, ok := byDate[date]
		t.Date = append(t.Date, date)
		if ok {
			t.NewUsers = append(t.NewUsers, d.NewUsers)
			t.ActiveUsers = append(t.ActiveUsers, d.ActiveUsers)
			t.Orders = append(t.Orders, d.Orders)
		} else {
			t.NewUsers = append(t.NewUsers, 0)
			t.ActiveUsers = append(t.ActiveUsers, 0)
			t.Orders = append(t.Orders, 0)
		}
	}
	return t
}

func last7Days() []string {
	now := time.Now()
	days := make([]string, 0, 7)
	for i := 6; i >= 0; i-- {
		days = append(days, now.AddDate(0, 0, -i).Format("01-02"))
	}
	return days
}
