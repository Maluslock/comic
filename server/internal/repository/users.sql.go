package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        int64     `json:"id"`
	Phone     string    `json:"phone"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
	Bio       *string   `json:"bio,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

func (r *UserRepo) UpsertByPhone(ctx context.Context, phone, name, avatar string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (phone, name, avatar)
		VALUES ($1, $2, $3)
		ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, avatar = EXCLUDED.avatar
		RETURNING id, COALESCE(phone, ''), name, avatar, bio, status, created_at`,
		phone, name, avatar,
	).Scan(&u.ID, &u.Phone, &u.Name, &u.Avatar, &u.Bio, &u.Status, &u.CreatedAt)
	return u, err
}

// UpsertByWeChat 按 openid 建号或更新昵称/头像。微信登录用户可能没有手机号，
// 因此 phone 留空；同一用户后续绑定手机号时由绑定流程补写。
func (r *UserRepo) UpsertByWeChat(ctx context.Context, openid, unionid, name, avatar string) (User, error) {
	var u User
	var union *string
	if unionid != "" {
		union = &unionid
	}
	// 空昵称/头像表示「本次没带资料」：插入时给默认值，冲突时保留库里已有的，
	// 否则用户第二次静默登录会把微信昵称覆盖成「微信用户xxxx」。
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (wx_openid, wx_unionid, name, avatar)
		VALUES ($1, $2, COALESCE(NULLIF($3, ''), '微信用户'), $4)
		ON CONFLICT (wx_openid) DO UPDATE
		SET name = COALESCE(NULLIF($3, ''), users.name),
		    avatar = COALESCE(NULLIF($4, ''), users.avatar),
		    wx_unionid = COALESCE(EXCLUDED.wx_unionid, users.wx_unionid)
		RETURNING id, COALESCE(phone, ''), name, avatar, bio, status, created_at`,
		openid, union, name, avatar,
	).Scan(&u.ID, &u.Phone, &u.Name, &u.Avatar, &u.Bio, &u.Status, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		"SELECT id, COALESCE(phone, ''), name, avatar, bio, status, created_at FROM users WHERE id = $1", id,
	).Scan(&u.ID, &u.Phone, &u.Name, &u.Avatar, &u.Bio, &u.Status, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id int64, name, avatar, bio string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET name = $2, avatar = $3, bio = $4 WHERE id = $1", id, name, avatar, bio,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepo) CreateToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO user_tokens (token, user_id, expires_at) VALUES ($1, $2, $3)",
		token, userID, expiresAt,
	)
	return err
}

func (r *UserRepo) DeleteTokensByUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM user_tokens WHERE user_id = $1", userID)
	return err
}

func (r *UserRepo) GetUserByToken(ctx context.Context, token string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, COALESCE(u.phone, ''), u.name, u.avatar, u.status, u.created_at
		FROM user_tokens t JOIN users u ON u.id = t.user_id
		WHERE t.token = $1 AND t.expires_at > NOW()`, token,
	).Scan(&u.ID, &u.Phone, &u.Name, &u.Avatar, &u.Status, &u.CreatedAt)
	return u, err
}

// GetOpenID returns the WeChat openid bound to a user. Phone-only accounts have
// none, so callers must treat "" as "cannot push".
func (r *UserRepo) GetOpenID(ctx context.Context, id int64) (string, error) {
	var openid *string
	err := r.pool.QueryRow(ctx, "SELECT wx_openid FROM users WHERE id = $1", id).Scan(&openid)
	if err != nil {
		return "", err
	}
	if openid == nil {
		return "", nil
	}
	return *openid, nil
}
