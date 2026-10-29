package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
)

type CreateBookingRequest struct {
	PhotographerID int32  `json:"photographerId" binding:"required"`
	CoserID        int32  `json:"coserId"`
	ServiceID      int32  `json:"serviceId" binding:"required"`
	Date           string `json:"date" binding:"required"`
	Time           string `json:"time" binding:"required"`
	Remarks        string `json:"remarks"`
}

type BookingItem struct {
	ID                 int64    `json:"id"`
	PhotographerID     int32    `json:"photographerId"`
	CoserID            int32    `json:"coserId"`
	ServiceID          int32    `json:"serviceId"`
	Date               string   `json:"date"`
	Time               string   `json:"time"`
	Status             string   `json:"status"`
	SubStatus          *string  `json:"subStatus"`
	TotalPrice         int32    `json:"totalPrice"`
	PriceMode          string   `json:"priceMode"`
	QuotePrice         *int32   `json:"quotePrice"`
	PriceStatus        string   `json:"priceStatus"`
	Remarks            string   `json:"remarks"`
	CreatedAt          string   `json:"createdAt"`
	CheckinAt          *string  `json:"checkinAt"`
	DeliveredAt        *string  `json:"deliveredAt"`
	AcceptDeadline     *string  `json:"acceptDeadline"`
	DeliveryEvidence   []string `json:"deliveryEvidence"`
	CheckinCode        *string  `json:"checkinCode"`
	NoShowParty        *string  `json:"noShowParty"`
	CancelledBy        *string  `json:"cancelledBy"`
	PhotographerName   string   `json:"photographerName"`
	PhotographerAvatar string   `json:"photographerAvatar"`
	PhotographerUserID int64    `json:"photographerUserId"`
	ServiceName        string   `json:"serviceName"`
	// ServiceDuration is the下单时冻结的时长快照（分钟）。指针对待：快照列可能为空
	// （000027 之前的旧行），此时前端应显示「—」而不是 0 分钟。
	ServiceDuration *int32 `json:"serviceDuration"`
}

type BookingItemWithCoser struct {
	BookingItem
	CoserName   string `json:"coserName"`
	CoserAvatar string `json:"coserAvatar"`
	CoserPhone  string `json:"coserPhone"`
}

type BookingService struct {
	queries    *repository.Queries
	configSalt func() string
}

func NewBookingService(queries *repository.Queries) *BookingService {
	return NewBookingServiceWithSalt(queries, defaultCheckinSalt)
}

// NewBookingServiceWithSalt allows the main package to inject the server-side
// secret used for deterministic check-in code generation.
func NewBookingServiceWithSalt(queries *repository.Queries, saltFn func() string) *BookingService {
	return &BookingService{queries: queries, configSalt: saltFn}
}

// defaultCheckinSalt is a last-resort fallback; the real service uses WXAppSecret.
func defaultCheckinSalt() string { return "miracomic-checkin-fallback-salt" }

// checkinSalt returns the injected server-side secret.
func (s *BookingService) checkinSalt() string {
	return s.configSalt()
}

var (
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrForbidden         = errors.New("forbidden")
	ErrQuoteNotAllowed   = errors.New("quote not allowed in current state")
	// ErrPastDate：不允许预约已经过去的日期/时刻。前端选择器也把下限设成今天，
	// 但那是 UX；这里才是权威校验（小程序端可绕）。
	ErrPastDate = errors.New("cannot book a date in the past")
	// ErrSelfBooking：摄影师不能预约自己的套餐。自成交没业务意义，还会污染订单数、
	// 并为「自己给自己评价」铺路。
	ErrSelfBooking = errors.New("cannot book your own service")
)

// canTransition guards the four legacy top-level states. Sub-status flows are
// checked separately by canSubTransition.
func canTransition(from, to string) bool {
	switch from {
	case "pending":
		return to == "confirmed" || to == "cancelled"
	case "confirmed":
		return to == "completed" || to == "cancelled"
	default:
		return false
	}
}

// subStatus constants used by the escrow node state machine.
const (
	subStatusInService = "in_service"
	subStatusDelivered = "delivered"
	subStatusDisputed  = "disputed"
)

// subStatusOr returns the current sub-status or "" when nil.
func subStatusOr(b repository.Booking) string {
	if b.SubStatus == nil {
		return ""
	}
	return *b.SubStatus
}

