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

// GetOrderEvents returns the escrow timeline of a booking. Only the two
// parties of the booking may read it.
func (s *BookingService) GetOrderEvents(ctx context.Context, bookingID, actorUserID int64) ([]repository.OrderEvent, error) {
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
	events, err := s.queries.GetOrderEventsByBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []repository.OrderEvent{}
	}
	return events, nil
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
