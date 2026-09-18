package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrManageNotFound = errors.New("admin manage: not found")

type AdminPhotographer struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Avatar     string  `json:"avatar"`
	Location   string  `json:"location"`
	Rating     float64 `json:"rating"`
	Certified  bool    `json:"certified"`
	Mode       string  `json:"mode"`
	OrderCount int64   `json:"orderCount"`
	UserPhone  string  `json:"userPhone"`
}

type AdminOrder struct {
	ID               int64  `json:"id"`
	Status           string `json:"status"`
	Date             string `json:"date"`
	Time             string `json:"time"`
	Price            int32  `json:"price"`
	CreatedAt        string `json:"createdAt"`
	CoserName        string `json:"coserName"`
	PhotographerName string `json:"photographerName"`
}

type AdminEvent struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	Venue     string `json:"venue"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Status    string `json:"status"`
	TypeName  string `json:"typeName"`
	DelFlag   bool   `json:"delFlag"`
}

type AdminPhotographerDetail struct {
	AdminPhotographer
	Description  string `json:"description"`
	MutualIntro  string `json:"mutualIntro"`
	UserID       *int64 `json:"userId"`
	WorksCount   int64  `json:"worksCount"`
	ReviewsCount int64  `json:"reviewsCount"`
}

type AdminManageRepo struct {
	pool *pgxpool.Pool
}

func NewAdminManageRepo(pool *pgxpool.Pool) *AdminManageRepo {
	return &AdminManageRepo{pool: pool}
}

func (r *AdminManageRepo) ListPhotographers(ctx context.Context, certified *bool) ([]AdminPhotographer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, COALESCE(p.avatar, ''), COALESCE(p.location, ''), p.rating,
		       p.certified, p.mode, COALESCE(p.order_count, 0)::bigint, COALESCE(u.phone, '')
		FROM photographers p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE ($1::bool IS NULL OR p.certified = $1)
		ORDER BY p.id`, certified)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AdminPhotographer
	for rows.Next() {
		var i AdminPhotographer
		var rating pgtype.Numeric
		if err := rows.Scan(&i.ID, &i.Name, &i.Avatar, &i.Location, &rating,
			&i.Certified, &i.Mode, &i.OrderCount, &i.UserPhone); err != nil {
			return nil, err
		}
		i.Rating = numericToFloat(rating)
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *AdminManageRepo) SetCertified(ctx context.Context, id int64, certified bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE photographers SET certified = $2 WHERE id = $1", id, certified)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrManageNotFound
	}
	return nil
}

func (r *AdminManageRepo) ListOrders(ctx context.Context, status string, limit, offset int32) ([]AdminOrder, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM bookings WHERE ($1::text = '' OR status = $1)", status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT b.id, b.status, to_char(b.date, 'YYYY-MM-DD'), b.time, b.total_price, b.created_at::text,
		       COALESCE(u.name, ''), COALESCE(p.name, '')
		FROM bookings b
		LEFT JOIN users u ON u.id = b.coser_id
		LEFT JOIN photographers p ON p.id = b.photographer_id
		WHERE ($1::text = '' OR b.status = $1)
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminOrder, 0, limit)
	for rows.Next() {
		var i AdminOrder
		if err := rows.Scan(&i.ID, &i.Status, &i.Date, &i.Time, &i.Price, &i.CreatedAt,
			&i.CoserName, &i.PhotographerName); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *AdminManageRepo) ListEvents(ctx context.Context) ([]AdminEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, COALESCE(location, ''), COALESCE(venue, ''),
		       to_char(start_date, 'YYYY-MM-DD'), to_char(end_date, 'YYYY-MM-DD'),
		       status, COALESCE(type_name, ''), del_flag
		FROM comic_events
		ORDER BY start_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AdminEvent
	for rows.Next() {
		var i AdminEvent
		if err := rows.Scan(&i.ID, &i.Name, &i.Location, &i.Venue,
			&i.StartDate, &i.EndDate, &i.Status, &i.TypeName, &i.DelFlag); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *AdminManageRepo) SetEventStatus(ctx context.Context, id int64, delFlag bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE comic_events SET del_flag = $2 WHERE id = $1", id, delFlag)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrManageNotFound
	}
	return nil
}

func (r *AdminManageRepo) GetPhotographerDetail(ctx context.Context, id int64) (AdminPhotographerDetail, error) {
	var d AdminPhotographerDetail
	var rating pgtype.Numeric
	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.name, COALESCE(p.avatar, ''), COALESCE(p.location, ''), p.rating,
		       p.certified, p.mode, COALESCE(p.order_count, 0)::bigint, COALESCE(u.phone, ''),
		       COALESCE(p.description, ''), COALESCE(p.mutual_intro, ''), p.user_id,
		       (SELECT count(*) FROM works w WHERE w.photographer_id = p.id)::bigint,
		       (SELECT count(*) FROM reviews r WHERE r.photographer_id = p.id)::bigint
		FROM photographers p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.id = $1`, id).Scan(
		&d.ID, &d.Name, &d.Avatar, &d.Location, &rating,
		&d.Certified, &d.Mode, &d.OrderCount, &d.UserPhone,
		&d.Description, &d.MutualIntro, &d.UserID, &d.WorksCount, &d.ReviewsCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminPhotographerDetail{}, ErrManageNotFound
		}
		return AdminPhotographerDetail{}, err
	}
	d.Rating = numericToFloat(rating)
	return d, nil
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.NaN {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return f8.Float64
}
