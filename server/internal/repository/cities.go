package repository

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CityCount 是一个城市的在办漫展场次数。
type CityCount struct {
	Name       string `json:"name"`
	EventCount int64  `json:"eventCount"`
}

// CityRepo 提供城市清单查询。裸 pgxpool 手写，风格同 admin_stats.go：
// sqlc 二进制不在仓库里，加一张只读聚合表不值得为它手改 sqlc 生成物。
type CityRepo struct {
	pool *pgxpool.Pool
}

func NewCityRepo(pool *pgxpool.Pool) *CityRepo {
	return &CityRepo{pool: pool}
}

// List 返回所有「还有未结束漫展」的城市及其场次数，按场次降序、同名升序。
//
// 口径必须和 C 端列表一致：del_flag = false 且 end_date >= NOW()
// （end_date 而不是 start_date —— 多日漫展跨过开场仍在办，见 events.sql 的同一条修正）。
// location 为空或纯空白的行跳过：否则前端会多出一个点不动的空城市。
func (r *CityRepo) List(ctx context.Context) ([]CityCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT location AS name, count(*) AS event_count
		   FROM comic_events
		  WHERE del_flag = false
		    AND end_date >= NOW()
		    AND location IS NOT NULL
		    AND btrim(location) <> ''
		  GROUP BY location
		  ORDER BY count(*) DESC, location ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CityCount, 0, 80)
	for rows.Next() {
		var c CityCount
		if err := rows.Scan(&c.Name, &c.EventCount); err != nil {
			return nil, err
		}
		c.Name = strings.TrimSpace(c.Name)
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
