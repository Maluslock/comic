package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
)

type AdminAuditHandler struct {
	repo *repository.AdminAuditRepo
}

func NewAdminAuditHandler(repo *repository.AdminAuditRepo) *AdminAuditHandler {
	return &AdminAuditHandler{repo: repo}
}

func (h *AdminAuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total, err := h.repo.List(c.Request.Context(), pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "audit logs unavailable"})
		return
	}
	if list == nil {
		list = []repository.AuditLog{}
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}
