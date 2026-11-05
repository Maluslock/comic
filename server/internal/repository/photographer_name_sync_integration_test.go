package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 这两个用例盯的是「名字单向同步」的 SQL 谓词本身：
// 服务层单测只能证明参数传对了，真正决定「艺名会不会被冲掉」的是 WHERE 里的
// name_is_custom 判断，所以必须打到真库。
//
// TEST_DATABASE_URL 未设时整组跳过，`go test ./...` 在没有数据库的机器上照常可跑
// （与 services_pricing_integration_test.go 同一约定）。

func integrationUserRepo(t *testing.T) (*UserRepo, *pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed name-sync test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return NewUserRepo(pool), pool, ctx
}

// insertNameSyncFixture 造一组隔离数据：一个临时 user + 一条挂在他名下的摄影师档案。
// 名字带 __name_sync_it__ 前缀，便于事后识别；cleanup 按 id 精确删除，不碰任何真实行。
func insertNameSyncFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context, photographerName string, nameIsCustom bool) (userID, photographerID int64) {
	t.Helper()
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (name, avatar) VALUES ('__name_sync_it__user', '') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("insert user fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM photographers WHERE id = $1`, photographerID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if err := pool.QueryRow(ctx,
		`INSERT INTO photographers (name, user_id, mode, name_is_custom)
		 VALUES ($1, $2, 'both', $3) RETURNING id`,
		photographerName, userID, nameIsCustom).Scan(&photographerID); err != nil {
		t.Fatalf("insert photographer fixture: %v", err)
	}
	return userID, photographerID
}

func readPhotographerName(t *testing.T, pool *pgxpool.Pool, ctx context.Context, id int64) (string, bool) {
	t.Helper()
	var name string
	var custom bool
	if err := pool.QueryRow(ctx,
		`SELECT name, name_is_custom FROM photographers WHERE id = $1`, id).Scan(&name, &custom); err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return name, custom
}

// 用户没设过艺名（name_is_custom=false）时，账号昵称必须能同步下来 ——
// 这就是「改昵称后搜不到自己」的修复点；同步成功后要返回 true 让调用方去清缓存。
func TestIntegration_SyncPhotographerName_FollowsNicknameWhenNotCustom(t *testing.T) {
	repo, pool, ctx := integrationUserRepo(t)
	userID, photographerID := insertNameSyncFixture(t, pool, ctx, "旧名字", false)

	synced, err := repo.SyncPhotographerName(ctx, userID, "MalusLock")
	if err != nil {
		t.Fatalf("SyncPhotographerName: %v", err)
	}
	if !synced {
		t.Fatal("want synced=true when the row actually changed, got false")
	}
	name, custom := readPhotographerName(t, pool, ctx, photographerID)
	if name != "MalusLock" {
		t.Fatalf("name must follow the account nickname, got %q", name)
	}
	if custom {
		t.Fatal("name_is_custom must stay false, sync must not lock the name")
	}
}

// 名字没变就必须返回 false：调用方靠这个返回值决定要不要清缓存，
// 报 true 会让每次保存资料都白扫一遍 Redis。
func TestIntegration_SyncPhotographerName_NoopWhenNameUnchanged(t *testing.T) {
	repo, pool, ctx := integrationUserRepo(t)
	userID, photographerID := insertNameSyncFixture(t, pool, ctx, "同名", false)

	synced, err := repo.SyncPhotographerName(ctx, userID, "同名")
	if err != nil {
		t.Fatalf("SyncPhotographerName: %v", err)
	}
	if synced {
		t.Fatal("same name must not report a sync, got true")
	}
	if name, _ := readPhotographerName(t, pool, ctx, photographerID); name != "同名" {
		t.Fatalf("name must be untouched, got %q", name)
	}
}

