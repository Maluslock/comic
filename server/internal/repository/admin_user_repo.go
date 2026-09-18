package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminUser struct {
	ID            int64  `json:"id"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Avatar        string `json:"avatar"`
	Role          string `json:"role"` // 'photographer' | 'coser'（LEFT JOIN photographers 推导，默认 'coser'）
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	PhotographerID *int64 `json:"photographerId,omitempty"`
}

type AdminUserBooking struct {
	ID               int64  `json:"id"`
	Status           string `json:"status"`
	Date             string `json:"date"`
	Time             string `json:"time"`
	Price            int32  `json:"price"`
	PhotographerName string `json:"photographerName"`
	ServiceName      string `json:"serviceName"`
}

type AdminUserDetail struct {
	AdminUser
	Stats struct {
		BookingsCount  int64 `json:"bookingsCount"`
		ReviewsCount   int64 `json:"reviewsCount"`
		FavoritesCount int64 `json:"favoritesCount"`
		FollowsCount   int64 `json:"followsCount"`
	} `json:"stats"`
	RecentBookings []AdminUserBooking `json:"recentBookings"`
}

type AdminUserRepo struct {
	pool *pgxpool.Pool
}

func NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo {
	return &AdminUserRepo{pool: pool}
}

func (r *AdminUserRepo) ListUsers(ctx context.Context, keyword, role, status string, limit, offset int) ([]AdminUser, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM users u LEFT JOIN photographers p ON p.user_id = u.id
		WHERE ($1::text = '' OR u.phone ILIKE '%'||$1||'%' OR u.name ILIKE '%'||$1||'%')
		  AND ($2::text = '' OR $2::text = 'all'
		       OR ($2::text = 'photographer' AND p.user_id IS NOT NULL)
		       OR ($2::text = 'coser' AND p.user_id IS NULL))
		  AND ($3::text = '' OR $3::text = 'all' OR u.status = $3)`,
		keyword, role, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.phone, ''), COALESCE(u.name, ''), COALESCE(u.avatar, ''),
		       CASE WHEN p.user_id IS NOT NULL THEN 'photographer' ELSE 'coser' END,
		       u.status, u.created_at::text, p.id
		FROM users u LEFT JOIN photographers p ON p.user_id = u.id
		WHERE ($1::text = '' OR u.phone ILIKE '%'||$1||'%' OR u.name ILIKE '%'||$1||'%')
		  AND ($2::text = '' OR $2::text = 'all'
		       OR ($2::text = 'photographer' AND p.user_id IS NOT NULL)
		       OR ($2::text = 'coser' AND p.user_id IS NULL))
		  AND ($3::text = '' OR $3::text = 'all' OR u.status = $3)
		ORDER BY u.id DESC
		LIMIT $4 OFFSET $5`, keyword, role, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminUser, 0, limit)
	for rows.Next() {
		var i AdminUser
		if err := rows.Scan(&i.ID, &i.Phone, &i.Name, &i.Avatar, &i.Role, &i.Status,
			&i.CreatedAt, &i.PhotographerID); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *AdminUserRepo) GetUserDetail(ctx context.Context, id int64) (*AdminUserDetail, error) {
	var d AdminUserDetail
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, COALESCE(u.phone, ''), COALESCE(u.name, ''), COALESCE(u.avatar, ''),
		       CASE WHEN p.user_id IS NOT NULL THEN 'photographer' ELSE 'coser' END,
		       u.status, u.created_at::text, p.id
		FROM users u LEFT JOIN photographers p ON p.user_id = u.id
		WHERE u.id = $1`, id,
	).Scan(&d.ID, &d.Phone, &d.Name, &d.Avatar, &d.Role, &d.Status,
		&d.CreatedAt, &d.PhotographerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrManageNotFound
		}
		return nil, err
	}

	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM bookings WHERE coser_id = $1", id).Scan(&d.Stats.BookingsCount); err != nil {
		return nil, err
	}
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM reviews WHERE user_id = $1", id).Scan(&d.Stats.ReviewsCount); err != nil {
		return nil, err
	}
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM photographer_favorites WHERE user_id = $1", id).Scan(&d.Stats.FavoritesCount); err != nil {
		return nil, err
	}
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM event_follows WHERE user_id = $1", strconv.FormatInt(id, 10)).Scan(&d.Stats.FollowsCount); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT b.id, b.status, to_char(b.date, 'YYYY-MM-DD'), b.time, b.total_price,
		       COALESCE(p.name, ''), COALESCE(s.name, '')
		FROM bookings b
		LEFT JOIN photographers p ON p.id = b.photographer_id
		LEFT JOIN services s ON s.id = b.service_id
		WHERE b.coser_id = $1
		ORDER BY b.created_at DESC
		LIMIT 5`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	d.RecentBookings = make([]AdminUserBooking, 0, 5)
	for rows.Next() {
		var b AdminUserBooking
		if err := rows.Scan(&b.ID, &b.Status, &b.Date, &b.Time, &b.Price,
			&b.PhotographerName, &b.ServiceName); err != nil {
			return nil, err
		}
		d.RecentBookings = append(d.RecentBookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *AdminUserRepo) SetUserStatus(ctx context.Context, id int64, status string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET status = $2 WHERE id = $1", id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrManageNotFound
	}
	return nil
}
