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

// fakeInvalidator records the prefixes handed to DeletePrefix.
type fakeInvalidator struct {
	prefixes []string
}

func (f *fakeInvalidator) DeletePrefix(_ context.Context, prefix string) int {
	f.prefixes = append(f.prefixes, prefix)
	return 1
}

// newJSONGinContext builds a gin context carrying a JSON body and an authenticated user_id.
func newJSONGinContext(method, path, userID string, body interface{}, params map[string]string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("user_id", int64(7))
	for k, v := range params {
		ctx.Params = append(ctx.Params, gin.Param{Key: k, Value: v})
	}
	return w, ctx
}

// photographerRowWithTags is the GetPhotographerByUserID scan shape (12 columns).
func photographerRowWithTags() []interface{} {
	return []interface{}{
		int32(5), "测试摄影师", strPtr("avatar.svg"), strPtr("desc"), strPtr("深圳"),
		makeNumeric("4.5"), int32Ptr(12), int32Ptr(20), int64Ptr(7), "both", false, strPtr(""),
	}
}

// serviceHandlerDB fakes the two/three statements the write paths issue.
func serviceHandlerDB(execRows int64) *scriptableDBTX {
	return &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			switch {
			case strings.Contains(sql, "FROM photographers"):
				return &successRow{scanFn: func(dest ...interface{}) error {
					return scanInto(dest, photographerRowWithTags())
				}}
			case strings.Contains(sql, "INSERT INTO services"):
				return &successRow{scanFn: func(dest ...interface{}) error {
					return scanInto(dest, []interface{}{int64(42)})
				}}
			default:
				return &errorRow{err: pgx.ErrNoRows}
			}
		},
		execFn: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE " + strconv.FormatInt(execRows, 10)), nil
		},
	}
}

// scanInto reuses valueRows' type-directed assignment for a single row.
func scanInto(dest []interface{}, row []interface{}) error {
	r := &valueRows{rows: [][]interface{}{row}}
	r.Next()
	return r.Scan(dest...)
}

func newServiceHandlerWithFake(t *testing.T) (*PhotographerServiceHandler, *fakeInvalidator) {
	t.Helper()
	inv := &fakeInvalidator{}
	svc := makePhotographerService(serviceHandlerDB(1))
	return NewPhotographerServiceHandler(svc, inv), inv
}

// 套餐是列表卡片价格的数据源：任何一次成功的增删改都必须让列表缓存失效，
// 否则摄影师改完价，C 端最长 5 分钟仍显示旧价。
func TestServiceHandler_WritesInvalidateListCache(t *testing.T) {
	req := service.ServiceUpsertRequest{Name: "基础套餐", Duration: 120}

	t.Run("Create", func(t *testing.T) {
		h, inv := newServiceHandlerWithFake(t)
		w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/services", "7", req, nil)
		h.Create(ctx)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		assertInvalidated(t, inv)
	})

	t.Run("Update", func(t *testing.T) {
		h, inv := newServiceHandlerWithFake(t)
		w, ctx := newJSONGinContext(http.MethodPut, "/api/v1/photographers/services/9", "7", req, map[string]string{"id": "9"})
		h.Update(ctx)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		assertInvalidated(t, inv)
	})

	t.Run("Delete", func(t *testing.T) {
		h, inv := newServiceHandlerWithFake(t)
		w, ctx := newJSONGinContext(http.MethodDelete, "/api/v1/photographers/services/9", "7", nil, map[string]string{"id": "9"})
		h.Delete(ctx)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		assertInvalidated(t, inv)
	})
}

// 失败的写入没有改到任何价格，不能顺手清缓存。
func TestServiceHandler_FailedWriteDoesNotInvalidate(t *testing.T) {
	h, inv := newServiceHandlerWithFake(t)
	bad := service.ServiceUpsertRequest{Name: "负价套餐", Duration: 60, Price: int32Ptr(-1)}

	w, ctx := newJSONGinContext(http.MethodPost, "/api/v1/photographers/services", "7", bad, nil)
	h.Create(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative price, got %d: %s", w.Code, w.Body.String())
	}
	if len(inv.prefixes) != 0 {
		t.Errorf("expected no cache invalidation on rejected write, got %v", inv.prefixes)
	}
}

func assertInvalidated(t *testing.T, inv *fakeInvalidator) {
	t.Helper()
	if len(inv.prefixes) != 1 {
		t.Fatalf("expected exactly 1 invalidation, got %v", inv.prefixes)
	}
	if inv.prefixes[0] != "photographers:" {
		t.Errorf("expected prefix 'photographers:', got %q", inv.prefixes[0])
	}
}
