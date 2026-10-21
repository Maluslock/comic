package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// --- Mock DBTX infrastructure ---

// scriptableDBTX implements repository.DBTX with per-query configurability.
// Each method delegates to a function field; if nil, returns a default error.
type scriptableDBTX struct {
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *scriptableDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, pgx.ErrNoRows
}

func (m *scriptableDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return nil, pgx.ErrNoRows
}

func (m *scriptableDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &errorRow{err: pgx.ErrNoRows}
}

// errorRow implements pgx.Row and always returns the given error.
type errorRow struct {
	err error
}

func (r *errorRow) Scan(dest ...interface{}) error {
	return r.err
}

// successRow implements pgx.Row with a custom scan function.
type successRow struct {
	scanFn func(dest ...interface{}) error
}

func (r *successRow) Scan(dest ...interface{}) error {
	return r.scanFn(dest...)
}

// valueRows implements pgx.Rows backed by a slice of row data.
type valueRows struct {
	rows    [][]interface{}
	current int
}

func (r *valueRows) Close()                                          {}
func (r *valueRows) Err() error                                      { return nil }
func (r *valueRows) CommandTag() pgconn.CommandTag                    { return pgconn.CommandTag{} }
func (r *valueRows) FieldDescriptions() []pgconn.FieldDescription     { return nil }
func (r *valueRows) RawValues() [][]byte                              { return nil }
func (r *valueRows) Conn() *pgx.Conn                                  { return nil }
func (r *valueRows) Next() bool {
	if r.current >= len(r.rows) {
		return false
	}
	r.current++
	return true
}
func (r *valueRows) Scan(dest ...interface{}) error {
	row := r.rows[r.current-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		v := row[i]
		// Type-based copy: dest is always a pointer.
		switch dp := d.(type) {
		case *int64:
			*dp = v.(int64)
		case *int32:
			*dp = v.(int32)
		case *string:
			*dp = v.(string)
		case **string:
			*dp = v.(*string)
		case *[]string:
			*dp = v.([]string)
		case **int32:
			*dp = v.(*int32)
		case *bool:
			*dp = v.(bool)
		case **bool:
			*dp = v.(*bool)
		case *time.Time:
			*dp = v.(time.Time)
		case *pgtype.Numeric:
			*dp = v.(pgtype.Numeric)
		case *[]byte:
			*dp = v.([]byte)
		default:
			// Skip unknown types
		}
	}
	return nil
}
func (r *valueRows) Values() ([]interface{}, error) {
	if r.current == 0 || r.current > len(r.rows) {
		return nil, nil
	}
	return r.rows[r.current-1], nil
}

// --- Test helpers ---

func newGinContext(method, path string, params map[string]string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	ctx.Request = req
	for k, v := range params {
		ctx.Params = append(ctx.Params, gin.Param{Key: k, Value: v})
	}
	return w, ctx
}

// makeEventService creates an EventService backed by the given DBTX.
func makeEventService(db repository.DBTX) *service.EventService {
	return service.NewEventService(repository.New(db))
}

// --- Test data ---

var sampleStartDate = time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
var sampleEndDate = time.Date(2026, 8, 16, 17, 0, 0, 0, time.UTC)

var sampleEvent = repository.ComicEvent{
	ID:           1,
	AllcppID:     1001,
	Name:         "CP30",
	Location:     strPtr("Shanghai"),
	Venue:        strPtr("National Exhibition Center"),
	Address:      strPtr("上海市 青浦区 崧泽大道333号"),
	StartDate:    sampleStartDate,
	EndDate:      sampleEndDate,
	CoverUrl:     strPtr("https://example.com/cover.jpg"),
	Tags:         []string{"cosplay", "anime"},
	ImageGallery: []string{"https://picsum.photos/400/300?1", "https://picsum.photos/400/300?2"},
	Status:       "upcoming",
	TypeName:     strPtr("大型综合漫展"),
}

var samplePhotographerWithTags = repository.PhotographerWithTags{
	ID:          10,
	Name:        "Test Photographer",
	Avatar:      strPtr("https://api.dicebear.com/7.x/avataaars/svg?seed=test"),
	Description: strPtr("A test photographer"),
	Location:    strPtr("Shanghai"),
	Rating:      makeNumeric("4.5"),
	ReviewCount: int32Ptr(100),
	OrderCount:  int32Ptr(200),
	Tags:        []string{"portrait", "cosplay"},
}

var sampleFeaturedWork = repository.FeaturedWork{
	ID:               1,
	PhotographerID:   10,
	Title:            "Test Work",
	Images:           []string{"https://picsum.photos/400/300"},
	Description:      strPtr("A test work"),
	CreatedAt:        sampleStartDate,
	PhotographerName: "Test Photographer",
}

func strPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func makeNumeric(s string) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(s)
	return n
}

// ============================================================================
// Event Handler Tests
// ============================================================================

