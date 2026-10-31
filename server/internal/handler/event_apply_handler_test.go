package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// stubEventApplySvc 是 handler 测试用的最小 store，行为由各用例注入。
// 轮询/分页等 SQL 细节由 service 单测覆盖，这里只验证 HTTP 契约。
type stubEventApplySvc struct {
	applyErr    error
	reviewErr   error
	mineErr     error
	withdrawErr error

	lastStatus string
	lastReason string
	lastAdmin  int64
}

func newEventApplyHandlerForTest(s *stubEventApplySvc) *EventApplyHandler {
	return NewEventApplyHandler(service.NewEventApplyService(s, s, nil))
}

func newAdminEventApplyHandlerForTest(s *stubEventApplySvc) *AdminEventApplyHandler {
	return NewAdminEventApplyHandler(service.NewEventApplyService(s, s, nil))
}

// --- eventApplyStore implementation ---

func (s *stubEventApplySvc) ApplyEventPhotographer(_ context.Context, _, _, _ int64, _ *string) (repository.EventApplication, error) {
	if s.applyErr != nil {
		return repository.EventApplication{}, s.applyErr
	}
	return repository.EventApplication{ID: 1, EventID: 12, PhotographerID: 1, UserID: 1, Status: "pending"}, nil
}

func (s *stubEventApplySvc) ReviveWithdrawnEventApplication(_ context.Context, _, _ int64, _ *string) (repository.EventApplication, error) {
	return repository.EventApplication{}, pgx.ErrNoRows
}

func (s *stubEventApplySvc) GetMyEventApplication(_ context.Context, _, _ int64) (repository.EventApplication, error) {
	if s.mineErr != nil {
		return repository.EventApplication{}, s.mineErr
	}
	return repository.EventApplication{ID: 1, EventID: 12, Status: "pending"}, nil
}

func (s *stubEventApplySvc) ListMyEventApplications(_ context.Context, _ int64) ([]repository.MyEventApplication, error) {
	return nil, nil
}

func (s *stubEventApplySvc) WithdrawEventApplication(_ context.Context, _, _ int64) (int64, error) {
	if s.withdrawErr != nil {
		return 0, s.withdrawErr
	}
	return 1, nil
}

func (s *stubEventApplySvc) ListApprovedPhotographersByEvent(_ context.Context, _ int64) ([]repository.PhotographerWithTags, error) {
	return nil, nil
}

