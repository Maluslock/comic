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

// UpdateNickname 只写 users.name（账号昵称），**永远不碰 photographers 表**。
//
// 口径（单向跟随，方向恒为 photographers → users）：
//   - photographers.name 是「对外展示的摄影师艺名」，只有两个入口能改——
//     摄影师本人在 PUT /api/v1/photographers/profile 里改、或 Activate 首次建档；
//     账号昵称没有资格覆盖艺名（否则任何人改自己昵称就能"改名"某个摄影师）。
//   - 本方法用于反向的那一半：摄影师主动改艺名 / 首次激活时，把同一个名字写进
//     users.name，让「我的」页和列表里显示一致。
//   - name 为空表示「本次没填」，直接返回不改库（避免清空昵称）。
func (r *UserRepo) UpdateNickname(ctx context.Context, userID int64, name string) error {
	if userID == 0 || name == "" {
		return nil
	}
	_, err := r.pool.Exec(ctx, "UPDATE users SET name = $2 WHERE id = $1", userID, name)
	return err
}

// UpdateAvatar 只写 users.avatar（账号头像，用于「我的」页、聊天头像）。
//
// 为什么需要单独一个方法：`PUT /api/v1/me/profile` 走的 UpdateProfile 要求 name 必填，
// 而「上传头像」只有一个头像 URL 要落库。此前头像只能靠整份资料一起提交，
// 于是出现「我上传了头像但页面还是默认图」——见 SyncPhotographerAvatar 的注释。
func (r *UserRepo) UpdateAvatar(ctx context.Context, userID int64, avatar string) error {
	if userID == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, "UPDATE users SET avatar = $2 WHERE id = $1", userID, avatar)
	return err
}

// SyncPhotographerName 把账号昵称同步到该用户自己的摄影师档案（users → photographers）。
//
// 根因记录：名字在库里有两列 —— users.name（账号昵称，「我的」页与聊天里显示）
// 与 photographers.name（对外展示的摄影师名，列表 / 搜索 / 详情 / 预约卡片读的全是这列）。
// 摄影师列表与搜索的 SQL（searchPhotographers）只匹配 p.name / p.description，
// 而 `PUT /api/v1/me/profile`（「编辑资料」页）此前只写 users.name。
// 结果：用户把昵称改成 MalusLock，「我的」页变了，摄影师列表里还是注册时那个名字，
// 搜自己的新昵称 total=0 —— 用户原话「我注册了摄影师，叫 MalusLock，搜索摄影师没结果」。
//
// 语义边界（与 SyncPhotographerAvatar 同款：只碰自己的行、只碰该跟随的列）：
//   - 只更新该 user_id 自己的档案，绝不跨账号。
//   - name_is_custom = false 才跟随：摄影师一旦在「摄影师资料」页显式设过艺名
//     （写路径 UpdatePhotographerProfile 会把标记置 true），那个名字是品牌资产，
//     账号昵称没有资格覆盖它。
//   - COALESCE(name, 空串) <> $2 才写：名字没变就不产生无意义的 UPDATE，
//     updated_at 不该因为「提交了一份没改名的表单」而跳动。
//
// 返回值 bool = 真的改到了行。调用方据此决定要不要清缓存：
// 只清「确实变了」的情况，避免每次保存资料都白扫一遍 Redis。
func (r *UserRepo) SyncPhotographerName(ctx context.Context, userID int64, name string) (bool, error) {
	if userID == 0 || name == "" {
		return false, nil
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE photographers
		SET name = $2, updated_at = NOW()
		WHERE user_id = $1
		  AND name_is_custom = false
		  AND COALESCE(name, '') <> $2`,
		userID, name)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SyncPhotographerAvatar 把账号头像同步到该用户自己的摄影师档案。
//
// 根因记录：头像在库里有两列 —— users.avatar（账号头像）与 photographers.avatar
// （对外展示的摄影师头像）。摄影师列表、详情页、预约页卡片读的全是
// photographers.avatar，而 `PUT /api/v1/me/profile` 与微信登录只写 users.avatar。
// 结果：用户在「资料编辑」里成功换了头像，「我的」页也变了，但一进摄影师列表
// 自己的卡片还是默认图 —— 用户原话「上传了头像，进去之后是阿伟摄影，然后头像
// 还是默认的」。
//
// 语义边界（与 syncAccountName 方向相反，各自只负责自己的列）：
//   - 账号头像是「这个人长什么样」，摄影师头像是「这个摄影师对外展示成什么样」，
//     两者默认应一致，故账号头像变化时向下同步。
//   - 但摄影师若已在「摄影师资料」页单独设过头像（photographers.avatar 非空且不等于
//     旧账号头像），说明是刻意区分，此时不覆盖。
//   - 只更新该 user_id 自己的档案，绝不跨账号。
func (r *UserRepo) SyncPhotographerAvatar(ctx context.Context, userID int64, avatar, prevAvatar string) error {
	if userID == 0 || avatar == "" {
		return nil
	}
	// prevAvatar 为空说明之前没有账号头像，此时档案头像要么为空要么是系统默认图，
	// 直接覆盖即可；prevAvatar 非空时只在档案头像「仍等于旧账号头像」的情况下跟随，
	// 避免抹掉摄影师单独设置的头像。
	_, err := r.pool.Exec(ctx, `
		UPDATE photographers
		SET avatar = $2, updated_at = NOW()
		WHERE user_id = $1
		  AND ($3 = '' OR COALESCE(avatar, '') = '' OR avatar = $3)`,
		userID, avatar, prevAvatar)
	return err
}
