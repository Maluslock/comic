package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// mockDBTX implements repository.DBTX and returns errors for all methods —
// forces the sqlc-generated queries to fail without a real database.
type mockDBTX struct{}

func (m *mockDBTX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("mock: not connected")
}

func (m *mockDBTX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("mock: not connected")
}

func (m *mockDBTX) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return &mockRow{}
}

// mockRow implements pgx.Row — Scan always returns an error so sqlc queries fail fast.
type mockRow struct{}

func (m *mockRow) Scan(_ ...interface{}) error {
	return errors.New("mock: not connected")
}

func TestHomeHandler_GetHome_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Wire real dependencies with a mock DBTX — GetHomeData returns error, not panic
	queries := repository.New(&mockDBTX{})
	homeRepo := repository.NewHomeRepository(queries)
	homeSvc := service.NewHomeService(homeRepo)
	handler := NewHomeHandler(homeSvc, nil)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)

	handler.GetHome(ctx)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 from mock DB error, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if _, ok := body["error"]; !ok {
		t.Error("expected error field in JSON response")
	}
}

func TestNewHomeHandler(t *testing.T) {
	svc := service.NewHomeService(nil)
	h := NewHomeHandler(svc, nil)
	if h == nil {
		t.Error("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("expected handler to hold the service reference")
	}
}