// canSubTransition validates transitions of the escrow sub-status layer.
// Returns (next, allowed). When allowed=false, next equals from.
func canSubTransition(from string) (string, bool) {
	switch from {
	case "":
		return subStatusInService, true
	case subStatusInService:
		return subStatusDelivered, true
	case subStatusDelivered:
		return "", true // completion clears sub_status
	case subStatusDisputed:
		return "", false
	default:
		return from, false
	}
}

// disputeStatus constants.
const (
	disputeStatusOpen       = "open"
	disputeStatusResponding = "responding"
	disputeStatusEscalated  = "escalated"
	disputeStatusResolved   = "resolved"
	disputeStatusClosed     = "closed"
)

// dispute type constants.
const (
	disputeTypeNoShow      = "no_show"
	disputeTypeNotDelivered = "not_delivered"
	disputeTypeQuality     = "quality"
	disputeTypeOther       = "other"
)

// isPastSlot 判断请求的档期是否已经过去，按**服务器本地时区**判断（唯一能自证的时钟）。
// 日期是硬约束；时刻是尽力而为：今天的、已经过去的小时同样拒绝；若时刻字符串解析不了，
// 退化为「只比日期」——宁可放行一个过期时刻，也不要误杀一个合法预约。
func isPastSlot(date time.Time, clock string, now time.Time) bool {
	y, m, d := date.Date()
	if h, mm, ok := parseClock(clock); ok {
		slot := time.Date(y, m, d, h, mm, 0, 0, now.Location())
		return slot.Before(now)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Before(today)
}

func parseClock(s string) (int, int, bool) {
	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Hour(), t.Minute(), true
		}
	}
	return 0, 0, false
}

func bookingToItem(b repository.Booking) *BookingItem {
	item := &BookingItem{
		ID:               b.ID,
		PhotographerID:   b.PhotographerID,
		CoserID:          b.CoserID,
		ServiceID:        b.ServiceID,
		Date:             b.Date.Format("2006-01-02"),
		Time:             b.Time,
		Status:           b.Status,
		SubStatus:        b.SubStatus,
		TotalPrice:       b.TotalPrice,
		PriceMode:        b.PriceMode,
		QuotePrice:       b.QuotePrice,
		PriceStatus:      b.PriceStatus,
		Remarks:          derefString(b.Remarks),
		CreatedAt:        b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		DeliveryEvidence: b.DeliveryEvidence,
		NoShowParty:      b.NoShowParty,
		CancelledBy:      b.CancelledBy,
		ServiceName:      derefString(b.ServiceName),
		ServiceDuration:  b.ServiceDuration,
	}
	if b.CheckinAt != nil {
		s := b.CheckinAt.Format("2006-01-02T15:04:05Z07:00")
		item.CheckinAt = &s
	}
	if b.DeliveredAt != nil {
		s := b.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		item.DeliveredAt = &s
	}
	if b.AcceptDeadline != nil {
		s := b.AcceptDeadline.Format("2006-01-02T15:04:05Z07:00")
		item.AcceptDeadline = &s
	}
	return item
}

// notify writes a notification after a booking event; errors are logged and
// ignored — notification loss must never fail the booking action itself.
const (
	notifyLinkOrder             = "order"
	notifyLinkPhotographerOrder = "photographer_orders"
)

// orderEventType for order_events.event_type.
type orderEventType string

const (
	eventConfirmed      orderEventType = "confirmed"
	eventCheckin                       = "checkin"
	eventDeliver                       = "deliver"
	eventAccept                        = "accept"
	eventAutoAccept                    = "auto_accept"
	eventCompleteLegacy                = "complete_legacy"
	eventCancel                        = "cancel"
	eventAutoCancel                    = "auto_cancel"
	eventNoShow                        = "noshow"
	eventDisputeOpen                   = "dispute_open"
	eventDisputeRespond                = "dispute_respond"
	eventDisputeEscalate               = "dispute_escalate"
	eventDisputeResolve                = "dispute_resolve"
	eventCreditDelta                   = "credit_delta"
)

func (s *BookingService) notify(ctx context.Context, userID int32, typ, title, content string, bookingID int64) {
	s.notifyLinked(ctx, userID, typ, title, content, notifyLinkOrder, bookingID)
}

func (s *BookingService) notifyPhotographer(ctx context.Context, photographerID int32, typ, title, content string, bookingID int64) {
	profile, err := s.queries.GetPhotographerById(ctx, photographerID)
	if err != nil || profile.UserID == nil {
		return
	}
	s.notifyLinked(ctx, int32(*profile.UserID), typ, title, content, notifyLinkPhotographerOrder, bookingID)
}

