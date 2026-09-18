package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrReviewConflict signals that a conditional review update matched no
// pending row — the application was concurrently reviewed or deleted.
var ErrReviewConflict = errors.New("cert application review conflict")

// CertApplication is the raw row shape of photographer_cert_applications.
type CertApplication struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"userId"`
	PhotographerID int64      `json:"photographerId"`
	EvidenceImages []string   `json:"evidenceImages"`
	EvidenceDesc   string     `json:"evidenceDesc"`
	Status         string     `json:"status"`
	ReviewReason   *string    `json:"reviewReason,omitempty"`
	AdminID        *int64     `json:"adminId,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// AdminCertApplication is the admin-facing view with joined user/photographer names.
type AdminCertApplication struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"userId"`
	UserName         string     `json:"userName"`
	PhotographerID   int64      `json:"photographerId"`
	PhotographerName string     `json:"photographerName"`
	EvidenceImages   []string   `json:"evidenceImages"`
	EvidenceDesc     string     `json:"evidenceDesc"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	ReviewReason     *string    `json:"reviewReason,omitempty"`
	AdminID          *int64     `json:"adminId,omitempty"`
	ReviewedAt       *time.Time `json:"reviewedAt,omitempty"`
}

type CertApplicationRepo struct {
	pool *pgxpool.Pool
}

func NewCertApplicationRepo(pool *pgxpool.Pool) *CertApplicationRepo {
	return &CertApplicationRepo{pool: pool}
}

func (r *CertApplicationRepo) Create(ctx context.Context, userID, photographerID int64, evidenceImages []string, evidenceDesc string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO photographer_cert_applications (user_id, photographer_id, evidence_images, evidence_desc)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, userID, photographerID, evidenceImages, evidenceDesc).Scan(&id)
	return id, err
}

// GetByPhotographerID returns the latest application for the photographer;
// pgx.ErrNoRows is returned as-is when the photographer never applied.
func (r *CertApplicationRepo) GetByPhotographerID(ctx context.Context, photographerID int64) (*CertApplication, error) {
	var a CertApplication
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, photographer_id, evidence_images, evidence_desc,
		       status, review_reason, admin_id, reviewed_at, created_at
		FROM photographer_cert_applications
		WHERE photographer_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, photographerID,
	).Scan(&a.ID, &a.UserID, &a.PhotographerID, &a.EvidenceImages, &a.EvidenceDesc,
		&a.Status, &a.ReviewReason, &a.AdminID, &a.ReviewedAt, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *CertApplicationRepo) ListByPhotographerID(ctx context.Context, photographerID int64) ([]CertApplication, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, photographer_id, evidence_images, evidence_desc,
		       status, review_reason, admin_id, reviewed_at, created_at
		FROM photographer_cert_applications
		WHERE photographer_id = $1
		ORDER BY created_at DESC`, photographerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CertApplication
	for rows.Next() {
		var a CertApplication
		if err := rows.Scan(&a.ID, &a.UserID, &a.PhotographerID, &a.EvidenceImages, &a.EvidenceDesc,
			&a.Status, &a.ReviewReason, &a.AdminID, &a.ReviewedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *CertApplicationRepo) List(ctx context.Context, status string, limit, offset int) ([]AdminCertApplication, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM photographer_cert_applications
		WHERE ($1::text = '' OR status = $1)`, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.user_id, COALESCE(u.name, ''), a.photographer_id, COALESCE(p.name, ''),
		       a.evidence_images, a.evidence_desc, a.status, a.created_at,
		       a.review_reason, a.admin_id, a.reviewed_at
		FROM photographer_cert_applications a
		LEFT JOIN users u ON u.id = a.user_id
		LEFT JOIN photographers p ON p.id = a.photographer_id
		WHERE ($1::text = '' OR a.status = $1)
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminCertApplication, 0, limit)
	for rows.Next() {
		var i AdminCertApplication
		if err := rows.Scan(&i.ID, &i.UserID, &i.UserName, &i.PhotographerID, &i.PhotographerName,
			&i.EvidenceImages, &i.EvidenceDesc, &i.Status, &i.CreatedAt,
			&i.ReviewReason, &i.AdminID, &i.ReviewedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *CertApplicationRepo) GetByID(ctx context.Context, id int64) (*AdminCertApplication, error) {
	var i AdminCertApplication
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.user_id, COALESCE(u.name, ''), a.photographer_id, COALESCE(p.name, ''),
		       a.evidence_images, a.evidence_desc, a.status, a.created_at,
		       a.review_reason, a.admin_id, a.reviewed_at
		FROM photographer_cert_applications a
		LEFT JOIN users u ON u.id = a.user_id
		LEFT JOIN photographers p ON p.id = a.photographer_id
		WHERE a.id = $1`, id,
	).Scan(&i.ID, &i.UserID, &i.UserName, &i.PhotographerID, &i.PhotographerName,
		&i.EvidenceImages, &i.EvidenceDesc, &i.Status, &i.CreatedAt,
		&i.ReviewReason, &i.AdminID, &i.ReviewedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrManageNotFound
		}
		return nil, err
	}
	return &i, nil
}

// Review updates an application only while it is still pending (reject path —
// a single write, no transaction needed). ErrReviewConflict is returned when
// the conditional update matched no row; the service re-reads to resolve the
// race into 409 (already reviewed) or 404 (deleted).
func (r *CertApplicationRepo) Review(ctx context.Context, id int64, action string, reason string, adminID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE photographer_cert_applications
		SET status = $2,
		    review_reason = CASE WHEN $3 = '' THEN NULL ELSE $3 END,
		    admin_id = $4,
		    reviewed_at = NOW()
		WHERE id = $1 AND status = 'pending'`, id, action, reason, adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewConflict
	}
	return nil
}

// ReviewApproveTx atomically marks the application approved and flips the
// photographer's certified flag in ONE transaction, so an approved row can
// never be left with certified=false. ErrReviewConflict means the application
// was concurrently reviewed (or deleted).
func (r *CertApplicationRepo) ReviewApproveTx(ctx context.Context, id int64, adminID int64, photographerID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE photographer_cert_applications
		SET status = 'approved', review_reason = NULL, admin_id = $2, reviewed_at = NOW()
		WHERE id = $1 AND status = 'pending'`, id, adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewConflict
	}

	if _, err := tx.Exec(ctx,
		"UPDATE photographers SET certified = true WHERE id = $1", photographerID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CertApplicationRepo) SetCertifiedByPhotographerID(ctx context.Context, photographerID int64, certified bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE photographers SET certified = $2 WHERE id = $1", photographerID, certified)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrManageNotFound
	}
	return nil
}
