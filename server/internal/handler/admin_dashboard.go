package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

type AdminDashboardHandler struct {
	svc *service.AdminStatsService
}

func NewAdminDashboardHandler(svc *service.AdminStatsService) *AdminDashboardHandler {
	return &AdminDashboardHandler{svc: svc}
}

func (h *AdminDashboardHandler) Dashboard(c *gin.Context) {
	d, err := h.svc.Dashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard unavailable"})
		return
	}
	c.JSON(http.StatusOK, d)
}