func (s *BookingService) notifyLinked(ctx context.Context, userID int32, typ, title, content, linkType string, bookingID int64) {
	if _, err := s.queries.InsertNotification(ctx, int64(userID), typ, title, content, &linkType, &bookingID); err != nil {
		log.Printf("booking notify failed: %v", err)
	}
}

// emitOrderEvent writes an append-only event for a booking. System/admin events
// are deduplicated by (booking_id, event_type). Errors are logged and ignored.
func (s *BookingService) emitOrderEvent(ctx context.Context, bookingID int64, actorID int64, actorRole string, eventType orderEventType, payload map[string]any) {
	if eventType == eventAutoAccept || eventType == eventAutoCancel || eventType == eventNoShow {
		// actorRole system/admin events: dedupe by (booking_id, event_type).
		if n, err := s.queries.CountSystemEventByBooking(ctx, bookingID, string(eventType)); err == nil && n > 0 {
			return
		}
	}
	var payloadBytes []byte
	if len(payload) > 0 {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			log.Printf("booking event payload marshal failed: %v", err)
			payloadBytes = []byte("{}")
		}
	}
	actorIDPtr := &actorID
	actorRolePtr := &actorRole
	if actorID == 0 {
		actorIDPtr = nil
	}
	if actorRole == "" {
		actorRolePtr = nil
	}
	if _, err := s.queries.CreateOrderEvent(ctx, repository.CreateOrderEventParams{
		BookingID: bookingID,
		ActorID:   actorIDPtr,
		ActorRole: actorRolePtr,
		EventType: string(eventType),
		Payload:   payloadBytes,
	}); err != nil {
		log.Printf("booking event emit failed: %v", err)
	}
}

// dedupedNotify prevents duplicate notifications for the same booking transition
// if the database layer supports lookup; otherwise falls back to unconditional notify.
func (s *BookingService) dedupedNotify(ctx context.Context, userID int32, typ, title, content string, bookingID int64, eventType orderEventType) {
	if eventType == eventConfirmed || eventType == eventCheckin || eventType == eventDeliver ||
		eventType == eventAccept || eventType == eventCompleteLegacy {
		if n, err := s.queries.CountSystemEventByBooking(ctx, bookingID, string(eventType)); err == nil && n > 0 {
			return
		}
	}
	s.notify(ctx, userID, typ, title, content, bookingID)
}

func (s *BookingService) Create(ctx context.Context, req CreateBookingRequest) (*BookingItem, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}
	if isPastSlot(date, req.Time, time.Now()) {
		return nil, ErrPastDate
	}

	if profile, err := s.queries.GetPhotographerById(ctx, req.PhotographerID); err == nil {
		// 摄影师不能给自己下单：自成交没有业务含义，且会让「自己给自己评价」变成可能。
		if profile.UserID != nil && *profile.UserID == int64(req.CoserID) {
			return nil, ErrSelfBooking
		}
		if profile.UserID != nil {
			blocked, err := s.queries.IsBlockedPair(ctx, int64(req.CoserID), *profile.UserID)
			if err == nil && blocked {
				return nil, ErrForbidden
			}
		}
	}

	conflicts, err := s.queries.CountConflictBookings(ctx, req.PhotographerID, date, req.Time)
	if err != nil {
		return nil, err
	}
	if conflicts > 0 {
		return nil, ErrConflict
	}

	var remarks *string
	if req.Remarks != "" {
		remarks = &req.Remarks
	}

	// 套餐归属校验：套餐必须归属于被预约的摄影师，且**必须处于上架状态**。平台模板套餐
	// （photographer_id IS NULL）不作为可售商品，一律拒绝；套餐不存在或已下架（ErrNoRows）
	// 同样视为无效引用 —— 下架校验放在这里，避免「列表里看不到、下单却能成」。
	// 不掩盖真实 DB 故障。
	serviceID64 := int64(req.ServiceID)
	ownerID, err := s.queries.GetBookableServicePhotographerID(ctx, serviceID64)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidReference
		}
		return nil, err
	}
	if ownerID == nil || int64(req.PhotographerID) != *ownerID {
		return nil, ErrInvalidReference
	}

	// 按套餐价格推导定价模式，并冻结名称/时长快照。
	// price IS NULL = 面议；price = 0 = 互勉；price > 0 = 固定价。
	// total_price 为 NOT NULL，互勉/面议写 0 占位，展示层靠 price_status 区分。
	var totalPrice int32
	priceMode := "fixed"
	priceStatus := "agreed"
	var snapName *string
	var snapDuration *int32
	svc, err := s.queries.GetServiceById(ctx, serviceID64)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidReference
		}
		return nil, err
	}
	if svc.Price != nil {
		totalPrice = *svc.Price
	}
	snapDuration = &svc.Duration
	name := svc.Name
	snapName = &name
	switch {
	case svc.Price == nil:
		priceMode, priceStatus = "negotiable", "awaiting_quote"
	case *svc.Price == 0:
		priceMode = "mutual"
	}

	booking, err := s.queries.CreateBooking(ctx, repository.CreateBookingParams{
		PhotographerID:  req.PhotographerID,
		CoserID:         req.CoserID,
		ServiceID:       req.ServiceID,
		Date:            date,
		Time:            req.Time,
		Status:          "pending",
		TotalPrice:      totalPrice,
		Remarks:         remarks,
		PriceMode:       priceMode,
		PriceStatus:     priceStatus,
		ServiceName:     snapName,
		ServiceDuration: snapDuration,
	})
	if err != nil {
		// 唯一约束 uniq_bookings_active_slot 是并发下的最终防线：两个请求可能都通过了上面的
		// 冲突检查（TOCTOU），但只有一条能落库，另一条在这里被 DB 挡下 → 仍然回 409。
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		if isForeignKeyViolation(err) {
			return nil, ErrInvalidReference
		}
		return nil, err
	}

	s.notify(ctx, req.CoserID, "success", "预约成功", fmt.Sprintf("您的摄影预约(#%d)已提交，等待摄影师确认", booking.ID), booking.ID)
	s.notifyPhotographer(ctx, req.PhotographerID, "info", "收到新预约", fmt.Sprintf("您收到一条新预约(#%d)，请及时确认接单", booking.ID), booking.ID)

	return bookingToItem(booking), nil
}

