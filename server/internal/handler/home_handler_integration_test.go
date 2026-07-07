//go:build integration

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// TestHomeHandlerIntegration is an integration test that requires a real PostgreSQL.
// Skip if no DATABASE_URL is set and localhost connection fails.
//
// Run with:
//
//	DATABASE_URL=postgres://comic:comic123@localhost:5432/comic?sslmode=disable \
//	  go test -tags=integration ./internal/handler/ -run Integration -v
func TestHomeHandlerIntegration(t *testing.T) {
	pool := getTestDB(t)
	defer pool.Close()

	// Verify seed data exists
	var count int
	err := pool.QueryRow(context.Background(), "SELECT count(*) FROM tags").Scan(&count)
	if err != nil || count == 0 {
		t.Skipf("skipping integration test: no seed data (tags count=%d, err=%v)", count, err)
	}

	// Wire dependencies
	queries := repository.New(pool)
	homeRepo := repository.NewHomeRepository(queries)
	homeSvc := service.NewHomeService(homeRepo)
	handler := NewHomeHandler(homeSvc)

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)

	handler.GetHome(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.HomeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// --- Verify response structure ---

	t.Run("banners", func(t *testing.T) {
		if len(resp.Banners) == 0 {
			t.Error("expected at least 1 banner")
			return
		}
		b := resp.Banners[0]
		if b.ID == 0 {
			t.Error("banner ID should not be zero")
		}
		if b.ImageURL == "" {
			t.Error("banner ImageURL should not be empty")
		}
	})

	t.Run("hotTags", func(t *testing.T) {
		if len(resp.HotTags) == 0 {
			t.Error("expected at least 1 tag")
			return
		}
		tg := resp.HotTags[0]
		if tg.Name == "" {
			t.Error("tag name should not be empty")
		}
	})

	t.Run("recommendedPhotographers", func(t *testing.T) {
		if len(resp.RecommendedPhotographers) == 0 {
			t.Error("expected at least 1 photographer")
			return
		}
		p := resp.RecommendedPhotographers[0]
		if p.Name == "" {
			t.Error("photographer name should not be empty")
		}
		if p.Rating < 0 || p.Rating > 5 {
			t.Errorf("invalid rating: %f", p.Rating)
		}
	})

	t.Run("featuredWorks", func(t *testing.T) {
		if len(resp.FeaturedWorks) == 0 {
			t.Error("expected at least 1 work")
			return
		}
		w := resp.FeaturedWorks[0]
		if w.Title == "" {
			t.Error("work title should not be empty")
		}
		if len(w.Images) == 0 {
			t.Error("work should have at least 1 image")
		}
	})

	// upcomingEvents may be empty (no allcpp sync yet) — that's OK for this test
	t.Logf("upcomingEvents count: %d", len(resp.UpcomingEvents))
}

// getTestDB connects to the test PostgreSQL or skips the test.
// Set DATABASE_URL to override the default localhost connection.
func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://comic:comic123@localhost:5432/comic?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to DB: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: cannot ping DB: %v", err)
	}

	return pool
}
