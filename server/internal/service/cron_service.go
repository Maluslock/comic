package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
)

// CronService executes escrow background tasks.
type CronService struct {
	queries *repository.Queries
}

// NewCronService builds a cron runner from repository queries.
func NewCronService(queries *repository.Queries) *CronService {
	return &CronService{queries: queries}
}

// AutoAcceptDeliveries completes bookings whose acceptance deadline expired.
func (c *CronService) AutoAcceptDeliveries(ctx context.Context) error {
	bookings, err := c.queries.AutoAcceptDelivery(ctx)
	if err != nil {
		return fmt.Errorf("auto_accept delivery query failed: %w", err)
	}
	for _, b := range bookings {
		log.Printf("[cron:auto_accept] booking #%d auto-accepted", b.ID)
		if err := c.emitOrderEventForCron(ctx, b.ID, eventAutoAccept, map[string]any{
			"original_accept_deadline": b.AcceptDeadline,
		}); err != nil {
			log.Printf("[cron:auto_accept] event emit failed for #%d: %v", b.ID, err)
		}
		if err := c.notifyAutoAccept(ctx, b); err != nil {
			log.Printf("[cron:auto_accept] notify failed for #%d: %v", b.ID, err)
		}
		c.recomputePhotographerStats(ctx, b.PhotographerID)
		// Coser 履约信用 +2
		c.applyCreditDelta(ctx, int64(b.CoserID), 2, creditReasonOrderComplete, strPtr("booking"), &b.ID, &b.ID, "订单自动验收完成")
	}
	return nil
}

// AutoNoShow flags confirmed bookings whose slot has passed without check-in.
func (c *CronService) AutoNoShow(ctx context.Context) (int, error) {
	candidates, err := c.queries.ListNoShowCandidates(ctx)
	if err != nil {
		return 0, fmt.Errorf("noshow list query failed: %w", err)
	}
	processed := 0
	for _, b := range candidates {
		// No check-in after slot start → photographer flagged as no-show by default.
		party := actorPhotographer
		if b.CheckinAt != nil {
			// Already checked in but still marked as candidate? Treat as no noshow.
			continue
		}
		updated, err := c.queries.MarkBookingNoShow(ctx, b.ID, party)
		if err != nil {
			log.Printf("[cron:noshow] mark failed for #%d: %v", b.ID, err)
			continue
		}
		processed++
		log.Printf("[cron:noshow] booking #%d marked noshow=%s", updated.ID, party)
		if err := c.emitOrderEventForCron(ctx, b.ID, eventNoShow, map[string]any{"party": party}); err != nil {
			log.Printf("[cron:noshow] event emit failed for #%d: %v", b.ID, err)
		}
		if err := c.notifyNoShow(ctx, b, party); err != nil {
			log.Printf("[cron:noshow] notify failed for #%d: %v", b.ID, err)
		}
		// Photographer no-show -8.
		profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
		if err == nil && profile.UserID != nil {
			c.applyCreditDelta(ctx, *profile.UserID, -8, creditReasonNoShow, strPtr("booking"), &b.ID, &b.ID, "摄影师未到场")
		}
	}
	return processed, nil
}

// AutoCancelPending cancels bookings that have been pending too long.
func (c *CronService) AutoCancelPending(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-30 * time.Minute)
	bookings, err := c.queries.AutoCancelPending(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("auto_cancel pending query failed: %w", err)
	}
	for _, b := range bookings {
		log.Printf("[cron:auto_cancel] booking #%d auto-cancelled (created_at=%s)", b.ID, b.CreatedAt)
		if err := c.emitOrderEventForCron(ctx, b.ID, eventAutoCancel, map[string]any{
			"reason": "pending_timeout",
		}); err != nil {
			log.Printf("[cron:auto_cancel] event emit failed for #%d: %v", b.ID, err)
		}
		if err := c.notifyAutoCancel(ctx, b); err != nil {
			log.Printf("[cron:auto_cancel] notify failed for #%d: %v", b.ID, err)
		}
		c.recomputePhotographerStats(ctx, b.PhotographerID)
	}
	return len(bookings), nil
}

