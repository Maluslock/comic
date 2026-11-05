package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Maluslock/comic/server/internal/repository"
)

// fakeNameSync 记录「账号昵称」侧被写了几次、写了什么。
// 真实实现是 repository.UserRepo.UpdateNickname（只 UPDATE users.name）。
type fakeNameSync struct {
	calls []struct {
		userID int64
		name   string
	}
	err error
}

func (f *fakeNameSync) UpdateNickname(_ context.Context, userID int64, name string) error {
	f.calls = append(f.calls, struct {
		userID int64
		name   string
	}{userID, name})
	return f.err
}

// photographerByUserScanValues 是 GetPhotographerByUserID 的扫描顺序（12 列，无 tags）。
func photographerByUserScanValues(id int32, name string, userID *int64, mode string, intro *string) []any {
	return []any{
		id, name, (*string)(nil), (*string)(nil), (*string)(nil),
		pgtype.Numeric{}, (*int32)(nil), (*int32)(nil),
		userID, mode, false, intro,
	}
}

func newIdentityTestSvc(store worksStore, db repository.DBTX, names userNameSync) *PhotographerService {
	return &PhotographerService{queries: repository.New(db), works: store, users: names}
}

// 线上那条脏数据：photographers#15.name = '11'（纯数字占位）。用户重复点「开通」并
// 带上真名时，必须能修掉。
func TestActivate_Idempotent_ReplacesPlaceholderName(t *testing.T) {
	userID := int64(147)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 15, Name: "11", Mode: "both", UserID: &userID,
	}}
	names := &fakeNameSync{}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(15, "11", &userID, "both", nil)},
	}}
	svc := newIdentityTestSvc(store, db, names)

	id, err := svc.Activate(context.Background(), userID, "光影行者", "both", "")
	if err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if id != 15 {
		t.Fatalf("Activate must return the existing photographer id 15, got %d", id)
	}
	if store.updateActivationID != 15 {
		t.Fatalf("placeholder name must be repaired, UpdatePhotographerActivationFields id=%d", store.updateActivationID)
	}
	if store.updateActivationName != "光影行者" {
		t.Fatalf("want name 光影行者, got %q", store.updateActivationName)
	}
	if !store.updateActivationNameOverridden {
		t.Fatal("repairing a placeholder is a real name override, name_is_custom must be set")
	}
	// users.name 也要跟上（但只写 users，不写 photographers——后者上面已单独写）
	if len(names.calls) != 1 || names.calls[0].userID != userID || names.calls[0].name != "光影行者" {
		t.Fatalf("account nickname must follow the repair, got %+v", names.calls)
	}
}

// 摄影师自己经营的艺名是品牌资产：重复激活不许把它冲掉。
func TestActivate_Idempotent_KeepsRealStageName(t *testing.T) {
	userID := int64(200)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 7, Name: "光影行者", Mode: "both", UserID: &userID,
	}}
	names := &fakeNameSync{}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(7, "光影行者", &userID, "both", nil)},
	}}
	svc := newIdentityTestSvc(store, db, names)

	id, err := svc.Activate(context.Background(), userID, "阿伟", "both", "")
	if err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("want existing id 7, got %d", id)
	}
	if store.updateActivationID != 0 {
		t.Fatalf("a real stage name must not be overwritten by re-activation, UpdatePhotographerActivationFields ran with id=%d name=%q",
			store.updateActivationID, store.updateActivationName)
	}
	if len(names.calls) != 0 {
		t.Fatalf("account nickname must not be touched when nothing changed, got %+v", names.calls)
	}
}

// 占位符 + 入参为空 = 不该凭空清空，也不该白跑一次 UPDATE。
func TestActivate_Idempotent_PlaceholderWithoutNameIsNoop(t *testing.T) {
	userID := int64(147)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 15, Name: "11", Mode: "both", UserID: &userID,
	}}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(15, "11", &userID, "both", nil)},
	}}
	svc := newIdentityTestSvc(store, db, &fakeNameSync{})

	if _, err := svc.Activate(context.Background(), userID, "", "both", ""); err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if store.updateActivationID != 0 {
		t.Fatalf("empty name must not trigger a profile UPDATE, ran with id=%d", store.updateActivationID)
	}
}

// 迁移 000042 给「两边都是占位值」的档案落的兜底名「摄影师15」也必须可被覆盖，
// 否则用户填了真名照样改不掉——这正是 t2 要解开的死锁。
func TestActivate_Idempotent_ReplacesMigrationFallbackName(t *testing.T) {
	userID := int64(147)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 15, Name: "摄影师15", Mode: "both", UserID: &userID,
	}}
	names := &fakeNameSync{}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(15, "摄影师15", &userID, "both", nil)},
	}}
	svc := newIdentityTestSvc(store, db, names)

	id, err := svc.Activate(context.Background(), userID, "阿伟", "both", "")
	if err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if id != 15 {
		t.Fatalf("want existing id 15, got %d", id)
	}
	if store.updateActivationID != 15 || store.updateActivationName != "阿伟" {
		t.Fatalf("fallback name 摄影师15 must be replaceable, got id=%d name=%q",
			store.updateActivationID, store.updateActivationName)
	}
	if !store.updateActivationNameOverridden {
		t.Fatal("replacing the fallback name is a real override, name_is_custom must be set")
	}
	if len(names.calls) != 1 || names.calls[0].name != "阿伟" {
		t.Fatalf("account nickname must follow, got %+v", names.calls)
	}
}