func (s *BookingService) ListByUser(ctx context.Context, userID int32) ([]BookingItem, error) {
	bookings, err := s.queries.GetBookingsByUserWithDetails(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]BookingItem, 0, len(bookings))
	for _, b := range bookings {
		item := bookingToItem(b.Booking)
		item.PhotographerName = b.PhotographerName
		item.PhotographerAvatar = b.PhotographerAvatar
		item.PhotographerUserID = derefInt64(b.PhotographerUserID)
		// 显示优先级：快照 b.service_name > 实时 join s.name（为快照回填前的旧行兜底）。
		item.ServiceName = derefString(b.Booking.ServiceName)
		if item.ServiceName == "" {
			item.ServiceName = b.ServiceName
		}
		// Coser 视角：confirmed 且没码时惰性生成一次； always 返回真实码。
		if b.Booking.Status == "confirmed" {
			updated, gerr := s.queries.GenerateCheckinCode(ctx, b.Booking.ID, s.checkinSalt())
			if gerr == nil {
				item.CheckinCode = updated.CheckinCode
			} else {
				item.CheckinCode = b.Booking.CheckinCode
			}
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *BookingService) ListByPhotographerForUser(ctx context.Context, photographerID int32, userID int64) ([]BookingItemWithCoser, error) {
	profile, err := s.queries.GetPhotographerByUserID(ctx, userID)
	if err != nil || profile.ID != photographerID {
		return nil, ErrForbidden
	}
	return s.ListByPhotographer(ctx, photographerID)
}

func (s *BookingService) ListByPhotographer(ctx context.Context, photographerID int32) ([]BookingItemWithCoser, error) {
	rows, err := s.queries.GetBookingsByPhotographer(ctx, photographerID)
	if err != nil {
		return nil, err
	}
	items := make([]BookingItemWithCoser, 0, len(rows))
	for _, b := range rows {
		item := bookingToItem(b.Booking)
		// 显示优先级：快照 b.service_name > 实时 join s.name（为快照回填前的旧行兜底）。
		item.ServiceName = derefString(b.Booking.ServiceName)
		if item.ServiceName == "" {
			item.ServiceName = derefString(b.ServiceName)
		}
		// 摄影师视角：核销码必须对摄影师隐藏。
		item.CheckinCode = nil
		items = append(items, BookingItemWithCoser{
			BookingItem: *item,
			CoserName:   b.CoserName,
			CoserAvatar: b.CoserAvatar,
			CoserPhone:  b.CoserPhone,
		})
	}
	return items, nil
}

func (s *BookingService) UpdateStatus(ctx context.Context, bookingID int64, newStatus string, actorUserID int64, actorTag string) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if actorTag == "photographer" {
		profile, err := s.queries.GetPhotographerByUserID(ctx, actorUserID)
		if err != nil {
			return nil, ErrForbidden
		}
		if int64(profile.ID) != int64(b.PhotographerID) {
			return nil, ErrForbidden
		}
	} else if actorTag == "coser" {
		if int64(b.CoserID) != actorUserID {
			return nil, ErrForbidden
		}
	} else {
		// unknown actorTag — default-deny so a stray value cannot bypass ownership checks
		return nil, ErrForbidden
	}
	if !canTransition(b.Status, newStatus) {
		return nil, ErrInvalidTransition
	}
	// 面议订单在价格达成一致前不得置为 confirmed：否则会跳过报价/响应流程
	//（二者都要求 status='pending'），永远停在 total_price=0。
	if newStatus == "confirmed" && b.PriceMode == "negotiable" && b.PriceStatus != "agreed" {
		return nil, ErrInvalidTransition
	}
	updated, err := s.queries.UpdateBookingStatus(ctx, bookingID, newStatus, b.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The status changed between our read and this write (double confirm,
			// confirm-vs-cancel). Tell the caller it lost the race instead of
			// silently overwriting the winner.
			return nil, ErrConflict
		}
		return nil, err
	}

	// 确认接单后立刻生成核销码，保证 coser 在到场前能出示。
	if newStatus == "confirmed" {
		if g, gerr := s.queries.GenerateCheckinCode(ctx, bookingID, s.checkinSalt()); gerr == nil {
			updated = g
		}
	}

	switch newStatus {
	case "confirmed":
		s.notify(ctx, b.CoserID, "info", "预约已确认", "摄影师已确认接单，请按时赴约", b.ID)
	case "cancelled":
		s.notify(ctx, b.CoserID, "warning", "预约已取消", "您的预约已被取消", b.ID)
		if actorTag == "coser" {
			s.notifyPhotographer(ctx, b.PhotographerID, "warning", "预约已取消", "对方取消了预约。", b.ID)
		}
	case "completed":
		s.notify(ctx, b.CoserID, "success", "拍摄已完成", "记得去评价本次拍摄哦", b.ID)
	}

	if newStatus == "completed" {
		// 旧 UpdateStatus 直接置 completed 视为 legacy 完成事件，保留审计。
		s.emitOrderEvent(ctx, b.ID, actorUserID, actorTag, eventCompleteLegacy, nil)
	}

	// 单量（order_count）随状态流转变化：完成 +1、取消 -1。整表重算而不是加减，
	// 幂等且能自我修正历史上的错误计数。失败只记日志，不影响订单流转结果。
	s.recomputeStats(ctx, b.PhotographerID)

	return bookingToItem(updated), nil
}

