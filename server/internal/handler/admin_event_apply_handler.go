package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// AdminEventApplyHandler 是「摄影师申请入驻某漫展」的管理端审核入口。
// 错误映射复用 writeEventApplyError（同一 package），保证 C 端与 admin 口径一致。
type AdminEventApplyHandler struct {
	svc *service.EventApplyService
}

func NewAdminEventApplyHandler(svc *service.EventApplyService) *AdminEventApplyHandler {
	return &AdminEventApplyHandler{svc: svc}
}

type reviewEventApplicationRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// List 管理端列表：GET /api/admin/v1/event-applications?status=&page=&size=
func (h *AdminEventApplyHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	list, total, err := h.svc.AdminList(c.Request.Context(), status, page, size)
	if err != nil {
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

// Review 管理端审核：PUT /api/admin/v1/event-applications/:id/review
// body {status:"approved"|"rejected", reason}；rejected 必填 reason。
func (h *AdminEventApplyHandler) Review(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}
	var req reviewEventApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.svc.Review(c.Request.Context(), id, req.Status, req.Reason, middleware.AdminID(c)); err != nil {
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
