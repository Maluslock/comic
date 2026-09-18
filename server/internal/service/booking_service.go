package service

import (
	"context"
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
	ID                 int64  `json:"id"`
	PhotographerID     int32  `json:"photographerId"`
	CoserID            int32  `json:"coserId"`
	ServiceID          int32  `json:"serviceId"`
	Date               string `json:"date"`
	Time               string `json:"time"`
	Status             string `json:"status"`
	TotalPrice         int32  `json:"totalPrice"`
	PriceMode          string `json:"priceMode"`
	QuotePrice         *int32 `json:"quotePrice"`
	PriceStatus        string `json:"priceStatus"`
	Remarks            string `json:"remarks"`
	CreatedAt          string `json:"createdAt"`
	PhotographerName   string `json:"photographerName"`
	PhotographerAvatar string `json:"photographerAvatar"`
	PhotographerUserID int64  `json:"photographerUserId"`
	ServiceName        string `json:"serviceName"`
}

type BookingItemWithCoser struct {
	BookingItem
	CoserName   string `json:"coserName"`
	CoserAvatar string `json:"coserAvatar"`
	CoserPhone  string `json:"coserPhone"`
}

type BookingService struct {
	queries *repository.Queries
}

func NewBookingService(queries *repository.Queries) *BookingService {
	return &BookingService{queries: queries}
}

var (
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrForbidden         = errors.New("forbidden")
	ErrQuoteNotAllowed   = errors.New("quote not allowed in current state")
)

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

func bookingToItem(b repository.Booking) *BookingItem {
	return &BookingItem{
		ID:             b.ID,
		PhotographerID: b.PhotographerID,
		CoserID:        b.CoserID,
		ServiceID:      b.ServiceID,
		Date:           b.Date.Format("2006-01-02"),
		Time:           b.Time,
		Status:         b.Status,
		TotalPrice:     b.TotalPrice,
		PriceMode:      b.PriceMode,
		QuotePrice:     b.QuotePrice,
		PriceStatus:    b.PriceStatus,
		Remarks:        derefString(b.Remarks),
		CreatedAt:      b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// notify writes a notification after a booking event; errors are logged and
// ignored — notification loss must never fail the booking action itself.
const (
	notifyLinkOrder             = "order"
	notifyLinkPhotographerOrder = "photographer_orders"
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

func (s *BookingService) Create(ctx context.Context, req CreateBookingRequest) (*BookingItem, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	if profile, err := s.queries.GetPhotographerById(ctx, req.PhotographerID); err == nil && profile.UserID != nil {
		blocked, err := s.queries.IsBlockedPair(ctx, int64(req.CoserID), *profile.UserID)
		if err == nil && blocked {
			return nil, ErrForbidden
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

	// 套餐归属校验：套餐必须归属于被预约的摄影师。平台模板套餐（photographer_id IS NULL）
	// 不作为可售商品，一律拒绝；套餐不存在（ErrNoRows）同样视为无效引用，但不掩盖真实 DB 故障。
	serviceID64 := int64(req.ServiceID)
	ownerID, err := s.queries.GetServicePhotographerID(ctx, serviceID64)
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
	if svc, err := s.queries.GetServiceById(ctx, serviceID64); err == nil {
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
		item.ServiceName = b.ServiceName
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
	updated, err := s.queries.UpdateBookingStatus(ctx, bookingID, newStatus)
	if err != nil {
		return nil, err
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

	return bookingToItem(updated), nil
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
	updated, err := s.queries.UpdateBookingStatus(ctx, bookingID, newStatus)
	if err != nil {
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
	return bookingToItem(updated), nil
}

// Quote records a single-round price offer from the booking's own photographer.
// Allowed only while price_mode='negotiable' AND price_status='awaiting_quote';
// the state guard lives in SQL and reports RowsAffected==0 for anything else.
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