// recomputeStats 重算摄影师对外的展示数字（评分/评价数/单量）。best-effort。
func (s *BookingService) recomputeStats(ctx context.Context, photographerID int32) {
	if err := s.queries.RecomputePhotographerStats(ctx, int64(photographerID)); err != nil {
		log.Printf("recompute photographer stats failed: %v", err)
	}
}

// AdminUpdateStatus bypasses actor ownership checks; the admin is neither coser nor photographer.
// State-machine transition validation (canTransition) is STILL enforced.
func (s *BookingService) AdminUpdateStatus(ctx context.Context, bookingID int64, newStatus string) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if !canTransition(b.Status, newStatus) {
		return nil, ErrInvalidTransition
	}
	// 面议订单在价格达成一致前不得置为 confirmed（含管理端），否则会锁死报价流程。
	if newStatus == "confirmed" && b.PriceMode == "negotiable" && b.PriceStatus != "agreed" {
		return nil, ErrInvalidTransition
	}
	updated, err := s.queries.UpdateBookingStatus(ctx, bookingID, newStatus, b.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The status changed between our read and this write (double confirm,
			// confirm-vs-cancel). Tell the caller it lost the race instead of
			// silently overwriting the winner.
			return nil, ErrConflict
		}
		return nil, err
	}
	switch newStatus {
	case "confirmed":
		s.notify(ctx, b.CoserID, "info", "预约已确认", "摄影师已确认接单，请按时赴约", b.ID)
		s.notifyPhotographer(ctx, b.PhotographerID, "info", "预约已确认", "客户预约已由管理员确认。", b.ID)
	case "cancelled":
		s.notify(ctx, b.CoserID, "warning", "预约已取消", "您的预约已被取消", b.ID)
		s.notifyPhotographer(ctx, b.PhotographerID, "warning", "预约已取消", "该预约已被取消。", b.ID)
	case "completed":
		s.notify(ctx, b.CoserID, "success", "拍摄已完成", "记得去评价本次拍摄哦", b.ID)
		s.notifyPhotographer(ctx, b.PhotographerID, "success", "拍摄已完成", "该预约已标记完成。", b.ID)
	}
	// 管理员流转同样影响单量，走同一套重算。
	s.recomputeStats(ctx, b.PhotographerID)
	return bookingToItem(updated), nil
}

