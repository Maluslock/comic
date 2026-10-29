package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type BookingHandler struct {
	svc *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) Create(c *gin.Context) {
	var req service.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.CoserID = int32(middleware.UserID(c))

	data, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "该时段已被预约"})
			return
		}
		if errors.Is(err, service.ErrInvalidReference) {
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer or service not found"})
			return
		}
		if errors.Is(err, service.ErrPastDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能预约已经过去的日期"})
			return
		}
		if errors.Is(err, service.ErrSelfBooking) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能预约自己发布的套餐"})
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot book this photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, data)
}

func (h *BookingHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Status   string `json:"status" binding:"required"`
		ActorTag string `json:"actorTag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.ActorTag == "" {
		req.ActorTag = "coser"
	}
	data, err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status, middleware.UserID(c), req.ActorTag)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBookingNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		case errors.Is(err, service.ErrInvalidTransition):
			c.JSON(http.StatusConflict, gin.H{"error": "invalid status transition"})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) Quote(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Price int32 `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price is required"})
		return
	}
	data, err := h.svc.Quote(c.Request.Context(), id, middleware.UserID(c), req.Price)
	if err != nil {
		h.mapQuoteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) RespondQuote(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Accept *bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Accept == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accept is required"})
		return
	}
	data, err := h.svc.RespondQuote(c.Request.Context(), id, middleware.UserID(c), *req.Accept)
	if err != nil {
		h.mapQuoteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) mapQuoteErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, service.ErrQuoteNotAllowed):
		c.JSON(http.StatusConflict, gin.H{"error": "quote not allowed in current state"})
	case errors.Is(err, service.ErrInvalidPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be between 1 and 99999"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (h *BookingHandler) ListByPhotographer(c *gin.Context) {
	pid, err := strconv.Atoi(c.Param("photographerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	data, err := h.svc.ListByPhotographerForUser(c.Request.Context(), int32(pid), middleware.UserID(c))
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) ListByUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if !assertSelf(c, int64(userID)) {
		return
	}

	data, err := h.svc.ListByUser(c.Request.Context(), int32(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Checkin validates the coser's check-in code and records arrival.
func (h *BookingHandler) Checkin(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	data, err := h.svc.Checkin(c.Request.Context(), id, middleware.UserID(c), req.Code)
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// MarkDelivered records photographer delivery with optional evidence.
func (h *BookingHandler) MarkDelivered(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Evidence []string `json:"evidence"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	data, err := h.svc.MarkDelivered(c.Request.Context(), id, middleware.UserID(c), req.Evidence)
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// AcceptDelivery lets the coser accept a delivered booking.
func (h *BookingHandler) AcceptDelivery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	data, err := h.svc.AcceptDelivery(c.Request.Context(), id, middleware.UserID(c))
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GetEvents returns the escrow timeline of a booking (check-in, delivery,
// acceptance, dispute events). Only the two parties of the booking may read it.
func (h *BookingHandler) GetEvents(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	events, err := h.svc.GetOrderEvents(c.Request.Context(), id, middleware.UserID(c))
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": events})
}

// GetDispute returns the C-end view of the dispute for a booking.
func (h *BookingHandler) GetDispute(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	data, err := h.svc.GetBookingDispute(c.Request.Context(), id, middleware.UserID(c))
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// MarkNoShow lets a photographer mark the coser as a no-show. The action is
// idempotent and applies at most one -15 credit delta per booking.
func (h *BookingHandler) MarkNoShow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	data, err := h.svc.MarkCoserNoShow(c.Request.Context(), id, middleware.UserID(c))
	if err != nil {
		h.mapNodeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *BookingHandler) mapNodeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, service.ErrInvalidTransition):
		c.JSON(http.StatusConflict, gin.H{"error": "invalid status transition"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