func TestEventHandler_Detail_Success(t *testing.T) {
	db := &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			if len(args) > 0 {
				if id, ok := args[0].(int64); ok && id == 1 {
					return &successRow{
						scanFn: func(dest ...interface{}) error {
							*dest[0].(*int64) = sampleEvent.ID
							*dest[1].(*int32) = sampleEvent.AllcppID
							*dest[2].(*string) = sampleEvent.Name
							*dest[3].(**string) = sampleEvent.Location
							*dest[4].(**string) = sampleEvent.Venue
							*dest[5].(**string) = sampleEvent.Address
							*dest[6].(*time.Time) = sampleEvent.StartDate
							*dest[7].(*time.Time) = sampleEvent.EndDate
							*dest[8].(**string) = sampleEvent.CoverUrl
							*dest[9].(*[]string) = sampleEvent.Tags
							*dest[10].(*[]string) = sampleEvent.ImageGallery
							*dest[11].(**string) = sampleEvent.TypeName
							*dest[12].(*string) = sampleEvent.Status
							return nil
						},
					}
				}
			}
			return &errorRow{err: pgx.ErrNoRows}
		},
		// Query returns error (silently ignored; photographers/works become empty)
	}
	svc := makeEventService(db)
	handler := NewEventHandler(svc, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/events/1", map[string]string{"id": "1"})
	handler.Detail(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.EventDetail
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected event id 1, got %d", resp.ID)
	}
	if resp.Name != "CP30" {
		t.Errorf("expected name CP30, got %s", resp.Name)
	}
	if resp.Location != "Shanghai" {
		t.Errorf("expected location Shanghai, got %s", resp.Location)
	}
	if resp.Status != "upcoming" {
		t.Errorf("expected status upcoming, got %s", resp.Status)
	}
	if len(resp.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(resp.Tags))
	}
}

func TestEventHandler_Detail_NotFound(t *testing.T) {
	db := &scriptableDBTX{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &errorRow{err: pgx.ErrNoRows}
		},
	}
	svc := makeEventService(db)
	handler := NewEventHandler(svc, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/events/999", map[string]string{"id": "999"})
	handler.Detail(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	errMsg, ok := body["error"].(string)
	if !ok || errMsg != "event not found" {
		t.Errorf("expected 'event not found' error, got '%v'", body["error"])
	}
}

func TestEventHandler_Detail_InvalidID(t *testing.T) {
	// No DB needed — handler returns 400 before calling service
	handler := NewEventHandler(nil, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/events/abc", map[string]string{"id": "abc"})
	handler.Detail(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	errMsg, ok := body["error"].(string)
	if !ok || errMsg != "invalid event id" {
		t.Errorf("expected 'invalid event id' error, got '%v'", body["error"])
	}
}

func TestEventHandler_List_Success(t *testing.T) {
	event2 := sampleEvent
	event2.ID = 2
	event2.Name = "CD28"

	db := &scriptableDBTX{
		queryFn: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &valueRows{
				rows: [][]interface{}{
					{
						int64(1), int32(1001), "CP30", strPtr("Shanghai"), strPtr("NEC"), strPtr("上海市 青浦区"),
						sampleStartDate, sampleEndDate, strPtr("https://example.com/cover.jpg"),
						[]string{"cosplay", "anime"}, []string{"https://picsum.photos/400/300?1"},
						strPtr("大型综合漫展"), "upcoming",
					},
					{
						int64(2), int32(1002), "CD28", strPtr("Chengdu"), strPtr("Chengdu Expo"), strPtr("成都市 高新区"),
						sampleStartDate, sampleEndDate, strPtr("https://example.com/cover2.jpg"),
						[]string{"doujin"}, []string{"https://picsum.photos/400/300?2"},
						strPtr("同人祭"), "upcoming",
					},
				},
			}, nil
		},
	}
	svc := makeEventService(db)
	handler := NewEventHandler(svc, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/events", nil)
	handler.List(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.EventListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(resp.List) == 0 {
		t.Error("expected non-empty event list")
	}
	if len(resp.List) != 2 {
		t.Errorf("expected 2 events, got %d", len(resp.List))
	}
	if resp.List[0].Name != "CP30" {
		t.Errorf("expected first event CP30, got %s", resp.List[0].Name)
	}
	if resp.List[1].Name != "CD28" {
		t.Errorf("expected second event CD28, got %s", resp.List[1].Name)
	}
}

func TestEventHandler_List_Empty(t *testing.T) {
	db := &scriptableDBTX{
		queryFn: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &valueRows{rows: [][]interface{}{}}, nil
		},
	}
	svc := makeEventService(db)
	handler := NewEventHandler(svc, nil)

	w, ctx := newGinContext(http.MethodGet, "/api/v1/events", nil)
	handler.List(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp service.EventListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(resp.List) != 0 {
		t.Errorf("expected empty event list, got %d items", len(resp.List))
	}
}

func TestNewEventHandler(t *testing.T) {
	svc := service.NewEventService(nil)
	h := NewEventHandler(svc, nil)
	if h == nil {
		t.Error("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("expected handler to hold the service reference")
	}
}
