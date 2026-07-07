package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

// HomeHandler handles HTTP requests for the home page.
type HomeHandler struct {
	svc *service.HomeService
}

// NewHomeHandler creates a new HomeHandler.
func NewHomeHandler(svc *service.HomeService) *HomeHandler {
	return &HomeHandler{svc: svc}
}

// GetHome handles GET /api/v1/home — returns aggregated home page data as JSON.
func (h *HomeHandler) GetHome(c *gin.Context) {
	data, err := h.svc.GetHomeData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
