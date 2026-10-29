package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

// ------------------------------------------------------------------
// BookingService: dispute helpers that the C-end and the admin console
// share. The state machine lives here only; handlers never re-check
// status transitions themselves.
// ------------------------------------------------------------------

// ActorRoleForUser resolves the acting role of a user from the database
// instead of trusting a client supplied header: a user who owns a
// photographer profile acts as photographer, everybody else acts as coser.
func (s *BookingService) ActorRoleForUser(ctx context.Context, userID int64) (string, error) {
	if _, err := s.queries.GetPhotographerByUserID(ctx, userID); err == nil {
		return actorPhotographer, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return actorCoser, nil
}

// BookingDisputeDTO is the inner dispute object returned to C-end.
type BookingDisputeDTO struct {
	ID               int64    `json:"id"`
	BookingID        int64    `json:"bookingId"`
	OpenedBy         int64    `json:"openedBy"`
	OpenedByRole     string   `json:"openedByRole"`
	Type             string   `json:"type"`
	Reason           string   `json:"reason"`
	Status           string   `json:"status"`
	RespondDeadline  string   `json:"respondDeadline"`
	EvidenceDeadline string   `json:"evidenceDeadline"`
	Resolution       *string  `json:"resolution"`
	AdminID          *int64   `json:"adminId"`
	ReviewedAt       *string  `json:"reviewedAt"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

// BookingDisputeResponse wraps the dispute DTO and the caller's role.
type BookingDisputeResponse struct {
	Dispute *BookingDisputeDTO `json:"dispute"`
	MyRole  string             `json:"myRole"`
}

// OrderEventItem is the C-end DTO for booking timeline events.
type OrderEventItem struct {
	ID        int64          `json:"id"`
	EventType string         `json:"eventType"`
	ActorRole *string        `json:"actorRole"`
	ActorID   *int64         `json:"actorId"`
	CreatedAt string         `json:"createdAt"`
	Payload   map[string]any `json:"payload"`
}

// GetOrderEvents returns the escrow timeline of a booking. Only the two
// parties of the booking may read it. Payload is returned as a parsed JSON
// object under payloadObj.
func (s *BookingService) GetOrderEvents(ctx context.Context, bookingID, actorUserID int64) ([]OrderEventItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if int64(b.CoserID) != actorUserID {
		profile, perr := s.queries.GetPhotographerByUserID(ctx, actorUserID)
		if perr != nil || int64(profile.ID) != int64(b.PhotographerID) {
			return nil, ErrForbidden
		}
	}
	rawEvents, err := s.queries.GetOrderEventsByBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if rawEvents == nil {
		rawEvents = []repository.OrderEvent{}
	}
	items := make([]OrderEventItem, 0, len(rawEvents))
	for _, e := range rawEvents {
		payload := map[string]any{}
		if len(e.Payload) > 0 {
			if err := json.Unmarshal(e.Payload, &payload); err != nil {
				payload = map[string]any{"_raw": string(e.Payload)}
			}
		}
		items = append(items, OrderEventItem{
			ID:        e.ID,
			EventType: e.EventType,
			ActorRole: e.ActorRole,
			ActorID:   e.ActorID,
			CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Payload:   payload,
		})
	}
	return items, nil
}

// RecordDisputeEvent appends one entry to the dispute timeline.
func (s *BookingService) RecordDisputeEvent(ctx context.Context, disputeID, actorUserID int64, actorRole, eventType string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte("{}")
	}
	var actor *int64
	if actorUserID != 0 {
		actor = &actorUserID
	}
	var role *string
	if actorRole != "" {
		role = &actorRole
	}
	if _, err := s.queries.CreateDisputeEvent(ctx, repository.CreateDisputeEventParams{
		DisputeID: disputeID,
		ActorID:   actor,
		ActorRole: role,
		EventType: eventType,
		Payload:   raw,
	}); err != nil {
		log.Printf("dispute event write failed (dispute=%d type=%s): %v", disputeID, eventType, err)
	}
}

// GetBookingDispute returns the C-end view of the most recent dispute on a
// booking, plus the caller's role relative to the booking. Only the two
// parties of the booking may read it.
func (s *BookingService) GetBookingDispute(ctx context.Context, bookingID, actorUserID int64) (*BookingDisputeResponse, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	var myRole string
	if int64(b.CoserID) == actorUserID {
		myRole = actorCoser
	} else {
		profile, perr := s.queries.GetPhotographerByUserID(ctx, actorUserID)
		if perr != nil || int64(profile.ID) != int64(b.PhotographerID) {
			return nil, ErrForbidden
		}
		myRole = actorPhotographer
	}
	resp := &BookingDisputeResponse{MyRole: myRole}
	d, err := s.queries.GetDisputeByBooking(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return resp, nil
		}
		return nil, err
	}
	dto := &BookingDisputeDTO{
		ID:               d.ID,
		BookingID:        d.BookingID,
		OpenedBy:         d.OpenedBy,
		OpenedByRole:     d.OpenedByRole,
		Type:             d.Type,
		Reason:           d.Reason,
		Status:           d.Status,
		RespondDeadline:  d.RespondDeadline.Format("2006-01-02T15:04:05Z07:00"),
		EvidenceDeadline: d.EvidenceDeadline.Format("2006-01-02T15:04:05Z07:00"),
		Resolution:       d.Resolution,
		AdminID:          d.AdminID,
		CreatedAt:        d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if d.ReviewedAt != nil {
		s := d.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		dto.ReviewedAt = &s
	}
	resp.Dispute = dto
	return resp, nil
}

// MarkCoserNoShow lets the photographer mark the coser as a no-show for a
// confirmed booking. The booking must not already have a no-show party. The
// action is idempotent: repeated calls return the already-marked booking
// without applying a second credit penalty.
func (s *BookingService) MarkCoserNoShow(ctx context.Context, bookingID int64, actorUserID int64) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	profile, err := s.queries.GetPhotographerByUserID(ctx, actorUserID)
	if err != nil || int64(profile.ID) != int64(b.PhotographerID) {
		return nil, ErrForbidden
	}
	if b.Status != "confirmed" {
		return nil, ErrInvalidTransition
	}
	if b.NoShowParty != nil {
		if *b.NoShowParty == actorCoser {
			return bookingToItem(b), nil
		}
		return nil, ErrInvalidTransition
	}
	updated, err := s.queries.MarkBookingNoShow(ctx, bookingID, actorCoser)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return nil, err
	}
	s.emitOrderEvent(ctx, bookingID, actorUserID, actorPhotographer, eventNoShow, map[string]any{"party": actorCoser})
	// Photographer-initiated coser no-show: coser -15, idempotent via unique index.
	s.applyCreditDelta(ctx, int64(b.CoserID), -15, creditReasonCoserNoShow, strPtr("booking"), &bookingID, &bookingID, "模特缺席")
	s.notify(ctx, b.CoserID, "warning", "您被标记缺席", fmt.Sprintf("预约(#%d)摄影师标记您缺席，信用分将受到影响", bookingID), bookingID)
	s.notifyPhotographer(ctx, b.PhotographerID, "info", "缺席标记已提交", fmt.Sprintf("预约(#%d)已标记模特缺席", bookingID), bookingID)
	return bookingToItem(updated), nil
}

// EscalateDispute moves a dispute into the escalated state. Both parties may
// escalate an open/responding dispute; the admin console may escalate too
// (actorRole = admin). The booking stays in the disputed sub-status.
func (s *BookingService) EscalateDispute(ctx context.Context, disputeID, actorUserID int64, actorRole, reason string) (*repository.Dispute, error) {
	d, err := s.queries.GetDisputeByID(ctx, disputeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	switch d.Status {
	case disputeStatusOpen, disputeStatusResponding:
	case disputeStatusEscalated:
		// Already escalated — idempotent.
		return &d, nil
	default:
		return nil, ErrInvalidTransition
	}

	if actorRole != actorAdmin {
		b, err := s.queries.GetBookingByID(ctx, d.BookingID)
		if err != nil {
			return nil, err
		}
		if actorRole == actorCoser {
			if int64(b.CoserID) != actorUserID {
				return nil, ErrForbidden
			}
		} else if actorRole == actorPhotographer {
			profile, err := s.queries.GetPhotographerByUserID(ctx, actorUserID)
			if err != nil || int64(profile.ID) != int64(b.PhotographerID) {
				return nil, ErrForbidden
			}
		} else {
			return nil, ErrForbidden
		}
	}

	updated, err := s.queries.UpdateDisputeStatus(ctx, repository.UpdateDisputeStatusParams{
		ID:     disputeID,
		Status: disputeStatusEscalated,
	})
	if err != nil {
		return nil, err
	}
	s.RecordDisputeEvent(ctx, disputeID, actorUserID, actorRole, "escalate", map[string]any{
		"reason": reason,
	})
	s.emitOrderEvent(ctx, d.BookingID, actorUserID, actorRole, eventDisputeEscalate, map[string]any{
		"dispute_id": disputeID,
		"reason":     reason,
	})
	b, err := s.queries.GetBookingByID(ctx, d.BookingID)
	if err == nil {
		s.notify(ctx, b.CoserID, "warning", "争议已升级", fmt.Sprintf("预约(#%d)的争议已升级，等待平台介入", d.BookingID), d.BookingID)
		s.notifyPhotographer(ctx, b.PhotographerID, "warning", "争议已升级", fmt.Sprintf("预约(#%d)的争议已升级，等待平台介入", d.BookingID), d.BookingID)
	}
	return &updated, nil
}

// ------------------------------------------------------------------
// Admin dispute console
// ------------------------------------------------------------------

// adminDisputeStore is the query side of the admin dispute console.
type adminDisputeStore interface {
	ListDisputes(ctx context.Context, status string, limit, offset int32) ([]repository.AdminDispute, int64, error)
	GetDisputeDetail(ctx context.Context, id int64) (repository.AdminDisputeDetail, error)
}

// AdminDisputeService backs GET /admin/v1/disputes, GET /admin/v1/disputes/:id
// and PUT /admin/v1/disputes/:id/resolve.
type AdminDisputeService struct {
	store      adminDisputeStore
	bookingSvc *BookingService
}

func NewAdminDisputeService(repo *repository.AdminDisputeRepo, bookingSvc *BookingService) *AdminDisputeService {
	return &AdminDisputeService{store: repo, bookingSvc: bookingSvc}
}

// Dispute resolve actions accepted by the admin console.
const (
	DisputeActionApprove  = "approve" // 支持申请方
	DisputeActionReject   = "reject"  // 驳回申请
	DisputeActionEscalate = "escalate"
)

// 复用 cert_application_service.go 里的 ErrInvalidAction / ErrReasonRequired：
// 同一个 package 不能重复声明，且管理端两类审批本来就该返回一致的错误语义。

// ListDisputes returns the paginated admin dispute list.
func (s *AdminDisputeService) ListDisputes(ctx context.Context, status string, page, pageSize int) ([]repository.AdminDispute, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListDisputes(ctx, status, int32(pageSize), int32((page-1)*pageSize))
}

// GetDisputeDetail returns one dispute with its timeline.
func (s *AdminDisputeService) GetDisputeDetail(ctx context.Context, id int64) (repository.AdminDisputeDetail, error) {
	return s.store.GetDisputeDetail(ctx, id)
}

// Resolve applies an admin decision to a dispute.
//
//	approve  → 申请方胜诉，被申请方违约（信用 -10）
//	reject   → 驳回争议申请，申请方违约（信用 -10）
//	escalate → 争议升级，等待平台进一步介入（不扣信用）
//
// The actual write (dispute row, order_events, notifications, credit delta)
// is delegated to BookingService so there is exactly one implementation of
// the dispute state machine.
func (s *AdminDisputeService) Resolve(ctx context.Context, id, adminUserID int64, action, reason string) (*repository.Dispute, error) {
	switch action {
	case DisputeActionApprove, DisputeActionReject:
		if action == DisputeActionReject && reason == "" {
			return nil, ErrReasonRequired
		}
	case DisputeActionEscalate:
		return s.bookingSvc.EscalateDispute(ctx, id, adminUserID, actorAdmin, reason)
	default:
		return nil, ErrInvalidAction
	}

	d, err := s.bookingSvc.queries.GetDisputeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}

	// The loser of the dispute is the party the ruling goes against.
	loser := d.OpenedByRole
	resolution := "裁决通过：申请方胜诉"
	if action == DisputeActionReject {
		loser = d.OpenedByRole
		resolution = "驳回争议申请：被申请方胜诉"
	} else {
		if d.OpenedByRole == actorCoser {
			loser = actorPhotographer
		} else {
			loser = actorCoser
		}
	}
	if reason != "" {
		resolution = resolution + "。裁决说明：" + reason
	}

	resolved, err := s.bookingSvc.ResolveDispute(ctx, id, adminUserID, resolution, &loser)
	if err != nil {
		return nil, err
	}
	s.bookingSvc.RecordDisputeEvent(ctx, id, adminUserID, actorAdmin, "resolve", map[string]any{
		"action":     action,
		"resolution": resolution,
		"reason":     reason,
		"loser":      loser,
	})
	return resolved, nil
}