func (s *stubEventApplySvc) CountApprovedByEvent(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (s *stubEventApplySvc) ListEventApplications(_ context.Context, _ string, _, _ int32) ([]repository.AdminEventApplication, int64, error) {
	return []repository.AdminEventApplication{{EventApplication: repository.EventApplication{ID: 1, Status: "pending"}}}, 1, nil
}

func (s *stubEventApplySvc) CountEventApplications(_ context.Context, _ string) (int64, error) {
	return 1, nil
}

func (s *stubEventApplySvc) ReviewEventApplication(_ context.Context, _ int64, status, reason string, adminID int64) (int64, error) {
	s.lastStatus = status
	s.lastReason = reason
	s.lastAdmin = adminID
	if s.reviewErr != nil {
		return 0, s.reviewErr
	}
	return 1, nil
}

func (s *stubEventApplySvc) GetEventApplicationByID(_ context.Context, _ int64) (repository.EventApplication, error) {
	return repository.EventApplication{ID: 1, UserID: 42, Status: "pending"}, nil
}

func (s *stubEventApplySvc) GetEventById(_ context.Context, _ int64) (repository.ComicEvent, error) {
	return repository.ComicEvent{ID: 12, Name: "赤马EDC", EndDate: time.Now().Add(48 * time.Hour)}, nil
}

func (s *stubEventApplySvc) GetPhotographerByUserID(_ context.Context, _ int64) (repository.PhotographerWithTags, error) {
	return repository.PhotographerWithTags{ID: 1, Name: "光影行者"}, nil
}

func TestEventApplyHandler_Apply_NotPhotographerIs403(t *testing.T) {
	h := newEventApplyHandlerForTest(&stubEventApplySvc{applyErr: service.ErrEventApplyNotPhotographer})
	rec, c := newGinContext("POST", "/api/v1/events/12/photographer-apply", map[string]string{"id": "12"})

	h.Apply(c)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestEventApplyHandler_Apply_DuplicateIs409(t *testing.T) {
	h := newEventApplyHandlerForTest(&stubEventApplySvc{applyErr: service.ErrEventApplyDuplicate})
	rec, c := newGinContext("POST", "/api/v1/events/12/photographer-apply", map[string]string{"id": "12"})

	h.Apply(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already applied") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestEventApplyHandler_Apply_EventNotFoundIs404(t *testing.T) {
	h := newEventApplyHandlerForTest(&stubEventApplySvc{applyErr: service.ErrEventNotFound})
	rec, c := newGinContext("POST", "/api/v1/events/12/photographer-apply", map[string]string{"id": "12"})

	h.Apply(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestEventApplyHandler_Apply_UniqueViolationNeverFiveHundred(t *testing.T) {
	// 23505 必须被 service 翻成 ErrEventApplyDuplicate → 409，绝不能落 500。
	store := &stubEventApplySvc{applyErr: &pgconn.PgError{Code: "23505"}}
	h := newEventApplyHandlerForTest(store)
	rec, c := newGinContext("POST", "/api/v1/events/12/photographer-apply", map[string]string{"id": "12"})

	h.Apply(c)

	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("23505 must not become 500: %s", rec.Body.String())
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestEventApplyHandler_Apply_InvalidEventIDIs400(t *testing.T) {
	h := newEventApplyHandlerForTest(&stubEventApplySvc{})
	rec, c := newGinContext("POST", "/api/v1/events/abc/photographer-apply", map[string]string{"id": "abc"})

	h.Apply(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestEventApplyHandler_MyApplication_NullWhenNeverApplied(t *testing.T) {
	h := newEventApplyHandlerForTest(&stubEventApplySvc{mineErr: pgx.ErrNoRows})
	rec, c := newGinContext("GET", "/api/v1/events/12/photographer-apply/mine", map[string]string{"id": "12"})

	h.MyApplication(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	app, ok := body["application"]
	if !ok {
		t.Fatalf("missing application key: %s", rec.Body.String())
	}
	if app != nil {
		t.Fatalf("application must be null when never applied, got %v", app)
	}
}

func TestEventApplyHandler_AdminReview_ReasonRequiredIs400(t *testing.T) {
	h := newAdminEventApplyHandlerForTest(&stubEventApplySvc{reviewErr: service.ErrEventApplyReasonRequired})
	rec, c := newGinContext("PUT", "/api/admin/v1/event-applications/1/review", map[string]string{"id": "1"})
	c.Request = httptest.NewRequest("PUT", "/api/admin/v1/event-applications/1/review",
		strings.NewReader(`{"status":"rejected"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Review(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestEventApplyHandler_AdminReview_AlreadyReviewedIs409(t *testing.T) {
	h := newAdminEventApplyHandlerForTest(&stubEventApplySvc{reviewErr: service.ErrEventApplyAlreadyReviewed})
	rec, c := newGinContext("PUT", "/api/admin/v1/event-applications/1/review", map[string]string{"id": "1"})
	c.Request = httptest.NewRequest("PUT", "/api/admin/v1/event-applications/1/review",
		strings.NewReader(`{"status":"approved"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Review(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestEventApplyHandler_AdminReview_InvalidIDIs400(t *testing.T) {
	h := newAdminEventApplyHandlerForTest(&stubEventApplySvc{})
	rec, c := newGinContext("PUT", "/api/admin/v1/event-applications/xx/review", map[string]string{"id": "xx"})

	h.Review(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}
