package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/service"
)

// ============================================================================
// Activate 错误映射（M1）
//
// 此前 Activate 的 err 分支一律 500，把「mode 填错」这种纯客户端错误也报成
// 服务端故障：前端拿不到可用提示，日志里也失去区分度。
// ============================================================================

// activateDB 造一个「已有档案」的库：GetPhotographerByUserID 成功，
// UPDATE photographers 返回 affected。
func activateDB(affected int64) *scriptableDBTX {
	return &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			if strings.Contains(sql, "FROM photographers") {
				return &successRow{scanFn: func(dest ...interface{}) error {
					return scanInto(dest, photographerRowWithTags())
				}}
			}
			return &errorRow{err: pgx.ErrNoRows}
		},
		execFn: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE " + strconv.FormatInt(affected, 10)), nil
		},
	}
}

func TestPhotographerHandler_Activate_InvalidModeIs400(t *testing.T) {
	svc := makePhotographerService(activateDB(1))
	handler := NewPhotographerHandler(svc, nil, nil)

	body := map[string]string{"name": "阿伟", "mode": "paid"}
	w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/activate", "7", body, nil)
	handler.Activate(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "mode must be one of free/pay/both") {
		t.Errorf("expected actionable mode message, got %s", w.Body.String())
	}
}

// 档案在写入过程中消失（并发删号）→ 404，而不是 500。
func TestPhotographerHandler_Activate_ProfileNotFoundIs404(t *testing.T) {
	// 占位名 '11' 才会触发名字覆盖 → 走 UPDATE；UPDATE 返回 0 行即档案已消失。
	db := &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			if strings.Contains(sql, "FROM photographers") {
				return &successRow{scanFn: func(dest ...interface{}) error {
					row := photographerRowWithTags()
					row[1] = "11" // 占位名，触发名字覆盖 → 走 UPDATE
					return scanInto(dest, row)
				}}
			}
			return &errorRow{err: pgx.ErrNoRows}
		},
		execFn: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}
	handler := NewPhotographerHandler(makePhotographerService(db), nil, nil)

	body := map[string]string{"name": "阿伟", "mode": "both"}
	w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/activate", "7", body, nil)
	handler.Activate(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when the profile vanished, got %d: %s", w.Code, w.Body.String())
	}
}

// 成功后必须清掉所有渲染价格的缓存：展示名与 mode 都印在列表 / 首页 / 活动卡片上。
func TestPhotographerHandler_Activate_InvalidatesCaches(t *testing.T) {
	inv := &fakeInvalidator{}
	handler := NewPhotographerHandler(makePhotographerService(activateDB(1)), nil, nil)
	handler.invalidator = inv

	body := map[string]string{"name": "阿伟", "mode": "both"}
	w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/activate", "7", body, nil)
	handler.Activate(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	assertInvalidated(t, inv)
}

// 未登录必须在碰数据库之前拦下，也不能顺手清缓存。
func TestPhotographerHandler_Activate_Unauthorized(t *testing.T) {
	inv := &fakeInvalidator{}
	handler := NewPhotographerHandler(makePhotographerService(activateDB(1)), nil, nil)
	handler.invalidator = inv

	// newJSONGinContext 会强制写入 user_id=7，这里要的是「没有 user_id」，
	// 故手工构造 context。
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	raw, _ := json.Marshal(map[string]string{"name": "阿伟", "mode": "both"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/photographers/activate", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	handler.Activate(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if len(inv.prefixes) != 0 {
		t.Errorf("expected no invalidation on 401, got %v", inv.prefixes)
	}
}

// ============================================================================
// UpdateProfile 缓存失效（B4）
//
// 摄影师改完展示名 / 城市 / 简介，列表与首页卡片必须立刻变；否则他自己刷新
// 仍看到旧信息，会以为没保存成功而重复提交。
// ============================================================================

func TestPhotographerHandler_UpdateProfile_InvalidatesCaches(t *testing.T) {
	inv := &fakeInvalidator{}
	handler := NewPhotographerHandler(makePhotographerService(serviceHandlerDB(1)), nil, nil)
	handler.invalidator = inv

	body := service.ProfileUpdate{
		Name:        "光影行者",
		Description: "十年漫展跟拍",
		Location:    "上海",
		Mode:        "both",
	}
	w, ctx := newJSONGinContext(http.MethodPut, "/api/v1/photographers/profile", "7", body, nil)
	handler.UpdateProfile(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	assertInvalidated(t, inv)
}

// 校验失败（mode 非法）没有改到任何展示字段，不能清缓存。
func TestPhotographerHandler_UpdateProfile_FailureDoesNotInvalidate(t *testing.T) {
	inv := &fakeInvalidator{}
	handler := NewPhotographerHandler(makePhotographerService(serviceHandlerDB(1)), nil, nil)
	handler.invalidator = inv

	body := service.ProfileUpdate{Name: "光影行者", Mode: "paid"}
	w, ctx := newJSONGinContext(http.MethodPut, "/api/v1/photographers/profile", "7", body, nil)
	handler.UpdateProfile(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d: %s", w.Code, w.Body.String())
	}
	if len(inv.prefixes) != 0 {
		t.Errorf("expected no invalidation on rejected update, got %v", inv.prefixes)
	}
}

// Redis 不可用时 NewRedisCache 返回 nil，清理必须安全空转（不能 panic，也不能
// 把写请求拖成 500）。
func TestPhotographerHandler_NilInvalidatorIsSafe(t *testing.T) {
	handler := NewPhotographerHandler(makePhotographerService(activateDB(1)), nil, nil)
	if handler.invalidator != nil {
		t.Fatal("nil cache must leave invalidator nil")
	}

	body := map[string]string{"name": "阿伟", "mode": "both"}
	w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/activate", "7", body, nil)
	handler.Activate(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 with nil cache, got %d: %s", w.Code, w.Body.String())
	}
}
