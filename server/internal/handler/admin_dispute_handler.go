package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

// AdminDisputeHandler backs the admin dispute console:
//
//	GET  /api/admin/v1/disputes          list + status filter + pagination
//	GET  /api/admin/v1/disputes/:id      detail with evidence and timeline
//	PUT  /api/admin/v1/disputes/:id/resolve  approve / reject / escalate
type AdminDisputeHandler struct {
	svc *service.AdminDisputeService
}

func NewAdminDisputeHandler(svc *service.AdminDisputeService) *AdminDisputeHandler {
	return &AdminDisputeHandler{svc: svc}
}

// List returns the paginated dispute list for the console.
func (h *AdminDisputeHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.svc.ListDisputes(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disputes unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

// Detail returns one dispute with its evidence and event timeline.
func (h *AdminDisputeHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dispute id"})
		return
	}

	detail, err := h.svc.GetDisputeDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrManageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dispute not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dispute unavailable"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// Resolve applies an admin ruling to a dispute.
func (h *AdminDisputeHandler) Resolve(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dispute id"})
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action is required"})
		return
	}

	dispute, err := h.svc.Resolve(c.Request.Context(), id, middleware.AdminID(c), req.Action, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAction), errors.Is(err, service.ErrReasonRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resolve request"})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrBookingNotFound), errors.Is(err, repository.ErrManageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "dispute not found"})
		case errors.Is(err, service.ErrInvalidTransition):
			c.JSON(http.StatusConflict, gin.H{"error": "invalid dispute state"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "resolve failed"})
		}
		return
	}
	c.JSON(http.StatusOK, dispute)
}