// 重复激活时 mode 非空才覆盖：空串表示这次没填。
func TestActivate_Idempotent_UpdatesModeWhenProvided(t *testing.T) {
	userID := int64(200)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 7, Name: "光影行者", Mode: "free", UserID: &userID,
	}}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(7, "光影行者", &userID, "free", nil)},
	}}
	svc := newIdentityTestSvc(store, db, &fakeNameSync{})

	if _, err := svc.Activate(context.Background(), userID, "光影行者", "pay", ""); err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if store.updateActivationMode != "pay" {
		t.Fatalf("want mode pay, got %q", store.updateActivationMode)
	}
	if store.updateActivationName != "光影行者" {
		t.Fatalf("name must stay untouched, got %q", store.updateActivationName)
	}
	if store.updateActivationNameOverridden {
		t.Fatal("mode-only update must not claim the name, name_is_custom must stay as-is")
	}
}

func TestActivate_Idempotent_RejectsInvalidMode(t *testing.T) {
	userID := int64(200)
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{
		ID: 7, Name: "光影行者", Mode: "free", UserID: &userID,
	}}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{values: photographerByUserScanValues(7, "光影行者", &userID, "free", nil)},
	}}
	svc := newIdentityTestSvc(store, db, &fakeNameSync{})

	_, err := svc.Activate(context.Background(), userID, "光影行者", "paid", "")
	if err == nil {
		t.Fatal("want ErrInvalidMode, got nil")
	}
	if store.updateActivationID != 0 {
		t.Fatalf("invalid mode must not write, ran with id=%d", store.updateActivationID)
	}
}

// 首次激活建档后，账号昵称跟着对外展示名走。
func TestActivate_FirstTime_SyncsAccountNickname(t *testing.T) {
	userID := int64(300)
	names := &fakeNameSync{}
	db := &fakeDBTX{rows: []pgx.Row{
		fakeRow{err: pgx.ErrNoRows}, // GetPhotographerByUserID → 还没有档案
		fakeRow{values: []any{
			int32(21), "新摄影师", (*string)(nil), (*string)(nil), (*string)(nil),
			pgtype.Numeric{}, (*int32)(nil), (*int32)(nil), &userID,
		}}, // InsertPhotographer（9 列：id 是 int32）
	}}
	svc := newIdentityTestSvc(nil, db, names)

	id, err := svc.Activate(context.Background(), userID, "新摄影师", "both", "")
	if err != nil {
		t.Fatalf("Activate unexpected error: %v", err)
	}
	if id != 21 {
		t.Fatalf("want new id 21, got %d", id)
	}
	if len(names.calls) != 1 || names.calls[0].name != "新摄影师" || names.calls[0].userID != userID {
		t.Fatalf("first activation must sync account nickname, got %+v", names.calls)
	}
}

// UpdateProfile 改的是「艺名」，方向恒为 photographers → users。
func TestUpdateProfile_SyncsAccountNickname(t *testing.T) {
	userID := int64(147)
	names := &fakeNameSync{}
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{ID: 15, Name: "旧名", UserID: &userID}}
	svc := newIdentityTestSvc(store, &fakeDBTX{}, names)

	err := svc.UpdateProfile(context.Background(), userID, ProfileUpdate{Name: "新艺名", Mode: "both"})
	if err != nil {
		t.Fatalf("UpdateProfile unexpected error: %v", err)
	}
	if len(names.calls) != 1 || names.calls[0].name != "新艺名" || names.calls[0].userID != userID {
		t.Fatalf("artist name must flow into account nickname, got %+v", names.calls)
	}
}

// 没有注入同步器时（单元测试/只装配 Queries 的场景）不能 panic。
func TestSyncAccountName_NilSyncIsSafe(t *testing.T) {
	store := &fakeWorksStore{profile: repository.PhotographerWithTags{ID: 15, Name: "旧名"}}
	svc := newIdentityTestSvc(store, &fakeDBTX{}, nil)
	if err := svc.UpdateProfile(context.Background(), 147, ProfileUpdate{Name: "新艺名", Mode: "both"}); err != nil {
		t.Fatalf("UpdateProfile must tolerate a nil nickname sink, got %v", err)
	}
}

func TestIsPlaceholderName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", true},
		{"   ", true},
		{"11", true},
		{"007", true},
		{"微信用户", true},
		{" 微信用户 ", true},
		// 迁移 000042 第二段落下的系统兜底名，必须可被真名覆盖
		{"摄影师15", true},
		{"摄影师1", true},
		{" 摄影师15 ", true},
		{"光影行者", false},
		{"11号摄影师", false},
		{"阿伟", false},
		{"A1", false},
		// 不是「摄影师+纯数字」就不算占位名：这是真人可能起的艺名
		{"摄影师", false},
		{"摄影师小王", false},
		{"摄影师15号", false},
		{"摄影师 15", false},
	}
	for _, c := range cases {
		if got := isPlaceholderName(c.name); got != c.want {
			t.Errorf("isPlaceholderName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

// MarkSelf 是纯函数、跑在 handler 里：未登录时全 false，登录时只标自己。
func TestMarkSelf(t *testing.T) {
	items := []PhotographerListItem{
		{PhotographerItem: PhotographerItem{ID: 15, UserID: 147}},
		{PhotographerItem: PhotographerItem{ID: 16, UserID: 200}},
		{PhotographerItem: PhotographerItem{ID: 17, UserID: 0}}, // 无主档案
	}

	anon := MarkSelf(append([]PhotographerListItem(nil), items...), 0)
	for i := range anon {
		if anon[i].IsSelf {
			t.Errorf("anonymous request must never mark isSelf, item %d", i)
		}
	}

	mine := MarkSelf(append([]PhotographerListItem(nil), items...), 147)
	if !mine[0].IsSelf {
		t.Error("viewer 147 must see photographer 15 as self")
	}
	if mine[1].IsSelf {
		t.Error("viewer 147 must not see photographer 16 as self")
	}
	if mine[2].IsSelf {
		t.Error("a row without user_id must never be marked as self")
	}
}