// 用户显式设过艺名（name_is_custom=true）时，账号昵称一律不许覆盖它。
func TestIntegration_SyncPhotographerName_KeepsCustomStageName(t *testing.T) {
	repo, pool, ctx := integrationUserRepo(t)
	userID, photographerID := insertNameSyncFixture(t, pool, ctx, "艺名独立", true)

	synced, err := repo.SyncPhotographerName(ctx, userID, "账号昵称随便改")
	if err != nil {
		t.Fatalf("SyncPhotographerName: %v", err)
	}
	if synced {
		t.Fatal("a custom stage name must never be overwritten, got synced=true")
	}
	name, custom := readPhotographerName(t, pool, ctx, photographerID)
	if name != "艺名独立" {
		t.Fatalf("stage name must survive, got %q", name)
	}
	if !custom {
		t.Fatal("name_is_custom must stay true")
	}
}

// Activate 幂等分支的窄写：只改 mode 时不得把 name_is_custom 刷成 true，
// 否则一个从没设过艺名的用户展示名会被永久锁死，账号昵称再也同步不过来。
func TestIntegration_UpdatePhotographerActivationFields_OnlyLocksWhenNameOverridden(t *testing.T) {
	repo, pool, ctx := integrationUserRepo(t)
	_ = repo
	q := New(pool)
	_, photographerID := insertNameSyncFixture(t, pool, ctx, "光影行者", false)

	if err := q.UpdatePhotographerActivationFields(ctx, photographerID, "光影行者", "pay", nil, false); err != nil {
		t.Fatalf("mode-only update: %v", err)
	}
	name, custom := readPhotographerName(t, pool, ctx, photographerID)
	if name != "光影行者" || custom {
		t.Fatalf("mode-only update must leave name and marker alone, got name=%q custom=%v", name, custom)
	}

	// 真的覆盖了占位名 → 标记置 true，从此账号昵称不许再冲掉它。
	if err := q.UpdatePhotographerActivationFields(ctx, photographerID, "新艺名", "pay", nil, true); err != nil {
		t.Fatalf("rename update: %v", err)
	}
	name, custom = readPhotographerName(t, pool, ctx, photographerID)
	if name != "新艺名" || !custom {
		t.Fatalf("a real override must lock the name, got name=%q custom=%v", name, custom)
	}
}

// updatePhotographerProfile 是「保存整个摄影师档案」的全量写入：用户可能只改城市或简介，
// 名字一个字没动。那种情况绝不能把 name_is_custom 刷成 true —— 一旦刷了，账号昵称就再也
// 同步不过来，等于把「我改了昵称搜不到自己」原样搬到一个更隐蔽的入口（编辑所在地就锁名）。
// 回归防线：只有 name 真的与旧值不同才置标记。
func TestIntegration_UpdatePhotographerProfile_OnlyLocksWhenNameChanged(t *testing.T) {
	repo, pool, ctx := integrationUserRepo(t)
	_ = repo
	q := New(pool)
	userID, photographerID := insertNameSyncFixture(t, pool, ctx, "光影行者", false)

	// 只改 location / description，name 原样传回。
	if err := q.UpdatePhotographerProfile(ctx, photographerID, "光影行者", "新的简介", "北京", "both", nil, ""); err != nil {
		t.Fatalf("location-only save: %v", err)
	}
	name, custom := readPhotographerName(t, pool, ctx, photographerID)
	if name != "光影行者" || custom {
		t.Fatalf("saving without renaming must keep marker false, got name=%q custom=%v", name, custom)
	}

	// 账号昵称仍然能同步下来（这正是标记为 false 的意义）。
	if _, err := repo.SyncPhotographerName(ctx, userID, "新账号昵称"); err != nil {
		t.Fatalf("sync after location-only save: %v", err)
	}
	name, custom = readPhotographerName(t, pool, ctx, photographerID)
	if name != "新账号昵称" || custom {
		t.Fatalf("sync must still work after a non-rename save, got name=%q custom=%v", name, custom)
	}

	// 真的改名 → 标记置 true，从此账号昵称不许再冲掉它。
	if err := q.UpdatePhotographerProfile(ctx, photographerID, "自己起的艺名", "", "北京", "both", nil, ""); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	name, custom = readPhotographerName(t, pool, ctx, photographerID)
	if name != "自己起的艺名" || !custom {
		t.Fatalf("a real rename must lock the name, got name=%q custom=%v", name, custom)
	}
}
