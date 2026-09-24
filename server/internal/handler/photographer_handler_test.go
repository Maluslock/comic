package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// makePhotographerService creates a PhotographerService backed by the given DBTX.
func makePhotographerService(db repository.DBTX) *service.PhotographerService {
	return service.NewPhotographerService(repository.New(db))
}

// ============================================================================
// Photographer Handler Tests
// ============================================================================

func TestPhotographerHandler_Detail_Success(t *testing.T) {
	db := &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &successRow{
				scanFn: func(dest ...interface{}) error {
					// Field order: ID, Name, Avatar, Description, Location, Rating, ReviewCount, OrderCount, UserID, Mode, Certified, MutualIntro, Tags
					*dest[0].(*int32) = 10
					*dest[1].(*string) = "Test Photographer"
					*dest[2].(**string) = strPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=test")
					*dest[3].(**string) = strPtr("A test photographer")
					*dest[4].(**string) = strPtr("Shanghai")
					*dest[5].(*pgtype.Numeric) = makeNumeric("4.5")
					*dest[6].(**int32) = int32Ptr(100)
					*dest[7].(**int32) = int32Ptr(200)
					*dest[8].(**int64) = int64Ptr(1001)
					*dest[9].(*string) = "both"
					*dest[10].(*bool) = true
					*dest[11].(**string) = strPtr("互勉互拍")
					*dest[12].(*[]string) = []string{"portrait", "cosplay"}
					return nil
				},
			}
		},
	}
	svc := makePhotographerService(db)
	handler := NewPhotographerHandler(svc, nil, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/photographers/10", map[string]string{"id": "10"})
	handler.Detail(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.PhotographerDetail
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if resp.Name != "Test Photographer" {
		t.Errorf("expected name 'Test Photographer', got '%s'", resp.Name)
	}
	if resp.Location != "Shanghai" {
		t.Errorf("expected location 'Shanghai', got '%s'", resp.Location)
	}
	if resp.Rating != 4.5 {
		t.Errorf("expected rating 4.5, got %f", resp.Rating)
	}
	if len(resp.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(resp.Tags))
	}
	if resp.ReviewCount != 100 {
		t.Errorf("expected reviewCount 100, got %d", resp.ReviewCount)
	}
	if resp.MutualIntro != "互勉互拍" {
		t.Errorf("expected mutualIntro '互勉互拍', got '%s'", resp.MutualIntro)
	}
}

func TestPhotographerHandler_Detail_NotFound(t *testing.T) {
	db := &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &errorRow{err: pgx.ErrNoRows}
		},
	}
	svc := makePhotographerService(db)
	handler := NewPhotographerHandler(svc, nil, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/photographers/999", map[string]string{"id": "999"})
	handler.Detail(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	errMsg, ok := body["error"].(string)
	if !ok || errMsg != "photographer not found" {
		t.Errorf("expected 'photographer not found' error, got '%v'", body["error"])
	}
}

func TestPhotographerHandler_Detail_InvalidID(t *testing.T) {
	// No DB needed — handler returns 400 before calling service
	handler := NewPhotographerHandler(nil, nil, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/photographers/abc", map[string]string{"id": "abc"})
	handler.Detail(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	errMsg, ok := body["error"].(string)
	if !ok || errMsg != "invalid photographer id" {
		t.Errorf("expected 'invalid photographer id' error, got '%v'", body["error"])
	}
}

func TestPhotographerHandler_List_Success(t *testing.T) {
	db := &scriptableDBTX{
		queryFn: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			// List() now issues a second batch query for the price roll-up; the fake
			// must answer each query with its own shape, not one canned result.
			if strings.Contains(sql, "bool_or") {
				return &valueRows{
					rows: [][]interface{}{
						// 摄影师 10：有固定价 399 + 面议 → 「¥399 起」
						{int64(10), int32Ptr(399), false, true, int32(2)},
						// 摄影师 20：只有 0 元套餐 → 「互勉」
						{int64(20), (*int32)(nil), true, false, int32(1)},
					},
				}, nil
			}
			return &valueRows{
				rows: [][]interface{}{
					{
						int32(10), "Photographer A", strPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=a"),
						strPtr("Desc A"), strPtr("Shanghai"), makeNumeric("4.8"),
						int32Ptr(150), int32Ptr(300), []string{"portrait", "cosplay"},
					},
					{
						int32(20), "Photographer B", strPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=b"),
						strPtr("Desc B"), strPtr("Beijing"), makeNumeric("4.2"),
						int32Ptr(80), int32Ptr(120), []string{"landscape"},
					},
				},
			}, nil
		},
	}
	svc := makePhotographerService(db)
	handler := NewPhotographerHandler(svc, nil, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/photographers", nil)
	handler.List(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.PhotographerListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(resp.List) == 0 {
		t.Error("expected non-empty photographer list")
	}
	if len(resp.List) != 2 {
		t.Errorf("expected 2 photographers, got %d", len(resp.List))
	}
	if resp.List[0].Name != "Photographer A" {
		t.Errorf("expected first photographer 'Photographer A', got '%s'", resp.List[0].Name)
	}
	if resp.List[1].Name != "Photographer B" {
		t.Errorf("expected second photographer 'Photographer B', got '%s'", resp.List[1].Name)
	}
	// 价格汇总必须真的从第二个查询落到响应里（只断言 Name 会漏掉整条装配断链）
	if resp.List[0].MinPrice == nil || *resp.List[0].MinPrice != 399 {
		t.Errorf("expected photographer A minPrice 399, got %v", resp.List[0].MinPrice)
	}
	if !resp.List[0].HasNegotiable || resp.List[0].HasFree || resp.List[0].ServiceCount != 2 {
		t.Errorf("unexpected A roll-up: %+v", resp.List[0])
	}
	if resp.List[1].MinPrice != nil || !resp.List[1].HasFree || resp.List[1].ServiceCount != 1 {
		t.Errorf("unexpected B roll-up: %+v", resp.List[1])
	}
}

func TestNewPhotographerHandler(t *testing.T) {
	svc := service.NewPhotographerService(nil)
	h := NewPhotographerHandler(svc, nil, nil)
	if h == nil {
		t.Error("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("expected handler to hold the service reference")
	}
}