// actorRole helpers.
const (
	actorCoser        = "coser"
	actorPhotographer = "photographer"
	actorAdmin        = "admin"
	actorSystem       = "system"
)



// Credit score constants.
const (
	creditReasonNoShow          = "photographer_no_show"
	creditReasonCoserNoShow     = "coser_no_show"
	creditReasonDisputeLost     = "dispute_lost"
	creditReasonOrderComplete   = "order_completed"
	creditReasonLateCancel      = "late_cancel"
	creditReasonDeliveryOverdue = "delivery_overdue"
)

// applyCreditDelta idempotently applies a credit delta for a user. The unique
// index on (user_id, ref_type, ref_id, reason) makes concurrent/duplicate calls
// a no-op. Errors are logged and ignored so credit never blocks the main flow.
//
// 顺序：先尝试插入 credit_events（唯一索引保证幂等）。如果事件已存在，说明该事实
// 已经计过分，直接跳过 score 更新，避免重复扣分/加分。
func (s *BookingService) applyCreditDelta(ctx context.Context, userID int64, delta int32, reason string, refType *string, refID *int64, bookingID *int64, desc string) {
	if _, err := s.queries.UpsertCreditScore(ctx, userID, nil); err != nil {
		log.Printf("credit upsert score failed: %v", err)
		return
	}
	ce, err := s.queries.CreateCreditEvent(ctx, repository.CreateCreditEventParams{
		UserID:      userID,
		Delta:       delta,
		Reason:      reason,
		RefType:     refType,
		RefID:       refID,
		BookingID:   bookingID,
		Description: &desc,
	})
	if err != nil {
		// Unique constraint violation is expected for idempotent retries.
		if !isUniqueViolation(err) {
			log.Printf("credit create event failed: %v", err)
		}
		return
	}
	if ce.ID == 0 {
		// DO NOTHING won the conflict: the event already exists, do not mutate score again.
		return
	}
	if _, err := s.queries.AddCreditDelta(ctx, userID, delta); err != nil {
		log.Printf("credit add delta failed: %v", err)
	}
}

// Checkin lets the booking's photographer confirm the coser has arrived.
// It is idempotent: repeated calls return the already-checked-in booking.
func (s *BookingService) Checkin(ctx context.Context, bookingID int64, actorUserID int64, code string) (*BookingItem, error) {
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
	// Already checked in: idempotent read (only if sub_status is in_service).
	if b.CheckinAt != nil && b.SubStatus != nil && *b.SubStatus == subStatusInService {
		return bookingToItem(b), nil
	}
	// Prevent accidental "checkin then deliver" skip: a delivered booking must
	// not be allowed to check in again.
	if subStatusOr(b) == subStatusDelivered {
		return nil, ErrInvalidTransition
	}
	if code == "" || b.CheckinCode == nil || code != *b.CheckinCode {
		return nil, ErrForbidden
	}
	updated, err := s.queries.Checkin(ctx, bookingID, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return nil, err
	}
	s.emitOrderEvent(ctx, bookingID, actorUserID, actorPhotographer, eventCheckin, map[string]any{"code": code})
	s.dedupedNotify(ctx, b.CoserID, "info", "预约已签到", "摄影师已确认您已到场，拍摄即将开始", bookingID, eventCheckin)
	return bookingToItem(updated), nil
}