// DefaultResolveDisputes resolves overdue disputes in favor of the opener.
// open + respond_deadline passed, or escalated + evidence_deadline passed.
func (c *CronService) DefaultResolveDisputes(ctx context.Context) (int, error) {
	disputes, err := c.queries.ListOverdueDisputes(ctx)
	if err != nil {
		return 0, fmt.Errorf("overdue dispute query failed: %w", err)
	}
	processed := 0
	for _, d := range disputes {
		b, err := c.queries.GetBookingByID(ctx, d.BookingID)
		if err != nil {
			log.Printf("[cron:dispute_default] booking #%d lookup failed: %v", d.BookingID, err)
			continue
		}

		resolution := "默认裁决：对方逾期未响应，申请方胜诉"
		updated, err := c.queries.UpdateDisputeStatus(ctx, repository.UpdateDisputeStatusParams{
			ID:         d.ID,
			Status:     disputeStatusResolved,
			Resolution: &resolution,
		})
		if err != nil {
			log.Printf("[cron:dispute_default] resolve dispute #%d failed: %v", d.ID, err)
			continue
		}
		processed++

		// Restore booking sub-status so the normal flow can continue.
		var restored *string
		if b.DeliveredAt != nil {
			restored = strPtr(subStatusDelivered)
		} else {
			restored = strPtr(subStatusInService)
		}
		if _, err := c.queries.UpdateBookingSubStatus(ctx, d.BookingID, restored); err != nil {
			log.Printf("[cron:dispute_default] restore sub_status for booking #%d failed: %v", d.BookingID, err)
		}

		if err := c.emitOrderEventForCron(ctx, d.BookingID, eventDisputeResolve, map[string]any{
			"dispute_id": d.ID,
			"resolution": resolution,
			"default":    true,
		}); err != nil {
			log.Printf("[cron:dispute_default] order event emit failed for #%d: %v", d.ID, err)
		}
		if _, err := c.queries.CreateDisputeEvent(ctx, repository.CreateDisputeEventParams{
			DisputeID: d.ID,
			ActorRole: strPtr(actorSystem),
			EventType: "auto_resolve",
			Payload:   []byte(`{"default":true,"resolution":"对方逾期未响应，申请方胜诉"}`),
		}); err != nil {
			log.Printf("[cron:dispute_default] dispute event emit failed for #%d: %v", d.ID, err)
		}
		if err := c.notifyDefaultResolve(ctx, b, d); err != nil {
			log.Printf("[cron:dispute_default] notify failed for #%d: %v", d.ID, err)
		}

		// Default resolution: the non-opener loses. Penalty -10, best-effort.
		loserUserID, loserRole := c.defaultLoser(ctx, b, d)
		if loserUserID > 0 {
			c.applyCreditDelta(ctx, loserUserID, -10, creditReasonDisputeLost, strPtr("dispute"), &d.ID, &d.BookingID, "争议默认裁决败诉")
			_ = loserRole
		}
		_ = updated
	}
	return processed, nil
}

// defaultLoser returns the user ID and role of the party that failed to respond.
func (c *CronService) defaultLoser(ctx context.Context, b repository.Booking, d repository.Dispute) (int64, string) {
	if d.OpenedByRole == actorCoser {
		profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
		if err == nil && profile.UserID != nil {
			return *profile.UserID, actorPhotographer
		}
	} else if d.OpenedByRole == actorPhotographer {
		return int64(b.CoserID), actorCoser
	}
	return 0, ""
}

// --- helpers shared with CronService ---

