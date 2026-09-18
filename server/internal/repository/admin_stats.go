package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Totals struct {
	Users         int64 `json:"users"`
	Photographers int64 `json:"photographers"`
	Certified     int64 `json:"certified"`
	Orders        int64 `json:"orders"`
	PendingOrders int64 `json:"pendingOrders"`
	Events        int64 `json:"events"`
}

type DayStat struct {
	Date        string `json:"date"`
	NewUsers    int64  `json:"newUsers"`
	ActiveUsers int64  `json:"activeUsers"`
	Orders      int64  `json:"orders"`
}

type AdminStatsRepo struct {
	pool *pgxpool.Pool
}

func NewAdminStatsRepo(pool *pgxpool.Pool) *AdminStatsRepo {
	return &AdminStatsRepo{pool: pool}
}

func (r *AdminStatsRepo) Totals(ctx context.Context) (Totals, error) {
	var t Totals
	queries := []struct {
		sql  string
		dest *int64
	}{
		{"SELECT count(*) FROM users", &t.Users},
		{"SELECT count(*) FROM photographers", &t.Photographers},
		{"SELECT count(*) FROM photographers WHERE certified = true", &t.Certified},
		{"SELECT count(*) FROM bookings", &t.Orders},
		{"SELECT count(*) FROM bookings WHERE status = 'pending'", &t.PendingOrders},
		{"SELECT count(*) FROM comic_events WHERE del_flag = false", &t.Events},
	}
	for _, q := range queries {
		if err := r.pool.QueryRow(ctx, q.sql).Scan(q.dest); err != nil {
			return Totals{}, err
		}
	}
	return t, nil
}

func (r *AdminStatsRepo) Past7Days(ctx context.Context) ([]DayStat, error) {
	newUsers, err := r.countByDay(ctx,
		`SELECT to_char(date_trunc('day', created_at)::date, 'MM-DD') AS day, count(*) AS n
		 FROM users
		 WHERE created_at >= date_trunc('day', NOW()) - interval '6 days'
		 GROUP BY day`)
	if err != nil {
		return nil, err
	}
	activeUsers, err := r.countByDay(ctx,
		`SELECT to_char(date_trunc('day', created_at)::date, 'MM-DD') AS day, count(DISTINCT user_id) AS n
		 FROM user_tokens
		 WHERE created_at >= date_trunc('day', NOW()) - interval '6 days'
		 GROUP BY day`)
	if err != nil {
		return nil, err
	}
	orders, err := r.countByDay(ctx,
		`SELECT to_char(date_trunc('day', created_at)::date, 'MM-DD') AS day, count(*) AS n
		 FROM bookings
		 WHERE created_at >= date_trunc('day', NOW()) - interval '6 days'
		 GROUP BY day`)
	if err != nil {
		return nil, err
	}

	days := make([]DayStat, 0, 7)
	for _, d := range last7Dates() {
		days = append(days, DayStat{
			Date:        d,
			NewUsers:    newUsers[d],
			ActiveUsers: activeUsers[d],
			Orders:      orders[d],
		})
	}
	return days, nil
}

// countByDay runs a "SELECT day, n" query and returns a map keyed by the day string.
func (r *AdminStatsRepo) countByDay(ctx context.Context, sql string) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		result[day] = n
	}
	return result, rows.Err()
}

func (r *AdminStatsRepo) OrdersByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, "SELECT status, count(*) FROM bookings GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func last7Dates() []string {
	// Returns "MM-DD" for the last 7 days ending today, oldest first.
	days := make([]string, 0, 7)
	for i := 6; i >= 0; i-- {
		t := time.Now().AddDate(0, 0, -i)
		days = append(days, t.Format("01-02"))
	}
	return days
}