// MarkDelivered records the photographer's delivery of the service.
func (s *BookingService) MarkDelivered(ctx context.Context, bookingID int64, actorUserID int64, evidence []string) (*BookingItem, error) {
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
	ss := subStatusOr(b)
	if ss == subStatusDelivered {
		return bookingToItem(b), nil
	}
	if ss != subStatusInService {
		return nil, ErrInvalidTransition
	}
	deadline := time.Now().Add(7 * 24 * time.Hour)
	updated, err := s.queries.DeliverBooking(ctx, repository.DeliverBookingParams{
		ID:               bookingID,
		DeliveryEvidence: evidence,
		AcceptDeadline:   deadline,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return nil, err
	}
	s.emitOrderEvent(ctx, bookingID, actorUserID, actorPhotographer, eventDeliver, map[string]any{"evidence_count": len(evidence)})
	s.dedupedNotify(ctx, b.CoserID, "info", "拍摄已交付", "摄影师已标记拍摄完成，请在 7 天内确认验收", bookingID, eventDeliver)
	return bookingToItem(updated), nil
}

// AcceptDelivery lets the coser accept a delivered booking and mark it complete.
func (s *BookingService) AcceptDelivery(ctx context.Context, bookingID int64, actorUserID int64) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if int64(b.CoserID) != actorUserID {
		return nil, ErrForbidden
	}
	if b.Status != "confirmed" || subStatusOr(b) != subStatusDelivered {
		return nil, ErrInvalidTransition
	}
	if b.DeliveredAt == nil {
		return nil, ErrInvalidTransition
	}
	updated, err := s.queries.AcceptDelivery(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return nil, err
	}
	s.emitOrderEvent(ctx, bookingID, actorUserID, actorCoser, eventAccept, nil)
	s.notify(ctx, b.CoserID, "success", "拍摄已完成", "您已确认验收，记得去评价本次拍摄哦", bookingID)
	s.notifyPhotographer(ctx, b.PhotographerID, "success", "拍摄已完成", "客户已确认验收", bookingID)
	s.recomputeStats(ctx, b.PhotographerID)
	// 履约完成 +2（单笔 booking 维度幂等，靠 credit_events 唯一索引兜底）。
	s.applyCreditDelta(ctx, actorUserID, 2, creditReasonOrderComplete, strPtr("booking"), &bookingID, &bookingID, "订单履约完成")
	return bookingToItem(updated), nil
}