func (c *CronService) emitOrderEventForCron(ctx context.Context, bookingID int64, eventType orderEventType, payload map[string]any) error {
	if n, err := c.queries.CountSystemEventByBooking(ctx, bookingID, string(eventType)); err == nil && n > 0 {
		return nil
	}
	var payloadBytes []byte
	if len(payload) > 0 {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	_, err := c.queries.CreateOrderEvent(ctx, repository.CreateOrderEventParams{
		BookingID: bookingID,
		ActorRole: strPtr(actorSystem),
		EventType: string(eventType),
		Payload:   payloadBytes,
	})
	return err
}

func (c *CronService) notifyAutoAccept(ctx context.Context, b repository.Booking) error {
	linkType := notifyLinkOrder
	if _, err := c.queries.InsertNotification(ctx, int64(b.CoserID), "success", "拍摄已自动验收", fmt.Sprintf("预约(#%d)逾期未确认，系统自动完成验收", b.ID), &linkType, &b.ID); err != nil {
		return err
	}
	profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
	if err == nil && profile.UserID != nil {
		if _, err := c.queries.InsertNotification(ctx, *profile.UserID, "info", "拍摄已自动验收", fmt.Sprintf("预约(#%d)客户逾期未确认，系统自动完成", b.ID), &linkType, &b.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *CronService) notifyNoShow(ctx context.Context, b repository.Booking, party string) error {
	linkType := notifyLinkOrder
	title := "摄影师未到场"
	content := fmt.Sprintf("预约(#%d)摄影师未到场，已记录到平台", b.ID)
	if party == actorCoser {
		title = "客户未到场"
		content = fmt.Sprintf("预约(#%d)客户未到场", b.ID)
	}
	if _, err := c.queries.InsertNotification(ctx, int64(b.CoserID), "warning", title, content, &linkType, &b.ID); err != nil {
		return err
	}
	profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
	if err == nil && profile.UserID != nil {
		if _, err := c.queries.InsertNotification(ctx, *profile.UserID, "warning", title, content, &linkType, &b.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *CronService) notifyAutoCancel(ctx context.Context, b repository.Booking) error {
	linkType := notifyLinkOrder
	if _, err := c.queries.InsertNotification(ctx, int64(b.CoserID), "warning", "预约已自动取消", fmt.Sprintf("预约(#%d)长时间未确认，已自动取消", b.ID), &linkType, &b.ID); err != nil {
		return err
	}
	profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
	if err == nil && profile.UserID != nil {
		if _, err := c.queries.InsertNotification(ctx, *profile.UserID, "warning", "预约已自动取消", fmt.Sprintf("预约(#%d)长时间未确认，已自动取消", b.ID), &linkType, &b.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *CronService) notifyDefaultResolve(ctx context.Context, b repository.Booking, d repository.Dispute) error {
	linkType := notifyLinkOrder
	title := "争议已默认裁决"
	content := fmt.Sprintf("预约(#%d)的争议因对方逾期未响应，已默认裁决", d.BookingID)
	if _, err := c.queries.InsertNotification(ctx, int64(b.CoserID), "info", title, content, &linkType, &d.BookingID); err != nil {
		return err
	}
	profile, err := c.queries.GetPhotographerById(ctx, b.PhotographerID)
	if err == nil && profile.UserID != nil {
		if _, err := c.queries.InsertNotification(ctx, *profile.UserID, "info", title, content, &linkType, &d.BookingID); err != nil {
			return err
		}
	}
	return nil
}

func (c *CronService) recomputePhotographerStats(ctx context.Context, photographerID int32) {
	if err := c.queries.RecomputePhotographerStats(ctx, int64(photographerID)); err != nil {
		log.Printf("cron recompute stats failed for photographer %d: %v", photographerID, err)
	}
}

func (c *CronService) applyCreditDelta(ctx context.Context, userID int64, delta int32, reason string, refType *string, refID *int64, bookingID *int64, desc string) {
	if _, err := c.queries.UpsertCreditScore(ctx, userID, nil); err != nil {
		log.Printf("credit upsert score failed: %v", err)
		return
	}
	ce, err := c.queries.CreateCreditEvent(ctx, repository.CreateCreditEventParams{
		UserID:      userID,
		Delta:       delta,
		Reason:      reason,
		RefType:     refType,
		RefID:       refID,
		BookingID:   bookingID,
		Description: &desc,
	})
	if err != nil {
		if !isUniqueViolation(err) {
			log.Printf("credit create event failed: %v", err)
		}
		return
	}
	if ce.ID == 0 {
		return
	}
	if _, err := c.queries.AddCreditDelta(ctx, userID, delta); err != nil {
		log.Printf("credit add delta failed: %v", err)
	}
}

// dummy no-op to keep pgx import used by helper code paths if needed.
var _ = pgx.ErrNoRows
