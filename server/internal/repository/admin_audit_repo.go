package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLog struct {
	ID        int64  `json:"id"`
	AdminID   int64  `json:"adminId"`
	AdminName string `json:"adminName"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type AdminAuditRepo struct {
	pool *pgxpool.Pool
}

func NewAdminAuditRepo(pool *pgxpool.Pool) *AdminAuditRepo {
	return &AdminAuditRepo{pool: pool}
}

func (r *AdminAuditRepo) Insert(ctx context.Context, adminID int64, method, path string, status int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO admin_audit_logs (admin_id, method, path, status) VALUES ($1, $2, $3, $4)`,
		adminID, method, path, status)
	return err
}

func (r *AdminAuditRepo) List(ctx context.Context, limit, offset int) ([]AuditLog, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.admin_id, COALESCE(ad.username, ''), a.method, a.path, a.status, a.created_at::text
		FROM admin_audit_logs a
		LEFT JOIN admins ad ON ad.id = a.admin_id
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []AuditLog
	for rows.Next() {
		var i AuditLog
		if err := rows.Scan(&i.ID, &i.AdminID, &i.AdminName, &i.Method, &i.Path, &i.Status, &i.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}