// OpenDispute starts an escrow dispute for a confirmed booking.
func (s *BookingService) OpenDispute(ctx context.Context, bookingID int64, actorUserID int64, actorRole, disputeType, reason string) (*repository.Dispute, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
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
	if b.Status != "confirmed" {
		return nil, ErrInvalidTransition
	}
	ss := subStatusOr(b)
	if ss != subStatusInService && ss != subStatusDelivered && ss != "" {
		return nil, ErrInvalidTransition
	}
	// 已存在未关闭争议则禁止重复开启。
	if existing, err := s.queries.GetOpenDisputeByBooking(ctx, bookingID); err == nil && existing.ID != 0 {
		return nil, ErrConflict
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// Snapshot current sub_status as the restore point on resolution.
	now := time.Now()
	dispute, err := s.queries.CreateDispute(ctx, repository.CreateDisputeParams{
		BookingID:        bookingID,
		OpenedBy:         actorUserID,
		OpenedByRole:     actorRole,
		Type:             disputeType,
		Reason:           reason,
		RespondDeadline:  now.Add(48 * time.Hour),
		EvidenceDeadline: now.Add(72 * time.Hour),
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.queries.UpdateBookingSubStatus(ctx, bookingID, strPtr(subStatusDisputed)); err != nil {
		log.Printf("dispute set sub_status failed: %v", err)
	}
	s.emitOrderEvent(ctx, bookingID, actorUserID, actorRole, eventDisputeOpen, map[string]any{"dispute_id": dispute.ID, "type": disputeType})
	s.notify(ctx, b.CoserID, "warning", "争议已开启", fmt.Sprintf("预约(#%d)已发起争议，等待处理", bookingID), bookingID)
	s.notifyPhotographer(ctx, b.PhotographerID, "warning", "争议已开启", fmt.Sprintf("预约(#%d)已被对方发起争议", bookingID), bookingID)
	return &dispute, nil
}

// RespondDispute lets the non-opener reply within the response window.
func (s *BookingService) RespondDispute(ctx context.Context, disputeID int64, actorUserID int64, actorRole, response string) (*repository.Dispute, error) {
	d, err := s.queries.GetDisputeByID(ctx, disputeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if d.Status != disputeStatusOpen && d.Status != disputeStatusResponding {
		return nil, ErrInvalidTransition
	}
	if time.Now().After(d.RespondDeadline) {
		return nil, ErrInvalidTransition
	}
	if actorRole == d.OpenedByRole {
		return nil, ErrForbidden
	}
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
	updated, err := s.queries.UpdateDisputeStatus(ctx, repository.UpdateDisputeStatusParams{
		ID:     disputeID,
		Status: disputeStatusResponding,
	})
	if err != nil {
		return nil, err
	}
	s.emitOrderEvent(ctx, d.BookingID, actorUserID, actorRole, eventDisputeRespond, map[string]any{"dispute_id": disputeID, "response": response})
	return &updated, nil
}

// ResolveDispute closes a dispute and restores the booking sub-status.
func (s *BookingService) ResolveDispute(ctx context.Context, disputeID int64, adminUserID int64, resolution string, penaltyParty *string) (*repository.Dispute, error) {
	d, err := s.queries.GetDisputeByID(ctx, disputeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if d.Status != disputeStatusOpen && d.Status != disputeStatusResponding && d.Status != disputeStatusEscalated {
		return nil, ErrInvalidTransition
	}
	b, err := s.queries.GetBookingByID(ctx, d.BookingID)
	if err != nil {
		return nil, err
	}
	updated, err := s.queries.UpdateDisputeStatus(ctx, repository.UpdateDisputeStatusParams{
		ID:         disputeID,
		Status:     disputeStatusClosed,
		Resolution: &resolution,
		AdminID:    &adminUserID,
	})
	if err != nil {
		return nil, err
	}
	// Restore booking sub-status to allow the normal flow to continue.
	// A closed dispute after delivery stays delivered; otherwise back to in_service.
	var restored *string
	if b.DeliveredAt != nil {
		restored = strPtr(subStatusDelivered)
	} else {
		restored = strPtr(subStatusInService)
	}
	if _, err := s.queries.UpdateBookingSubStatus(ctx, d.BookingID, restored); err != nil {
		log.Printf("dispute restore sub_status failed: %v", err)
	}
	s.emitOrderEvent(ctx, d.BookingID, adminUserID, actorAdmin, eventDisputeResolve, map[string]any{"dispute_id": disputeID, "resolution": resolution})
	s.notify(ctx, b.CoserID, "info", "争议已处理", fmt.Sprintf("预约(#%d)的争议已由平台处理，结果：%s", d.BookingID, resolution), d.BookingID)
	s.notifyPhotographer(ctx, b.PhotographerID, "info", "争议已处理", fmt.Sprintf("预约(#%d)的争议已由平台处理", d.BookingID), d.BookingID)
	// 败诉 -10。
	if penaltyParty != nil {
		if *penaltyParty == actorCoser {
			s.applyCreditDelta(ctx, int64(b.CoserID), -10, creditReasonDisputeLost, strPtr("dispute"), &disputeID, &d.BookingID, "争议败诉")
		} else if *penaltyParty == actorPhotographer {
			profile, err := s.queries.GetPhotographerById(ctx, b.PhotographerID)
			if err == nil && profile.UserID != nil {
				s.applyCreditDelta(ctx, *profile.UserID, -10, creditReasonDisputeLost, strPtr("dispute"), &disputeID, &d.BookingID, "争议败诉")
			}
		}
	}
	return &updated, nil
}

// Quote records a single-round price offer from the booking's own photographer.
// Allowed only while the booking is still pending and price_mode='negotiable'
// AND price_status='awaiting_quote'; the Go-level status guard rejects non-pending
// bookings first, and the SQL state guard reports RowsAffected==0 for anything else.
func (s *BookingService) Quote(ctx context.Context, bookingID int64, actorUserID int64, price int32) (*BookingItem, error) {
	if price <= 0 || price > 99999 {
		return nil, ErrInvalidPrice
	}
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
	if b.Status != "pending" {
		return nil, ErrQuoteNotAllowed
	}
	n, err := s.queries.QuoteBooking(ctx, bookingID, price, b.PhotographerID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrQuoteNotAllowed
	}
	updated, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return bookingToItem(updated), nil
}

// RespondQuote lets the booking's own coser accept or reject a pending quote.
// Accept sets total_price=quote_price, price_status='agreed', status='confirmed';
// reject sets price_status='rejected', status='cancelled' (both atomically in SQL).
func (s *BookingService) RespondQuote(ctx context.Context, bookingID int64, actorUserID int64, accept bool) (*BookingItem, error) {
	b, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	if int64(b.CoserID) != actorUserID {
		return nil, ErrForbidden
	}
	if b.Status != "pending" {
		return nil, ErrQuoteNotAllowed
	}
	n, err := s.queries.RespondBookingQuote(ctx, bookingID, b.CoserID, accept)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrQuoteNotAllowed
	}
	updated, err := s.queries.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return bookingToItem(updated), nil
}

func (s *BookingService) GetOccupiedTimes(ctx context.Context, photographerID int32, dateStr string) ([]string, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	times, err := s.queries.GetOccupiedTimesByPhotographerDate(ctx, photographerID, date)
	if err != nil {
		return nil, err
	}
	if times == nil {
		times = []string{}
	}
	return times, nil
}
