package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRow struct {
	ID             int64
	Username       string
	PasswordHash   string
	Role           string
	Status         string
	Token          *string
	TokenExpiresAt *time.Time
	CreatedAt      string
}

var ErrAdminNotFound = errors.New("admin not found")

type AdminRepo struct {
	pool *pgxpool.Pool
}

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo { return &AdminRepo{pool: pool} }

const adminCols = "id, username, password_hash, role, status, token, token_expires_at, created_at::text"

func scanAdmin(scan func(dest ...any) error) (*AdminRow, error) {
	var a AdminRow
	err := scan(&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.Status, &a.Token, &a.TokenExpiresAt, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AdminRepo) GetAdminByUsername(ctx context.Context, username string) (*AdminRow, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx,
		"SELECT "+adminCols+" FROM admins WHERE username = $1", username).Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminNotFound
	}
	return a, err
}

func (r *AdminRepo) GetAdminByToken(ctx context.Context, token string) (*AdminRow, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx,
		"SELECT "+adminCols+" FROM admins WHERE token = $1 AND token_expires_at > NOW() AND status = 'active'", token).Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminNotFound
	}
	return a, err
}

func (r *AdminRepo) GetByID(ctx context.Context, id int64) (*AdminRow, error) {
	a, err := scanAdmin(r.pool.QueryRow(ctx,
		"SELECT "+adminCols+" FROM admins WHERE id = $1", id).Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminNotFound
	}
	return a, err
}

func (r *AdminRepo) List(ctx context.Context) ([]AdminRow, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+adminCols+" FROM admins ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminRow, 0)
	for rows.Next() {
		a, err := scanAdmin(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, *a)
	}
	return items, rows.Err()
}

func (r *AdminRepo) SetStatus(ctx context.Context, id int64, status string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE admins SET status = $2 WHERE id = $1",
		id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *AdminRepo) ClearToken(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE admins SET token = NULL, token_expires_at = NULL WHERE id = $1",
		id)
	return err
}

func (r *AdminRepo) Create(ctx context.Context, username, passwordHash, role string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		"INSERT INTO admins (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id",
		username, passwordHash, role).Scan(&id)
	return id, err
}

func (r *AdminRepo) SetToken(ctx context.Context, id int64, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE admins SET token = $2, token_expires_at = $3 WHERE id = $1",
		id, token, expiresAt)
	return err
}

func (r *AdminRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE admins SET password_hash = $2 WHERE id = $1",
		id, hash)
	return err
}
