package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// DisputeHandler exposes escrow dispute endpoints for C-end users.
type DisputeHandler struct {
	svc *service.BookingService
}

func NewDisputeHandler(svc *service.BookingService) *DisputeHandler {
	return &DisputeHandler{svc: svc}
}

// actorRole resolves the acting role server side: the acting user is either
// the photographer of the booking or the coser. Never trust a client header.
func (h *DisputeHandler) actorRole(c *gin.Context) (string, error) {
	role, err := h.svc.ActorRoleForUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		return "", err
	}
	return role, nil
}

// Open creates a dispute for a confirmed booking.
func (h *DisputeHandler) Open(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	var req struct {
		Type     string   `json:"type" binding:"required"`
		Reason   string   `json:"reason" binding:"required"`
		Evidence []string `json:"evidence"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and reason are required"})
		return
	}
	role, err := h.actorRole(c)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	userID := middleware.UserID(c)
	data, err := h.svc.OpenDispute(c.Request.Context(), id, userID, role, req.Type, req.Reason)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	// Evidence submitted with the dispute is stored as the first timeline
	// entry so the admin console can show what the applicant attached.
	evidence := req.Evidence
	if evidence == nil {
		evidence = []string{}
	}
	h.svc.RecordDisputeEvent(c.Request.Context(), data.ID, userID, role, "open", map[string]any{
		"type":     req.Type,
		"reason":   req.Reason,
		"evidence": evidence,
	})
	// The C-end reads camelCase; the repository struct is snake_case because it
	// doubles as the DB row. Map explicitly instead of leaking DB tags.
	c.JSON(http.StatusCreated, gin.H{
		"id":              data.ID,
		"bookingId":       data.BookingID,
		"type":            data.Type,
		"status":          data.Status,
		"respondDeadline": data.RespondDeadline.Format("2006-01-02T15:04:05Z07:00"),
		"createdAt":       data.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Respond lets the counterparty reply to a dispute.
func (h *DisputeHandler) Respond(c *gin.Context) {
	disputeID, err := strconv.ParseInt(c.Param("disputeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dispute id"})
		return
	}
	var req struct {
		Response string   `json:"response" binding:"required"`
		Evidence []string `json:"evidence"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "response is required"})
		return
	}
	role, err := h.actorRole(c)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	userID := middleware.UserID(c)
	data, err := h.svc.RespondDispute(c.Request.Context(), disputeID, userID, role, req.Response)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	evidence := req.Evidence
	if evidence == nil {
		evidence = []string{}
	}
	h.svc.RecordDisputeEvent(c.Request.Context(), disputeID, userID, role, "respond", map[string]any{
		"response": req.Response,
		"evidence": evidence,
	})
	c.JSON(http.StatusOK, data)
}

// Escalate moves a dispute into the escalated state so the platform has to
// take over. Either party may escalate.
func (h *DisputeHandler) Escalate(c *gin.Context) {
	disputeID, err := strconv.ParseInt(c.Param("disputeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dispute id"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // body is optional
	role, err := h.actorRole(c)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	data, err := h.svc.EscalateDispute(c.Request.Context(), disputeID, middleware.UserID(c), role, req.Reason)
	if err != nil {
		h.mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *DisputeHandler) mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, service.ErrInvalidTransition):
		c.JSON(http.StatusConflict, gin.H{"error": "invalid dispute state"})
	case errors.Is(err, service.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "dispute already open"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
