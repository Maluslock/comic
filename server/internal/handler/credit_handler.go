package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// CreditHandler exposes the read-only credit endpoints used by the C-end card.
type CreditHandler struct {
	svc *service.CreditService
}

func NewCreditHandler(svc *service.CreditService) *CreditHandler {
	return &CreditHandler{svc: svc}
}

// Summary serves GET /api/v1/me/credit.
func (h *CreditHandler) Summary(c *gin.Context) {
	data, err := h.svc.Summary(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Events serves GET /api/v1/me/credit/events?limit=&offset=.
func (h *CreditHandler) Events(c *gin.Context) {
	limit := 20
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && v > 0 {
		limit = v
	}
	if limit > 100 {
		limit = 100
	}
	offset := 0
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v > 0 {
		offset = v
	}

	data, err := h.svc.Events(c.Request.Context(), middleware.UserID(c), int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}
