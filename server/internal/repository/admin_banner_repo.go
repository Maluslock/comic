package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBannerNotFound = errors.New("banner not found")

type AdminBanner struct {
	ID        int64  `json:"id"`
	ImageURL  string `json:"imageUrl"`
	Title     string `json:"title"`
	LinkType  string `json:"linkType"`
	LinkID    int32  `json:"linkId"`
	SortOrder int32  `json:"sortOrder"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type AdminBannerRepo struct {
	pool *pgxpool.Pool
}

func NewAdminBannerRepo(pool *pgxpool.Pool) *AdminBannerRepo {
	return &AdminBannerRepo{pool: pool}
}

const adminBannerColumns = `id, image_url, title, link_type, COALESCE(link_id, 0), COALESCE(sort_order, 0), is_active, created_at::text`

func scanAdminBanner(scan func(dest ...any) error) (AdminBanner, error) {
	var b AdminBanner
	err := scan(&b.ID, &b.ImageURL, &b.Title, &b.LinkType, &b.LinkID, &b.SortOrder, &b.IsActive, &b.CreatedAt)
	return b, err
}

func (r *AdminBannerRepo) List(ctx context.Context) ([]AdminBanner, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+adminBannerColumns+` FROM banners ORDER BY sort_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminBanner, 0)
	for rows.Next() {
		b, err := scanAdminBanner(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (r *AdminBannerRepo) Create(ctx context.Context, imageURL, title, linkType string, linkID, sortOrder int32) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO banners (image_url, title, link_type, link_id, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, imageURL, title, linkType, linkID, sortOrder).Scan(&id)
	return id, err
}

func (r *AdminBannerRepo) GetByID(ctx context.Context, id int64) (*AdminBanner, error) {
	b, err := scanAdminBanner(r.pool.QueryRow(ctx,
		`SELECT `+adminBannerColumns+` FROM banners WHERE id = $1`, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBannerNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *AdminBannerRepo) Update(ctx context.Context, id int64, imageURL, title, linkType string, linkID, sortOrder int32, isActive bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE banners
		SET image_url = $2, title = $3, link_type = $4, link_id = $5, sort_order = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1`, id, imageURL, title, linkType, linkID, sortOrder, isActive)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBannerNotFound
	}
	return nil
}

func (r *AdminBannerRepo) SetStatus(ctx context.Context, id int64, isActive bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE banners SET is_active = $2, updated_at = NOW() WHERE id = $1", id, isActive)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBannerNotFound
	}
	return nil
}
