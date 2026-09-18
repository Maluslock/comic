package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrContentNotFound signals that a content row (work/review/tag) matched no rows.
var ErrContentNotFound = errors.New("admin content: not found")

// ErrTagInUse signals that a tag still has photographer references.
var ErrTagInUse = errors.New("tag in use")

type AdminWork struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	Images           []string  `json:"images"`
	PhotographerName string    `json:"photographerName"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AdminReview struct {
	ID               int64     `json:"id"`
	UserName         string    `json:"userName"`
	UserAvatar       string    `json:"userAvatar"`
	Rating           int32     `json:"rating"`
	Content          string    `json:"content"`
	PhotographerName string    `json:"photographerName"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AdminTag struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	UsageCount int64  `json:"usageCount"`
}

type AdminContentRepo struct {
	pool *pgxpool.Pool
}

func NewAdminContentRepo(pool *pgxpool.Pool) *AdminContentRepo {
	return &AdminContentRepo{pool: pool}
}

func (r *AdminContentRepo) ListWorks(ctx context.Context, photographerID *int64, status string, limit, offset int) ([]AdminWork, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM works w
		WHERE ($1::bigint IS NULL OR w.photographer_id = $1)
		  AND ($2::text = '' OR w.status = $2)`, photographerID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.title, w.images, COALESCE(p.name, ''), w.status, w.created_at
		FROM works w
		LEFT JOIN photographers p ON p.id = w.photographer_id
		WHERE ($1::bigint IS NULL OR w.photographer_id = $1)
		  AND ($2::text = '' OR w.status = $2)
		ORDER BY w.created_at DESC
		LIMIT $3 OFFSET $4`, photographerID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminWork, 0, limit)
	for rows.Next() {
		var i AdminWork
		if err := rows.Scan(&i.ID, &i.Title, &i.Images, &i.PhotographerName, &i.Status, &i.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *AdminContentRepo) SetWorkStatus(ctx context.Context, id int64, status string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE works SET status = $2, updated_at = NOW() WHERE id = $1", id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrContentNotFound
	}
	return nil
}

func (r *AdminContentRepo) ListReviews(ctx context.Context, keyword string, photographerID *int64, limit, offset int) ([]AdminReview, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM reviews r
		WHERE ($1::text = '' OR r.content ILIKE '%' || $1 || '%' OR r.user_name ILIKE '%' || $1 || '%')
		  AND ($2::bigint IS NULL OR r.photographer_id = $2)`, keyword, photographerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT r.id, COALESCE(r.user_name, ''), COALESCE(r.user_avatar, ''), r.rating,
		       COALESCE(r.content, ''), COALESCE(p.name, ''), r.created_at
		FROM reviews r
		LEFT JOIN photographers p ON p.id = r.photographer_id
		WHERE ($1::text = '' OR r.content ILIKE '%' || $1 || '%' OR r.user_name ILIKE '%' || $1 || '%')
		  AND ($2::bigint IS NULL OR r.photographer_id = $2)
		ORDER BY r.created_at DESC
		LIMIT $3 OFFSET $4`, keyword, photographerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminReview, 0, limit)
	for rows.Next() {
		var i AdminReview
		if err := rows.Scan(&i.ID, &i.UserName, &i.UserAvatar, &i.Rating,
			&i.Content, &i.PhotographerName, &i.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *AdminContentRepo) DeleteReview(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM reviews WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrContentNotFound
	}
	return nil
}

func (r *AdminContentRepo) ListTags(ctx context.Context) ([]AdminTag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name,
		       (SELECT COUNT(*) FROM photographer_tags pt WHERE pt.tag_id = t.id) AS usage_count
		FROM tags t
		ORDER BY t.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AdminTag
	for rows.Next() {
		var i AdminTag
		if err := rows.Scan(&i.ID, &i.Name, &i.UsageCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *AdminContentRepo) CreateTag(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		"INSERT INTO tags (name) VALUES ($1) RETURNING id", name).Scan(&id)
	return id, err
}

func (r *AdminContentRepo) UpdateTag(ctx context.Context, id int64, name string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE tags SET name = $2 WHERE id = $1", id, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrContentNotFound
	}
	return nil
}

func (r *AdminContentRepo) MergeTag(ctx context.Context, fromID, toID int64) error {
	var n int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM tags WHERE id IN ($1, $2)", fromID, toID).Scan(&n); err != nil {
		return err
	}
	if n < 2 {
		return ErrContentNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO photographer_tags (photographer_id, tag_id)
		SELECT photographer_id, $2 FROM photographer_tags WHERE tag_id = $1
		ON CONFLICT DO NOTHING`, fromID, toID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM photographer_tags WHERE tag_id = $1", fromID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM tags WHERE id = $1", fromID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AdminContentRepo) DeleteTag(ctx context.Context, id int64) error {
	var refs int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM photographer_tags WHERE tag_id = $1", id).Scan(&refs); err != nil {
		return err
	}
	if refs > 0 {
		return ErrTagInUse
	}
	tag, err := r.pool.Exec(ctx, "DELETE FROM tags WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrContentNotFound
	}
	return nil
}
